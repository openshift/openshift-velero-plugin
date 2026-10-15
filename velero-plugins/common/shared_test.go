package common

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/rest"
)

func TestGetRegistryInfoConfigMap(t *testing.T) {
	const configMapPath = "/api/v1/namespaces/openshift-apiserver/configmaps/config"

	tests := []struct {
		name           string
		configMapCode  int
		configMapBody  string
		wantRegistry   string
		wantErr        bool
		wantCached     bool
		wantConfigGets int32 // configmap requests after two calls
	}{
		{
			name:           "configmap not found returns empty registry and caches",
			configMapCode:  http.StatusNotFound,
			configMapBody:  `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"NotFound","code":404,"details":{"name":"config","kind":"configmaps"}}`,
			wantRegistry:   "",
			wantCached:     true,
			wantConfigGets: 1,
		},
		{
			name:           "configmap with registry hostname",
			configMapCode:  http.StatusOK,
			configMapBody:  `{"kind":"ConfigMap","apiVersion":"v1","metadata":{"name":"config"},"data":{"config.yaml":"{\"imagePolicyConfig\":{\"internalRegistryHostname\":\"image-registry.openshift-image-registry.svc:5000\"}}"}}`,
			wantRegistry:   "image-registry.openshift-image-registry.svc:5000",
			wantCached:     true,
			wantConfigGets: 1,
		},
		{
			name:           "other errors are returned and not cached",
			configMapCode:  http.StatusInternalServerError,
			configMapBody:  `{"kind":"Status","apiVersion":"v1","status":"Failure","reason":"InternalError","code":500}`,
			wantErr:        true,
			wantCached:     false,
			wantConfigGets: 2,
		},
	}

	var (
		configMapCode atomic.Int32
		configMapBody atomic.Value
		configGets    atomic.Int32
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/apis/image.openshift.io/v1/namespaces/openshift/imagestreams":
			w.Write([]byte(`{"kind":"ImageStreamList","apiVersion":"image.openshift.io/v1","items":[]}`))
		case configMapPath:
			configGets.Add(1)
			w.WriteHeader(int(configMapCode.Load()))
			w.Write([]byte(configMapBody.Load().(string)))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	config := &rest.Config{Host: srv.URL}
	clients.SetInClusterConfig(config)
	defer clients.SetInClusterConfig(nil)

	origVersion := serverVersion
	defer func() { serverVersion = origVersion }()
	serverVersion = &serverVersionStruct{Major: 1, Minor: 30}

	log := logrus.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registryInfo = nil
			defer func() { registryInfo = nil }()
			configGets.Store(0)
			configMapCode.Store(int32(tt.configMapCode))
			configMapBody.Store(tt.configMapBody)
			if _, err := clients.CoreClientFromConfig(config); err != nil {
				t.Fatal(err)
			}

			for i := 0; i < 2; i++ {
				got, err := GetRegistryInfo(log)
				if (err != nil) != tt.wantErr {
					t.Fatalf("call %d: err = %v, wantErr %v", i, err, tt.wantErr)
				}
				if got != tt.wantRegistry {
					t.Fatalf("call %d: registry = %q, want %q", i, got, tt.wantRegistry)
				}
			}
			if cached := registryInfo != nil; cached != tt.wantCached {
				t.Errorf("cached = %v, want %v", cached, tt.wantCached)
			}
			if got := configGets.Load(); got != tt.wantConfigGets {
				t.Errorf("configmap GETs = %d, want %d", got, tt.wantConfigGets)
			}
		})
	}
}
