package service

import (
	"encoding/json"
	"github.com/chaitanyab2311/krm-fn-execution-lib/fn"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to services
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"services"},
	}, nil
}

// Execute action for the restore plugin for the service resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {

	p.Log.Info("[service-restore] Entering Service restore plugin")

	inputfunc, _ := json.Marshal(input.Item)

	p.Log.Info("[service-restore] Input to plugin: \n%s", string(inputfunc))

	executeFn := fn.ExecuteFn{}
	output, err := executeFn.Execute(inputfunc, "/data/fnconfigs/servicefnconfig.yaml")
	if err != nil {
		p.Log.Error(err)
	}

	p.Log.Info("[service-restore] Output of executable: \n%s", string(output))

	var out map[string]interface{}
	objrec, err := yaml.YAMLToJSON(output)
	json.Unmarshal(objrec, &out)

	obj := unstructured.Unstructured{}
	err = obj.UnmarshalJSON(objrec)

	return velero.NewRestoreItemActionExecuteOutput(&obj), nil
}
