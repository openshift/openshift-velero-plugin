package imagestream

import (
	"context"
	"errors"

	"github.com/containers/image/v5/types"
	"github.com/openshift/client-go/route/clientset/versioned/scheme"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	velero "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	internalRegistrySystemContextVar *types.SystemContext
	oadpRegistryRoute map[k8stypes.UID]*string
	bslNameForBackup map[k8stypes.UID]string
)
	

func internalRegistrySystemContext() (*types.SystemContext, error) {
	if internalRegistrySystemContextVar != nil {
		return internalRegistrySystemContextVar, nil
	}


	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	if config.BearerToken == "" {
		return nil, errors.New("BearerToken not found, can't authenticate with registry")
	}
	ctx := &types.SystemContext{
		DockerDaemonInsecureSkipTLSVerify: true,
		DockerInsecureSkipTLSVerify:       types.OptionalBoolTrue,
		DockerDisableDestSchema1MIMETypes: true,
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: "ignored",
			Password: config.BearerToken,
		},
	}
	internalRegistrySystemContextVar = ctx // cache the system context
	return ctx, nil
}

func migrationRegistrySystemContext() (*types.SystemContext, error) {
	ctx := &types.SystemContext{
		DockerDaemonInsecureSkipTLSVerify: true,
		DockerInsecureSkipTLSVerify:       types.OptionalBoolTrue,
		DockerDisableDestSchema1MIMETypes: true,
	}
	return ctx, nil
}

// Takes Namesapce where the operator resides, name of the BackupStorageLocation and name of configMap as input and returns the Route of backup registry.
func getOADPRegistryRoute(uid k8stypes.UID, namespace string, location string, configMap string) (string, error) {
	if route, found := oadpRegistryRoute[uid]; found && route != nil {
		return *route, nil
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return "cannot load in-cluster config", err
	}
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "could not create client", err
	}
	cMap := client.CoreV1().ConfigMaps(namespace)
	mapClient, err := cMap.Get(context.Background(), configMap, metav1.GetOptions{})
	if err != nil {
		return "failed to find registry configmap", err
	}
	osClient, err := routev1client.NewForConfig(config)
	if err != nil {
		return "failed to generate route client", err
	}
	routeClient := osClient.Routes(namespace)
	route, err := routeClient.Get(context.Background(), mapClient.Data[location], metav1.GetOptions{})
	if err != nil {
		return "failed to find OADP registry route", err
	}

	// save the registry hostname to a temporary file
	oadpRegistryRoute[uid] = &route.Spec.Host
	return route.Spec.Host, nil
}

// Takes Backup Name an Namespace where the operator resides and returns the name of the BackupStorageLocation
func getBackupStorageLocationForBackup(uid k8stypes.UID, name string, namespace string) (string, error) {
	if bslNameForBackup != nil {
		return bslNameForBackup[uid], nil
	} else {
		bslNameForBackup = make(map[k8stypes.UID]string)
	}

	config, err := rest.InClusterConfig()
	crdConfig := *config
	crdConfig.ContentConfig.GroupVersion = &schema.GroupVersion{Group: "velero.io", Version: "v1"}
	crdConfig.APIPath = "/apis"
	crdConfig.NegotiatedSerializer = serializer.NewCodecFactory(scheme.Scheme)
	crdConfig.UserAgent = rest.DefaultKubernetesUserAgent()
	result := velero.BackupList{}

	if err != nil {
		return "", err
	}
	client, err := rest.UnversionedRESTClientFor(&crdConfig)
	if err != nil {
		return "", err
	}

	err = client.
		Get().
		Namespace(namespace).
		Resource("backups").
		Do(context.Background()).
		Into(&result)
	if err != nil {
		return "", err
	}

	for _, element := range result.Items {
		if element.Name == name {
			bslNameForBackup[uid] = element.Spec.StorageLocation
			return element.Spec.StorageLocation, nil
		}
	}
	return "", errors.New("BackupStorageLocation not found")
}