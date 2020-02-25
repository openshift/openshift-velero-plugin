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

BUILD_IMAGE ?= openshift/origin-release:golang-1.11

IMAGE ?= docker.io/konveyor/openshift-velero-plugin

ARCH ?= amd64

all: $(addprefix build-, $(BINS))

build-%:
	$(MAKE) --no-print-directory BIN=$* build

build: _output/$(BIN)

_output/$(BIN): $(BIN)/*.go
	mkdir -p .go/src/$(REPO) .go/pkg .go/.cache .go/std/$(ARCH) _output
	cp -rp * .go/src/$(REPO)
	docker run \
				 --rm \
				 -u $$(id -u):$$(id -g) \
				 -v $$(pwd)/.go/pkg:/go/pkg:z \
				 -v $$(pwd)/.go/src:/go/src:z \
				 -v $$(pwd)/.go/std:/go/std:z \
				 -v $$(pwd)/.go/.cache:/go/.cache:z \
				 -v $$(pwd)/_output:/go/src/$(REPO)/_output:z \
				 -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static:z \
				 -e CGO_ENABLED=0 \
				 -w /go/src/$(REPO) \
				 $(BUILD_IMAGE) \
				 go build -installsuffix "static"  -i -v -o _output/$(BIN) ./$(BIN)

container:
	docker build -t $(IMAGE) .

test:
	go test -installsuffix "static"  ./velero-plugins/...

ci: all test

clean:
	rm -rf .go _output
