// unit tests for restore.go in pvc

package pvc

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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"persistentvolumeclaims"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := map[string]struct{
		pvc	   apiv1.PersistentVolumeClaim
		restore    velerov1.Restore
		exp	   apiv1.PersistentVolumeClaim
	}{
		"1": {
			pvc: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.PVCSelectedNodeAnnotation: "someAnnotation",
					},
				},
				//Spec: apiv1.PersistentVolumeClaimSpec {
				//},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
				//	Annotations: map[string]string{
				//	},
				},
                                //Spec: apiv1.PersistentVolumeClaimSpec {
                                //},
                        },
		},

		"2": {
			pvc: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.PVCSelectedNodeAnnotation: "someAnnotation",
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						common.MigrateAccessModeAnnotation: "",
						apiv1.BetaStorageClassAnnotation: "",
					},
				},
				Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                common.MigrateAccessModeAnnotation: "",
                                                apiv1.BetaStorageClassAnnotation: "",
					},
				},
                                Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: nil,
					StorageClassName: str_ptr("storageClassName"),
                                },
                        },
		},

		"3": {
			pvc: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.PVCSelectedNodeAnnotation: "someAnnotation",
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						common.MigrateAccessModeAnnotation: "accessMode",
						apiv1.BetaStorageClassAnnotation: "",
					},
				},
				Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                common.MigrateAccessModeAnnotation: "accessMode",
                                                apiv1.BetaStorageClassAnnotation: "",
					},
				},
                                Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: nil,
					StorageClassName: str_ptr("storageClassName"),
					AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.PersistentVolumeAccessMode("accessMode")},
                                },
                        },
		},

		"4": {
			pvc: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.PVCSelectedNodeAnnotation: "someAnnotation",
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						common.MigrateAccessModeAnnotation: "accessMode",
						apiv1.BetaStorageClassAnnotation: "non-empty",
					},
				},
				Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                common.MigrateAccessModeAnnotation: "accessMode",
                                                apiv1.BetaStorageClassAnnotation: "storageClassName",
					},
				},
                                Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: nil,
					StorageClassName: str_ptr("storageClassName"),
					AccessModes: []apiv1.PersistentVolumeAccessMode{apiv1.PersistentVolumeAccessMode("accessMode")},
                                },
                        },
		},


		"5": {
			pvc: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.PVCSelectedNodeAnnotation: "someAnnotation",
						common.MigrateTypeAnnotation: "copy",
						common.MigrateStorageClassAnnotation: "storageClassName",
						common.MigrateAccessModeAnnotation: "accessMode",
						apiv1.BetaStorageClassAnnotation: "non-empty",
					},
				},
				Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
				},
			},
			restore: velerov1.Restore{
				ObjectMeta: metav1.ObjectMeta {
					Labels: map[string]string {
						common.MigrationApplicationLabelKey: "not-" + common.MigrationApplicationLabelValue,
					},
				},
			},
			exp: apiv1.PersistentVolumeClaim {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						common.PVCSelectedNodeAnnotation: "someAnnotation",
                                                common.MigrateTypeAnnotation: "copy",
                                                common.MigrateStorageClassAnnotation: "storageClassName",
                                                common.MigrateAccessModeAnnotation: "accessMode",
						apiv1.BetaStorageClassAnnotation: "non-empty",
					},
				},
                                Spec: apiv1.PersistentVolumeClaimSpec {
					Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
                                },
                        },
		},
	}

	for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			pvcRec, _ := json.Marshal(tc.pvc) // Marshal it to JSON
			json.Unmarshal(pvcRec, &out) // Unmarshal into the proper format
			item.SetUnstructuredContent(out) // Set unstructured object
			input := &velero.RestoreItemActionExecuteInput{Item: &item, Restore: &tc.restore}

			output, _ := restorePlugin.Execute(input)

			pvcOut := apiv1.PersistentVolumeClaim{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &pvcOut)

			if !reflect.DeepEqual(pvcOut, tc.exp) {
                                t.Fatalf("expected: \n%v, got: \n%v", tc.exp, pvcOut)
                        }
		})
        }
}

func str_ptr(str string) *string {
	return &str
}
