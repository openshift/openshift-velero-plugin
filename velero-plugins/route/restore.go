package route

import (
	"encoding/json"
	"github.com/chaitanyab2311/krm-fn-execution-lib/fn"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"reflect"
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

	inputfunc, _ := json.Marshal(input.Item)

	p.Log.Info("Input to plugin: \n%s", string(inputfunc))

	executeFn := fn.ExecuteFn{}
	output, err := executeFn.Execute(inputfunc, "/data/fnconfigs/routefnconfig.yaml")
	if err != nil {
		p.Log.Error(err)
	}

	p.Log.Info("Output of executable: \n%s", string(output))

	var out map[string]interface{}
	objrec, err := yaml.YAMLToJSON(output)
	json.Unmarshal(objrec, &out)

	annotations := out["spec"]
	v := reflect.ValueOf(annotations)
	i := v.Interface()
	a := i.(map[string]interface{})

	if a["host"] == "" {
		p.Log.Info("Host empty")
		obj := unstructured.Unstructured{}
		err = obj.UnmarshalJSON(objrec)

		return velero.NewRestoreItemActionExecuteOutput(&obj), nil
	}

	p.Log.Info("[route-restore] Route has statically-defined host so leaving as-is")
	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
