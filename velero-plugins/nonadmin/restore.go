package nonadmin

import (
	"github.com/sirupsen/logrus"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"strings"
)

type RestorePluginNonAdmin struct {
	Log logrus.FieldLogger
}

const (
	// API Group
	GroupOADP = "oadp.openshift.io"

	// Resource Kinds
	KindNonAdminBackup                = "NonAdminBackup"
	KindNonAdminRestore               = "NonAdminRestore"
	KindNonAdminBackupStorageLocation = "NonAdminBackupStorageLocation"
)

var includedResources = []string{
	"nonadminbackups.oadp.openshift.io",
	"nonadminrestores.oadp.openshift.io",
	"nonadminbackupstoragelocations.oadp.openshift.io",
}

func (p *RestorePluginNonAdmin) AppliesTo() (velero.ResourceSelector, error) {
	p.Log.Info("[nonadmin-restore] Applying to NonAdmin resources")
	return velero.ResourceSelector{
		IncludedResources: includedResources,
	}, nil
}

func (p *RestorePluginNonAdmin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	obj, ok := input.Item.(*unstructured.Unstructured)
	if !ok {
		p.Log.Warn("[nonadmin-restore] Failed to cast item to Unstructured, skipping")
		return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
	}

	apiVersion := obj.GetAPIVersion()
	kind := obj.GetKind()
	group := strings.Split(apiVersion, "/")[0]

	if group == GroupOADP && (kind == KindNonAdminBackup || kind == KindNonAdminRestore || kind == KindNonAdminBackupStorageLocation) {
		p.Log.Infof("[nonadmin-restore] Skipping restore of %s/%s (kind: %s)", obj.GetNamespace(), obj.GetName(), kind)
		return velero.NewRestoreItemActionExecuteOutput(input.Item).WithoutRestore(), nil
	}

	p.Log.Infof("[nonadmin-restore] Restoring %s/%s (kind: %s)", obj.GetNamespace(), obj.GetName(), kind)
	return velero.NewRestoreItemActionExecuteOutput(input.Item), nil
}
