package imagestream

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/containers/image/v5/types"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	k8stypes "k8s.io/apimachinery/pkg/types"
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

