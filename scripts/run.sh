#!/bin/bash
# QChain KPI launcher (macOS with Colima; the Linux branch arrives in phase 1b).
#
#   scripts/run.sh start --plan NAME --machine ID [--variant b1-prod] [--seed N]
#                        [--cooling TEXT] [--rebuild] [--force]
#   scripts/run.sh resume RUN_ID
#   scripts/run.sh stop RUN_ID        # graceful: stops after the current block
#   scripts/run.sh status
#
# The launcher only prepares and supervises; all measurement and
# orchestration logic is Go inside the image (coding plan §2, §9). It:
#   - refuses to start on a machine that is not ready (trial and main plans;
#     smoke plans only warn), unless --force is given (recorded);
#   - builds the image from a clean commit and labels it with that commit;
#   - creates the run's container and named volume, measures the host-to-
#     container clock offset and (Mac) the native P-core/E-core canary
#     references, and delivers them as session.json;
#   - runs the host monitor and delivers its chunks into the volume every
#     10 s, keeps the Mac awake (caffeinate) and attaches to the batch;
#   - when the batch is done: copies the results out, verifies SHA256SUMS,
#     runs the privacy scan, adds the INDEX row, and removes our container and
#     volume (only ours, found by name and label). Pushing stays manual.
#
# Paths may contain spaces: every expansion is quoted.
set -euo pipefail

LAUNCHER="run.sh 1"
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
PERIOD_S=5        # host monitor sampling period
ROWS_PER_CHUNK=12 # one chunk per minute
GO=/opt/homebrew/opt/go@1.24/bin/go

die() { echo "run.sh: $*" >&2; exit 1; }
note() { echo "run.sh: $*" >&2; }

case "$(uname -s)" in
  Darwin) osdir=mac; runtime=colima-vz; cmem=3g ;;
  Linux) die "the Linux launcher arrives in phase 1b" ;;
  *) die "unsupported operating system" ;;
esac

now_ns() { /usr/bin/perl -MTime::HiRes=gettimeofday -e '($s,$u)=gettimeofday; printf "%d%06d000\n",$s,$u'; }

# json_get FILE KEY...: prints a value from a JSON file (Perl core JSON::PP).
json_get() {
  /usr/bin/perl -MJSON::PP -0777 -e '
    my $f = shift; open(my $h, "<", $f) or die "$f: $!"; local $/; my $d = decode_json(<$h>);
    for my $k (@ARGV) { $d = (ref $d eq "HASH") ? $d->{$k} : undef; }
    print defined $d ? $d : "";' "$@"
}

json_str() { local s="${1//\\/}"; s="${s//\"/}"; printf '"%s"' "$s"; }

# ------------------------------------------------------------------ checks

problems=()
problem() { problems+=("$1"); }

