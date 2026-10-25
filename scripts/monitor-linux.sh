#!/bin/bash
# Host monitor for the VM (Linux; coding plan §8). Started and stopped by
# scripts/run-linux.sh, pinned to vCPU 0 at the lowest priority (the benchmark
# runs on vCPU 1).
#
#   monitor-linux.sh <work-dir> <period-s> <rows-per-chunk> <launcher-pid>
#
# Every sample becomes one CSV row (format in internal/quality/host.go):
#
#   utc_ns,other_pct,throttle,on_ac,alert,detail
#
#   other_pct  -1: the container runs on the VM's own kernel, so the batch's
#              /proc/stat reading already covers every CPU of the machine
#              (other CPU and steal, per block); a second reading here would
#              count the same time twice
#   throttle   -1: a VM reports no thermal or power limit
#   on_ac      -1: not applicable
#   alert      1 while something that must not run during the study runs: a
#              container that is not ours (QChain's Fabric network, qchain-api,
#              qchain-web-gateway or any other), MySQL, IPFS or the GitHub
#              Actions runner. Checked every <rows-per-chunk> samples.
#   detail     the machine's busy and steal % over the sample, the 1-minute
#              load and what the last check found (counts and program names)
#
# Rows go into chunk files <work-dir>/host-NNNNNN.csv; a finished chunk gets
# a .ready marker, and the launcher delivers it into the run volume. All rows are
# also kept in <work-dir>/host-all.csv. No user names, host names or paths are
# recorded. The monitor ends when the launcher does.
set -euo pipefail

dir="$1"
period="$2"
rows="$3"
launcher="$4"
header="utc_ns,other_pct,throttle,on_ac,alert,detail"
mkdir -p "$dir"
[ -s "$dir/host-all.csv" ] || echo "$header" > "$dir/host-all.csv"

seq=0 nrow=0 chunk=""
open_chunk() {
  seq=$((seq + 1))
  chunk="$(printf '%s/host-%06d' "$dir" "$seq")"
  echo "$header" > "$chunk.csv"
  nrow=0
}
close_chunk() {
  [ -n "$chunk" ] || return 0
  : > "$chunk.ready"
  chunk=""
}

# Unix time in ns. EPOCHREALTIME (bash 5) may use the locale's decimal mark.
now_ns() {
  if [ -n "${EPOCHREALTIME:-}" ]; then printf '%s000\n' "${EPOCHREALTIME//[!0-9]/}"; else date +%s%N; fi
}

# read_cpu: the machine's busy, total and steal jiffies (/proc/stat, first line).
read_cpu() {
  local f
  read -r -a f < /proc/stat
  busy=$((f[1] + f[2] + f[3] + f[6] + f[7]))
  total=$((f[1] + f[2] + f[3] + f[4] + f[5] + f[6] + f[7] + f[8]))
  steal=${f[8]}
}

pct() { # part whole -> percent with one decimal
  local v=0
  [ "$2" -gt 0 ] && v=$(((1000 * $1 + $2 / 2) / $2))
  printf '%d.%d' $((v / 10)) $((v % 10))
}

# check: sets alert (0/1) and found (what ran).
alert=0
found="not_checked"
check() {
  local n=0 out what=()
  if out="$(timeout 20 docker ps --format '{{.Label "org.qchain.kpi"}}' 2>/dev/null)"; then
    [ -z "$out" ] || n="$(grep -c -v -x 1 <<< "$out" || true)"
  else
    what+=("docker_unreachable")
  fi
  [ "$n" = 0 ] || what+=("containers_not_ours=$n")
  if pgrep -x mysqld >/dev/null; then what+=("mysqld"); fi
  if pgrep -x ipfs >/dev/null; then what+=("ipfs"); fi
  if pgrep -x 'Runner.Listener|Runner.Worker' >/dev/null; then what+=("github_runner"); fi
  if [ "${#what[@]}" -gt 0 ] && [ "${what[*]}" != "docker_unreachable" ]; then alert=1; else alert=0; fi
  found="none"
  [ "${#what[@]}" -eq 0 ] || found="$(IFS=+; printf '%s' "${what[*]}")"
}

trap 'close_chunk; exit 0' TERM INT HUP
read_cpu
b0=$busy t0=$total s0=$steal
samples=0
while kill -0 "$launcher" 2>/dev/null; do
  sleep "$period" &
  wait $! || true
  if [ $((samples % rows)) -eq 0 ]; then check; fi
  samples=$((samples + 1))
  read_cpu
  read -r load1 _ < /proc/loadavg
  row="$(printf '%s,-1,-1,-1,%d,vm_busy_pct=%s;steal_pct=%s;load1=%s;found=%s' "$(now_ns)" "$alert" \
    "$(pct $((busy - b0)) $((total - t0)))" "$(pct $((steal - s0)) $((total - t0)))" "$load1" "$found")"
  b0=$busy t0=$total s0=$steal
  [ -n "$chunk" ] || open_chunk
  echo "$row" >> "$chunk.csv"
  echo "$row" >> "$dir/host-all.csv"
  nrow=$((nrow + 1))
  [ "$nrow" -lt "$rows" ] || close_chunk
done
close_chunk
