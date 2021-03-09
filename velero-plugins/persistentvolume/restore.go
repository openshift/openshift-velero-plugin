package persistentvolume

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

// AppliesTo returns a velero.ResourceSelector that applies to PVs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"persistentvolumes"},
	}, nil
}

// Execute action for the restore plugin for the pv resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {

	if input.Restore.Labels[common.MigrationApplicationLabelKey] != common.MigrationApplicationLabelValue{
		p.Log.Info("[pv-restore] Returning pv object as is since this is not a migration activity")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	p.Log.Info("[pv-restore] Entering Persistent Volume restore plugin")

	pv := corev1API.PersistentVolume{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &pv)
	p.Log.Infof("[pv-restore] pv: %s", pv.Name)
	if pv.Annotations[common.MigrateTypeAnnotation] == common.PvCopyAction {
		// Skip the PV if this is a stage restore for a stage migration *and* it's a snapshot copy
		// since snapshot restore is not incremental
		if input.Restore.Annotations[common.StageOrFinalMigrationAnnotation] == common.StageMigration &&
			len(input.Restore.Labels[common.StageRestoreLabel])>0 &&
			pv.Annotations[common.MigrateCopyMethodAnnotation] == common.PvSnapshotCopyMethod {
			p.Log.Infof("[pv-restore] skipping restore of pv %s, snapshot PVs restored only on final migration", pv.Name)
			return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
		}
		p.Log.Infof("[pv-restore] Setting storage class, %s.", pv.Name)
		storageClassName := pv.Annotations[common.MigrateStorageClassAnnotation]
		pv.Spec.StorageClassName = storageClassName
		if pv.Annotations[corev1API.BetaStorageClassAnnotation] != "" {
			pv.Annotations[corev1API.BetaStorageClassAnnotation] = storageClassName
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(pv)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
