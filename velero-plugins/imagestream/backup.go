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
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Heptio Ark.
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to imagestreams.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"imagestreams"},
	}, nil
}

// Execute copies local registry images into migration registry
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {

	p.Log.Info("[is-backup] Entering ImageStream backup plugin")
	imageStream := imagev1API.ImageStream{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &imageStream)
	p.Log.Info(fmt.Sprintf("[is-backup] image: %#v", imageStream))
	annotations := imageStream.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if val, ok := backup.Annotations[common.DisableImageCopy]; ok && len(val) != 0 && val == "true" {
		annotations[common.DisableImageCopy] = val
		imageStream.Annotations = annotations
		imageStream.Spec.Tags = nil
		imageStream.Status.Tags = nil
		var out map[string]interface{}
		objrec, _ := json.Marshal(imageStream)
		json.Unmarshal(objrec, &out)
		item.SetUnstructuredContent(out)
		p.Log.Info("[is-backup] Image copy is excluded for backup; skipping image copy.")
		return item, nil, nil
	}

	skipImages := annotations[common.SkipImageCopy]
	if len(skipImages) != 0 && skipImages == "true" {
		p.Log.Info("Not running in OADP/CAM context, skipping copy of image.")
		return item, nil, nil
	}

	internalRegistry := annotations[common.BackupRegistryHostname]
	migrationRegistry := annotations[common.MigrationRegistry]
	if len(migrationRegistry) == 0 {
		return nil, nil, errors.New("migration registry not found for annotation \"openshift.io/migration\"")
	}
	p.Log.Info(fmt.Sprintf("[is-backup] internal registry: %#v", internalRegistry))

	sourceCtx, err := internalRegistrySystemContext()
	if err != nil {
		return nil, nil, err
	}
	destinationCtx, err := migrationRegistrySystemContext()
	if err != nil {
		return nil, nil, err
	}
	err = imagecopy.CopyLocalImageStreamImages(
		imageStream,
		internalRegistry,
		internalRegistry,
		migrationRegistry,
		imageStream.Namespace,
		&copy.Options{
			SourceCtx:      sourceCtx,
			DestinationCtx: destinationCtx,
		},
		logrusr.New(p.Log),
		true)
	if err != nil {
		return nil, nil, err
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(imageStream)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, nil, nil

}
