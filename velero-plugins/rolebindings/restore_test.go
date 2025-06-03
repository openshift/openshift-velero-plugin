package rolebindings

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"rolebinding.authorization.openshift.io"}}, actual)
}

// Note: Execute() functionality is tested through the helper functions below:
// - SwapSubjectNamespaces(): Updates subject namespaces based on namespace mapping
// - SwapUserNamesNamespaces(): Updates UserNames with service account namespace format
// - SwapGroupNamesNamespaces(): Updates GroupNames with system:serviceaccounts namespace format
// These functions handle the core logic of the Execute() method.

func TestSwapSubjectNamespaces(t *testing.T) {
	tests := []struct {
		name             string
		subjects         []corev1.ObjectReference
		namespaceMapping map[string]string
		expected         []corev1.ObjectReference
	}{
		{
			name: "Simple namespace swap",
			subjects: []corev1.ObjectReference{
				{
					Kind:      "ServiceAccount",
					Name:      "my-sa",
					Namespace: "old-ns",
				},
			},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{
					Kind:      "ServiceAccount",
					Name:      "my-sa",
					Namespace: "new-ns",
				},
			},
		},
		{
			name: "System group serviceaccounts namespace swap",
			subjects: []corev1.ObjectReference{
				{
					Kind: "SystemGroup",
					Name: "system:serviceaccounts:old-ns",
				},
			},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{
					Kind: "SystemGroup",
					Name: "system:serviceaccounts:new-ns",
				},
			},
		},
		{
			name: "No mapping exists",
			subjects: []corev1.ObjectReference{
				{
					Kind:      "ServiceAccount",
					Name:      "my-sa",
					Namespace: "old-ns",
				},
			},
			namespaceMapping: map[string]string{"other-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{
					Kind:      "ServiceAccount",
					Name:      "my-sa",
					Namespace: "old-ns",
				},
			},
		},
		{
			name: "Subject without namespace",
			subjects: []corev1.ObjectReference{
				{
					Kind: "User",
					Name: "my-user",
				},
			},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected: []corev1.ObjectReference{
				{
					Kind: "User",
					Name: "my-user",
				},
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
			name:             "Service account username swap",
			userNames:        []string{"system:serviceaccount:old-ns:my-sa"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"system:serviceaccount:new-ns:my-sa"},
		},
		{
			name:             "Regular username no swap",
			userNames:        []string{"regular-user"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"regular-user"},
		},
		{
			name:             "No mapping for namespace",
			userNames:        []string{"system:serviceaccount:old-ns:my-sa"},
			namespaceMapping: map[string]string{"other-ns": "new-ns"},
			expected:         []string{"system:serviceaccount:old-ns:my-sa"},
		},
		{
			name:             "Multiple usernames mixed",
			userNames:        []string{"regular-user", "system:serviceaccount:old-ns:my-sa", "system:serviceaccount:other-ns:other-sa"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"regular-user", "system:serviceaccount:new-ns:my-sa", "system:serviceaccount:other-ns:other-sa"},
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
			name:             "Service accounts group swap",
			groupNames:       []string{"system:serviceaccounts:old-ns"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"system:serviceaccounts:new-ns"},
		},
		{
			name:             "Regular group no swap",
			groupNames:       []string{"regular-group"},
			namespaceMapping: map[string]string{"old-ns": "new-ns"},
			expected:         []string{"regular-group"},
		},
		{
			name:             "No mapping for namespace",
			groupNames:       []string{"system:serviceaccounts:old-ns"},
			namespaceMapping: map[string]string{"other-ns": "new-ns"},
			expected:         []string{"system:serviceaccounts:old-ns"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SwapGroupNamesNamespaces(tt.groupNames, tt.namespaceMapping)
			assert.Equal(t, tt.expected, result)
		})
	}
}
