module github.com/konveyor/openshift-velero-plugin

go 1.14

require (
	github.com/bombsimon/logrusr/v3 v3.0.0
	github.com/containers/image/v5 v5.21.1
	github.com/go-logr/logr v1.2.3
	github.com/hashicorp/go-hclog v1.0.0 // indirect
	github.com/kaovilai/udistribution v0.0.7-oadp-1.1
	github.com/mitchellh/mapstructure v1.4.3 // indirect
	github.com/openshift/api v0.0.0-20210805075156-d8fab4513288
	github.com/openshift/client-go v0.0.0-20210112165513-ebc401615f47
	github.com/openshift/library-go v0.0.0-20200521120150-e4959e210d3a
	github.com/openshift/oadp-operator v1.0.3
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.8.0
	github.com/vmware-tanzu/velero v1.11.0
	k8s.io/api v0.25.6
	k8s.io/apimachinery v0.25.6
	k8s.io/client-go v0.25.6
)

// CVE-2021-41190
replace github.com/opencontainers/image-spec => github.com/opencontainers/image-spec v1.0.2-0.20211123152302-43a7dee1ec31

// CVE-2021-3121
replace github.com/gogo/protobuf => github.com/gogo/protobuf v1.3.2

replace bitbucket.org/ww/goautoneg v0.0.0-20120707110453-75cd24fc2f2c => github.com/markusthoemmes/goautoneg v0.0.0-20190713162725-c6008fefa5b1

replace github.com/vmware-tanzu/velero => github.com/openshift/velero v0.10.2-0.20230313143326-9741b1028da4

// quickfix: GCS driver until https://github.com/distribution/distribution/pull/3702 is merged
replace github.com/distribution/distribution/v3 => github.com/konveyor/distribution v0.0.0-20220729202420-aa6c4b213d75
