#!/bin/bash
# QChain KPI launcher for native macOS runs (E5: the E1 cells run directly on
# macOS, outside Docker; decisions of 19 Oct).
#
#   scripts/native-mac.sh start --plan NAME [--seed N] [--cooling TEXT] [--force]
#   scripts/native-mac.sh resume RUN_ID [--cooling TEXT] [--force]
#   scripts/native-mac.sh extend RUN_ID --blocks ID[,ID...] [--cooling TEXT] [--force]
#   scripts/native-mac.sh stop RUN_ID        # graceful: stops after the current block
#   scripts/native-mac.sh build              # only builds and tests the native binaries
#   scripts/native-mac.sh status
#
# Needs the native toolchain of scripts/native-setup.sh. It does what run.sh
# does, without Docker:
#   - refuses to start on a machine that is not ready (the same host checks as
#     run.sh, plus: Colima must be stopped, so its VM does not compete);
#   - builds kpi-prim, kpi-batch and kpi-env from the clean harness commit
#     (git archive, so nothing uncommitted can enter), after go vet and go test
#     have passed natively (as the image's test stage), into
#     /Users/Shared/qchain-kpi-native/harness/<commit>/ with the plans, the
#     thresholds and the build records of that commit;
#   - runs the batch in /Users/Shared/qchain-kpi-native/runs/<run id> with
#     GOMAXPROCS=1 and TZ=Asia/Dubai (the container's settings); kpi-prim raises
#     itself to the user-interactive QoS class, so macOS keeps it on the
#     performance cores; nothing pins a CPU (macOS has no CPU pinning);
#   - the host monitor, the background-service pause and the Wi-Fi switch-off
#     are run.sh's own (loaded from run.sh between its "shared" markers), so
#     both launchers behave the same; host samples are delivered by copying;
#   - when the batch is done: verifies SHA256SUMS, runs the privacy scan, moves
#     the run into results/mac/, adds the INDEX row, and removes the run's work
#     folders. Pushing stays manual.
#
# The batch runs in its own process group and Ctrl-C is forwarded to it alone,
# as docker forwards it to the container: the batch then discards its current
# block and stops (resume later).
# shellcheck disable=SC2154,SC2034 # osdir, bg_early_log and bg_paused belong to run.sh's shared block, loaded below
set -euo pipefail

LAUNCHER="native-mac.sh 1"
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
N=/Users/Shared/qchain-kpi-native
PERIOD_S=5        # host monitor sampling period
ROWS_PER_CHUNK=12 # one chunk per minute
GO=/opt/homebrew/opt/go@1.24/bin/go
MACHINE=mac-m1

# run.sh's shared functions: die/note, JSON helpers, repo_commit, the host
# preflight, native_refs, the background-service pause, Wi-Fi, cleanup and
# its traps, the privacy scan and the INDEX row.
# (eval, not "source <(...)": macOS's bash 3.2 silently sources nothing from a
# process substitution.)
shared="$(awk '/^# >>> shared with scripts\/native-mac.sh/{f=1; next} /^# <<< shared with scripts\/native-mac.sh/{f=0} f' "$repo/scripts/run.sh")"
eval "$shared"
for fn in repo_commit preflight_host preflight_verdict free_space_check native_refs pause_bg resume_bg wifi_off cleanup persist_dir privacy_scan index_row json_get json_str; do
  declare -F "$fn" >/dev/null || { echo "native-mac.sh: $fn was not loaded from run.sh's shared blocks" >&2; exit 1; }
done
die() { echo "native-mac.sh: $*" >&2; exit 1; }
note() { echo "native-mac.sh: $*" >&2; }
runtime=native-macos

if [ ! -f "$N/env.sh" ] || [ ! -e "$N/.done-gate" ]; then die "the native toolchain is missing; run scripts/native-setup.sh first"; fi

# ------------------------------------------------------------------ build

