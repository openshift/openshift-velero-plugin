package imagetag

import (
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to secrets
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"imagetags"},
	}, nil
}

// Execute action for the restore plugin for the secret resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {

	p.Log.Infof("[imagetag-restore] skipping restore of imagetag")
	return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
