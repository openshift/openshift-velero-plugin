package deploymentconfig

import (
	"encoding/json"
	"strconv"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	appsv1API "github.com/openshift/api/apps/v1"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/label"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"github.com/vmware-tanzu/velero/pkg/util/boolptr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to deploymentconfigs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"deploymentconfigs"},
	}, nil
}

// Execute action for the restore plugin for the deployment config resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[deploymentconfig-restore] Entering DeploymentConfig restore plugin")

	deploymentConfig := appsv1API.DeploymentConfig{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &deploymentConfig)
	p.Log.Infof("[deploymentconfig-restore] deploymentConfig: %s", deploymentConfig.Name)

	backupRegistry, registry, err := common.GetSrcAndDestRegistryInfo(input.Item)
	if err != nil {
		return nil, err
	}
	common.SwapContainerImageRefs(deploymentConfig.Spec.Template.Spec.Containers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)
	common.SwapContainerImageRefs(deploymentConfig.Spec.Template.Spec.InitContainers, backupRegistry, registry, p.Log, input.Restore.Spec.NamespaceMapping)

	namespaceMapping := input.Restore.Spec.NamespaceMapping
	newNamespace := namespaceMapping[deploymentConfig.Namespace]
	if len(input.Restore.Spec.NamespaceMapping) > 0 {
		for i := range deploymentConfig.Spec.Triggers {
			if deploymentConfig.Spec.Triggers[i].ImageChangeParams == nil {
				continue
			}

			// if trigger namespace is mapped to new one, swap it
			triggerNamespace := deploymentConfig.Spec.Triggers[i].ImageChangeParams.From.Namespace
			if namespaceMapping[triggerNamespace] != "" {
				deploymentConfig.Spec.Triggers[i].ImageChangeParams.From.Namespace = newNamespace
			}
		}
	}

	// Set replicas to 0 if defaultVolumesToRestic is true

	// get backup associated with the restore
	backupName := input.Restore.Spec.BackupName
	backup, err := common.GetBackup(input.Restore.GetUID(), backupName, input.Restore.Namespace)
	if err != nil {
		p.Log.Infof("[deploymentconfig-restore] could not fetch backup associated with the restore, got error: %s", err.Error())
	}
	var defaultVolumesToFsBackup *bool = nil
	if err == nil {
		// check for default fsbackup/restic flag
		if boolptr.IsSetToTrue(backup.Spec.DefaultVolumesToRestic) || boolptr.IsSetToTrue(backup.Spec.DefaultVolumesToFsBackup) {
			defaultVolumesToFsBackup = pointer.Bool(true)
		}
	}
	if deploymentConfig.Spec.Replicas > 0 &&
		defaultVolumesToFsBackup != nil && *defaultVolumesToFsBackup {
		if deploymentConfig.Annotations == nil {
			deploymentConfig.Annotations = make(map[string]string)
		}
		deploymentConfig.Annotations[common.DCOriginalReplicas] = strconv.FormatInt(int64(deploymentConfig.Spec.Replicas), 10)
		deploymentConfig.Annotations[common.DCOriginalPaused] = strconv.FormatBool(deploymentConfig.Spec.Paused)
		deploymentConfig.Spec.Replicas = 0
		deploymentConfig.Spec.Paused = false
		if deploymentConfig.Labels == nil {
			deploymentConfig.Labels = make(map[string]string)
		}
		labelVal := label.GetValidName(input.Restore.Name)
		deploymentConfig.Labels[common.DCReplicasModifiedLabel] = labelVal
		p.Log.Infof("[deploymentconfig-restore] scaling down deploymentconfig, setting original-replicas, original-paused annotations to %ss,%s, setting replicas-modified label to %s", deploymentConfig.Annotations[common.DCOriginalReplicas], deploymentConfig.Annotations[common.DCOriginalPaused], labelVal)
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(deploymentConfig)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}