# native_build COMMIT: sets hb (the harness folder of that commit), building
# and testing it first if needed.
native_build() {
  local commit="$1" tmp
  hb="$N/harness/$commit"
  [ -e "$hb/.done" ] && return 0
  note "building the native harness for $commit (go vet and go test run first)"
  rm -rf "$hb"
  mkdir -p "$hb/src" "$hb/bin"
  git -C "$repo" archive "$commit" | tar -x -C "$hb/src"
  cp -R "$N/build-info" "$hb/build-info"
  (
    # shellcheck source=/dev/null
    source "$N/env.sh"
    cd "$hb/src"
    "$GO" vet ./...
    if "$GO" test -count=1 ./... > "$hb/build-info/go-test.txt" 2>&1 &&
       "$GO" test -count=1 -race -run 'Concurrent|Run' ./internal/flow ./internal/conc >> "$hb/build-info/go-test.txt" 2>&1; then
      cat "$hb/build-info/go-test.txt" >&2
    else
      cat "$hb/build-info/go-test.txt" >&2
      echo "go test FAILED" >&2
      exit 1
    fi
    "$GO" build -ldflags "-X qchain-kpi/internal/buildinfo.HarnessCommit=$commit" -o "$hb/bin/" ./cmd/kpi-prim ./cmd/kpi-batch ./cmd/kpi-env
    "$GO" version > "$hb/build-info/go-version.txt"
  ) || { rm -rf "$hb"; die "the native build or its tests failed"; }
  cp -R "$hb/src/plans" "$hb/plans"
  cp -R "$hb/src/config" "$hb/config"
  rm -rf "$hb/src"
  # Like the image ID: changes whenever the binaries or the libraries they load change.
  tmp="$(shasum -a 256 "$hb"/bin/* "$N"/liboqs/lib/liboqs.*.dylib "$N/openssl/lib/libcrypto.3.dylib" | awk '{print $1}' | shasum -a 256 | awk '{print $1}')"
  printf 'native-sha256:%s\n' "$tmp" > "$hb/image-id.txt"
  : > "$hb/.done"
}

# ---------------------------------------------------------------- checks

preflight_native() { # kind
  local kind="$1" out
  problems=()
  out="$(colima status 2>&1 || true)"
  [[ "$out" == *"is not running"* ]] || problem "Colima is running; stop it for native runs (colima stop); run.sh starts it again"
  if pgrep -q -f '/qchain-kpi-native/harness/.*/kpi-batch'; then problem "another native batch is running"; fi
  free_space_check "$repo" "the repository"
  free_space_check "$N" "the native work folder"
  preflight_host
  preflight_verdict "$kind"
}

# ----------------------------------------------------------------- session

deliver_native() { # workdir out session
  local w="$1" out="$2" s="$3" r base name
  for r in "$w"/host-*.ready; do
    [ -e "$r" ] || continue
    base="${r%.ready}"
    name="host-s$s-${base##*/host-}"
    # The batch reads a chunk only once its .ok marker exists.
    if cp "$base.csv" "$out/monitor/$name.csv" 2>/dev/null && : > "$out/monitor/$name.ok"; then
      mv "$r" "$base.delivered" 2>/dev/null || true
    fi
  done
}

