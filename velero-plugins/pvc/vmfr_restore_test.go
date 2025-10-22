package pvc

import (
	"context"
	"regexp"
	"strings"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	velerov1api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1API "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestVMFRRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &VMFRRestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"persistentvolumeclaims"}}, actual)
}

func TestVMFRRestorePlugin_Execute(t *testing.T) {
	// Setup envtest environment
	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	require.NoError(t, err)
	defer testEnv.Stop()

	// Initialize clients using envtest config
	coreClient, err := clients.CoreClientFromConfig(cfg)
	require.NoError(t, err)

	tests := []struct {
		name                 string
		hasVMFRAnnotation    bool
		restoreName          string
		backupName           string
		pvcName              string
		pvcNamespace         string
		pvcAlreadyExists     bool // Simulate existing PVC with same name
		expectedNamePattern  string // Regex pattern since suffix is random
		expectedMaxLen       int
		shouldHaveSuffix     bool
		expectedAnnotations  map[string]string
		expectedLabels       map[string]string
	}{
		{
			name:              "First VMFR restore should keep original name and mark as primary",
			hasVMFRAnnotation: true,
			restoreName:       "vmfr-my-instance-backup-20250101",
			backupName:        "backup-20250101",
			pvcName:           "my-app-pvc",
			pvcNamespace:      "test-ns",
			pvcAlreadyExists:  false, // First restore
			expectedNamePattern: `^my-app-pvc$`, // Keep original name
			expectedMaxLen:      253,
			shouldHaveSuffix:    false,
			expectedAnnotations: map[string]string{
				VMFRBackupAnnotation: "backup-20250101",
			},
			expectedLabels: map[string]string{
				VMFRBackupAnnotation: "backup-20250101",
				VMFRPrimaryLabel:     "true",
			},
		},
		{
			name:              "Subsequent VMFR restore should rename with suffix when PVC exists",
			hasVMFRAnnotation: true,
			restoreName:       "vmfr-my-instance-backup-20250115",
			backupName:        "backup-20250115",
			pvcName:           "my-app-pvc",
			pvcNamespace:      "test-ns",
			pvcAlreadyExists:  true, // PVC already exists from first restore
			expectedNamePattern: `^backup-20250115-[a-z0-9]{10}$`, // Renamed
			expectedMaxLen:      253,
			shouldHaveSuffix:    true,
			expectedAnnotations: map[string]string{
				VMFRBackupAnnotation:       "backup-20250115",
				VMFROriginalNameAnnotation: "my-app-pvc",
			},
			expectedLabels: map[string]string{
				VMFRBackupAnnotation: "backup-20250115",
				VMFRPrimaryLabel:     "false",
			},
		},
		{
			name:                 "Non-VMFR restore should not modify PVC",
			hasVMFRAnnotation:    false,
			restoreName:          "regular-restore",
			backupName:           "backup-20250101",
			pvcName:              "my-app-pvc",
			pvcNamespace:         "test-ns",
			pvcAlreadyExists:     false,
			expectedNamePattern:  `^my-app-pvc$`,
			expectedMaxLen:       253,
			shouldHaveSuffix:     false,
			expectedAnnotations:  map[string]string{},
			expectedLabels:       map[string]string{},
		},
		{
			name:              "VMFR restore with very long backup name truncates when renaming",
			hasVMFRAnnotation: true,
			restoreName:       "vmfr-long-restore",
			backupName:        "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy",
			pvcName:           "my-app-pvc",
			pvcNamespace:      "test-ns",
			pvcAlreadyExists:  true, // Force renaming
			// Should have truncated backup name (242 chars max) and 10-char suffix
			expectedNamePattern: `^backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy-[a-z0-9]{10}$`,
			expectedMaxLen:      253,
			shouldHaveSuffix:    true,
			expectedAnnotations: map[string]string{
				VMFRBackupAnnotation:       "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy",
				VMFROriginalNameAnnotation: "my-app-pvc",
			},
			expectedLabels: map[string]string{
				VMFRBackupAnnotation: "backup-for-production-environment-disaster-recovery-scenario-2024-12-01-full-system-backup-including-all-persistent-data-and-configuration-files-with-retention-policy",
				VMFRPrimaryLabel:     "false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create namespace for this test
			_, err := coreClient.Namespaces().Create(context.Background(), &corev1API.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.pvcNamespace,
				},
			}, metav1.CreateOptions{})
			if k8serrors.IsAlreadyExists(err) {
				t.Logf("namespace %s already exists, which is fine for testing", tt.pvcNamespace)
				err = nil
			}
			require.NoError(t, err)

			// Clean up any existing PVC with the same name from previous tests
			existingPVC, err := coreClient.PersistentVolumeClaims(tt.pvcNamespace).Get(context.Background(), tt.pvcName, metav1.GetOptions{})
			if err == nil {
				// Remove finalizers to allow immediate deletion in envtest
				existingPVC.Finalizers = nil
				_, err = coreClient.PersistentVolumeClaims(tt.pvcNamespace).Update(context.Background(), existingPVC, metav1.UpdateOptions{})
				require.NoError(t, err)

				// Delete the PVC (should be immediate with finalizers removed)
				err = coreClient.PersistentVolumeClaims(tt.pvcNamespace).Delete(context.Background(), tt.pvcName, metav1.DeleteOptions{})
				require.NoError(t, err)
			} else if !k8serrors.IsNotFound(err) {
				require.NoError(t, err)
			}

			// If test requires existing PVC, create it
			if tt.pvcAlreadyExists {
				_, err := coreClient.PersistentVolumeClaims(tt.pvcNamespace).Create(context.Background(), &corev1API.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Name:      tt.pvcName,
						Namespace: tt.pvcNamespace,
					},
					Spec: corev1API.PersistentVolumeClaimSpec{
						AccessModes: []corev1API.PersistentVolumeAccessMode{corev1API.ReadWriteOnce},
						Resources: corev1API.VolumeResourceRequirements{
							Requests: corev1API.ResourceList{
								corev1API.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				}, metav1.CreateOptions{})
				require.NoError(t, err)
			}

			// Create test PVC (from backup)
			pvc := &corev1API.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.pvcName,
					Namespace: tt.pvcNamespace,
				},
				Spec: corev1API.PersistentVolumeClaimSpec{
					AccessModes: []corev1API.PersistentVolumeAccessMode{corev1API.ReadWriteOnce},
					Resources: corev1API.VolumeResourceRequirements{
						Requests: corev1API.ResourceList{
							corev1API.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}

			// Convert PVC to unstructured for plugin input
			pvcMap := map[string]interface{}{
				"apiVersion": "v1",
				"kind":       "PersistentVolumeClaim",
				"metadata": map[string]interface{}{
					"name":      pvc.Name,
					"namespace": pvc.Namespace,
				},
				"spec": map[string]interface{}{
					"accessModes": []interface{}{"ReadWriteOnce"},
					"resources": map[string]interface{}{
						"requests": map[string]interface{}{
							"storage": "1Gi",
						},
					},
				},
			}
			pvcUnstructured := &unstructured.Unstructured{Object: pvcMap}

			// Create test restore
			restore := &velerov1api.Restore{
				ObjectMeta: metav1.ObjectMeta{
					Name: tt.restoreName,
				},
				Spec: velerov1api.RestoreSpec{
					BackupName:       tt.backupName,
					NamespaceMapping: map[string]string{},
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
				Log:        test.NewLogger(),
				CoreClient: coreClient,
			}
			output, err := plugin.Execute(input)
			require.NoError(t, err)

			// Verify results - convert unstructured back to PVC
			updatedUnstructured, ok := output.UpdatedItem.(*unstructured.Unstructured)
			require.True(t, ok, "Expected *unstructured.Unstructured, got %T", output.UpdatedItem)

			resultName := updatedUnstructured.GetName()
			resultLabels := updatedUnstructured.GetLabels()
			resultAnnotations := updatedUnstructured.GetAnnotations()
			resultGenerateName := updatedUnstructured.GetGenerateName()

			// Check PVC name matches pattern
			matched, err := regexp.MatchString(tt.expectedNamePattern, resultName)
			require.NoError(t, err, "Invalid regex pattern")
			assert.True(t, matched, "PVC name %s does not match expected pattern %s", resultName, tt.expectedNamePattern)

			// Check name length is within limit
			assert.LessOrEqual(t, len(resultName), tt.expectedMaxLen, "PVC name length exceeds maximum")

			// Check GenerateName is cleared
			assert.Empty(t, resultGenerateName, "Expected PVC GenerateName to be empty")

			// Check suffix is present for VMFR restores
			if tt.shouldHaveSuffix {
				parts := strings.Split(resultName, "-")
				require.GreaterOrEqual(t, len(parts), 2, "Expected PVC name to have suffix")
				suffix := parts[len(parts)-1]
				assert.Equal(t, 10, len(suffix), "Expected 10-character suffix")
				matched, _ := regexp.MatchString(`^[a-z0-9]{10}$`, suffix)
				assert.True(t, matched, "Expected alphanumeric 10-character suffix, got %s", suffix)
			}

			// Check annotations
			for expectedKey, expectedValue := range tt.expectedAnnotations {
				actualValue, exists := resultAnnotations[expectedKey]
				assert.True(t, exists, "Expected annotation %s to exist", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Annotation %s value mismatch", expectedKey)
			}

			// Check labels
			for expectedKey, expectedValue := range tt.expectedLabels {
				actualValue, exists := resultLabels[expectedKey]
				assert.True(t, exists, "Expected label %s to exist", expectedKey)
				assert.Equal(t, expectedValue, actualValue, "Label %s value mismatch", expectedKey)
			}
		})
	}
}