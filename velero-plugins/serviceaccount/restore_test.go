package serviceaccount

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"serviceaccounts"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		serviceAccount corev1API.ServiceAccount
		wantS          int
		wantI          int
	}{

		"NoMatches": {
			serviceAccount: corev1API.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oadp-operator"},
				Secrets: []corev1API.ObjectReference{{
					Name: "oadp-operator-token-vld4b"}},
				ImagePullSecrets: []corev1API.LocalObjectReference{{
					Name: "oadp-operator-token-v1d4b"}},
			},
			wantS: 1,
			wantI: 1,
		},

		"WithMatches": {
			serviceAccount: corev1API.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oadp-operator"},
				Secrets: []corev1API.ObjectReference{{
					Name: "oadp-operator-dockercfg-"}},
				ImagePullSecrets: []corev1API.LocalObjectReference{{
					Name: "oadp-operator-dockercfg-"}},
			},
			wantS: 0,
			wantI: 0,
		},

		"SomeMatches": {
			serviceAccount: corev1API.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oadp-operator"},
				Secrets: []corev1API.ObjectReference{{
					Name: "oadp-operator-dockercfg-"},
					{
						Name: "oadp-operator-token",
					},
				},
				ImagePullSecrets: []corev1API.LocalObjectReference{{
					Name: "oadp-operator-dockercfg-"}},
			},
			wantS: 1,
			wantI: 0,
		},

		"SomeMatches2": {
			serviceAccount: corev1API.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oadp-operator"},
				Secrets: []corev1API.ObjectReference{{
					Name: "oadp-operator-dockercfg-"},
					{
						Name: "oadp-operator-token",
					},
				},
				ImagePullSecrets: []corev1API.LocalObjectReference{{
					Name: "oadp-operator-dockercfg-"}, {
					Name: "oadp-operator",
				}},
			},
			wantS: 1,
			wantI: 1,
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			serviceRec, _ := json.Marshal(tc.serviceAccount)
			json.Unmarshal(serviceRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item}
			output, _ := restorePlugin.Execute(&input)

			serviceAccount := corev1API.ServiceAccount{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &serviceAccount)

			if len(serviceAccount.Secrets) != tc.wantS && len(serviceAccount.ImagePullSecrets) != tc.wantI {
				t.Fatalf("Expected: %v Secrets and %v ImagePullSecrets, got: %v Secrets and %v ImagePullSecrets",
					tc.wantS, tc.wantI, len(serviceAccount.Secrets), len(serviceAccount.ImagePullSecrets))
			}
		})
	}
}
