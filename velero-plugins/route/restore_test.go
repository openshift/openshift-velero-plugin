package route

import (
	"testing"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/heptio/velero/pkg/plugin/velero"
	routev1API "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"routes"}}, actual)
}

func TestReplaceSubdomain(t *testing.T) {
	t.Run("Execute restore action on route", func(t *testing.T) {
		route := &routev1API.Route{
			Spec: routev1API.RouteSpec{Host: "name.subdomain1.example.com"},
		}

		routeUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&route)
		require.NoError(t, err)

		input := &velero.RestoreItemActionExecuteInput{
			Item:           &unstructured.Unstructured{Object: routeUnstructured},
			ItemFromBackup: &unstructured.Unstructured{Object: routeUnstructured},
			Restore:        nil,
		}

		subdomain := "subdomain2.example.com"

		output := replaceSubdomain(input.Item, route, subdomain)

		var resRoute routev1API.Route

		err = runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &resRoute)
		require.NoError(t, err)

		assert.Equal(t, "name.subdomain2.example.com", resRoute.Spec.Host)
	})
}
