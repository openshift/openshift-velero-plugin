# OpenShift Velero Plugin [![Build Status](https://travis-ci.com/konveyor/openshift-velero-plugin.svg?branch=master)](https://travis-ci.com/konveyor/openshift-velero-plugin) [![Maintainability](https://api.codeclimate.com/v1/badges/95d3aaf8af1cfdd529c4/maintainability)](https://codeclimate.com/github/konveyoropenshift-velero-plugin/maintainability)

## Kinds of Plugins

Velero currently supports the following kinds of plugins:

- **Backup Item Action** - performs arbitrary logic on individual items prior to storing them in the backup file.
- **Restore Item Action** - performs arbitrary logic on individual items prior to restoring them in the Kubernetes cluster.

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
