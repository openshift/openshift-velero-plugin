package imagestream

import (
	"errors"
	"fmt"

	"github.com/containers/image/v5/types"
	"github.com/kaovilai/udistribution/pkg/image/udistribution"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
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
