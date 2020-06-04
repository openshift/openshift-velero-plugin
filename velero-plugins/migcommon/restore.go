package migcommon

import (
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
)

// RestorePlugin is a restore item action plugin for Velero.
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to everything.
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}

// Execute sets a custom annotation on the item being restored.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[common-restore] Entering common migration restore plugin")

	metadata, err := meta.Accessor(input.Item)
	if err != nil {
		return nil, err
	}
	// Skip cluster-scoped resources
	if len(metadata.GetNamespace()) == 0 {
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	name := metadata.GetName()
	p.Log.Infof("[common-restore] common migration restore plugin for %s", name)

	// Set migmigraiton label on all resources, except ServiceAccounts
	switch input.Item.DeepCopyObject().(type) {
	case *corev1.ServiceAccount:
		break
	default:
		migMigrationLabel, exist := input.Restore.Labels[MigMigrationLabelKey]
		if !exist {
			return nil, errors.New("migmigration label is not found on restore")
		}
		labels := metadata.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[MigratedByLabel] = migMigrationLabel
		metadata.SetLabels(labels)
	}

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
