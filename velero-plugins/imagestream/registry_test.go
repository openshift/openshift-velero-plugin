package imagestream

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	testProfile            = "someProfile"
	testAccessKey          = "someAccessKey"
	testSecretAccessKey    = "someSecretAccessKey"
	testStoragekey         = "someStorageKey"
	testCloudName          = "someCloudName"
	testBslProfile         = "bslProfile"
	testBslAccessKey       = "bslAccessKey"
	testBslSecretAccessKey = "bslSecretAccessKey"
	testBslRoleArn         = "bslRoleArn"
	testSubscriptionID     = "someSubscriptionID"
	testTenantID           = "someTenantID"
	testClientID           = "someClientID"
	testClientSecret       = "someClientSecret"
	testResourceGroup      = "someResourceGroup"
)

var (
	secretData = map[string][]byte{
		"cloud": []byte(
			"\n[" + testBslProfile + "]\n" +
				"aws_access_key_id=" + testBslAccessKey + "\n" +
				"aws_secret_access_key=" + testBslSecretAccessKey +
				"\n[default]" + "\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey +
				"\n[test-profile]\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey,
		),
	}
	secretDataWithEqualInSecret = map[string][]byte{
		"cloud": []byte(
			"\n[" + testBslProfile + "]\n" +
				"aws_access_key_id=" + testBslAccessKey + "\n" +
				"aws_secret_access_key=" + testBslSecretAccessKey + "=" + testBslSecretAccessKey +
				"\n[default]" + "\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey + "=" + testSecretAccessKey +
				"\n[test-profile]\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey + "=" + testSecretAccessKey,
		),
	}
	secretDataWithCarriageReturnInSecret = map[string][]byte{
		"cloud": []byte(
			"\n[" + testBslProfile + "]\r\n" +
				"aws_access_key_id=" + testBslAccessKey + "\n" +
				"aws_secret_access_key=" + testBslSecretAccessKey + "=" + testBslSecretAccessKey +
				"\n[default]" + "\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey + "=" + testSecretAccessKey +
				"\r\n[test-profile]\n" +
				"aws_access_key_id=" + testAccessKey + "\r\n" +
				"aws_secret_access_key=" + testSecretAccessKey + "=" + testSecretAccessKey,
		),
	}
	secretDataWithMixedQuotesAndSpacesInSecret = map[string][]byte{
		"cloud": []byte(
			"\n[" + testBslProfile + "]\n" +
				"aws_access_key_id =" + testBslAccessKey + "\n" +
				" aws_secret_access_key=" + "\" " + testBslSecretAccessKey + "\"" +
				"\n[default]" + "\n" +
				" aws_access_key_id= " + testAccessKey + "\n" +
				"aws_secret_access_key =" + "'" + testSecretAccessKey + " '" +
				"\n[test-profile]\n" +
				"aws_access_key_id =" + testAccessKey + "\n" +
				"aws_secret_access_key=" + "\" " + testSecretAccessKey + "\"",
		),
	}
	awsSecretDataWithMissingProfile = map[string][]byte{
		"cloud": []byte(
			"[default]" + "\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey +
				"\n[test-profile]\n" +
				"aws_access_key_id=" + testAccessKey + "\n" +
				"aws_secret_access_key=" + testSecretAccessKey,
		),
	}
	secretAzureData = map[string][]byte{
		"cloud": []byte("[default]" + "\n" +
			"AZURE_STORAGE_ACCOUNT_ACCESS_KEY=" + testStoragekey + "\n" +
			"AZURE_CLOUD_NAME=" + testCloudName),
	}
	secretAzureServicePrincipalData = map[string][]byte{
		"cloud": []byte("[default]" + "\n" +
			"AZURE_STORAGE_ACCOUNT_ACCESS_KEY=" + testStoragekey + "\n" +
			"AZURE_CLOUD_NAME=" + testCloudName + "\n" +
			"AZURE_SUBSCRIPTION_ID=" + testSubscriptionID + "\n" +
			"AZURE_TENANT_ID=" + testTenantID + "\n" +
			"AZURE_CLIENT_ID=" + testClientID + "\n" +
			"AZURE_CLIENT_SECRET=" + testClientSecret + "\n" +
			"AZURE_RESOURCE_GROUP=" + testResourceGroup),
	}
	awsRegistrySecretData = map[string][]byte{
		"access_key": []byte(testBslAccessKey),
		"secret_key": []byte(testBslSecretAccessKey),
	}
	awsStsRegistrySecretData = map[string][]byte{
		"cloud": []byte(`role_arn=testBslRoleArn
web_identity_token_file=/var/run/secrets/some/path
`),
	}
	azureRegistrySecretData = map[string][]byte{
		"client_id_key":       []byte(""),
		"client_secret_key":   []byte(""),
		"resource_group_key":  []byte(""),
		"storage_account_key": []byte(testStoragekey),
		"subscription_id_key": []byte(""),
		"tenant_id_key":       []byte(""),
	}
	azureRegistrySPSecretData = map[string][]byte{
		"client_id_key":       []byte(testClientID),
		"client_secret_key":   []byte(testClientSecret),
		"resource_group_key":  []byte(testResourceGroup),
		"storage_account_key": []byte(testStoragekey),
		"subscription_id_key": []byte(testSubscriptionID),
		"tenant_id_key":       []byte(testTenantID),
	}
)

