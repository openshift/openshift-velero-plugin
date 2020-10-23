package imagecopy

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/manifest"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/go-logr/logr"
	imagev1API "github.com/openshift/api/image/v1"
	//"github.com/sirupsen/logrus"
)

// CopyLocalImageStreamImages copies all local images associated with the ImageStream
// is: ImageStream resource that images are being copied for
// internalRegistryPath: The internal registry path for the cluster in which is comes from, used to determine which images are local
// srcRegistry: the registry to copy the images from
// destRegistry: the registry to copy the images to
// destNamespace: the namespace to copy to
// log: the logger to log to
// updateDigest: whether to update the input imageStream if the digest changes on pushing to the new registry
func CopyLocalImageStreamImages(
	imageStream imagev1API.ImageStream,
	internalRegistryPath string,
	srcRegistry string,
	destRegistry string,
	destNamespace string,
	copyOptions *copy.Options,
	log logr.Logger,
	updateDigest bool) error {
	localImageCopied := false
	localImageCopiedByTag := false
	for tagIndex, tag := range imageStream.Status.Tags {
		log.Info(fmt.Sprintf("[imagecopy] Copying tag: %#v", tag.Tag))
		specTag := findSpecTag(imageStream.Spec.Tags, tag.Tag)
		copyToTag := true
		if specTag != nil && specTag.From != nil {
			// we have a tag.
			log.Info(fmt.Sprintf("[imagecopy] image tagged: %s, %s", specTag.From.Kind, specTag.From.Name))
			// Use the tag if it references an ImageStreamImage in the current namespace
			if !(specTag.From.Kind == "ImageStreamImage" && (specTag.From.Namespace == "" || specTag.From.Namespace == imageStream.Namespace)) {
				log.Info(fmt.Sprintf("[imagecopy] not using tag for copy (either out-of-namespace or not an ImageStreamImage tag"))
				copyToTag = false
			}
		}
		// Iterate over items in reverse order so most recently tagged is copied last
		for i := len(tag.Items) - 1; i >= 0; i-- {
			dockerImageReference := tag.Items[i].DockerImageReference
			if len(internalRegistryPath) > 0 && strings.HasPrefix(dockerImageReference, internalRegistryPath) {
				if len(srcRegistry) == 0 {
					return errors.New("copy source registry not found but ImageStream has internal images")
				}
				if len(destRegistry) == 0 {
					return errors.New("copy destination registry not found but ImageStream has internal images")
				}
				localImageCopied = true
				destTag := ""
				if copyToTag {
					localImageCopiedByTag = true
					destTag = ":" + tag.Tag
				}
				srcPath := fmt.Sprintf("docker://%s%s", srcRegistry, strings.TrimPrefix(dockerImageReference, internalRegistryPath))
				destPath := fmt.Sprintf("docker://%s/%s/%s%s", destRegistry, destNamespace, imageStream.Name, destTag)
				log.Info(fmt.Sprintf("[imagecopy] copying from: %s", srcPath))
				log.Info(fmt.Sprintf("[imagecopy] copying to: %s", destPath))

				imgManifest, err := copyImage(log, srcPath, destPath, copyOptions)
				if err != nil {
					log.Info(fmt.Sprintf("[imagecopy] Error copying image: %v", err))
					return err
				}
				newDigest, err := manifest.Digest(imgManifest)
				if err != nil {
					log.Info(fmt.Sprintf("[imagecopy] Error computing image digest for manifest: %v", err))
					return err
				}
				log.Info(fmt.Sprintf("[imagecopy] src image digest: %s", tag.Items[i].Image))
				if updateDigest && string(newDigest) != tag.Items[i].Image {
					log.Info(fmt.Sprintf("[imagecopy] migration registry image digest: %s", newDigest))
					imageStream.Status.Tags[tagIndex].Items[i].Image = string(newDigest)
					digestSplit := strings.Split(dockerImageReference, "@")
					// update sha in dockerImageRef found
					if len(digestSplit) == 2 {
						imageStream.Status.Tags[tagIndex].Items[i].DockerImageReference = digestSplit[0] +
							"@" + string(newDigest)
					}
				}
				log.Info(fmt.Sprintf("[imagecopy] manifest of copied image: %s", imgManifest))
			}
		}
	}
	log.Info(fmt.Sprintf("[imagecopy] copied at least one local image: %t", localImageCopied))
	log.Info(fmt.Sprintf("[imagecopy] copied at least one local image by tag: %t", localImageCopiedByTag))
	return nil
}

func copyImage(log logr.Logger,src, dest string, copyOptions *copy.Options) ([]byte, error) {
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
	retryWait := 0
	log.Info(fmt.Sprintf("copying image: %s; will attempt up to 7 times...", src))
	for i := 0; i < 7; i++ {
		time.Sleep(time.Duration(retryWait) * time.Second)
		retryWait += 5
		var manifest []byte
		manifest, err = copy.Image(context.Background(), policyContext, destRef, srcRef, copyOptions)
		if err == nil {
			return manifest, err
		}
		if strings.Contains(err.Error(), "blob unknown to registry") {
			log.Info(fmt.Sprintf("encountered `blob unknown to registry error` for image %s", src))
		}
		log.Info(fmt.Sprintf("attempt #%v failed, waiting %vs and then retrying", i+1, retryWait))
	}
	return []byte{}, err
}

func getPolicyContext() (*signature.PolicyContext, error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	return signature.NewPolicyContext(policy)
}

func findSpecTag(tags []imagev1API.TagReference, name string) *imagev1API.TagReference {
	for _, tag := range tags {
		if tag.Name == name {
			return &tag
		}
	}
	return nil
}
