package pvc

import (
	"encoding/json"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

const (
	// VMFRRestoreAnnotation is the annotation that triggers VMFR PVC renaming
	VMFRRestoreAnnotation = "oadp.openshift.io/vmfr-restore"
	// VMFRBackupLabel identifies which backup a restored PVC came from
	VMFRBackupLabel = "oadp.openshift.io/vmfr-backup"
	// VMFROriginalNameLabel stores the original PVC name before VMFR renaming
	VMFROriginalNameLabel = "oadp.openshift.io/vmfr-original-name"
)

// VMFRRestorePlugin is a restore item action plugin for VMFR PVC renaming
type VMFRRestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to PVCs
func (p *VMFRRestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"persistentvolumeclaims"},
	}, nil
}

// Execute performs the VMFR PVC renaming action
func (p *VMFRRestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[vmfr-pvc-restore] Executing VMFR PVC Restore Item Action")

	// Check if this restore has the VMFR annotation
	if !isVMFRRestore(input) {
		p.Log.Debug("[vmfr-pvc-restore] Not a VMFR restore, skipping PVC name modification")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	// Convert the item to a PVC
	pvc := corev1API.PersistentVolumeClaim{}
	itemMarshal, err := json.Marshal(input.Item)
	if err != nil {
		return nil, fmt.Errorf("[vmfr-pvc-restore] error marshaling item: %v", err)
	}
	if err := json.Unmarshal(itemMarshal, &pvc); err != nil {
		return nil, fmt.Errorf("[vmfr-pvc-restore] error unmarshaling PVC: %v", err)
	}

	// Generate unique PVC name using GenerateName to avoid collisions.
	// The Kubernetes API server will automatically truncate long prefixes (>58 chars)
	// and append a 5-character random suffix to ensure uniqueness.
	originalName := pvc.Name
	backupName := input.Restore.Spec.BackupName
	generateNamePrefix := fmt.Sprintf("%s-%s-", backupName, originalName)

	p.Log.Infof("[vmfr-pvc-restore] Setting GenerateName for PVC from %s to %s for VMFR multi-backup restore", originalName, generateNamePrefix)

	// Use GenerateName instead of Name to let Kubernetes generate unique suffix
	pvc.GenerateName = generateNamePrefix
	pvc.Name = ""

	// Add tracking labels for VMFR management
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}
	pvc.Labels[VMFRBackupLabel] = backupName
	pvc.Labels[VMFROriginalNameLabel] = originalName

	// Convert back to unstructured
	var out map[string]interface{}
	objrec, err := json.Marshal(pvc)
	if err != nil {
		return nil, fmt.Errorf("[vmfr-pvc-restore] error marshaling updated PVC: %v", err)
	}
	if err := json.Unmarshal(objrec, &out); err != nil {
		return nil, fmt.Errorf("[vmfr-pvc-restore] error unmarshaling to unstructured: %v", err)
	}

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// isVMFRRestore checks if the restore has the VMFR annotation
func isVMFRRestore(input *velero.RestoreItemActionExecuteInput) bool {
	if input.Restore.Annotations == nil {
		return false
	}
	value, exists := input.Restore.Annotations[VMFRRestoreAnnotation]
	return exists && value == "true"
}
