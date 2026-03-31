package job

import (
	"testing"

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"jobs"}}, actual)
}

func TestExecuteSkipsJobOwnedByCronJob(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name              string
		ownerReferences   []metav1.OwnerReference
		expectSkipRestore bool
	}{
		{
			name:              "Job with no owner references",
			ownerReferences:   nil,
			expectSkipRestore: false,
		},
		{
			name: "Job owned by CronJob",
			ownerReferences: []metav1.OwnerReference{
				{
					APIVersion: "batch/v1",
					Kind:       "CronJob",
					Name:       "test-cronjob",
					UID:        "12345",
				},
			},
			expectSkipRestore: true,
		},
		{
			name: "Job owned by other resource",
			ownerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "test-deployment",
					UID:        "12345",
				},
			},
			expectSkipRestore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{}
			item.SetAPIVersion("batch/v1")
			item.SetKind("Job")
			item.SetNamespace("test-ns")
			item.SetName("test-job")

			itemFromBackup := &unstructured.Unstructured{}
			itemFromBackup.SetAPIVersion("batch/v1")
			itemFromBackup.SetKind("Job")
			itemFromBackup.SetNamespace("test-ns")
			itemFromBackup.SetName("test-job")
			if tt.ownerReferences != nil {
				itemFromBackup.SetOwnerReferences(tt.ownerReferences)
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item:           item,
				ItemFromBackup: itemFromBackup,
				Restore:        &velerov1.Restore{},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectSkipRestore, output.SkipRestore)
		})
	}
}
