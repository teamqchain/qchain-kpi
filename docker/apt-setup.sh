#!/bin/bash
# Installs Debian packages from snapshot.debian.org at a fixed date, so every
# machine and every rebuild gets byte-identical package versions.
# If the snapshot service is unreachable, it falls back to the live Debian
# mirrors and records that fact (the manifest flags such builds).
#
# Usage: apt-setup.sh <stage-name> <snapshot YYYYMMDDTHHMMSSZ> <package>...
set -euo pipefail

stage="$1"; snap="$2"; shift 2
info=/opt/kpi/build-info
mkdir -p "$info"

backup=/opt/kpi/apt-backup
mkdir -p "$backup"
[ -f /etc/apt/sources.list.d/debian.sources ] && mv /etc/apt/sources.list.d/debian.sources "$backup/"
[ -f /etc/apt/sources.list ] && cp /etc/apt/sources.list "$backup/sources.list" || true

cat > /etc/apt/sources.list <<EOF
deb [check-valid-until=no] http://snapshot.debian.org/archive/debian/${snap} bookworm main
deb [check-valid-until=no] http://snapshot.debian.org/archive/debian/${snap} bookworm-updates main
deb [check-valid-until=no] http://snapshot.debian.org/archive/debian-security/${snap} bookworm-security main
EOF

export DEBIAN_FRONTEND=noninteractive
if apt-get -o Acquire::Retries=3 update && apt-get install -y --no-install-recommends "$@"; then
    echo "snapshot.debian.org ${snap}" > "$info/apt-source-${stage}.txt"
else
    echo "WARNING: snapshot.debian.org ${snap} unavailable; falling back to live Debian mirrors" >&2
    rm -f /etc/apt/sources.list
    [ -f "$backup/debian.sources" ] && cp "$backup/debian.sources" /etc/apt/sources.list.d/debian.sources
    [ -f "$backup/sources.list" ] && cp "$backup/sources.list" /etc/apt/sources.list
    apt-get -o Acquire::Retries=3 update
    apt-get install -y --no-install-recommends "$@"
    echo "LIVE-MIRRORS (snapshot ${snap} unavailable)" > "$info/apt-source-${stage}.txt"
fi
rm -rf /var/lib/apt/lists/*
dpkg-query -W -f='${Package}\t${Version}\t${Architecture}\n' | sort > "$info/dpkg-${stage}.tsv"
