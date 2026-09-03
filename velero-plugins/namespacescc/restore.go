package namespacescc

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin compares the SCC UID/GID-range annotations stashed at backup
// time (see backup.go) against the actual annotations on the restored
// namespace, and logs a warning on mismatch. This is the closest available
// signal: Namespace objects never pass through RestoreItemAction plugins, so
// there's no way to add to the restore's Warnings the way other item actions
// can (RestoreItemActionExecuteOutput has no Warning field in this Velero API
// version).
type RestorePlugin struct {
	Log logrus.FieldLogger

	// namespaceAnnotationCache avoids one Namespaces().Get() per ServiceAccount
	// when a namespace has multiple service accounts. Cleared whenever
	// cachedForRestore no longer matches the current restore, so entries don't
	// leak across restores handled by the same long-lived plugin process.
	// Guarded by mu since the shared plugin process may serve concurrent
	// operations.
	mu                       sync.Mutex
	namespaceAnnotationCache map[string]map[string]string
	cachedForRestore         string
}

// AppliesTo returns a velero.ResourceSelector that applies to service accounts.
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"serviceaccounts"},
	}, nil
}

// Execute compares backed-up vs. actual namespace SCC UID/GID-range annotations and strips the bookkeeping annotations.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[namespacescc-restore] Entering namespace SCC range restore plugin")

	backedUp := corev1.ServiceAccount{}
	backedUpMarshal, _ := json.Marshal(input.ItemFromBackup)
	json.Unmarshal(backedUpMarshal, &backedUp)

	expected := expectedSCCAnnotations(backedUp.Annotations)

	serviceAccount := corev1.ServiceAccount{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &serviceAccount)

	if len(expected) > 0 {
		targetNamespace := serviceAccount.Namespace
		if mapped := input.Restore.Spec.NamespaceMapping[targetNamespace]; mapped != "" {
			targetNamespace = mapped
		}

		namespaceAnnotations, err := p.getNamespaceAnnotations(input.Restore.Name, targetNamespace)
		if err != nil {
			p.Log.Warnf("[namespacescc-restore] unable to verify namespace %s SCC range: %v", targetNamespace, err)
		} else {
			for _, m := range sccAnnotationMismatches(targetNamespace, expected, namespaceAnnotations) {
				p.Log.Warn(m)
			}
		}
	}

	serviceAccount.Annotations = stripSCCBookkeepingAnnotations(serviceAccount.Annotations)

	var out map[string]interface{}
	objrec, _ := json.Marshal(serviceAccount)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// getNamespaceAnnotations returns the annotations of the named namespace,
// caching the result for the lifetime of the current restore so a namespace
// with multiple service accounts only needs one Get() call.
func (p *RestorePlugin) getNamespaceAnnotations(restoreName, namespace string) (map[string]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cachedForRestore != restoreName {
		p.namespaceAnnotationCache = map[string]map[string]string{}
		p.cachedForRestore = restoreName
	}
	if annotations, ok := p.namespaceAnnotationCache[namespace]; ok {
		return annotations, nil
	}

	client, err := clients.CoreClient()
	if err != nil {
		return nil, err
	}
	ns, err := client.Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	p.namespaceAnnotationCache[namespace] = ns.Annotations
	return ns.Annotations, nil
}

// expectedSCCAnnotations extracts the backed-up namespace SCC annotations
// stashed on the ServiceAccount's bookkeeping annotations, keyed by the real
// SCC annotation name.
func expectedSCCAnnotations(backedUpAnnotations map[string]string) map[string]string {
	expected := map[string]string{}
	for src, dst := range sccAnnotationCarrier {
		if v, ok := backedUpAnnotations[dst]; ok && v != "" {
			expected[src] = v
		}
	}
	return expected
}

// sccAnnotationMismatches compares expected (backed-up) vs. actual namespace SCC
// annotations and returns a human-readable message per mismatch, sorted by
// annotation name for deterministic output.
func sccAnnotationMismatches(namespaceName string, expected, actual map[string]string) []string {
	var annotations []string
	for annotation := range expected {
		annotations = append(annotations, annotation)
	}
	sort.Strings(annotations)

	var mismatches []string
	for _, annotation := range annotations {
		expectedValue := expected[annotation]
		actualValue := actual[annotation]
		if actualValue != expectedValue {
			mismatches = append(mismatches, fmt.Sprintf(
				"[namespacescc-restore] namespace %q annotation %q mismatch: backed up as %q, restored as %q. "+
					"Workloads relying on the backed-up UID/GID range may have incorrect file ownership.",
				namespaceName, annotation, expectedValue, actualValue))
		}
	}
	return mismatches
}

// stripSCCBookkeepingAnnotations removes the bookkeeping annotations added at backup time.
func stripSCCBookkeepingAnnotations(annotations map[string]string) map[string]string {
	for _, dst := range sccAnnotationCarrier {
		delete(annotations, dst)
	}
	return annotations
}
