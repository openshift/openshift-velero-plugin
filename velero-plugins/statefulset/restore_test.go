package statefulset

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"statefulsets.apps"}}, actual)
}

// Note: Execute() functionality is tested through:
// - common.GetSrcAndDestRegistryInfo() tests for extracting registry info from annotations
// - common.SwapContainerImageRefs() tests for swapping image references from backup to restore registry
// The Execute() method uses these tested components to update container and init container images.
