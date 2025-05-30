package buildconfig

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"buildconfigs"}}, actual)
}

// Note: Execute() functionality is tested through:
// - build.UpdateCommonSpec() tests in build/restore_test.go which handles:
//   - Docker image reference swapping from backup to restore registry
//   - Pull/push secret updates to match destination cluster
// - common.GetSrcAndDestRegistryInfo() tests for registry info extraction
// The updateSecretsAndDockerRefs() helper function wraps these tested components.
