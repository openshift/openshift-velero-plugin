package pod

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_podHasRestoreHooks(t *testing.T) {
	type args struct {
		pod       corev1API.Pod
		resources []velerov1.RestoreResourceHookSpec
	}
	tests := []struct {
		name    string
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
				resources: []velerov1.RestoreResourceHookSpec{},
			},
			want:    false,
			wantErr: false,
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
				resources: []velerov1.RestoreResourceHookSpec{},
			},
			want:    true,
			wantErr: false,
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
				resources: []velerov1.RestoreResourceHookSpec{},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "pod has restore hooks via restore spec using namespace and specified exec command",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1",},

				},
				resources: []velerov1.RestoreResourceHookSpec{
					{
						Name: "hook1",
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
			want:    true,
			wantErr: false,
		},
		{
			name: "pod has restore hooks via restore spec but no PostHooks so actually has no restore hooks to run",
			args: args{
				pod: corev1API.Pod{
					ObjectMeta: metav1.ObjectMeta{Name: "pod-1", Namespace: "ns-1",},

				},
				resources: []velerov1.RestoreResourceHookSpec{
					{
						Name: "hook1",
						IncludedNamespaces: []string{"ns-1"},
					},
				},
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := podHasRestoreHooks(tt.args.pod, tt.args.resources)
			if (err != nil) != tt.wantErr {
				t.Errorf("podHasRestoreHooks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("podHasRestoreHooks() = %v, want %v", got, tt.want)
			}
		})
	}
}
