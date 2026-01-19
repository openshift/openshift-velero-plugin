package configmap

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"configmaps"}}, actual)
}

func TestExecute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name              string
		annotations       map[string]string
		expectSkipRestore bool
	}{
		{
			name:              "ConfigMap with no annotations",
			annotations:       nil,
			expectSkipRestore: false,
		},
		{
			name:              "ConfigMap with empty annotations",
			annotations:       map[string]string{},
			expectSkipRestore: false,
		},
		{
			name: "ConfigMap with skip annotation set to true",
			annotations: map[string]string{
				common.SkipBuildConfigConfigMapRestore: "true",
			},
			expectSkipRestore: true,
		},
		{
			name: "ConfigMap with skip annotation set to false",
			annotations: map[string]string{
				common.SkipBuildConfigConfigMapRestore: "false",
			},
			expectSkipRestore: false,
		},
		{
			name: "ConfigMap with skip annotation set to invalid value",
			annotations: map[string]string{
				common.SkipBuildConfigConfigMapRestore: "invalid",
			},
			expectSkipRestore: false,
		},
		{
			name: "ConfigMap with other annotations",
			annotations: map[string]string{
				"some-other-annotation": "value",
			},
			expectSkipRestore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{}
			item.SetAPIVersion("v1")
			item.SetKind("ConfigMap")
			item.SetNamespace("test-ns")
			item.SetName("test-configmap")
			if tt.annotations != nil {
				item.SetAnnotations(tt.annotations)
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item: item,
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectSkipRestore, output.SkipRestore)
		})
	}
}
