# AGENTS.md — AI Agent Instructions for openshift/openshift-velero-plugin

## Project Overview
The OpenShift Velero Plugin provides Velero backup and restore plugins specifically designed for OpenShift resources. It handles OpenShift-specific resource types during backup/restore operations, including proper handling of security contexts, image streams, build configs, deployment configs, routes, and other OpenShift-native resources.

- **Primary Language**: Go
- **Module**: `github.com/konveyor/openshift-velero-plugin`
- **Default Branch**: `oadp-dev`

## Build Instructions
```bash
# Build all plugin binaries
make all

# Build a specific plugin binary
make build

# Build container image
make container
```

## Test Instructions
```bash
# Run all tests (uses envtest)
make test

# Run CI checks (build + test)
make ci

# Run specific tests
go test ./velero-plugins/... -run TestName

# Vet code
go vet ./...
```

## Code Conventions
- All plugin implementations in `velero-plugins/` directory
- Each OpenShift resource type has its own plugin file
- Follow Velero plugin interface patterns (BackupItemAction, RestoreItemAction)
- Uses envtest for controller testing
- OpenShift API types used extensively

## Project Structure
```
velero-plugins/        - All plugin implementations
  build/               - BuildConfig backup/restore
  common/              - Shared utilities across plugins
  daemonset/           - DaemonSet handling
  deployment/          - Deployment handling
  deploymentconfig/    - DeploymentConfig handling
  imagestream/         - ImageStream handling
  imagestreamtag/      - ImageStreamTag handling
  pod/                 - Pod handling
  replicaset/          - ReplicaSet handling
  role/                - Role/RoleBinding handling
  route/               - Route handling
  scc/                 - SecurityContextConstraints handling
  service/             - Service handling
  serviceaccount/      - ServiceAccount handling
  statefulset/         - StatefulSet handling
```

## CI/CD
- Prow CI for OpenShift integration
- Reproduce CI locally:
  ```bash
  make ci
  ```

## Common Tasks

### Adding a plugin for a new OpenShift resource type
1. Create a new directory under `velero-plugins/` for the resource
2. Implement the `BackupItemAction` and/or `RestoreItemAction` interfaces
3. Register the plugin in the main binary
4. Add tests using envtest
5. Build and test: `make ci`

### Modifying resource handling during backup/restore
1. Find the relevant plugin in `velero-plugins/<resource>/`
2. Modify the `AppliesTo()`, `Execute()` methods
3. Update shared logic in `velero-plugins/common/` if needed
4. Run `make test` to validate
