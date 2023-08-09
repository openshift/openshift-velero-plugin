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
FROM quay.io/konveyor/builder as builder
ENV GOPATH=$APP_ROOT
ENV BUILDTAGS containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp exclude_graphdriver_overlay include_gcs include_oss
ENV BIN velero-plugins
WORKDIR $APP_ROOT/src/github.com/konveyor/openshift-velero-plugin
COPY --chown=1001 go.mod go.sum $APP_ROOT/src/github.com/konveyor/openshift-velero-plugin/
RUN go mod download
COPY --chown=1001 . $APP_ROOT/src/github.com/konveyor/openshift-velero-plugin
RUN go build -buildvcs=false -installsuffix "static" -tags "$BUILDTAGS" -o _output/$BIN ./$BIN

FROM registry.access.redhat.com/ubi8-minimal
RUN mkdir /plugins
COPY --from=builder /opt/app-root/src/github.com/konveyor/openshift-velero-plugin/_output/$BIN /plugins/
USER 65534:65534
ENTRYPOINT ["/bin/bash", "-c", "cp /plugins/* /target/."]