preflight() { # kind
  local kind="$1" n free_kb busy app out
  problems=()
  docker info >/dev/null 2>&1 || die "Docker is not reachable (is Colima running?)"
  [ "$(docker context show)" = "colima" ] || problem "docker context is not colima"
  out="$(colima status 2>&1 || true)"
  [[ "$out" == *"Virtualization.Framework"* ]] || problem "Colima is not running with --vm-type vz"
  colima list 2>/dev/null | awk 'NR==2{ if ($4 != "4" || $5 != "4GiB") exit 1 }' || problem "Colima VM is not 4 CPUs / 4 GiB"
  free_kb="$(df -Pk "$repo" | awk 'NR==2{print $4}')"
  [ "$free_kb" -ge 10485760 ] || problem "less than 10 GB free on the disk of the repository"
  n="$(docker ps -q | wc -l | tr -d ' ')"
  [ "$n" = "0" ] || problem "$n container(s) running; none may run during a batch"
  out="$(pmset -g batt)"
  [[ "$out" == *"AC Power"* ]] || problem "on battery power"
  n="$(pmset -g batt | grep -Eo '[0-9]+%' | head -n 1 | tr -d '%')"
  if [ -n "$n" ] && [ "$n" -lt 98 ]; then
    note "warning: battery at $n% (plan: 100%; use 'Charge to Full Now' in the battery menu)"
  fi
  [ "$(notifyutil -g com.apple.system.thermalpressurelevel | awk '{print $2}')" = "0" ] || problem "thermal pressure is above nominal"
  [ "$(pmset -g | awk '/lowpowermode/{print $2}')" != "1" ] || problem "Low Power Mode is on"
  for app in "MSTeams" "Microsoft Teams" "Google Chrome" "Microsoft Edge" "Code" "OneDrive"; do
    if pgrep -xq "$app"; then problem "$app is running (close it during trial and main runs)"; fi
  done
  busy="$(top -l 3 -s 2 -n 0 | awk '/^CPU usage/{u=$3; s=$5} END{gsub("%","",u); gsub("%","",s); printf "%.0f", u+s}')"
  [ "$busy" -le 10 ] || problem "host CPU busy at ${busy}% over 6 s (machine not quiet)"
  if [ "${#problems[@]}" -gt 0 ]; then
    for p in "${problems[@]}"; do note "preflight: $p"; done
    if [ "$kind" = "smoke" ]; then
      note "smoke plan: preflight problems are recorded, not enforced"
    elif [ "$force" = 1 ]; then
      note "--force: continuing despite preflight problems (recorded in the launcher files)"
    else
      die "preflight failed; fix the problems above (or use --force, which is recorded)"
    fi
  fi
}

repo_commit() {
  local dirty
  dirty="$(git -C "$repo" status --porcelain -- . ':!results')"
  [ -z "$dirty" ] || die "the harness has uncommitted changes (outside results/); commit them first"
  git -C "$repo" log -1 --format=%H -- . ':!results'
}

built=0
build_image() { # variant commit rebuild -> sets image_tag (and built=1 if it built)
  local variant="$1" commit="$2" rebuild="$3" tag label
  tag="kpi-bench:${variant}-${commit:0:12}"
  if [ "$rebuild" = 1 ] || ! docker image inspect "$tag" >/dev/null 2>&1; then
    built=1
    note "building $tag (liboqs gate and harness tests run inside the build)"
    docker buildx build --builder kpi --load --progress=plain \
      --build-arg "VARIANT=$variant" --build-arg "HARNESS_COMMIT=$commit" \
      -t "$tag" -f "$repo/docker/Dockerfile" "$repo" >&2
    docker buildx stop kpi >/dev/null 2>&1 || true
  fi
  label="$(docker image inspect -f '{{index .Config.Labels "org.qchain.kpi.harness-commit"}}' "$tag")"
  [ "$label" = "$commit" ] || die "image $tag is labelled with commit $label, not $commit"
  image_tag="$tag"
}

# ----------------------------------------------------------------- session

measure_offset() { # tag -> "offset_ns uncertainty_ns" (host clock minus container clock)
  local tag="$1" i h0 h1 c best_off=0 best_unc=-1 off unc
  for i in 1 2 3; do
    h0="$(now_ns)"
    c="$(docker run --rm --network none --entrypoint /opt/kpi/bin/kpi-batch "$tag" clock)"
    h1="$(now_ns)"
    off=$(( h0 / 2 + h1 / 2 - c ))
    unc=$(( (h1 - h0) / 2 ))
    if [ "$best_unc" -lt 0 ] || [ "$unc" -lt "$best_unc" ]; then best_off=$off; best_unc=$unc; fi
    : "$i"
  done
  printf '%s %s\n' "$best_off" "$best_unc"
}

native_refs() { # workdir persistdir n -> "P E" (0 0 when unusable)
  local work="$1" persist="$2" n="$3" p e
  GOTOOLCHAIN=local CGO_ENABLED=0 "$GO" build -C "$repo" -o "$work/kpi-canary-native" ./cmd/kpi-canary-native
  "$work/kpi-canary-native" --label plain > "$persist/native-canary-s$n-plain.json"
  taskpolicy -b "$work/kpi-canary-native" --label background > "$persist/native-canary-s$n-background.json"
  p="$(json_get "$persist/native-canary-s$n-plain.json" canary median_rate_per_s)"
  e="$(json_get "$persist/native-canary-s$n-background.json" canary median_rate_per_s)"
  if /usr/bin/perl -e 'exit(($ARGV[0] > 1.5 * $ARGV[1] && $ARGV[1] > 0) ? 0 : 1)' "$p" "$e"; then
    printf '%s %s\n' "$p" "$e"
  else
    note "warning: native canary references are not separated (P $p, E $e); core-type check disabled this session"
    printf '0 0\n'
  fi
}

