package pvc

import (
	"fmt"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestVMFRRestorePlugin_Execute(t *testing.T) {
	tests := []struct {
		name                     string
		hasVMFRAnnotation        bool
		restoreName              string
		backupName               string
		pvcName                  string
		expectedGenerateNamePrefix string
		expectedName             string
		expectedLabels           map[string]string
	}{
		{
			name:                     "VMFR restore with annotation should use GenerateName",
			hasVMFRAnnotation:        true,
			restoreName:              "vmfr-my-instance-backup-20250101",
			backupName:               "backup-20250101",
			pvcName:                  "my-app-pvc",
			expectedGenerateNamePrefix: "backup-20250101-my-app-pvc-",
			expectedName:             "", // Name should be cleared when using GenerateName
			expectedLabels: map[string]string{
				VMFRBackupLabel:       "backup-20250101",
				VMFROriginalNameLabel: "my-app-pvc",
			},
		},
		{
			name:                     "Non-VMFR restore should not modify PVC",
			hasVMFRAnnotation:        false,
			restoreName:              "regular-restore",
			backupName:               "backup-20250101",
			pvcName:                  "my-app-pvc",
			expectedGenerateNamePrefix: "",
			expectedName:             "my-app-pvc",
			expectedLabels:           map[string]string{},
		},
		{
			name:                     "VMFR restore with very long names should truncate prefix",
			hasVMFRAnnotation:        true,
			restoreName:              "vmfr-long-restore",
			backupName:               "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy",
			pvcName:                  "application-database-persistent-volume-claim-for-postgresql-primary-instance-with-high-availability-configuration-and-automated-backup-scheduling",
			expectedGenerateNamePrefix: "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy-application-database-persistent-volume-claim-for-postgresql-primary-instance-wit-",
			expectedName:             "", // Name should be cleared when using GenerateName
			expectedLabels: map[string]string{
				VMFRBackupLabel:       "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy",
				VMFROriginalNameLabel: "application-database-persistent-volume-claim-for-postgresql-primary-instance-with-high-availability-configuration-and-automated-backup-scheduling",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test PVC
			pvc := &corev1API.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.pvcName,
				},
				Spec: corev1API.PersistentVolumeClaimSpec{
					AccessModes: []corev1API.PersistentVolumeAccessMode{corev1API.ReadWriteOnce},
				},
			}

			pvcUnstructured, err := toUnstructured(pvc)
			if err != nil {
				t.Fatalf("Failed to convert PVC to unstructured: %v", err)
			}

			// Create test restore
			restore := &velerov1api.Restore{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.restoreName,
				},
				Spec: velerov1api.RestoreSpec{
					BackupName: tt.backupName,
				},
			}

			if tt.hasVMFRAnnotation {
				restore.Annotations = map[string]string{
					VMFRRestoreAnnotation: "true",
				}
			}

			// Create plugin input
			input := &velero.RestoreItemActionExecuteInput{
				Item:    pvcUnstructured,
				Restore: restore,
			}

			// Execute plugin
			plugin := &VMFRRestorePlugin{
				Log: logrus.NewEntry(logrus.New()),
			}
			output, err := plugin.Execute(input)
			if err != nil {
				t.Fatalf("Plugin execution failed: %v", err)
			}

			// Verify results
			resultPVC := &corev1API.PersistentVolumeClaim{}
			updatedUnstructured, ok := output.UpdatedItem.(*unstructured.Unstructured)
			if !ok {
				t.Fatalf("Expected *unstructured.Unstructured, got %T", output.UpdatedItem)
			}
			err = fromUnstructured(updatedUnstructured, resultPVC)
			if err != nil {
				t.Fatalf("Failed to convert result to PVC: %v", err)
			}

			// Check PVC name and GenerateName
			if resultPVC.Name != tt.expectedName {
				t.Errorf("Expected PVC name %s, got %s", tt.expectedName, resultPVC.Name)
			}
			if resultPVC.GenerateName != tt.expectedGenerateNamePrefix {
				t.Errorf("Expected PVC GenerateName %s, got %s", tt.expectedGenerateNamePrefix, resultPVC.GenerateName)
			}

			// Check labels
			for expectedKey, expectedValue := range tt.expectedLabels {
				if actualValue, exists := resultPVC.Labels[expectedKey]; !exists || actualValue != expectedValue {
					t.Errorf("Expected label %s=%s, got %s=%s (exists: %v)", expectedKey, expectedValue, expectedKey, actualValue, exists)
				}
			}
		})
	}
}


