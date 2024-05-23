package common

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/sirupsen/logrus"
	corev1API "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ReplaceImageRefPrefix replaces an image reference prefix with newPrefix.
// If the input image reference does not start with oldPrefix, an error is returned
func ReplaceImageRefPrefix(s, oldPrefix, newPrefix string, namespaceMapping map[string]string) (string, error) {
	refSplit := strings.SplitN(s, "/", 2)
	if len(refSplit) != 2 {
		err := fmt.Errorf("image reference [%v] does not have prefix [%v]", s, oldPrefix)
		return "", err
	}
	if refSplit[0] != oldPrefix {
		err := fmt.Errorf("image reference [%v] does not have prefix [%v]", s, oldPrefix)
		return "", err
	}
	outPath := refSplit[1]
	namespaceSplit := strings.SplitN(refSplit[1], "/", 2)
	if len(namespaceSplit) == 2 && namespaceMapping[namespaceSplit[0]] != "" { // change namespace if mapping is enabled
		outPath = strings.Join([]string{namespaceMapping[namespaceSplit[0]], namespaceSplit[1]}, "/")
	}
	if len(namespaceSplit) == 2 && namespaceSplit[0] == "openshift" {
		shaSplit := strings.SplitN(refSplit[1], "@", 2)
		if len(shaSplit) == 2 {
			outPath = shaSplit[0]
		}
	}
	return fmt.Sprintf("%s/%s", newPrefix, outPath), nil
}

// HasImageRefPrefix returns true if the input image reference begins with
// the input prefix followed by "/"
func HasImageRefPrefix(s, prefix string) bool {
	refSplit := strings.SplitN(s, "/", 2)
	if len(refSplit) != 2 {
		return false
	}
	return refSplit[0] == prefix
}

// LocalImageReference describes an image in the internal openshift registry
type LocalImageReference struct {
	Registry  string
	Namespace string
	Name      string
	Tag       string
	Digest    string
}

// ParseLocalImageReference
func ParseLocalImageReference(s, prefix string) (*LocalImageReference, error) {
	refSplit := strings.Split(s, "/")
	if refSplit[0] != prefix {
		return nil, fmt.Errorf("image reference is not local")
	}
	if len(refSplit) != 3 {
		return nil, fmt.Errorf("Unexpected image reference %s", s)
	}
	parsed := LocalImageReference{Registry: prefix, Namespace: refSplit[1]}
	digestSplit := strings.Split(refSplit[2], "@")
	if len(digestSplit) > 2 {
		return nil, fmt.Errorf("Unexpected image reference %s", s)
	} else if len(digestSplit) == 2 {
		parsed.Name = digestSplit[0]
		parsed.Digest = digestSplit[1]
		return &parsed, nil
	}
	tagSplit := strings.Split(refSplit[2], ":")
	if len(tagSplit) > 2 {
		return nil, fmt.Errorf("Unexpected image reference %s", s)
	} else if len(tagSplit) == 2 {
		parsed.Tag = tagSplit[1]
	}
	parsed.Name = tagSplit[0]
	return &parsed, nil
}

// SwapContainerImageRefs updates internal image references from
// backup registry to restore registry pathnames
func SwapContainerImageRefs(containers []corev1API.Container, oldRegistry, newRegistry string, log logrus.FieldLogger, namespaceMapping map[string]string) {
	if oldRegistry == "" || newRegistry == "" {
		return
	}
	for n, container := range containers {
		imageRef := container.Image
		log.Infof("[util] container image ref %s", imageRef)
		newImageRef, err := ReplaceImageRefPrefix(imageRef, oldRegistry, newRegistry, namespaceMapping)
		if err == nil {
			// Replace local image
			log.Infof("[util] replacing container image ref %s with %s", imageRef, newImageRef)
			containers[n].Image = newImageRef
		}
	}

}

// UpdatePullSecret updates registry pull (or push) secret
// with a secret found in the dest cluster
func UpdatePullSecret(
	secretRef *corev1API.LocalObjectReference,
	secretList *corev1API.SecretList,
	log logrus.FieldLogger,
) (*corev1API.LocalObjectReference, error) {
	// If secret is empty or doesn't begin with "{builder|default|deployer}-dockercfg-"
	// then leave it as-is. Either there's no secret or there's a custom one that
	// should be migrated
	if secretRef == nil {
		return secretRef, nil
	}

	for _, prefix := range []string{"builder-dockercfg-", "default-dockercfg-", "deployer-dockercfg-"} {
		if strings.HasPrefix(secretRef.Name, prefix) {
			for _, secret := range secretList.Items {
				if strings.HasPrefix(secret.Name, prefix) {
					// get serviceAccount
					c1cc, err := clients.CoreClient()
					if err != nil {
						log.Infof("[util] CoreClient() failed with err %s", err.Error())
						return nil, err
					}
					// use 4.16+ annotations
					if saName := secret.Annotations[RegistrySANameAnnotation]; saName != "" {
						sa, err := c1cc.ServiceAccounts(secret.Namespace).Get(context.TODO(), saName, metav1.GetOptions{})
						if err != nil {
							log.Infof("[util] client.ServiceAccounts(ns) for ns %s failed with err %s", secret.Namespace, err.Error())
							return nil, err
						}
						// check if secret is associated with serviceAccount in dest cluster by comparing SA pull secret annotation with secret name
						if sa.Annotations[RegistryPullSecretAnnotation] == secret.Name {
							log.Info(fmt.Sprintf("[util] Found new dockercfg secret: %v", secret))
							newSecret := corev1API.LocalObjectReference{Name: secret.Name}
							return &newSecret, nil
						}
					} else if saName := secret.Annotations[LegacySANameAnnotation]; saName != "" {
						sa, err := c1cc.ServiceAccounts(secret.Namespace).Get(context.TODO(), saName, metav1.GetOptions{})
						if err != nil {
							log.Infof("[util] client.ServiceAccounts(ns) for ns %s failed with err %s", secret.Namespace, err.Error())
							return nil, err
						}
						// check if secret is associated with serviceAccount in dest cluster by comparing uids
						if secret.Annotations[LegacySAUIDAnnotation] == string(sa.UID) {
							log.Info(fmt.Sprintf("[util] Found new dockercfg secret: %v", secret))
							newSecret := corev1API.LocalObjectReference{Name: secret.Name}
							return &newSecret, nil
						}
					}
				}
			}
			return nil, errors.New("secret not found")
		}
	}
	return secretRef, nil
}

// GetSrcAndDestRegistryInfo returns the Registry hostname for both src and dest clusters
func GetSrcAndDestRegistryInfo(item runtime.Unstructured) (string, string, error) {
	_, annotations, err := getMetadataAndAnnotations(item)
	if err != nil {
		return "", "", err
	}
	backupRegistry := annotations[BackupRegistryHostname]
	restoreRegistry := annotations[RestoreRegistryHostname]
	return backupRegistry, restoreRegistry, nil
}

// GetOwnerReferences returns the array of OwnerReferences associated with the resource
func GetOwnerReferences(item runtime.Unstructured) ([]metav1.OwnerReference, error) {
	metadata, err := meta.Accessor(item)
	if err != nil {
		return nil, err
	}
	return metadata.GetOwnerReferences(), nil
}