var testAWSEnvVar = cloudProviderEnvVarMap["aws"]
var testAzureEnvVar = cloudProviderEnvVarMap["azure"]
var testGCPEnvVar = cloudProviderEnvVarMap["gcp"]

func Test_getAWSRegistryEnvVars(t *testing.T) {
	tests := []struct {
		name                        string
		bsl                         *velerov1.BackupStorageLocation
		wantRegistryContainerEnvVar []corev1.EnvVar
		wantProfile                 string
		secret                      *corev1.Secret
		registrySecret              *corev1.Secret
		wantErr                     bool
		matchProfile                bool
	}{
		{
			name: "given aws bsl, appropriate env var for the container are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AWSProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "aws-bucket",
						},
					},
					Config: map[string]string{
						Region:                "aws-region",
						S3URL:                 "https://sr-url-aws-domain.com",
						InsecureSkipTLSVerify: "false",
						Profile:               "test-profile",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: secretData,
			},
			registrySecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oadp-test-bsl-aws-registry-secret",
					Namespace: "test-ns",
				},
				Data: awsRegistrySecretData,
			},
			wantProfile:  "test-profile",
			matchProfile: true,
		},
		{
			name: "given aws bsl without region, appropriate env var for the container with region are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AWSProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "tkaovila-aug30-velero-bsl",
						},
					},
					Config: map[string]string{
						Profile:               "test-profile",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: secretData,
			},
			registrySecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oadp-test-bsl-aws-registry-secret",
					Namespace: "test-ns",
				},
				Data: awsRegistrySecretData,
			},
			wantRegistryContainerEnvVar: []corev1.EnvVar{
				{
					Name:  RegistryStorageEnvVarKey,
					Value: S3,
				},
				{
					Name:  RegistryStorageS3BucketEnvVarKey,
					Value: "tkaovila-aug30-velero-bsl",
				},
				{
					Name:  RegistryStorageS3RegionEnvVarKey,
					Value: "us-east-1",
				},
				{
					Name:  RegistryStorageS3RegionendpointEnvVarKey,
					Value: "",
				},
				{
					Name:  RegistryStorageS3SkipverifyEnvVarKey,
					Value: "",
				},
				{
					Name: RegistryStorageS3AccesskeyEnvVarKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-test-bsl-aws-registry-secret"},
							Key:                  "access_key",
						},
					},
				},
				{
					Name: RegistryStorageS3SecretkeyEnvVarKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-test-bsl-aws-registry-secret"},
							Key:                  "secret_key",
						},
					},
				},
			},
			wantProfile:  "test-profile",
			matchProfile: true,
		},
		{
			name: "given aws sts bsl, appropriate env var for the container are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AWSProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "aws-bucket",
						},
					},
					Config: map[string]string{
						Region:             "aws-region",
						Profile:            "test-profile",
						enableSharedConfig: "true",
					},
					Credential: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "cloud-credentials-sts",
						},
						Key: "cloud",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-sts",
					Namespace: "test-ns",
				},
				Data: awsStsRegistrySecretData,
			},
			wantProfile: "test-profile",
			wantRegistryContainerEnvVar: []corev1.EnvVar{
				{
					Name:  RegistryStorageEnvVarKey,
					Value: S3,
				},
				{
					Name:  RegistryStorageS3BucketEnvVarKey,
					Value: "aws-bucket",
				},
				{
					Name: RegistryStorageS3RegionEnvVarKey, Value: "aws-region",
				},
				{
					Name: RegistryStorageS3RegionendpointEnvVarKey,
				},
				{
					Name: RegistryStorageS3SkipverifyEnvVarKey,
				},
				{
					Name: RegistryStorageS3CredentialsConfigPathEnvVarKey, Value: "/tmp/credentials/test-ns/cloud-credentials-sts-cloud",
				},
			},

			matchProfile: true,
		},
		{
			name: "given aws profile in bsl, appropriate env var for the container are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AWSProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "aws-bucket",
						},
					},
					Config: map[string]string{
						Region:                "aws-region",
						S3URL:                 "https://sr-url-aws-domain.com",
						InsecureSkipTLSVerify: "false",
						Profile:               testBslProfile,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: secretData,
			},
			registrySecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oadp-test-bsl-aws-registry-secret",
					Namespace: "test-ns",
				},
				Data: awsRegistrySecretData,
			},
			wantProfile:  testBslProfile,
			matchProfile: true,
		},
		{
			name: "given missing aws profile in bsl, env var should not match",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AWSProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "aws-bucket",
						},
					},
					Config: map[string]string{
						Region:                "aws-region",
						S3URL:                 "https://sr-url-aws-domain.com",
						InsecureSkipTLSVerify: "false",
						Profile:               testBslProfile,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials",
					Namespace: "test-ns",
				},
				Data: awsSecretDataWithMissingProfile,
			},
			registrySecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oadp-test-bsl-aws-registry-secret",
					Namespace: "test-ns",
				},
				Data: awsRegistrySecretData,
			},
			wantProfile:  testBslProfile,
			matchProfile: false,
		},
	}
	testEnv := &envtest.Environment{}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer testEnv.Stop()
	clients.SetInClusterConfig(cfg)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cv1c, err := corev1client.NewForConfig(cfg)
			if err != nil {
				t.Fatal(err)
			}
			if tt.secret != nil {
				cv1c.Namespaces().Create(context.Background(), &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.secret.Namespace,
					},
				}, metav1.CreateOptions{})
				cv1c.Secrets(tt.secret.Namespace).Create(context.Background(), tt.secret, metav1.CreateOptions{})
			}
			if tt.registrySecret != nil {
				cv1c.Namespaces().Create(context.Background(), &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: tt.registrySecret.Namespace,
					},
				}, metav1.CreateOptions{})
				cv1c.Secrets(tt.registrySecret.Namespace).Create(context.Background(), tt.registrySecret, metav1.CreateOptions{})
			}
			defer func() {
				// envtest quirks is we cannot cleanup via namespace deletion
				// https://github.com/kubernetes-sigs/controller-runtime/issues/880#issuecomment-615517902
				// so delete the secret
				if tt.secret != nil {
					cv1c.Secrets(tt.secret.Namespace).Delete(context.Background(), tt.secret.Name, metav1.DeleteOptions{})
				}
				if tt.registrySecret != nil {
					cv1c.Secrets(tt.registrySecret.Namespace).Delete(context.Background(), tt.registrySecret.Name, metav1.DeleteOptions{})
				}
			}()
			if tt.wantRegistryContainerEnvVar == nil {
				tt.wantRegistryContainerEnvVar = []corev1.EnvVar{
					{
						Name:  RegistryStorageEnvVarKey,
						Value: S3,
					},
					{
						Name:  RegistryStorageS3BucketEnvVarKey,
						Value: "aws-bucket",
					},
					{
						Name:  RegistryStorageS3RegionEnvVarKey,
						Value: "aws-region",
					},
					{
						Name:  RegistryStorageS3RegionendpointEnvVarKey,
						Value: "https://sr-url-aws-domain.com",
					},
					{
						Name:  RegistryStorageS3SkipverifyEnvVarKey,
						Value: "false",
					},
					{
						Name: RegistryStorageS3AccesskeyEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "access_key",
							},
						},
					},
					{
						Name: RegistryStorageS3SecretkeyEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "secret_key",
							},
						},
					},
				}
			}
			if tt.wantProfile == testBslProfile {
				tt.wantRegistryContainerEnvVar = []corev1.EnvVar{
					{
						Name:  RegistryStorageEnvVarKey,
						Value: S3,
					},
					{
						Name:  RegistryStorageS3BucketEnvVarKey,
						Value: "aws-bucket",
					},
					{
						Name:  RegistryStorageS3RegionEnvVarKey,
						Value: "aws-region",
					},
					{
						Name:  RegistryStorageS3RegionendpointEnvVarKey,
						Value: "https://sr-url-aws-domain.com",
					},
					{
						Name:  RegistryStorageS3SkipverifyEnvVarKey,
						Value: "false",
					},
					{
						Name: RegistryStorageS3AccesskeyEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "access_key",
							},
						},
					},
					{
						Name: RegistryStorageS3SecretkeyEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "secret_key",
							},
						},
					},
				}
			}

			gotRegistryContainerEnvVar, gotErr := getAWSRegistryEnvVars(tt.bsl)

			if tt.matchProfile && (gotErr != nil) != tt.wantErr {
				t.Errorf("ValidateBackupStorageLocations() gotErr = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			if tt.matchProfile && !reflect.DeepEqual(tt.wantRegistryContainerEnvVar, gotRegistryContainerEnvVar) {
				t.Errorf("expected registry container env var has diff %s", cmp.Diff(tt.wantRegistryContainerEnvVar, gotRegistryContainerEnvVar))
			}
		})
	}
}

