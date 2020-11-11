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

// AppliesTo returns a velero.ResourceSelector that applies to everything.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{}, nil
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
	registryHostname, err := GetRegistryInfo(major, minor, p.Log)
	if err != nil {
		return nil, nil, err
	}
	annotations[BackupRegistryHostname] = registryHostname

	if backup.Labels[MigrationApplicationLabelKey] != MigrationApplicationLabelValue {
		// if the current workflow is not CAM(i.e B/R) then get the backup registry route and set the same on annotation to use in plugins.
		backupRegistryRoute, err := getOADPRegistryRoute(backup.Namespace, backup.Spec.StorageLocation, RegistryConfigMap)
		if err != nil {
			p.Log.Info(fmt.Sprintf("[common-backup] Error in getting route: %s. Assuming this is outside of OADP context.", err))
			annotations[SkipImages] = "true"
		} else {
			annotations[MigrationRegistry] = backupRegistryRoute
		}
	} else {
		// if the current workflow is CAM then get migration registry from backup object and set the same on annotation to use in plugins.
		annotations[MigrationRegistry] = backup.Annotations[MigrationRegistry]
	}
	metadata.SetAnnotations(annotations)
	return item, nil, nil
}
