package scc

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	apisecurity "github.com/openshift/api/security/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vm "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
	"strings"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"securitycontextconstraints"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		scc       apisecurity.SecurityContextConstraints
		want      []string
		namespace string
	}{
		"WithNameSpace": {
			scc:       apisecurity.SecurityContextConstraints{
				Users: []string{"role:serviceaccount:namespace:serviceaccountname"}},
			want: []string{"role:serviceaccount:namespace:testNameSpace"},
		},
		"WithNoNameSpace": {
			scc:       apisecurity.SecurityContextConstraints{
				Users: []string{"role:serviceaccount:namespace:"}},
			want: []string{"role:serviceaccount:namespace:"},
		},
		"WithIncorrectServiceAccount": {
			scc:       apisecurity.SecurityContextConstraints{
				Users: []string{"role:service:namespace:test"}},
			want: []string{"role:serviceaccount:namespace:test"},
		},
		"WithIncorrectUser": {
			scc:       apisecurity.SecurityContextConstraints{
				Users: []string{"role:serviceaccount"}},
			want: []string{"role:serviceaccount"},
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			sccRec, _ := json.Marshal(tc.scc)
			json.Unmarshal(sccRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							"namespace": "testNameSpace",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			scc := apisecurity.SecurityContextConstraints{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &scc)

			for _, user := range scc.Users {
				splitUsername := strings.Split(user, ":")

				if reflect.DeepEqual(tc.want, splitUsername) {
					t.Fatalf("Expected: %v, Got: %v", tc.want, splitUsername)
				}
			}
		})
	}
}
