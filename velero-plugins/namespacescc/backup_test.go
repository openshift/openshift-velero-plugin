package namespacescc

import (
	"reflect"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	apisecurity "github.com/openshift/api/security/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

func TestBackupPluginAppliesTo(t *testing.T) {
	backupPlugin := &BackupPlugin{Log: test.NewLogger()}
	actual, err := backupPlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"serviceaccounts"}}, actual)
}

func Test_stashNamespaceSCCAnnotations(t *testing.T) {
	tests := []struct {
		name                 string
		saAnnotations        map[string]string
		namespaceAnnotations map[string]string
		wantAnnotations      map[string]string
		wantStashed          bool
	}{
		{
			name:                 "namespace has no SCC annotations, nothing stashed",
			saAnnotations:        nil,
			namespaceAnnotations: map[string]string{},
			wantAnnotations:      nil,
			wantStashed:          false,
		},
		{
			name:          "namespace has all 3 SCC annotations, all stashed onto nil SA annotations",
			saAnnotations: nil,
			namespaceAnnotations: map[string]string{
				apisecurity.UIDRangeAnnotation:           "1000700000/10000",
				apisecurity.SupplementalGroupsAnnotation: "1000700000/10000",
				apisecurity.MCSAnnotation:                "s0:c26,c5",
			},
			wantAnnotations: map[string]string{
				"oadp.openshift.io/backup-ns-scc-uid-range":           "1000700000/10000",
				"oadp.openshift.io/backup-ns-scc-supplemental-groups": "1000700000/10000",
				"oadp.openshift.io/backup-ns-scc-mcs":                 "s0:c26,c5",
			},
			wantStashed: true,
		},
		{
			name: "existing SA annotations are preserved alongside stashed ones",
			saAnnotations: map[string]string{
				"kubernetes.io/service-account.name": "default",
			},
			namespaceAnnotations: map[string]string{
				apisecurity.UIDRangeAnnotation: "1000700000/10000",
			},
			wantAnnotations: map[string]string{
				"kubernetes.io/service-account.name":        "default",
				"oadp.openshift.io/backup-ns-scc-uid-range": "1000700000/10000",
			},
			wantStashed: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAnnotations, gotStashed := stashNamespaceSCCAnnotations(tt.saAnnotations, tt.namespaceAnnotations)
			if gotStashed != tt.wantStashed {
				t.Errorf("stashNamespaceSCCAnnotations() stashed = %v, want %v", gotStashed, tt.wantStashed)
			}
			if !reflect.DeepEqual(gotAnnotations, tt.wantAnnotations) {
				t.Errorf("stashNamespaceSCCAnnotations() annotations = %v, want %v", gotAnnotations, tt.wantAnnotations)
			}
		})
	}
}
