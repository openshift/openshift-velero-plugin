package imagetag

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"imagetags"}}, actual)
}

func TestExecuteAlwaysSkipsRestore(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	item := &unstructured.Unstructured{}
	item.SetAPIVersion("image.openshift.io/v1")
	item.SetKind("ImageTag")
	item.SetNamespace("test-ns")
	item.SetName("test-imagetag")

	input := &velero.RestoreItemActionExecuteInput{
		Item:    item,
		Restore: &velerov1.Restore{},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.True(t, output.SkipRestore)
}
