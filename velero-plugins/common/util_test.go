// tests for util.go

package common

import (
	"testing"
	"errors"
	"reflect"
        "github.com/sirupsen/logrus"
        corev1API "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)


// ReplaceImageRefPrefix replaces an image reference prefix with newPrefix.
// If the input image reference does not start with oldPrefix, an error is returned
func TestReplaceImageRefPrefix(t *testing.T) {
	tests := map[string]struct {
		s, oldPrefix, newPrefix string
		namespaceMapping map[string]string
                exp string
		expErr error
        }{
		"1": {s: "foo/baz/cat", oldPrefix: "foo", newPrefix: "bar", namespaceMapping: map[string]string{"baz": "qux"}, exp: "bar/qux/cat", expErr: nil},
		"2": {s: "foo/baz/cat", oldPrefix: "foo", newPrefix: "bar", namespaceMapping: map[string]string{}, exp: "bar/baz/cat", expErr: nil},
		"3": {s: "foo/baz", oldPrefix: "foo", newPrefix: "bar", namespaceMapping: map[string]string{}, exp: "bar/baz", expErr: nil},
		"4": {s: "foo/baz", oldPrefix: "foob", newPrefix: "bar", namespaceMapping: map[string]string{}, exp: "", expErr: errors.New("")},
		"5": {s: "foo", oldPrefix: "fo", newPrefix: "bar", namespaceMapping: map[string]string{}, exp: "", expErr: errors.New("")},
		"6": {s: "foo/openshift/cat@swan", oldPrefix: "foo", newPrefix: "bar", namespaceMapping: map[string]string{}, exp: "bar/openshift/cat", expErr: nil},
        }

        for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
                        got, err := ReplaceImageRefPrefix(tc.s, tc.oldPrefix, tc.newPrefix, tc.namespaceMapping)
			if tc.expErr == nil && got != tc.exp {
                                t.Fatalf("expected: %v, got: %v", tc.exp, got)
                        }
			if tc.expErr != nil && err == nil {
				t.Fatalf("expected error, got no error")
			}
                })
        }
}


// HasImageRefPrefix returns true if the input image reference begins with
// the input prefix followed by "/"
func TestHasImageRefPrefix(t *testing.T) {
	tests := map[string]struct {
		s, prefix string
		want	  bool
	}{
		"1": {s: "cat/", prefix: "cat", want: true},
		"2": {s: "catt/", prefix: "cat", want: false},
		"3": {s: "cat/dog/spider", prefix: "cat", want: true},
		"4": {s: "//ss", prefix: "", want: true},
		"5": {s: "abc", prefix: "", want: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := HasImageRefPrefix(tc.s, tc.prefix)
			if got != tc.want {
				t.Fatalf("expected: %v, got: %v", tc.want, got)
			}
		})
	}
}


// ParseLocalImageReference
func TestParseLocalImageReference(t *testing.T) {
        tests := map[string]struct {
                s, prefix string
                exp       *LocalImageReference
		expErr	  error
        }{
		"1": {s: "reg/ns/name@dig", prefix: "reg", exp: &LocalImageReference{Registry: "reg", Namespace: "ns", Name: "name", Digest: "dig"}, expErr: nil},
		"2": {s: "reg/ns/name@dig:est", prefix: "reg", exp: &LocalImageReference{Registry: "reg", Namespace: "ns", Name: "name", Digest: "dig:est"}, expErr: nil},
		"3": {s: "reg/ns/name:tg", prefix: "reg", exp: &LocalImageReference{Registry: "reg", Namespace: "ns", Name: "name", Tag: "tg"}, expErr: nil},
		"4": {s: "reg/ns/name@dig", prefix: "cat", exp: nil, expErr: errors.New("")},
		"5": {s: "reg/cat", prefix: "reg", exp: nil, expErr: errors.New("")},
		"6": {s: "reg/ns/name/dig", prefix: "reg", exp: nil, expErr: errors.New("")},
		"7": {s: "reg/ns/name@dig@est", prefix: "reg", exp: nil, expErr: errors.New("")},
		"8": {s: "reg/ns/name:ta:g", prefix: "reg", exp: nil, expErr: errors.New("")},
		"9": {s: "reg/ns/name", prefix: "reg", exp: &LocalImageReference{Registry: "reg", Namespace: "ns", Name: "name"}, expErr: nil},
        }

        for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
                        got, err := ParseLocalImageReference(tc.s, tc.prefix)
			if tc.expErr == nil && !reflect.DeepEqual(got, tc.exp) {
                                t.Fatalf("expected: %v, got: %v", tc.exp, got)
                        }
			if tc.expErr != nil && err == nil {
				t.Fatalf("expected error, got no error")
			}
                })
        }
}


