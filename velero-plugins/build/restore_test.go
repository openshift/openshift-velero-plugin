package build

import (
	"testing"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/util/test"
	"github.com/heptio/velero/pkg/plugin/velero"
	buildv1API "github.com/openshift/api/build/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1API "k8s.io/api/core/v1"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}
	actual, err := restorePlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"builds"}}, actual)
}

func TestRestorePluginExecute(t *testing.T) {
	t.Run("Test Execute() for build", func(t *testing.T) {
		oldPushSecret := &corev1API.LocalObjectReference{Name: "oldsecret"}

		build := buildv1API.Build{
			Spec: buildv1API.BuildSpec{
				CommonSpec: buildv1API.CommonSpec{
					Strategy: buildv1API.BuildStrategy{
						SourceStrategy: &buildv1API.SourceBuildStrategy{
							PullSecret: oldPushSecret,
						},
					},
					Output: buildv1API.BuildOutput{
						PushSecret: oldPushSecret,
					},
				},
			},
		}

		expectedPushSecret := &corev1API.LocalObjectReference{Name: "newsecret"}

		build = createNewPushSecret(build, "newsecret")

		assert.Equal(t, expectedPushSecret, build.Spec.Output.PushSecret)
		assert.Equal(t, expectedPushSecret, build.Spec.Strategy.SourceStrategy.PullSecret)
	})
}
