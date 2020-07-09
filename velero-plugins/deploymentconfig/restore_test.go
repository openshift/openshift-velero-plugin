// unit tests for restore.go in deployment

package deploymentconfig

import (
	"testing"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
        //appsv1API "k8s.io/api/apps/v1"
	OSappsv1API "github.com/openshift/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"encoding/json"
	"reflect"
        velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	//"fmt"
	//"k8s.io/apimachinery/pkg/runtime"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"deploymentconfigs"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := map[string]struct{
		deploymentConfig OSappsv1API.DeploymentConfig
		exp	         OSappsv1API.DeploymentConfig
	}{
		"1": {
			deploymentConfig: OSappsv1API.DeploymentConfig {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "foo",
						"openshift.io/restore-registry-hostname": "bar",
					},
					Namespace: "old-namespace",
				},
				Spec: OSappsv1API.DeploymentConfigSpec {
					Template: &apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "foo/cat"},
							},
						},
					},
					Triggers: OSappsv1API.DeploymentTriggerPolicies{
						OSappsv1API.DeploymentTriggerPolicy{
							ImageChangeParams: &OSappsv1API.DeploymentTriggerImageChangeParams{
								From: apiv1.ObjectReference{
									Namespace: "old-namespace",
								},
							},
						},
						OSappsv1API.DeploymentTriggerPolicy{
							ImageChangeParams: &OSappsv1API.DeploymentTriggerImageChangeParams{
								From: apiv1.ObjectReference{
									Namespace: "old-namespace-2",
								},
							},
						},
					},
				},
			},
			exp: OSappsv1API.DeploymentConfig {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "foo",
						"openshift.io/restore-registry-hostname": "bar",
					},
					Namespace: "old-namespace",
				},
                                Spec: OSappsv1API.DeploymentConfigSpec {
                                        Template: &apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "bar/cat"},
                                                        },
                                                },
                                        },
					Triggers: OSappsv1API.DeploymentTriggerPolicies{
						OSappsv1API.DeploymentTriggerPolicy{
							ImageChangeParams: &OSappsv1API.DeploymentTriggerImageChangeParams{
								From: apiv1.ObjectReference{
									Namespace: "new-namespace",
								},
							},
						},
						OSappsv1API.DeploymentTriggerPolicy{
							ImageChangeParams: &OSappsv1API.DeploymentTriggerImageChangeParams{
								From: apiv1.ObjectReference{
									Namespace: "new-namespace",
								},
							},
						},
					},
                                },
                        },
		},
	}


	for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			deploymentConfigRec, _ := json.Marshal(tc.deploymentConfig) // Marshal it to JSON
			json.Unmarshal(deploymentConfigRec, &out) // Unmarshal into the proper format
			item.SetUnstructuredContent(out) // Set unstructured object
			restore := velerov1.Restore{
				Spec: velerov1.RestoreSpec{
					NamespaceMapping: map[string]string{
						"old-namespace": "new-namespace",
						"old-namespace-2": "new-namespace-2",
					},
				},
			}
			input := &velero.RestoreItemActionExecuteInput{Item: &item, Restore: &restore}

			output, _ := restorePlugin.Execute(input)

			deploymentConfigOut := OSappsv1API.DeploymentConfig{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &deploymentConfigOut)

			if !reflect.DeepEqual(deploymentConfigOut, tc.exp) {
                                t.Fatalf("expected: %v, got: %v", tc.exp, deploymentConfigOut)
                        }
		})
        }
}