// Helper functions for testing
func toUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	unstructuredObj := &unstructured.Unstructured{}
	unstructuredObj.Object = make(map[string]interface{})

	// Simple conversion for testing - in real usage, use proper serialization
	if pvc, ok := obj.(*corev1API.PersistentVolumeClaim); ok {
		unstructuredObj.SetName(pvc.Name)
		unstructuredObj.SetNamespace(pvc.Namespace)
		unstructuredObj.SetKind("PersistentVolumeClaim")
		unstructuredObj.SetAPIVersion("v1")
		return unstructuredObj, nil
	}

	return nil, fmt.Errorf("unsupported object type")
}

func fromUnstructured(u *unstructured.Unstructured, obj interface{}) error {
	// Simple conversion for testing
	if pvc, ok := obj.(*corev1API.PersistentVolumeClaim); ok {
		pvc.Name = u.GetName()
		pvc.GenerateName = u.GetGenerateName()
		pvc.Namespace = u.GetNamespace()
		pvc.Labels = u.GetLabels()
		return nil
	}

	return fmt.Errorf("unsupported object type")
}

func TestMakeGenerateName(t *testing.T) {
	tests := []struct {
		name         string
		backupName   string
		originalName string
		expectedLen  int
		description  string
	}{
		{
			name:         "normal length names",
			backupName:   "backup-20250101",
			originalName: "my-app-pvc",
			expectedLen:  len("backup-20250101-my-app-pvc-"),
			description:  "Should not truncate normal length names",
		},
		{
			name:         "empty names",
			backupName:   "",
			originalName: "",
			expectedLen:  len("--"),
			description:  "Should handle empty names gracefully",
		},
		{
			name:         "single character names",
			backupName:   "b",
			originalName: "p",
			expectedLen:  len("b-p-"),
			description:  "Should handle single character names",
		},
		{
			name:         "very long names requiring truncation",
			backupName:   "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy",
			originalName: "application-database-persistent-volume-claim-for-postgresql-primary-instance-with-high-availability-configuration-and-automated-backup-scheduling",
			expectedLen:  validation.DNS1123SubdomainMaxLength - 5,
			description:  "Should truncate long names to fit within Kubernetes limits",
		},
		{
			name:         "names that end with dash after truncation",
			backupName:   "backup-with-many-dashes-at-end-----",
			originalName: "pvc-with-dashes----",
			expectedLen:  len("backup-with-many-dashes-at-end-----pvc-with-dashes----"),
			description:  "Should handle names with trailing dashes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.MakeGenerateName(tt.backupName, tt.originalName)

			// Verify length constraints
			maxAllowed := validation.DNS1123SubdomainMaxLength - 5
			if len(result) > maxAllowed {
				t.Errorf("Generated name length %d exceeds maximum allowed %d", len(result), maxAllowed)
			}

			// Verify it ends with a dash
			if result != "" && result[len(result)-1] != '-' {
				t.Errorf("Generated name should end with dash, got: %s", result)
			}

			// For normal length names, verify no truncation occurred
			if tt.name == "normal length names" && len(result) != tt.expectedLen {
				t.Errorf("Expected length %d for normal names, got %d", tt.expectedLen, len(result))
			}

			// For very long names, verify truncation occurred
			if tt.name == "very long names requiring truncation" {
				if len(result) != tt.expectedLen {
					t.Errorf("Expected truncated length %d, got %d", tt.expectedLen, len(result))
				}
				if result[len(result)-1:] != "-" {
					t.Errorf("Truncated name should end with dash")
				}
			}

			// Verify basic structure for non-empty inputs
			if tt.backupName != "" && tt.originalName != "" {
				if len(result) < len(tt.backupName)+1 { // At least backupName + separator
					t.Errorf("Result should contain at least the backup name and separator")
				}
			}

			t.Logf("Input: backup='%s', original='%s' -> Result: '%s' (len=%d)",
				tt.backupName, tt.originalName, result, len(result))
		})
	}
}