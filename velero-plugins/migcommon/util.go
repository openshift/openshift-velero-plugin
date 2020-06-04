package migcommon

import (
	corev1API "k8s.io/api/core/v1"
)

// ConfigureContainerSleep replaces the pod, cmd, and arg on containers so that
// post stage or migrate phase 1 restores applications do not start
func ConfigureContainerSleep(containers []corev1API.Container, duration string) {
	for n, _ := range containers {
		if containers[n].Name != "restic-wait" {
			containers[n].Image = "registry.access.redhat.com/rhel7"
			containers[n].Command = []string{"sleep"}
			containers[n].Args = []string{duration}
		}
	}
}
