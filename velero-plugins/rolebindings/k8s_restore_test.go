package rolebindings

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestK8sRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &K8sRestorePlugin{Log: test.NewLogger()}

	expectedResources := []string{"rolebindings"}

	selectedResources, err := restorePlugin.AppliesTo()
	require.NoError(t, err)

	assert.Equal(t, expectedResources, selectedResources.IncludedResources)
}

func TestK8sExecuteSystemRoleBindings(t *testing.T) {
	restorePlugin := &K8sRestorePlugin{Log: logrus.New()}

	tests := []struct {
		name       string
		rbName     string
		shouldSkip bool
	}{
		{
			name:       "Skip system:image-pullers",
			rbName:     "system:image-pullers",
			shouldSkip: true,
		},
		{
			name:       "Skip system:image-builders",
			rbName:     "system:image-builders",
			shouldSkip: true,
		},
		{
			name:       "Skip system:deployers",
			rbName:     "system:deployers",
			shouldSkip: true,
		},
		{
			name:       "Don't skip regular rolebinding",
			rbName:     "my-custom-rolebinding",
			shouldSkip: false,
		},
		{
			name:       "Don't skip rolebinding with system: prefix but not in list",
			rbName:     "system:custom-role",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			roleBinding := rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "rbac.authorization.k8s.io/v1",
					Kind:       "RoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.rbName,
					Namespace: "test-namespace",
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "test-role",
				},
			}

			unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&roleBinding)
			require.NoError(t, err)

			item := &unstructured.Unstructured{Object: unstructuredObj}

			input := &velero.RestoreItemActionExecuteInput{
				Item: item,
				Restore: &velerov1.Restore{
					Spec: velerov1.RestoreSpec{
						NamespaceMapping: map[string]string{},
					},
				},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)
			assert.Equal(t, tt.shouldSkip, output.SkipRestore)
		})
	}
}
