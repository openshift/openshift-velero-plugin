package configmap

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/openshift"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Velero.
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to configmaps.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"configmaps"},
	}, nil
}

// Execute adds annotation to skip restore of configMaps belonging to a build pod, which will get regenerated on build restore, to avoid restoring cacert from different environment.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {

	p.Log.Info("[cm-backup] Entering ConfigMap backup plugin")
	configMap := corev1.ConfigMap{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &configMap)
	p.Log.Infof("[cm-backup] ConfigMap: %v/%v", configMap.Namespace, configMap.Name)
	// return if not build configmap name
	if !nameHasBuildSuffix(configMap.Name) {
		p.Log.Info("[cm-backup] Leaving ConfigMap backup plugin, not a buildconfig-build's configmap name")
		return item, nil, nil
	}
	// return if no ownerRef
	if len(configMap.OwnerReferences) == 0 {
		p.Log.Info("[cm-backup] Leaving ConfigMap backup plugin, not a buildconfig-build's configmap with ownerRef")
		return item, nil, nil
	}
	// return if ownerRef is not pod
	foundBuildPodRef := false
	for _, ref := range configMap.OwnerReferences {
		if ref.Kind == "Pod" {
			if isBuildPod, err := openshift.IsBuildPod(ref.Name, configMap.Namespace); err != nil {
				p.Log.Warnf("[cm-backup] could not determine if ownerRef is buildconfig-build's pod: %v", err)
			} else if isBuildPod {
				foundBuildPodRef = true
				break
			}
		}
	}
	// return if ownerRef to build pod not found
	if !foundBuildPodRef {
		p.Log.Info("[cm-backup] Leaving ConfigMap backup plugin, not a build configmap with buildconfig-build's pod's ownerRef")
		return item, nil, nil
	}

	// Build Pod Owned Configmap, annotate to skip restore
	if configMap.Annotations == nil {
		configMap.Annotations = make(map[string]string)
	}
	if _, exists := configMap.Annotations[common.SkipBuildConfigConfigMapRestore]; !exists {
		p.Log.Info("[cm-backup] buildconfig-build's configmap has pod's ownerRef, adding skip annotation")
		configMap.Annotations[common.SkipBuildConfigConfigMapRestore] = "true"
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(configMap)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, nil, nil

}
