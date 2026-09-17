package namespacescc

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	apisecurity "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// sccAnnotationCarrier maps the namespace's SCC UID/GID-range annotations to the
// bookkeeping annotation keys they're stashed under on the ServiceAccount.
var sccAnnotationCarrier = map[string]string{
	apisecurity.UIDRangeAnnotation:           common.BackupNsSccUIDRange,
	apisecurity.SupplementalGroupsAnnotation: common.BackupNsSccSupplementalGroups,
	apisecurity.MCSAnnotation:                common.BackupNsSccMcs,
}

// BackupPlugin stashes the parent namespace's SCC UID/GID-range annotations onto
// each ServiceAccount at backup time. Namespace objects never pass through
// RestoreItemAction plugins on restore (Velero core special-cases and skips
// them), so ServiceAccounts - always present in every namespace - are used as
// the carrier to detect a range mismatch at restore time (see restore.go).
type BackupPlugin struct {
	Log logrus.FieldLogger

	// namespaceAnnotationCache avoids one Namespaces().Get() per ServiceAccount
	// when a namespace has multiple service accounts. Cleared whenever
	// cachedForBackup no longer matches the current backup, so entries don't
	// leak across backups handled by the same long-lived plugin process.
	// Guarded by mu since the shared plugin process may serve concurrent
	// operations.
	mu                       sync.Mutex
	namespaceAnnotationCache map[string]map[string]string
	cachedForBackup          string
}

// AppliesTo returns a velero.ResourceSelector that applies to service accounts.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"serviceaccounts"},
	}, nil
}

// Execute stashes the namespace's SCC UID/GID-range annotations onto the service account being backed up.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[namespacescc-backup] Entering namespace SCC range backup plugin")

	serviceAccount := corev1.ServiceAccount{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &serviceAccount)

	namespaceAnnotations, err := p.getNamespaceAnnotations(backup.Name, serviceAccount.Namespace)
	if err != nil {
		return nil, nil, err
	}

	annotations, stashed := stashNamespaceSCCAnnotations(serviceAccount.Annotations, namespaceAnnotations)
	if !stashed {
		return item, nil, nil
	}
	serviceAccount.Annotations = annotations

	var out map[string]interface{}
	objrec, _ := json.Marshal(serviceAccount)
	json.Unmarshal(objrec, &out)

	return &unstructured.Unstructured{Object: out}, nil, nil
}

// getNamespaceAnnotations returns the annotations of the named namespace,
// caching the result for the lifetime of the current backup so a namespace
// with multiple service accounts only needs one Get() call.
func (p *BackupPlugin) getNamespaceAnnotations(backupName, namespace string) (map[string]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cachedForBackup != backupName {
		p.namespaceAnnotationCache = map[string]map[string]string{}
		p.cachedForBackup = backupName
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

// stashNamespaceSCCAnnotations returns a copy of saAnnotations with the
// namespace's SCC UID/GID-range annotations (if present on namespaceAnnotations)
// stashed under bookkeeping keys. The second return value reports whether
// anything was stashed.
func stashNamespaceSCCAnnotations(saAnnotations, namespaceAnnotations map[string]string) (map[string]string, bool) {
	var stashed bool
	out := saAnnotations
	for src, dst := range sccAnnotationCarrier {
		v, ok := namespaceAnnotations[src]
		if !ok || v == "" {
			continue
		}
		if out == nil {
			out = map[string]string{}
		}
		out[dst] = v
		stashed = true
	}
	return out, stashed
}
