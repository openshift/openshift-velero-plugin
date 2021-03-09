package rolebindings

import (
	"encoding/json"
	"strings"

	apiauthorization "github.com/openshift/api/authorization/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to PVCs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"rolebinding.authorization.openshift.io"},
	}, nil
}

// Execute action for the restore plugin for the pvc resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[rolebinding-restore] Entering Role Bindings restore plugin")

	roleBinding := apiauthorization.RoleBinding{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &roleBinding)

	p.Log.Infof("[rolebinding-restore] role binding - %s, API version", roleBinding.Name, roleBinding.APIVersion)

	namespaceMapping := input.Restore.Spec.NamespaceMapping
	if len(namespaceMapping) > 0 {
		newRoleRefNamespace := namespaceMapping[roleBinding.RoleRef.Namespace]
		if newRoleRefNamespace != "" {
			roleBinding.RoleRef.Namespace = newRoleRefNamespace
		}

		roleBinding.Subjects = SwapSubjectNamespaces(roleBinding.Subjects, namespaceMapping)
		roleBinding.UserNames = SwapUserNamesNamespaces(roleBinding.UserNames, namespaceMapping)
		roleBinding.GroupNames = SwapGroupNamesNamespaces(roleBinding.GroupNames, namespaceMapping)
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(roleBinding)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}

func SwapSubjectNamespaces(subjects []corev1.ObjectReference, namespaceMapping map[string]string) []corev1.ObjectReference {
	for i, subject := range subjects {
		newSubjectNamespace := namespaceMapping[subject.Namespace]

		// If subject has namespace swap it
		if subject.Namespace != "" && newSubjectNamespace != "" {
			subjects[i].Namespace = newSubjectNamespace
		}

		// subject names can point to all service accounts in a namespace(SystemGroup) - xxx:serviceaccounts:oldnamespace
		splitName := strings.Split(subject.Name, ":")
		if len(splitName) < 4 {
			continue
		}

		if splitName[1] == "serviceaccounts" && namespaceMapping[splitName[2]] != "" {
			splitName[2] = namespaceMapping[splitName[2]]
			subjects[i].Name = strings.Join(splitName, ":")
		}
	}

	return subjects
}

func SwapUserNamesNamespaces(userNames []string, namespaceMapping map[string]string) []string {
	for i, userName := range userNames {
		// User name can point to a service account and username format is role:serviceaccount:namespace:serviceaccountname
		splitUsername := strings.Split(userName, ":")
		if len(splitUsername) <= 2 { // safety check
			continue
		}

		if splitUsername[1] != "serviceaccount" {
			continue
		}

		// if second element is serviceaccount then third element is namespace
		newNamespace := namespaceMapping[splitUsername[2]]
		if newNamespace == "" {
			continue
		}
		// swap namespaces when namespace mapping is enabled
		splitUsername[2] = newNamespace
		joinedUsername := strings.Join(splitUsername, ":")
		userNames[i] = joinedUsername
	}

	return userNames
}

func SwapGroupNamesNamespaces(groupNames []string, namespaceMapping map[string]string) []string {
	for i, group := range groupNames {
		// group names can point to all service accounts in a namespace(SystemGroup) - xxx:serviceaccounts:oldnamespace
		splitGroup := strings.Split(group, ":")
		if len(splitGroup) < 4 {
			continue
		}

		if splitGroup[1] == "serviceaccounts" && namespaceMapping[splitGroup[2]] != "" {
			splitGroup[2] = namespaceMapping[splitGroup[2]]
			groupNames[i] = strings.Join(splitGroup, ":")
		}
	}

	return groupNames
}
