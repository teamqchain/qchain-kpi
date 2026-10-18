#!/bin/bash
# Native macOS toolchain for E5 (decisions 19 Oct: Q4 = A, nothing is installed
# system-wide; Q5 = A, Apple clang only). Everything is built under one folder,
# which is deleted after the study (revert checklist):
#
#   /Users/Shared/qchain-kpi-native/
#     venv/        Python venv from /usr/bin/python3 with pip wheels: cmake, ninja
#                  and liboqs's test tools (the versions Debian bookworm ships,
#                  i.e. the container's)
#     src/         downloaded sources, the NIST ACVP vector files, build folders
#     openssl/     OpenSSL 3.0.22, the container's version, from the official
#                  tarball (SHA-256 pinned) with Debian's configure options
#     liboqs/      liboqs 0.16.0 (pinned commit), B1 flags as docker/oqs-build.sh
#     bin/pkg-config   answers cgo's pkg-config queries (the Mac has no pkg-config)
#     env.sh       the environment for building the harness natively
#     build-info/  the toolchain's build records, copied into every native run
#     harness/, runs/, prep/   created by scripts/native-mac.sh
#
# The folder is outside the home folder, so the build records contain no user
# name or home path (the per-run privacy scan), and has no spaces in its path
# (OpenSSL's build and cgo's flags cannot handle them; the repository's path
# has spaces).
#
# Every step is skipped when it has finished before (a stamp file); a step
# that fails leaves no stamp. liboqs must pass the same correctness gate as in
# the image (docker/oqs-gate.sh), and OpenSSL its own test suite, or the setup
# fails. Needs the internet; changes nothing outside the folder.
#
#   scripts/native-setup.sh
set -euo pipefail

N=/Users/Shared/qchain-kpi-native
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
OPENSSL_VERSION=3.0.22
# Published by the OpenSSL project on github.com and openssl.org (identical, checked 19 Oct).
OPENSSL_SHA256=67ebca7e50d17383028045486653492195b83db95f8558709701bb47b5c1ef81
# Debian bookworm's options for its arm64 libssl3 (debian/rules of 3.0.20-1~deb12u2;
# enable-ec_nistp_64_gcc_128 is amd64 only); the container runs 3.0.22-1~deb12u1.
OPENSSL_OPTS=(shared no-idea no-mdc2 no-rc5 no-zlib no-ssl3 no-ssl3-method enable-rfc3779 enable-cms no-capieng no-rdrand)
LIBOQS_TAG=0.16.0
LIBOQS_COMMIT=5a1a854b0dc9f2141bdc771c555ee60c37950183
# Debian bookworm's versions of the image's build and test tools.
PIP_PINS=(cmake==3.25.2 ninja==1.11.1 pytest==7.2.1 pytest-xdist==3.1.0 requests==2.28.1 PyYAML==6.0.1)
GO=/opt/homebrew/opt/go@1.24/bin/go
CC=/usr/bin/clang
jobs="$(sysctl -n hw.ncpu)"

die() { echo "native-setup.sh: $*" >&2; exit 1; }
note() { echo "native-setup.sh: $*" >&2; }
done_step() { [ -e "$N/.done-$1" ]; }
mark() { : > "$N/.done-$1"; }

if [ "$(uname -s)" != Darwin ] || [ "$(uname -m)" != arm64 ]; then die "this setup is for the M1 Mac only"; fi
"$CC" --version >/dev/null 2>&1 || die "Apple clang not found (xcode-select --install)"
[ -x /usr/bin/python3 ] || die "/usr/bin/python3 not found"
[ "$("$GO" env GOVERSION)" = go1.24.13 ] || die "Go 1.24.13 not found at $GO"
mkdir -p "$N/src" "$N/bin" "$N/build-info"
info="$N/build-info"

# ------------------------------------------------------------------- venv
if ! done_step venv; then
  note "creating the Python venv (cmake, ninja, liboqs's test tools)"
  rm -rf "$N/venv"
  /usr/bin/python3 -m venv "$N/venv"
  "$N/venv/bin/python" -m pip install --quiet --disable-pip-version-check --only-binary :all: "${PIP_PINS[@]}"
  "$N/venv/bin/python" -m pip freeze --disable-pip-version-check > "$info/pip-freeze.txt"
  mark venv
fi
export PATH="$N/venv/bin:$PATH"

