package build

import (
	"errors"
	"fmt"
	"strings"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/common"
	buildv1API "github.com/openshift/api/build/v1"
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to builds
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"builds"},
	}, nil
}

// Execute action for the restore plugin for the build resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[build-restore] Skipping restore of build to allow buildconfig to recreate it")
	return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil

}

func (p *RestorePlugin) updateSecretsAndDockerRefs(build buildv1API.Build, namespaceMapping map[string]string) (buildv1API.Build, error) {
	client, err := clients.CoreClient()
	if err != nil {
		return build, err
	}

	buildNamespace := build.Namespace
	if namespaceMapping[buildNamespace] != "" {
		buildNamespace = namespaceMapping[buildNamespace]
	}

	secretList, err := client.Secrets(buildNamespace).List(metav1.ListOptions{})
	if err != nil {
		return build, err
	}

	registry := build.Annotations[common.RestoreRegistryHostname]
	if registry == "" {
		err = fmt.Errorf("failed to find restore registry annotation")
		return build, err
	}
	backupRegistry := build.Annotations[common.BackupRegistryHostname]
	if backupRegistry == "" {
		err = fmt.Errorf("failed to find backup registry annotation")
		return build, err
	}

	newCommonSpec, err := UpdateCommonSpec(build.Spec.CommonSpec, registry, backupRegistry, secretList, p.Log, namespaceMapping)
	if err != nil {
		return build, err
	}
	build.Spec.CommonSpec = newCommonSpec
	return build, nil
}

func updateDockerReference(
	fromRef corev1API.ObjectReference,
	registry string,
	backupRegistry string,
	log logrus.FieldLogger,
	namespaceMapping map[string]string,
) (corev1API.ObjectReference, error) {
	if fromRef.Kind != "DockerImage" {
		return fromRef, nil
	}
	newName, err := common.ReplaceImageRefPrefix(fromRef.Name, backupRegistry, registry, namespaceMapping)
	if err != nil {
		// Does not have internal registry hostname, skip
		log.Infof("[build-restore-common] build is not from internal source image, skipping image reference swap")
		return fromRef, nil
	}
	fromRef.Name = newName
	return fromRef, nil
}
func updateDockerSecret(
	secretRef *corev1API.LocalObjectReference,
	secretList *corev1API.SecretList,
	log logrus.FieldLogger,
) (*corev1API.LocalObjectReference, error) {
	// If secret is empty or is anything other than "builder-dockercfg-<generated>"
	// then leave it as-is. Either there's no secret or there's a custom one that
	// should be migrated
	if secretRef == nil || !strings.HasPrefix(secretRef.Name, "builder-dockercfg-") {
		return secretRef, nil
	}

	for _, secret := range secretList.Items {
		if strings.HasPrefix(secret.Name, "builder-dockercfg") {
			log.Info(fmt.Sprintf("[build-restore-common] Found new dockercfg secret: %v", secret))
			newSecret := corev1API.LocalObjectReference{Name: secret.Name}
			return &newSecret, nil
		}
	}

	return nil, errors.New("Secret not found")
}

// UpdateCommonSpec Updates docker references and secrets using CommonSpec, for both Build and BuildConfig
func UpdateCommonSpec(
	spec buildv1API.CommonSpec,
	registry string,
	backupRegistry string,
	secretList *corev1API.SecretList,
	log logrus.FieldLogger,
	namespaceMapping map[string]string,
) (buildv1API.CommonSpec, error) {
	newSecret, err := updateDockerSecret(spec.Output.PushSecret, secretList, log)
	if err != nil {
		return spec, err
	}
	spec.Output.PushSecret = newSecret
	if spec.Output.To != nil {
		newTo, err := updateDockerReference(*spec.Output.To, registry, backupRegistry, log, namespaceMapping)
		if err != nil {
			return spec, err
		}
		spec.Output.To = &newTo
	}

	if spec.Strategy.SourceStrategy != nil {
		newSecret, err := updateDockerSecret(spec.Strategy.SourceStrategy.PullSecret, secretList, log)
		if err != nil {
			return spec, err
		}
		spec.Strategy.SourceStrategy.PullSecret = newSecret
		newFrom, err := updateDockerReference(spec.Strategy.SourceStrategy.From, registry, backupRegistry, log, namespaceMapping)
		if err != nil {
			return spec, err
		}
		spec.Strategy.SourceStrategy.From = newFrom

	}
	if spec.Strategy.DockerStrategy != nil {
		newSecret, err := updateDockerSecret(spec.Strategy.DockerStrategy.PullSecret, secretList, log)
		if err != nil {
			return spec, err
		}
		spec.Strategy.DockerStrategy.PullSecret = newSecret
		if spec.Strategy.DockerStrategy.From != nil {
			newFrom, err := updateDockerReference(*spec.Strategy.DockerStrategy.From, registry, backupRegistry, log, namespaceMapping)
			if err != nil {
				return spec, err
			}
			spec.Strategy.DockerStrategy.From = &newFrom
		}
	}
	if spec.Strategy.CustomStrategy != nil {
		newSecret, err := updateDockerSecret(spec.Strategy.CustomStrategy.PullSecret, secretList, log)
		if err != nil {
			return spec, err
		}
		spec.Strategy.CustomStrategy.PullSecret = newSecret
		newFrom, err := updateDockerReference(spec.Strategy.CustomStrategy.From, registry, backupRegistry, log, namespaceMapping)
		if err != nil {
			return spec, err
		}
		spec.Strategy.CustomStrategy.From = newFrom
	}
	if spec.Source.Images != nil {
		for _, imageSource := range spec.Source.Images {
			newSecret, err := updateDockerSecret(imageSource.PullSecret, secretList, log)
			if err != nil {
				return spec, err
			}
			imageSource.PullSecret = newSecret
			newFrom, err := updateDockerReference(imageSource.From, registry, backupRegistry, log, namespaceMapping)
			if err != nil {
				return spec, err
			}
			imageSource.From = newFrom
		}
	}
	return spec, err
}
