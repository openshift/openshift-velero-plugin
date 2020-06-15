package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/openshift/library-go/pkg/image/reference"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func GetRegistryInfo(major, minor int, log logrus.FieldLogger) (string, error) {
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

// returns major, minor versions for kube
func GetServerVersion() (int, int, error) {
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

	return major, minor, nil
}