write_native_session() { # file p e cooling
  local f="$1" lpm therm batt power ncpu mem go_v
  ncpu="$(sysctl -n hw.logicalcpu)"
  mem=$(( $(sysctl -n hw.memsize) / 1073741824 ))
  go_v="$("$GO" version | awk '{print $3}')"
  lpm="$(pmset -g | awk '/lowpowermode/{print $2}')"
  therm="$(notifyutil -g com.apple.system.thermalpressurelevel | awk '{print $2}')"
  batt="$(pmset -g batt | grep -Eo '[0-9]+%' | head -n 1 | tr -d '%')"
  if [[ "$(pmset -g batt)" == *"AC Power"* ]]; then power=AC; else power=battery; fi
  {
    printf '{\n'
    # The batch and the monitor read the same clock.
    printf '  "host_offset_ns": 0,\n  "host_offset_unc_ns": 0,\n'
    printf '  "native_p_rate": %s,\n  "native_e_rate": %s,\n' "$2" "$3"
    printf '  "host_logical_cpus": %s,\n  "host_monitor_period_s": %s,\n' "$ncpu" "$PERIOD_S"
    printf '  "host": {\n'
    printf '    "runtime": %s,\n' "$(json_str "$runtime")"
    printf '    "host_os": %s,\n' "$(json_str "macOS $(sw_vers -productVersion) ($(sw_vers -buildVersion))")"
    printf '    "host_cpu": %s,\n' "$(json_str "$(sysctl -n machdep.cpu.brand_string)")"
    printf '    "host_mem_gb": %s,\n' "$(json_str "$mem")"
    printf '    "power": %s,\n    "battery_pct": %s,\n' "$(json_str "$power")" "$(json_str "$batt")"
    printf '    "cooling": %s,\n' "$(json_str "$4")"
    printf '    "low_power_mode": %s,\n    "thermal_at_start": %s,\n' "$(json_str "$lpm")" "$(json_str "$therm")"
    printf '    "launcher": %s,\n    "native_go": %s\n' "$(json_str "$LAUNCHER")" "$(json_str "$go_v")"
    printf '  }\n}\n'
  } > "$f"
  json_get "$f" host runtime >/dev/null || die "session.json is not valid JSON"
}

# run_native RID ARGS...: one start, resume or extension, from session.json to
# the batch's exit. ARGS are the batch's own arguments after "run". Sets
# session_code to the batch's exit code.
session_code=""
run_native() {
  local rid="$1" persist n work refs p e monpid delpid cafpid bpid out console
  shift
  out="$N/runs/$rid"
  console="$N/runs/$rid.console.txt"
  persist="$(persist_dir "$rid")"
  mkdir -p "$persist" "$out/monitor" "$N/prep"
  n=$(( $(find "$persist" -maxdepth 1 -name 'session-s*.json' | wc -l) + 1 ))
  work="$(mktemp -d "${TMPDIR:-/tmp}/kpi-mon.XXXXXX")"
  if [ "${#problems[@]}" -gt 0 ]; then printf '%s\n' "${problems[@]}" > "$persist/preflight-s$n.txt"; else : > "$persist/preflight-s$n.txt"; fi
  [ "$force" = 1 ] && echo "forced" >> "$persist/preflight-s$n.txt"

  : > "$work/empty.ok"
  "$repo/scripts/monitor-mac.sh" "$work" "$PERIOD_S" "$ROWS_PER_CHUNK" "$$" >/dev/null &
  monpid=$!
  cat "$bg_early_log" >> "$persist/bg-paused-s$n.txt" 2>/dev/null || true
  : > "$bg_early_log"
  pause_bg "$persist/bg-paused-s$n.txt"
  bg_paused=1
  wifi_off "$persist/wifi-s$n.txt"
  note "paused macOS background analysis services for this run (resumed automatically at the end)"
  ( while :; do deliver_native "$work" "$out" "$n"; pause_bg "$persist/bg-paused-s$n.txt"; sleep 10; done ) >/dev/null 2>&1 &
  delpid=$!
  disown "$delpid"

  refs="$(native_refs "$work" "$persist" "$n")"
  p="${refs% *}"; e="${refs#* }"
  note "native canary references: P-core ${p}/s, E-core ${e}/s"
  write_native_session "$persist/session-s$n.json" "$p" "$e" "$cooling"
  cp "$persist/session-s$n.json" "$out/session.json"

  caffeinate -i -s -w "$$" >/dev/null 2>&1 &
  cafpid=$!
  disown "$cafpid"

  note "starting batch $rid (session $n); stop gracefully with: scripts/native-mac.sh stop $rid"
  printf '\n=== session %s ===\n' "$n" >> "$console"
  # Own process group (setpgrp): Ctrl-C reaches only this script, which
  # forwards it to the batch. The batch's output goes to the console file and
  # is shown here by tail.
  GOMAXPROCS=1 TZ=Asia/Dubai /usr/bin/perl -e 'setpgrp(0, 0); exec @ARGV or die "exec: $!"' -- \
    "$hb/bin/kpi-batch" run --run-id "$rid" --machine "$MACHINE" --out "$out" --prep-dir "$N/prep/$rid" \
    --prim "$hb/bin/kpi-prim" --env "$hb/bin/kpi-env" --build-info "$hb/build-info" \
    --thresholds "$hb/config/thresholds.json" --image-id "$(cat "$hb/image-id.txt")" "$@" >> "$console" 2>&1 &
  bpid=$!
  tail -n 0 -f "$console" >&2 &
  local tailpid=$!
  trap 'note "interrupt: the batch discards its current block and stops; resume with: scripts/native-mac.sh resume '"$rid"'"; kill -INT '"$bpid"' 2>/dev/null || true' INT
  set +e
  wait "$bpid"
  session_code=$?
  while kill -0 "$bpid" 2>/dev/null; do
    wait "$bpid"
    session_code=$?
  done
  set -e
  trap - INT
  sleep 1
  kill "$tailpid" 2>/dev/null || true

  kill "$delpid" 2>/dev/null || true
  resume_bg
  bg_paused=0
  note "resumed the paused macOS background services"
  kill "$monpid" 2>/dev/null || true
  wait "$monpid" 2>/dev/null || true
  deliver_native "$work" "$out" "$n"
  cp "$work/host-all.csv" "$persist/host-monitor-s$n.csv" 2>/dev/null || true
  kill "$cafpid" 2>/dev/null || true
  rm -rf "$work"
}

