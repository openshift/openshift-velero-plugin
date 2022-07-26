package service

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

// AppliesTo returns a velero.ResourceSelector that applies to services
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"services"},
	}, nil
}

// Execute action for the restore plugin for the service resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	
	p.Log.Info("[service-restore] Entering Service restore plugin")

	marshalledinput, _ := json.Marshal(input.Item)

	functions := []fn.Function{
		{
			Exec: "/data/executables/service",
		},
	}

	runner := fn.NewRunner().
			  WithInput(marshalledinput).
			  WithFunctions(functions...).
			  WhereExecWorkingDir("/usr")

	functionRunner, err := runner.Build()
	if err != nil {
		p.Log.Info("[service-restore] Error occured while building function runner")
	}

	output, err := functionRunner.Execute()
	resource := output.Items[0].(*unstructured.Unstructured)

	return velero.NewRestoreItemActionExecuteOutput(resource), nil
}
