#!/bin/bash
# Runs several plans on the VM one after another, unattended: each is
# scripts/run-linux.sh start --plan P --machine vm-x86v2 --foreground. It stops at the
# first plan that does not finish (a refused preflight, a failure, or a batch
# that stopped and can be resumed: resume it with run-linux.sh, then run the
# series again for the rest).
#
#   scripts/run-linux-series.sh PLAN[@VARIANT] [PLAN[@VARIANT] ...]
#   scripts/run-linux-series.sh trial-e2-noslh trial-e11-curated
#   scripts/run-linux-series.sh e1 e10 e4@b4-noossl
#   scripts/run-linux-series.sh status        # is a series running, and where is it
#   scripts/run-linux-series.sh stop          # stop it after the current block (graceful)
#
# It detaches from the terminal (setsid, nohup) and logs to
# ~/.local/state/qchain-kpi/series-<stamp>.log (the newest is series.log). A lock refuses a
# second series while one runs (two launchers at once would measure on top of
# each other). The log lines to look for:
#   === <time> <plan>          a plan starts
#   run-linux.sh: done: ...    a plan finished (checksums verified, privacy scan clean)
#   preflight refused ...      the machine was not quiet at that moment; it waits 5 minutes and tries again (6 times)
#   FAILED: <plan> (exit N)    the series stopped here
#   === series finished       every plan finished
set -euo pipefail

repo="$(cd "$(dirname "$0")/.." && pwd -P)"
state="${XDG_STATE_HOME:-$HOME/.local/state}/qchain-kpi"
lock="$state/series.lock"
gap="${KPI_SERIES_GAP:-65}" # run ids have minute resolution: the next plan starts in a later minute (the tests use 1)
retries="${KPI_SERIES_RETRIES:-6}"      # a plan whose preflight is refused (exit 3) is tried again this many times,
retry_wait="${KPI_SERIES_RETRY_WAIT:-300}" # after this many seconds: a short burst of system work must not end a days-long series

die() { echo "run-linux-series.sh: $*" >&2; exit 1; }

running_pid() { # prints the pid of a running series, or nothing
  local p
  [ -f "$lock/pid" ] || return 0
  p="$(cat "$lock/pid")"
  if [ -n "$p" ] && kill -0 "$p" 2>/dev/null; then printf '%s\n' "$p"; fi
}

case "${1:-}" in
  "") die "usage: run-linux-series.sh PLAN[@VARIANT] ... | status | stop" ;;
  status)
    pid="$(running_pid)"
    if [ -n "$pid" ]; then
      echo "a series is running (process $pid); log: $state/series.log"
      grep -E '^===|done:|FAILED' "$state/series.log" 2>/dev/null | cut -c1-150 | tail -n 8 || true
    else
      echo "no series is running"
      if [ -f "$state/series.log" ]; then grep -E '^===|done:|FAILED' "$state/series.log" | cut -c1-150 | tail -n 4 || true; fi
    fi
    exit 0
    ;;
  stop)
    pid="$(running_pid)"
    [ -n "$pid" ] || die "no series is running"
    : > "$state/series.stop"
    # Ask the batch running now to stop after its current block; the series then ends.
    rid="$(docker ps --filter label=org.qchain.kpi=1 --format '{{.Names}}' | sed 's/^kpi-run-//' | head -n 1)"
    if [ -n "$rid" ]; then "$repo/scripts/run-linux.sh" stop "$rid"; else echo "no batch is running at this moment; the series ends before its next plan"; fi
    exit 0
    ;;
esac

# Every argument is a plan, optionally with the liboqs build: PLAN@VARIANT.
for spec in "$@"; do
  [[ "$spec" =~ ^[a-z0-9-]+(@[a-z0-9-]+)?$ ]] || die "bad plan name: $spec"
  plan="${spec%%@*}"
  [ -f "$repo/plans/vm/$plan.json" ] || die "no plan plans/vm/$plan.json"
done

mkdir -p "$state"
if [ "${KPI_SERIES_CHILD:-0}" != 1 ]; then
  # The parent: refuse a second series, then run the child detached.
  pid="$(running_pid)"
  [ -z "$pid" ] || die "a series is already running (process $pid; log $state/series.log); not starting another"
  stamp="$(date -u +%Y%m%d-%H%M%S)"
  log="$state/series-$stamp.log"
  KPI_SERIES_CHILD=1 setsid nohup "$0" "$@" > "$log" 2>&1 < /dev/null &
  child=$!
  ln -sf "series-$stamp.log" "$state/series.log"
  echo "series started (process $child): $*"
  echo "log: $log"
  echo "watch: scripts/run-linux-series.sh status"
  exit 0
fi

# The child. The lock is a directory (mkdir is atomic); a stale one (its process gone) is replaced.
if ! mkdir "$lock" 2>/dev/null; then
  sleep 1 # the winner of a near-simultaneous start may not have written its pid yet
  if [ -n "$(running_pid)" ]; then die "a series is already running"; fi
  rm -rf "$lock"
  mkdir "$lock"
fi
echo "$$" > "$lock/pid"
rm -f "$state/series.stop"
trap 'rm -rf "$lock"' EXIT
trap 'exit 143' TERM
trap 'exit 130' INT

cd "$repo"
for spec in "$@"; do
  if [ -f "$state/series.stop" ]; then
    echo "STOPPED on request before $spec"
    exit 5
  fi
  plan="${spec%%@*}"
  args=(start --plan "$plan" --machine vm-x86v2 --foreground)
  if [[ "$spec" == *@* ]]; then args+=(--variant "${spec#*@}"); fi
  echo "=== $(date +%H:%M:%S) $spec"
  tried=0
  while :; do
    rc=0
    scripts/run-linux.sh "${args[@]}" || rc=$?
    [ "$rc" = 3 ] && [ "$tried" -lt "$retries" ] || break
    tried=$((tried + 1))
    echo "preflight refused for $spec; trying again in ${retry_wait}s ($tried of $retries)"
    sleep "$retry_wait"
    if [ -f "$state/series.stop" ]; then echo "STOPPED on request before $spec"; exit 5; fi
  done
  if [ "$rc" != 0 ]; then
    echo "FAILED: $spec (exit $rc)"
    exit "$rc"
  fi
  sleep "$gap"
done
echo "=== series finished $(date +%H:%M:%S)"
