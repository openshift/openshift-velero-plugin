package clients

import (
	ocpappsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	buildv1 "github.com/openshift/client-go/build/clientset/versioned/typed/build/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"k8s.io/client-go/discovery"
	appsv1 "k8s.io/client-go/kubernetes/typed/apps/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
)

var coreClient *corev1.CoreV1Client
var coreClientError error

var appsClient *appsv1.AppsV1Client
var appsClientError error

var ocpAppsClient *ocpappsv1.AppsV1Client
var ocpAppsClientError error

var imageClient *imagev1.ImageV1Client
var imageClientError error

var discoveryClient *discovery.DiscoveryClient
var discoveryClientError error

var routeClient *routev1.RouteV1Client
var routeClientError error

var buildClient *buildv1.BuildV1Client
var buildClientError error

// CoreClient returns a kubernetes CoreV1Client
func CoreClient() (*corev1.CoreV1Client, error) {
	if coreClient == nil && coreClientError == nil {
		coreClient, coreClientError = newCoreClient()
	}
	return coreClient, coreClientError
}

func newCoreClient() (*corev1.CoreV1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := corev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// ImageClient returns an openshift ImageV1Client
func ImageClient() (*imagev1.ImageV1Client, error) {
	if imageClient == nil && imageClientError == nil {
		imageClient, imageClientError = newImageClient()
	}
	return imageClient, imageClientError
}

func newImageClient() (*imagev1.ImageV1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := imagev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// DiscoveryClient returns a client-go DiscoveryClient
func DiscoveryClient() (*discovery.DiscoveryClient, error) {
	if discoveryClient == nil && discoveryClientError == nil {
		discoveryClient, discoveryClientError = newDiscoveryClient()
	}
	return discoveryClient, discoveryClientError
}

func newDiscoveryClient() (*discovery.DiscoveryClient, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// RouteClient returns an openshift RouteV1Client
func RouteClient() (*routev1.RouteV1Client, error) {
	if routeClient == nil && routeClientError == nil {
		routeClient, routeClientError = newRouteClient()
	}
	return routeClient, routeClientError
}

func newRouteClient() (*routev1.RouteV1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := routev1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// BuildClient returns an openshift BuildV1Client
func BuildClient() (*buildv1.BuildV1Client, error) {
	if buildClient == nil && buildClientError == nil {
		buildClient, buildClientError = newBuildClient()
	}
	return buildClient, buildClientError
}

func newBuildClient() (*buildv1.BuildV1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := buildv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// AppsClient returns an openshift AppsV1Client
func AppsClient() (*appsv1.AppsV1Client, error) {
	if appsClient == nil && appsClientError == nil {
		appsClient, appsClientError = newAppsClient()
	}
	return appsClient, appsClientError
}

func newAppsClient() (*appsv1.AppsV1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := appsv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// OCPAppsClient returns an openshift AppsV1Client
func OCPAppsClient() (*ocpappsv1.AppsV1Client, error) {
	if ocpAppsClient == nil && ocpAppsClientError == nil {
		ocpAppsClient, ocpAppsClientError = newOCPAppsClient()
	}
	return ocpAppsClient, ocpAppsClientError
}

func newOCPAppsClient() (*ocpappsv1.AppsV1Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	client, err := ocpappsv1.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func init() {
	coreClient, coreClientError = nil, nil
	imageClient, imageClientError = nil, nil
	discoveryClient, discoveryClientError = nil, nil
	routeClient, routeClientError = nil, nil
	buildClient, buildClientError = nil, nil
	ocpAppsClient, ocpAppsClientError = nil, nil
	appsClient, appsClientError = nil, nil
}
