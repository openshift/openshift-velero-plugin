package common

type routingConfig struct {
	Subdomain string `json:"subdomain"`
}
type imagePolicyConfig struct {
	InternalRegistryHostname string `json:"internalRegistryHostname"`
}

// APIServerConfig stores configuration information about the current cluster
type APIServerConfig struct {
	ImagePolicyConfig imagePolicyConfig `json:"imagePolicyConfig"`
	RoutingConfig     routingConfig     `json:"routingConfig"`
}

// src/dest cluster annotations, general migration
const (
	BackupServerVersion     string = "openshift.io/backup-server-version"
	RestoreServerVersion    string = "openshift.io/restore-server-version"
	BackupRegistryHostname  string = "openshift.io/backup-registry-hostname"
	RestoreRegistryHostname string = "openshift.io/restore-registry-hostname"
	MigrationRegistry       string = "openshift.io/migration-registry"
	PausedOwnerRef          string = "openshift.io/paused-owner-ref"
	// distinction for B/R and migration
	MigrationApplicationLabelKey   string = "app.kubernetes.io/part-of"
	MigrationApplicationLabelValue string = "openshift-migration"
)

const SkipImageCopy string = "openshift.io/skip-image-copy"
const DisableImageCopy string = "migration.openshift.io/disable-image-copy"

// annotations and labels related to stage vs. initial/final migrations/restores
const (
	// Whether the backup/restore is associated with a stage or a final migration
	StageOrFinalMigrationAnnotation string = "migration.openshift.io/migmigration-type" // (stage|final)
	StageMigration                  string = "stage"
	FinalMigration                  string = "final"
	// Resources included in the stage backup.
	// Referenced by the Backup.LabelSelector. The value is the Task.UID().
	IncludedInStageBackupLabel = "migration-included-stage-backup"
	// Designated as an `initial` Backup.
	// The value is the Task.UID().
	InitialBackupLabel = "migration-initial-backup"
	// Designated as an `stage` Backup.
	// The value is the Task.UID().
	StageBackupLabel = "migration-stage-backup"
	// Designated as an `stage` Restore.
	// The value is the Task.UID().
	StageRestoreLabel = "migration-stage-restore"
	// Designated as a `final` Restore.
	// The value is the Task.UID().
	FinalRestoreLabel = "migration-final-restore"
	// Exclude PVCs if marked "move" or "snapshot" in the plan
	ExcludePVCPodAnnotation = "migration.openshift.io/exclude-pvcs"
)

// PV selection annotations
const (
	MigrateTypeAnnotation         string = "openshift.io/migrate-type" // copy, move
	MigrateStorageClassAnnotation string = "openshift.io/target-storage-class"
	MigrateAccessModeAnnotation   string = "openshift.io/target-access-mode"
	MigrateCopyMethodAnnotation   string = "migration.openshift.io/copy-method" // snapshot, filesystem
	PvCopyAction                  string = "copy"
	PvMoveAction                  string = "move"
	PvFilesystemCopyMethod        string = "filesystem"
	PvSnapshotCopyMethod          string = "snapshot"
)

// Annotations related to image pull secrets and SAs
const (
	RegistrySANameAnnotation     string = "openshift.io/internal-registry-auth-token.service-account" // OCP version >= 4.16
	RegistryPullSecretAnnotation string = "openshift.io/internal-registry-pull-secret-ref"            // OCP version >= 4.16
	LegacySANameAnnotation       string = "kubernetes.io/service-account.name"                        // OCP version <= 4.15
	LegacySAUIDAnnotation        string = "kubernetes.io/service-account.uid"                         // OCP version <= 4.15
)
// Other annotations
const (
	StagePodImageAnnotation   string = "migration.openshift.io/stage-pod-image"     // Stage pod (sleep) image
	RelatedIsTagNsAnnotation  string = "migration.openshift.io/related-istag-ns"    // Related istag ns
	RelatedIsTagAnnotation    string = "migration.openshift.io/related-istag"       // Related istag name
	PVCSelectedNodeAnnotation string = "volume.kubernetes.io/selected-node"         // kubernetes PVC annotations
	PVOriginalReclaimPolicy   string = "migration.openshift.io/orig-reclaim-policy" // Original PersistentVolumeReclaimPolicy
)

// DC-related labels/annotations
const (
	DCPodDeploymentLabel       string = "deployment"                              // identifies associated RC
	DCPodDeploymentConfigLabel string = "deploymentconfig"                        // identifies associated DC
	DCPodDisconnectedLabel     string = "oadp.openshift.io/disconnected-from-dc"  // pod had DC labels removed on restore
	DCOriginalReplicas         string = "oadp.openshift.io/original-replicas"     // replicas value from backup
	DCOriginalPaused           string = "oadp.openshift.io/original-paused"       // replicas value from backup
	DCReplicasModifiedLabel    string = "oadp.openshift.io/replicas-modified"     // DC replicas modified on restore
	DCIncludesDMFix            string = "oadp.openshift.io/includes-dm-dc-fix"    // Set on pods and DCs to indicate that the datamover backup fix is applied
	DCHasPodRestoreHooks       string = "oadp.openshift.io/has-pod-restore-hooks" // True if DC pods have restore hooks defined on them
	DCPodsHaveVolumes          string = "oadp.openshift.io/pods-have-volumes"     // True if DC pods have volumes to restore
	DCPodLabels                string = "oadp.openshift.io/pod-labels"            // labels from DC pod
)

// Configmap Name
const RegistryConfigMap string = "oadp-registry-config"

// Restored items label
const (
	MigMigrationLabelKey string = "migration.openshift.io/migrated-by-migmigration"
	MigPlanLabelKey      string = "migration.openshift.io/migrated-by-migplan"
)

const (
	// Post Restore Hook must contain this annotation
	PostRestoreHookAnnotation string = "post.hook.restore.velero.io/command"
	//InitContainer restore hook must contain this annotation
	InitContainerRestoreHookAnnotation string = "init.hook.restore.velero.io/container-image"
)
