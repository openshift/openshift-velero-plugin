package service

import (
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"services"}}, actual)
}

func TestExecuteClearsExternalIPsForLoadBalancer(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	tests := []struct {
		name                     string
		serviceType              corev1.ServiceType
		externalIPs              []string
		expectExternalIPsCleared bool
	}{
		{
			name:                     "LoadBalancer service - clears ExternalIPs",
			serviceType:              corev1.ServiceTypeLoadBalancer,
			externalIPs:              []string{"1.2.3.4", "5.6.7.8"},
			expectExternalIPsCleared: true,
		},
		{
			name:                     "ClusterIP service - keeps ExternalIPs",
			serviceType:              corev1.ServiceTypeClusterIP,
			externalIPs:              []string{"1.2.3.4", "5.6.7.8"},
			expectExternalIPsCleared: false,
		},
		{
			name:                     "NodePort service - keeps ExternalIPs",
			serviceType:              corev1.ServiceTypeNodePort,
			externalIPs:              []string{"1.2.3.4", "5.6.7.8"},
			expectExternalIPsCleared: false,
		},
		{
			name:                     "ExternalName service - keeps ExternalIPs",
			serviceType:              corev1.ServiceTypeExternalName,
			externalIPs:              []string{"1.2.3.4", "5.6.7.8"},
			expectExternalIPsCleared: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name":      "test-service",
						"namespace": "test-ns",
					},
					"spec": map[string]interface{}{
						"type":        string(tt.serviceType),
						"externalIPs": tt.externalIPs,
					},
				},
			}

			input := &velero.RestoreItemActionExecuteInput{
				Item:    item,
				Restore: &velerov1.Restore{},
			}

			output, err := restorePlugin.Execute(input)
			require.NoError(t, err)

			// Extract spec from the output
			outputObj := output.UpdatedItem.(*unstructured.Unstructured).Object
			spec, ok := outputObj["spec"].(map[string]interface{})
			require.True(t, ok)

			if tt.expectExternalIPsCleared {
				// ExternalIPs should be nil for LoadBalancer
				assert.Nil(t, spec["externalIPs"])
			} else {
				// ExternalIPs should be preserved for other types
				externalIPs, ok := spec["externalIPs"].([]interface{})
				if ok {
					assert.Len(t, externalIPs, len(tt.externalIPs))
				}
			}
		})
	}
}
