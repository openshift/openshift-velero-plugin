module github.com/konveyor/openshift-velero-plugin

go 1.14

require (
	github.com/containers/image/v5 v5.5.1
	github.com/hashicorp/go-plugin v1.3.0 // indirect
	github.com/openshift/api v0.0.0-20190813152110-b5570061b31f
	github.com/openshift/client-go v0.0.0-20190813201236-5a5508328169
	github.com/openshift/library-go v0.0.0-20190813153448-1eb1131507bf
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/afero v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0 // indirect
	github.com/stretchr/testify v1.6.1
	github.com/vmware-tanzu/velero v0.0.0-eafaef2a4fb13029d7533e501dc57f9073f4e06b
	google.golang.org/grpc v1.30.0 // indirect
	k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77

)

replace github.com/vmware-tanzu/velero v0.0.0-eafaef2a4fb13029d7533e501dc57f9073f4e06b => github.com/konveyor/velero v0.10.2-0.20200122145352-eafaef2a4fb1
