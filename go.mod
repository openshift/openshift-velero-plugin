module github.com/konveyor/openshift-velero-plugin

go 1.14

require (
	github.com/Azure/go-autorest v11.1.2+incompatible // indirect
	github.com/bombsimon/logrusr v0.0.0-20200131103305-03a291ce59b4
	github.com/containers/image/v5 v5.5.1
	github.com/go-logr/logr v0.1.0
	github.com/hashicorp/go-plugin v1.3.0 // indirect
	github.com/openshift/api v0.0.0-20200210091934-a0e53e94816b
	github.com/openshift/client-go v0.0.0-20200116152001-92a2713fa240
	github.com/openshift/library-go v0.0.0-20200521120150-e4959e210d3a
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/afero v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/vmware-tanzu/velero v1.4.2
	google.golang.org/grpc v1.30.0 // indirect
	k8s.io/api v0.17.4
	k8s.io/apimachinery v0.17.4
	k8s.io/client-go v0.17.4

)

replace github.com/vmware-tanzu/velero => github.com/konveyor/velero v0.10.2-0.20200805201016-d2b7e756cba8
