package pod

import (
	"strings"

	"github.com/openshift/library-go/pkg/build/naming"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

// This is a copy of the function `needsDockercfgSecret` and its dependencies from the OpenShift to decide if we need to wait for the docker secret to be created

// We follow `func needsDockercfgSecret(serviceAccount *v1.ServiceAccount) bool {` logic
// to decide if we need to wait for the secret to be created where the service account checked is "default"
// https://github.com/openshift/openshift-controller-manager/blob/master/pkg/serviceaccounts/controllers/create_dockercfg_secrets.go#L304

const (
	// These constants are here to create a name that is short enough to survive chopping by generate name
	maxNameLength             = 63
	randomLength              = 5
	maxSecretPrefixNameLength = maxNameLength - randomLength
)

func needsDockercfgSecret(serviceAccount *v1.ServiceAccount) bool {
	mountableDockercfgSecrets, imageDockercfgPullSecrets := getGeneratedDockercfgSecretNames(serviceAccount)

	// look for an ImagePullSecret in the form
	if len(imageDockercfgPullSecrets) > 0 && len(mountableDockercfgSecrets) > 0 {
		return false
	}

	return true
}

func getGeneratedDockercfgSecretNames(serviceAccount *v1.ServiceAccount) (sets.String, sets.String) {
	mountableDockercfgSecrets := sets.String{}
	imageDockercfgPullSecrets := sets.String{}

	secretNamePrefix := getDockercfgSecretNamePrefix(serviceAccount.Name)

	for _, s := range serviceAccount.Secrets {
		if strings.HasPrefix(s.Name, secretNamePrefix) {
			mountableDockercfgSecrets.Insert(s.Name)
		}
	}
	for _, s := range serviceAccount.ImagePullSecrets {
		if strings.HasPrefix(s.Name, secretNamePrefix) {
			imageDockercfgPullSecrets.Insert(s.Name)
		}
	}
	return mountableDockercfgSecrets, imageDockercfgPullSecrets
}

func getDockercfgSecretNamePrefix(serviceAccountName string) string {
	return naming.GetName(serviceAccountName, "dockercfg-", maxSecretPrefixNameLength)
}
