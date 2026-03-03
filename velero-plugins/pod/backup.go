package pod

import (
	"context"
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// BackupPlugin is a backup item action plugin for Velero
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to replicasets
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"pods"},
	}, nil
}

// Execute action for the backup plugin for the pod resource
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[pod-backup] Entering Pod backup plugin")

	var additionalItems []velero.ResourceIdentifier
	var err error = nil
	pod := corev1API.Pod{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &pod)
	p.Log.Infof("[pod-backup] pod: %s", pod.Name)

	annotations := pod.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[common.DCIncludesDMFix] = "true"
	pod.Annotations = annotations

	if sccName, exists := pod.Annotations[common.SCCPodAnnotation]; exists {
		p.Log.Infof("[pod-backup] Pod %s has SCC annotation with value: %s", pod.Name, sccName)

		sccIdentifier, localerr := p.addSCC(sccName)
		if localerr == nil {
			additionalItems = append(additionalItems, sccIdentifier)
		} else {
			// continuing backup despite error, SCCs can be deleted after Pod creation
			if errors.IsNotFound(localerr) {
				p.Log.Warnf("[pod-backup] SCC %s for pod %s not found: %s", sccName, pod.Name, localerr.Error())
			} else {
				p.Log.Errorf("[pod-backup] Error adding SCC %s for pod %s: %s", sccName, pod.Name, localerr.Error())
				// error will go out of scope otherwise
				err = localerr
			}
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(pod)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, additionalItems, err
}

func (p *BackupPlugin) addSCC(sccName string) (velero.ResourceIdentifier, error) {
	securityClient, err := clients.SecurityClient()
	if err != nil {
		return velero.ResourceIdentifier{}, err
	}

	scc, err := securityClient.SecurityContextConstraints().Get(context.Background(), sccName, metav1.GetOptions{})
	if err != nil {
		return velero.ResourceIdentifier{}, err
	}
	// resource plural can be retrieved from the openshift api but more trouble than it is worth
	return velero.ResourceIdentifier{
		GroupResource: schema.GroupResource{
			Group:    scc.GroupVersionKind().Group,
			Resource: "securitycontextconstraints",
		},
		Name: scc.Name,
	}, nil
}
