package imagestreamtag

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to imagestreamtags
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"imagestreamtags"},
	}, nil
}

func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[istag-restore] Entering ImageStreamTag restore plugin")
	imageStreamTag := imagev1API.ImageStreamTag{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &imageStreamTag)
	annotations := imageStreamTag.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	p.Log.Info(fmt.Sprintf("[istag-restore] Restoring imagestreamtag %s", imageStreamTag.Name))

	backupInternalRegistry := annotations[common.BackupRegistryHostname]
	p.Log.Info(fmt.Sprintf("[istag-restore] backup internal registry: %#v", backupInternalRegistry))
	dockerImageReference := imageStreamTag.Image.DockerImageReference
	localImage := len(backupInternalRegistry) > 0 && common.HasImageRefPrefix(dockerImageReference, backupInternalRegistry)
	if localImage {
		p.Log.Info(fmt.Sprintf("[istag-restore] Local image: %v", dockerImageReference))
	}
	var additionalItems []velero.ResourceIdentifier
	if len(annotations[common.RelatedIsTagAnnotation]) > 0 && len(annotations[common.RelatedIsTagNsAnnotation]) > 0 {
		p.Log.Info(fmt.Sprintf("[istag-restore] Setting additionalItems: %v/%v", annotations[common.RelatedIsTagNsAnnotation], annotations[common.RelatedIsTagAnnotation]))
		additionalItems = []velero.ResourceIdentifier{
			{
				GroupResource: schema.GroupResource{
					Group:    "image.openshift.io",
					Resource: "imagestreamtags",
				},
				Namespace: annotations[common.RelatedIsTagNsAnnotation],
				Name:      annotations[common.RelatedIsTagAnnotation],
			},
		}
	}
	referenceTag := imageStreamTag.Tag != nil && imageStreamTag.Tag.From != nil
	if referenceTag {
		p.Log.Info(fmt.Sprintf("[istag-restore] Reference tag: %v, tag: %v", imageStreamTag.Tag.From.Kind, imageStreamTag.Tag.From.Name))

		// Removing annotations from the tag, to prevent mismatch
		imageStreamTag.Tag.Annotations = nil
		namespaceMapping := input.Restore.Spec.NamespaceMapping
		if imageStreamTag.Tag.From.Kind == "ImageStreamTag" {
			p.Log.Info("[istag-restore] ImageStreamTag reference")
			if imageStreamTag.Tag.From.Namespace != "" && namespaceMapping[imageStreamTag.Tag.From.Namespace] != "" {
				imageStreamTag.Tag.From.Namespace = namespaceMapping[imageStreamTag.Tag.From.Namespace]
			}
		} else if imageStreamTag.Tag.From.Kind == "ImageStreamImage" {
			if imageStreamTag.Tag.From.Namespace == "" || imageStreamTag.Tag.From.Namespace == imageStreamTag.Namespace {
				referenceTag = false
			}
			if imageStreamTag.Tag.From.Namespace != "" && namespaceMapping[imageStreamTag.Tag.From.Namespace] != "" {
				imageStreamTag.Tag.From.Namespace = namespaceMapping[imageStreamTag.Tag.From.Namespace]
			}
		}
	}

	// Restore the tag if this is a reference tag *or* an external image. Otherwise,
	// image import will create the imagestreamtag automatically.
	if referenceTag || !localImage {
		var out map[string]interface{}
		objrec, _ := json.Marshal(imageStreamTag)
		json.Unmarshal(objrec, &out)
		input.Item.SetUnstructuredContent(out)

		p.Log.Info("[istag-restore] Restoring reference or remote imagestreamtag")
		return &velero.RestoreItemActionExecuteOutput{
			UpdatedItem:     input.Item,
			AdditionalItems: additionalItems,
			WaitForAdditionalItems: true,
		}, nil
	}
	p.Log.Info("[istag-restore] Not restoring local imagestreamtag")
	return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
}

// AreAdditionalItemsReady returns whether or not passed-in ImageStreamTags are ready
func (p *RestorePlugin) AreAdditionalItemsReady(restore *v1.Restore, additionalItems []velero.ResourceIdentifier) (bool, error) {
	p.Log.Debug("[istag-restore] AreAdditionalItemsReady called")
	ready := true
	client, err := clients.ImageClient()
	if err != nil {
		p.Log.Debug("[istag-restore] AreAdditionalItemsReady ImageClient error: %v", err)
		return true, err
	}
	for i, itemIdentifier := range additionalItems {
		p.Log.Debug(fmt.Sprintf("[istag-restore] AreAdditionalItemsReady: %v", i))
		if itemIdentifier.GroupResource.Group != "image.openshift.io" ||
			itemIdentifier.GroupResource.Resource != "imagestreamtags" {
			p.Log.Debug(fmt.Sprintf("[istag-restore] AreAdditionalItemsReady: wrong GroupResource: %v, %v", itemIdentifier.GroupResource.Group, itemIdentifier.GroupResource.Resource))
			continue
		}
		_, err = client.ImageStreamTags(itemIdentifier.Namespace).Get(context.TODO(), itemIdentifier.Name, metav1.GetOptions{})
		if err != nil {
			// istag not ready
			ready = false
			p.Log.Debug(fmt.Sprintf("[istag-restore] AreAdditionalItemsReady: not ready: %v/%v", itemIdentifier.Namespace, itemIdentifier.Name))
			break
		}
		p.Log.Debug(fmt.Sprintf("[istag-restore] AreAdditionalItemsReady: ready: %v/%v", itemIdentifier.Namespace, itemIdentifier.Name))
	}
	return ready, nil
}
