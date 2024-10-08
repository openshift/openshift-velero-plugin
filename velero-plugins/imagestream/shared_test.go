package imagestream

import (
	"context"
	"slices"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/yaml"
)

func TestGetRegistryEnvsForLocation(t *testing.T) {
	type args struct {
		location  string
		namespace string
	}
	tests := []struct {
		name          string
		args          args
		objs          []runtime.Object
		want          []string
		wantErr       bool
		wantErrString string
	}{
		{
			name: "simple aws case, operator created registry secret",
			args: args{
				location:  "bsl1",
				namespace: "ns1",
			},
			objs: []runtime.Object{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Namespace",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "oadp-bsl1-aws-registry-secret",
						Namespace: "ns1",
					},
					Data: map[string][]byte{
						"access_key": []byte("ak"),
						"secret_key": []byte("sk"),
					},
				},
				&velerov1.BackupStorageLocation{
					TypeMeta: metav1.TypeMeta{
						Kind:       "BackupStorageLocation",
						APIVersion: velerov1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bsl1",
						Namespace: "ns1",
					},
					Spec: velerov1.BackupStorageLocationSpec{
						Provider: "aws",
						Default:  true,
						StorageType: velerov1.StorageType{
							ObjectStorage: &velerov1.ObjectStorageLocation{
								Bucket: "buc",
								Prefix: "prefix",
							},
						},
						Credential: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "cloud-credentials",
							},
							Key: "cloud",
						},
					},
				},
			},
			want: []string{
				"REGISTRY_STORAGE=s3",
				"REGISTRY_STORAGE_S3_BUCKET=buc",
				"REGISTRY_STORAGE_S3_REGION=us-east-2",
				"REGISTRY_STORAGE_S3_ACCESSKEY=ak",
				"REGISTRY_STORAGE_S3_SECRETKEY=sk",
			},
		},
		{
			name: "aws case, operator did not create registry secret",
			args: args{
				location:  "bsl1",
				namespace: "ns1",
			},
			objs: []runtime.Object{
				&corev1.Namespace{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Namespace",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: "ns1",
					},
				},
				&velerov1.BackupStorageLocation{
					TypeMeta: metav1.TypeMeta{
						Kind:       "BackupStorageLocation",
						APIVersion: velerov1.SchemeGroupVersion.String(),
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "bsl1",
						Namespace: "ns1",
					},
					Spec: velerov1.BackupStorageLocationSpec{
						Provider: "aws",
						Default:  true,
						StorageType: velerov1.StorageType{
							ObjectStorage: &velerov1.ObjectStorageLocation{
								Bucket: "buc",
								Prefix: "prefix",
							},
						},
						Credential: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "cloud-credentials",
							},
							Key: "cloud",
						},
					},
				},
			},
			want:          []string{},
			wantErr:       true,
			wantErrString: "secrets \"oadp-bsl1-aws-registry-secret\" not found",
		},
	}
	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	metav1.AddMetaToScheme(scheme)
	velerov1.AddToScheme(scheme)
	var bslCRD apiextensionsv1.CustomResourceDefinition
	err := yaml.Unmarshal([]byte(bslCRDyaml), &bslCRD)
	if err != nil {
		panic(err)
	}
	testEnv := &envtest.Environment{
		Scheme: scheme,
		CRDs: []*apiextensionsv1.CustomResourceDefinition{
			&bslCRD,
		},
	}
	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer testEnv.Stop()
	clients.SetInClusterConfig(cfg)
	client, err := dynamic.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}
	// corev1client, err := corev1c.NewForConfig(cfg)
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		panic(err)
	}
	// GR
	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		panic(err)
	}
	// GVR
	rm := restmapper.NewDiscoveryRESTMapper(groupResources)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// cleanup funcs
			var cleanupFuncs []func()
			// create objects defined in the test case
			for _, obj := range tt.objs {
				uobjMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
				if err != nil {
					panic(err)
				}
				uobj := unstructured.Unstructured{Object: uobjMap}

				mapping, err := rm.RESTMapping(schema.GroupKind{Group: uobj.GroupVersionKind().Group, Kind: uobj.GroupVersionKind().Kind}, uobj.GroupVersionKind().Version)
				if err != nil {
					panic(err)
				}
				resourceClient := client.Resource(mapping.Resource).Namespace(uobj.GetNamespace())
				_, err = resourceClient.Create(context.Background(), &uobj, metav1.CreateOptions{})
				if err != nil {
					if !apierrs.IsAlreadyExists(err) {
						panic(err)
					}
				}
				// don't cleanup namespace since its problematic with envtest https://github.com/kubernetes-sigs/controller-runtime/issues/880
				if uobj.GroupVersionKind().Kind != "Namespace" {
					cleanupFuncs = append(cleanupFuncs, func() {
						resourceClient.Delete(context.Background(), uobj.GetName(), metav1.DeleteOptions{})
					})
				}
			}
			defer func() {
				// cleanup after test
				for i := range cleanupFuncs {
					cleanupFuncs[i]()
				}
			}()
			got, err := GetRegistryEnvsForLocation(tt.args.location, tt.args.namespace)
			if tt.wantErrString != "" && (err == nil || tt.wantErrString != err.Error()) {
				t.Errorf("GetRegistryEnvsForLocation() error = %v, wantErrString %v", err, tt.wantErrString)
				return
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("GetRegistryEnvsForLocation() = %v, want %v", got, tt.want)
			}
		})
	}
}

