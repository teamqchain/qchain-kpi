#!/bin/bash
# Correctness gate run inside the image build, right after liboqs is built.
# The image build FAILS if any check fails, so no benchmark can ever run on a
# library that has not passed these tests on that very machine.
#
#   1. NIST ACVP vector files: downloaded once, checked against pinned SHA-256.
#   2. liboqs's own pytest suites for the enabled algorithms:
#        tests/test_cmdline.py      test_kem / test_sig round trips per algorithm
#        tests/test_kat.py          known-answer-test hashes (liboqs skips SLH-DSA here)
#        tests/test_acvp_vectors.py NIST ACVP vectors for ML-KEM, ML-DSA and SLH-DSA
#      The example programs are skipped (SKIP_TESTS=examples): they include a
#      stateful-signature demo, and stateful algorithms are not built.
set -euo pipefail

info=/opt/kpi/build-info
acvp=/opt/kpi/acvp
root="https://raw.githubusercontent.com/usnistgov/ACVP-Server/refs/tags/v1.1.0.41/gen-val/json-files"
mkdir -p "$info" "$acvp"

while read -r sum path; do
    case "$sum" in ''|'#'*) continue ;; esac
    mkdir -p "$acvp/$(dirname "$path")"
    curl -sSfL --retry 3 -o "$acvp/$path" "$root/$path"
    echo "$sum  $acvp/$path" | sha256sum -c - >> "$info/acvp-sha256-check.txt"
done < /opt/kpi/docker/acvp-sha256.txt

cp /opt/kpi/docker/acvp_conftest.py /src/liboqs/tests/conftest.py

cd /src/liboqs
set +e
OQS_BUILD_DIR=/src/liboqs/build KPI_ACVP_DIR="$acvp" SKIP_TESTS=examples \
    python3 -m pytest -q -p no:cacheprovider --numprocesses=auto \
    tests/test_cmdline.py tests/test_kat.py tests/test_acvp_vectors.py \
    > "$info/oqs-gate.txt" 2>&1
status=$?
set -e
tail -n 5 "$info/oqs-gate.txt"
if [ "$status" -ne 0 ]; then
    echo "liboqs correctness gate FAILED (pytest exit $status)" >&2
    tail -n 80 "$info/oqs-gate.txt" >&2
    exit 1
fi
echo "liboqs correctness gate passed"
