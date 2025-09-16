FROM brew.registry.redhat.io/rh-osbs/openshift-golang-builder:rhel_9_golang_1.24 AS builder
COPY . /app-root
RUN mkdir -p /workspace/src/github.com/konveyor/openshift-velero-plugin
RUN mv /app-root/* /workspace/src/github.com/konveyor/openshift-velero-plugin
WORKDIR /workspace/src/github.com/konveyor/openshift-velero-plugin
ENV BUILDTAGS containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp exclude_graphdriver_overlay include_gcs include_oss strictfipsruntime
ENV BIN velero-plugins
ENV GOEXPERIMENT strictfipsruntime
RUN go build -installsuffix "static" -tags "$BUILDTAGS" -mod=mod -o _output/$BIN ./$BIN

# FROM ubuntu:bionic //
FROM registry.redhat.io/ubi9/ubi:latest
RUN dnf -y install openssl && dnf -y reinstall tzdata && dnf clean all
RUN mkdir /plugins
COPY --from=builder /workspace/src/github.com/konveyor/openshift-velero-plugin/_output/$BIN /plugins/
COPY LICENSE /licenses/
USER nobody:nogroup
ENTRYPOINT ["/bin/bash", "-c", "cp /plugins/* /target/."]

