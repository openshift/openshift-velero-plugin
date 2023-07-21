package mock

import "k8s.io/client-go/rest"

func MockInClusterConfig() (*rest.Config, error) {
	return nil, nil
}