package main

import (
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/build"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/buildconfig"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clusterrolebindings"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/cronjob"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/daemonset"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/deployment"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/deploymentconfig"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestreamtag"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/imagetag"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/job"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/pod"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/replicaset"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/replicationcontroller"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/rolebindings"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/route"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/scc"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/secret"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/service"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/serviceaccount"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/statefulset"
	apisecurity "github.com/openshift/api/security/v1"
	"github.com/sirupsen/logrus"
	veleroplugin "github.com/vmware-tanzu/velero/pkg/plugin/framework"
)

func main() {
	veleroplugin.NewServer().
		RegisterBackupItemAction("openshift.io/01-common-backup-plugin", newCommonBackupPlugin).
		RegisterRestoreItemAction("openshift.io/01-common-restore-plugin", newCommonRestorePlugin).
		RegisterBackupItemAction("openshift.io/02-serviceaccount-backup-plugin", newServiceAccountBackupPlugin).
		RegisterRestoreItemAction("openshift.io/02-serviceaccount-restore-plugin", newServiceAccountRestorePlugin).
		RegisterBackupItemAction("openshift.io/03-pv-backup-plugin", newPVBackupPlugin).
		RegisterRestoreItemAction("openshift.io/03-pv-restore-plugin", newPVRestorePlugin).
		RegisterRestoreItemAction("openshift.io/04-pvc-backup-plugin", newPVCBackupPlugin).
		RegisterRestoreItemAction("openshift.io/04-pvc-restore-plugin", newPVCRestorePlugin).
		RegisterBackupItemAction("openshift.io/04-imagestreamtag-backup-plugin", newImageStreamTagBackupPlugin).
		RegisterRestoreItemAction("openshift.io/04-imagestreamtag-restore-plugin", newImageStreamTagRestorePlugin).
		RegisterRestoreItemAction("openshift.io/05-route-restore-plugin", newRouteRestorePlugin).
		RegisterRestoreItemAction("openshift.io/06-build-restore-plugin", newBuildRestorePlugin).
		RegisterRestoreItemAction("openshift.io/07-pod-restore-plugin", newPodRestorePlugin).
		RegisterRestoreItemAction("openshift.io/08-deploymentconfig-restore-plugin", newDeploymentConfigRestorePlugin).
		RegisterRestoreItemAction("openshift.io/09-replicationcontroller-restore-plugin", newReplicationControllerRestorePlugin).
		RegisterRestoreItemAction("openshift.io/10-job-restore-plugin", newJobRestorePlugin).
		RegisterRestoreItemAction("openshift.io/11-daemonset-restore-plugin", newDaemonSetRestorePlugin).
		RegisterRestoreItemAction("openshift.io/12-replicaset-restore-plugin", newReplicaSetRestorePlugin).
		RegisterRestoreItemAction("openshift.io/13-deployment-restore-plugin", newDeploymentRestorePlugin).
		RegisterRestoreItemAction("openshift.io/14-statefulset-restore-plugin", newStatefulSetRestorePlugin).
		RegisterRestoreItemAction("openshift.io/15-service-restore-plugin", newServiceRestorePlugin).
		RegisterRestoreItemAction("openshift.io/16-cronjob-restore-plugin", newCronJobRestorePlugin).
		RegisterRestoreItemAction("openshift.io/17-buildconfig-restore-plugin", newBuildConfigRestorePlugin).
		RegisterRestoreItemAction("openshift.io/18-secret-restore-plugin", newSecretRestorePlugin).
		RegisterBackupItemAction("openshift.io/19-is-backup-plugin", newImageStreamBackupPlugin).
		RegisterRestoreItemAction("openshift.io/19-is-restore-plugin", newImageStreamRestorePlugin).
		RegisterRestoreItemAction("openshift.io/20-SCC-restore-plugin", newSCCRestorePlugin).
		RegisterRestoreItemAction("openshift.io/21-role-bindings-restore-plugin", newRoleBindingRestorePlugin).
		RegisterRestoreItemAction("openshift.io/22-cluster-role-bindings-restore-plugin", newClusterRoleBindingRestorePlugin).
		RegisterRestoreItemAction("openshift.io/23-imagetag-restore-plugin", newImageTagRestorePlugin).
		Serve()
}

func newCommonBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &common.BackupPlugin{Log: logger}, nil
}

func newCommonRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &common.RestorePlugin{Log: logger}, nil
}

func newBuildRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &build.RestorePlugin{Log: logger}, nil
}

func newBuildConfigRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &buildconfig.RestorePlugin{Log: logger}, nil
}

func newDaemonSetRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &daemonset.RestorePlugin{Log: logger}, nil
}

func newDeploymentRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &deployment.RestorePlugin{Log: logger}, nil
}

func newDeploymentConfigRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &deploymentconfig.RestorePlugin{Log: logger}, nil
}

func newJobRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &job.RestorePlugin{Log: logger}, nil
}

func newCronJobRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &cronjob.RestorePlugin{Log: logger}, nil
}

func newPodRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &pod.RestorePlugin{Log: logger}, nil
}

func newReplicaSetRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &replicaset.RestorePlugin{Log: logger}, nil
}

func newReplicationControllerRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &replicationcontroller.RestorePlugin{Log: logger}, nil
}

func newRouteRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &route.RestorePlugin{Log: logger}, nil
}

func newServiceRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &service.RestorePlugin{Log: logger}, nil
}

func newServiceAccountRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &serviceaccount.RestorePlugin{Log: logger}, nil
}

func newStatefulSetRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &statefulset.RestorePlugin{Log: logger}, nil
}

func newSecretRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &secret.RestorePlugin{Log: logger}, nil
}

func newPVCBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &pvc.BackupPlugin{Log: logger}, nil
}

func newPVCRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &pvc.RestorePlugin{Log: logger}, nil
}

func newSCCRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &scc.RestorePlugin{Log: logger}, nil
}

func newRoleBindingRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &rolebindings.RestorePlugin{Log: logger}, nil
}

func newClusterRoleBindingRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &clusterrolebindings.RestorePlugin{Log: logger}, nil
}

func newServiceAccountBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	saBackupPlugin := &serviceaccount.BackupPlugin{Log: logger}
	saBackupPlugin.UpdatedForBackup = make(map[string]bool)
	// we need to create a dependency between scc and service accounts. Service accounts are listed in SCC's users list.
	saBackupPlugin.SCCMap = make(map[string]map[string][]apisecurity.SecurityContextConstraints)
	return saBackupPlugin, nil
}

func newPVBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &persistentvolume.BackupPlugin{Log: logger}, nil
}

func newPVRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &persistentvolume.RestorePlugin{Log: logger}, nil
}

func newImageStreamBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &imagestream.BackupPlugin{Log: logger}, nil
}

func newImageStreamRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &imagestream.RestorePlugin{Log: logger}, nil
}

func newImageStreamTagBackupPlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &imagestreamtag.BackupPlugin{Log: logger}, nil
}

func newImageStreamTagRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &imagestreamtag.RestorePlugin{Log: logger}, nil
}

func newImageTagRestorePlugin(logger logrus.FieldLogger) (interface{}, error) {
	return &imagetag.RestorePlugin{Log: logger}, nil
}
