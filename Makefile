# Copyright 2019 Red Hat Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

BINS = $(wildcard velero-*)

REPO ?= github.com/konveyor/openshift-velero-plugin

BUILD_IMAGE ?= openshift/origin-release:golang-1.14

IMAGE ?= docker.io/konveyor/openshift-velero-plugin

ARCH ?= amd64
BUILDTAGS ?= "containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp exclude_graphdriver_overlay include_gcs include_oss"

all: $(addprefix build-, $(BINS))

build-%:
	$(MAKE) --no-print-directory BIN=$* build

build: _output/$(BIN)

_output/$(BIN): $(BIN)/*.go
	mkdir -p .go/src/$(REPO) .go/pkg .go/.cache .go/std/$(ARCH) _output
	cp -rp * .go/src/$(REPO)
	docker run \
				 --rm \
				 -v $$(pwd)/.go/pkg:/go/pkg:z \
				 -v $$(pwd)/.go/src:/go/src:z \
				 -v $$(pwd)/.go/std:/go/std:z \
				 -v $$(pwd)/.go/.cache:/go/.cache:z \
				 -v $$(pwd)/_output:/go/src/$(REPO)/_output:z \
				 -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static:z \
				 -w /go/src/$(REPO) \
				 $(BUILD_IMAGE) \
				 go build -installsuffix "static" -tags $(BUILDTAGS) -i -v -o _output/$(BIN) ./$(BIN)

DOCKER_BUILD_ARGS ?= --platform=linux/amd64
container:
	docker build -t $(IMAGE) . $(DOCKER_BUILD_ARGS)

test: envtest
	KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS) go test -installsuffix "static" -tags $(BUILDTAGS) ./velero-plugins/...

ci: all test

clean:
	rm -rf .go _output

GOPATH:=$(shell go env GOPATH)
GOBIN:=$(GOPATH)/bin
GOSRC:=$(GOPATH)/src
#  if KUBEBUILDER_ASSETS contains space, escape it
KUBEBUILDER_ASSETS=$(shell echo $(shell $(GOBIN)/setup-envtest use -p path) | sed 's/ /\\ /g')
.PHONY: envtest
# When debugging tests in vscode, this is example content of .vscode/settings.json
# which is the output of `setup-envtest use -p path`
# {
#     "go.testEnvVars": {
#         "KUBEBUILDER_ASSETS": "/Users/tiger/Library/Application Support/io.kubebuilder.envtest/k8s/1.26.1-darwin-arm64"
#     }
# }
envtest: $(GOBIN)/setup-envtest
	$(GOBIN)/setup-envtest use -p path

$(GOBIN)/setup-envtest:
	@echo Installing envtest tools
	GOFLAGS= go install sigs.k8s.io/controller-runtime/tools/setup-envtest@v0.0.0-20240320141353-395cfc7486e6
	@echo Installed envtest tools
