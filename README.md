# OpenShift Velero Plugin [![Build Status](https://travis-ci.com/konveyor/openshift-velero-plugin.svg?branch=master)](https://travis-ci.com/konveyor/openshift-velero-plugin) [![Maintainability](https://api.codeclimate.com/v1/badges/95d3aaf8af1cfdd529c4/maintainability)](https://codeclimate.com/github/konveyoropenshift-velero-plugin/maintainability)

## Introduction

The OpenShift Velero Plugin helps backup and restore projects on an OpenShift cluster. 
The plugin executes additional logic (such as adding annotations and swapping namespaces) during the backup/restore process on top of what Velero executes (without the plugin). Each OpenShift or Kubernetes resource has its own backup/restore plugin.

## Prerequisites 

The [OADP Operator](https://github.com/konveyor/oadp-operator) needs to be installed on the OpenShift cluster.

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
- Set the BackupRegistryHostname annotation to the corect hostname 

#### Restore Plugin
- Set the RestoreServerVersion annotation to correct server version
- Set the RestoreRegistryHostname annotation to the corect hostname 

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
- Fetches all the images referenced by namespace from internal image registry of openshift, `image-registry.openshift-image-registry.svc:5000/`,  and push the same to to defined docker registry, `oadp-default-aws-registry-route-oadp-operator.apps.<route>`.

#### Restore Plugin 
- Fetches all the images that were pushed into registry initialized at backup time and pushes the same to internal opendhift image registry.

### Image Stream Tag
#### Restore Plugin 

### Image Tag
#### Restore Plugin 
- Set SkipRestore to true, so that Image Tags are not restored

### Persistent Volume
#### Restore Plugin 
- Don't modify the PV if the migration application label key does not map to corresponding value
- If the migrate type annotation is set to "copy":
	- Change the storage class name to Migration Storage Class annotation
	- If the Beta Storage Class annotation is not empty, then also set it to the Migration Storage Class annotation

### Persistent Volume Claim
#### Restore Plugin 
- Don't modify the PVC if the migration application label key does not map to corresponding value
- If the migrate type annotation is set to "copy":
	- Remove the label selectors from the PVC (to prevent the PV dynamic provisioner from getting stuck)
	- Change the storage class name to Migration Storage Class annotation
	- If the Beta Storage Class annotation is not empty, then also set it to the Migration Storage Class annotation
	- If the Migrate Access Modes annotation is not empty, then add it to the Access Mode spec
- Delete the PVC Selected Node annotation

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

#### Restore Plugin 

### Stateful Set
#### Restore Plugin 
- Updates internal image references from backup registry to restore registry pathnames

