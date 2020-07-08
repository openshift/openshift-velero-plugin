package route

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	routev1API "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"routes"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		route routev1API.Route
		want  string
	}{
		"empty": {route: routev1API.Route{Spec:
		routev1API.RouteSpec{Host: ""},
		}, want: ""},
		"true": {route: routev1API.Route{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"openshift.io/host.generated": "true",
			},
		},
			Spec:
			routev1API.RouteSpec{Host: "test"},
		}, want: ""},
		"static": {route: routev1API.Route{ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				"openshift.io/host.generated": "static",
			},
		},
			Spec:
			routev1API.RouteSpec{Host: "static"},
		}, want: "static"},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			routeRec, _ := json.Marshal(tc.route)
			json.Unmarshal(routeRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
			}

			output, _ := restorePlugin.Execute(&input)

			route := routev1API.Route{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &route)

			if tc.want != route.Spec.Host {
				t.Log(route.Spec.Host)
				t.Fatalf("expected: %v, got: %v", tc.want, route.Spec.Host)
			}
		})
	}

}