// https://github.com/openshift/velero/blob/konveyor-dev/config/crd/v1/bases/velero.io_backupstoragelocations.yaml
// with ` escaped with +"`"+
const bslCRDyaml = `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  annotations:
    controller-gen.kubebuilder.io/version: v0.14.0
  name: backupstoragelocations.velero.io
spec:
  group: velero.io
  names:
    kind: BackupStorageLocation
    listKind: BackupStorageLocationList
    plural: backupstoragelocations
    shortNames:
    - bsl
    singular: backupstoragelocation
  scope: Namespaced
  versions:
  - additionalPrinterColumns:
    - description: Backup Storage Location status such as Available/Unavailable
      jsonPath: .status.phase
      name: Phase
      type: string
    - description: LastValidationTime is the last time the backup store location was
        validated
      jsonPath: .status.lastValidationTime
      name: Last Validated
      type: date
    - jsonPath: .metadata.creationTimestamp
      name: Age
      type: date
    - description: Default backup storage location
      jsonPath: .spec.default
      name: Default
      type: boolean
    name: v1
    schema:
      openAPIV3Schema:
        description: BackupStorageLocation is a location where Velero stores backup
          objects
        properties:
          apiVersion:
            description: |-
              APIVersion defines the versioned schema of this representation of an object.
              Servers should convert recognized schemas to the latest internal value, and
              may reject unrecognized values.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources
            type: string
          kind:
            description: |-
              Kind is a string value representing the REST resource this object represents.
              Servers may infer this from the endpoint the client submits requests to.
              Cannot be updated.
              In CamelCase.
              More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
            type: string
          metadata:
            type: object
          spec:
            description: BackupStorageLocationSpec defines the desired state of a
              Velero BackupStorageLocation
            properties:
              accessMode:
                description: AccessMode defines the permissions for the backup storage
                  location.
                enum:
                - ReadOnly
                - ReadWrite
                type: string
              backupSyncPeriod:
                description: BackupSyncPeriod defines how frequently to sync backup
                  API objects from object storage. A value of 0 disables sync.
                nullable: true
                type: string
              config:
                additionalProperties:
                  type: string
                description: Config is for provider-specific configuration fields.
                type: object
              credential:
                description: Credential contains the credential information intended
                  to be used with this location
                properties:
                  key:
                    description: The key of the secret to select from.  Must be a
                      valid secret key.
                    type: string
                  name:
                    description: |-
                      Name of the referent.
                      More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
                      TODO: Add other useful fields. apiVersion, kind, uid?
                    type: string
                  optional:
                    description: Specify whether the Secret or its key must be defined
                    type: boolean
                required:
                - key
                type: object
                x-kubernetes-map-type: atomic
              default:
                description: Default indicates this location is the default backup
                  storage location.
                type: boolean
              objectStorage:
                description: ObjectStorageLocation specifies the settings necessary
                  to connect to a provider's object storage.
                properties:
                  bucket:
                    description: Bucket is the bucket to use for object storage.
                    type: string
                  caCert:
                    description: CACert defines a CA bundle to use when verifying
                      TLS connections to the provider.
                    format: byte
                    type: string
                  prefix:
                    description: Prefix is the path inside a bucket to use for Velero
                      storage. Optional.
                    type: string
                required:
                - bucket
                type: object
              provider:
                description: Provider is the provider of the backup storage.
                type: string
              validationFrequency:
                description: ValidationFrequency defines how frequently to validate
                  the corresponding object storage. A value of 0 disables validation.
                nullable: true
                type: string
            required:
            - objectStorage
            - provider
            type: object
          status:
            description: BackupStorageLocationStatus defines the observed state of
              BackupStorageLocation
            properties:
              accessMode:
                description: |-
                  AccessMode is an unused field.


                  Deprecated: there is now an AccessMode field on the Spec and this field
                  will be removed entirely as of v2.0.
                enum:
                - ReadOnly
                - ReadWrite
                type: string
              lastSyncedRevision:
                description: |-
                  LastSyncedRevision is the value of the ` + "`metadata/revision`" + ` file in the backup
                  storage location the last time the BSL's contents were synced into the cluster.


                  Deprecated: this field is no longer updated or used for detecting changes to
                  the location's contents and will be removed entirely in v2.0.
                type: string
              lastSyncedTime:
                description: |-
                  LastSyncedTime is the last time the contents of the location were synced into
                  the cluster.
                format: date-time
                nullable: true
                type: string
              lastValidationTime:
                description: |-
                  LastValidationTime is the last time the backup store location was validated
                  the cluster.
                format: date-time
                nullable: true
                type: string
              message:
                description: Message is a message about the backup storage location's
                  status.
                type: string
              phase:
                description: Phase is the current state of the BackupStorageLocation.
                enum:
                - Available
                - Unavailable
                type: string
            type: object
        type: object
    served: true
    storage: true
    subresources: {}`
