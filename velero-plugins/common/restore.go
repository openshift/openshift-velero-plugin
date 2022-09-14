package common

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

// RestorePlugin is a restore item action plugin for Heptio Ark.
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to the listed resources in the slice.
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{
			"pods",
			"imagestreams",
			"imagestreams.image.openshift.io",
			"imagestreamtags",
			"imagestreamtags.image.openshift.io",
			"deployments",
			"deployments.apps",
			"deployments.extensions",
			"deploymentconfigs",
			"deploymentconfigs.apps.openshift.io",
			"jobs",
			"jobs.batch",
			"cronjobs",
			"cronjobs.batch",
			"statefulsets",
			"statefulsets.apps",
			"daemonsets",
			"daemonsets.apps",
			"daemonsets.extensions",
			"replicasets",
			"replicasets.apps",
			"replicasets.extensions",
			"replicationcontroller",
			"buildconfigs",
			"buildconfigs.build.openshift.io"},
	}, nil
}

// Execute sets a custom annotation on the item being restored.
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[common-restore] Entering common restore plugin")

	metadata, annotations, err := getMetadataAndAnnotations(input.Item)
	if err != nil {
		return nil, err
	}
	name := metadata.GetName()
	p.Log.Infof("[common-restore] common restore plugin for %s", name)

	major, minor, err := GetServerVersion()
	if err != nil {
		return nil, err
	}

	annotations[RestoreServerVersion] = fmt.Sprintf("%v.%v", major, minor)
	registryHostname, err := GetRegistryInfo(p.Log)
	if err != nil {
		return nil, err
	}
	annotations[RestoreRegistryHostname] = registryHostname

	metadata.SetAnnotations(annotations)

	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
