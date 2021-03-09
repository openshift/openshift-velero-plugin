package deployment

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	appsv1API "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to deployments
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"deployments.apps"},
	}, nil
}

// Execute action for the restore plugin for the deployment resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[deployment-restore] Entering Deployment restore plugin")

	deployment := appsv1API.Deployment{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &deployment)
	p.Log.Infof("[deployment-restore] deployment: %s", deployment.Name)

	backupRegistry, registry, err := common.GetSrcAndDestRegistryInfo(input.Item)
	if err != nil {
		return nil, err
	}
	common.SwapContainerImageRefs(deployment.Spec.Template.Spec.Containers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)
	common.SwapContainerImageRefs(deployment.Spec.Template.Spec.InitContainers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)

	var out map[string]interface{}
	objrec, _ := json.Marshal(deployment)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
