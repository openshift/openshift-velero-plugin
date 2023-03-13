package common

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/kaovilai/udistribution/pkg/image/udistribution"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/openshift/client-go/route/clientset/versioned/scheme"
	"github.com/openshift/library-go/pkg/image/reference"
	"github.com/openshift/oadp-operator/api/v1alpha1"
	"github.com/openshift/oadp-operator/pkg/credentials"
	"github.com/sirupsen/logrus"
	velero "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

var (
	registryInfo  *string
	serverVersion *serverVersionStruct
	BackupUidMap  map[types.UID]*CommonStruct
	lastGarbageCollect time.Time
)
// common cache for backup UIDs
type CommonStruct struct {
	Backup *velero.Backup
	Ut *udistribution.UdistributionTransport
	lastAccessed time.Time
}

// JustAccessed marks the item as recently accessed and triggers garbage collection
func (c *CommonStruct) JustAccessed() {
	c.lastAccessed = time.Now()
	GarbageCollectCommonStruct()
}

// GarbageCollectCommonStructs removes old entries from the cache
func GarbageCollectCommonStruct(){
	if lastGarbageCollect.Add(time.Minute).After(time.Now()) {
		// do not run garbage collection more than once per minute
		return
	}
	for k, _ := range BackupUidMap {
		if BackupUidMap[k].lastAccessed.Add(time.Minute).Before(time.Now()) {
			// remove old entries
			delete(BackupUidMap, k)
		}
	}
	lastGarbageCollect = time.Now()
}
type serverVersionStruct struct {
	Major int
	Minor int
}

