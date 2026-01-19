package deploymentconfig

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"deploymentconfigs"}}, actual)
}

// Note: Execute() functionality is tested through:
// - common.GetSrcAndDestRegistryInfo() tests for extracting registry info from annotations
// - common.SwapContainerImageRefs() tests for swapping image references from backup to restore registry
// - pod/restore_test.go tests PodHasVolumesToBackUp() and PodHasRestoreHooks() used for DC pod handling
// Additional DC-specific functionality:
// - Image change trigger namespace mapping (when namespace mapping is enabled)
// - Special pod restoration logic (setting replicas=0 when pods have volumes/hooks to prevent deletion)
// These would typically be tested in integration tests due to their complex dependencies.