write_session() { # file offset unc p e cooling
  local f="$1" lpm therm batt power ncpu mem docker_v colima_v lima_v go_v
  ncpu="$(sysctl -n hw.logicalcpu)"
  mem=$(( $(sysctl -n hw.memsize) / 1073741824 ))
  docker_v="$(docker version --format '{{.Server.Version}}')"
  colima_v="$(colima version 2>/dev/null | awk 'NR==1{print $3}')"
  lima_v="$(limactl --version 2>/dev/null | awk '{print $3}')"
  go_v="$("$GO" version | awk '{print $3}')"
  lpm="$(pmset -g | awk '/lowpowermode/{print $2}')"
  therm="$(notifyutil -g com.apple.system.thermalpressurelevel | awk '{print $2}')"
  batt="$(pmset -g batt | grep -Eo '[0-9]+%' | head -n 1 | tr -d '%')"
  if [[ "$(pmset -g batt)" == *"AC Power"* ]]; then power=AC; else power=battery; fi
  {
    printf '{\n'
    printf '  "host_offset_ns": %s,\n  "host_offset_unc_ns": %s,\n' "$2" "$3"
    printf '  "native_p_rate": %s,\n  "native_e_rate": %s,\n' "$4" "$5"
    printf '  "host_logical_cpus": %s,\n  "host_monitor_period_s": %s,\n' "$ncpu" "$PERIOD_S"
    printf '  "host": {\n'
    printf '    "runtime": %s,\n' "$(json_str "$runtime")"
    printf '    "host_os": %s,\n' "$(json_str "macOS $(sw_vers -productVersion) ($(sw_vers -buildVersion))")"
    printf '    "host_cpu": %s,\n' "$(json_str "$(sysctl -n machdep.cpu.brand_string)")"
    printf '    "host_mem_gb": %s,\n' "$(json_str "$mem")"
    printf '    "docker": %s,\n    "colima": %s,\n    "lima": %s,\n' "$(json_str "$docker_v")" "$(json_str "$colima_v")" "$(json_str "$lima_v")"
    printf '    "vm_cpus": "4",\n    "vm_mem_gb": "4",\n'
    printf '    "container_mem": %s,\n    "container_cpuset": "1",\n' "$(json_str "$cmem")"
    printf '    "power": %s,\n    "battery_pct": %s,\n' "$(json_str "$power")" "$(json_str "$batt")"
    printf '    "cooling": %s,\n' "$(json_str "$6")"
    printf '    "low_power_mode": %s,\n    "thermal_at_start": %s,\n' "$(json_str "$lpm")" "$(json_str "$therm")"
    printf '    "launcher": %s,\n    "native_go": %s\n' "$(json_str "$LAUNCHER")" "$(json_str "$go_v")"
    printf '  }\n}\n'
  } > "$f"
  json_get "$f" host runtime >/dev/null || die "session.json is not valid JSON"
}

deliver_chunks() { # workdir rid session
  local w="$1" rid="$2" s="$3" r base name
  for r in "$w"/host-*.ready; do
    [ -e "$r" ] || continue
    base="${r%.ready}"
    name="host-s$s-${base##*/host-}"
    if docker cp "$base.csv" "kpi-run-$rid:/out/monitor/$name.csv" >/dev/null 2>&1 &&
       docker cp "$w/empty.ok" "kpi-run-$rid:/out/monitor/$name.ok" >/dev/null 2>&1; then
      mv "$r" "$base.delivered" 2>/dev/null || true
    fi
  done
}

persist_dir() { printf '%s\n' "$HOME/Library/Application Support/qchain-kpi/$1"; }

