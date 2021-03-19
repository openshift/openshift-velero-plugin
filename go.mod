module github.com/konveyor/openshift-velero-plugin

go 1.14

require (
	github.com/Azure/go-autorest v11.1.2+incompatible // indirect
	github.com/bombsimon/logrusr v1.0.0
	github.com/containers/image/v5 v5.5.1
	github.com/go-logr/logr v0.4.0
	github.com/hashicorp/go-plugin v1.3.0 // indirect
	github.com/magefile/mage v1.11.0 // indirect
	github.com/openshift/api v0.0.0-20210105115604-44119421ec6b
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/openshift/library-go v0.0.0-20200521120150-e4959e210d3a
	github.com/sirupsen/logrus v1.8.0
	github.com/spf13/afero v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/vmware-tanzu/velero v1.5.3
	golang.org/x/sys v0.0.0-20210303074136-134d130e1a04 // indirect
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0

)

replace k8s.io/api => k8s.io/api v0.18.4

replace k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.18.4

replace k8s.io/apimachinery => k8s.io/apimachinery v0.18.4

replace k8s.io/apiserver => k8s.io/apiserver v0.18.4

replace k8s.io/cli-runtime => k8s.io/cli-runtime v0.18.4

replace k8s.io/client-go => k8s.io/client-go v0.18.4

replace k8s.io/cloud-provider => k8s.io/cloud-provider v0.18.4

replace k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.18.4

replace k8s.io/code-generator => k8s.io/code-generator v0.18.4

replace k8s.io/component-base => k8s.io/component-base v0.18.4

replace k8s.io/cri-api => k8s.io/cri-api v0.18.4

replace k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.18.4

replace k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.18.4

replace k8s.io/kube-proxy => k8s.io/kube-proxy v0.18.4

replace k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.18.4

replace k8s.io/kubectl => k8s.io/kubectl v0.18.4

replace k8s.io/kubelet => k8s.io/kubelet v0.18.4

replace k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.18.4

replace k8s.io/metrics => k8s.io/metrics v0.18.4

replace k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.18.4

replace k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.18.4

replace github.com/vmware-tanzu/velero => github.com/sseago/velero v0.10.2-0.20210316144755-b3538d9469d5
