package imagestream

import (
	"errors"

	"github.com/openshift/oadp-operator/pkg/credentials"

	oadpv1alpha1 "github.com/openshift/oadp-operator/api/v1alpha1"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
)

// Registry Env var keys
const (
	// AWS registry env vars
	RegistryStorageEnvVarKey                 = "REGISTRY_STORAGE"
	RegistryStorageS3AccesskeyEnvVarKey      = "REGISTRY_STORAGE_S3_ACCESSKEY"
	RegistryStorageS3BucketEnvVarKey         = "REGISTRY_STORAGE_S3_BUCKET"
	RegistryStorageS3RegionEnvVarKey         = "REGISTRY_STORAGE_S3_REGION"
	RegistryStorageS3SecretkeyEnvVarKey      = "REGISTRY_STORAGE_S3_SECRETKEY"
	RegistryStorageS3RegionendpointEnvVarKey = "REGISTRY_STORAGE_S3_REGIONENDPOINT"
	RegistryStorageS3RootdirectoryEnvVarKey  = "REGISTRY_STORAGE_S3_ROOTDIRECTORY"
	RegistryStorageS3SkipverifyEnvVarKey     = "REGISTRY_STORAGE_S3_SKIPVERIFY"
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

// provider specific object storage
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
	"gcp": {
		{
			Name:  RegistryStorageEnvVarKey,
			Value: GCS,
		},
		{
			Name:  RegistryStorageGCSBucket,
			Value: "",
		},
		{
			Name:  RegistryStorageGCSKeyfile,
			Value: "",
		},
	},
}

const (
	// Location to store secret if a file is needed
	secretTmpPrefix = "/tmp/registry-"
)

type azureCredentials struct {
	subscriptionID     string
	tenantID           string
	clientID           string
	clientSecret       string
	resourceGroup      string
	strorageAccountKey string
}

func getRegistryEnvVars(bsl *velerov1.BackupStorageLocation) ([]corev1.EnvVar, error) {
	envVar := []corev1.EnvVar{}
	provider := bsl.Spec.Provider
	var err error
	switch provider {
	case AWSProvider:
		envVar, err = getAWSRegistryEnvVars(bsl)

	case AzureProvider:
		envVar, err = getAzureRegistryEnvVars(bsl, cloudProviderEnvVarMap[AzureProvider])

	case GCPProvider:
		envVar, err = getGCPRegistryEnvVars(bsl, cloudProviderEnvVarMap[GCPProvider])
	default:
		return nil, errors.New("unsupported provider")
	}
	if err != nil {
		return nil, err
	}
	return envVar, nil
}

func getAWSRegistryEnvVars(bsl *velerov1.BackupStorageLocation) ([]corev1.EnvVar, error) {
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
			Name: RegistryStorageS3AccesskeyEnvVarKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
					Key:                  "access_key",
				},
			},
		},
		{
			Name: RegistryStorageS3BucketEnvVarKey,
			Value: bsl.Spec.StorageType.ObjectStorage.Bucket,
		},
		{
			Name: RegistryStorageS3RegionEnvVarKey,
			Value: bslSpecRegion,
		},
		{
			Name: RegistryStorageS3SecretkeyEnvVarKey,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + bsl.Name + "-" + bsl.Spec.Provider + "-registry-secret"},
					Key:                  "secret_key",
				},
			},
		},
		{
			Name: RegistryStorageS3RegionendpointEnvVarKey,
			Value: bsl.Spec.Config[S3URL],
		},
		{
			Name: RegistryStorageS3SkipverifyEnvVarKey,
			Value: bsl.Spec.Config[InsecureSkipTLSVerify],
		},
	}
	return awsEnvs, nil
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

func getGCPRegistryEnvVars(bsl *velerov1.BackupStorageLocation, gcpEnvVars []corev1.EnvVar) ([]corev1.EnvVar, error) {
	for i := range gcpEnvVars {
		if gcpEnvVars[i].Name == RegistryStorageGCSBucket {
			gcpEnvVars[i].Value = bsl.Spec.StorageType.ObjectStorage.Bucket
		}

		if gcpEnvVars[i].Name == RegistryStorageGCSKeyfile {
			// check for secret name
			secretName, secretKey := getSecretNameAndKey(&bsl.Spec, oadpv1alpha1.DefaultPluginGCP)
			// get secret value and save it to /tmp/registry-<secretName>

			secretEnvVarSource := &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Key:                  secretKey,
			}
			secretData, err := getSecretKeyRefData(secretEnvVarSource, bsl.Namespace)
			if err != nil {
				return nil, err
			}
			// write secret data to /tmp/registry-<secretName>
			secretPath := secretTmpPrefix + secretName
			err = saveDataToFile(secretData, secretPath)
			if err != nil {
				return nil, err
			}
			gcpEnvVars[i].Value = secretPath
		}
	}
	return gcpEnvVars, nil
}

func getSecretNameAndKey(bslSpec *velerov1.BackupStorageLocationSpec, plugin oadpv1alpha1.DefaultPlugin) (string, string) {
	// Assume default values unless user has overriden them
	secretName := credentials.PluginSpecificFields[plugin].SecretName
	secretKey := credentials.PluginSpecificFields[plugin].PluginSecretKey
	if _, ok := bslSpec.Config["credentialsFile"]; ok {
		if secretName, secretKey, err :=
			credentials.GetSecretNameKeyFromCredentialsFileConfigString(bslSpec.Config["credentialsFile"]); err == nil {
			return secretName, secretKey
		}
	}
	// check if user specified the Credential Name and Key
	credential := bslSpec.Credential
	if credential != nil {
		if len(credential.Name) > 0 {
			secretName = credential.Name
		}
		if len(credential.Key) > 0 {
			secretKey = credential.Key
		}
	}

	return secretName, secretKey
}
