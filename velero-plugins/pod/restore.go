package pod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/openshift"
	"github.com/sirupsen/logrus"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/kuberesource"
	"github.com/vmware-tanzu/velero/pkg/label"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"github.com/vmware-tanzu/velero/pkg/util/boolptr"
	"github.com/vmware-tanzu/velero/pkg/util/collections"
	"github.com/vmware-tanzu/velero/pkg/util/podvolume"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log                logrus.FieldLogger
	WaitForPullSecrets *bool
}

// AppliesTo returns a velero.ResourceSelector that applies to pods
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"pods"},
	}, nil
}

// Check if pod has volumes to back up
func PodHasVolumesToBackUp(pod corev1API.Pod) bool {
	// always pass in true for defaultVolumesToFsBackup because we just want to know whether there
	// are any volumes to back up at all. This func filters out volumes not to back up and then
	// splits the list between fs backup and snapshot. If false is passed in, only fs backup files
	// are returned
	vols, optedOutVols := podvolume.GetVolumesByPod(&pod, true)
	return len(vols) > 0 || len(optedOutVols) > 0
}

// Check if pod has restore hooks via pod annotations or via restore hook rules
func PodHasRestoreHooks(pod corev1API.Pod, restore *velerov1.Restore, log logrus.FieldLogger) (bool, error) {
	if PodHasRestoreHookAnnotations(pod, log) {
		return true, nil
	}
	return RestoreHasRestoreHooks(restore, pod.Namespace, pod.Labels, log)
}

func PodHasRestoreHookAnnotations(pod corev1API.Pod, log logrus.FieldLogger) bool {
	if pod.Annotations == nil {
		return false
	}
	_, postRestoreHookDefined := pod.Annotations[common.PostRestoreHookAnnotation]
	_, initContainerRestoreHookDefined := pod.Annotations[common.InitContainerRestoreHookAnnotation]
	if postRestoreHookDefined || initContainerRestoreHookDefined {
		log.Info("[pod-restore] pod has restore hooks via annotations")
		return true
	}
	log.Info("[pod-restore] pod has no restore hooks via annotations")
	return false
}

