package serviceaccount

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"serviceaccounts"}}, actual)
}

func TestExecuteRemovesDockercfgSecrets(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name                     string
		serviceAccountName       string
		secrets                  []corev1.ObjectReference
		imagePullSecrets         []corev1.LocalObjectReference
		expectedSecrets          []corev1.ObjectReference
		expectedImagePullSecrets []corev1.LocalObjectReference
	}{
		{
			name:               "Remove dockercfg secret from secrets",
			serviceAccountName: "my-sa",
			secrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
				{Name: "my-sa-dockercfg-abc123"},
				{Name: "another-secret"},
			},
			imagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
			},
			expectedSecrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
				{Name: "another-secret"},
			},
			expectedImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
			},
		},
		{
			name:               "Remove dockercfg secret from image pull secrets",
			serviceAccountName: "test-sa",
			secrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
			},
			imagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
				{Name: "test-sa-dockercfg-xyz789"},
				{Name: "another-image-pull-secret"},
			},
			expectedSecrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
			},
			expectedImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
				{Name: "another-image-pull-secret"},
			},
		},
		{
			name:               "No dockercfg secrets to remove",
			serviceAccountName: "my-sa",
			secrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
				{Name: "another-secret"},
			},
			imagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
			},
			expectedSecrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
				{Name: "another-secret"},
			},
			expectedImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
			},
		},
		{
			name:               "Remove from both secrets and image pull secrets",
			serviceAccountName: "app-sa",
			secrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
				{Name: "app-sa-dockercfg-abc123"},
			},
			imagePullSecrets: []corev1.LocalObjectReference{
				{Name: "app-sa-dockercfg-xyz789"},
				{Name: "image-pull-secret"},
			},
			expectedSecrets: []corev1.ObjectReference{
				{Name: "regular-secret"},
			},
			expectedImagePullSecrets: []corev1.LocalObjectReference{
				{Name: "image-pull-secret"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert to unstructured format
			secrets := make([]interface{}, len(tt.secrets))
			for i, s := range tt.secrets {
				secrets[i] = map[string]interface{}{"name": s.Name}
			}

			imagePullSecrets := make([]interface{}, len(tt.imagePullSecrets))
			for i, s := range tt.imagePullSecrets {
				imagePullSecrets[i] = map[string]interface{}{"name": s.Name}
			}

			item := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ServiceAccount",
					"metadata": map[string]interface{}{
						"name":      tt.serviceAccountName,
						"namespace": "test-ns",
					},
					"secrets":          secrets,
					"imagePullSecrets": imagePullSecrets,
				},
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item:    item,
				Restore: &velerov1.Restore{},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)

			// Extract and verify results
			outputObj := output.UpdatedItem.(*unstructured.Unstructured).Object

			// Check secrets
			outputSecrets, ok := outputObj["secrets"].([]interface{})
			require.True(t, ok)
			assert.Len(t, outputSecrets, len(tt.expectedSecrets))

			// Check imagePullSecrets
			outputImagePullSecrets, ok := outputObj["imagePullSecrets"].([]interface{})
			require.True(t, ok)
			assert.Len(t, outputImagePullSecrets, len(tt.expectedImagePullSecrets))
		})
	}
}
