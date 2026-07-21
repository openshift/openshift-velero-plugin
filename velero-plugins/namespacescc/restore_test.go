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

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"serviceaccounts"}}, actual)
}

func Test_expectedSCCAnnotations(t *testing.T) {
	tests := []struct {
		name                string
		backedUpAnnotations map[string]string
		want                map[string]string
	}{
		{
			name:                "no bookkeeping annotations",
			backedUpAnnotations: map[string]string{"kubernetes.io/service-account.name": "default"},
			want:                map[string]string{},
		},
		{
			name: "bookkeeping annotations decoded back to real SCC annotation names",
			backedUpAnnotations: map[string]string{
				"oadp.openshift.io/backup-ns-scc-uid-range": "1000700000/10000",
				"oadp.openshift.io/backup-ns-scc-mcs":       "s0:c26,c5",
			},
			want: map[string]string{
				apisecurity.UIDRangeAnnotation: "1000700000/10000",
				apisecurity.MCSAnnotation:      "s0:c26,c5",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expectedSCCAnnotations(tt.backedUpAnnotations); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("expectedSCCAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_sccAnnotationMismatches(t *testing.T) {
	tests := []struct {
		name     string
		expected map[string]string
		actual   map[string]string
		want     []string
	}{
		{
			name: "matching ranges produce no mismatches",
			expected: map[string]string{
				apisecurity.UIDRangeAnnotation: "1000700000/10000",
			},
			actual: map[string]string{
				apisecurity.UIDRangeAnnotation: "1000700000/10000",
			},
			want: nil,
		},
		{
			name: "differing uid-range produces one mismatch",
			expected: map[string]string{
				apisecurity.UIDRangeAnnotation: "1000700000/10000",
			},
			actual: map[string]string{
				apisecurity.UIDRangeAnnotation: "1000710000/10000",
			},
			want: []string{
				`[namespacescc-restore] namespace "ns-1" annotation "openshift.io/sa.scc.uid-range" mismatch: backed up as "1000700000/10000", restored as "1000710000/10000". Workloads relying on the backed-up UID/GID range may have incorrect file ownership.`,
			},
		},
		{
			name: "annotation missing on actual namespace produces a mismatch",
			expected: map[string]string{
				apisecurity.MCSAnnotation: "s0:c26,c5",
			},
			actual: map[string]string{},
			want: []string{
				`[namespacescc-restore] namespace "ns-1" annotation "openshift.io/sa.scc.mcs" mismatch: backed up as "s0:c26,c5", restored as "". Workloads relying on the backed-up UID/GID range may have incorrect file ownership.`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sccAnnotationMismatches("ns-1", tt.expected, tt.actual)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("sccAnnotationMismatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stripSCCBookkeepingAnnotations(t *testing.T) {
	tests := []struct {
		name        string
		annotations map[string]string
		want        map[string]string
	}{
		{
			name:        "nil annotations stay nil",
			annotations: nil,
			want:        nil,
		},
		{
			name: "bookkeeping annotations removed, others preserved",
			annotations: map[string]string{
				"kubernetes.io/service-account.name":        "default",
				"oadp.openshift.io/backup-ns-scc-uid-range": "1000700000/10000",
			},
			want: map[string]string{
				"kubernetes.io/service-account.name": "default",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripSCCBookkeepingAnnotations(tt.annotations)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripSCCBookkeepingAnnotations() = %v, want %v", got, tt.want)
			}
		})
	}
}
