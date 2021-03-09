package deploymentconfig

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	appsv1API "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
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
	common.SwapContainerImageRefs(deploymentConfig.Spec.Template.Spec.Containers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)
	common.SwapContainerImageRefs(deploymentConfig.Spec.Template.Spec.InitContainers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)

	namespaceMapping := input.Restore.Spec.NamespaceMapping
	newNamespace := namespaceMapping[deploymentConfig.Namespace]
	if len(input.Restore.Spec.NamespaceMapping) > 0 {
		for i := range deploymentConfig.Spec.Triggers {
			if deploymentConfig.Spec.Triggers[i].ImageChangeParams == nil {
				continue
			}

			// if trigger namespace is mapped to new one, swap it
			triggerNamespace := deploymentConfig.Spec.Triggers[i].ImageChangeParams.From.Namespace
			if namespaceMapping[triggerNamespace] != "" {
				deploymentConfig.Spec.Triggers[i].ImageChangeParams.From.Namespace = newNamespace
			}
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(deploymentConfig)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
