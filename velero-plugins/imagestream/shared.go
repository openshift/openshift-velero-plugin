package imagestream

import (
	"context"
	"errors"
	"fmt"

	"github.com/containers/image/v5/types"
	"github.com/kaovilai/udistribution/pkg/image/udistribution"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	routev1client "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

var (
	internalRegistrySystemContextVar  *types.SystemContext
	oadpRegistryRoute                 map[k8stypes.UID]*string
	udistributionTransportForLocation map[k8stypes.UID]*udistribution.UdistributionTransport // unique transport per backup uid
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

func GetUdistributionTransportForLocation(uid k8stypes.UID, location, namespace string, log logrus.FieldLogger) (*udistribution.UdistributionTransport, error) {
	if ut, found := udistributionTransportForLocation[uid]; found && ut != nil {
		log.Info("Got udistribution transport from cache")
		return ut, nil
	}
	log.Info("Getting registry envs for udistribution transport")
	envs, err := GetRegistryEnvsForLocation(location, namespace)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("errors getting registryenv: %v", err))
	}
	log.Info("Creating udistribution transport")
	ut, err := udistribution.NewTransportFromNewConfig("", envs)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("errors creating new udistribution transport from config: %v", err))
	}
	log.Info("Got udistribution transport")
	if udistributionTransportForLocation == nil {
		udistributionTransportForLocation = make(map[k8stypes.UID]*udistribution.UdistributionTransport)
	}
	udistributionTransportForLocation[uid] = ut // cache the transport
	return ut, nil
}

func GetUdistributionKey(location, namespace string) string {
	return fmt.Sprintf("%s-%s", namespace, location)
}

// Get Registry environment variables to create registry client
// This should be called once per backup.
func GetRegistryEnvsForLocation(location string, namespace string) ([]string, error) {
	// secret, key, err := common.GetSecretKeyForBackupStorageLocation(location, namespace)
	// if err != nil {
	// 	return nil, errors.New(fmt.Sprintf("errors getting secret key for bsl: %v", err))
	// }
	bsl, err := common.GetBackupStorageLocation(location, namespace)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("errors getting bsl: %v", err))
	}

	envVars, err := getRegistryEnvVars(bsl)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("errors getting registry env vars: %v", err))
	}
	return coreV1EnvVarArrToStringArr(envVars), nil
}

func coreV1EnvVarArrToStringArr(envVars []corev1.EnvVar) []string {
	var envVarsStr []string
	for _, envVar := range envVars {
		envVarsStr = append(envVarsStr, coreV1EnvVarToString(envVar))
	}
	return envVarsStr
}
func coreV1EnvVarToString(envVar corev1.EnvVar) string {
	return fmt.Sprintf("%s=%s", envVar.Name, envVar.Value)
}
