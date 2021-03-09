package route

import (
	"encoding/json"

	routev1API "github.com/openshift/api/route/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to everything
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"routes"},
	}, nil
}

// Execute fixes the route path on restore to use the target cluster's domain name
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[route-restore] Entering Route restore plugin")
	route := routev1API.Route{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &route)

	hostGenerated := route.Annotations["openshift.io/host.generated"]
	if hostGenerated == "true" {
		p.Log.Info("[route-restore] Stripping src cluster host from Route")
		route.Spec.Host = ""

		var out map[string]interface{}
		objrec, _ := json.Marshal(route)
		json.Unmarshal(objrec, &out)

		return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
	}
	p.Log.Info("[route-restore] Route has statically-defined host so leaving as-is")

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
