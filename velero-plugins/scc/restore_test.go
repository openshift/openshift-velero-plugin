package scc

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"securitycontextconstraints"}}, actual)
}

func TestExecuteWithNamespaceMapping(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name             string
		users            []string
		namespaceMapping map[string]string
		expectedUsers    []string
	}{
		{
			name:             "Service account user namespace swap",
			users:            []string{"system:serviceaccount:old-ns:my-sa"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expectedUsers:    []string{"system:serviceaccount:new-ns:my-sa"},
		},
		{
			name:             "Regular user no swap",
			users:            []string{"regular-user"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expectedUsers:    []string{"regular-user"},
		},
		{
			name:             "Multiple users mixed",
			users:            []string{"regular-user", "system:serviceaccount:old-ns:my-sa", "system:serviceaccount:other-ns:other-sa"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expectedUsers:    []string{"regular-user", "system:serviceaccount:new-ns:my-sa", "system:serviceaccount:other-ns:other-sa"},
		},
		{
			name:             "No namespace mapping",
			users:            []string{"system:serviceaccount:old-ns:my-sa"},
			namespaceMapping: map[string]string{},
			expectedUsers:    []string{"system:serviceaccount:old-ns:my-sa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "security.openshift.io/v1",
					"kind":       "SecurityContextConstraints",
					"metadata": map[string]interface{}{
						"name": "test-scc",
					},
					"users": tt.users,
				},
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item: item,
				Restore: &velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						NamespaceMapping: tt.namespaceMapping,
					},
				},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)

			// Extract users from the output
			outputObj := output.UpdatedItem.(*unstructured.Unstructured).Object
			outputUsers, ok := outputObj["users"].([]interface{})
			require.True(t, ok)

			// Convert to string slice for comparison
			actualUsers := make([]string, len(outputUsers))
			for i, u := range outputUsers {
				actualUsers[i] = u.(string)
			}

			assert.Equal(t, tt.expectedUsers, actualUsers)
		})
	}
}
