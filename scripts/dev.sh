#!/bin/bash
# Development check: builds the image's test stage on this machine, which runs
# go vet and go test of the whole harness against the real liboqs 0.16.0
# build (after liboqs's own correctness gate). The Mac needs no native liboqs
# or OpenSSL for this.
#
#   scripts/dev.sh [VARIANT]     (default b1-prod)
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
variant="${1:-b1-prod}"
status=0
docker buildx build --builder kpi --target test --progress=plain \
  --build-arg "VARIANT=$variant" -f "$repo/docker/Dockerfile" "$repo" || status=$?
# The builder container is stopped after every build so it never runs
# alongside a measurement.
docker buildx stop kpi >/dev/null 2>&1 || true
exit "$status"
