package imagestreamtag

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Velero
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to imagestreamtags
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"imagestreamtags"},
	}, nil
}

// Execute sets annotations on imagestreamtags that depend on others to be restored first
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {

	p.Log.Info("[istag-backup] Entering ImageStreamTag backup plugin")
	imageStreamTag := imagev1API.ImageStreamTag{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &imageStreamTag)
	annotations := imageStreamTag.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}
	// clear out any previous istag annotations from old migrations
	delete(annotations, common.RelatedIsTagAnnotation)
	delete(annotations, common.RelatedIsTagNsAnnotation)
	
	p.Log.Info(fmt.Sprintf("[istag-backup] Backing up imagestreamtag %s", imageStreamTag.Name))

	referenceTag := imageStreamTag.Tag != nil && imageStreamTag.Tag.From != nil
	if referenceTag {
		p.Log.Info(fmt.Sprintf("[istag-backup] Reference tag: %v, tag: %v", imageStreamTag.Tag.From.Kind, imageStreamTag.Tag.From.Name))

		tagNamespace := imageStreamTag.Tag.From.Namespace
		if tagNamespace == "" {
			tagNamespace = imageStreamTag.Namespace
		}
		client, err := clients.ImageClient()
		if imageStreamTag.Tag.From.Kind == "ImageStreamTag" {
			p.Log.Info(fmt.Sprintf("[istag-backup] Looking up reference tag: %s/%s", tagNamespace, imageStreamTag.Tag.From.Name))
			_, err = client.ImageStreamTags(tagNamespace).Get(context.Background(), imageStreamTag.Tag.From.Name, metav1.GetOptions{})
			if err == nil {
				p.Log.Info("[istag-backup] istag reference tag found in cluster")
				annotations[common.RelatedIsTagNsAnnotation] = tagNamespace
				annotations[common.RelatedIsTagAnnotation] = imageStreamTag.Tag.From.Name
			}
		} else if imageStreamTag.Tag.From.Kind == "ImageStreamImage" {
			tagNameSplit := strings.Split(imageStreamTag.Tag.From.Name, "@")
			if len(tagNameSplit) == 2 && len(tagNameSplit[1]) > 0 {
				istagList, err := client.ImageStreamTags(tagNamespace).List(context.Background(), metav1.ListOptions{})
				if err == nil {
					for _, istag := range istagList.Items {
						if istag.Image.Name == tagNameSplit[1] &&
							(istag.Tag == nil || istag.Tag.From != nil && istag.Tag.From.Kind == "DockerImage") {
							p.Log.Info("[istag-backup] isimage reference tag found in cluster")
							annotations[common.RelatedIsTagNsAnnotation] = tagNamespace
							annotations[common.RelatedIsTagAnnotation] = istag.Name
							break
						}
					}
				}
			}
		}
	}
	imageStreamTag.Annotations = annotations
	var out map[string]interface{}
	objrec, _ := json.Marshal(imageStreamTag)
	json.Unmarshal(objrec, &out)
	item.SetUnstructuredContent(out)
	return item, nil, nil
}
