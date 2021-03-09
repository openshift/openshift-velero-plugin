package replicationcontroller

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to replicationcontrollers
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"replicationcontrollers"},
	}, nil
}

// Execute action for the restore plugin for the replication controller resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[replicationcontroller-restore] Entering ReplicationController restore plugin")

	replicationController := corev1API.ReplicationController{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &replicationController)
	p.Log.Infof("[replicationcontroller-restore] replicationController: %s", replicationController.Name)

	backupRegistry, registry, err := common.GetSrcAndDestRegistryInfo(input.Item)
	if err != nil {
		return nil, err
	}
	common.SwapContainerImageRefs(replicationController.Spec.Template.Spec.Containers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)
	common.SwapContainerImageRefs(replicationController.Spec.Template.Spec.InitContainers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)

	ownerRefs, err := common.GetOwnerReferences(input.ItemFromBackup)
	if err != nil {
		return nil, err
	}
	// Don't restore ReplicationController if owned by DeploymentConfig
	for i := range ownerRefs {
		ref := ownerRefs[i]
		if ref.Kind == "DeploymentConfig" {
			p.Log.Infof("[replicationcontroller-restore] skipping restore of ReplicationController %s, belongs to DeploymentConfig", replicationController.Name)
			return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(replicationController)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
