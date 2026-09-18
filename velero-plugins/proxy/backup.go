package proxy

import (
	"context"
	"encoding/json"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

const (
	openshiftConfigNamespace = "openshift-config"
)

type BackupPlugin struct {
	Log    logrus.FieldLogger
	Client corev1.CoreV1Interface
}

func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"proxies.config.openshift.io"},
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
		p.Log.Infof("[proxy-backup] Proxy %s references ConfigMap: %s/%s", proxy.Name, openshiftConfigNamespace, proxy.Spec.TrustedCA.Name)
		cmIdentifier, localErr := p.addConfigMap(proxy.Spec.TrustedCA.Name)
		if localErr == nil {
			additionalItems = append(additionalItems, cmIdentifier)
		} else {
			if errors.IsNotFound(localErr) {
				p.Log.Warnf("[proxy-backup] ConfigMap %s/%s for proxy %s not found: %s", openshiftConfigNamespace, proxy.Spec.TrustedCA.Name, proxy.Name, localErr.Error())
			} else {
				p.Log.Errorf("[proxy-backup] Error adding ConfigMap %s/%s for proxy %s: %s", openshiftConfigNamespace, proxy.Spec.TrustedCA.Name, proxy.Name, localErr.Error())
				err = localErr
			}
		}
	}

	if len(additionalItems) > 0 {
		out := item.UnstructuredContent()
		if metadata, ok := out["metadata"].(map[string]interface{}); ok {
			annotations, ok := metadata["annotations"].(map[string]interface{})
			if !ok {
				annotations = make(map[string]interface{})
				metadata["annotations"] = annotations
			}
			annotations[v1.MustIncludeAdditionalItemAnnotation] = "true"
		}
		item.SetUnstructuredContent(out)
		p.Log.Infof("[proxy-backup] Set %s annotation on Proxy to include additional items", v1.MustIncludeAdditionalItemAnnotation)
	}
	return item, additionalItems, err
}

func (p *BackupPlugin) addConfigMap(cmName string) (velero.ResourceIdentifier, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cm, err := p.Client.ConfigMaps(openshiftConfigNamespace).Get(ctx, cmName, metav1.GetOptions{})
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
