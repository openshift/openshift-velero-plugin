package imagetag

import (
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"imagetags"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	var out map[string]interface{}
	item := unstructured.Unstructured{}
	item.SetUnstructuredContent(out)

	input := velero.RestoreItemActionExecuteInput{Item: &item,
	}

	output, _ := restorePlugin.Execute(&input)

	if output.SkipRestore != true {
		t.Fatalf("expected: %v, got: %v", true, output.SkipRestore)
	}
}