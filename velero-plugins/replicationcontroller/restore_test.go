package replicationcontroller

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"replicationcontrollers"}}, actual)
}

func TestExecuteSkipsRCOwnedByDeploymentConfig(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name              string
		ownerReferences   []metav1.OwnerReference
		annotations       map[string]string
		expectSkipRestore bool
	}{
		{
			name:              "RC with no owner references",
			ownerReferences:   nil,
			expectSkipRestore: false,
		},
		{
			name: "RC owned by DeploymentConfig without paused annotation",
			ownerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.openshift.io/v1",
					Kind:       "DeploymentConfig",
					Name:       "test-dc",
					UID:        "12345",
				},
			},
			expectSkipRestore: true,
		},
		{
			name: "RC owned by DeploymentConfig with paused annotation",
			ownerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps.openshift.io/v1",
					Kind:       "DeploymentConfig",
					Name:       "test-dc",
					UID:        "12345",
				},
			},
			annotations: map[string]string{
				common.PausedOwnerRef: "true",
			},
			expectSkipRestore: false,
		},
		{
			name: "RC owned by other resource",
			ownerReferences: []metav1.OwnerReference{
				{
					APIVersion: "apps/v1",
					Kind:       "Deployment",
					Name:       "test-deployment",
					UID:        "12345",
				},
			},
			expectSkipRestore: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ReplicationController",
					"metadata": map[string]interface{}{
						"name":      "test-rc",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "test-container",
										"image": "test-image:latest",
									},
								},
							},
						},
					},
				},
			}
			if tt.annotations != nil {
				item.SetAnnotations(tt.annotations)
			}

			itemFromBackup := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ReplicationController",
					"metadata": map[string]interface{}{
						"name":      "test-rc",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"template": map[string]interface{}{
							"spec": map[string]interface{}{
								"containers": []interface{}{
									map[string]interface{}{
										"name":  "test-container",
										"image": "test-image:latest",
									},
								},
							},
						},
					},
				},
			}
			if tt.ownerReferences != nil {
				itemFromBackup.SetOwnerReferences(tt.ownerReferences)
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item:           item,
				ItemFromBackup: itemFromBackup,
				Restore: &velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						NamespaceMapping: map[string]string{},
					},
				},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			assert.Equal(t, tt.expectSkipRestore, output.SkipRestore)
		})
	}
}
