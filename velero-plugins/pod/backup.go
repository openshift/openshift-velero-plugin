package pod

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Velero
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to replicasets
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"pods"},
	}, nil
}

// Execute action for the backup plugin for the pod resource
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[pod-backup] Entering Pod backup plugin")

	pod := corev1API.Pod{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &pod)
	p.Log.Infof("[pod-backup] pod: %s", pod.Name)

	annotations := pod.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	annotations[common.DCIncludesDMFix] = "true"
	pod.Annotations = annotations

	var out map[string]interface{}
	objrec, _ := json.Marshal(pod)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, nil, nil
}