func RestoreHasRestoreHooks(restore *velerov1.Restore, namespace string, podLabels map[string]string, log logrus.FieldLogger) (bool, error) {
	for _, restoreHookSpec := range restore.Spec.Hooks.Resources {
		log.Infof("[pod-restore] hook spec: %v", restoreHookSpec)
		if len(restoreHookSpec.PostHooks) == 0 {
			continue
		}
		//convert MatchLabels to labels.Selector
		var restoreHookLabelSelector labels.Selector
		var err error
		if restoreHookSpec.LabelSelector != nil {
			restoreHookLabelSelector, err = metav1.LabelSelectorAsSelector(restoreHookSpec.LabelSelector)
			if err != nil {
				log.Errorf("[pod-restore] restore hook labelSelector conversion error: %v", err)
				return false, err
			}
		}
		restoreHookSelector := common.ResourceHookSelector{
			Namespaces:    collections.NewIncludesExcludes().Includes(restoreHookSpec.IncludedNamespaces...).Excludes(restoreHookSpec.ExcludedNamespaces...),
			Resources:     collections.NewIncludesExcludes().Includes(restoreHookSpec.IncludedResources...).Excludes(restoreHookSpec.ExcludedResources...),
			LabelSelector: restoreHookLabelSelector,
		}
		if restoreHookSelector.ApplicableTo(kuberesource.Pods, namespace, podLabels) {
			return true, nil
		}
	}
	log.Info("[pod-restore] pod has no restore hooks")
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

	var defaultVolumesToFsBackup *bool = nil

	if err == nil {
		// check for default fsbackup/restic flag
		if boolptr.IsSetToTrue(backup.Spec.DefaultVolumesToRestic) || boolptr.IsSetToTrue(backup.Spec.DefaultVolumesToFsBackup) {
			defaultVolumesToFsBackup = pointer.Bool(true)
		}
	}

	podHasRestoreHooks := false
	p.Log.Info("[pod-restore] checking if pod has restore hooks")
	if input.Restore.Spec.Hooks.Resources != nil {
		podHasRestoreHooks, err = PodHasRestoreHooks(pod, input.Restore, p.Log)
		if err != nil {
			p.Log.Errorf("[pod-restore] checking if pod has restore hooks failed, got error: %s", err.Error())
			return nil, err
		}
	}

	p.Log.Info("[pod-restore] checking if pod has volumes that were backed up")
	podHasVolumesToBackUp := PodHasVolumesToBackUp(pod)

	// Check if pod has owner Refs and defaultVolumesToRestic flag as false/nil
	if len(ownerRefs) > 0 && !podHasVolumesToBackUp && !podHasRestoreHooks {
		p.Log.Infof("[pod-restore] skipping restore of pod %s, has owner references, no volumes to back up, and no restore hooks", pod.Name)
		return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
	}

	// If pod has both "deployment" and "deploymentconfig" labels, it belongs to a DeploymentConfig
	// As needed (see below for conditions, as it depends on when backup was taken) remove these labels
	// so that the DC won't immediately delete the pod on restore, and add disconnected-from-dc label
	// with restore name for post-restore cleanup
	disconnectIfDC := false
	// For backups made with OADP 1.3 or later, base this on the presence of any volumes to back up or restore hooks
	if pod.Annotations != nil && len(pod.Annotations[common.DCIncludesDMFix]) > 0 {
		disconnectIfDC = podHasRestoreHooks || podHasVolumesToBackUp
		// For backups made with OADP 1.2 or earlier, use only the defaultVolumesToRestic flag
	} else {
		disconnectIfDC = defaultVolumesToFsBackup != nil && *defaultVolumesToFsBackup
	}
	if pod.Labels != nil &&
		pod.Labels[common.DCPodDeploymentLabel] != "" &&
		pod.Labels[common.DCPodDeploymentConfigLabel] != "" &&
		disconnectIfDC {
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

	var secretList *corev1API.SecretList
	nameSpace, err := client.Namespaces().Get(context.Background(), destNamespace, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if p.WaitForPullSecrets == nil {
		wait, err := p.UpdateWaitForPullSecrets()
		if err != nil {
			return nil, err
		}

		p.WaitForPullSecrets = &wait
	}
	// We check for the existence of OpenShift Image Registry replicas to determine whether ImageRegistry Cluster capabilities are enabled
	// Additionally we also need to check OCP version
	// Based on the above 2 things we determine whether to skip waiting for docker secret i.e.  if image registry is not enabled and OCP cluster is above 4.15
	if *p.WaitForPullSecrets {
		for {
			secretList, err = client.Secrets(destNamespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				return nil, err
			}
			flag := 0
			for _, secret := range secretList.Items {
				if strings.HasPrefix(secret.Name, "default-dockercfg-") {
					p.Log.Info(fmt.Sprintf("[pod-restore] Found new dockercfg secret: %v", secret))
					flag = 1
					break
				}
			}
			if flag == 1 {
				p.Log.Info("[pod-restore] the dockercfg secret is created")
				break
			}
			if time.Since(nameSpace.CreationTimestamp.Time) >= 5*time.Minute {
				return nil, errors.New("default-dockercfg- Secret is not getting created within 5 minutes, exiting")
			}
			time.Sleep(time.Second)
		}
		for n, secret := range pod.Spec.ImagePullSecrets {
			newSecret, err := common.UpdatePullSecret(&secret, secretList, p.Log)
			if err != nil {
				return nil, err
			}
			pod.Spec.ImagePullSecrets[n] = *newSecret
		}
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

// checks the OCP Image registry existence
func (p *RestorePlugin) DoRegistryReplicasExist() (bool, error) {
	ocpRegistryHasReplicas, err := openshift.ImageRegistryHasReplicas()
	if err != nil {
		return false, err
	}
	return ocpRegistryHasReplicas, nil
}

// Fetches OCP version information
func (p *RestorePlugin) GetOCPVersion() (int, int, error) {
	ocpVersion, err := openshift.GetClusterVersion()
	if err != nil {
		return 0, 0, err
	}

	majorVersion, minorVersion, _, err := common.ParseOCPVersion(ocpVersion.Status.Desired.Version)
	if err != nil {
		return 0, 0, err
	}

	majorVersionInt, _ := strconv.Atoi(majorVersion)
	minorVersionInt, _ := strconv.Atoi(minorVersion)

	return majorVersionInt, minorVersionInt, nil
}

// Update OCP cluster details
func (p *RestorePlugin) UpdateWaitForPullSecrets() (bool, error) {
	registryReplicasExist, err := p.DoRegistryReplicasExist()
	if err != nil {
		return false, err
	}

	majorVersionInt, minorVersionInt, err := p.GetOCPVersion()
	if err != nil {
		return false, err
	}

	if !registryReplicasExist && (majorVersionInt == 4 && minorVersionInt >= 15 || majorVersionInt > 4) {
		return false, nil
	}

	return true, nil
}
