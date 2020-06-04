package mignamespace

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/migcommon"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to namespaces
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"namespaces"},
	}, nil
}

// Execute action for the restore plugin for the namespace resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[ns-restore] Entering Namespace restore plugin")

	namespace := corev1API.Namespace{}
	originalNamespace := corev1API.Namespace{}

	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &namespace)
	itemMarshal, _ = json.Marshal(input.ItemFromBackup)
	json.Unmarshal(itemMarshal, &originalNamespace)

	p.Log.Infof("[ns-restore] namespace: %s", namespace.Name)
	// Preserve scc annotations
	annotations := make(map[string]string)
	annotations[migcommon.NamespaceSCCAnnotationMCS] = originalNamespace.Annotations[migcommon.NamespaceSCCAnnotationMCS]
	annotations[migcommon.NamespaceSCCAnnotationGroups] = originalNamespace.Annotations[migcommon.NamespaceSCCAnnotationGroups]
	annotations[migcommon.NamespaceSCCAnnotationUidRange] = originalNamespace.Annotations[migcommon.NamespaceSCCAnnotationUidRange]
	namespace.Annotations = annotations

	var out map[string]interface{}
	objrec, _ := json.Marshal(namespace)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}