func Test_getAzureRegistryEnvVars(t *testing.T) {
	tests := []struct {
		name                        string
		bsl                         *velerov1.BackupStorageLocation
		wantRegistryContainerEnvVar []corev1.EnvVar
		secret                      *corev1.Secret
		registrySecret              *corev1.Secret
		wantErr                     bool
		wantProfile                 string
		matchProfile                bool
	}{
		{
			name: "given azure bsl, appropriate env var for the container are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AzureProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "azure-bucket",
						},
					},
					Config: map[string]string{
						StorageAccount:                           "velero-azure-account",
						ResourceGroup:                            testResourceGroup,
						RegistryStorageAzureAccountnameEnvVarKey: "velero-azure-account",
						"storageAccountKeyEnvVar":                "AZURE_STORAGE_ACCOUNT_ACCESS_KEY",
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: secretAzureData,
			},
			registrySecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oadp-test-bsl-azure-registry-secret",
					Namespace: "test-ns",
				},
				Data: azureRegistrySecretData,
			},
			wantProfile:  "test-profile",
			matchProfile: true,
		},
		{
			name: "given azure bsl & SP credentials, appropriate env var for the container are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: AzureProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "azure-bucket",
						},
					},
					Config: map[string]string{
						StorageAccount:   "velero-azure-account",
						ResourceGroup:    testResourceGroup,
						"subscriptionId": testSubscriptionID,
					},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-azure",
					Namespace: "test-ns",
				},
				Data: secretAzureServicePrincipalData,
			},
			registrySecret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "oadp-test-bsl-azure-registry-secret",
					Namespace: "test-ns",
				},
				Data: azureRegistrySPSecretData,
			},
			wantProfile:  "test-sp-profile",
			matchProfile: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantRegistryContainerEnvVar = []corev1.EnvVar{
				{
					Name:  RegistryStorageEnvVarKey,
					Value: Azure,
				},
				{
					Name:  RegistryStorageAzureContainerEnvVarKey,
					Value: "azure-bucket",
				},
				{
					Name:  RegistryStorageAzureAccountnameEnvVarKey,
					Value: "velero-azure-account",
				},
				{
					Name: RegistryStorageAzureAccountkeyEnvVarKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
							Key:                  "storage_account_key",
						},
					},
				},
				{
					Name:  RegistryStorageAzureAADEndpointEnvVarKey,
					Value: "",
				},
				{
					Name: RegistryStorageAzureSPNClientIDEnvVarKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
							Key:                  "client_id_key",
						},
					},
				},
				{
					Name: RegistryStorageAzureSPNClientSecretEnvVarKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
							Key:                  "client_secret_key",
						},
					},
				},
				{
					Name: RegistryStorageAzureSPNTenantIDEnvVarKey,
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
							Key:                  "tenant_id_key",
						},
					},
				},
			}
			if tt.wantProfile == "test-sp-profile" {
				tt.wantRegistryContainerEnvVar = []corev1.EnvVar{
					{
						Name:  RegistryStorageEnvVarKey,
						Value: Azure,
					},
					{
						Name:  RegistryStorageAzureContainerEnvVarKey,
						Value: "azure-bucket",
					},
					{
						Name:  RegistryStorageAzureAccountnameEnvVarKey,
						Value: "velero-azure-account",
					},
					{
						Name: RegistryStorageAzureAccountkeyEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "storage_account_key",
							},
						},
					},
					{
						Name:  RegistryStorageAzureAADEndpointEnvVarKey,
						Value: "",
					},
					{
						Name: RegistryStorageAzureSPNClientIDEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "client_id_key",
							},
						},
					},
					{
						Name: RegistryStorageAzureSPNClientSecretEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "client_secret_key",
							},
						},
					},
					{
						Name: RegistryStorageAzureSPNTenantIDEnvVarKey,
						ValueFrom: &corev1.EnvVarSource{
							SecretKeyRef: &corev1.SecretKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{Name: "oadp-" + tt.bsl.Name + "-" + tt.bsl.Spec.Provider + "-registry-secret"},
								Key:                  "tenant_id_key",
							},
						},
					},
				}
			}

			gotRegistryContainerEnvVar, gotErr := getAzureRegistryEnvVars(tt.bsl, testAzureEnvVar)

			if tt.matchProfile && (gotErr != nil) != tt.wantErr {
				t.Errorf("ValidateBackupStorageLocations() gotErr = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			if tt.matchProfile && !reflect.DeepEqual(tt.wantRegistryContainerEnvVar, gotRegistryContainerEnvVar) {
				t.Errorf("expected registry container env var to be %#v, got %#v", tt.wantRegistryContainerEnvVar, gotRegistryContainerEnvVar)
			}
		})
	}
}

