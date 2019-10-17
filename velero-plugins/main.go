package main

import (
	"github.com/fusor/openshift-velero-plugin/velero-plugins/build"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/buildconfig"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/common"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/cronjob"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/daemonset"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/deployment"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/deploymentconfig"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/job"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/pod"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/replicaset"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/replicationcontroller"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/route"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/secret"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/service"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/serviceaccount"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/statefulset"
	veleroplugin "github.com/heptio/velero/pkg/plugin/framework"
	"github.com/sirupsen/logrus"
)

func main() {
	veleroplugin.NewServer().
		RegisterBackupItemAction("openshift.io/01-common-backup-plugin", newCommonBackupPlugin).
		RegisterRestoreItemAction("openshift.io/01-common-restore-plugin", newCommonRestorePlugin).
		RegisterRestoreItemAction("openshift.io/02-serviceaccount-restore-plugin", newServiceAccountRestorePlugin).
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
