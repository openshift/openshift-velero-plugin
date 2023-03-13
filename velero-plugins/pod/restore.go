package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/kuberesource"
	"github.com/vmware-tanzu/velero/pkg/label"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"github.com/vmware-tanzu/velero/pkg/util/collections"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
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

// Check if pod has restore hooks via pod annotations or via restore hook rules
func (p *RestorePlugin) podHasRestoreHooks(pod corev1API.Pod, resources []velerov1.RestoreResourceHookSpec) (bool, error) {
	_, postRestoreHookDefined := pod.Annotations[common.PostRestoreHookAnnotation]
	_, initContainerRestoreHookDefined := pod.Annotations[common.InitContainerRestoreHookAnnotation]
	if postRestoreHookDefined || initContainerRestoreHookDefined {
		p.Log.Info("[pod-restore] pod has restore hooks via annotations")
		return true, nil
	}
	p.Log.Info("[pod-restore] pod has no restore hooks via annotations")
	for _, restoreHookSpec := range resources {
		p.Log.Infof("[pod-restore] hook spec: %v", restoreHookSpec)
		if len(restoreHookSpec.PostHooks) == 0 {
			continue
		}
		//convert MatchLabels to labels.Selector
		var restoreHookLabelSelector labels.Selector
		var err error
		if restoreHookSpec.LabelSelector != nil {
			restoreHookLabelSelector, err = metav1.LabelSelectorAsSelector(restoreHookSpec.LabelSelector)
			if err != nil {
				p.Log.Errorf("[pod-restore] restore hook labelSelector conversion error: %v", err)
				return false, err
			}
		}
		restoreHookSelector := common.ResourceHookSelector{
			Namespaces:    collections.NewIncludesExcludes().Includes(restoreHookSpec.IncludedNamespaces...).Excludes(restoreHookSpec.ExcludedNamespaces...),
			Resources:     collections.NewIncludesExcludes().Includes(restoreHookSpec.IncludedResources...).Excludes(restoreHookSpec.ExcludedResources...),
			LabelSelector: restoreHookLabelSelector,
		}
		if restoreHookSelector.ApplicableTo(kuberesource.Pods, pod.Namespace, pod.Labels) {
			return true, nil
		}
	}
	p.Log.Info("[pod-restore] pod has no restore hooks")
	return false, nil
}

// Execute action for the restore plugin for the pod resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[pod-restore] Entering Pod restore plugin")

	pod := corev1API.Pod{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &pod)
	p.Log.Infof("[pod-restore] pod: %s", pod.Name)

	podUnmodified := corev1API.Pod{}
	itemMarshal, _ = json.Marshal(input.ItemFromBackup)
	json.Unmarshal(itemMarshal, &podUnmodified)

	// ISSUE-61 : removing the node selectors from pods
	// to avoid pod being `unschedulable` on destination
	pod.Spec.NodeSelector = nil

	ownerRefs, err := common.GetOwnerReferences(input.ItemFromBackup)
	if err != nil {
		return nil, err
	}

	// get backup associated with the restore
	backupName := input.Restore.Spec.BackupName
	backup, err := common.GetBackup(input.Restore.GetUID(), backupName, input.Restore.Namespace)
	if err != nil {
		p.Log.Infof("[pod-restore] could not fetch backup associated with the restore, got error: %s", err.Error())
	}

	var defaultVolumesToResticFlag *bool = nil

	if err == nil {
		// check for default restic flag
		defaultVolumesToResticFlag = backup.Spec.DefaultVolumesToRestic
	}

	podHasRestoreHooks := false
	p.Log.Info("[pod-restore] checking if pod has restore hooks")
	if input.Restore.Spec.Hooks.Resources != nil {
		podHasRestoreHooks, err = p.podHasRestoreHooks(pod, input.Restore.Spec.Hooks.Resources)
		if err != nil {
			p.Log.Errorf("[pod-restore] checking if pod has restore hooks failed, got error: %s", err.Error())
			return nil, err
		}
	}
	// Check if pod has owner Refs and defaultVolumesToRestic flag as false/nil
	if (len(ownerRefs) > 0 && pod.Annotations[common.ResticBackupAnnotation] == "" && (defaultVolumesToResticFlag == nil || !*defaultVolumesToResticFlag)) && !podHasRestoreHooks {
		p.Log.Infof("[pod-restore] skipping restore of pod %s, has owner references, no restic backup, and no restore hooks", pod.Name)
		return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
	}

	// If pod has both "deployment" and "deploymentconfig" labels, it belongs to a DeploymentConfig
	// If defaultVolumesToRestic, remove these labels so that the DC won't immediately delete the
	// pod on restore, and add disconnected-from-dc label with restore name for post-restore cleanup
	if pod.Labels != nil &&
		pod.Labels[common.DCPodDeploymentLabel] != "" &&
		pod.Labels[common.DCPodDeploymentConfigLabel] != "" &&
		defaultVolumesToResticFlag != nil && *defaultVolumesToResticFlag {
		delete(pod.Labels, common.DCPodDeploymentLabel)
		delete(pod.Labels, common.DCPodDeploymentConfigLabel)
		labelVal := label.GetValidName(input.Restore.Name)
		pod.Labels[common.DCPodDisconnectedLabel] = labelVal
		p.Log.Infof("[pod-restore] clearing deployment, deploymentconfig labels, setting disconnected-from-dc label to %s", labelVal)
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
	destNamespace := podUnmodified.Namespace
	if input.Restore.Spec.NamespaceMapping[destNamespace] != "" {
		destNamespace = input.Restore.Spec.NamespaceMapping[destNamespace]
	}
	secretList, err := client.Secrets(destNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	nameSpace, err := client.Namespaces().Get(context.Background(), destNamespace, metav1.GetOptions{})
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
		secretList, err = client.Secrets(destNamespace).List(context.Background(), metav1.ListOptions{})
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
		pvcVolumes := []corev1API.Volume{}
		excludePVC := []string{}
		if len(pod.Annotations[common.ExcludePVCPodAnnotation]) > 0 {
			excludePVC = strings.Split(pod.Annotations[common.ExcludePVCPodAnnotation], ",")
		}
		contains := func(volumeName string) bool {
			for _, name := range excludePVC {
				if name == volumeName {
					return true
				}
			}
			return false
		}
		for _, volume := range pod.Spec.Volumes {
			if volume.PersistentVolumeClaim == nil {
				continue
			}

			if len(excludePVC) > 0 && contains(volume.Name) {
				continue
			}
			pvcVolumes = append(pvcVolumes, volume)
		}
		pod.Spec.Volumes = pvcVolumes
		inVolumes := func(mount corev1API.VolumeMount) bool {
			for _, volume := range pvcVolumes {
				if volume.Name == mount.Name {
					return true
				}
			}
			return false
		}

		for n, container := range pod.Spec.Containers {
			p.Log.Infof("[pod-restore] swapping stage pod image from %s to %s", container.Image, destStagePodImage)
			pod.Spec.Containers[n].Image = destStagePodImage
			pod.Spec.Containers[n].Command = []string{"sleep"}
			pod.Spec.Containers[n].Args = []string{"infinity"}
			volumeMount := []corev1API.VolumeMount{}
			for _, vol := range container.VolumeMounts {
				if inVolumes(vol) {
					volumeMount = append(volumeMount, vol)
				}
			}
			pod.Spec.Containers[n].VolumeMounts = volumeMount
		}
	}
	var out map[string]interface{}
	objrec, _ := json.Marshal(pod)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}
