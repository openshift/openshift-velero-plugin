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

// Other annotations
const (
	StagePodImageAnnotation   string = "migration.openshift.io/stage-pod-image"     // Stage pod (sleep) image
	RelatedIsTagNsAnnotation  string = "migration.openshift.io/related-istag-ns"    // Related istag ns
	RelatedIsTagAnnotation    string = "migration.openshift.io/related-istag"       // Related istag name
	PVCSelectedNodeAnnotation string = "volume.kubernetes.io/selected-node"         // kubernetes PVC annotations
	ResticBackupAnnotation    string = "backup.velero.io/backup-volumes"            // Restic annotations
	PVOriginalReclaimPolicy   string = "migration.openshift.io/orig-reclaim-policy" // Original PersistentVolumeReclaimPolicy
	BackupNameAnnotation      string = "migration.openshift.io/migrated-by-backup"  // Name of Backup that migrated this resource
)

// Configmap Name
const RegistryConfigMap string = "oadp-registry-config"

// Restored items label
const (
	MigMigrationLabelKey string = "migration.openshift.io/migrated-by-migmigration"
	MigPlanLabelKey      string = "migration.openshift.io/migrated-by-migplan"
)
