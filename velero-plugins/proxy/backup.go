package proxy

import (
	"context"
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	MustIncludeAnnotation = "backup.velero.io/must-include-additional-items"
)

type BackupPlugin struct {
	Log logrus.FieldLogger
}

func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"proxies"},
	}, nil
}

func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[proxy-backup] Entering Proxy backup plugin")

	var additionalItems []velero.ResourceIdentifier
	var err error

	proxy := &configv1.Proxy{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, proxy)
	p.Log.Infof("[proxy-backup] proxy: %s", proxy.Name)

	if proxy.Spec.TrustedCA.Name != "" {
		p.Log.Infof("[proxy-backup] Proxy %s references ConfigMap: %s/%s", proxy.Name, "openshift-config", proxy.Spec.TrustedCA.Name)
		cmIdentifier, localerr := p.addConfigMap(proxy.Spec.TrustedCA.Name)
		if localerr == nil {
			additionalItems = append(additionalItems, cmIdentifier)
		} else {
			if errors.IsNotFound(localerr) {
				p.Log.Warnf("[proxy-backup] ConfigMap openshift-config/%s for proxy %s not found: %s", proxy.Spec.TrustedCA.Name, proxy.Name, localerr.Error())
			} else {
				p.Log.Errorf("[proxy-backup] Error adding ConfigMap openshift-config/%s for proxy %s: %s", proxy.Spec.TrustedCA.Name, proxy.Name, localerr.Error())
				err = localerr
			}
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(proxy)
	json.Unmarshal(objrec, &out)

	if len(additionalItems) > 0 {
		if metadata, ok := out["metadata"].(map[string]interface{}); ok {
			if annotations, ok := metadata["annotations"].(map[string]interface{}); ok {
				annotations[MustIncludeAnnotation] = "true"
			} else {
				metadata["annotations"] = map[string]interface{}{
					MustIncludeAnnotation: "true",
				}
			}
		}
		p.Log.Infof("[proxy-backup] Set %s annotation on Proxy to include additional items", MustIncludeAnnotation)
	}

	item.SetUnstructuredContent(out)
	return item, additionalItems, err
}

func (p *BackupPlugin) addConfigMap(cmName string) (velero.ResourceIdentifier, error) {
	coreClient, err := clients.CoreClient()
	if err != nil {
		return velero.ResourceIdentifier{}, err
	}

	cm, err := coreClient.ConfigMaps("openshift-config").Get(context.Background(), cmName, metav1.GetOptions{})
	if err != nil {
		return velero.ResourceIdentifier{}, err
	}

	return velero.ResourceIdentifier{
		GroupResource: schema.GroupResource{
			Group:    "",
			Resource: "configmaps",
		},
		Name:      cm.Name,
		Namespace: cm.Namespace,
	}, nil
}
