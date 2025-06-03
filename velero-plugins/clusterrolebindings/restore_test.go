package clusterrolebindings

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"clusterrolebinding.authorization.openshift.io"}}, actual)
}

// Note: Execute() functionality is tested through:
// - rolebindings/restore_test.go tests the namespace mapping helper functions:
//   - SwapSubjectNamespaces(): Updates subject namespaces based on namespace mapping
//   - SwapUserNamesNamespaces(): Updates UserNames with service account namespace format
//   - SwapGroupNamesNamespaces(): Updates GroupNames with system:serviceaccounts namespace format
// These same functions are used by ClusterRoleBindings Execute() method.
