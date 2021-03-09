package pod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
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

	// ISSUE-61 : removing the node selectors from pods
	// to avoid pod being `unschedulable` on destination
	pod.Spec.NodeSelector = nil

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
	secretList, err := client.Secrets(pod.Namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nameSpace, err := client.Namespaces().Get(context.Background(), pod.Namespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	for true {
		flag := 0
		for _, secret := range secretList.Items {
			if strings.HasPrefix(secret.Name, "default-dockercfg-") {
				p.Log.Info(fmt.Sprintf("[pod-restore] Found new dockercfg secret: %v", secret))
				flag = 1
				break
			}
		}
		if flag == 1 {
			p.Log.Info(fmt.Sprintf("[pod-restore] the secret is created"))
			break
		}
		if time.Now().Sub(nameSpace.CreationTimestamp.Time) >= 5*time.Minute {
			return nil, errors.New("Secret is not getting created")
		}
		time.Sleep(time.Second)
		secretList, err = client.Secrets(pod.Namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
	}
	for n, secret := range pod.Spec.ImagePullSecrets {
		newSecret, err := common.UpdatePullSecret(&secret, secretList, p.Log)
		if err != nil {
			return nil, err
		}
		pod.Spec.ImagePullSecrets[n] = *newSecret
	}
	// if this is a stage pod and there's a stage pod image found
	destStagePodImage := input.Restore.Annotations[common.StagePodImageAnnotation]
	if len(pod.Labels[common.IncludedInStageBackupLabel]) > 0 && len(destStagePodImage) > 0 {
		p.Log.Infof("[pod-restore] swapping stage pod images for pod %s", pod.Name)
		for n, container := range pod.Spec.Containers {
			p.Log.Infof("[pod-restore] swapping stage pod image from %s to %s", container.Image, destStagePodImage)
			pod.Spec.Containers[n].Image = destStagePodImage
		}
	}
	var out map[string]interface{}
	objrec, _ := json.Marshal(pod)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

// This plugin doesn't need to wait for items
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	return true, nil
}
