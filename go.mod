module github.com/konveyor/openshift-velero-plugin

go 1.14

require (
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
	github.com/stretchr/testify v1.6.1
	github.com/vmware-tanzu/velero v1.6.0
	golang.org/x/sys v0.0.0-20210303074136-134d130e1a04 // indirect
	k8s.io/api v0.20.0
	k8s.io/apimachinery v0.20.0
	k8s.io/client-go v0.20.0
)

replace bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c => github.com/markusthoemmes/goautoneg v0.0.0-20190713162725-c6008fefa5b1

replace github.com/vmware-tanzu/velero => github.com/konveyor/velero v0.10.2-0.20210415185542-05bae5e157fa
