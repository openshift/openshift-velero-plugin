package build

import (
	"context"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	buildv1API "github.com/openshift/api/build/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"builds"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	t.Run("Test Execute() for build", func(t *testing.T) {
		testEnv := & envtest.Environment{}
		cfg, err := testEnv.Start()
		require.NoError(t, err)
		defer testEnv.Stop()
		// initialize clients using envtest config
		cv1c, err := clients.CoreClientFromConfig(cfg)
		require.NoError(t, err)
		// create namespace default
		_, err = cv1c.Namespaces().Create(context.Background(), &corev1API.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		}, metav1.CreateOptions{})
		if k8serrors.IsAlreadyExists(err) {
			t.Log("namespace default already exists, which is fine for testing")
			err = nil
		} else {
			require.NoError(t, err)
		}
		// add service account to testEnv
		_, err = cv1c.ServiceAccounts("default").Create(context.Background(), &corev1API.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: "builder",
			},
		}, metav1.CreateOptions{})
		require.NoError(t, err)
		// get service account UID
		sa, err := cv1c.ServiceAccounts("default").Get(context.Background(), "builder", metav1.GetOptions{})
		require.NoError(t, err)
		destUID := string(sa.UID)
		secretList := corev1API.SecretList{
			Items: []corev1API.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "builder-dockercfg-old",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/service-account.name": "builder",
							"kubernetes.io/service-account.uid":  "wrong-uid-on-dest",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "builder-dockercfg-new",
						Namespace: "default",
						Annotations: map[string]string{
							"kubernetes.io/service-account.name": "builder",
							"kubernetes.io/service-account.uid":  destUID,
						},
					},
				},
			},
		}

		oldDockercfgSecret := &corev1API.LocalObjectReference{Name: "builder-dockercfg-old"}
		newDockercfgSecret := &corev1API.LocalObjectReference{Name: "builder-dockercfg-new"}
		oldCustomSecret := &corev1API.LocalObjectReference{Name: "custom-old"}

		build := buildv1API.Build{
			Spec: buildv1API.BuildSpec{
				CommonSpec: buildv1API.CommonSpec{
					Strategy: buildv1API.BuildStrategy{
						SourceStrategy: &buildv1API.SourceBuildStrategy{
							PullSecret: oldDockercfgSecret,
						},
					},
					Output: buildv1API.BuildOutput{
						PushSecret: oldCustomSecret,
					},
				},
			},
		}

		namespaceMapping := make(map[string]string)
		newCommonSpec, err := UpdateCommonSpec(build.Spec.CommonSpec, "registry", "backupRegistry", &secretList, test.NewLogger(), namespaceMapping)
		assert.Equal(t, err, nil)
		build.Spec.CommonSpec = newCommonSpec

		assert.Equal(t, oldCustomSecret, build.Spec.Output.PushSecret)
		assert.Equal(t, newDockercfgSecret, build.Spec.Strategy.SourceStrategy.PullSecret)
	})
}