# ----------------------------------------------------------------- finish

finish_native() { # rid code
  local rid="$1" code="$2" stage dest persist out
  persist="$(persist_dir "$rid")"
  out="$N/runs/$rid"
  case "$code" in
    0) ;;
    5)
      note "batch stopped and can be resumed: scripts/native-mac.sh resume $rid"
      return 0 ;;
    *)
      stage="$repo/scratch/failed/$rid"
      rm -rf "$stage"; mkdir -p "$stage"
      cp -R "$out/." "$stage/"
      cp "$N/runs/$rid.console.txt" "$stage/console.txt" 2>/dev/null || true
      note "batch ended with exit code $code; its files were copied to scratch/failed/$rid for inspection"
      note "the run folder $out was kept"
      return 1 ;;
  esac
  stage="$repo/scratch/staging/$rid"
  dest="$repo/results/$osdir/$rid"
  if [ "$extending" != 1 ]; then [ ! -e "$dest" ] || die "$dest already exists"; fi
  rm -rf "$stage"; mkdir -p "$stage"
  cp -R "$out/." "$stage/"
  cp "$N/runs/$rid.console.txt" "$stage/console.txt"
  mkdir -p "$stage/launcher"
  cp "$persist"/* "$stage/launcher/"
  ( cd "$stage" && shasum -a 256 -c --quiet SHA256SUMS ) || die "SHA256SUMS verification failed in scratch/staging/$rid (run folder kept)"
  privacy_scan "$stage" || die "privacy scan failed: nothing was moved into results/ (scratch/staging/$rid; run folder kept)"
  mkdir -p "$(dirname "$dest")"
  if [ "$extending" = 1 ]; then
    # The original folder is kept in scratch/backup/ (made before the extension) and in git.
    rm -rf "$dest"
    grep -v -F "| $rid |" "$repo/results/$osdir/INDEX.md" > "$repo/results/$osdir/INDEX.md.new" || true
    mv "$repo/results/$osdir/INDEX.md.new" "$repo/results/$osdir/INDEX.md"
  fi
  mv "$stage" "$dest"
  index_row "$dest" "$rid"
  rm -rf "$out" "$N/runs/$rid.console.txt" "$N/runs/$rid.harness" "$N/prep/$rid" "$persist"
  note "done: results/$osdir/$rid (checksums verified, privacy scan clean); native work folders removed"
  note "to commit: git -C \"$repo\" add \"results/$osdir/$rid\" \"results/$osdir/INDEX.md\" && git -C \"$repo\" commit -m \"results: $rid\""
}

# ------------------------------------------------------------------- main

cmd="${1:-}"
[ -n "$cmd" ] || die "usage: native-mac.sh start|resume|extend|stop|build|status ..."
shift
force=0
cooling="unstated"
extending=0

case "$cmd" in
  build)
    commit="$(repo_commit)"
    native_build "$commit"
    note "native harness ready: $hb"
    ;;
  start)
    plan="" seed=""
    while [ $# -gt 0 ]; do
      case "$1" in
        --plan) plan="$2"; shift 2 ;;
        --seed) seed="$2"; shift 2 ;;
        --cooling) cooling="$2"; shift 2 ;;
        --force) force=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    [ -n "$plan" ] || die "start needs --plan"
    commit="$(repo_commit)"
    native_build "$commit"
    planfile="$hb/plans/$plan.json"
    [ -f "$planfile" ] || die "no plan plans/$plan.json"
    kind="$(json_get "$planfile" kind)"
    variant="$(sed -n 's/^variant=//p' "$hb/build-info/oqs-build.txt")"
    pause_early
    preflight_native "$kind"
    [ -n "$seed" ] || seed="$(od -An -N8 -tu8 /dev/urandom | tr -d ' \n')"
    rid="$(date -u +%Y%m%d-%H%M)_${MACHINE}_${runtime}_${variant}"
    [ ! -e "$repo/results/$osdir/$rid" ] || die "results/$osdir/$rid already exists"
    [ ! -e "$N/runs/$rid" ] || die "$N/runs/$rid already exists"
    mkdir -p "$N/runs"
    printf '%s\n' "$commit" > "$N/runs/$rid.harness"
    note "run $rid: plan $plan ($kind), seed $seed, native harness $commit"
    run_native "$rid" --plan "$planfile" --seed "$seed"
    finish_native "$rid" "$session_code"
    ;;
  resume)
    rid="${1:-}"
    [ -n "$rid" ] || die "resume needs a run id"
    shift
    while [ $# -gt 0 ]; do
      case "$1" in
        --cooling) cooling="$2"; shift 2 ;;
        --force) force=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    if [ ! -d "$N/runs/$rid" ] || [ ! -f "$N/runs/$rid.harness" ]; then die "no native run $rid"; fi
    commit="$(cat "$N/runs/$rid.harness")"
    hb="$N/harness/$commit"
    [ -e "$hb/.done" ] || die "the native harness $commit of this run is missing"
    plan="$(json_get "$N/runs/$rid/state.json" plan_name)"
    kind="$(json_get "$N/runs/$rid/state.json" plan_kind)"
    seed="$(json_get "$N/runs/$rid/state.json" seed)"
    pause_early
    preflight_native "$kind"
    run_native "$rid" --plan "$hb/plans/$plan.json" --seed "$seed"
    finish_native "$rid" "$session_code"
    ;;
  extend)
    rid="${1:-}"
    [ -n "$rid" ] || die "extend needs a run id"
    shift
    blocks=""
    while [ $# -gt 0 ]; do
      case "$1" in
        --blocks) blocks="$2"; shift 2 ;;
        --cooling) cooling="$2"; shift 2 ;;
        --force) force=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    [ -n "$blocks" ] || die "extend needs --blocks ID[,ID...] (for example E5.hqc-3.encaps.r3)"
    folder="$repo/results/$osdir/$rid"
    [ -d "$folder" ] || die "no results folder $folder"
    ( cd "$folder" && shasum -a 256 -c --quiet SHA256SUMS ) || die "the SHA256SUMS of $rid does not verify"
    [ "$(json_get "$folder/manifest.json" status)" = "done" ] || die "only a finished run can be extended"
    kind="$(json_get "$folder/manifest.json" plan kind)"
    [ "$kind" = "main" ] || die "only main runs can be extended (this one is $kind)"
    [ "$(json_get "$folder/manifest.json" machine)" = "$MACHINE" ] || die "this launcher runs on $MACHINE only"
    plan="$(json_get "$folder/manifest.json" plan name)"
    seed="$(json_get "$folder/manifest.json" seed)"
    old_commit="$(json_get "$folder/manifest.json" harness_commit)"
    commit="$(repo_commit)"
    git -C "$repo" cat-file -e "$old_commit^{commit}" 2>/dev/null || die "the run's harness commit $old_commit is not in this repository"
    # As run.sh: everything that produces the numbers must be identical; the
    # orchestrator may differ (recorded in the run's manifest).
    if ! git -C "$repo" diff --quiet "$old_commit" -- cmd/kpi-prim cmd/kpi-env cmd/kpi-canary-native internal csrc docker plans config go.mod go.sum; then
      die "the measuring sources changed since $old_commit; the run cannot be extended"
    fi
    orch_changed="$(git -C "$repo" diff --name-only "$old_commit" -- cmd/kpi-batch | tr '\n' ',' | sed 's/,$//')"
    note "measuring sources are identical to those of the run ($old_commit); orchestrator files that differ: ${orch_changed:-none}"
    [ ! -e "$N/runs/$rid" ] || die "$N/runs/$rid already exists"
    mkdir -p "$repo/scratch/backup"
    rm -rf "$repo/scratch/backup/$rid"
    cp -R "$folder" "$repo/scratch/backup/$rid"
    note "backup of the finished run: scratch/backup/$rid"
    native_build "$commit"
    pause_early
    preflight_native "$kind"
    mkdir -p "$N/runs/$rid"
    cp -R "$folder/." "$N/runs/$rid/"
    rm -rf "$N/runs/$rid/launcher" "$N/runs/$rid/console.txt"
    find "$N/runs/$rid" -name .DS_Store -delete   # Finder files are not part of a run
    { cat "$folder/console.txt" 2>/dev/null || true
      printf '\n=== extension session (blocks without a clean attempt) ===\n'; } > "$N/runs/$rid.console.txt"
    printf '%s\n' "$commit" > "$N/runs/$rid.harness"
    persist="$(persist_dir "$rid")"
    rm -rf "$persist"; mkdir -p "$persist"
    cp "$folder/launcher/"* "$persist/"
    extending=1
    note "extending run $rid with: $blocks (native harness $commit)"
    run_native "$rid" --plan "$hb/plans/$plan.json" --seed "$seed" --extend "$blocks" --allow-rebuild \
      --orchestrator-changed "$orch_changed"
    finish_native "$rid" "$session_code"
    ;;
  stop)
    rid="${1:-}"
    [ -n "$rid" ] || die "stop needs a run id"
    pgrep -q -f "kpi-batch run --run-id $rid " || die "the native batch $rid is not running"
    : > "$N/runs/$rid/STOP"
    note "requested a graceful stop of $rid: it stops after the current block"
    ;;
  status)
    for d in "$N"/runs/*/; do
      [ -d "$d" ] || continue
      r="$(basename "$d")"
      if pgrep -q -f "kpi-batch run --run-id $r "; then s=running; else s=stopped; fi
      printf '%s\t%s\t%s\n' "$r" "$s" "$(json_get "$d/state.json" status 2>/dev/null || true)"
    done
    ;;
  *)
    die "unknown command $cmd"
    ;;
esac
