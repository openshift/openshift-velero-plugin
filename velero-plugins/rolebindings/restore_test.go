package rolebindings

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	apiauthorization "github.com/openshift/api/authorization/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	expectedResources := []string{"rolebinding.authorization.openshift.io"}

	selectedResources, err := restorePlugin.AppliesTo()
	require.NoError(t, err)

	assert.Equal(t, expectedResources, selectedResources.IncludedResources)
}

func TestExecuteSystemRoleBindings(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: logrus.New()}

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
			roleBinding := apiauthorization.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "authorization.openshift.io/v1",
					Kind:       "RoleBinding",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.rbName,
					Namespace: "test-namespace",
				},
				RoleRef: corev1.ObjectReference{
					Namespace: "test-namespace",
					Name:      "test-role",
				},
			}

			// Convert to unstructured
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

func TestExecuteNamespaceMapping(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: logrus.New()}

	roleBinding := apiauthorization.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "authorization.openshift.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-rolebinding",
			Namespace: "test-namespace",
		},
		RoleRef: corev1.ObjectReference{
			Namespace: "old-namespace",
			Name:      "test-role",
		},
		Subjects: []corev1.ObjectReference{
			{
				Kind:      "ServiceAccount",
				Namespace: "old-namespace",
				Name:      "test-sa",
			},
			{
				Kind: "Group",
				Name: "system:serviceaccounts:old-namespace",
			},
		},
		UserNames: []string{
			"system:serviceaccount:old-namespace:test-sa",
			"regular-user",
		},
		GroupNames: []string{
			"system:serviceaccounts:old-namespace",
			"regular-group",
		},
	}

	// Convert to unstructured
	unstructuredObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&roleBinding)
	require.NoError(t, err)

	item := &unstructured.Unstructured{Object: unstructuredObj}

	input := &velero.RestoreItemActionExecuteInput{
		Item: item,
		Restore: &velerov1.Restore{
			Spec: velerov1.RestoreSpec{
				NamespaceMapping: map[string]string{
					"old-namespace": "new-namespace",
				},
			},
		},
	}

	output, err := restorePlugin.Execute(input)
	require.NoError(t, err)
	assert.False(t, output.SkipRestore)

	// Convert output back to RoleBinding to verify changes
	restoredRB := apiauthorization.RoleBinding{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(output.UpdatedItem.UnstructuredContent(), &restoredRB)
	require.NoError(t, err)

	// Verify namespace mappings were applied
	assert.Equal(t, "new-namespace", restoredRB.RoleRef.Namespace)
	assert.Equal(t, "new-namespace", restoredRB.Subjects[0].Namespace)
	assert.Equal(t, "system:serviceaccounts:new-namespace", restoredRB.Subjects[1].Name)
	assert.Equal(t, "system:serviceaccount:new-namespace:test-sa", restoredRB.UserNames[0])
	assert.Equal(t, "regular-user", restoredRB.UserNames[1])
	assert.Equal(t, "system:serviceaccounts:new-namespace", restoredRB.GroupNames[0])
	assert.Equal(t, "regular-group", restoredRB.GroupNames[1])
}

func TestSwapSubjectNamespaces(t *testing.T) {
	tests := []struct {
		name             string
		subjects         []corev1.ObjectReference
		namespaceMapping map[string]string
		expected         []corev1.ObjectReference
	}{
		{
			name: "Swap namespace in subject",
			subjects: []corev1.ObjectReference{
				{Namespace: "old-ns", Name: "test-sa"},
			},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{Namespace: "new-ns", Name: "test-sa"},
			},
		},
		{
			name: "Swap namespace in system group",
			subjects: []corev1.ObjectReference{
				{Name: "system:serviceaccounts:old-ns"},
			},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{Name: "system:serviceaccounts:new-ns"},
			},
		},
		{
			name: "No swap when namespace not in mapping",
			subjects: []corev1.ObjectReference{
				{Namespace: "unmapped-ns", Name: "test-sa"},
			},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{Namespace: "unmapped-ns", Name: "test-sa"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SwapSubjectNamespaces(tt.subjects, tt.namespaceMapping)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSwapUserNamesNamespaces(t *testing.T) {
	tests := []struct {
		name             string
		userNames        []string
		namespaceMapping map[string]string
		expected         []string
	}{
		{
			name:             "Swap service account namespace",
			userNames:        []string{"system:serviceaccount:old-ns:test-sa"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"system:serviceaccount:new-ns:test-sa"},
		},
		{
			name:             "No swap for regular user",
			userNames:        []string{"regular-user"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"regular-user"},
		},
		{
			name:             "No swap when namespace not in mapping",
			userNames:        []string{"system:serviceaccount:unmapped-ns:test-sa"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"system:serviceaccount:unmapped-ns:test-sa"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SwapUserNamesNamespaces(tt.userNames, tt.namespaceMapping)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSwapGroupNamesNamespaces(t *testing.T) {
	tests := []struct {
		name             string
		groupNames       []string
		namespaceMapping map[string]string
		expected         []string
	}{
		{
			name:             "Swap service accounts group namespace",
			groupNames:       []string{"system:serviceaccounts:old-ns"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"system:serviceaccounts:new-ns"},
		},
		{
			name:             "No swap for regular group",
			groupNames:       []string{"regular-group"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"regular-group"},
		},
		{
			name:             "No swap when namespace not in mapping",
			groupNames:       []string{"system:serviceaccounts:unmapped-ns"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"system:serviceaccounts:unmapped-ns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SwapGroupNamesNamespaces(tt.groupNames, tt.namespaceMapping)
			assert.Equal(t, tt.expected, result)
		})
	}
}
