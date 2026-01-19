package pod

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRestorePlugin_podHasRestoreHooks(t *testing.T) {
	type fields struct {
		Log logrus.FieldLogger
	}
	type args struct {
		pod     corev1API.Pod
		restore velerov1.Restore
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "pod has no restore hooks via annotations, nor via restore hook spec",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1",
						Annotations: map[string]string{},
					},
				},
				restore: velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						Hooks: velerov1.RestoreHooks{
							Resources: []velerov1.RestoreResourceHookSpec{},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
			fields: fields{
				Log: logrus.WithField("plugin", "restore-hooks"),
			},
		},
		{
			name: "pod has restore hooks via annotations, empty command",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1",
						Annotations: map[string]string{
							common.PostRestoreHookAnnotation: "",
						},
					},
				},
				restore: velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						Hooks: velerov1.RestoreHooks{
							Resources: []velerov1.RestoreResourceHookSpec{},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			fields: fields{
				Log: logrus.WithField("plugin", "restore-hooks"),
			},
		},
		{
			name: "pod has restore hooks via annotations, with echo command",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1",
						Annotations: map[string]string{
							common.PostRestoreHookAnnotation: "echo 'hello'",
						},
					},
				},
				restore: velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						Hooks: velerov1.RestoreHooks{
							Resources: []velerov1.RestoreResourceHookSpec{},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			fields: fields{
				Log: logrus.WithField("plugin", "restore-hooks"),
			},
		},
		{
			name: "pod has restore hooks via restore spec using namespace and specified exec command",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1"},
				},
				restore: velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						Hooks: velerov1.RestoreHooks{
							Resources: []velerov1.RestoreResourceHookSpec{
								{
									Name:               "hook1",
									IncludedNamespaces: []string{"ns-1"},
									PostHooks: []velerov1.RestoreResourceHook{
										{
											Exec: &velerov1.ExecRestoreHook{
												Command: []string{"echo", "hello"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want:    true,
			wantErr: false,
			fields: fields{
				Log: logrus.WithField("plugin", "restore-hooks"),
			},
		},
		{
			name: "pod has restore hooks via restore spec but no PostHooks so actually has no restore hooks to run",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1"},
				},
				restore: velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						Hooks: velerov1.RestoreHooks{
							Resources: []velerov1.RestoreResourceHookSpec{
								{
									Name:               "hook1",
									IncludedNamespaces: []string{"ns-1"},
								},
							},
						},
					},
				},
			},
			want:    false,
			wantErr: false,
			fields: fields{
				Log: logrus.WithField("plugin", "restore-hooks"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &RestorePlugin{
				Log: tt.fields.Log,
			}
			got, err := PodHasRestoreHooks(tt.args.pod, &tt.args.restore, p.Log)
			if (err != nil) != tt.wantErr {
				t.Errorf("RestorePlugin.podHasRestoreHooks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RestorePlugin.podHasRestoreHooks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"pods"}}, actual)
}

func TestPodHasVolumesToBackUp(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1API.Pod
		want bool
	}{
		{
			name: "pod with no volumes",
			pod: corev1API.Pod{
				Spec: corev1API.PodSpec{
					Volumes: []corev1API.Volume{},
				},
			},
			want: false,
		},
		{
			name: "pod with PVC volume",
			pod: corev1API.Pod{
				Spec: corev1API.PodSpec{
					Volumes: []corev1API.Volume{
						{
							Name: "pvc-volume",
							VolumeSource: corev1API.VolumeSource{
								PersistentVolumeClaim: &corev1API.PersistentVolumeClaimVolumeSource{
									ClaimName: "test-pvc",
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "pod with configmap volume only",
			pod: corev1API.Pod{
				Spec: corev1API.PodSpec{
					Volumes: []corev1API.Volume{
						{
							Name: "config-volume",
							VolumeSource: corev1API.VolumeSource{
								ConfigMap: &corev1API.ConfigMapVolumeSource{
									LocalObjectReference: corev1API.LocalObjectReference{
										Name: "test-configmap",
									},
								},
							},
						},
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PodHasVolumesToBackUp(tt.pod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestPodHasRestoreHookAnnotations(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1API.Pod
		want bool
	}{
		{
			name: "pod with no annotations",
			pod: corev1API.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: nil,
				},
			},
			want: false,
		},
		{
			name: "pod with post restore hook annotation",
			pod: corev1API.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.PostRestoreHookAnnotation: "echo 'hello'",
					},
				},
			},
			want: true,
		},
		{
			name: "pod with init container restore hook annotation",
			pod: corev1API.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.InitContainerRestoreHookAnnotation: "init-container",
					},
				},
			},
			want: true,
		},
		{
			name: "pod with unrelated annotations",
			pod: corev1API.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"some-other-annotation": "value",
					},
				},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := test.NewLogger()
			got := PodHasRestoreHookAnnotations(tt.pod, log)
			assert.Equal(t, tt.want, got)
		})
	}
}

// Note: The following functions are not tested here due to their dependencies:
// - Execute(): Requires mocking of multiple dependencies including clients, secrets, namespaces
// - GetOCPVersion(): Requires mocking openshift.GetClusterVersion() which depends on external cluster state
// - UpdateWaitForPullSecrets(): Requires mocking openshift functions that depend on external cluster state
// These functions would typically be tested in integration tests or with extensive mocking frameworks.
