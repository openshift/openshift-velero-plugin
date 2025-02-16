package nonadmin

import (
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePluginNonAdmin{Log: test.NewLogger()}

	expectedResources := []string{
		"nonadminbackups.oadp.openshift.io",
		"nonadminrestores.oadp.openshift.io",
		"nonadminbackupstoragelocations.oadp.openshift.io"}

	selectedResources, err := restorePlugin.AppliesTo()
	require.NoError(t, err)

	assert.Equal(t, expectedResources, selectedResources.IncludedResources)
}

func TestExecuteSkipsRestore(t *testing.T) {
	restorePlugin := &RestorePluginNonAdmin{Log: logrus.New()}

	tests := []struct {
		name       string
		apiVersion string
		kind       string
		shouldSkip bool
	}{
		{
			name:       "Skip NonAdminBackup",
			apiVersion: GroupOADP + "/v1alpha1",
			kind:       KindNonAdminBackup,
			shouldSkip: true,
		},
		{
			name:       "Skip NonAdminRestore",
			apiVersion: GroupOADP + "/v1alpha1",
			kind:       KindNonAdminRestore,
			shouldSkip: true,
		},
		{
			name:       "Skip NonAdminBackupStorageLocation",
			apiVersion: GroupOADP + "/v1alpha1",
			kind:       KindNonAdminBackupStorageLocation,
			shouldSkip: true,
		},
		{
			name:       "Don't skip unrelated kind",
			apiVersion: "apps/v1",
			kind:       "Deployment",
			shouldSkip: false,
		},
		{
			name:       "Don't skip unrelated group",
			apiVersion: "customgroup.io/v1",
			kind:       KindNonAdminBackup,
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			item := &unstructured.Unstructured{}
			item.SetAPIVersion(tt.apiVersion)
			item.SetKind(tt.kind)
			item.SetNamespace("test-ns")
			item.SetName("test-resource")

			input := &velero.RestoreItemActionExecuteInput{
				Item: item,
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldSkip, output.SkipRestore)
		})
	}
}
