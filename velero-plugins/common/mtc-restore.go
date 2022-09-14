package common

import (
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/meta"
)

// RestorePlugin is a restore item action plugin for Heptio Ark.
type MTCRestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to the listed resources in the slice.
func (p *MTCRestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
}

// Execute sets a custom annotation on the item being restored.
func (p *MTCRestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("mtc-common-restore] Entering MTC common restore plugin")

	metadata, err := meta.Accessor(input.Item)
	if err != nil {
		return nil, err
	}
	name := metadata.GetName()
	p.Log.Infof("[mtc-common-restore] MTC common restore plugin for %s", name)

	if input.Restore.Labels[MigrationApplicationLabelKey] == MigrationApplicationLabelValue {

		// Set migmigration and migplan labels on all resources, except ServiceAccounts
		migMigrationLabel, exist := input.Restore.Labels[MigMigrationLabelKey]
		if !exist {
			p.Log.Info("migmigration label was not found on restore")
		}
		migPlanLabel, exist := input.Restore.Labels[MigPlanLabelKey]
		if !exist {
			p.Log.Info("migplan label was not found on restore")
		}
		labels := metadata.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}
		labels[MigMigrationLabelKey] = migMigrationLabel
		labels[MigPlanLabelKey] = migPlanLabel

		metadata.SetLabels(labels)
	}

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
