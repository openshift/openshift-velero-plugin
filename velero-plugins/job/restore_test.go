// unit tests for restore.go in job

package job

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
	batchv1API "k8s.io/api/batch/v1"
	//"fmt"
	//"k8s.io/apimachinery/pkg/runtime"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"jobs"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := map[string]struct{
		job	       batchv1API.Job
		jobFromBackup  batchv1API.Job
		exp	       batchv1API.Job
		skipRestore bool
	}{
		"1": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "backup-host/namespace-old/foo"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
                                                },
                                        },
                                },
                        },
			skipRestore: false,
		},

		"2": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "backup-host/namespace-old/foo"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
                                                },
					},
				},
			},
			skipRestore: false,
		},

		"3": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace/foo"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace/foo"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "restore-host/namespace/foo"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "restore-host/namespace/foo"},
							},
                                                },
					},
				},
			},
			skipRestore: false,
		},

		"4": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/namespace-old/foo"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "restore-host/namespace-new/foo"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "restore-host/namespace-new/foo"},
							},
                                                },
                                        },
                                },
                        },
			skipRestore: false,
		},

		"5": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host-2/namespace-old/foo"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host-2/namespace-old/foo"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "backup-host-2/namespace-old/foo"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host-2/namespace-old/foo"},
							},
                                                },
                                        },
                                },
                        },
			skipRestore: false,
		},

		"6": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/openshift/foo@bar"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/openshift/foo@bar"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "restore-host/openshift/foo"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "restore-host/openshift/foo"},
							},
                                                },
                                        },
                                },
                        },
			skipRestore: false,
		},


		"7": {
			job: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
				Spec: batchv1API.JobSpec {
					Template: apiv1.PodTemplateSpec {
						Spec: apiv1.PodSpec {
							Containers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/openshift/foo@bar"},
							},
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/openshift/foo@bar"},
							},
						},
					},
				},
			},
			jobFromBackup: batchv1API.Job {
                                ObjectMeta: metav1.ObjectMeta {
					OwnerReferences: []metav1.OwnerReference{
						metav1.OwnerReference{Kind: "CronJob"},
					},
				},
			},
			exp: batchv1API.Job {
				ObjectMeta: metav1.ObjectMeta {
					Annotations: map[string]string{
						"openshift.io/backup-registry-hostname": "backup-host",
						"openshift.io/restore-registry-hostname": "restore-host",
					},
				},
                                Spec: batchv1API.JobSpec {
                                        Template: apiv1.PodTemplateSpec {
                                                Spec: apiv1.PodSpec {
                                                        Containers: []apiv1.Container {
                                                                apiv1.Container{Image: "backup-host/openshift/foo@bar"},
                                                        },
							InitContainers: []apiv1.Container {
								apiv1.Container{Image: "backup-host/openshift/foo@bar"},
							},
                                                },
                                        },
                                },
                        },
			skipRestore: true,
		},

	}



	for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			jobRec, _ := json.Marshal(tc.job) // Marshal it to JSON
			json.Unmarshal(jobRec, &out) // Unmarshal into the proper format
			item.SetUnstructuredContent(out) // Set unstructured object

			restore := velerov1.Restore{
				Spec: velerov1.RestoreSpec{
					NamespaceMapping: map[string]string{
						"namespace-old": "namespace-new",
					},
				},
			}

			var out2 map[string]interface{}
			itemFromBackup := unstructured.Unstructured{}
			jobRec2, _ := json.Marshal(tc.jobFromBackup) // Marshal it to JSON
			json.Unmarshal(jobRec2, &out2) // Unmarshal into the proper format
			itemFromBackup.SetUnstructuredContent(out2) // Set unstructured object

			input := &velero.RestoreItemActionExecuteInput{Item: &item, ItemFromBackup: &itemFromBackup, Restore: &restore}

			output, _ := restorePlugin.Execute(input)

			jobOut := batchv1API.Job{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &jobOut)

			if tc.skipRestore != output.SkipRestore {
				t.Fatalf("expected SkipRestore: %v, got: %v", tc.skipRestore, output.SkipRestore)
			}

			if !reflect.DeepEqual(jobOut, tc.exp) {
                                t.Fatalf("expected: %v, got: %v", tc.exp, jobOut)
                        }
		})
        }
}

