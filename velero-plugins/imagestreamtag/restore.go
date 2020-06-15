package imagestreamtag

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to everything
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"imagestreamtags"},
	}, nil
}

func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	if input.Restore.Labels[common.MigrationApplicationLabelKey] != common.MigrationApplicationLabelValue {
		p.Log.Info("[istag-restore] skipping ImageStreamTag plugin since restore is not part of CAM")
		return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
	} else {
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
		referenceTag := imageStreamTag.Tag != nil && imageStreamTag.Tag.From != nil
		var additionalItems []velero.ResourceIdentifier
		if referenceTag {
			p.Log.Info(fmt.Sprintf("[istag-restore] Reference tag: %v, tag: %v", imageStreamTag.Tag.From.Kind, imageStreamTag.Tag.From.Name))

			// Removing annotations from the tag, to prevent mismatch
			imageStreamTag.Tag.Annotations = nil
			namespaceMapping := input.Restore.Spec.NamespaceMapping
			if imageStreamTag.Tag.From.Kind == "ImageStreamTag" {
				p.Log.Info("[istag-restore] ImageStreamTag reference")
				refOldNamespace := imageStreamTag.Namespace
				if imageStreamTag.Tag.From.Namespace != "" {
					refOldNamespace = imageStreamTag.Tag.From.Namespace
					if namespaceMapping[imageStreamTag.Tag.From.Namespace] != "" {
						imageStreamTag.Tag.From.Namespace = namespaceMapping[imageStreamTag.Tag.From.Namespace]
					}
				}
				refNewNamespace := refOldNamespace
				if namespaceMapping[refOldNamespace] != "" {
					refNewNamespace = namespaceMapping[refOldNamespace]
				}
				p.Log.Info(fmt.Sprintf("[istag-restore] Looking up reference tag: %s/%s", refOldNamespace, imageStreamTag.Tag.From.Name))
				client, err := clients.ImageClient()
				_, err = client.ImageStreamTags(refNewNamespace).Get(imageStreamTag.Tag.From.Name, metav1.GetOptions{})
				if err == nil {
					p.Log.Info("[istag-restore] reference tag found in cluster")
				} else {
					p.Log.Info("[istag-restore] reference tag not found in cluster, adding to restore")
					addImageStream := false
					nameSplit := strings.SplitN(imageStreamTag.Tag.From.Name, ":", 2)
					if len(nameSplit) == 2 {
						_, err = client.ImageStreams(refNewNamespace).Get(nameSplit[0], metav1.GetOptions{})
						if err == nil {
							p.Log.Info("[istag-restore] reference imagestream found in cluster")
						} else {
							p.Log.Info("[istag-restore] reference imagestream not found in cluster, adding to restore")
							addImageStream = true
						}
					}

					additionalItems = []velero.ResourceIdentifier{
						{
							GroupResource: schema.GroupResource{
								Group:    "image.openshift.io",
								Resource: "imagestreamtags",
							},
							Namespace: refOldNamespace,
							Name:      imageStreamTag.Tag.From.Name,
						},
					}
					if addImageStream {
						additionalItems = append([]velero.ResourceIdentifier{
							{
								GroupResource: schema.GroupResource{
									Group:    "image.openshift.io",
									Resource: "imagestreams",
								},
								Namespace: refOldNamespace,
								Name:      nameSplit[0],
							},
						}, additionalItems...)
					}
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
			}, nil
		}
		p.Log.Info("[istag-restore] Not restoring local imagestreamtag")
		return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
	}
}
