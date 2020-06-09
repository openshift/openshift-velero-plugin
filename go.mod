module github.com/konveyor/openshift-velero-plugin

go 1.14

require (
	github.com/containers/image/v5 v5.4.4
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/golang/protobuf v1.3.3-0.20190805180045-4c88cc3f1a34 // indirect
	github.com/google/gofuzz v1.0.0 // indirect
	github.com/googleapis/gnostic v0.3.0 // indirect
	github.com/hashicorp/go-hclog v0.9.2 // indirect
	github.com/hashicorp/go-plugin v1.0.1-0.20190610192547-a1bc61569a26 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	github.com/inconshreveable/mousetrap v1.0.0 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.1 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/openshift/api v0.0.0-20190813152110-b5570061b31f
	github.com/openshift/client-go v0.0.0-20190813201236-5a5508328169
	github.com/openshift/library-go v0.0.0-20190813153448-1eb1131507bf
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/sirupsen/logrus v1.6.0
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/spf13/pflag v1.0.3 // indirect
	github.com/stretchr/testify v1.5.1
	github.com/vmware-tanzu/velero v0.0.0-eafaef2a4fb13029d7533e501dc57f9073f4e06b
	golang.org/x/oauth2 v0.0.0-20190604053449-0f29369cfe45 // indirect
	golang.org/x/text v0.3.2 // indirect
	google.golang.org/appengine v1.6.1 // indirect
	google.golang.org/genproto v0.0.0-20190801165951-fa694d86fc64 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	k8s.io/api v0.0.0-20190819141258-3544db3b9e44
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190819141724-e14f31a72a77
	k8s.io/klog v0.4.0 // indirect
	k8s.io/utils v0.0.0-20190809000727-6c36bc71fc4a // indirect
	sigs.k8s.io/yaml v1.1.0 // indirect
)

replace github.com/vmware-tanzu/velero v0.0.0-eafaef2a4fb13029d7533e501dc57f9073f4e06b => github.com/konveyor/velero v0.10.2-0.20200122145352-eafaef2a4fb1
