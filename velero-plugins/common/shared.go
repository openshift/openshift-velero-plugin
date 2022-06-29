package common

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/openshift/client-go/route/clientset/versioned/scheme"
	"github.com/openshift/library-go/pkg/image/reference"
	"github.com/sirupsen/logrus"
	velero "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)
var (
	registryInfo *string
	serverVersion *serverVersionStruct
	backupMap map[types.UID]velero.Backup
)

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

// fetches backup for a given backup name
func GetBackup(uid types.UID, name string, namespace string) (*velero.Backup, error) {
	// this function is only used by pod/restore.go to check for backup.Spec.DefaultVolumesToRestic. We can cache this
	// value in backupMap for performance.
	if backupMap == nil {
		backupMap = make(map[types.UID]velero.Backup)
	} else {
		if backup, ok := backupMap[uid]; ok && backup.Name == name && backup.Namespace == namespace {
			return &backup, nil
		}
	}

	if name == "" {
		return nil, errors.New("cannot get backup for an empty name")
	}

	if namespace == "" {
		return nil, errors.New("cannot get backup for an empty namespace")
	}

	config, err := rest.InClusterConfig()
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "velero.io", Version: "v1"}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	result := velero.BackupList{}

	if err != nil {
		return nil, err
	}
	client, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		return nil, err
	}

	err = client.
		Get().
		Namespace(namespace).
		Resource("backups").
		Do(context.Background()).
		Into(&result)
	if err != nil {
		return nil, err
	}

	for _, backup := range result.Items {
		if backup.Name == name {
			backupMap[uid] = backup // save cache

			return &backup, nil
		}
	}
	return nil, errors.New("cannot find backup for the given name")
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