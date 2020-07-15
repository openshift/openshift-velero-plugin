// unit tests for restore.go in persistentVolume

package persistentvolume

import (
	"testing"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
        //appsv1API "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"encoding/json"
	"reflect"
        velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	//"fmt"
	//"k8s.io/apimachinery/pkg/runtime"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	//"fmt"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"persistentvolumes"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := map[string]struct{
		persistentVolume apiv1.PersistentVolume
		restore          velerov1.Restore
		exp	         apiv1.PersistentVolume
	}{
		"1": {
			persistentVolume: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
                        },
		},

		"2": {
			persistentVolume: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						apiv1.BetaStorageClassAnnotation: "",
					},
				},
				Spec: apiv1.PersistentVolumeSpec {
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                apiv1.BetaStorageClassAnnotation: "",
					},
				},
                                Spec: apiv1.PersistentVolumeSpec {
					StorageClassName: "storageClassName",
                                },
                        },
		},

		"3": {
			persistentVolume: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						apiv1.BetaStorageClassAnnotation: "not-empty",
					},
				},
				Spec: apiv1.PersistentVolumeSpec {
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                apiv1.BetaStorageClassAnnotation: "storageClassName",
					},
				},
                                Spec: apiv1.PersistentVolumeSpec {
					StorageClassName: "storageClassName",
                                },
                        },
		},

		"4": {
			persistentVolume: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						apiv1.BetaStorageClassAnnotation: "not-empty",
					},
				},
				Spec: apiv1.PersistentVolumeSpec {
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: "not-" + common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolume {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                apiv1.BetaStorageClassAnnotation: "not-empty",
					},
				},
                                Spec: apiv1.PersistentVolumeSpec {
                                },
                        },
		},

	}

	for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			persistentVolumeRec, _ := json.Marshal(tc.persistentVolume) // Marshal it to JSON
			json.Unmarshal(persistentVolumeRec, &out) // Unmarshal into the proper format
			item.SetUnstructuredContent(out) // Set unstructured object
			input := &velero.RestoreItemActionExecuteInput{Item: &item, Restore: &tc.restore}

			output, _ := restorePlugin.Execute(input)

			persistentVolumeOut := apiv1.PersistentVolume{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &persistentVolumeOut)

			if !reflect.DeepEqual(persistentVolumeOut, tc.exp) {
                                t.Fatalf("expected: \n%v, got: \n%v", tc.exp, persistentVolumeOut)
                        }
		})
        }
}

func str_ptr(str string) *string {
	return &str
}
