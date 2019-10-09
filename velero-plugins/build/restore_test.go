package build

import (
	"testing"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/heptio/velero/pkg/plugin/velero"
	buildv1API "github.com/openshift/api/build/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"builds"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	t.Run("Test Execute() for build", func(t *testing.T) {
		secretList := corev1API.SecretList{
			Items: []corev1API.Secret{
				corev1API.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "builder-dockercfg-new",
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
