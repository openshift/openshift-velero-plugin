package statefulset

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vm "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	v1Apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"statefulsets.apps"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		statefulset v1Apps.StatefulSet
		registryInit    string
		registryCont	string
		temp             string
	}{
		"Swapping": {
			statefulset: v1Apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "statefulsetBackup",
						common.RestoreRegistryHostname: "statefulsetRestore",
					},
				},
				Spec: v1Apps.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
							Containers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
						},
					},
				},
			},
			registryInit: "statefulsetRestore/test",
			registryCont: "statefulsetRestore/test",
			temp: "",
		},
		"NoSwapping": {
			statefulset: v1Apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "",
						common.RestoreRegistryHostname: "",
					},
				},
				Spec: v1Apps.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
							Containers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
						},
					},
				},
			},
			registryCont: "statefulsetBackup/test",
			registryInit: "statefulsetBackup/test",
		},
		"EmptyRegistry1": {
			statefulset: v1Apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "statefulsetBackup",
						common.RestoreRegistryHostname: "",
					},
				},
				Spec: v1Apps.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
							Containers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
						},
					},
				},
			},
			registryCont: "statefulsetBackup/test",
			registryInit: "statefulsetBackup/test",
		},

		"EmptyRegistry2": {
			statefulset: v1Apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "",
						common.RestoreRegistryHostname: "statefulsetRestore",
					},
				},
				Spec: v1Apps.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
							Containers: []corev1.Container{
								{Image: "statefulsetBackup/test"},
							},
						},
					},
				},
			},
			registryCont: "statefulsetBackup/test",
			registryInit: "statefulsetBackup/test",
		},

		"NamespaceSwapping": {
			statefulset: v1Apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "statefulsetBackup",
						common.RestoreRegistryHostname: "statefulsetRestore",
					},
				},
				Spec: v1Apps.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{Image: "statefulsetBackup/enabled/test"},
							},
							Containers: []corev1.Container{
								{Image: "statefulsetBackup/enabled/test"},
							},
						},
					},
				},
			},
			registryCont: "statefulsetRestore/newNameSpace/test",
			registryInit: "statefulsetRestore/newNameSpace/test",
			temp: "enabled",
		},

		"NamespaceOpenShiftSwapping": {
			statefulset: v1Apps.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname:  "statefulsetBackup",
						common.RestoreRegistryHostname: "statefulsetRestore",
					},
				},
				Spec: v1Apps.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							InitContainers: []corev1.Container{
								{Image: "statefulsetBackup/openshift/test@test"},
							},
							Containers: []corev1.Container{
								{Image: "statefulsetBackup/openshift/test@test"},
							},
						},
					},
				},
			},
			registryCont: "statefulsetRestore/openshift/test",
			registryInit: "statefulsetRestore/openshift/test",
			temp: "",
		},

	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			statefulRec, _ := json.Marshal(tc.statefulset)
			json.Unmarshal(statefulRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							tc.temp: "newNameSpace",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			statefulset := v1Apps.StatefulSet{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &statefulset)

			if statefulset.Spec.Template.Spec.Containers[0].Image != tc.registryCont {
				t.Fatalf("Expected: %v, Got: %v", tc.registryCont, statefulset.Spec.Template.Spec.Containers[0].Image)
			}
			if statefulset.Spec.Template.Spec.InitContainers[0].Image != tc.registryInit {
				t.Fatalf("Expected: %v, Got: %v", tc.registryInit, statefulset.Spec.Template.Spec.InitContainers[0].Image)
			}
		})
	}
}
