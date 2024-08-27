package configmap

import (
	"strconv"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to configmaps
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"configmaps"},
	}, nil
}

// Execute action for the restore plugin for the configmap resource
// We want to skip restore ConfigMaps that contain skip annotation added by ocp plugin at backup time.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[cm-restore] Entering ConfigMap restore plugin")
	metadata, err := meta.Accessor(input.Item)
	if err != nil {
		p.Log.Warnf("[cm-restore] Unable to access metadata, err: %v", err)
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	annotations := metadata.GetAnnotations()
	if annotations == nil {
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	if boolVal, ok := annotations[common.SkipBuildConfigConfigMapRestore]; ok {
		shouldSkip, _ := strconv.ParseBool(boolVal)
		if shouldSkip {
			p.Log.Infof("[cm-restore] Skipping restore of ConfigMap %s, belongs to build pod, will regenerate as needed", metadata.GetName())
			return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
		}
	}
	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
