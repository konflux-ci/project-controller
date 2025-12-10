# Build the manager binary
FROM registry.access.redhat.com/ubi9/go-toolset:9.7-1765311584 AS builder
ARG TARGETOS
ARG TARGETARCH

ENV GOTOOLCHAIN=auto
WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/
COPY pkg/ pkg/

# Build
# the GOARCH has not a default value to allow the binary be built according to the host where the command
# was called. For example, if we call make docker-build in a local env which has the Apple Silicon M1 SO
# the docker BUILDPLATFORM arg will be linux/arm64 when for Apple x86 it will be linux/amd64. Therefore,
# by leaving it empty we can ensure that the container and binary shipped on it will have the same platform.
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o /opt/app-root/manager cmd/main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://catalog.redhat.com/software/containers/ubi9-micro/61832b36dd607bfc82e66399 for more details
FROM registry.access.redhat.com/ubi9-micro@sha256:e14a8cbcaa0c26b77140ac85d40a47b5e910a4068686b02ebcad72126e9b5f86
WORKDIR /
COPY --from=builder /opt/app-root/manager .
COPY LICENSE /licenses/
USER 65532:65532

# It is mandatory to set these labels
LABEL name="Konflux Project Controller"
LABEL description="Konflux Project Controller"
LABEL com.redhat.component="Konflux Project Controller"
LABEL io.k8s.description="Konflux Project Controller"
LABEL io.k8s.display-name="konflux-project-controller"
LABEL version="1.0"
LABEL release="1"
LABEL vendor="Red Hat, Inc."
LABEL distribution-scope="public"
LABEL url="https://github.com/konflux-ci/project-controller"

ENTRYPOINT ["/manager"]