# ---------------------------------------------------------------- OpenSSL
if ! done_step openssl; then
  note "building OpenSSL $OPENSSL_VERSION (with its test suite; several minutes)"
  tgz="$N/src/openssl-$OPENSSL_VERSION.tar.gz"
  curl -sSfL --retry 3 -o "$tgz" "https://github.com/openssl/openssl/releases/download/openssl-$OPENSSL_VERSION/openssl-$OPENSSL_VERSION.tar.gz"
  echo "$OPENSSL_SHA256  $tgz" | shasum -a 256 -c - >/dev/null || die "OpenSSL tarball checksum mismatch"
  rm -rf "$N/src/openssl-$OPENSSL_VERSION" "$N/openssl"
  tar -xzf "$tgz" -C "$N/src"
  (
    cd "$N/src/openssl-$OPENSSL_VERSION"
    ./Configure darwin64-arm64-cc "${OPENSSL_OPTS[@]}" --prefix="$N/openssl" --openssldir="$N/openssl/ssl" --libdir=lib CC="$CC" > "$info/openssl-configure.txt" 2>&1
    make -j"$jobs" > "$info/openssl-make.txt" 2>&1 || { tail -n 40 "$info/openssl-make.txt" >&2; exit 1; }
    make test HARNESS_JOBS="$jobs" > "$info/openssl-test.txt" 2>&1 || { tail -n 60 "$info/openssl-test.txt" >&2; die "OpenSSL's test suite failed"; }
    make install_sw > /dev/null
  )
  # Only the summary of the test run is kept as a record.
  tail -n 5 "$info/openssl-test.txt" > "$info/openssl-test-summary.txt"
  rm -f "$info/openssl-make.txt" "$info/openssl-test.txt"
  {
    echo "openssl_version=$("$N/openssl/bin/openssl" version)"
    echo "openssl_tarball_sha256=$OPENSSL_SHA256"
    echo "openssl_configure=darwin64-arm64-cc ${OPENSSL_OPTS[*]}"
    echo "openssl_cflags=$("$N/openssl/bin/openssl" version -f | sed 's/^compiler: //')"
  } > "$info/openssl-build.txt"
  otool -D "$N/openssl/lib/libcrypto.3.dylib" | grep -q "^$N/openssl/lib/libcrypto.3.dylib$" || die "libcrypto's install name is not absolute"
  mark openssl
fi

# ----------------------------------------------------------------- liboqs
# The same 25 algorithms and B1 flags as docker/oqs-build.sh.
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
variant_flags=(-DOQS_DIST_BUILD=ON)
src="$N/src/liboqs"

if ! done_step liboqs; then
  note "building liboqs $LIBOQS_TAG (B1 flags, Apple clang)"
  rm -rf "$src" "$N/liboqs"
  git clone --quiet -c advice.detachedHead=false --depth 1 --branch "$LIBOQS_TAG" https://github.com/open-quantum-safe/liboqs.git "$src"
  actual="$(git -C "$src" rev-parse HEAD)"
  [ "$actual" = "$LIBOQS_COMMIT" ] || die "liboqs commit mismatch: tag $LIBOQS_TAG is $actual, expected $LIBOQS_COMMIT"
  cmake -S "$src" -B "$src/build" -GNinja \
    -DCMAKE_BUILD_TYPE=Release \
    -DCMAKE_C_COMPILER="$CC" \
    -DCMAKE_INSTALL_PREFIX="$N/liboqs" \
    -DCMAKE_INSTALL_NAME_DIR="$N/liboqs/lib" \
    -DOPENSSL_ROOT_DIR="$N/openssl" \
    -DBUILD_SHARED_LIBS=ON \
    -DOQS_BUILD_ONLY_LIB=OFF \
    "-DOQS_MINIMAL_BUILD=${algs}" \
    "${variant_flags[@]}" > "$info/oqs-cmake.txt" 2>&1 || { cat "$info/oqs-cmake.txt" >&2; exit 1; }
  ninja -C "$src/build" > "$info/oqs-ninja.txt" 2>&1 || { tail -n 60 "$info/oqs-ninja.txt" >&2; exit 1; }
  ninja -C "$src/build" install > /dev/null
  cp "$src/build/CMakeCache.txt" "$info/oqs-CMakeCache.txt"
  cp "$N/liboqs/include/oqs/oqsconfig.h" "$info/oqsconfig.h"
  missing=0
  IFS=';' read -r -a toks <<< "$algs"
  for t in "${toks[@]}"; do
    if ! grep -q "^#define OQS_ENABLE_${t} 1" "$info/oqsconfig.h"; then
      echo "algorithm not enabled in this build: ${t}" >&2
      missing=1
    fi
  done
  [ "$missing" -eq 0 ] || die "not all 25 algorithms are enabled"
  lib="$(find "$N/liboqs/lib" -name 'liboqs.*.dylib' -type f | head -n 1)"
  otool -D "$lib" | tail -n 1 | grep -q "^$N/liboqs/lib/" || die "liboqs's install name is not absolute"
  otool -L "$lib" | grep -q "$N/openssl/lib/libcrypto.3.dylib" || die "liboqs is not linked to the OpenSSL built here"
  {
    echo "variant=b1-prod"
    echo "platform=native-macos"
    echo "liboqs_tag=${LIBOQS_TAG}"
    echo "liboqs_commit=${actual}"
    echo "arch=$(uname -m)"
    echo "minimal_build=${algs}"
    echo "variant_flags=${variant_flags[*]}"
    echo "cc=$("$CC" --version | head -n 1)"
    echo "cmake=$(cmake --version | head -n 1)"
    echo "openssl=$("$N/openssl/bin/openssl" version)"
  } > "$info/oqs-build.txt"
  mark liboqs
