package serviceaccount

import (
	"encoding/json"
	"strings"

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

// AppliesTo returns a velero.ResourceSelector that applies to everything
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"serviceaccounts"},
	}, nil
}

// Execute fixes the route path on restore to use the target cluster's domain name
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[serviceaccount-restore] Entering ServiceAccount restore plugin")

	serviceAccount := corev1.ServiceAccount{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &serviceAccount)

	p.Log.Info("[serviceaccount-restore] Checking for pull secrets to remove")
	check := serviceAccount.Name + "-dockercfg-"
	for i := len(serviceAccount.Secrets) - 1; i >= 0; i-- {
		secret := &serviceAccount.Secrets[i]
		p.Log.Infof("[serviceaccount-restore] Checking if secret %s matches %s", secret.Name, check)

		if strings.HasPrefix(secret.Name, check) {
			// Copy all secrets *except* -dockercfg-
			p.Log.Info("[serviceaccount-restore] Match found - excluding this secret")
			serviceAccount.Secrets = append(serviceAccount.Secrets[:i], serviceAccount.Secrets[i+1:]...)
			break
		} else {
			p.Log.Info("[serviceaccount-restore] No match found - including this secret")
		}
	}
	for i := len(serviceAccount.ImagePullSecrets) - 1; i >= 0; i-- {
		secret := &serviceAccount.ImagePullSecrets[i]
		p.Log.Infof("[serviceaccount-restore] Checking if image pull secret %s matches %s", secret.Name, check)

		if strings.HasPrefix(secret.Name, check) {
			// Copy all secrets *except* -dockercfg-
			p.Log.Info("[serviceaccount-restore] Match found - excluding this secret")
			serviceAccount.ImagePullSecrets = append(serviceAccount.ImagePullSecrets[:i], serviceAccount.ImagePullSecrets[i+1:]...)
			break
		} else {
			p.Log.Info("[serviceaccount-restore] No match found - including this secret")
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(serviceAccount)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
