package proxy

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestAppliesTo(t *testing.T) {
	plugin := &BackupPlugin{
		Log: logrus.New(),
	}

	selector, err := plugin.AppliesTo()
	assert.NoError(t, err)
	assert.Contains(t, selector.IncludedResources, "proxies.config.openshift.io")
}

func TestExecuteWithoutTrustedCA(t *testing.T) {
	plugin := &BackupPlugin{
		Log: logrus.New(),
	}

	proxy := &configv1.Proxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.ProxySpec{},
	}

	proxyUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(proxy)
	assert.NoError(t, err)

	obj := &unstructured.Unstructured{Object: proxyUnstructured}

	_, additionalItems, err := plugin.Execute(obj, &velerov1.Backup{})

	assert.NoError(t, err)
	assert.Len(t, additionalItems, 0)
}

func TestExecuteWithTrustedCA(t *testing.T) {
	// Create a fake Kubernetes client with the ConfigMap
	fakeClientset := fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "etcd-ca-bundle",
				Namespace: "openshift-config",
			},
			Data: map[string]string{
				"ca-bundle.crt": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
			},
		},
	)

	plugin := &BackupPlugin{
		Log:    logrus.New(),
		Client: fakeClientset.CoreV1(),
	}

	proxy := &configv1.Proxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.ProxySpec{
			TrustedCA: configv1.ConfigMapNameReference{
				Name: "etcd-ca-bundle",
			},
		},
	}

	proxyUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(proxy)
	assert.NoError(t, err)

	obj := &unstructured.Unstructured{Object: proxyUnstructured}

	modifiedProxy, additionalItems, err := plugin.Execute(obj, &velerov1.Backup{})

	assert.NoError(t, err)
	assert.Len(t, additionalItems, 1)
	assert.Equal(t, "configmaps", additionalItems[0].GroupResource.Resource)
	assert.Equal(t, "openshift-config", additionalItems[0].Namespace)
	assert.Equal(t, "etcd-ca-bundle", additionalItems[0].Name)

	metadata, ok := modifiedProxy.UnstructuredContent()["metadata"].(map[string]interface{})
	assert.True(t, ok, "metadata not found in modified object")

	annotations, ok := metadata["annotations"].(map[string]interface{})
	assert.True(t, ok, "annotations not found in metadata")

	annotationValue, ok := annotations[velerov1.MustIncludeAdditionalItemAnnotation]
	assert.True(t, ok, "must-include-additional-items annotation not found")
	assert.Equal(t, "true", annotationValue)
}
