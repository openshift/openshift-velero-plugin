package pod

import (
	"encoding/json"

	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"github.com/vmware-tanzu/velero/pkg/restic"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Heptio Ark.
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to pods.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"pods"},
	}, nil
}

const buildPodVolumesToExclude = "buildworkdir,container-storage-root,build-blob-cache"

// Execute copies local registry images into migration registry, if this is a build pod, we will skip volumes
func (p *BackupPlugin) Execute(input runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	pod := corev1API.Pod{}
	itemMarshal, _ := json.Marshal(input)
	json.Unmarshal(itemMarshal, &pod)
	p.Log.Infof("[pod-backup] pod: %s", pod.Name)
	// if pod is a build pod and it might already be completed.
	// we still want build pods to be in the backup so skip volumes to avoid restic backup failure of a completed build pod
	if (pod.Labels != nil && pod.Labels["openshift.io/build.name"] != "") || (pod.Annotations != nil && pod.Annotations["openshift.io/build.name"] != "") {
		if pod.Annotations == nil || pod.Annotations[restic.VolumesToExcludeAnnotation] == "" {
			p.Log.Infof("[pod-backup] pod: %s is a build pod, skipping volumes using annotations", pod.Name)
			if pod.Annotations == nil {
				pod.Annotations = make(map[string]string)
			}
			pod.Annotations[restic.VolumesToExcludeAnnotation] = buildPodVolumesToExclude
		} else {
			p.Log.Infof("[pod-backup] pod: %s is a build pod, already have skip volumes using annotations, left as is", pod.Name)
		}
	}
	var out map[string]interface{}
	objrec, _ := json.Marshal(pod)
	json.Unmarshal(objrec, &out)
	input.SetUnstructuredContent(out)
	return input, nil, nil
}
