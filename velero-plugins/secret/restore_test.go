package secret

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"secrets"}}, actual)
}

func TestExecuteSkipsSecretWithOriginatingServiceAnnotation(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name              string
		annotations       map[string]string
		expectSkipRestore bool
	}{
		{
			name:              "Secret with no annotations",
			annotations:       nil,
			expectSkipRestore: false,
		},
		{
			name: "Secret with originating service annotation",
			annotations: map[string]string{
				serviceOriginAnnotation: "my-service",
			},
			expectSkipRestore: true,
		},
		{
			name: "Secret with other annotations",
			annotations: map[string]string{
				"some-other-annotation": "value",
			},
			expectSkipRestore: false,
		},
		{
			name: "Secret with mixed annotations including originating service",
			annotations: map[string]string{
				serviceOriginAnnotation: "my-service",
				"other-annotation":      "value",
			},
			expectSkipRestore: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{}
			item.SetAPIVersion("v1")
			item.SetKind("Secret")
			item.SetNamespace("test-ns")
			item.SetName("test-secret")
			if tt.annotations != nil {
				item.SetAnnotations(tt.annotations)
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item:    item,
				Restore: &velerov1.Restore{},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectSkipRestore, output.SkipRestore)
		})
	}
}
