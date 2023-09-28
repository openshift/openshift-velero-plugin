package deploymentconfig

import (
	"context"
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/pod"
	appsv1API "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Velero
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to replicasets
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"deploymentconfigs"},
	}, nil
}

// Execute action for the backup plugin for the deploymentconfig resource
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[deploymentconfig-backup] Entering DeploymentConfig backup plugin")

	deploymentConfig := appsv1API.DeploymentConfig{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &deploymentConfig)
	p.Log.Infof("[deploymentconfig-backup] deploymentconfig: %s", deploymentConfig.Name)

	annotations := deploymentConfig.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	hasHooks := "false"
	hasVolumes := "false"

	// get pods for DC
	client, err := clients.CoreClient()
	if err != nil {
		return nil, nil, err
	}
	podList, err := client.Pods(deploymentConfig.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(deploymentConfig.Spec.Selector).String(),
	})
	if err != nil {
		return nil, nil, err
	}
	podLabels := ""
	for _, dcPod := range podList.Items {
		if pod.PodHasRestoreHookAnnotations(dcPod, p.Log) {
			hasHooks = "true"
		}
		if pod.PodHasVolumesToBackUp(dcPod) {
			hasVolumes = "true"
		}
		// take labels from first pod found
		if podLabels == "" {
			podLabels = labels.Set(dcPod.Labels).String()
		}
	}

	annotations[common.DCIncludesDMFix] = "true"
	annotations[common.DCHasPodRestoreHooks] = hasHooks
	annotations[common.DCPodsHaveVolumes] = hasVolumes
	annotations[common.DCPodLabels] = podLabels
	deploymentConfig.Annotations = annotations

	var out map[string]interface{}
	objrec, _ := json.Marshal(deploymentConfig)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, nil, nil
}
