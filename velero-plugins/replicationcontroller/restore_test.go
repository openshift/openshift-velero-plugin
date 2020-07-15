package replicationcontroller

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vm "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"replicationcontrollers"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		rc corev1API.ReplicationController
		want bool
	}{
		"WithDeploymentConfig": {
			rc: corev1API.ReplicationController{
				Spec: corev1API.ReplicationControllerSpec{
					Template: &corev1API.PodTemplateSpec{
						Spec: corev1API.PodSpec{
							InitContainers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
							Containers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "rcBackup",
						common.RestoreRegistryHostname: "rcRestore",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "DeploymentConfig"},
					},
				},
			},
			want: true,
		},

		"WithoutDeploymentConfig": {
			rc: corev1API.ReplicationController{
				Spec: corev1API.ReplicationControllerSpec{
					Template: &corev1API.PodTemplateSpec{
						Spec: corev1API.PodSpec{
							InitContainers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
							Containers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "rcBackup",
						common.RestoreRegistryHostname: "rcRestore",
					},
					OwnerReferences: []metav1.OwnerReference{
						{},
					},
				},
			},
			want: false,
		},

		"WithADeploymentConfig": {
			rc: corev1API.ReplicationController{
				Spec: corev1API.ReplicationControllerSpec{
					Template: &corev1API.PodTemplateSpec{
						Spec: corev1API.PodSpec{
							InitContainers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
							Containers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "rcBackup",
						common.RestoreRegistryHostname: "rcRestore",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "NoDeploymentConfig1"},
						{Kind: "NoDeploymentConfig2"},
						{Kind: "DeploymentConfig"},
					},
				},
			},
			want: true,
		},

		"WithNoDeploymentConfigs": {
			rc: corev1API.ReplicationController{
				Spec: corev1API.ReplicationControllerSpec{
					Template: &corev1API.PodTemplateSpec{
						Spec: corev1API.PodSpec{
							InitContainers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
							Containers: []corev1API.Container{
								{Image: "rcBackup/test"},
							},
						},
					},
				},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "rcBackup",
						common.RestoreRegistryHostname: "rcRestore",
					},
					OwnerReferences: []metav1.OwnerReference{
						{Kind: "NoDeploymentConfig1"},
						{Kind: "NoDeploymentConfig2"},
						{Kind: "NoDeploymentConfig3"},
					},
				},
			},
			want: false,
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			rcRec, _ := json.Marshal(tc.rc)
			json.Unmarshal(rcRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item, ItemFromBackup: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							"disable": "newNameSpace",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			if tc.want != output.SkipRestore {
				t.Fatalf("expected: %v, got: %v", tc.want, output.SkipRestore)
			}
		})
	}
}
