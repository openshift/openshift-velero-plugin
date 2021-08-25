package replicaset

import (
	"context"
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	appsv1API "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Velero
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to replicasets
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"replicasets.apps"},
	}, nil
}

// Execute action for the backup plugin for the replicaset resource
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[replicaset-backup] Entering ReplicaSet backup plugin")

	replicaSet := appsv1API.ReplicaSet{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &replicaSet)
	p.Log.Infof("[replicaset-backup] replicaset: %s", replicaSet.Name)

	ownerRefs, err := common.GetOwnerReferences(item)
	if err != nil {
		return nil, nil, err
	}
	// Mark replicaset for restore if owned by a paused Deployment
	for i := range ownerRefs {
		ref := ownerRefs[i]
		if ref.Kind == "Deployment" {

			client, err := clients.AppsClient()
			if err != nil {
				return nil, nil, err
			}

			deployment, err := client.Deployments(replicaSet.Namespace).Get(context.Background(), ref.Name, metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}

			if deployment.Spec.Paused {
				p.Log.Infof("[replicaset-backup] owner deployment paused, annotating ReplicaSet %s for restore", replicaSet.Name)
				annotations := replicaSet.Annotations
				if annotations == nil {
					annotations = make(map[string]string)
				}

				annotations[common.PausedOwnerRef] = "true"
				replicaSet.Annotations = annotations
			}
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(replicaSet)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, nil, nil
}
