package persistentvolume

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"persistentvolumes"}}, actual)
}

func TestExecuteForNonMigration(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolume")
	item.SetName("test-pv")

	// Test non-migration restore
	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.False(t, output.SkipRestore)
	// Item should be returned as-is for non-migration
	assert.Equal(t, item, output.UpdatedItem)
}

func TestExecuteSkipsSnapshotPVOnStageRestore(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolume")
	item.SetName("test-pv")
	item.SetAnnotations(map[string]string{
		common.MigrateTypeAnnotation:       common.PvCopyAction,
		common.MigrateCopyMethodAnnotation: common.PvSnapshotCopyMethod,
	})

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					common.StageRestoreLabel:            "true",
				},
				Annotations: map[string]string{
					common.StageOrFinalMigrationAnnotation: common.StageMigration,
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.True(t, output.SkipRestore)
}

func TestExecuteSetsStorageClassForPvCopy(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name                     string
		annotations              map[string]string
		expectStorageClassUpdate bool
		expectedStorageClass     string
	}{
		{
			name: "PV copy with storage class annotation",
			annotations: map[string]string{
				common.MigrateTypeAnnotation:         common.PvCopyAction,
				common.MigrateStorageClassAnnotation: "new-storage-class",
			},
			expectStorageClassUpdate: true,
			expectedStorageClass:     "new-storage-class",
		},
		{
			name: "PV without copy action",
			annotations: map[string]string{
				common.MigrateStorageClassAnnotation: "new-storage-class",
			},
			expectStorageClassUpdate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "PersistentVolume",
					"metadata": map[string]interface{}{
						"name":        "test-pv",
						"annotations": tt.annotations,
					},
					"spec": map[string]interface{}{},
				},
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item: item,
				Restore: &velerov1.Restore{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
						},
					},
				},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)

			outputObj := output.UpdatedItem.(*unstructured.Unstructured).Object
			spec, ok := outputObj["spec"].(map[string]interface{})
			require.True(t, ok)

			if tt.expectStorageClassUpdate {
				assert.Equal(t, tt.expectedStorageClass, spec["storageClassName"])
			} else {
				_, hasStorageClass := spec["storageClassName"]
				assert.False(t, hasStorageClass)
			}
		})
	}
}
