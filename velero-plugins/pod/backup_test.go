package pod

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"github.com/vmware-tanzu/velero/pkg/restic"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	regularPod = corev1API.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "regular-pod",
				Namespace: "default",
			},
			Spec: corev1API.PodSpec{
				Containers: []corev1API.Container{
					{
						Name:  "container-1",
						Image: "image-1",
					},
				},
			},
		}
	regularPodMap, _ = runtime.DefaultUnstructuredConverter.ToUnstructured(&regularPod)
	buildPod = corev1API.Pod{	
			ObjectMeta: metav1.ObjectMeta{
				Name:      "build-pod",
				Namespace: "default",
				Labels: map[string]string{
					"openshift.io/build.name": "build-1",
				},
				Annotations: map[string]string{
					"openshift.io/build.name": "build-1",
				},
			},
			Spec: corev1API.PodSpec{
				Containers: []corev1API.Container{
					{
						Name:  "container-1",
						Image: "image-1",
					},
				},
			},
		}
	buildPodMap, _ = runtime.DefaultUnstructuredConverter.ToUnstructured(&buildPod)
)

func annotatedBuildPodMap() map[string]interface{} {
	annotatedBuildPod := buildPod.DeepCopy()
	annotatedBuildPod.Annotations[restic.VolumesToExcludeAnnotation] += buildPodVolumesToExclude
	podMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(&annotatedBuildPod)
	return podMap
}

func TestBackupPlugin_Execute(t *testing.T) {
	type fields struct {
		Log logrus.FieldLogger
	}
	type args struct {
		input  runtime.Unstructured
		backup *v1.Backup
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    runtime.Unstructured
		want1   []velero.ResourceIdentifier
		wantErr bool
	}{
		{
			name: "Normal pod backup no changes",
			fields: fields{
				Log: logrus.StandardLogger(),
			},
			args: args{
				input:  &unstructured.Unstructured{Object: regularPodMap},
				backup: nil,
			},
			want:    &unstructured.Unstructured{Object: regularPodMap},
			want1:   nil,
			wantErr: false,
		},
		{
			name: "Build pod contains volume exclusion annotation",
			fields: fields{
				Log: logrus.StandardLogger(),
			},
			args: args{
				input:  &unstructured.Unstructured{Object: buildPodMap},
				backup: nil,
			},
			want:    &unstructured.Unstructured{Object: annotatedBuildPodMap()},
			want1:   nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &BackupPlugin{
				Log: tt.fields.Log,
			}
			fmt.Printf("input: %v backup: %v", tt.args.input, tt.args.backup)
			got, got1, err := p.Execute(tt.args.input, tt.args.backup)
			if (err != nil) != tt.wantErr {
				t.Errorf("BackupPlugin.Execute() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			fmt.Printf("got: %v want: %v", got, tt.want)
			if !reflect.DeepEqual(got, tt.want) {
				fmt.Print(cmp.Diff(got, tt.want))
				t.Errorf("BackupPlugin.Execute() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("BackupPlugin.Execute() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
