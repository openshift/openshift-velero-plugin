package pvc

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"persistentvolumeclaims"}}, actual)
}

func TestExecuteForNonMigration(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")

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

func TestExecuteReclaimSpaceAnnotationNonMigration(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{},
				Annotations: map[string]string{
					common.ReclaimSpaceScheduleAnnotation: "@weekly",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.False(t, output.SkipRestore)

	pvcOut := output.UpdatedItem.(*unstructured.Unstructured)
	annotations := pvcOut.GetAnnotations()
	assert.Equal(t, "@weekly", annotations[common.CSIAddonsReclaimSpaceScheduleAnnotation])
}

func TestExecuteReclaimSpaceAnnotationMigration(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{
					common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
				},
				Annotations: map[string]string{
					common.ReclaimSpaceScheduleAnnotation: "@daily",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.False(t, output.SkipRestore)

	pvcOut := output.UpdatedItem.(*unstructured.Unstructured)
	annotations := pvcOut.GetAnnotations()
	assert.Equal(t, "@daily", annotations[common.CSIAddonsReclaimSpaceScheduleAnnotation])
}

func TestExecuteNoReclaimSpaceAnnotation(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      map[string]string{},
				Annotations: nil,
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.False(t, output.SkipRestore)
	assert.Equal(t, item, output.UpdatedItem)
}

func TestExecuteReclaimSpacePreservesExistingAnnotations(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")
	item.SetAnnotations(map[string]string{
		"existing-annotation": "existing-value",
	})

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{},
				Annotations: map[string]string{
					common.ReclaimSpaceScheduleAnnotation: "@weekly",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)

	pvcOut := output.UpdatedItem.(*unstructured.Unstructured)
	annotations := pvcOut.GetAnnotations()
	assert.Equal(t, "@weekly", annotations[common.CSIAddonsReclaimSpaceScheduleAnnotation])
	assert.Equal(t, "existing-value", annotations["existing-annotation"])
}

func TestExecuteReclaimSpaceOnPVCWithNoAnnotations(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{},
				Annotations: map[string]string{
					common.ReclaimSpaceScheduleAnnotation: "@daily",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)

	pvcOut := output.UpdatedItem.(*unstructured.Unstructured)
	annotations := pvcOut.GetAnnotations()
	assert.Equal(t, "@daily", annotations[common.CSIAddonsReclaimSpaceScheduleAnnotation])
}

func TestExecuteEmptyReclaimSpaceScheduleIsNoop(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			ObjectMeta: metav1.ObjectMeta{
				Labels: map[string]string{},
				Annotations: map[string]string{
					common.ReclaimSpaceScheduleAnnotation: "",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.False(t, output.SkipRestore)
	assert.Equal(t, item, output.UpdatedItem)
}

func TestExecuteStageSnapshotSkipWithReclaimSpace(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")
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
					common.ReclaimSpaceScheduleAnnotation:  "@weekly",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.True(t, output.SkipRestore)
}

func TestExecuteSkipsSnapshotPVCOnStageRestore(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("v1")
	item.SetKind("PersistentVolumeClaim")
	item.SetNamespace("test-ns")
	item.SetName("test-pvc")
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
