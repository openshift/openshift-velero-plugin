package horizontalpodautoscaler

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	appsv1API "github.com/openshift/api/apps/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"horizontalpodautoscalers"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	tests := []struct {
		name                   string
		hpa                    *unstructured.Unstructured
		expectedAPIVersion     string
		expectedScaleTargetRef v2beta1.CrossVersionObjectReference
		shouldModify           bool
	}{
		{
			name: "HPA with DeploymentConfig and v1 API version should be updated",
			hpa: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "autoscaling/v2beta1",
					"kind":       "HorizontalPodAutoscaler",
					"metadata": map[string]interface{}{
						"name":      "test-hpa",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"scaleTargetRef": map[string]interface{}{
							"apiVersion": "v1",
							"kind":       "DeploymentConfig",
							"name":       "test-dc",
						},
					},
				},
			},
			expectedAPIVersion: appsv1API.GroupVersion.String(),
			shouldModify:       true,
		},
		{
			name: "HPA with DeploymentConfig and apps.openshift.io/v1 API version should not be modified",
			hpa: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "autoscaling/v2beta1",
					"kind":       "HorizontalPodAutoscaler",
					"metadata": map[string]interface{}{
						"name":      "test-hpa",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"scaleTargetRef": map[string]interface{}{
							"apiVersion": "apps.openshift.io/v1",
							"kind":       "DeploymentConfig",
							"name":       "test-dc",
						},
					},
				},
			},
			expectedAPIVersion: "apps.openshift.io/v1",
			shouldModify:       false,
		},
		{
			name: "HPA with Deployment should not be modified",
			hpa: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "autoscaling/v2beta1",
					"kind":       "HorizontalPodAutoscaler",
					"metadata": map[string]interface{}{
						"name":      "test-hpa",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"scaleTargetRef": map[string]interface{}{
							"apiVersion": "apps/v1",
							"kind":       "Deployment",
							"name":       "test-deployment",
						},
					},
				},
			},
			expectedAPIVersion: "apps/v1",
			shouldModify:       false,
		},
		{
			name: "HPA without scaleTargetRef should not be modified",
			hpa: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "autoscaling/v2beta1",
					"kind":       "HorizontalPodAutoscaler",
					"metadata": map[string]interface{}{
						"name":      "test-hpa",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{},
				},
			},
			shouldModify: false,
		},
		{
			name: "HPA with invalid API version should return error",
			hpa: &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "autoscaling/v2beta1",
					"kind":       "HorizontalPodAutoscaler",
					"metadata": map[string]interface{}{
						"name":      "test-hpa",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"scaleTargetRef": map[string]interface{}{
							"apiVersion": "invalid/api/version/format",
							"kind":       "DeploymentConfig",
							"name":       "test-dc",
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
				Item: tt.hpa,
			}

			output, err := restorePlugin.Execute(input)

			if tt.name == "HPA with invalid API version should return error" {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, output)

			if tt.shouldModify {
				// Check that the output item was modified
				spec, ok := output.UpdatedItem.UnstructuredContent()["spec"].(map[string]interface{})
				require.True(t, ok)
				scaleTargetRef, ok := spec["scaleTargetRef"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, tt.expectedAPIVersion, scaleTargetRef["apiVersion"])
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