# run_session RID: one start or resume, from session.json to the batch's exit.
# Sets session_code to the batch's exit code.
session_code=""
run_session() {
  local rid="$1" tag persist n work offs offset unc refs p e monpid delpid cafpid started
  tag="$(docker inspect -f '{{.Config.Image}}' "kpi-run-$rid")"
  persist="$(persist_dir "$rid")"
  mkdir -p "$persist"
  n=$(( $(find "$persist" -maxdepth 1 -name 'session-s*.json' | wc -l) + 1 ))
  work="$(mktemp -d "${TMPDIR:-/tmp}/kpi-mon.XXXXXX")"
  if [ "${#problems[@]}" -gt 0 ]; then printf '%s\n' "${problems[@]}" > "$persist/preflight-s$n.txt"; else : > "$persist/preflight-s$n.txt"; fi
  [ "$force" = 1 ] && echo "forced" >> "$persist/preflight-s$n.txt"

  offs="$(measure_offset "$tag")"
  offset="${offs% *}"; unc="${offs#* }"
  note "host-to-container clock offset ${offset} ns (+/- ${unc} ns)"
  refs="$(native_refs "$work" "$persist" "$n")"
  p="${refs% *}"; e="${refs#* }"
  note "native canary references: P-core ${p}/s, E-core ${e}/s"
  write_session "$persist/session-s$n.json" "$offset" "$unc" "$p" "$e" "$cooling"
  docker cp "$persist/session-s$n.json" "kpi-run-$rid:/out/session.json" >/dev/null

  : > "$work/empty.ok"
  "$repo/scripts/monitor-mac.sh" "$work" "$PERIOD_S" "$ROWS_PER_CHUNK" "$$" >/dev/null &
  monpid=$!
  ( while :; do deliver_chunks "$work" "$rid" "$n"; sleep 10; done ) >/dev/null 2>&1 &
  delpid=$!
  disown "$delpid"
  caffeinate -i -s -w "$$" >/dev/null 2>&1 &
  cafpid=$!
  disown "$cafpid"

  trap 'note "interrupt: the batch discards its current block and stops; resume with: scripts/run.sh resume '"$rid"'"' INT
  note "starting batch $rid (session $n); stop gracefully with: scripts/run.sh stop $rid"
  started="$(docker inspect -f '{{.State.StartedAt}}' "kpi-run-$rid")"
  # -a attaches the batch's output and forwards signals (Ctrl-C) to it.
  docker start -a "kpi-run-$rid" >&2 || true
  while [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid")" = "true" ]; do sleep 2; done
  trap - INT
  if [ "$(docker inspect -f '{{.State.StartedAt}}' "kpi-run-$rid")" = "$started" ]; then
    session_code=start-failed
  else
    session_code="$(docker inspect -f '{{.State.ExitCode}}' "kpi-run-$rid")"
  fi

  kill "$delpid" 2>/dev/null || true
  kill "$monpid" 2>/dev/null || true
  wait "$monpid" 2>/dev/null || true
  deliver_chunks "$work" "$rid" "$n"
  cp "$work/host-all.csv" "$persist/host-monitor-s$n.csv" 2>/dev/null || true
  kill "$cafpid" 2>/dev/null || true
  offs="$(measure_offset "$tag")"
  printf 'start %s %s\nend %s\n' "$offset" "$unc" "$offs" > "$persist/clock-offset-s$n.txt"
  rm -rf "$work"
}

# ----------------------------------------------------------------- finish

privacy_patterns() {
  {
    hostname; hostname -s
    scutil --get ComputerName 2>/dev/null || true
    scutil --get LocalHostName 2>/dev/null || true
    id -un; id -F 2>/dev/null || true
    printf '%s\n' "$HOME"
    ifconfig -a 2>/dev/null | awk '/inet /{print $2}' | grep -v '^127\.' || true
    ifconfig -a 2>/dev/null | awk '/ether /{print $2}' || true
    ioreg -rd1 -c IOPlatformExpertDevice 2>/dev/null | awk -F'"' '/IOPlatformSerialNumber|IOPlatformUUID/{print $4}' || true
  } | awk 'length($0) >= 4' | sort -u
}

