package replicationcontroller

import (
	"encoding/json"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/common"
	"github.com/heptio/velero/pkg/plugin/velero"
	"github.com/sirupsen/logrus"
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
	common.SwapContainerImageRefs(replicationController.Spec.Template.Spec.Containers, backupRegistry, registry, p.Log)
	common.SwapContainerImageRefs(replicationController.Spec.Template.Spec.InitContainers, backupRegistry, registry, p.Log)

	var out map[string]interface{}
	objrec, _ := json.Marshal(replicationController)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}
