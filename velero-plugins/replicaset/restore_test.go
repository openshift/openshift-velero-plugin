package replicaset

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
	appsv1API "k8s.io/api/apps/v1"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"replicasets.apps"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		rs appsv1API.ReplicaSet
		want bool
	}{
		"WithDeployment": {
			rs: appsv1API.ReplicaSet{
				Spec: appsv1API.ReplicaSetSpec{
					Template: corev1API.PodTemplateSpec{
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
						{Kind: "Deployment"},
					},
				},
			},
			want: true,
		},
		"WithoutDeployment": {
			rs: appsv1API.ReplicaSet{
				Spec: appsv1API.ReplicaSetSpec{
					Template: corev1API.PodTemplateSpec{
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
						{Kind: "NoDeployment"},
					},
				},
			},
			want: false,
		},
		"WithSomeDeployment": {
			rs: appsv1API.ReplicaSet{
				Spec: appsv1API.ReplicaSetSpec{
					Template: corev1API.PodTemplateSpec{
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
						{Kind: "NoDeployment"},
						{Kind: "NoDeployment"},
						{Kind: "Deployment"},
					},
				},
			},
			want: true,
		},

		"WithoutAnyDeployment": {
			rs: appsv1API.ReplicaSet{
				Spec: appsv1API.ReplicaSetSpec{
					Template: corev1API.PodTemplateSpec{
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
						{Kind: "NoDeployment"},
						{Kind: "NoDeployment"},
						{Kind: "NoDeployment"},
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
			rcRec, _ := json.Marshal(tc.rs)
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