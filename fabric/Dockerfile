# syntax=docker/dockerfile:1
#
# E14 client image (the build context is the fabric/ folder):
#
#   docker buildx build --builder kpi --load -f fabric/Dockerfile \
#       --build-arg HARNESS_COMMIT=<sha> -t kpi-fabric-client:<shortsha> fabric
#
# Targets:
#   modfiles  go.mod and go.sum after `go mod tidy` (exported with --output)
#   test      go vet and the client's unit tests
#   runtime   the client (default)
#
# The base images are the harness's (docker/Dockerfile), pinned by digest.
ARG GO_IMAGE=golang:1.24.13-bookworm@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac
ARG RUNTIME_IMAGE=debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587

FROM ${GO_IMAGE} AS gobase
ENV CGO_ENABLED=0 GOTOOLCHAIN=local GOFLAGS=-trimpath
WORKDIR /src

FROM gobase AS tidy
COPY client/ ./
RUN go mod tidy

FROM scratch AS modfiles
COPY --from=tidy /src/go.mod /src/go.sum /

FROM gobase AS deps
COPY client/go.mod client/go.sum ./
RUN go mod download && go mod verify

FROM deps AS test
# The plan files sit next to the client folder, as in the repository
# (TestPlanFiles reads ../plans).
WORKDIR /work/client
COPY client/ ./
COPY plans/ /work/plans/
RUN go vet ./... && (go test -count=1 ./... > /test.txt 2>&1; s=$?; cat /test.txt; exit $s)

FROM deps AS build
ARG HARNESS_COMMIT=unknown
COPY client/ ./
COPY --from=test /test.txt /build-info/go-test.txt
RUN go build -ldflags "-X main.harnessCommit=${HARNESS_COMMIT}" -o /bin-out/kpi-fabric . && \
    go version > /build-info/go-version.txt && go list -m all > /build-info/go-modules.txt

FROM ${RUNTIME_IMAGE} AS runtime
ARG HARNESS_COMMIT=unknown
COPY --from=build /bin-out/kpi-fabric /opt/kpi/bin/kpi-fabric
COPY --from=build /build-info/ /opt/kpi/build-info/
COPY plans/ /opt/kpi/plans/
RUN mkdir -p /out/monitor
# Production parity: the backend runs in Dubai time (the binary embeds tzdata).
ENV TZ=Asia/Dubai PATH=/opt/kpi/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
LABEL org.qchain.kpi="1" org.qchain.kpi.e14="1" org.qchain.kpi.harness-commit="${HARNESS_COMMIT}"
WORKDIR /out
ENTRYPOINT ["/opt/kpi/bin/kpi-fabric"]
