package imagestream

import (
	"context"
	"errors"
	"fmt"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
	"strings"
	"time"
)

func copyImage(log logrus.FieldLogger,src, dest string, sourceCtx, destinationCtx *types.SystemContext) ([]byte, error) {
	policyContext, err := getPolicyContext()
	if err != nil {
		return []byte{}, fmt.Errorf("Error loading trust policy: %v", err)
	}
	defer policyContext.Destroy()
	srcRef, err := alltransports.ParseImageName(src)
	if err != nil {
		return []byte{}, fmt.Errorf("Invalid source name %s: %v", src, err)
	}
	destRef, err := alltransports.ParseImageName(dest)
	if err != nil {
		return []byte{}, fmt.Errorf("Invalid destination name %s: %v", dest, err)
	}
	// Let's retry the image copy up to 10 times
	// Each retry will wait 5 seconds longer
	// Let's log a warning if we encounter `blob unknown to registry`
	retryWait := 5
	log.Info(fmt.Sprintf("copying image: %s; will attempt up to 5 times...", src))
	for i := 0; i < 5; i++ {
		manifest, err := copy.Image(context.Background(), policyContext, destRef, srcRef, &copy.Options{
			SourceCtx:      sourceCtx,
			DestinationCtx: destinationCtx,
		})
		if err == nil {
			return manifest, err
		}
		if strings.Contains(err.Error(), "blob unknown to registry") {
			log.Warn(fmt.Sprintf("encountered `blob unknown to registry error` for image %s", src))
		}
		log.Info(fmt.Sprintf("attempt #%v failed, waiting %vs and then retrying", i, retryWait))
		time.Sleep(time.Duration(retryWait) * time.Second)
		retryWait += 5
	}
	return []byte{}, err
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	return signature.NewPolicyContext(policy)
}

func internalRegistrySystemContext() (*types.SystemContext, error) {
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

func findSpecTag(tags []imagev1API.TagReference, name string) *imagev1API.TagReference {
	for _, tag := range tags {
		if tag.Name == name {
			return &tag
		}
	}
	return nil
}
