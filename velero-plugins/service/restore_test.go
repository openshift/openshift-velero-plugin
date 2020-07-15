package service

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"services"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		service corev1API.Service
		want    []string
	}{
		"WithoutLoadBalancer": {service: corev1API.Service{Spec: corev1API.ServiceSpec{
			Type:        "WithoutLoadBalancer",
			ExternalIPs: []string{"31.234.23.456"},
		},
		}, want: []string{"31.234.23.456"},
		},

		"WithLoadBalancer": {service: corev1API.Service{Spec: corev1API.ServiceSpec{
			Type:        corev1API.ServiceTypeLoadBalancer,
			ExternalIPs: []string{"31.234.23.456"},
		},
		}, want: nil,
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			serviceRec, _ := json.Marshal(tc.service)
			json.Unmarshal(serviceRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item}
			output, _ := restorePlugin.Execute(&input)

			service := corev1API.Service{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &service)

			if !reflect.DeepEqual(service.Spec.ExternalIPs, tc.want) {
				t.Fatalf("Expected: %v, got: %v", tc.want, service.Spec.ExternalIPs)
			}
		})
	}

}
