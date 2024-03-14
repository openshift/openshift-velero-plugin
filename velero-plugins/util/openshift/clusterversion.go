package openshift

import (
	"context"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// default clusterversions name
	// https://github.com/openshift/cluster-version-operator/blob/6a76ba95ed441893e1bdf6616c47701c0464b7f4/pkg/start/start.go#L47
	clusterversionsName = "version"
)

// GetClusterVersion returns the ClusterVersion object
func GetClusterVersion() (*configv1.ClusterVersion, error) {
	client, err := clients.OCPConfigClient()
	if err != nil {
		return nil, err
	}
	return client.ClusterVersions().Get(context.Background(), clusterversionsName, metav1.GetOptions{})
}