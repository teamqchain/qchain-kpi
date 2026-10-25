#!/bin/bash
# Tests scripts/run-linux.sh and scripts/monitor-linux.sh without the VM. They
# run as an ordinary user in a throwaway Ubuntu container, against a fake docker
# command that plays the containers, the volume and the batch
# (testdata/launcher-linux/): the refusals, a foreground start, a detached
# start in tmux, stop and resume, extend, and a privacy-scan failure. Prints
# PASS or FAIL per check; takes about 8 minutes. The container installs
# python3, git, tmux and iproute2 from apt and is deleted afterwards.
#
#   scripts/test-run-linux.sh [IMAGE]   (default hyperledger/fabric-tools:2.5.15, an Ubuntu 22.04 image)
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
image="${1:-hyperledger/fabric-tools:2.5.15}"
stage="$(mktemp -d "${TMPDIR:-/tmp}/kpi-lt.XXXXXX")"
trap 'rm -rf "$stage"' EXIT
chmod 755 "$stage" # tar carries this mode to the container's /t, which the test user must enter
mkdir -p "$stage/repo"
# The working tree as it is (committed or not), without results and scratch.
(cd "$repo" && git ls-files -co --exclude-standard -z | grep -z -v -e '^results/' -e '^scratch/' |
  COPYFILE_DISABLE=1 tar --null -T - -cf -) | tar -xf - -C "$stage/repo"
cp -R "$repo/testdata/launcher-linux" "$stage/lt"
COPYFILE_DISABLE=1 tar -C "$stage" -cf - . |
  docker run --rm -i --entrypoint bash "$image" -c 'mkdir /t && tar -xf - -C /t 2>/dev/null && bash /t/lt/setup.sh'
