package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"errors"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/openshift/library-go/pkg/image/reference"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	velero "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/openshift/client-go/route/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
)

func GetRegistryInfo(major, minor string, log logrus.FieldLogger) (string, error) {
	imageClient, err := clients.ImageClient()
	if err != nil {
		return "", err
	}
	imageStreams, err := imageClient.ImageStreams("openshift").List(metav1.ListOptions{})
	if err == nil && len(imageStreams.Items) > 0 {
		if value := imageStreams.Items[0].Status.DockerImageRepository; len(value) > 0 {
			ref, err := reference.Parse(value)
			if err == nil {
				log.Info("[GetRegistryInfo] value from imagestream")
				return ref.Registry, nil
			}
		}
	}

	if major != "1" {
		return "", fmt.Errorf("server version %v.%v not supported. Must be 1.x", major, minor)
	}
	intVersion, err := strconv.Atoi(minor)
	if err != nil {
		return "", fmt.Errorf("server minor version %v invalid value: %v", minor, err)
	}

	cClient, err := clients.CoreClient()
	if err != nil {
		return "", err
	}
	if intVersion < 7 {
		return "", fmt.Errorf("Kubernetes version 1.%v not supported. Must be 1.7 or greater", minor)
	} else if intVersion <= 11 {
		registrySvc, err := cClient.Services("default").Get("docker-registry", metav1.GetOptions{})
		if err != nil {
			// Return empty registry host but no error; registry not found
			return "", nil
		}
		internalRegistry := registrySvc.Spec.ClusterIP + ":" + strconv.Itoa(int(registrySvc.Spec.Ports[0].Port))
		log.Info("[GetRegistryInfo] value from clusterIP")
		return internalRegistry, nil
	} else {
		config, err := cClient.ConfigMaps("openshift-apiserver").Get("config", metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		serverConfig := APIServerConfig{}
		err = json.Unmarshal([]byte(config.Data["config.yaml"]), &serverConfig)
		if err != nil {
			return "", err
		}
		internalRegistry := serverConfig.ImagePolicyConfig.InternalRegistryHostname
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

func GetServerVersion() (*version.Info, error) {
	client, err := clients.DiscoveryClient()
	if err != nil {
		return nil, err
	}
	version, err := client.ServerVersion()
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(version.Minor, "+") {
		version.Minor = strings.TrimSuffix(version.Minor, "+")
	}

	return version, nil
}

func getRoute(namespace string, location string, configMap string) (string, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return "cannot load in-cluster config", err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "could not create client", err
	}
	cMap := client.CoreV1().ConfigMaps(namespace)
	mapClient, err := cMap.Get(configMap, metav1.GetOptions{})
	if err != nil {
		return "failed to find registry configmap", err
	}
	osClient, err := routev1client.NewForConfig(config)
	if err != nil {
		return "failed to generate route client", err
	}
	routeClient := osClient.Routes(namespace)
	route, err := routeClient.Get(mapClient.Data[location], metav1.GetOptions{})
	if err != nil {
		return "failed to find OADP registry route", err
	}
	return route.Spec.Host, nil
}

func getBackupStorageLocationForBackup(name string, namespace string) (string, error) {
	config, err := rest.InClusterConfig()
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "velero.io", Version: "v1"}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	result := velero.BackupList{}

	if err != nil {
		panic(err)
	}
	client, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		return "", err
	}

	err = client.
		Get().
		Namespace(namespace).
		Resource("backups").
		Do().
		Into(&result)
	if err != nil {
		return "", err
	}

	for _, element := range result.Items{
		if element.Name == name{
			return element.Spec.StorageLocation, nil
		}
	}
	return "", errors.New("BackupStorageLocation not found")
}
