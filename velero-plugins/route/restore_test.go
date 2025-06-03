package route

import (
	"testing"

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"routes"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	tests := []struct {
		name         string
		route        *unstructured.Unstructured
		shouldModify bool
	}{
		{
			name: "Route with generated host should be stripped",
			route: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "route.openshift.io/v1",
					"kind":       "Route",
					"metadata": map[string]interface{}{
						"name":      "test-route",
						"namespace": "test-ns",
						"annotations": map[string]interface{}{
							"openshift.io/host.generated": "true",
						},
					},
					"spec": map[string]interface{}{
						"host": "test-route-test-ns.apps.example.com",
						"to": map[string]interface{}{
							"kind": "Service",
							"name": "test-service",
						},
					},
				},
			},
			shouldModify: true,
		},
		{
			name: "Route without generated host annotation should not be modified",
			route: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "route.openshift.io/v1",
					"kind":       "Route",
					"metadata": map[string]interface{}{
						"name":      "test-route",
						"namespace": "test-ns",
						"annotations": map[string]interface{}{
							"some-other-annotation": "value",
						},
					},
					"spec": map[string]interface{}{
						"host": "custom.example.com",
						"to": map[string]interface{}{
							"kind": "Service",
							"name": "test-service",
						},
					},
				},
			},
			shouldModify: false,
		},
		{
			name: "Route with generated host set to false should not be modified",
			route: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "route.openshift.io/v1",
					"kind":       "Route",
					"metadata": map[string]interface{}{
						"name":      "test-route",
						"namespace": "test-ns",
						"annotations": map[string]interface{}{
							"openshift.io/host.generated": "false",
						},
					},
					"spec": map[string]interface{}{
						"host": "custom.example.com",
						"to": map[string]interface{}{
							"kind": "Service",
							"name": "test-service",
						},
					},
				},
			},
			shouldModify: false,
		},
		{
			name: "Route without annotations should not be modified",
			route: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "route.openshift.io/v1",
					"kind":       "Route",
					"metadata": map[string]interface{}{
						"name":      "test-route",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"host": "custom.example.com",
						"to": map[string]interface{}{
							"kind": "Service",
							"name": "test-service",
						},
					},
				},
			},
			shouldModify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			restorePlugin := &RestorePlugin{Log: test.NewLogger()}

			input := &velero.RestoreItemActionExecuteInput{
				Item: tt.route,
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			require.NotNil(t, output)

			if tt.shouldModify {
				// Check that the host was stripped
				spec, ok := output.UpdatedItem.UnstructuredContent()["spec"].(map[string]interface{})
				require.True(t, ok)
				host, exists := spec["host"]
				// The host field should either not exist or be an empty string
				if exists {
					assert.Equal(t, "", host, "Host should be empty string")
				}
			} else {
				// Check that the output item was not modified
				assert.Equal(t, input.Item, output.UpdatedItem)
			}
		})
	}
}

// Note: json.Marshal and json.Unmarshal error handling is not tested here as they are
// unlikely to fail with the structured objects we're using. The errors are logged
// but not returned, making them difficult to test without mocking.
