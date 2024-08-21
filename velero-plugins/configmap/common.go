package configmap

import "strings"

// Interested ConfigMap Suffixes
// https://github.com/openshift/openshift-controller-manager/blob/aabcbc2cf5d944f64e6ebdbc4e0ce7f1b95bd127/pkg/build/buildutil/util.go#L44-L46
const (
	// ca covers caconfig and global-ca configmap suffixes
	caConfigMapSuffix        = "ca"
	sysConfigConfigMapSuffix = "sys-config"
)

// if configmap name has suffix that could be related to ocp build.
func nameHasBuildSuffix(name string) bool {
	// name pattern is buildPod-<suffix>
	return strings.HasSuffix(name, caConfigMapSuffix) || // this covers globalCA as well
		strings.HasSuffix(name, sysConfigConfigMapSuffix)
}
