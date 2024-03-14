package openshift

import (
	"context"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	configv1 "github.com/openshift/api/config/v1"
	v1 "github.com/openshift/api/imageregistry/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// If you disable the ImageRegistry capability or if you disable the integrated OpenShift image registry in the Cluster Image Registry Operator’s configuration,
// the service account token secret and image pull secret are not generated for each service account.
// https://docs.openshift.com/container-platform/4.15/installing/cluster-capabilities.html#additional-resources_cluster-capabilities:~:text=If%20you%20disable%20the%20ImageRegistry%20capability%20or%20if%20you%20disable%20the%20integrated%20OpenShift%20image%20registry%20in%20the%20Cluster%20Image%20Registry%20Operator%E2%80%99s%20configuration%2C%20the%20service%20account%20token%20secret%20and%20image%20pull%20secret%20are%20not%20generated%20for%20each%20service%20account.

var imageRegistryCapabilityEnabled bool
func ImageRegistryCapabilityEnabled() (bool, error) {
	// Cache the result of the image registry capability check, once enabled, it should not change
	// Cluster administrators cannot disable a cluster capability after it is enabled.
	// https://docs.openshift.com/container-platform/4.15/post_installation_configuration/enabling-cluster-capabilities.html#enabling-cluster-capabilities:~:text=Cluster%20administrators%20cannot%20disable%20a%20cluster%20capability%20after%20it%20is%20enabled.
	if imageRegistryCapabilityEnabled {
		return true, nil
	}
	clusterVersion, err := GetClusterVersion()
	if err != nil {
		return false, err
	}
	ec := clusterVersion.Status.Capabilities.EnabledCapabilities
	for _, c := range ec {
		if c == configv1.ClusterVersionCapabilityImageRegistry {
			imageRegistryCapabilityEnabled = true
			return true, nil
		}
	}
	return false, nil
}

func ImageRegistryHasReplicas() (bool, error) {
	c, err := GetImageRegistryConfig()
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return c.Status.ReadyReplicas > 0, nil
}

// https://github.com/openshift/cluster-image-registry-operator/blob/48875d3ccb4595be9d3bec563d1fda2eb940cecf/pkg/defaults/defaults.go#L19
// avoiding indirect imports and version conflicts
const ImageRegistryResourceName = "cluster"

func GetImageRegistryConfig() (*v1.Config, error) {
	client, err := clients.OCPImageRegistryConfigClient()
	if err != nil {
		return nil, err
	}
	return client.Configs().Get(context.Background(), ImageRegistryResourceName, metav1.GetOptions{})
}