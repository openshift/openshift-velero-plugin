package common

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
)

// BackupPlugin is a backup item action plugin for Heptio Ark.
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to the listed resources in the slice.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
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

// Execute sets a custom annotation on the item being backed up.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {
	p.Log.Info("[common-backup] Entering common backup plugin")

	metadata, annotations, err := getMetadataAndAnnotations(item)
	if err != nil {
		return nil, nil, err
	}

	major, minor, err := GetServerVersion()
	if err != nil {
		return nil, nil, err
	}

	annotations[BackupServerVersion] = fmt.Sprintf("%v.%v", major, minor)
	registryHostname, err := GetRegistryInfo(p.Log)
	if err != nil {
		return nil, nil, err
	}
	annotations[BackupRegistryHostname] = registryHostname

	metadata.SetAnnotations(annotations)
	return item, nil, nil
}
