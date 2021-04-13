package pvc

import (
	"context"
	"encoding/json"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// BackupPlugin is a backup item action plugin for Heptio Ark.
type BackupPlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a backup.ResourceSelector that applies to everything.
func (p *BackupPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"persistentvolumeclaims"},
	}, nil
}

// Execute sets a custom annotation on the item being backed up.
func (p *BackupPlugin) Execute(item runtime.Unstructured, backup *v1.Backup) (runtime.Unstructured, []velero.ResourceIdentifier, error) {

	if backup.Labels[common.MigrationApplicationLabelKey] != common.MigrationApplicationLabelValue {
		p.Log.Info("[pvc-backup] Returning pvc object as is since this is not a migration activity")
		return item, nil, nil
	}
	p.Log.Info("[pvc-backup] Entering Persistent Volume Claim backup plugin")
	// Convert to PVC
	backupPVC := corev1API.PersistentVolumeClaim{}
	itemMarshal, _ := json.Marshal(item)
	json.Unmarshal(itemMarshal, &backupPVC)

	client, err := clients.CoreClient()
	if err != nil {
		return nil, nil, err
	}
	// Get and update PVC on the running cluster to have Velero Backup label
	// Validate PVC wasn't deleted by getting the object from the cluster
	pvc, err := client.PersistentVolumeClaims(backupPVC.Namespace).Get(context.Background(), backupPVC.Name, metav1.GetOptions{})
	if err != nil {
		return nil, nil, err
	}

	p.Log.Info("[pvc-backup] Setting 'migration.openshift.io/migrated-by-backup' label to track backup that moved PVC")
	backupPVC.Labels["migration.openshift.io/migrated-by-backup"] = backup.Name

	// Update PVC on cluster
	p.Log.Info("[pvc-backup] Updating PVC on cluster with 'migration.openshift.io/migrated-by-backup' label to track backup that moved PVC")
	pvc, err = client.PersistentVolumeClaims(backupPVC.Namespace).Update(context.Background(), pvc, metav1.UpdateOptions{})
	if err != nil {
		return nil, nil, err
	}

	out := make(map[string]interface{})
	marsh, _ := json.Marshal(backupPVC)
	json.Unmarshal(marsh, &out)
	item.SetUnstructuredContent(out)

	return item, nil, nil
}
