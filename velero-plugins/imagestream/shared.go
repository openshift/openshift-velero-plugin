package imagestream

import (
	"context"
	"errors"

	"github.com/containers/image/v5/types"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	internalRegistrySystemContextVar *types.SystemContext
	oadpRegistryRoute map[k8stypes.UID]*string
)
	

func internalRegistrySystemContext() (*types.SystemContext, error) {
	if internalRegistrySystemContextVar != nil {
		return internalRegistrySystemContextVar, nil
	}


	config, err := clients.GetInClusterConfig()
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

	config, err := clients.GetInClusterConfig()
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
func getBackupStorageLocationNameForBackup(uid k8stypes.UID, name string, namespace string) (string, error) {
	b, err := common.GetBackup(uid, name, namespace)
	if err != nil {
		return "", err
	} else if b == nil {
		return "", errors.New("backup is nil")
	}
	return b.Spec.StorageLocation, nil
}