func GetRegistryInfo(log logrus.FieldLogger) (string, error) {
	if registryInfo != nil {
		return *registryInfo, nil //use cache
	}

	imageClient, err := clients.ImageClient()
	if err != nil {
		return "", err
	}
	imageStreams, err := imageClient.ImageStreams("openshift").List(context.Background(), metav1.ListOptions{})
	if err == nil && len(imageStreams.Items) > 0 {
		if value := imageStreams.Items[0].Status.DockerImageRepository; len(value) > 0 {
			ref, err := reference.Parse(value)
			if err == nil {
				log.Info("[GetRegistryInfo] value from imagestream")
				registryInfo = StringPtr(ref.Registry) //save cache
				return ref.Registry, nil
			}
		}
	}

	major, minor, err := GetServerVersion()
	if err != nil {
		return "", err
	}

	if major != 1 {
		return "", fmt.Errorf("server version %v.%v not supported. Must be 1.x", major, minor)
	}

	cClient, err := clients.CoreClient()
	if err != nil {
		return "", err
	}
	if minor < 7 {
		return "", fmt.Errorf("Kubernetes version 1.%v not supported. Must be 1.7 or greater", minor)
	} else if minor <= 11 {
		registrySvc, err := cClient.Services("default").Get(context.Background(), "docker-registry", metav1.GetOptions{})
		if err != nil {
			// Return empty registry host but no error; registry not found
			registryInfo = StringPtr("")
			return "", nil
		}
		internalRegistry := registrySvc.Spec.ClusterIP + ":" + strconv.Itoa(int(registrySvc.Spec.Ports[0].Port))
		registryInfo = &internalRegistry //save cache
		log.Info("[GetRegistryInfo] value from clusterIP")
		return internalRegistry, nil
	} else {
		config, err := cClient.ConfigMaps("openshift-apiserver").Get(context.Background(), "config", metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		serverConfig := APIServerConfig{}
		err = json.Unmarshal([]byte(config.Data["config.yaml"]), &serverConfig)
		if err != nil {
			return "", err
		}
		internalRegistry := serverConfig.ImagePolicyConfig.InternalRegistryHostname
		registryInfo = &internalRegistry //save cache
		if len(internalRegistry) == 0 {
			return "", nil
		}
		log.Info("[GetRegistryInfo] value from clusterIP")
		return internalRegistry, nil
	}
}

func getMetadataAndAnnotations(item runtime.Unstructured) (metav1.Object, map[string]string, error) {
	metadata, err := meta.Accessor(item)
	if err != nil {
		return nil, nil, err
	}

	annotations := metadata.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	return metadata, annotations, nil
}

// returns major, minor versions for kube
func GetServerVersion() (int, int, error) {
	// save server version to tmp file
	if serverVersion != nil {
		return serverVersion.Major, serverVersion.Minor, nil
	}

	client, err := clients.DiscoveryClient()
	if err != nil {
		return 0, 0, err
	}
	version, err := client.ServerVersion()
	if err != nil {
		return 0, 0, err
	}

	// Attempt parsing version.Major/Minor first, fall back to parsing gitVersion
	major, err1 := strconv.Atoi(version.Major)
	minor, err2 := strconv.Atoi(strings.Trim(version.Minor, "+"))

	if err1 != nil || err2 != nil {
		// gitVersion format ("v1.11.0+d4cacc0")
		r, _ := regexp.Compile(`v[0-9]+\.[0-9]+\.`)
		valid := r.MatchString(version.GitVersion)
		if !valid {
			return 0, 0, errors.New("gitVersion does not match expected format")
		}
		majorMinorArr := strings.Split(strings.Split(version.GitVersion, "v")[1], ".")

		major, err = strconv.Atoi(majorMinorArr[0])
		if err != nil {
			return 0, 0, err
		}
		minor, err = strconv.Atoi(majorMinorArr[1])
		if err != nil {
			return 0, 0, err
		}
	}

	serverVersion = &serverVersionStruct{Major: major, Minor: minor} //save cache

	return major, minor, nil
}

func GetVeleroV1Client() (*rest.RESTClient, error) {
	config, err := clients.GetInClusterConfig()
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "velero.io", Version: "v1"}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()

	if err != nil {
		return nil, err
	}
	client, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// fetches backup for a given backup name and requester's uid
func GetBackup(uid types.UID, name string, namespace string) (*velero.Backup, error) {
	if BackupUidMap == nil {
		BackupUidMap = make(map[types.UID]*CommonStruct)
	}
	if BackupUidMap[uid] == nil {
		BackupUidMap[uid] = &CommonStruct{}
	}
	BackupUidMap[uid].JustAccessed()
	if BackupUidMap[uid].Backup != nil {
		return BackupUidMap[uid].Backup, nil
	}

	if name == "" {
		return nil, errors.New("cannot get backup for an empty name")
	}

	if namespace == "" {
		return nil, errors.New("cannot get backup for an empty namespace")
	}

	client, err := GetVeleroV1Client()
	if err != nil {
		return nil, err
	}
	result := velero.Backup{}
	err = client.
		Get().
		Namespace(namespace).
		Resource("backups").
		Name(name).
		Do(context.Background()).
		Into(&result)
	if err != nil {
		return nil, err
	}

	BackupUidMap[uid].Backup = &result
	return &result, nil
}

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func StringPtr(s string) *string {
	return &s
}

func GetBackupStorageLocation(name, namespace string) (*velero.BackupStorageLocation, error) {
	client, err := GetVeleroV1Client()
	if err != nil {
		return nil, err
	}
	result := velero.BackupStorageLocation{}
	err = client.
		Get().
		Namespace(namespace).
		Resource("backupstoragelocations").
		Name(name).
		Do(context.Background()).
		Into(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get secret for backup storage location along with key to use to get secret data.
func GetSecretKeyForBackupStorageLocation(name, namespace string) (*corev1.Secret, string, error) {
	if name == "" {
		return nil, "", errors.New("cannot get secret for an empty name")
	}
	if namespace == "" {
		return nil, "", errors.New("cannot get secret for an empty namespace")
	}
	bsl, err := GetBackupStorageLocation(name, namespace)
	if err != nil {
		return nil, "", err
	}
	var sName, sKey string
	if bsl.Spec.Credential != nil {
		sName = bsl.Spec.Credential.Name
		sKey = bsl.Spec.Credential.Key
	} else {
		// discover secret name from OADP default for storage location's plugin
		provider := strings.TrimPrefix(bsl.Spec.Provider, "velero.io/")
		if psf, found := credentials.PluginSpecificFields[v1alpha1.DefaultPlugin(provider)]; found && psf.IsCloudProvider {
			sName = psf.SecretName
			sKey = "cloud" // default key
		}
	}
	if sName == "" {
		return nil, "", errors.New("cannot get secret for a storage location without a credential")
	}
	icc, err := clients.GetInClusterConfig()
	if err != nil {
		return nil, "", err
	}
	cv1c, err := corev1client.NewForConfig(icc)
	if err != nil {
		return nil, "", err
	}
	secret, err := cv1c.Secrets(namespace).Get(context.Background(), sName, metav1.GetOptions{})
	if err != nil {
		return nil, "", err
	}
	return secret, sKey, nil
}
