package pod

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
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

func TestStripCNIAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		annotations         map[string]string
		expectedAnnotations map[string]string
	}{
		{
			name:                "pod with no annotations",
			annotations:         nil,
			expectedAnnotations: nil,
		},
		{
			name: "pod with all CNI annotations",
			annotations: map[string]string{
				common.OVNPodNetworksAnnotation:      `{"default":{"ip_addresses":["10.129.2.14/23"]}}`,
				common.MultusNetworkStatusAnnotation:  `[{"name":"ovn-kubernetes","interface":"eth0"}]`,
				common.MultusNetworksStatusAnnotation: `[{"name":"ovn-kubernetes","interface":"eth0"}]`,
				"app": "test",
			},
			expectedAnnotations: map[string]string{
				"app": "test",
			},
		},
		{
			name: "pod with only OVN-K annotation",
			annotations: map[string]string{
				common.OVNPodNetworksAnnotation: `{"default":{"ip_addresses":["10.129.2.14/23"]}}`,
				"app":                           "test",
			},
			expectedAnnotations: map[string]string{
				"app": "test",
			},
		},
		{
			name: "pod with no CNI annotations",
			annotations: map[string]string{
				"app":     "test",
				"version": "v1",
			},
			expectedAnnotations: map[string]string{
				"app":     "test",
				"version": "v1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pod := corev1API.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-pod",
					Namespace:   "test-ns",
					Annotations: tt.annotations,
				},
			}

			if pod.Annotations != nil {
				for _, annotation := range common.CNIAnnotationsToStrip {
					delete(pod.Annotations, annotation)
				}
			}

			assert.Equal(t, tt.expectedAnnotations, pod.Annotations)
		})
	}
}
