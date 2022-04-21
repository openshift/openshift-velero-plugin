package horizontalpodautoscaler

import (
	"encoding/json"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	appsv1API "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/api/autoscaling/v2beta1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to everything
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"horizontalpodautoscalers"},
	}, nil
}

// Execute fixes apiVersion in ScaleTargetRef of HPA
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[hpa-restore] Entering HorizontalPodAutoscaler restore plugin")
	hpa := v2beta1.HorizontalPodAutoscaler{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &hpa)

	if (v2beta1.CrossVersionObjectReference{}) != hpa.Spec.ScaleTargetRef {
		gv, err := schema.ParseGroupVersion(hpa.Spec.ScaleTargetRef.APIVersion)
		if err != nil {
			p.Log.Error("[hpa-restore] error parsing API version of spec.scaleTargetRef: ", err)
			return nil, err
		}

		if gv == (schema.GroupVersion{Group: "", Version: "v1"}) && hpa.Spec.ScaleTargetRef.Kind == "DeploymentConfig" {
			p.Log.Info("[hpa-restore] Fixing DeploymentConfig apiVersion on scaleTargetRef")
			hpa.Spec.ScaleTargetRef.APIVersion = appsv1API.GroupVersion.String()

			var out map[string]interface{}
			objrec, _ := json.Marshal(hpa)
			json.Unmarshal(objrec, &out)

			return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
		}
	}

	p.Log.Info("[hpa-restore] ScaleTargetRef not a DeploymentConfig, leaving as-is")

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
