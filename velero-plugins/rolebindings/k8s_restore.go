package rolebindings

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	rbacv1 "k8s.io/api/rbac/v1"
)

// K8sRestorePlugin is a restore item action plugin for k8s RBAC rolebindings
type K8sRestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to k8s RBAC rolebindings
func (p *K8sRestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"rolebindings"},
	}, nil
}

// Execute skips system rolebindings that OpenShift auto-creates for new namespaces
func (p *K8sRestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[rbac-rolebinding-restore] Entering RBAC Role Bindings restore plugin")

	roleBinding := rbacv1.RoleBinding{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &roleBinding)

	p.Log.Infof("[rbac-rolebinding-restore] role binding - %s, API version %s", roleBinding.Name, roleBinding.APIVersion)

	if SystemRoleBindings[roleBinding.Name] {
		p.Log.Infof("[rbac-rolebinding-restore] Skipping system rolebinding %s as it will be automatically created", roleBinding.Name)
		return &velero.RestoreItemActionExecuteOutput{
			SkipRestore: true,
		}, nil
	}

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
