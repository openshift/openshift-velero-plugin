package clusterrolebindings

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	apiauthorization "github.com/openshift/api/authorization/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	vm "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"clusterrolebinding.authorization.openshift.io"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		rolebinding apiauthorization.ClusterRoleBinding
		want        string
		wantSubject string
		wantUser    string
		wantGroup   string
	}{
		"RoleNamespaceSwap": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "oldRoleNamespace",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"",
				},
			},
			want: "newRoleNameSpace",
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			rcRec, _ := json.Marshal(tc.rolebinding)
			json.Unmarshal(rcRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							tc.rolebinding.RoleRef.Namespace:     "newRoleNameSpace",
							tc.rolebinding.Subjects[0].Namespace: "newSubjectNameSpace",
							tc.rolebinding.UserNames[0]:          "newUserNameSpace",
							tc.rolebinding.GroupNames[0]:         "newGroupNameSpace",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			rb := apiauthorization.ClusterRoleBinding{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &rb)

			if tc.want != rb.RoleRef.Namespace {
				t.Fatalf("expected: %v, got: %v", tc.want, rb.RoleRef.Namespace)
			}
		})
	}

	testcase2 := map[string]struct {
		rolebinding     apiauthorization.ClusterRoleBinding
		want            string
		wantSubject     string
		wantSubjectName string
		wantUser        string
		wantGroup       string
	}{
		"SubjectNamespaceSwap": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: "SubjectNamespace"},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"",
				},
			},
			wantSubject: "newSubjectNameSpace",
		},
		"SubjectNameSwap": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: "SubjectNamespace", Name: "role:serviceaccounts:ServiceNamespace:tempValue"},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"",
				},
			},
			wantSubject:     "newSubjectNameSpace",
			wantSubjectName: "role:serviceaccounts:newServiceNamespace:tempValue",
		},
	}

	for i, tc := range testcase2 {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			rcRec, _ := json.Marshal(tc.rolebinding)
			json.Unmarshal(rcRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							tc.rolebinding.RoleRef.Namespace:     "newRoleNameSpace",
							tc.rolebinding.Subjects[0].Namespace: "newSubjectNameSpace",
							"ServiceNamespace":                   "newServiceNamespace",
							tc.rolebinding.UserNames[0]:          "newUserNameSpace",
							tc.rolebinding.GroupNames[0]:          "newGroupNameSpace",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			rb := apiauthorization.ClusterRoleBinding{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &rb)

			if tc.wantSubject != rb.Subjects[0].Namespace {
				t.Fatalf("expected: %v, got: %v", tc.wantSubject, rb.Subjects[0].Namespace)
			}
			if tc.wantSubjectName != "" && tc.wantSubjectName != rb.Subjects[0].Name {
				t.Fatalf("expected: %v, got: %v", tc.wantSubjectName, rb.Subjects[0].Name)
			}
		})
	}

	testcase3 := map[string]struct {
		rolebinding apiauthorization.ClusterRoleBinding
		want        string
		wantSubject string
		wantUser    string
		wantGroup   string
	}{
		"UserNamespaceSwap": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"role:serviceaccount:namespace:tempValue",
				},
				GroupNames: []string{
					"",
				},
			},
			wantUser:	"role:serviceaccount:newNamespace:tempValue",
		},
		"NoSwapCase1": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"temp",
				},
				GroupNames: []string{
					"",
				},
			},
			wantUser:	"temp",
		},
		"NoSwapCase2": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"role:wrongserviceaccount:newNamespace:tempValue",
				},
				GroupNames: []string{
					"",
				},
			},
			wantUser:	"role:wrongserviceaccount:newNamespace:tempValue",
		},
		"NoSwapCase3": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"role:wrongserviceaccount:emptyNamespace:tempValue",
				},
				GroupNames: []string{
					"",
				},
			},
			wantUser:	"role:wrongserviceaccount:emptyNamespace:tempValue",
		},
	}

	for i, tc := range testcase3 {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			rcRec, _ := json.Marshal(tc.rolebinding)
			json.Unmarshal(rcRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							tc.rolebinding.RoleRef.Namespace:     "newRoleNameSpace",
							tc.rolebinding.Subjects[0].Namespace: "newSubjectNameSpace",
							"namespace": "newNamespace",
							"emptynamespace": "",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			rb := apiauthorization.ClusterRoleBinding{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &rb)

			if tc.wantUser != rb.UserNames[0] {
				t.Fatalf("expected: %v, got: %v", tc.wantUser, rb.UserNames[0])
			}
		})
	}

	testcase4 := map[string]struct {
		rolebinding apiauthorization.ClusterRoleBinding
		want        string
		wantSubject string
		wantUser    string
		wantGroup   string
	}{
		"GroupNamespaceSwap": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"role:serviceaccounts:namespace:tempValue",
				},
			},
			wantGroup:	"role:serviceaccounts:newNamespace:tempValue",
		},
		"NoGroupNamespaceSwapCase1": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"role:serviceaccounts",
				},
			},
			wantGroup:	"role:serviceaccounts",
		},
		"NoGroupNamespaceSwapCase2": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"role:serviceaccount:namespace:tempValue",
				},
			},
			wantGroup:	"role:serviceaccount:namespace:tempValue",
		},
		"NoGroupNamespaceSwapCase3": {
			rolebinding: apiauthorization.ClusterRoleBinding{
				RoleRef: corev1API.ObjectReference{
					Namespace: "",
				},
				Subjects: []corev1API.ObjectReference{
					{Namespace: ""},
				},
				UserNames: []string{
					"",
				},
				GroupNames: []string{
					"role:serviceaccount:emptyNamespace:tempValue",
				},
			},
			wantGroup:	"role:serviceaccount:emptyNamespace:tempValue",
		},
	}

	for i, tc := range testcase4 {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			rcRec, _ := json.Marshal(tc.rolebinding)
			json.Unmarshal(rcRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item,
				Restore: &vm.Restore{
					Spec: vm.RestoreSpec{
						NamespaceMapping: map[string]string{
							tc.rolebinding.RoleRef.Namespace:     "newRoleNameSpace",
							tc.rolebinding.Subjects[0].Namespace: "newSubjectNameSpace",
							"namespace": "newNamespace",
							"emptyNamespace": "",
						},
					},
				},
			}
			output, _ := restorePlugin.Execute(&input)

			rb := apiauthorization.ClusterRoleBinding{}
			itemMarshal, _ := json.Marshal(output.UpdatedItem)
			json.Unmarshal(itemMarshal, &rb)

			if tc.wantUser != rb.UserNames[0] {
				t.Fatalf("expected: %v, got: %v", tc.wantGroup, rb.GroupNames[0])
			}
		})
	}
}