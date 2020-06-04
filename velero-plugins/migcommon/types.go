package migcommon

const MigrationRegistry string = "openshift.io/migration-registry"

const SwingPVAnnotation string = "openshift.io/swing-pv"

// copy, swing, TODO: others (snapshot, custom, etc.)
const MigrateTypeAnnotation string = "openshift.io/migrate-type"

// target storage class
const MigrateStorageClassAnnotation string = "openshift.io/target-storage-class"

//target access mode
const MigrateAccessModeAnnotation string = "openshift.io/target-access-mode"

//stage, final. Only valid for copy type.
const MigrateCopyPhaseAnnotation string = "openshift.io/migrate-copy-phase"

const MigrateQuiesceAnnotation string = "openshift.io/migrate-quiesce-pods"

const PodStageLabel string = "migration-stage-pod"

// kubernetes PVC annotations
const PVCSelectedNodeAnnotation string = "volume.kubernetes.io/selected-node"

// Restic annotations
const ResticRestoreAnnotationPrefix string = "snapshot.velero.io"
const ResticBackupAnnotation string = "backup.velero.io/backup-volumes"

// Namespace SCC annotations
const NamespaceSCCAnnotationMCS string = "openshift.io/sa.scc.mcs"
const NamespaceSCCAnnotationGroups string = "openshift.io/sa.scc.supplemental-groups"
const NamespaceSCCAnnotationUidRange string = "openshift.io/sa.scc.uid-range"

// Restored items label
const MigMigrationLabelKey string = "migmigration"
const MigratedByLabel string = "migration.openshift.io/migrated-by" // (migmigration UID)