fi

# ------------------------------------------------- liboqs correctness gate
# As docker/oqs-gate.sh: the ACVP vector files checked against the pinned
# SHA-256s, then liboqs's own pytest suites; any failure fails the setup.
if ! done_step gate; then
  note "running liboqs's correctness gate (a few minutes)"
  acvp="$N/src/acvp"
  root="https://raw.githubusercontent.com/usnistgov/ACVP-Server/refs/tags/v1.1.0.41/gen-val/json-files"
  mkdir -p "$acvp"
  : > "$info/acvp-sha256-check.txt"
  while read -r sum path; do
    case "$sum" in ''|'#'*) continue ;; esac
    mkdir -p "$acvp/$(dirname "$path")"
    curl -sSfL --retry 3 -o "$acvp/$path" "$root/$path"
    echo "$sum  $acvp/$path" | shasum -a 256 -c - >> "$info/acvp-sha256-check.txt"
  done < "$repo/docker/acvp-sha256.txt"
  # The image's conftest plus SHA-512/224 and /256 from the OpenSSL built here
  # (macOS's system Python links LibreSSL, which has neither; see the file).
  cat "$repo/docker/acvp_conftest.py" "$repo/scripts/native-gate-hashlib.py" > "$src/tests/conftest.py"
  echo "SHA-512/224 and SHA-512/256 for the HashML-DSA ACVP vectors: computed by the OpenSSL built here (scripts/native-gate-hashlib.py); /usr/bin/python3 links $(/usr/bin/python3 -c 'import ssl; print(ssl.OPENSSL_VERSION)')" > "$info/oqs-gate-notes.txt"
  set +e
  ( cd "$src" && OQS_BUILD_DIR="$src/build" KPI_ACVP_DIR="$acvp" SKIP_TESTS=examples KPI_LIBCRYPTO="$N/openssl/lib/libcrypto.3.dylib" \
      "$N/venv/bin/python" -m pytest -q -p no:cacheprovider --numprocesses=auto \
      tests/test_cmdline.py tests/test_kat.py tests/test_acvp_vectors.py ) > "$info/oqs-gate.txt" 2>&1
  status=$?
  set -e
  tail -n 5 "$info/oqs-gate.txt"
  if [ "$status" -ne 0 ]; then
    tail -n 80 "$info/oqs-gate.txt" >&2
    die "liboqs correctness gate FAILED (pytest exit $status)"
  fi
  mark gate
fi

# ------------------------------------------------- pkg-config and env.sh
# cgo asks "pkg-config --cflags -- PKG..." and "pkg-config --libs -- PKG...".
cat > "$N/bin/pkg-config" <<EOF
#!/bin/bash
# Stand-in for pkg-config, written by scripts/native-setup.sh: answers cgo's
# queries for the two packages the harness uses, from the native toolchain.
mode=""
pkgs=()
for a in "\$@"; do
  case "\$a" in
    --cflags) mode=cflags ;;
    --libs) mode=libs ;;
    --) ;;
    -*) echo "pkg-config stand-in: unsupported option \$a" >&2; exit 1 ;;
    *) pkgs+=("\$a") ;;
  esac
done
[ -n "\$mode" ] || { echo "pkg-config stand-in: --cflags or --libs required" >&2; exit 1; }
out=()
for p in "\${pkgs[@]}"; do
  case "\$p:\$mode" in
    liboqs-go:cflags) out+=("-I$N/liboqs/include") ;;
    liboqs-go:libs) out+=("-L$N/liboqs/lib" "-loqs") ;;
    libcrypto:cflags) out+=("-I$N/openssl/include") ;;
    libcrypto:libs) out+=("-L$N/openssl/lib" "-lcrypto") ;;
    *) echo "pkg-config stand-in: unknown package \$p" >&2; exit 1 ;;
  esac
done
echo "\${out[*]}"
EOF
chmod 755 "$N/bin/pkg-config"
cat > "$N/env.sh" <<EOF
# Written by scripts/native-setup.sh: source it to build or test the harness natively.
export PKG_CONFIG=$N/bin/pkg-config
export CC=$CC
export CGO_ENABLED=1
export GOTOOLCHAIN=local
export GOFLAGS=-trimpath
EOF
"$CC" --version | head -n 1 > "$info/cc-version.txt"

# The records are copied into every run: they must not identify the machine.
for pat in "$HOME" "$(id -un)" "$(hostname -s)"; do
  if [ "${#pat}" -ge 4 ] && grep -rqiF -- "$pat" "$info"; then
    die "the build records contain identifying text ($(grep -rliF -- "$pat" "$info" | sed "s|^$info/||" | tr '\n' ' '))"
  fi
done
note "native toolchain ready in $N"
note "liboqs $(sed -n 's/^liboqs_tag=//p' "$info/oqs-build.txt"), $("$N/openssl/bin/openssl" version), $(cat "$info/cc-version.txt")"
