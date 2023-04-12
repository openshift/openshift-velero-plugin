module github.com/konveyor/openshift-velero-plugin

go 1.16

require (
	github.com/containers/image/v5 v5.19.0
	github.com/hashicorp/go-plugin v1.3.0 // indirect
	github.com/openshift/api v0.0.0-20210105115604-44119421ec6b
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/openshift/library-go v0.0.0-20200521120150-e4959e210d3a
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	github.com/vmware-tanzu/velero v1.7.0
	k8s.io/api v0.24.0
	k8s.io/apimachinery v0.24.0
	k8s.io/client-go v0.24.0
	sigs.k8s.io/controller-runtime v0.12.1
)

// CVE-2021-3121
replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c => github.com/markusthoemmes/goautoneg v0.0.0-20190713162725-c6008fefa5b1

replace github.com/vmware-tanzu/velero => github.com/openshift/velero v1.0.4-0.20230316211430-adb70f88d51e
