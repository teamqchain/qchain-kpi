# syntax=docker/dockerfile:1
#
# QChain KPI benchmarking image. Built natively on each machine
# (arm64 on the M1 Mac, amd64 on the Windows laptop and the VM), never emulated.
#
#   docker buildx build --builder kpi --load -f docker/Dockerfile \
#       --build-arg VARIANT=b1-prod --build-arg HARNESS_COMMIT=<sha> \
#       -t kpi-bench:b1-prod-<shortsha> .
#
# Targets:
#   oqs      liboqs 0.16.0 built for VARIANT + its correctness gate (oqs-gate.sh)
#   test     go vet + go test of the whole harness (used by scripts/dev.sh)
#   runtime  the measurement image (default)
#
# Base images are pinned by index digest (resolved 2026-10-10). The builder base
# is the image production's offchain/Dockerfile resolves to (golang:1.24-bookworm
# = Go 1.24.13); the runtime base matches production's debian:bookworm-slim.
ARG GO_IMAGE=golang:1.24.13-bookworm@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac
ARG RUNTIME_IMAGE=debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587
ARG APT_SNAPSHOT=20261009T000000Z

# ---------------------------------------------------------------------------
FROM ${GO_IMAGE} AS base
ARG APT_SNAPSHOT
COPY docker/apt-setup.sh /opt/kpi/docker/apt-setup.sh
RUN /opt/kpi/docker/apt-setup.sh builder "${APT_SNAPSHOT}" \
        ca-certificates curl git cmake ninja-build build-essential libssl-dev pkg-config \
        python3 python3-pytest python3-pytest-xdist python3-requests python3-yaml

# ---------------------------------------------------------------------------
FROM base AS oqs
ARG VARIANT=b1-prod
ARG LIBOQS_TAG=0.16.0
ARG LIBOQS_COMMIT=5a1a854b0dc9f2141bdc771c555ee60c37950183
COPY docker/oqs-build.sh docker/oqs-gate.sh docker/acvp-sha256.txt docker/acvp_conftest.py /opt/kpi/docker/
RUN VARIANT="${VARIANT}" LIBOQS_TAG="${LIBOQS_TAG}" LIBOQS_COMMIT="${LIBOQS_COMMIT}" /opt/kpi/docker/oqs-build.sh
RUN /opt/kpi/docker/oqs-gate.sh

# ---------------------------------------------------------------------------
FROM oqs AS harness
# Same pkg-config shim as production (offchain/Dockerfile), so liboqs-go's
# "#cgo pkg-config: liboqs-go" resolves to the liboqs we just built.
RUN mkdir -p /usr/lib/pkgconfig && printf '%s\n' \
        'Name: liboqs-go' \
        'Description: Go bindings for liboqs' \
        'Version: 0.16.0' \
        'Cflags: -I/usr/local/include' \
        'Libs: -L/usr/local/lib -loqs' > /usr/lib/pkgconfig/liboqs-go.pc
ENV PKG_CONFIG_PATH=/usr/lib/pkgconfig \
    LD_LIBRARY_PATH=/usr/local/lib \
    CGO_ENABLED=1 \
    GOTOOLCHAIN=local \
    GOFLAGS=-trimpath
WORKDIR /src/qchain-kpi
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY . .

# ---------------------------------------------------------------------------
FROM harness AS test
# /bin/sh is dash, which has no pipefail: the test output goes to a file and
# the status of go test itself decides whether the build continues.
RUN go vet ./... && \
    if go test -count=1 ./... > /opt/kpi/build-info/go-test.txt 2>&1; then \
        cat /opt/kpi/build-info/go-test.txt; \
    else \
        cat /opt/kpi/build-info/go-test.txt; echo "go test FAILED" >&2; exit 1; \
    fi

# ---------------------------------------------------------------------------
FROM harness AS build
ARG HARNESS_COMMIT=unknown
# The measurement image can only be built after the test stage has passed.
COPY --from=test /opt/kpi/build-info/go-test.txt /opt/kpi/build-info/go-test.txt
RUN mkdir -p /opt/kpi/bin && \
    go build -ldflags "-X qchain-kpi/internal/buildinfo.HarnessCommit=${HARNESS_COMMIT}" -o /opt/kpi/bin/ ./cmd/... && \
    go version > /opt/kpi/build-info/go-version.txt

# ---------------------------------------------------------------------------
FROM ${RUNTIME_IMAGE} AS runtime
ARG APT_SNAPSHOT
ARG VARIANT=b1-prod
ARG HARNESS_COMMIT=unknown
COPY docker/apt-setup.sh /opt/kpi/docker/apt-setup.sh
RUN /opt/kpi/docker/apt-setup.sh runtime "${APT_SNAPSHOT}" ca-certificates libssl3
COPY --from=build /usr/local/lib/liboqs.so* /usr/local/lib/
COPY --from=build /opt/kpi/bin/ /opt/kpi/bin/
COPY --from=build /opt/kpi/build-info/ /opt/kpi/build-info/
COPY plans/ /opt/kpi/plans/
COPY config/ /opt/kpi/config/
# /out is the mount point of the run's named volume; a new volume is filled
# from here, so monitor/ (where the launcher delivers host samples) exists
# from the first second and belongs to the unprivileged user.
RUN ldconfig && useradd --uid 10001 --create-home --shell /usr/sbin/nologin kpi && \
    mkdir -p /out/monitor && chown -R kpi:kpi /out
# Production parity: the backend runs in Dubai time and resolves it from the
# tzdata embedded in the binary (offchain/config.go imports time/tzdata).
ENV TZ=Asia/Dubai \
    LD_LIBRARY_PATH=/usr/local/lib \
    PATH=/opt/kpi/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
LABEL org.qchain.kpi="1" \
      org.qchain.kpi.variant="${VARIANT}" \
      org.qchain.kpi.liboqs="0.16.0" \
      org.qchain.kpi.harness-commit="${HARNESS_COMMIT}"
USER kpi
WORKDIR /out
ENTRYPOINT ["/opt/kpi/bin/kpi-batch"]
