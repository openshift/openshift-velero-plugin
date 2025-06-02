# OpenShift Velero Plugin [![Build Status](https://travis-ci.com/konveyor/openshift-velero-plugin.svg?branch=master)](https://travis-ci.com/konveyor/openshift-velero-plugin) [![Maintainability](https://api.codeclimate.com/v1/badges/95d3aaf8af1cfdd529c4/maintainability)](https://codeclimate.com/github/konveyoropenshift-velero-plugin/maintainability)

## Introduction

The OpenShift Velero Plugin helps backup and restore projects on an OpenShift cluster. 
The plugin executes additional logic (such as adding annotations and swapping namespaces) during the backup/restore process on top of what Velero executes (without the plugin). Each OpenShift or Kubernetes resource has its own backup/restore plugin.

## Prerequisites 

Velero needs to be installed on the cluster. There are two operators we explicitly support with this plugin; The [OADP Operator](https://github.com/konveyor/oadp-operator) and the [CAM Operator](https://github.com/konveyor/mig-operator).

## Kinds of Plugins

Velero currently supports the following kinds of plugins:

- **Backup Item Action** - performs arbitrary logic on individual items prior to storing them in the backup file.
- **Restore Item Action** - performs arbitrary logic on individual items prior to restoring them in the Kubernetes cluster.
- **Item Block Action** - determines whether a resource should be processed by returning a boolean value. In this plugin, we use it to exclude certain resources from backup.

## Resources Included in Plugin 

- Build
- Build Config
- Config Map
- Cluster Role Binding
- Cron Job
- Daemonset
- Deployment
- Deployment Config
- Horizontal Pod Autoscaler
- Image Stream
- Image Stream Tag
- Image Tag
- Job
- Persistent Volume
- Persistent Volume Claim
- Pod
- Replica Set
- Replication Controller
- Role Binding
- Route
- Security Context Constraints (SCC)
- Secret
- Service
- Service Account
- Stateful Set

## Enabling and Disabling the Plugin for Individual Resources

The [main.go](/velero-plugins/main.go#35) file includes code that registers individual plugins for each OpenShift resource. To disable the plugin code for a particular resource, comment out the respective line.
 
## Building the Plugins

To build the plugins, run

```bash
$ make
```

To build the image, run

```bash
$ make container
```

This builds an image tagged as `docker.io/konveyor/openshift-velero-plugin`. If you want to specify a
different name, run

```bash
$ make container IMAGE=your-repo/your-name:here
```

## Deploying the Plugins

To deploy your plugin image to a Velero server:

1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <image>`, e.g. `velero plugin add quay.io/ocpmigrate/velero-plugin`

Note: If Velero installed with the OADP Operator, this configuration would already be present in the setup. Kindly edit the configurations accordingly: [here](https://github.com/konveyor/oadp-operator#configure-velero-plugins) is some help.

## Registry Configuration

The plugin handles registry operations using [udistribution](https://github.com/migtools/udistribution), a tool for copying container images between registries. The plugin dynamically constructs configuration based on the Backup Storage Location (BSL) settings.

### How Registry Configuration Works

1. The plugin reads the Backup Storage Location configuration from Velero
2. Based on the cloud provider (AWS, Azure, or GCP), it constructs appropriate configuration
3. This configuration is passed as environment-variable-style strings to udistribution
4. udistribution uses this configuration to authenticate and copy images between registries

### Supported Cloud Providers

- **AWS**: Constructs S3-compatible storage configuration
- **Azure**: Constructs Azure Blob storage configuration  
- **GCP**: Constructs Google Cloud Storage configuration

The actual registry routes, credentials, and storage configurations come from:

- OpenShift's internal image registry settings
- Backup Storage Location credentials and configuration
- Pull secrets for registry authentication

## Backup/Restore Applications Using the Plugin

The [velero-example](https://github.com/konveyor/velero-examples) repository contains some basic examples of backup/restore using Velero.

Note: At this point, Velero is already installed with the OADP Operator so skip the Velero installation steps.

## How the Plugin Works

This plugin includes two workflows: OADP and Migration.

1. **OADP Workflow**: Default workflow for standard backup/restore operations.
   - Images are migrated through configured registry routes
   - Uses registry environment variables (REGISTRY_*, BACKUP_REGISTRY_*, RESTORE_REGISTRY_*)
   - Handles disconnected environments via configured pull secrets

2. **Migration Workflow**: For cross-cluster migrations using migration-registry.
   - Requires annotation: `openshift.io/migration-registry: <registry-url>`
   - Requires label: `app.kubernetes.io/part-of: openshift-migration`
   - Enables special handling for PVs, PVCs, and cross-cluster image migration

### Key Annotations

The plugins use various annotations to track state and control behavior:

- `openshift.io/backup-server-version`: Source cluster version
- `openshift.io/restore-server-version`: Destination cluster version
- `openshift.io/backup-registry-hostname`: Source registry hostname
- `openshift.io/restore-registry-hostname`: Destination registry hostname
- `openshift.io/migration-registry`: Migration/intermediate registry URL
- `openshift.io/skip-image-copy`: Skip image migration for imagestreams
- `openshift.io/disable-image-copy`: Disable all image copying
- `openshift.io/original-replicas`: Original replica count (for DCs)
- `openshift.io/dc-has-pod-restore-hooks`: DC has pods with restore hooks
- `openshift.io/dc-pods-have-volumes`: DC has pods with volumes
- `oadp.openshift.io/skip-restore`: Skip restore of specific non-admin resources

## Debug Logs

There are several Velero commands that help get the logs or status of the backup/restore process.

1. `velero get backup` returns all the backups present in Velero:

    ```cassandraql
    $ velero get backup
    NAME                  STATUS                       CREATED                         EXPIRES   STORAGE LOCATION   SELECTOR
    mssql-persistent      Completed                    2020-06-16 15:45:12 -0400 EDT   1d        default            <none>
    nginx-stateless       PartiallyFailed (2 errors)   2020-07-14 16:42:31 -0400 EDT   29d       default            <none>
    patroni-test          Completed                    2020-07-13 12:26:28 -0400 EDT   27d       default            <none>
    postgres-persistent   Completed                    2020-07-02 16:37:36 -0400 EDT   17d       default            <none>
    postgresql            Completed                    2020-07-07 17:24:28 -0400 EDT   22d       default            <none>
    
    ```

2. `velero backup logs <backup-name>` returns logs for the respective backup:

    ```bash
   $ velero backup logs nginx-stateless
   time="2020-07-14T20:42:34Z" level=info msg="Setting up backup temp file" backup=oadp-operator/nginx-stateless logSource="pkg/controller/backup_controller.go:494"
   time="2020-07-14T20:42:34Z" level=info msg="Setting up plugin manager" backup=oadp-operator/nginx-stateless logSource="pkg/controller/backup_controller.go:501"
   time="2020-07-14T20:42:34Z" level=info msg="Getting backup item actions" backup=oadp-operator/nginx-stateless logSource="pkg/controller/backup_controller.go:505"
   time="2020-07-14T20:42:34Z" level=info msg="Setting up backup store" backup=oadp-operator/nginx-stateless logSource="pkg/controller/backup_controller.go:511"
   time="2020-07-14T20:42:35Z" level=info msg="Writing backup version file" backup=oadp-operator/nginx-stateless logSource="pkg/backup/backup.go:213"
   time="2020-07-14T20:42:35Z" level=info msg="Including namespaces: nginx-example" backup=oadp-operator/nginx-stateless logSource="pkg/backup/backup.go:219"
   time="2020-07-14T20:42:35Z" level=info msg="Excluding namespaces: <none>" backup=oadp-operator/nginx-stateless logSource="pkg/backup/backup.go:220"
   time="2020-07-14T20:42:35Z" level=info msg="Including resources: *" backup=oadp-operator/nginx-stateless logSource="pkg/backup/backup.go:223"
   time="2020-07-14T20:42:35Z" level=info msg="Excluding resources: <none>" backup=oadp-operator/nginx-stateless logSource="pkg/backup/backup.go:224"
   time="2020-07-14T20:43:06Z" level=info msg="Backing up group" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/group_backupper.go:101"
   time="2020-07-14T20:43:06Z" level=info msg="Backing up resource" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/resource_backupper.go:105" resource=pods
   time="2020-07-14T20:43:06Z" level=info msg="Listing items" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/resource_backupper.go:226" namespace=nginx-example resource=pods
   time="2020-07-14T20:43:06Z" level=info msg="Retrieved 6 items" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/resource_backupper.go:240" namespace=nginx-example resource=pods
   time="2020-07-14T20:43:06Z" level=info msg="Backing up item" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/item_backupper.go:169" name=cakephp-ex-1-build namespace=nginx-example resource=pods
   time="2020-07-14T20:43:06Z" level=info msg="Executing custom action" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/item_backupper.go:330" name=cakephp-ex-1-build namespace=nginx-example resource=pods
   time="2020-07-14T20:43:06Z" level=info msg="Executing podAction" backup=oadp-operator/nginx-stateless cmd=/velero logSource="pkg/backup/pod_action.go:51" pluginName=velero
   time="2020-07-14T20:43:06Z" level=info msg="Done executing podAction" backup=oadp-operator/nginx-stateless cmd=/velero logSource="pkg/backup/pod_action.go:77" pluginName=velero
   time="2020-07-14T20:43:06Z" level=info msg="Executing custom action" backup=oadp-operator/nginx-stateless group=v1 logSource="pkg/backup/item_backupper.go:330" name=cakephp-ex-1-build namespace=nginx-example resource=pods
   time="2020-07-14T20:43:06Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
   ...
   ```

3. `velero get restore` returns all the restores present in Velero:
    
    ```bash
   $ velero get restore
     NAME           BACKUP         STATUS      WARNINGS   ERRORS   CREATED                         SELECTOR
     patroni-test   patroni-test   Completed   8          0        2020-07-13 12:34:04 -0400 EDT   <none>
   ```
   
4. `velero restore logs <restore-name>` returns logs for the respective restore:

    ```cassandraql
    $ velero restore logs patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="starting restore" logSource="pkg/controller/restore_controller.go:458" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Not including resource" groupResource=events logSource="pkg/restore/restore.go:138" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Not including resource" groupResource=nodes logSource="pkg/restore/restore.go:138" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Not including resource" groupResource=events.events.k8s.io logSource="pkg/restore/restore.go:138" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Not including resource" groupResource=resticrepositories.velero.io logSource="pkg/restore/restore.go:138" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Not including resource" groupResource=restores.velero.io logSource="pkg/restore/restore.go:138" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Not including resource" groupResource=backups.velero.io logSource="pkg/restore/restore.go:138" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Starting restore of backup oadp-operator/patroni-test" logSource="pkg/restore/restore.go:394" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Restoring cluster level resource 'persistentvolumes'" logSource="pkg/restore/restore.go:779" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Getting client for /v1, Kind=PersistentVolume" logSource="pkg/restore/restore.go:821" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Restoring persistent volume from snapshot." logSource="pkg/restore/restore.go:956" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="successfully restored persistent volume from snapshot" logSource="pkg/restore/pv_restorer.go:99" persistentVolume=pvc-9104f3ae-fb5f-4fd8-8062-297690939c8b providerSnapshotID=snap-0f49d6d6ebef09dad restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Executing item action for persistentvolumes" logSource="pkg/restore/restore.go:1030" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Executing ChangeStorageClassAction" cmd=/velero logSource="pkg/restore/change_storageclass_action.go:63" pluginName=velero restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Done executing ChangeStorageClassAction" cmd=/velero logSource="pkg/restore/change_storageclass_action.go:74" pluginName=velero restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="Executing item action for persistentvolumes" logSource="pkg/restore/restore.go:1030" restore=oadp-operator/patroni-test
    time="2020-07-13T16:34:05Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni-test
    ...
    ```

## Overview of Each Plugin

### Common

These backup and restore plugins provide shared functionality for multiple resource types.

#### Backup Plugin

- **Resources**: pods, imagestreams, deployments, deploymentconfigs, jobs, cronjobs, statefulsets, daemonsets, replicasets, replicationcontrollers, buildconfigs
- **Actions**:
  - Sets the `openshift.io/backup-server-version` annotation to source cluster version
  - Gets internal registry hostname and stores in `openshift.io/backup-registry-hostname` annotation
  - Sets the `openshift.io/migration-registry` annotation for migration workflows

```log
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
```

#### Restore Plugin

- **Resources**: Same as backup plugin
- **Actions**:
  - Sets the `openshift.io/restore-server-version` annotation to destination cluster version
  - Gets internal registry hostname and stores in `openshift.io/restore-registry-hostname` annotation

```log
time="2020-07-29T18:51:02Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:02Z" level=info msg="[common-restore] common restore plugin for pvc-2fbae99b-29d0-4853-a0e0-ee077ab60c18" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:29" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[common-restore] common restore plugin for pvc-64b3fc01-8d09-4f62-9612-f5d9b6ee80ff" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:29" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Build

#### Restore Plugin

- **Resources**: builds
- **Actions**:
  - Completely skips restore (returns `.WithoutRestore()`)
  - Allows BuildConfig to recreate builds as needed

### Build Config

#### Restore Plugin

- **Resources**: buildconfigs
- **Actions**:
  - Updates container image references from backup registry to restore registry
  - Updates pull secrets to match destination cluster registry
  - Handles namespace mapping for image references

### Cluster Role Binding

#### Restore Plugin

- **Resources**: clusterrolebindings
- **Actions**:
  - Updates namespaces in subjects when namespace mapping is enabled
  - Handles ServiceAccount subject namespace mapping
  - Preserves role references while updating namespace contexts

### Config Map

#### Backup Plugin

- **Resources**: configmaps
- **Actions**:
  - Identifies configmaps owned by build pods
  - Adds `openshift.io/skip-bc-configmap-restore: true` annotation to prevent restore conflicts

#### Restore Plugin

- **Resources**: configmaps
- **Actions**:
  - Skips restore if `openshift.io/skip-bc-configmap-restore: true` annotation is present
  - Prevents conflicts with build-related configmaps

### Cron Job

#### Restore Plugin

- **Resources**: cronjobs
- **Actions**:
  - Updates container image references in job template spec
  - Swaps images from backup registry to restore registry

### Daemonset

#### Restore Plugin

- **Resources**: daemonsets
- **Actions**:
  - Updates all container image references from backup registry to restore registry
  - Handles both init containers and regular containers

### Deployment

#### Restore Plugin

- **Resources**: deployments
- **Actions**:
  - Updates all container image references from backup registry to restore registry
  - Handles both init containers and regular containers

### Deployment Config

#### Backup Plugin

- **Resources**: deploymentconfigs
- **Actions**:
  - Detects if DC has pods with restore hooks or volumes
  - Stores pod selector labels for later restoration
  - Adds annotations:
    - `openshift.io/dc-has-pod-restore-hooks`: If pods have restore hooks
    - `openshift.io/dc-pods-have-volumes`: If pods have volumes
    - `openshift.io/dc-pod-labels`: Pod selector labels
    - `openshift.io/dc-includes-dm-fix: true`: Marks DC for disconnect fix

#### Restore Plugin

- **Resources**: deploymentconfigs
- **Actions**:
  - Updates all container image references from backup to restore registry
  - Updates image change trigger namespaces if namespace mapping is enabled
  - Sets replicas to 0 if DC has volumes or restore hooks (prevents pod startup conflicts)
  - Stores original replica count in `openshift.io/original-replicas` annotation
  - Stores original paused state in `openshift.io/original-paused` annotation
  - Adds `oadp.openshift.io/replicas-modified: true` label when replicas are modified

### Horizontal Pod Autoscaler

#### Restore Plugin

- **Resources**: horizontalpodautoscalers
- **Actions**:
  - Updates target deployment/deploymentconfig references for namespace mapping
  - Adjusts target namespace when resources are restored to different namespaces

### Image Stream

#### Backup Plugin

- **Resources**: imagestreams
- **Actions**:
  - Retrieves internal registry hostname from cluster configuration
  - Copies all images from internal registry to migration registry
  - Updates image references with new digest information
  - Handles OADP registry configuration via environment variables
  - Skips image copy if `openshift.io/skip-image-copy: true` or `openshift.io/disable-image-copy: true`
  - For disconnected environments, uses registry pull secrets for authentication
  - Preserves image digests and manifests during migration

```log
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] Entering ImageStream backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:35" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] image: v1.ImageStream{TypeMeta:v1.TypeMeta{Kind:\"ImageStream\", APIVersion:\"image.openshift.io/v1\"}, ObjectMeta:v1.ObjectMeta{Name:\"cakephp-ex\", GenerateName:\"\", Namespace:\"nginx-example\", SelfLink:\"/apis/image.openshift.io/v1/namespaces/nginx-example/imagestreams/cakephp-ex\", UID:\"ae5f4ffa-7bfa-4081-bf77-3e767d6fcc34\", ResourceVersion:\"25571924\", Generation:1, CreationTimestamp:v1.Time{Time:time.Time{wall:0x0, ext:63729988302, loc:(*time.Location)(0x2c752c0)}}, DeletionTimestamp:(*v1.Time)(nil), DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string{\"openshift.io/backup-registry-hostname\":\"image-registry.openshift-image-registry.svc:5000\", \"openshift.io/backup-server-version\":\"1.17\", \"openshift.io/migration-registry\":\"oadp-default-aws-registry-route-oadp-operator.apps.cluster-jgabani0518.jgabani0518.mg.dog8code.com\"}, OwnerReferences:[]v1.OwnerReference(nil), Initializers:(*v1.Initializers)(nil), Finalizers:[]string(nil), ClusterName:\"\", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, Spec:v1.ImageStreamSpec{LookupPolicy:v1.ImageLookupPolicy{Local:false}, DockerImageRepository:\"\", Tags:[]v1.TagReference(nil)}, Status:v1.ImageStreamStatus{DockerImageRepository:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex\", PublicDockerImageRepository:\"\", Tags:[]v1.NamedTagEventList{v1.NamedTagEventList{Tag:\"latest\", Items:[]v1.TagEvent{v1.TagEvent{Created:v1.Time{Time:time.Time{wall:0x0, ext:63729988386, loc:(*time.Location)(0x2c752c0)}}, DockerImageReference:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:21b2a2930c6afe8654b2d70f97b7f19ac741090d61e492c0783213f85f0dea8b\", Image:\"sha256:21b2a2930c6afe8654b2d70f97b7f19ac741090d61e492c0783213f85f0dea8b\", Generation:1}, v1.TagEvent{Created:v1.Time{Time:time.Time{wall:0x0, ext:63729988304, loc:(*time.Location)(0x2c752c0)}}, DockerImageReference:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:94b123a897a35f27ba6ba0e493537a336b344a045ca23c1b003639c0c1a17539\", Image:\"sha256:94b123a897a35f27ba6ba0e493537a336b344a045ca23c1b003639c0c1a17539\", Generation:1}, v1.TagEvent{Created:v1.Time{Time:time.Time{wall:0x0, ext:63729988302, loc:(*time.Location)(0x2c752c0)}}, DockerImageReference:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c\", Image:\"sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c\", Generation:1}}, Conditions:[]v1.TagEventCondition(nil)}}}" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:39" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] internal registry: \"image-registry.openshift-image-registry.svc:5000\"" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:50" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] Backing up tag: \"latest\"" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:55" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] copying from: docker://image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:78" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] copying to: docker://oadp-default-aws-registry-route-oadp-operator.apps.cluster-jgabani0518.jgabani0518.mg.dog8code.com/nginx-example/cakephp-ex:latest" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:79" pluginName=velero-plugins
time="2020-07-29T16:19:20Z" level=info msg="[is-backup] src image digest: sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:91" pluginName=velero-plugins
time="2020-07-29T16:19:20Z" level=info msg="[is-backup] manifest of copied image: {\"schemaVersion\":2,\"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\",\"config\":{\"mediaType\":\"application/vnd.docker.container.image.v1+json\",\"size\":13222,\"digest\":\"sha256:89d64a8c7b52e10bf1ec0f00122fdc2613436311976f7e802b58a724cea89ae4\"},\"layers\":[{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":76275160,\"digest\":\"sha256:a3ac36470b00df382448e79f7a749aa6833e4ac9cc90e3391f778820db9fa407\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":1598,\"digest\":\"sha256:82a8f4ea76cb6f833c5f179b3e6eda9f2267ed8ac7d1bf652f88ac3e9cc453d1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":7214297,\"digest\":\"sha256:a0672674b2e3d7610a4b45a54747607b7fc9b87940a478e913c04c46bc889ba1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":87860405,\"digest\":\"sha256:6dda4188fba3c3ff2c487dde3823bb1ad79af356362c1513e0ef139bade8896d\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":47584092,\"digest\":\"sha256:3c6372d310adcb9560e25b325a604de73c72204bfd6e72bf7929193636fc4138\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":13967829,\"digest\":\"sha256:bcc8001a83a82e190a8b7cacd8bf79ffa366921885c49f6cac19ae815f57e4ae\"}]}" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:102" pluginName=velero-plugins
```

#### Restore Plugin

- **Resources**: imagestreams
- **Actions**:
  - Copies images from migration registry to destination internal registry
  - Handles namespace mapping for cross-namespace image references
  - Updates all image references to point to destination cluster registry
  - Returns `.WithoutRestore()` to prevent direct resource restore
  - Images are imported via registry copy rather than Kubernetes API
  - Preserves image metadata and layer information during restore

```log
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] Entering ImageStream restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:30" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] image: \"cakephp-ex\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:34" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] backup internal registry: \"image-registry.openshift-image-registry.svc:5000\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:52" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] restore internal registry: \"image-registry.openshift-image-registry.svc:5000\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:53" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] Restoring tag: \"latest\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:56" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] copying from: docker://oadp-default-aws-registry-route-oadp-operator.apps.cluster-jgabani0518.jgabani0518.mg.dog8code.com/patroni/cakephp-ex@sha256:5469df614f9531e3834dea41d09360349030a3b4e58cf6a81f767878865fd91b" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:91" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] copying to: docker://image-registry.openshift-image-registry.svc:5000/patroni/cakephp-ex:latest" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:92" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:18Z" level=info msg="[is-restore] manifest of copied image: {\"schemaVersion\":2,\"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\",\"config\":{\"mediaType\":\"application/vnd.docker.container.image.v1+json\",\"size\":13237,\"digest\":\"sha256:67a334d247ca444d8924b11b229e2b625b31e749852d0b4090745847961a2dc6\"},\"layers\":[{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":76275160,\"digest\":\"sha256:a3ac36470b00df382448e79f7a749aa6833e4ac9cc90e3391f778820db9fa407\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":1598,\"digest\":\"sha256:82a8f4ea76cb6f833c5f179b3e6eda9f2267ed8ac7d1bf652f88ac3e9cc453d1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":7214297,\"digest\":\"sha256:a0672674b2e3d7610a4b45a54747607b7fc9b87940a478e913c04c46bc889ba1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":87860405,\"digest\":\"sha256:6dda4188fba3c3ff2c487dde3823bb1ad79af356362c1513e0ef139bade8896d\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":47584092,\"digest\":\"sha256:3c6372d310adcb9560e25b325a604de73c72204bfd6e72bf7929193636fc4138\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":13963676,\"digest\":\"sha256:f6fd97e4baabbfbe0e7cf564e4259f8695df847a833e332c6c510bfb158f5026\"}]}" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:98" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Image Stream Tag

#### Backup Plugin

- **Resources**: imagestreamtags
- **Actions**:
  - Identifies reference tags that point to other images or tags
  - Adds annotations for related tags:
    - `openshift.io/related-istag`: Name of related tag
    - `openshift.io/related-istag-ns`: Namespace of related tag

#### Restore Plugin

- **Resources**: imagestreamtags
- **Actions**:
  - Restores reference tags and external image references
  - Skips tags for images imported by imagestream (they're recreated automatically)
  - Handles namespace mapping for cross-namespace references
  - Waits for dependent tags to be created if needed
  - Updates tag references to match destination cluster structure

### Image Tag

#### Restore Plugin

- **Resources**: imagetags
- **Actions**:
  - Completely skips restore (returns `.WithoutRestore()`)
  - Image tags are automatically created by OpenShift

### Job

#### Restore Plugin

- **Resources**: jobs
- **Actions**:
  - Updates container image references from backup to restore registry
  - Skips restore if job is owned by a CronJob (CronJob will recreate it)
  - Handles both init containers and regular containers

### Persistent Volume

#### Backup Plugin

- **Resources**: persistentvolumes
- **Actions**:
  - Only processes PVs marked for migration (checks `app.kubernetes.io/part-of: openshift-migration` label)
  - Changes reclaim policy to Retain for migrated PVs
  - Stores original reclaim policy in `openshift.io/original-reclaim-policy` annotation

```log
time="2020-07-29T17:02:37Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
time="2020-07-29T17:02:40Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
time="2020-07-29T17:02:42Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
time="2020-07-29T17:02:44Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
```

#### Restore Plugin

- **Resources**: persistentvolumes
- **Actions**:
  - Only processes PVs marked for migration
  - Updates storage class based on `openshift.io/migration-storage-class` annotation
  - Handles both GA and beta storage class annotations
  - Skips snapshot-based PVs during stage restores

```log
time="2020-07-29T18:51:02Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Persistent Volume Claim

#### Restore Plugin

- **Resources**: persistentvolumeclaims
- **Actions**:
  - Removes volume.kubernetes.io/selected-node annotation
  - For migration workflow (with migration labels):
    - Removes label selectors to prevent provisioning conflicts
    - Updates storage class based on annotations
    - Adds additional access modes from `openshift.io/migration-access-mode` annotation
    - Handles both GA and beta storage class annotations

```log
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Pod

#### Backup Plugin

- **Resources**: pods
- **Actions**:
  - Adds `openshift.io/dc-includes-dm-fix: true` annotation for DeploymentConfig disconnect tracking

#### Restore Plugin

- **Resources**: pods
- **Actions**:
  - Skips restore of build pods
  - Removes node selectors to prevent scheduling conflicts
  - For pods owned by DeploymentConfigs with volumes or restore hooks:
    - Disconnects pod from DC by removing labels
    - Adds `oadp.openshift.io/disconnected-from-dc: true` label
  - Updates all container image references from backup to restore registry
  - Waits for and updates pull secrets for destination cluster
  - For stage migrations:
    - Handles special image swapping for stage pods
    - Clears node affinity for stage restore

### Replica Set

#### Restore Plugin

- **Resources**: replicasets
- **Actions**:
  - Updates all container image references from backup to restore registry
  - Handles both init containers and regular containers

### Replication Controller

#### Backup Plugin

- **Resources**: replicationcontrollers
- **Actions**:
  - Checks if RC is owned by a paused DeploymentConfig
  - Adds `openshift.io/paused-owner-ref: true` annotation if owner DC is paused

#### Restore Plugin

- **Resources**: replicationcontrollers
- **Actions**:
  - Updates all container image references from backup to restore registry
  - Handles both init containers and regular containers

### Role Binding

#### Restore Plugin

- **Resources**: rolebindings
- **Actions**:
  - Skips restore of system rolebindings ("system:image-pullers", "system:image-builders", "system:deployers") as these are automatically created by OpenShift
  - Updates namespaces in subjects when namespace mapping is enabled
  - Handles ServiceAccount subject namespace mapping
  - Preserves role references while updating namespace contexts
  - If restore namespace mapping is enabled, then the namespaces in RoleRef.Namespace, usernames, groupnames, and subjects are swapped accordingly

### Route

#### Restore Plugin

- **Resources**: routes
- **Actions**:
  - Clears the host field to allow destination cluster to generate new hostname
  - Lets OpenShift assign appropriate route URL for the destination cluster

### Security Context Constraints (SCC)

#### Restore Plugin

- **Resources**: securitycontextconstraints
- **Actions**:
  - Updates ServiceAccount usernames when namespace mapping is enabled
  - Adjusts user references to match destination cluster namespace structure

### Secret

#### Restore Plugin

- **Resources**: secrets
- **Actions**:
  - Updates namespace in secret data for service-account-token secrets
  - Handles namespace mapping for token data

### Service

#### Restore Plugin

- **Resources**: services
- **Actions**:
  - Clears externalIPs field for LoadBalancer services
  - Allows destination cluster to assign new external IPs

### Service Account

#### Backup Plugin

- **Resources**: serviceaccounts
- **Actions**:
  - Discovers SecurityContextConstraints (SCCs) that reference the service account
  - Returns discovered SCCs as additional items to be backed up
  - Ensures SCCs are included when backing up service accounts

#### Restore Plugin

- **Resources**: serviceaccounts
- **Actions**:
  - Removes references to dockercfg secrets (cluster will regenerate them)
  - Preserves other secrets and image pull secrets
  - Allows OpenShift to create new registry authentication secrets

### Stateful Set

#### Restore Plugin

- **Resources**: statefulsets
- **Actions**:
  - Updates all container image references from backup to restore registry
  - Handles both init containers and regular containers

## Special Features

### Non-Admin Controller (NAC) Support

The plugin includes support for non-admin backups:

- **nonadmin/restore.go**: Skips restore of resources with `oadp.openshift.io/skip-restore: true` annotation
- Prevents non-admin users from restoring certain cluster-scoped resources

### Item Block Actions

- **serviceaccount/itemblock.go**: Excludes temporary service accounts from backup when they have the `openshift.io/temp-service-account: true` annotation. The plugin returns `true` to block these service accounts from being included in the backup.
