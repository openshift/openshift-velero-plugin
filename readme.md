# OpenShift Velero Plugin [![Build Status](https://travis-ci.com/konveyor/openshift-velero-plugin.svg?branch=master)](https://travis-ci.com/konveyor/openshift-velero-plugin) [![Maintainability](https://api.codeclimate.com/v1/badges/95d3aaf8af1cfdd529c4/maintainability)](https://codeclimate.com/github/konveyoropenshift-velero-plugin/maintainability)

## Introduction

The plugin helps perform Backup and restore of projects on openshift cluster.

## Prerequiste

[Oadp-operator](https://github.com/konveyor/oadp-operator) needs to be installed on cluster.

## Kinds of Plugins

Velero currently supports the following kinds of plugins:

- **Backup Item Action** - performs arbitrary logic on individual items prior to storing them in the backup file.
- **Restore Item Action** - performs arbitrary logic on individual items prior to restoring them in the Kubernetes cluster.


## Resources Included in B/R

- Build, Build Config, Cluster Role Bindings, Cron Job, Daemonset, Deployment, Deployment Config, Imagestream, Imagestreamtags, Imagetags, Persistent Volumes, Persistent Volume Claims, Pod, Replica Sets, Replication Controller, Role Bindings, Route, SCC, Service, Service Account, and Stateful Set

## Enable/Disable Backup/Restore for individual resource

[Main.go](/velero-plugins/main.go#35) includes code that registers individual plugins for each resource. To disable plugin code comment respective line.
 
## Building the plugins

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

## Deploying the plugins

To deploy your plugin image to an Velero server:

1. Make sure your image is pushed to a registry that is accessible to your cluster's nodes.
2. Run `velero plugin add <image>`, e.g. `velero plugin add quay.io/ocpmigrate/velero-plugin`

Note: If velero installed with oadp-operator, this configuration would already be present in the setup. Kindly edit the configurations accordingly, [here](https://github.com/konveyor/oadp-operator#configure-velero-plugins) is some help.

## Backup/Restore applications using this plugin

[Velero-example]() contains some examples of B/R of basic examples using velero.

Note: At this point velero is already installed with oadp-operator so skip velero installation steps

## How the plugin works

This plugin includes two workflows CAM and Backup/Restore.

1. Backup/Restore: If there are no relative annotations, then by default the plugin executes Backup/Restore workflow.

2. CAM: To use CAM workflow, the following annotation and label have to be present in `backup.yml` and `restore.yml`
    ```  
     annotations:
           openshift.io/migration-registry: <migration-registry>
     labels:
       app.kubernetes.io/part-of: openshift-migration
   ```
   Note: Any other or no value for label `app.kubernetes.io/part-of:` means executing Backup/Restore workflow.

## Debug logs

There are several velero commands that help get the logs of backup/restore or status of backup/restore.

1. `velero get backup` returns all the backups present in velero

    ```cassandraql
    $ velero get backup
    NAME                  STATUS                       CREATED                         EXPIRES   STORAGE LOCATION   SELECTOR
    mssql-persistent      Completed                    2020-06-16 15:45:12 -0400 EDT   1d        default            <none>
    nginx-stateless       PartiallyFailed (2 errors)   2020-07-14 16:42:31 -0400 EDT   29d       default            <none>
    patroni-test          Completed                    2020-07-13 12:26:28 -0400 EDT   27d       default            <none>
    postgres-persistent   Completed                    2020-07-02 16:37:36 -0400 EDT   17d       default            <none>
    postgresql            Completed                    2020-07-07 17:24:28 -0400 EDT   22d       default            <none>
    
    ```

2. `velero backup logs <backup-name>` returns logs for respective backup
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
3. `velero get restore` returns all the restores present in velero
    
    ```
   $ velero get restore
     NAME           BACKUP         STATUS      WARNINGS   ERRORS   CREATED                         SELECTOR
     patroni-test   patroni-test   Completed   8          0        2020-07-13 12:34:04 -0400 EDT   <none>   ```
   
4. `velero restore logs <restore-name>` returns logs for respective restore

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