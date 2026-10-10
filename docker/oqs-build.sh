#!/bin/bash
# Builds liboqs 0.16.0 (pinned commit) for one build variant and installs it to
# /usr/local. Records the exact configuration in /opt/kpi/build-info/.
#
# Variants (research plan §6, coding plan §5):
#   b1-prod      production-like: runtime CPU dispatch (OQS_DIST_BUILD=ON), OpenSSL on
#   b2-portable  no optimised code paths: generic target (+ ARM extensions off on arm64)
#   b3-native    tuned for the build machine (-march/-mcpu=native), no runtime dispatch
#   b4-noossl    as b1 but without OpenSSL (also switches liboqs's RNG to the system source)
#
# Environment: VARIANT, LIBOQS_TAG, LIBOQS_COMMIT
set -euo pipefail

: "${VARIANT:?}" "${LIBOQS_TAG:?}" "${LIBOQS_COMMIT:?}"
info=/opt/kpi/build-info
mkdir -p "$info"

git clone --quiet --depth 1 --branch "$LIBOQS_TAG" https://github.com/open-quantum-safe/liboqs.git /src/liboqs
actual="$(git -C /src/liboqs rev-parse HEAD)"
if [ "$actual" != "$LIBOQS_COMMIT" ]; then
    echo "liboqs commit mismatch: tag $LIBOQS_TAG is $actual, expected $LIBOQS_COMMIT" >&2
    exit 1
fi

# The 25 post-quantum algorithms of the study (tokens verified against
# .CMake/alg_support.cmake at tag 0.16.0). The minimal-build filter also enables
# each algorithm's AVX2 / aarch64 variants.
algs="KEM_ml_kem_512;KEM_ml_kem_768;KEM_ml_kem_1024"
algs="$algs;KEM_hqc_1;KEM_hqc_3;KEM_hqc_5"
algs="$algs;SIG_ml_dsa_44;SIG_ml_dsa_65;SIG_ml_dsa_87"
algs="$algs;SIG_falcon_512;SIG_falcon_1024;SIG_falcon_padded_512;SIG_falcon_padded_1024"
for h in sha2 shake; do
    for l in 128 192 256; do
        for v in s f; do
            algs="$algs;SIG_slh_dsa_pure_${h}_${l}${v}"
        done
    done
done

arch="$(uname -m)"
variant_flags=()
case "$VARIANT" in
    b1-prod)
        variant_flags=(-DOQS_DIST_BUILD=ON)
        ;;
    b2-portable)
        variant_flags=(-DOQS_DIST_BUILD=OFF -DOQS_OPT_TARGET=generic)
        if [ "$arch" = "aarch64" ]; then
            variant_flags+=(-DOQS_USE_ARM_NEON_INSTRUCTIONS=OFF -DOQS_USE_ARM_AES_INSTRUCTIONS=OFF
                            -DOQS_USE_ARM_SHA2_INSTRUCTIONS=OFF -DOQS_USE_ARM_SHA3_INSTRUCTIONS=OFF)
        fi
        ;;
    b3-native)
        variant_flags=(-DOQS_DIST_BUILD=OFF -DOQS_OPT_TARGET=auto)
        ;;
    b4-noossl)
        variant_flags=(-DOQS_DIST_BUILD=ON -DOQS_USE_OPENSSL=OFF)
        ;;
    *)
        echo "unknown VARIANT '$VARIANT' (expected b1-prod, b2-portable, b3-native or b4-noossl)" >&2
        exit 1
        ;;
esac

cmake -S /src/liboqs -B /src/liboqs/build -GNinja \
    -DCMAKE_BUILD_TYPE=Release \
    -DBUILD_SHARED_LIBS=ON \
    -DOQS_BUILD_ONLY_LIB=OFF \
    "-DOQS_MINIMAL_BUILD=${algs}" \
    "${variant_flags[@]}" > "$info/oqs-cmake.txt" 2>&1 || { cat "$info/oqs-cmake.txt" >&2; exit 1; }
ninja -C /src/liboqs/build > "$info/oqs-ninja.txt" 2>&1 || { tail -n 60 "$info/oqs-ninja.txt" >&2; exit 1; }
ninja -C /src/liboqs/build install > /dev/null
ldconfig

cp /src/liboqs/build/CMakeCache.txt "$info/oqs-CMakeCache.txt"
cp /usr/local/include/oqs/oqsconfig.h "$info/oqsconfig.h"

# Every one of the 25 algorithms must be compiled in.
missing=0
IFS=';' read -r -a toks <<< "$algs"
for t in "${toks[@]}"; do
    if ! grep -q "^#define OQS_ENABLE_${t} 1" "$info/oqsconfig.h"; then
        echo "algorithm not enabled in this build: ${t}" >&2
        missing=1
    fi
done
[ "$missing" -eq 0 ]

{
    echo "variant=${VARIANT}"
    echo "liboqs_tag=${LIBOQS_TAG}"
    echo "liboqs_commit=${actual}"
    echo "arch=${arch}"
    echo "minimal_build=${algs}"
    echo "variant_flags=${variant_flags[*]}"
    echo "gcc=$(gcc -dumpfullversion)"
    echo "cmake=$(cmake --version | head -n 1)"
} > "$info/oqs-build.txt"

if [ "$VARIANT" = "b3-native" ]; then
    # Record what "native" resolved to on this machine.
    if [ "$arch" = "aarch64" ]; then
        gcc -Q --help=target -mcpu=native > "$info/gcc-native-target.txt" 2>&1 || true
    else
        gcc -Q --help=target -march=native > "$info/gcc-native-target.txt" 2>&1 || true
    fi
fi
echo "liboqs ${LIBOQS_TAG} (${VARIANT}) built and installed"
