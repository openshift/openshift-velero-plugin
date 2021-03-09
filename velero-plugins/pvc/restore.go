package pvc

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

// AppliesTo returns a velero.ResourceSelector that applies to PVCs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"persistentvolumeclaims"},
	}, nil
}

// Execute action for the restore plugin for the pvc resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {

	if input.Restore.Labels[common.MigrationApplicationLabelKey] != common.MigrationApplicationLabelValue{
		p.Log.Info("[pvc-restore] Returning pvc object as is since this is not a migration activity")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	p.Log.Info("[pvc-restore] Entering Persistent Volume Claim restore plugin")

	pvc := corev1API.PersistentVolumeClaim{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &pvc)
	p.Log.Infof("[pvc-restore] pvc: %s", pvc.Name)

	// Use default behavior (restore the PV) for a swing migration.
	// For copy we remove annotations and PV volumeName
	if pvc.Annotations[common.MigrateTypeAnnotation] == common.PvCopyAction {

		// Skip the PVC if this is a stage restore for a stage migration *and* it's a snapshot copy
		// since snapshot restore is not incremental
		if input.Restore.Annotations[common.StageOrFinalMigrationAnnotation] == common.StageMigration &&
			len(input.Restore.Labels[common.StageRestoreLabel])>0 &&
			pvc.Annotations[common.MigrateCopyMethodAnnotation] == common.PvSnapshotCopyMethod {
			p.Log.Infof("[pvc-restore] skipping restore of pv %s, snapshot PVCs restored only on final migration", pvc.Name)
			return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
		}
		// ISSUE-61 : removing the label selectors from PVC's
		// to avoid PV dynamic provisioner getting stuck
		pvc.Spec.Selector = nil
		storageClassName := pvc.Annotations[common.MigrateStorageClassAnnotation]
		pvc.Spec.StorageClassName = &storageClassName
		if pvc.Annotations[corev1API.BetaStorageClassAnnotation] != "" {
			pvc.Annotations[corev1API.BetaStorageClassAnnotation] = storageClassName
		}
		accessMode := pvc.Annotations[common.MigrateAccessModeAnnotation]
		if accessMode != "" {
			pvc.Spec.AccessModes = []corev1API.PersistentVolumeAccessMode{corev1API.PersistentVolumeAccessMode(accessMode)}
		}
	}
	delete(pvc.Annotations, common.PVCSelectedNodeAnnotation)

	var out map[string]interface{}
	objrec, _ := json.Marshal(pvc)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
