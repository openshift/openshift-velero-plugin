package openshift

import (
	"context"

	"github.com/konveyor/openshift-velero-plugin/velero-plugins/clients"
	buildv1 "github.com/openshift/api/build/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsBuildPod(name, namespace string) (bool, error) {
	cc, err := clients.CoreClient()
	if err != nil {
		return false, err
	}
	pod, err := cc.Pods(namespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	return isBuildPodInternal(pod), nil
}

// Check if pod is a build pod via it's annotations
// If used in RIA, use unmarshal from input.ItemFromBackup because input.Item may have metadata reset
func PodIsBuildPod(pod *corev1.Pod) bool {
	return isBuildPodInternal(pod)
}

// https://github.com/openshift/openshift-controller-manager/blob/aabcbc2cf5d944f64e6ebdbc4e0ce7f1b95bd127/pkg/build/controller/build/build_controller.go#L2549
func isBuildPodInternal(pod *corev1.Pod) bool {
	return len(getBuildName(pod)) > 0
}

// https://github.com/openshift/openshift-controller-manager/blob/aabcbc2cf5d944f64e6ebdbc4e0ce7f1b95bd127/pkg/build/controller/build/build_controller.go#L2715
func getBuildName(pod metav1.Object) string {
	if pod == nil {
		return ""
	}
	return pod.GetAnnotations()[buildv1.BuildAnnotation]
}