// SwapContainerImageRefs updates internal image references from
// backup registry to restore registry pathnames
func TestSwapContainerImageRefs(t *testing.T) {
        tests := map[string]struct {
		containers		 []corev1API.Container
		oldRegistry, newRegistry string
		log			 logrus.FieldLogger
		namespaceMapping	 map[string]string
                exp			 []corev1API.Container
        }{
		"1": {
			containers: []corev1API.Container{
				corev1API.Container{Image: "foo/cat"},
				corev1API.Container{Image: "foo/cat/y"},
				corev1API.Container{Image: "foo/dog/x"},
				corev1API.Container{Image: "boo/cat"}},
			oldRegistry: "foo",
			newRegistry: "bar",
			log: logrus.New(),
			namespaceMapping: map[string]string{"dog": "puppy", "cat": "kitten"},
			exp: []corev1API.Container{
				corev1API.Container{Image: "bar/cat"},
				corev1API.Container{Image: "bar/kitten/y"},
				corev1API.Container{Image: "bar/puppy/x"},
				corev1API.Container{Image: "boo/cat"}},
		},
        }

        for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
                        SwapContainerImageRefs(tc.containers, tc.oldRegistry, tc.newRegistry, tc.log, tc.namespaceMapping)
                        if !reflect.DeepEqual(tc.containers, tc.exp) {
                                t.Fatalf("expected: %v, got: %v", tc.exp, tc.containers)
                        }
                })
        }
}


// UpdatePullSecret updates registry pull (or push) secret
// with a secret found in the dest cluster
func TestUpdatePullSecret(t *testing.T) {
	tests := map[string]struct {
	        secretRef  *corev1API.LocalObjectReference
	        secretList *corev1API.SecretList
	        log	   logrus.FieldLogger
                exp        *corev1API.LocalObjectReference
                expErr     error
        }{
                "1": {
			secretRef: &corev1API.LocalObjectReference{Name: "foo"},
			secretList: &corev1API.SecretList{
				Items: []corev1API.Secret{
					corev1API.Secret{},
				},
			},
			log: logrus.New(),
			exp: &corev1API.LocalObjectReference{Name: "foo"},
                        expErr: nil,
                },
                "2": {
			secretRef: &corev1API.LocalObjectReference{Name: "default-dockercfg-foo"},
			secretList: &corev1API.SecretList{
				Items: []corev1API.Secret{
					corev1API.Secret{ObjectMeta: metav1.ObjectMeta{Name: "default-dockercfg-bar"}},
				},
			},
			log: logrus.New(),
			exp: &corev1API.LocalObjectReference{Name: "default-dockercfg-bar"},
                        expErr: nil,
                },
                "3": {
			secretRef: &corev1API.LocalObjectReference{Name: "deployer-dockercfg-foo"},
			secretList: &corev1API.SecretList{
				Items: []corev1API.Secret{
					corev1API.Secret{ObjectMeta: metav1.ObjectMeta{Name: "default-dockercfg-bar"}},
					corev1API.Secret{ObjectMeta: metav1.ObjectMeta{Name: "deployer-dockercfg-cat"}},
					corev1API.Secret{ObjectMeta: metav1.ObjectMeta{Name: "deployer-dockercfg-dog"}},
				},
			},
			log: logrus.New(),
			exp: &corev1API.LocalObjectReference{Name: "deployer-dockercfg-cat"},
                        expErr: nil,
                },
        }

        for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
                        got, err := UpdatePullSecret(tc.secretRef, tc.secretList, tc.log)
                        if tc.expErr == nil && !reflect.DeepEqual(got, tc.exp) {
                                t.Fatalf("expected: %v, got: %v", tc.exp, got)
                        }
                        if tc.expErr != nil && err == nil {
                                t.Fatalf("expected error, got no error")
                        }
                })
        }
}


// GetSrcAndDestRegistryInfo returns the Registry hostname for both src and dest clusters
func TestGetSrcAndDestRegistryInfo(t *testing.T) {
        tests := map[string]struct {
		item   runtime.Unstructured
		exp1   string
		exp2   string
		expErr error
	}{
		"1": {
			//item: runtime.Unstructured{Object{Annotations: {BackupRegistryHostname: "AA", RestoreRegistryHostname: "BB"}}},
			item: unstructured.Unstructured{Object: map[string]interface{}},
			exp1: "AA",
			exp2: "BB",
			expErr: nil,
		},
	}

        for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
			got1, got2, err := GetSrcAndDestRegistryInfo(tc.item)
			if tc.expErr == nil && (got1 != tc.exp1 || got2 != tc.exp2) {
                                t.Fatalf("expected: [%v, %v], got: [%v, %v]", tc.exp1, tc.exp2, got1, got2)
                        }
			if tc.expErr != nil && err == nil {
				t.Fatalf("expected error, got no error")
			}
                })
        }
}


