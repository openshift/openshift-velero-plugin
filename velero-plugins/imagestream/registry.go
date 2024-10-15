package imagestream

import (
	"fmt"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
	oadpCreds "github.com/openshift/oadp-operator/pkg/credentials"
	"github.com/pkg/errors"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
)

// Registry Env var keys
const (
	// AWS registry env vars
	RegistryStorageEnvVarKey                        = "REGISTRY_STORAGE"
	RegistryStorageS3AccesskeyEnvVarKey             = "REGISTRY_STORAGE_S3_ACCESSKEY"
	RegistryStorageS3BucketEnvVarKey                = "REGISTRY_STORAGE_S3_BUCKET"
	RegistryStorageS3RegionEnvVarKey                = "REGISTRY_STORAGE_S3_REGION"
	RegistryStorageS3SecretkeyEnvVarKey             = "REGISTRY_STORAGE_S3_SECRETKEY"
	RegistryStorageS3CredentialsConfigPathEnvVarKey = "REGISTRY_STORAGE_S3_CREDENTIALSCONFIGPATH"
	RegistryStorageS3RegionendpointEnvVarKey        = "REGISTRY_STORAGE_S3_REGIONENDPOINT"
	RegistryStorageS3RootdirectoryEnvVarKey         = "REGISTRY_STORAGE_S3_ROOTDIRECTORY"
	RegistryStorageS3SkipverifyEnvVarKey            = "REGISTRY_STORAGE_S3_SKIPVERIFY"
	// Azure registry env vars
	RegistryStorageAzureContainerEnvVarKey       = "REGISTRY_STORAGE_AZURE_CONTAINER"
	RegistryStorageAzureAccountnameEnvVarKey     = "REGISTRY_STORAGE_AZURE_ACCOUNTNAME"
	RegistryStorageAzureAccountkeyEnvVarKey      = "REGISTRY_STORAGE_AZURE_ACCOUNTKEY"
	RegistryStorageAzureSPNClientIDEnvVarKey     = "REGISTRY_STORAGE_AZURE_SPN_CLIENT_ID"
	RegistryStorageAzureSPNClientSecretEnvVarKey = "REGISTRY_STORAGE_AZURE_SPN_CLIENT_SECRET"
	RegistryStorageAzureSPNTenantIDEnvVarKey     = "REGISTRY_STORAGE_AZURE_SPN_TENANT_ID"
	RegistryStorageAzureAADEndpointEnvVarKey     = "REGISTRY_STORAGE_AZURE_AAD_ENDPOINT"
	// GCP registry env vars
	RegistryStorageGCSBucket        = "REGISTRY_STORAGE_GCS_BUCKET"
	RegistryStorageGCSKeyfile       = "REGISTRY_STORAGE_GCS_KEYFILE"
	RegistryStorageGCSRootdirectory = "REGISTRY_STORAGE_GCS_ROOTDIRECTORY"
)

// provider specific object storage config
const (
	S3                    = "s3"
	Azure                 = "azure"
	GCS                   = "gcs"
	AWSProvider           = "aws"
	AzureProvider         = "azure"
	GCPProvider           = "gcp"
	Region                = "region"
	Profile               = "profile"
	S3URL                 = "s3Url"
	S3ForcePathStyle      = "s3ForcePathStyle"
	InsecureSkipTLSVerify = "insecureSkipTLSVerify"
	StorageAccount        = "storageAccount"
	ResourceGroup         = "resourceGroup"
	enableSharedConfig    = "enableSharedConfig"
)

// velero constants
const (
	// https://github.com/vmware-tanzu/velero/blob/5afe837f76aea4dd59b1bf2792e7802d4966f0a7/pkg/cmd/server/server.go#L110
	defaultCredentialsDirectory = "/tmp/credentials"
)

// TODO: remove this map and just define them in each function
// creating skeleton for provider based env var map
var cloudProviderEnvVarMap = map[string][]corev1.EnvVar{
	"azure": {
		{
			Name:  RegistryStorageEnvVarKey,
			Value: Azure,
		},
		{
			Name:  RegistryStorageAzureContainerEnvVarKey,
			Value: "",
		},
		{
			Name:  RegistryStorageAzureAccountnameEnvVarKey,
			Value: "",
		},
		{
			Name:  RegistryStorageAzureAccountkeyEnvVarKey,
			Value: "",
		},
		{
			Name:  RegistryStorageAzureAADEndpointEnvVarKey,
			Value: "",
		},
		{
			Name:  RegistryStorageAzureSPNClientIDEnvVarKey,
			Value: "",
		},
		{
			Name:  RegistryStorageAzureSPNClientSecretEnvVarKey,
			Value: "",
		},
		{
			Name:  RegistryStorageAzureSPNTenantIDEnvVarKey,
			Value: "",
		},
	},
}

func getRegistryEnvVars(bsl *velerov1.BackupStorageLocation) ([]corev1.EnvVar, error) {
	var envVars []corev1.EnvVar
	provider := bsl.Spec.Provider
	var err error
	switch provider {
	case AWSProvider:
		envVars, err = getAWSRegistryEnvVars(bsl)

	case AzureProvider:
		envVars, err = getAzureRegistryEnvVars(bsl, cloudProviderEnvVarMap[AzureProvider])

	case GCPProvider:
		envVars, err = getGCPRegistryEnvVars(bsl)
	default:
		return nil, errors.New("unsupported provider")
	}
	if err != nil {
		return nil, err
	}
	return envVars, nil
}

