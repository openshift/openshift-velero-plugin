package secret

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"secrets"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		secret corev1API.Secret
		want   bool
	}{
		"NoAnnotation": {secret: corev1API.Secret{}, want: false},
		"WithAnnotation": {secret: corev1API.Secret{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				serviceOriginAnnotation: "value",
			},
		},
		}, want: true},
		"WrongAnnotation": {secret: corev1API.Secret{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"Wrong Annotation": "value",
			},
		},
		}, want: false},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			secretRec, _ := json.Marshal(tc.secret)
			json.Unmarshal(secretRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
			}
			output, _ := restorePlugin.Execute(&input)

			if tc.want != output.SkipRestore {
				t.Fatalf("expected: %v, got: %v", tc.want, output.SkipRestore)
			}
		})
	}
}
