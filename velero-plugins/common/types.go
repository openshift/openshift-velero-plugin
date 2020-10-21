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

const BackupServerVersion string = "openshift.io/backup-server-version"
const RestoreServerVersion string = "openshift.io/restore-server-version"

const BackupRegistryHostname string = "openshift.io/backup-registry-hostname"
const RestoreRegistryHostname string = "openshift.io/restore-registry-hostname"

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
)

// PV selection annotations
const (
	MigrateTypeAnnotation         string = "openshift.io/migrate-type"          // copy, move
	MigrateStorageClassAnnotation string = "openshift.io/target-storage-class"
	MigrateAccessModeAnnotation   string = "openshift.io/target-access-mode"
	MigrateCopyMethodAnnotation   string = "migration.openshift.io/copy-method" // snapshot, filesystem
	PvCopyAction                  string = "copy"
	PvMoveAction                  string = "move"
	PvFilesystemCopyMethod        string = "filesystem"
        PvSnapshotCopyMethod          string = "snapshot"
)

// Stage pod (sleep) image
const StagePodImageAnnotation = "migration.openshift.io/stage-pod-image"

// kubernetes PVC annotations
const PVCSelectedNodeAnnotation string = "volume.kubernetes.io/selected-node"

// distinction for B/R and migration
const MigrationApplicationLabelKey string = "app.kubernetes.io/part-of"
const MigrationApplicationLabelValue string = "openshift-migration"

const MigrationRegistry string = "openshift.io/migration-registry"

// Restic annotations
const ResticBackupAnnotation string = "backup.velero.io/backup-volumes"

// Configmap Name
const RegistryConfigMap string = "oadp-registry-config"

// Restored items label
const (
	MigMigrationLabelKey string = "migration.openshift.io/migrated-by-migmigration"
	MigPlanLabelKey      string = "migration.openshift.io/migrated-by-migplan"
)
