package secret

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
)

const (
	serviceOriginAnnotation = "service.alpha.openshift.io/originating-service-name"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to secrets
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"secrets"},
	}, nil
}

// Execute action for the restore plugin for the secret resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[secret-restore] Entering Secret restore plugin")

	secret := corev1API.Secret{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &secret)
	p.Log.Infof("[secret-restore] Secret: %s", secret.Name)

	// Don't restore secret if is owned by a service and has an `originating-service-name` annotation,
	// as it will be recreated, according to https://docs.openshift.com/container-platform/3.11/dev_guide/secrets.html#secrets-troubleshooting
	// Fix: https://bugzilla.redhat.com/show_bug.cgi?id=1751827

	for annotation, value := range secret.GetAnnotations() {
		if annotation != serviceOriginAnnotation {
			continue
		}
		p.Log.Infof("[secret-restore] Skip secret %s restore, as it will be recreated by a service %s", secret.Name, value)
		return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
	}

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