# privacy_scan DIR: fails if any file contains this machine's identifying
# text. The patterns are only held in memory; matches are reported by file
# name only.
privacy_scan() {
  local dir="$1" pats f n bad=0
  pats="$(privacy_patterns)"
  [ -n "$pats" ] || die "privacy scan: no patterns computed"
  while IFS= read -r -d '' f; do
    case "$f" in
      *.gz) n="$(gzip -dc -- "$f" | grep -F -i -c -f <(printf '%s\n' "$pats") || true)" ;;
      *) n="$(grep -F -i -c -f <(printf '%s\n' "$pats") -- "$f" || true)" ;;
    esac
    if [ "${n:-0}" != "0" ]; then
      note "privacy scan: identifying text in ${f#"$dir"/}"
      bad=1
    fi
  done < <(find "$dir" -type f -print0)
  return "$bad"
}

index_row() { # dest rid
  local dest="$1" rid="$2" idx
  idx="$repo/results/$osdir/INDEX.md"
  if [ ! -f "$idx" ]; then
    printf '# Runs on this machine\n\n| Run | Plan | Kind | Status | Attempts | Flagged | Harness commit | Seed |\n|---|---|---|---|---|---|---|---|\n' > "$idx"
  fi
  printf '| %s | %s | %s | %s | %s | %s | %s | %s |\n' "$rid" \
    "$(json_get "$dest/manifest.json" plan name)" "$(json_get "$dest/manifest.json" plan kind)" \
    "$(json_get "$dest/manifest.json" status)" "$(json_get "$dest/manifest.json" counts attempts)" \
    "$(json_get "$dest/manifest.json" counts flagged_attempts)" \
    "$(json_get "$dest/manifest.json" harness_commit | cut -c1-12)" "$(json_get "$dest/manifest.json" seed)" >> "$idx"
}