func getAWSRegistryEnvVars(bsl *velerov1.BackupStorageLocation) ([]corev1.EnvVar, error) {
	// if region is not set in bsl, then get it from bucket
	if bsl.Spec.Config == nil {
		bsl.Spec.Config = make(map[string]string)
	}
	if bsl.Spec.Config[S3URL] == ""  && bsl.Spec.Config[Region] == "" {
		var err error
		bsl.Spec.Config[Region], err = GetBucketRegion(bsl.Spec.StorageType.ObjectStorage.Bucket)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get bucket region")
		}
	}
	// validation
	bslSpecRegion, regionInConfig := bsl.Spec.Config[Region]
	if !regionInConfig {
		return nil, errors.New("region not found in backupstoragelocation spec")
	}
	// create secret data and fill up the values and return from here
	awsEnvs := []corev1.EnvVar{
		{
			Name:  RegistryStorageEnvVarKey,
			Value: S3,
		},
		{
			Name:  RegistryStorageS3BucketEnvVarKey,
			Value: bsl.Spec.StorageType.ObjectStorage.Bucket,
		},
		{
			Name:  RegistryStorageS3RegionEnvVarKey,
			Value: bslSpecRegion,
		},
		{
			Name:  RegistryStorageS3RegionendpointEnvVarKey,
			Value: bsl.Spec.Config[S3URL],
		},
		{
			Name:  RegistryStorageS3SkipverifyEnvVarKey,
			Value: bsl.Spec.Config[InsecureSkipTLSVerify],
		},
	}
	// if credential is sts, then add sts specific env vars
	if bsl.Spec.Config[enableSharedConfig] == "true" {
		awsEnvs = append(awsEnvs, corev1.EnvVar{
			Name:  RegistryStorageS3CredentialsConfigPathEnvVarKey,
			Value: getBslSecretPath(bsl),
		})
	} else {
		awsEnvs = append(awsEnvs,
			corev1.EnvVar{
				Name: RegistryStorageS3AccesskeyEnvVarKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
						Key:                  "access_key",
					},
				},
			},
			corev1.EnvVar{
				Name: RegistryStorageS3SecretkeyEnvVarKey,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
						Key:                  "secret_key",
					},
				},
			})
	}
	return awsEnvs, nil
}

// return path to cloud credentials secret file as defined by
// https://github.com/vmware-tanzu/velero/blob/5afe837f76aea4dd59b1bf2792e7802d4966f0a7/pkg/cmd/server/server.go#L334
// https://github.com/vmware-tanzu/velero/blob/5afe837f76aea4dd59b1bf2792e7802d4966f0a7/internal/credentials/file_store.go#L50
// https://github.com/vmware-tanzu/velero/blob/5afe837f76aea4dd59b1bf2792e7802d4966f0a7/internal/credentials/file_store.go#L72
// This file is written by velero server on startup
func getBslSecretPath(bsl *velerov1.BackupStorageLocation) string {
	var secretName, secretKey string
	if bsl.Spec.Credential != nil {
		secretName = bsl.Spec.Credential.LocalObjectReference.Name
		secretKey = bsl.Spec.Credential.Key
	}
	// if secretName or secretKey is not set, inherit from OADP defaults for each provider
	if bsl.Spec.Credential == nil || secretName == "" {
		secretName = oadpCreds.PluginSpecificFields[oadpv1alpha1.DefaultPlugin(bsl.Spec.Provider)].SecretName
	}
	if bsl.Spec.Credential == nil || secretKey == "" {
		secretKey = oadpCreds.PluginSpecificFields[oadpv1alpha1.DefaultPlugin(bsl.Spec.Provider)].PluginSecretKey
	}
	return fmt.Sprintf("%s/%s/%s-%s", defaultCredentialsDirectory, bsl.Namespace, secretName, secretKey)
}

func getAzureRegistryEnvVars(bsl *velerov1.BackupStorageLocation, azureEnvVars []corev1.EnvVar) ([]corev1.EnvVar, error) {

	for i := range azureEnvVars {
		if azureEnvVars[i].Name == RegistryStorageAzureContainerEnvVarKey {
			azureEnvVars[i].Value = bsl.Spec.StorageType.ObjectStorage.Bucket
		}

		if azureEnvVars[i].Name == RegistryStorageAzureAccountnameEnvVarKey {
			azureEnvVars[i].Value = bsl.Spec.Config[StorageAccount]
		}

		if azureEnvVars[i].Name == RegistryStorageAzureAccountkeyEnvVarKey {
			azureEnvVars[i].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
					Key:                  "storage_account_key",
				},
			}
		}
		if azureEnvVars[i].Name == RegistryStorageAzureSPNClientIDEnvVarKey {
			azureEnvVars[i].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
					Key:                  "client_id_key",
				},
			}
		}

		if azureEnvVars[i].Name == RegistryStorageAzureSPNClientSecretEnvVarKey {
			azureEnvVars[i].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
					Key:                  "client_secret_key",
				},
			}
		}
		if azureEnvVars[i].Name == RegistryStorageAzureSPNTenantIDEnvVarKey {
			azureEnvVars[i].ValueFrom = &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
					Key:                  "tenant_id_key",
				},
			}
		}
	}
	return azureEnvVars, nil
}

func getGCPRegistryEnvVars(bsl *velerov1.BackupStorageLocation) ([]corev1.EnvVar, error) {
	gcpEnvVars := []corev1.EnvVar{
		{
			Name:  RegistryStorageEnvVarKey,
			Value: GCS,
		},
		{
			Name:  RegistryStorageGCSBucket,
			Value: bsl.Spec.StorageType.ObjectStorage.Bucket,
		},
		{
			Name:  RegistryStorageGCSKeyfile,
			Value: getBslSecretPath(bsl),
		},
	}
	return gcpEnvVars, nil
}
