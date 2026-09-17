package proxy

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestAppliesTo(t *testing.T) {
	plugin := &BackupPlugin{
		Log: logrus.New(),
	}

	selector, err := plugin.AppliesTo()
	assert.NoError(t, err)
	assert.Contains(t, selector.IncludedResources, "proxies")
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

	_, additionalItems, err := plugin.Execute(obj, &v1.Backup{})

	assert.NoError(t, err)
	assert.Len(t, additionalItems, 0)
}

func TestExecuteWithTrustedCA(t *testing.T) {
	plugin := &BackupPlugin{
		Log: logrus.New(),
	}

	proxy := &configv1.Proxy{
		ObjectMeta: metav1.ObjectMeta{
			Name: "cluster",
		},
		Spec: configv1.ProxySpec{
			TrustedCA: configv1.ConfigMapNameReference{
				Name: "test-ca-bundle",
			},
		},
	}

	proxyUnstructured, err := runtime.DefaultUnstructuredConverter.ToUnstructured(proxy)
	assert.NoError(t, err)

	obj := &unstructured.Unstructured{Object: proxyUnstructured}

	_, _, err = plugin.Execute(obj, &v1.Backup{})

	assert.Error(t, err)
}
