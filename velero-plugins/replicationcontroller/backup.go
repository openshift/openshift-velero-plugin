package replicationcontroller

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

// AppliesTo returns a velero.ResourceSelector that applies to replicationcontrollers
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"replicationcontrollers"},
	}, nil
}

// Execute action for the backup plugin for the replicationcontroller resource
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[replicationcontroller-backup] Entering ReplicaSet backup plugin")

	replicaSet := appsv1API.ReplicaSet{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &replicaSet)
	p.Log.Infof("[replicationcontroller-backup] replicationcontroller: %s", replicaSet.Name)

	ownerRefs, err := common.GetOwnerReferences(item)
	if err != nil {
		return nil, nil, err
	}
	// Mark replicationcontroller for restore if owned by a paused Deployment
	for i := range ownerRefs {
		ref := ownerRefs[i]
		if ref.Kind == "DeploymentConfig" {

			client, err := clients.OCPAppsClient()
			if err != nil {
				return nil, nil, err
			}

			deploymentConfig, err := client.DeploymentConfigs(replicaSet.Namespace).Get(context.Background(), ref.Name, metav1.GetOptions{})
			if err != nil {
				return nil, nil, err
			}

			if deploymentConfig.Spec.Paused {
				p.Log.Infof("[replicationcontroller-backup] owner deploymentConfig paused, annotating ReplicaSet %s for restore", replicaSet.Name)
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
