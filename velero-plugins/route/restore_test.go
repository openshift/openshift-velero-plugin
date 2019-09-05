package route

import (
	"testing"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/heptio/velero/pkg/plugin/velero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"routes"}}, actual)
}
