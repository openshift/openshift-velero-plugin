package build

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/common"
	"github.com/heptio/velero/pkg/plugin/velero"
	buildv1API "github.com/openshift/api/build/v1"
	"github.com/sirupsen/logrus"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to everything
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"builds"},
	}, nil
}

// Execute action for the restore plugin for the build resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[build-restore] Entering build restore plugin")

	build := buildv1API.Build{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &build)

	secret, err := p.findBuilderDockercfgSecret(build)

	if err != nil {
		// TODO: Come back to this. This is ugly, should really return some type
		// of error but I don't know what that is exactly
		p.Log.Error("[build-restore] Skipping build: ", err)
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	p.Log.Info(fmt.Sprintf("[build-restore] Found new dockercfg secret: %v", secret))
	build = createNewPushSecret(build, secret)

	registry := build.Annotations[common.RestoreRegistryHostname]
	if registry == "" {
		err = fmt.Errorf("failed to find restore registry annotation")
		return nil, err
	}
	// Skip if not internal build
	name := build.Spec.Strategy.SourceStrategy.From.Name
	if !common.HasImageRefPrefix(name, build.Annotations[common.BackupRegistryHostname]) {
		// Does not have internal registry hostname, skip
		p.Log.Errorf("[build-restore] build is not from internal image, skipping")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}
	shaSplit := strings.Split(name, "@")
	if len(shaSplit) < 2 {
		err = fmt.Errorf("unexpected image reference [%v]", name)
		return nil, err
	}
	sha := shaSplit[1]
	splitName := strings.Split(shaSplit[0], "/")
	if len(splitName) < 2 {
		err = fmt.Errorf("unexpected image reference [%v]", name)
		return nil, err
	}
	namespacedName := splitName[len(splitName)-2:]
	var newName string
	if namespacedName[0] == "openshift" {
		// This is a default imagestream/image from existing cluster
		// Do NOT assume the same sha exists and use floating tag instead
		newName = fmt.Sprintf("%s/%s/%s", registry, namespacedName[0], namespacedName[1])
	} else {
		newName = fmt.Sprintf("%s/%s/%s@%s", registry, namespacedName[0], namespacedName[1], sha)
	}

	// Replace all imageRefs
	// This is safe because findBuilderDockercfgSecret will skip the build
	// if it is not sourceBuildStrategyType
	build.Spec.Strategy.SourceStrategy.From.Name = newName
	for _, trigger := range build.Spec.TriggeredBy {
		if trigger.ImageChangeBuild != nil {
			trigger.ImageChangeBuild.ImageID = newName
		}
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(build)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

func (p *RestorePlugin) findBuilderDockercfgSecret(build buildv1API.Build) (string, error) {
	if build.Spec.Strategy.Type != buildv1API.SourceBuildStrategyType {
		return "", errors.New("No source build strategy type found")
	}

	client, err := clients.CoreClient()
	if err != nil {
		return "", err
	}

	secretList, err := client.Secrets(build.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, secret := range secretList.Items {
		if strings.HasPrefix(secret.Name, "builder-dockercfg") {
			return secret.Name, nil
		}
	}

	return "", errors.New("Secret not found")
}

func createNewPushSecret(build buildv1API.Build, secret string) buildv1API.Build {
	newPushSecret := corev1API.LocalObjectReference{Name: secret}
	build.Spec.Output.PushSecret = &newPushSecret
	build.Spec.Strategy.SourceStrategy.PullSecret = &newPushSecret

	return build
}
