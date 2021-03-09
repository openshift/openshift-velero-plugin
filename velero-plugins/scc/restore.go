package scc

import (
	"encoding/json"
	"strings"

	apisecurity "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to PVCs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"securitycontextconstraints"},
	}, nil
}

// Execute action for the restore plugin for the pvc resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[scc-restore] Entering SCC restore plugin")

	scc := apisecurity.SecurityContextConstraints{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &scc)

	p.Log.Infof("[scc-restore] scc: %s", scc.Name)

	namespaceMapping := input.Restore.Spec.NamespaceMapping
	if len(namespaceMapping) != 0 {
		for i, user := range scc.Users {
			// Service account username format role:serviceaccount:namespace:serviceaccountname
			splitUsername := strings.Split(user, ":")
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
			scc.Users[i] = joinedUsername
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(scc)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