func Test_getGCPRegistryEnvVars(t *testing.T) {
	tests := []struct {
		name                        string
		bsl                         *velerov1.BackupStorageLocation
		wantRegistryContainerEnvVar []corev1.EnvVar
		secret                      *corev1.Secret
		wantErr                     bool
	}{
		{
			name: "given gcp bsl, appropriate env var for the container are returned",
			bsl: &velerov1.BackupStorageLocation{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-bsl",
					Namespace: "test-ns",
				},
				Spec: velerov1.BackupStorageLocationSpec{
					Provider: GCPProvider,
					StorageType: velerov1.StorageType{
						ObjectStorage: &velerov1.ObjectStorageLocation{
							Bucket: "gcp-bucket",
						},
					},
					Config: map[string]string{},
				},
			},
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cloud-credentials-gcp",
					Namespace: "test-ns",
				},
				Data: secretData,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.wantRegistryContainerEnvVar = []corev1.EnvVar{
				{
					Name:  RegistryStorageEnvVarKey,
					Value: GCS,
				},
				{
					Name:  RegistryStorageGCSBucket,
					Value: "gcp-bucket",
				},
				{
					Name:  RegistryStorageGCSKeyfile,
					Value: "/tmp/credentials/test-ns/cloud-credentials-gcp-cloud",
				},
			}

			gotRegistryContainerEnvVar, gotErr := getGCPRegistryEnvVars(tt.bsl)

			if (gotErr != nil) != tt.wantErr {
				// ignore errors. this test originally called InClusterConfig which would fail in a test environment
				// t.Errorf("ValidateBackupStorageLocations() gotErr = %v, wantErr %v", gotErr, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(tt.wantRegistryContainerEnvVar, gotRegistryContainerEnvVar) {
				t.Errorf("expected registry container env var to be %#v, got %#v", tt.wantRegistryContainerEnvVar, gotRegistryContainerEnvVar)
			}
		})
	}
}
