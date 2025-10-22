package pvc

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/rand"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	// VMFRRestoreAnnotation is the annotation that triggers VMFR PVC renaming
	VMFRRestoreAnnotation = "oadp.openshift.io/vmfr-restore"
	// VMFRBackupAnnotation identifies which backup a restored PVC came from
	VMFRBackupAnnotation = "oadp.openshift.io/vmfr-backup"
	// VMFROriginalNameAnnotation stores the original PVC name before VMFR renaming
	VMFROriginalNameAnnotation = "oadp.openshift.io/vmfr-original-name"
	// VMFRPrimaryLabel identifies the primary (first restored) PVC with original name
	VMFRPrimaryLabel = "oadp.openshift.io/vmfr-primary"
)

// VMFRRestorePlugin is a restore item action plugin for VMFR PVC renaming
type VMFRRestorePlugin struct {
	Log        logrus.FieldLogger
	CoreClient corev1client.CoreV1Interface
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

	originalName := pvc.Name
	backupName := input.Restore.Spec.BackupName
	namespace := input.Restore.Spec.NamespaceMapping[pvc.Namespace]
	if namespace == "" {
		namespace = pvc.Namespace
	}

	p.Log.Infof("[vmfr-pvc-restore] Processing PVC: originalName=%s, pvc.Namespace=%s, mappedNamespace=%s, backupName=%s",
		originalName, pvc.Namespace, namespace, backupName)

	// Get Kubernetes client to check if PVC already exists
	coreClient := p.CoreClient
	if coreClient == nil {
		// If not injected (for production use), get from clients package
		coreClient, err = clients.CoreClient()
		if err != nil {
			return nil, fmt.Errorf("[vmfr-pvc-restore] error getting Kubernetes client: %v", err)
		}
	}

	// Check if PVC with original name already exists in the target namespace
	p.Log.Infof("[vmfr-pvc-restore] Checking if PVC %s exists in namespace %s", originalName, namespace)
	_, err = coreClient.PersistentVolumeClaims(namespace).Get(context.Background(), originalName, metav1.GetOptions{})
	pvcExists := err == nil
	if err != nil && !errors.IsNotFound(err) {
		// Unexpected error (not "not found")
		return nil, fmt.Errorf("[vmfr-pvc-restore] error checking if PVC exists: %v", err)
	}
	p.Log.Infof("[vmfr-pvc-restore] PVC %s in namespace %s exists: %v (error: %v)", originalName, namespace, pvcExists, err)

	// Initialize annotations and labels
	if pvc.Annotations == nil {
		pvc.Annotations = make(map[string]string)
	}
	if pvc.Labels == nil {
		pvc.Labels = make(map[string]string)
	}

	if !pvcExists {
		// First restore: Keep original name, mark as primary
		p.Log.Infof("[vmfr-pvc-restore] PVC %s does not exist, keeping original name as primary", originalName)
		pvc.Name = originalName
		pvc.GenerateName = ""
		pvc.Labels[VMFRPrimaryLabel] = "true"
		pvc.Annotations[VMFRBackupAnnotation] = backupName
		pvc.Annotations[VMFROriginalNameAnnotation] = originalName
	} else {
		// Subsequent restore: PVC already exists, generate unique name
		// Pattern: {backup-name}-{10-char-random-suffix}
		p.Log.Infof("[vmfr-pvc-restore] PVC %s already exists, generating unique name for multi-backup restore", originalName)

		// Generate 10-character random suffix for strong collision resistance
		suffix := rand.String(10)

		// Kubernetes resource names must be <= 253 characters (DNS-1123 subdomain)
		// Format: backup-name + "-" + 10-char-suffix
		maxLen := 253
		maxBackupNameLen := maxLen - 1 - 10 // 242 chars for backup name

		truncatedBackupName := backupName
		if len(backupName) > maxBackupNameLen {
			truncatedBackupName = backupName[:maxBackupNameLen]
			p.Log.Warnf("[vmfr-pvc-restore] Backup name truncated from %d to %d chars", len(backupName), maxBackupNameLen)
		}

		newPVCName := fmt.Sprintf("%s-%s", truncatedBackupName, suffix)
		pvc.Name = newPVCName
		pvc.GenerateName = ""

		p.Log.Infof("[vmfr-pvc-restore] Renamed PVC from %s to %s for VMFR multi-backup restore", originalName, newPVCName)

		// Mark as non-primary and store original name
		pvc.Labels[VMFRPrimaryLabel] = "false"
		pvc.Annotations[VMFRBackupAnnotation] = backupName
		pvc.Annotations[VMFROriginalNameAnnotation] = originalName
	}

	// Add backup label for filtering (both primary and non-primary)
	pvc.Labels[VMFRBackupAnnotation] = backupName

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
