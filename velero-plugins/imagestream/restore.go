package imagestream

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bombsimon/logrusr/v3"
	"github.com/containers/image/v5/copy"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/imagecopy"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

// MyRestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to imagestreams
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"imagestreams"},
	}, nil
}

// Execute copies local registry images from migration registry into target cluster local registry
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {

	p.Log.Info("[is-restore] Entering ImageStream restore plugin")
	imageStream := imagev1API.ImageStream{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &imageStream)
	p.Log.Info(fmt.Sprintf("[is-restore] image: %#v", imageStream.Name))
	annotations := imageStream.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if input.Restore.Labels[common.MigrationApplicationLabelKey] != common.MigrationApplicationLabelValue {
		// if the current workflow is not CAM(i.e B/R) then get the backup registry route and set the same on annotation to use in plugins.
		backupLocation, err := getBackupStorageLocationForBackup(input.Restore.GetUID(), input.Restore.Spec.BackupName, input.Restore.Namespace)
		if err != nil {
			return nil, err
		}
		tempRegistry, err := getOADPRegistryRoute(input.Restore.GetUID(), input.Restore.Namespace, backupLocation, common.RegistryConfigMap)
		if err != nil {
			p.Log.Info(fmt.Sprintf("[common-restore] Error in getting route: %s, got %s. Assuming this is outside of OADP context.", err, tempRegistry))
			annotations[common.SkipImageCopy] = "true"
		} else {
			annotations[common.MigrationRegistry] = tempRegistry
		}
	} else {
		// if the current workflow is CAM then get migration registry from backup object and set the same on annotation to use in plugins.
		annotations[common.MigrationRegistry] = input.Restore.Annotations[common.MigrationRegistry]
	}
	if val, ok := annotations[common.DisableImageCopy]; ok && len(val) != 0 && val == "true" {
		p.Log.Info("[is-restore] Image copy is excluded for backup; skipping image copy.")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	imageStreamUnmodified := imagev1API.ImageStream{}
	itemMarshal, _ = json.Marshal(input.ItemFromBackup)
	json.Unmarshal(itemMarshal, &imageStreamUnmodified)

	skipImages := annotations[common.SkipImageCopy]
	if len(skipImages) != 0 && skipImages == "true" {
		p.Log.Info("Not running in OADP/CAM context, skipping copy of image.")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	backupInternalRegistry, internalRegistry, err := common.GetSrcAndDestRegistryInfo(input.Item)
	if err != nil {
		return nil, err
	}
	migrationRegistry := annotations[common.MigrationRegistry]
	if len(migrationRegistry) == 0 {
		return nil, errors.New("migration registry not found for annotation \"openshift.io/migration\"")
	}
	p.Log.Info(fmt.Sprintf("[is-restore] backup internal registry: %#v", backupInternalRegistry))
	p.Log.Info(fmt.Sprintf("[is-restore] restore internal registry: %#v", internalRegistry))

	destNamespace := imageStreamUnmodified.Namespace
	// if destination namespace is mapped to new one, swap it
	namespaceMapping := input.Restore.Spec.NamespaceMapping
	if namespaceMapping[destNamespace] != "" {
		destNamespace = namespaceMapping[imageStreamUnmodified.Namespace]
	}

	sourceCtx, err := migrationRegistrySystemContext()
	if err != nil {
		return nil, err
	}
	destinationCtx, err := internalRegistrySystemContext(input.Restore.GetUID())
	if err != nil {
		return nil, err
	}
	err = imagecopy.CopyLocalImageStreamImages(
		imageStreamUnmodified,
		backupInternalRegistry,
		migrationRegistry,
		internalRegistry,
		destNamespace,
		&copy.Options{
			SourceCtx:      sourceCtx,
			DestinationCtx: destinationCtx,
		},
		logrusr.New(p.Log),
		false)
	if err != nil {
		return nil, err
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(imageStream)
	json.Unmarshal(objrec, &out)
	input.Item.SetUnstructuredContent(out)
	return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
}
