package pod

import (
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to pods
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"pods"},
	}, nil
}

// Execute action for the restore plugin for the pod resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[pod-restore] Entering Pod restore plugin")

	pod := corev1API.Pod{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &pod)
	p.Log.Infof("[pod-restore] pod: %s", pod.Name)
	
	if (input.Restore.Labels[common.MigrationApplicationLabelKey] == common.MigrationApplicationLabelValue) && (input.Restore.Annotations[common.MigrateCopyPhaseAnnotation] == "stage"){
		pod.Labels[common.MigratePodStageLabel] = "true"
		pod.Spec.Affinity = nil
	} else {
		ownerRefs, err := common.GetOwnerReferences(input.ItemFromBackup)
		if err != nil {
			return nil, err
		}
		// Check if pod has owner Refs
		if len(ownerRefs) > 0 && pod.Annotations[common.ResticBackupAnnotation] == "" {
			p.Log.Infof("[pod-restore] skipping restore of pod %s, has owner references and no restic backup", pod.Name)
			return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
		}

		backupRegistry, registry, err := common.GetSrcAndDestRegistryInfo(input.Item)
		if err != nil {
			return nil, err
		}
		common.SwapContainerImageRefs(pod.Spec.Containers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)
		common.SwapContainerImageRefs(pod.Spec.InitContainers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)

		// update PullSecrets
		client, err := clients.CoreClient()
		if err != nil {
			return nil, err
		}
		secretList, err := client.Secrets(pod.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		for n, secret := range pod.Spec.ImagePullSecrets {
			newSecret, err := common.UpdatePullSecret(&secret, secretList, p.Log)
			if err != nil {
				return nil, err
			}
			pod.Spec.ImagePullSecrets[n] = *newSecret
		}
	}
	var out map[string]interface{}
	objrec, _ := json.Marshal(pod)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}
