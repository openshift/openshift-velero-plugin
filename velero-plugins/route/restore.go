package route

import (
	"encoding/json"
	"github.com/chaitanyab2311/krm-fn-execution-lib/fn"
	"github.com/sirupsen/logrus"
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

	marshalledinput, _ := json.Marshal(input.Item)

	//define functions that need to be passed to function runner
	functions := []fn.Function{
		{
			Exec: "/data/executables/route",
		},
	}

	//create a new function runner
	//Working dir '/' cannot be used so we need to use Execution directory as /usr
	runner := fn.NewRunner().
			WithInput(marshalledinput).
			WithFunctions(functions...).
			WhereExecWorkingDir("/usr")

	functionRunner, err := runner.Build()
	if err != nil {
		p.Log.Info("[route-restore] Error occured while building function runner")
	}

	//Excecute the binary
	output, err := functionRunner.Execute()

	//get the ouput resourcelist and parse host from it 
	resource := output.Items[0].(*unstructured.Unstructured)
	host,_,_ := unstructured.NestedString(resource.Object, "spec", "host")

	if host == "" {
		p.Log.Info("[route-restore] Stripping src cluster host from Route")
		return velero.NewRestoreItemActionExecuteOutput(resource), nil
	}

	p.Log.Info("[route-restore] Route has statically-defined host so leaving as-is")
	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
