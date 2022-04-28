package imagestream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/containers/image/v5/types"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
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


func internalRegistrySystemContext(uid k8stypes.UID) (*types.SystemContext, error) {
	internalRegistrySystemContextFilePath := common.TmpOADPPath + "/" + string(uid) + "/" + "internal-registry-system-context"
	internalRegistrySystemContextFileName := "internal-registry-system-context.txt"
	systemContextBytes, err := os.ReadFile(internalRegistrySystemContextFilePath + "/" + internalRegistrySystemContextFileName)
	if err == nil {
		systemContext := types.SystemContext{}
		err := json.Unmarshal(systemContextBytes, &systemContext)
		if err == nil {
			return &systemContext, nil
		}
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
	systemContextBytes, err = json.Marshal(ctx)
	if err != nil {
		return nil, err
	}
	err = common.WriteByteToDirPath(internalRegistrySystemContextFilePath, internalRegistrySystemContextFileName, systemContextBytes)
	if err != nil {
		return nil, err
	}
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

	registryTmpFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/%s/", common.TmpOADPPath, uid, "registry-route",namespace, location, configMap)
	registryTmpFileName := "registry.txt"
	// retrieve registry hostname from temporary file
	tmpSpecHost, err := os.ReadFile(registryTmpFilePath + registryTmpFileName)
	if err == nil && len(tmpSpecHost) > 0 {
		return string(tmpSpecHost), nil
	}
	err = nil // reset error


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
	err = common.WriteByteToDirPath(registryTmpFilePath, registryTmpFileName, []byte(route.Spec.Host))
	if err != nil {
		return "", err
	}
	return route.Spec.Host, nil
}

// Takes Backup Name an Namespace where the operator resides and returns the name of the BackupStorageLocation
func getBackupStorageLocationForBackup(uid k8stypes.UID, name string, namespace string) (string, error) {
	bslTmpFilePath := fmt.Sprintf("%s/%s/%s/%s/%s/", common.TmpOADPPath, uid, "backup-storage-location", namespace, name)
	bslTmpFileName := "bsl.txt"
	// retrieve bsl for backup from temporary file
	bslForBackupFromTmp, err := os.ReadFile(bslTmpFilePath + bslTmpFileName)
	if err == nil && len(bslForBackupFromTmp) > 0 {
		return string(bslForBackupFromTmp), nil
	}
	err = nil // reset error

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
			// save the bsl for backup to a temporary file
			err = common.WriteByteToDirPath(bslTmpFilePath, bslTmpFileName, []byte(element.Spec.StorageLocation))
			if err != nil {
				return "", err
			}
			return element.Spec.StorageLocation, nil
		}
	}
	return "", errors.New("BackupStorageLocation not found")
}