finish_run() { # rid code
  local rid="$1" code="$2" stage dest persist
  persist="$(persist_dir "$rid")"
  case "$code" in
    0) ;;
    start-failed)
      note "the batch container did not start (see the messages above); kpi-run-$rid and kpi-out-$rid were kept"
      return 1 ;;
    5)
      note "batch stopped and can be resumed: scripts/run.sh resume $rid"
      return 0 ;;
    *)
      stage="$repo/scratch/failed/$rid"
      rm -rf "$stage"; mkdir -p "$stage"
      docker cp "kpi-run-$rid:/out/." "$stage/" >/dev/null
      docker logs "kpi-run-$rid" > "$stage/console.txt" 2>&1 || true
      note "batch ended with exit code $code; its files were copied to scratch/failed/$rid for inspection"
      note "the container kpi-run-$rid and volume kpi-out-$rid were kept"
      return 1 ;;
  esac
  stage="$repo/scratch/staging/$rid"
  dest="$repo/results/$osdir/$rid"
  [ ! -e "$dest" ] || die "$dest already exists"
  rm -rf "$stage"; mkdir -p "$stage"
  docker cp "kpi-run-$rid:/out/." "$stage/" >/dev/null
  docker logs "kpi-run-$rid" > "$stage/console.txt" 2>&1 || true
  mkdir -p "$stage/launcher"
  cp "$persist"/* "$stage/launcher/"
  ( cd "$stage" && shasum -a 256 -c --quiet SHA256SUMS ) || die "SHA256SUMS verification failed in scratch/staging/$rid (container kept)"
  privacy_scan "$stage" || die "privacy scan failed: nothing was moved into results/ (scratch/staging/$rid; container kept)"
  mkdir -p "$(dirname "$dest")"
  mv "$stage" "$dest"
  index_row "$dest" "$rid"
  docker rm "kpi-run-$rid" >/dev/null
  docker volume rm "kpi-out-$rid" >/dev/null
  rm -rf "$persist"
  note "done: results/$osdir/$rid (checksums verified, privacy scan clean); container and volume removed"
  note "to commit: git -C \"$repo\" add \"results/$osdir/$rid\" \"results/$osdir/INDEX.md\" && git -C \"$repo\" commit -m \"results: $rid\""
}

# ------------------------------------------------------------------- main

cmd="${1:-}"
[ -n "$cmd" ] || die "usage: run.sh start|resume|stop|status ..."
shift
force=0
cooling="unstated"

case "$cmd" in
  start)
    plan="" machine="" variant="b1-prod" seed="" rebuild=0
    while [ $# -gt 0 ]; do
      case "$1" in
        --plan) plan="$2"; shift 2 ;;
        --machine) machine="$2"; shift 2 ;;
        --variant) variant="$2"; shift 2 ;;
        --seed) seed="$2"; shift 2 ;;
        --cooling) cooling="$2"; shift 2 ;;
        --rebuild) rebuild=1; shift ;;
        --force) force=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    [ -n "$plan" ] && [ -n "$machine" ] || die "start needs --plan and --machine"
    [ "$machine" = "mac-m1" ] || die "this launcher runs on mac-m1 only (got $machine)"
    planfile="$repo/plans/$plan.json"
    [ -f "$planfile" ] || die "no plan plans/$plan.json"
    kind="$(json_get "$planfile" kind)"
    commit="$(repo_commit)"
    build_image "$variant" "$commit" "$rebuild"
    tag="$image_tag"
    if [ "$built" = 1 ] && [ "$kind" != "smoke" ]; then
      note "cooling down for 120 s after the image build"
      sleep 120
    fi
    preflight "$kind"
    imageid="$(docker image inspect -f '{{.Id}}' "$tag")"
    [ -n "$seed" ] || seed="$(od -An -N8 -tu8 /dev/urandom | tr -d ' \n')"
    rid="$(date -u +%Y%m%d-%H%M)_${machine}_${runtime}_${variant}"
    [ ! -e "$repo/results/$osdir/$rid" ] || die "results/$osdir/$rid already exists"
    docker volume create --label org.qchain.kpi=1 "kpi-out-$rid" >/dev/null
    docker create --init --name "kpi-run-$rid" --stop-timeout 30 \
      --label org.qchain.kpi=1 --label "org.qchain.kpi.run=$rid" --label "org.qchain.kpi.plan=$plan" \
      --label "org.qchain.kpi.kind=$kind" --label "org.qchain.kpi.machine=$machine" \
      --network none --cpuset-cpus=1 --cpus=1 --memory="$cmem" --memory-swap="$cmem" -e GOMAXPROCS=1 \
      --mount "type=volume,src=kpi-out-$rid,dst=/out" \
      "$tag" run --plan "/opt/kpi/plans/$plan.json" --run-id "$rid" --machine "$machine" \
      --seed "$seed" --image-id "$imageid" >/dev/null
    note "run $rid: plan $plan ($kind), seed $seed, image $tag"
    run_session "$rid"
    finish_run "$rid" "$session_code"
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
    docker inspect "kpi-run-$rid" >/dev/null 2>&1 || die "no container kpi-run-$rid"
    [ "$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi"}}' "kpi-run-$rid")" = "1" ] || die "kpi-run-$rid is not a KPI container"
    [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid")" = "false" ] || die "kpi-run-$rid is already running"
    kind="$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi.kind"}}' "kpi-run-$rid")"
    preflight "$kind"
    run_session "$rid"
    finish_run "$rid" "$session_code"
    ;;
  stop)
    rid="${1:-}"
    [ -n "$rid" ] || die "stop needs a run id"
    [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid" 2>/dev/null)" = "true" ] || die "kpi-run-$rid is not running"
    tmp="$(mktemp "${TMPDIR:-/tmp}/kpi-stop.XXXXXX")"
    docker cp "$tmp" "kpi-run-$rid:/out/STOP" >/dev/null
    rm -f "$tmp"
    note "requested a graceful stop of $rid: it stops after the current block"
    ;;
  status)
    docker ps -a --filter label=org.qchain.kpi=1 --format 'table {{.Names}}\t{{.Status}}\t{{.Label "org.qchain.kpi.plan"}}'
    ;;
  *)
    die "unknown command $cmd"
    ;;
esac
