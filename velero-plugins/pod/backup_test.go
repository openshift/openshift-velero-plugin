package pod

import (
	"context"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	securityv1 "github.com/openshift/api/security/v1"
	fakeSecurityClient "github.com/openshift/client-go/security/clientset/versioned/fake"
	security "github.com/openshift/client-go/security/clientset/versioned/typed/security/v1"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ktesting "k8s.io/client-go/testing"
)

type ErrorType string

const (
	NotFoundError ErrorType = "NotFound"
	AccessDenied  ErrorType = "AccessDenied"
	NoError       ErrorType = "NoError"
)

func getNewSecurityClient(errorType ErrorType, withSecurityClientError error, objects ...*securityv1.SecurityContextConstraints) func() (security.SecurityV1Interface, error) {
	// the fake client has a weird bug where if the object is added to NewSimpleClientSet the resource value is set wrong
	// as a result the object will not be found
	// [key 0]: schema.GroupVersionResource {Group: "security.openshift.io", Version: "v1", Resource: "securitycontextconstraintses"} <--- should be securitycontextconstraints
	cs := fakeSecurityClient.NewSimpleClientset()

	// add error if enabled
	switch errorType {
	case NotFoundError:
		cs.Fake.PrependReactor("get", "securitycontextconstraints", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.NewNotFound(schema.GroupResource{Group: "security.openshift.io", Resource: "securitycontextconstraints"}, "test-scc")
		})
	case AccessDenied:
		cs.Fake.PrependReactor("get", "securitycontextconstraints", func(action ktesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, errors.NewForbidden(schema.GroupResource{Group: "security.openshift.io", Resource: "securitycontextconstraints"}, "test-scc", nil)
		})
	case NoError:
		// do nothing, client will return objects as normal
	}

	return func() (security.SecurityV1Interface, error) {
		client := cs.SecurityV1()
		for _, object := range objects {
			_, localError := client.SecurityContextConstraints().Create(context.Background(), object, metav1.CreateOptions{})
			if localError != nil {
				return nil, localError
			}
		}
		return client, withSecurityClientError
	}
}

func TestExecute_BackupPod(t *testing.T) {
	originalSecurityClient := clients.SecurityClient
	t.Cleanup(func() {
		clients.SecurityClient = originalSecurityClient
	})

	scc := &securityv1.SecurityContextConstraints{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scc",
		},
	}

	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
			Annotations: map[string]string{
				common.SCCPodAnnotation: "test-scc",
			},
		},
	}

	tests := []struct {
		name                string
		annotations         map[string]string
		inducedErrorType    ErrorType
		expectedSccCount    int
		shouldErr           bool
		expectedIdentifiers []velero.ResourceIdentifier
		securityClientError error
	}{
		{
			name:             "SCC exists and is added to additional items",
			inducedErrorType: NoError,
			expectedSccCount: 1,
			shouldErr:        false,
			expectedIdentifiers: []velero.ResourceIdentifier{
				{
					Name: "test-scc",
					GroupResource: schema.GroupResource{
						Group:    "security.openshift.io",
						Resource: "securitycontextconstraints",
					},
				},
			},
			annotations: map[string]string{
				common.SCCPodAnnotation: "test-scc",
			},
		},
		{
			name:                "SCC does not exist and error is handled",
			inducedErrorType:    NotFoundError,
			expectedSccCount:    0,
			shouldErr:           false,
			expectedIdentifiers: []velero.ResourceIdentifier{},
			annotations: map[string]string{
				common.SCCPodAnnotation: "test-scc",
			},
		},
		{
			name:                "error getting SCC for any other reason",
			inducedErrorType:    AccessDenied,
			expectedSccCount:    0,
			shouldErr:           true,
			expectedIdentifiers: []velero.ResourceIdentifier{},
			annotations: map[string]string{
				common.SCCPodAnnotation: "test-scc",
			},
		},
		{
			name:                "pod has no annotations",
			inducedErrorType:    NoError,
			expectedSccCount:    0,
			shouldErr:           false,
			expectedIdentifiers: []velero.ResourceIdentifier{},
			annotations:         nil,
		},
		{
			name:                "security client initialize error",
			inducedErrorType:    NoError,
			expectedSccCount:    0,
			shouldErr:           true,
			expectedIdentifiers: []velero.ResourceIdentifier{},
			securityClientError: assert.AnError,
			annotations: map[string]string{
				common.SCCPodAnnotation: "test-scc",
			},
		},
	}

	for _, test := range tests {
		pod.Annotations = test.annotations

		objs := []*securityv1.SecurityContextConstraints{scc}

		podData, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&pod)
		assert.NoError(t, err)
		unstructuredPod := unstructured.Unstructured{Object: podData}
		clients.SecurityClient = getNewSecurityClient(test.inducedErrorType, test.securityClientError, objs...)

		backupPlugin := &BackupPlugin{
			Log: logrus.WithField("plugin", "pod-backup-test"),
		}

		_, items, err := backupPlugin.Execute(&unstructuredPod, &velerov1.Backup{})
		if test.shouldErr {
			assert.Error(t, err, "Test %s errored when should not %v", test.name, err)
		} else {
			assert.NoError(t, err, "Test %s errored when should not %v", test.name, err)
		}
		assert.Len(t, items, test.expectedSccCount, "Test %s Expected %d additional items for the SCC", test.name, test.expectedSccCount)
		assert.ElementsMatch(t, items, test.expectedIdentifiers, "Test %s expected additional items to match expected identifiers", test.name)
	}
}

func TestAppliesTo(t *testing.T) {
	backupPlugin := &BackupPlugin{
		Log: logrus.WithField("plugin", "pod-backup-test"),
	}

	resourceSelector, err := backupPlugin.AppliesTo()
	assert.NoError(t, err)
	assert.Contains(t, resourceSelector.IncludedResources, "pods", "Expected resource selector to include pods")
}
