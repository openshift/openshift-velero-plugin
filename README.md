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

## Resources Included in Plugin 

- Build, Build Config, Cluster Role Binding, Cron Job, Daemonset, Deployment, Deployment Config, Image Stream, Image Stream Tag, Image Tag, Persistent Volume, Persistent Volume Claim, Pod, Replica Set, Replication Controller, Role Binding, Route, SCC, Service, Service Account, and Stateful Set

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

## Backup/Restore Applications Using the Plugin

The [velero-example](https://github.com/konveyor/velero-examples) repository contains some basic examples of backup/restore using Velero.

Note: At this point, Velero is already installed with the OADP Operator so skip the Velero installation steps.

## How the Plugin Works

This plugin includes two workflows: CAM and Backup/Restore.

1. Backup/Restore: If there are no relative annotations, then by default the plugin executes the Backup/Restore workflow.

2. CAM: To use CAM workflow, the following annotation and label have to be present in `backup.yml` and `restore.yml`:
    ```  
     annotations:
           openshift.io/migration-registry: <migration-registry>
     labels:
           app.kubernetes.io/part-of: openshift-migration
   ```
   Note: Any other or no value for the label `app.kubernetes.io/part-of:` means executing the Backup/Restore workflow.

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
    ```
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
    
    ```
   $ velero get restore
     NAME           BACKUP         STATUS      WARNINGS   ERRORS   CREATED                         SELECTOR
     patroni-test   patroni-test   Completed   8          0        2020-07-13 12:34:04 -0400 EDT   <none>   ```
   
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
These backup and restore plugins are for resources that do not need custom logic in either the backup or restore process. For example, Deployment uses the common backup plugin. 
  
#### Backup Plugin
- Set the BackupServerVersion annotation to correct server version
- Set the BackupRegistryHostname annotation to the correct hostname
- Set the MigrationRegistry annotation based on CAM or B/R workflow 
```time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
time="2020-07-29T16:19:08Z" level=info msg="[common-backup] Entering common backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/backup.go:25" pluginName=velero-plugins
```
#### Restore Plugin
- Set the RestoreServerVersion annotation to correct server version
- Set the RestoreRegistryHostname annotation to the correct hostname
- Set the MigrationRegistry annotation based on CAM or B/R workflow 

```time="2020-07-29T18:51:02Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:02Z" level=info msg="[common-restore] common restore plugin for pvc-2fbae99b-29d0-4853-a0e0-ee077ab60c18" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:29" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[common-restore] common restore plugin for pvc-64b3fc01-8d09-4f62-9612-f5d9b6ee80ff" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:29" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[common-restore] Entering common restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/common/restore.go:22" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Build
#### Restore Plugin 
- Skips restore of Build to allow Build Config to recreate it

### Build Config
#### Restore Plugin 
- Update Secrets and Docker references according to the namespace mapping 

### Cluster Role Binding 
#### Restore Plugin 
- If restore namespace mapping is enabled, then the namespaces in RoleRef.Namespace, usernames, groupnames, and subjects are swapped accordingly

### Cron Job
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames

### Daemonset
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames

### Deployment
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames

### Deployment Config
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames
- If the trigger namespace is mapped to a new one, then swap the trigger namespace accordingly

### Image Stream
#### Backup Plugin 
- Retrive internal registry and migration registry from annotaions.
- For all the tags check imagestream has any associated imagestreamtags so that we know we need to restore the tags as well.
- For all the Items in al the tags, fetch `dockerImageReference`, constructs source and destination path from `dockerImageReference` and `migrationRegistry`. Fetches all the images referenced by namespace from internal image registry of openshift, `image-registry.openshift-image-registry.svc:5000/`,  and push the same to to defined docker registry, `oadp-default-aws-registry-route-oadp-operator.apps.<route>`.

```time="2020-07-29T16:19:16Z" level=info msg="[is-backup] Entering ImageStream backup plugin" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:35" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] image: v1.ImageStream{TypeMeta:v1.TypeMeta{Kind:\"ImageStream\", APIVersion:\"image.openshift.io/v1\"}, ObjectMeta:v1.ObjectMeta{Name:\"cakephp-ex\", GenerateName:\"\", Namespace:\"nginx-example\", SelfLink:\"/apis/image.openshift.io/v1/namespaces/nginx-example/imagestreams/cakephp-ex\", UID:\"ae5f4ffa-7bfa-4081-bf77-3e767d6fcc34\", ResourceVersion:\"25571924\", Generation:1, CreationTimestamp:v1.Time{Time:time.Time{wall:0x0, ext:63729988302, loc:(*time.Location)(0x2c752c0)}}, DeletionTimestamp:(*v1.Time)(nil), DeletionGracePeriodSeconds:(*int64)(nil), Labels:map[string]string(nil), Annotations:map[string]string{\"openshift.io/backup-registry-hostname\":\"image-registry.openshift-image-registry.svc:5000\", \"openshift.io/backup-server-version\":\"1.17\", \"openshift.io/migration-registry\":\"oadp-default-aws-registry-route-oadp-operator.apps.cluster-jgabani0518.jgabani0518.mg.dog8code.com\"}, OwnerReferences:[]v1.OwnerReference(nil), Initializers:(*v1.Initializers)(nil), Finalizers:[]string(nil), ClusterName:\"\", ManagedFields:[]v1.ManagedFieldsEntry(nil)}, Spec:v1.ImageStreamSpec{LookupPolicy:v1.ImageLookupPolicy{Local:false}, DockerImageRepository:\"\", Tags:[]v1.TagReference(nil)}, Status:v1.ImageStreamStatus{DockerImageRepository:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex\", PublicDockerImageRepository:\"\", Tags:[]v1.NamedTagEventList{v1.NamedTagEventList{Tag:\"latest\", Items:[]v1.TagEvent{v1.TagEvent{Created:v1.Time{Time:time.Time{wall:0x0, ext:63729988386, loc:(*time.Location)(0x2c752c0)}}, DockerImageReference:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:21b2a2930c6afe8654b2d70f97b7f19ac741090d61e492c0783213f85f0dea8b\", Image:\"sha256:21b2a2930c6afe8654b2d70f97b7f19ac741090d61e492c0783213f85f0dea8b\", Generation:1}, v1.TagEvent{Created:v1.Time{Time:time.Time{wall:0x0, ext:63729988304, loc:(*time.Location)(0x2c752c0)}}, DockerImageReference:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:94b123a897a35f27ba6ba0e493537a336b344a045ca23c1b003639c0c1a17539\", Image:\"sha256:94b123a897a35f27ba6ba0e493537a336b344a045ca23c1b003639c0c1a17539\", Generation:1}, v1.TagEvent{Created:v1.Time{Time:time.Time{wall:0x0, ext:63729988302, loc:(*time.Location)(0x2c752c0)}}, DockerImageReference:\"image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c\", Image:\"sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c\", Generation:1}}, Conditions:[]v1.TagEventCondition(nil)}}}}" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:39" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] internal registry: \"image-registry.openshift-image-registry.svc:5000\"" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:50" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] Backing up tag: \"latest\"" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:55" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] copying from: docker://image-registry.openshift-image-registry.svc:5000/nginx-example/cakephp-ex@sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:78" pluginName=velero-plugins
time="2020-07-29T16:19:16Z" level=info msg="[is-backup] copying to: docker://oadp-default-aws-registry-route-oadp-operator.apps.cluster-jgabani0518.jgabani0518.mg.dog8code.com/nginx-example/cakephp-ex:latest" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:79" pluginName=velero-plugins
time="2020-07-29T16:19:20Z" level=info msg="[is-backup] src image digest: sha256:f6a67dc03928314bcc0cf7fd1969ae0803da5d1af03cc18ba697cd76a9cc2b5c" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:91" pluginName=velero-plugins
time="2020-07-29T16:19:20Z" level=info msg="[is-backup] manifest of copied image: {\"schemaVersion\":2,\"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\",\"config\":{\"mediaType\":\"application/vnd.docker.container.image.v1+json\",\"size\":13222,\"digest\":\"sha256:89d64a8c7b52e10bf1ec0f00122fdc2613436311976f7e802b58a724cea89ae4\"},\"layers\":[{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":76275160,\"digest\":\"sha256:a3ac36470b00df382448e79f7a749aa6833e4ac9cc90e3391f778820db9fa407\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":1598,\"digest\":\"sha256:82a8f4ea76cb6f833c5f179b3e6eda9f2267ed8ac7d1bf652f88ac3e9cc453d1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":7214297,\"digest\":\"sha256:a0672674b2e3d7610a4b45a54747607b7fc9b87940a478e913c04c46bc889ba1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":87860405,\"digest\":\"sha256:6dda4188fba3c3ff2c487dde3823bb1ad79af356362c1513e0ef139bade8896d\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":47584092,\"digest\":\"sha256:3c6372d310adcb9560e25b325a604de73c72204bfd6e72bf7929193636fc4138\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":13967829,\"digest\":\"sha256:bcc8001a83a82e190a8b7cacd8bf79ffa366921885c49f6cac19ae815f57e4ae\"}]}" backup=oadp-operator/nginx-stateless cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/backup.go:102" pluginName=velero-plugins
```

#### Restore Plugin 
- Retrive `backupInternalRegistry`, `internalRegistry`, and `migrationRegistry`.
- For all the tags check imagestream has any associated imagestreamtags, if so then, use the tag if it references an ImageStreamImage in the current namespace.
- For all the Items in al the tags, fetch `dockerImageReference`, constructs source and destination path from `migrationRegistry` and `internalRegistry`. Fetches all the images that were pushed into registry initialized at backup time and pushes the same to internal openshift image registry.

```time="2020-07-29T18:51:17Z" level=info msg="[is-restore] Entering ImageStream restore plugin" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:30" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] image: \"cakephp-ex\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:34" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] backup internal registry: \"image-registry.openshift-image-registry.svc:5000\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:52" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] restore internal registry: \"image-registry.openshift-image-registry.svc:5000\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:53" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] Restoring tag: \"latest\"" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:56" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] copying from: docker://oadp-default-aws-registry-route-oadp-operator.apps.cluster-jgabani0518.jgabani0518.mg.dog8code.com/patroni/cakephp-ex@sha256:5469df614f9531e3834dea41d09360349030a3b4e58cf6a81f767878865fd91b" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:91" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:17Z" level=info msg="[is-restore] copying to: docker://image-registry.openshift-image-registry.svc:5000/patroni/cakephp-ex:latest" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:92" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:18Z" level=info msg="[is-restore] manifest of copied image: {\"schemaVersion\":2,\"mediaType\":\"application/vnd.docker.distribution.manifest.v2+json\",\"config\":{\"mediaType\":\"application/vnd.docker.container.image.v1+json\",\"size\":13237,\"digest\":\"sha256:67a334d247ca444d8924b11b229e2b625b31e749852d0b4090745847961a2dc6\"},\"layers\":[{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":76275160,\"digest\":\"sha256:a3ac36470b00df382448e79f7a749aa6833e4ac9cc90e3391f778820db9fa407\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":1598,\"digest\":\"sha256:82a8f4ea76cb6f833c5f179b3e6eda9f2267ed8ac7d1bf652f88ac3e9cc453d1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":7214297,\"digest\":\"sha256:a0672674b2e3d7610a4b45a54747607b7fc9b87940a478e913c04c46bc889ba1\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":87860405,\"digest\":\"sha256:6dda4188fba3c3ff2c487dde3823bb1ad79af356362c1513e0ef139bade8896d\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar\",\"size\":47584092,\"digest\":\"sha256:3c6372d310adcb9560e25b325a604de73c72204bfd6e72bf7929193636fc4138\"},{\"mediaType\":\"application/vnd.docker.image.rootfs.diff.tar.gzip\",\"size\":13963676,\"digest\":\"sha256:f6fd97e4baabbfbe0e7cf564e4259f8695df847a833e332c6c510bfb158f5026\"}]}" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/imagestream/restore.go:98" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Image Stream Tag
#### Restore Plugin 
- Search for the tag corresponding to a particular imagestream to check if an image is present in the new namespace 
- If the tag is not present, look it up in the old, backup namespace and use that tag to pull the particular image required

### Image Tag
#### Restore Plugin 
- Set SkipRestore to true, so that Image Tags are not restored

### Persistent Volume
#### Backup Plugin
- Don't modify the PV if the migration application label key does not map to corresponding value
- If migrate type aanotations is set to "move":
    - Set reclaim policy to "retain" to properly move pv

```time="2020-07-29T17:02:37Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
time="2020-07-29T17:02:40Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
time="2020-07-29T17:02:42Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
time="2020-07-29T17:02:44Z" level=info msg="[pv-backup] Returning pv object as is since this is not a migration activity" backup=oadp-operator/patroni cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/backup.go:31" pluginName=velero-plugins
```

#### Restore Plugin 
- Don't modify the PV if the migration application label key does not map to corresponding value
- If the migrate type annotation is set to "copy":
	- Change the storage class name to Migration Storage Class annotation
	- If the Beta Storage Class annotation is not empty, then also set it to the Migration Storage Class annotation
```time="2020-07-29T18:51:02Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:03Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pv-restore] Returning pv object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/persistentvolume/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Persistent Volume Claim
#### Restore Plugin 
- Don't modify the PVC if the migration application label key does not map to corresponding value
- If the migrate type annotation is set to "copy":
	- Remove the label selectors from the PVC (to prevent the PV dynamic provisioner from getting stuck)
	- Change the storage class name to Migration Storage Class annotation
	- If the Beta Storage Class annotation is not empty, then also set it to the Migration Storage Class annotation
	- If the Migrate Access Modes annotation is not empty, then add it to the Access Mode spec
- Delete the PVC Selected Node annotation

```time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
time="2020-07-29T18:51:04Z" level=info msg="[pvc-restore] Returning pvc object as is since this is not a migration activity" cmd=/plugins/velero-plugins logSource="/go/src/github.com/konveyor/openshift-velero-plugin/velero-plugins/pvc/restore.go:28" pluginName=velero-plugins restore=oadp-operator/patroni
```

### Pod
#### Restore Plugin 
- Remove the node selectors from Pod (to avoid Pod being 'unschedulable' on destination)
- If the migration application label key maps to coresponding value and the Migrate Copy Phase annotation is "stage":
	- Set the Migrate Copy Phase annotation to "true"
	- Set the Affinity spec to nil
- If not:
	- If Pod has no owner references, then don't restore it
	- Update internal image references from backup registry to restore registry pathnames
 	- Update pull secrets

### Replica Set
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames
- If the Replica Set is owned by Deployment, set SkipRestore to true, so that the resource is not restored by Replica Set

### Replication Controller
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames
- If the Replication Controller is owned by Deployment Config, set SkipRestore to true, so that the resource is not restored by Replication Controller

### Role Binding
#### Restore Plugin 
- If restore namespace mapping is enabled, then the namespaces in RoleRef.Namespace, usernames, groupnames, and subjects are swapped accordingly

### Route
#### Restore Plugin 
- If the host generated annotation is set to true, then strip the source cluster host from the Route

### SCC
#### Restore Plugin 
- If restore namespace mapping is enabled, then swap namespaces in the Service account usernames 

### Service
#### Restore Plugin 
- If the Service is a LoadBalancer, then clear the external IPs

### Service Account
#### Backup Plugin 
- If there are any `SCC` references associated with service account, then include those `SCC` in backup as well. 

#### Restore Plugin 
- Copy all `Secrets` and `ImagePullSecrets` associated with a `ServiceAccount` except dockercfg secrets

### Stateful Set
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames

