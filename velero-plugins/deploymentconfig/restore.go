package deploymentconfig

import (
	"encoding/json"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/common"
	"github.com/heptio/velero/pkg/plugin/velero"
	appsv1API "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to deploymentconfigs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"deploymentconfigs"},
	}, nil
}

// Execute action for the restore plugin for the deployment config resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[deploymentconfig-restore] Entering DeploymentConfig restore plugin")

	deploymentConfig := appsv1API.DeploymentConfig{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &deploymentConfig)
	p.Log.Infof("[deploymentconfig-restore] deploymentConfig: %s", deploymentConfig.Name)

	backupRegistry, registry, err := common.GetSrcAndDestRegistryInfo(input.Item)
	if err != nil {
		return nil, err
	}
	common.SwapContainerImageRefs(deploymentConfig.Spec.Template.Spec.Containers, backupRegistry, registry, p.Log)
	common.SwapContainerImageRefs(deploymentConfig.Spec.Template.Spec.InitContainers, backupRegistry, registry, p.Log)

	var out map[string]interface{}
	objrec, _ := json.Marshal(deploymentConfig)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}
