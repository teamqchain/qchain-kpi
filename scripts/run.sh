#!/bin/bash
# QChain KPI launcher (macOS with Colima; the Linux branch arrives in phase 1b).
#
#   scripts/run.sh start --plan NAME --machine ID [--variant b1-prod] [--seed N]
#                        [--cooling TEXT] [--rebuild] [--force]
#   scripts/run.sh resume RUN_ID
#   scripts/run.sh extend RUN_ID --blocks ID[,ID...]   # run blocks without a clean attempt again, inside the finished run
#   scripts/run.sh stop RUN_ID        # graceful: stops after the current block
#   scripts/run.sh status
#
# The launcher only prepares and supervises; all measurement and
# orchestration logic is Go inside the image (coding plan §2, §9). The
# container is pinned to one CPU with --cpuset-cpus (E12 plans: the CPUs the
# plan names, for its concurrent workers) and has NO CPU-time quota
# (--cpus): a quota equal to the pinned CPU throttles a busy process every few
# periods (measured 29 of 100 periods, ~1.8 ms each; decision of 10 Oct). It:
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

# plan_cpuset PLANFILE: the container's CPUs. Every plan runs pinned to CPU 1
# except E12 plans, which name several (their "cpuset"; kpi-batch checks it).
plan_cpuset() {
  local c
  c="$(json_get "$1" cpuset)"
  [[ "$c" =~ ^[0-9]+([-,][0-9]+)*$ ]] || c=1
  printf '%s' "$c"
}

json_str() { local s="${1//\\/}"; s="${s//\"/}"; printf '"%s"' "$s"; }

# ------------------------------------------------------------------ checks

# The Colima VM does not survive a Mac sleep (it is stopped after wake), and the
# Mac sleeps whenever it is idle between runs. Start it again if it is down,
# with the study's settings (decision 10 Oct).
ensure_colima() {
  local out
  out="$(colima status 2>&1 || true)"
  if [[ "$out" == *"is not running"* ]]; then
    note "Colima is not running (the Mac probably slept); starting it"
    colima start --vm-type vz --cpus 4 --memory 4 --mount none >&2 || die "colima start failed"
  fi
}

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
  # macOS must not download or install an update (and restart) during the study.
  for key in AutomaticallyInstallMacOSUpdates AutomaticDownload; do
    out="$(defaults read /Library/Preferences/com.apple.SoftwareUpdate "$key" 2>/dev/null || true)"
    [ "$out" != "1" ] || problem "automatic macOS updates are on ($key); turn off 'Download new updates when available' and 'Install macOS updates' in System Settings > General > Software Update > Automatic updates"
  done
  out="$(defaults read /Library/Preferences/com.apple.commerce AutoUpdate 2>/dev/null || true)"
  [ "$out" != "1" ] || note "note: App Store apps update automatically; their downloads can disturb a run"
  for app in "MSTeams" "Microsoft Teams" "Google Chrome" "Microsoft Edge" "Code" "OneDrive"; do
    # -a: macOS pgrep otherwise skips its own ancestors (e.g. VS Code when run from its terminal).
    if pgrep -a -x -q "$app"; then problem "$app is running (close it during trial and main runs)"; fi
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
    printf '    "container_mem": %s,\n    "container_cpuset": %s,\n' "$(json_str "$cmem")" "$(json_str "${container_cpuset:-1}")"
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

# macOS background analysis services (Photos analysis, the Spotlight AI
# indexer, suggestions) run whenever the Mac is idle on mains power, which are
# exactly the test conditions. In the 10 Oct trials they used up to two cores
# and caused thermal pressure even with their settings switched off. They are
# paused (SIGSTOP) for the duration of a session and resumed (SIGCONT) on every
# exit path (decision 10 Oct). They run as the user, so no admin rights are
# needed; a logout or restart resumes them as well.
BG_PATTERN='/(mediaanalysisd|photoanalysisd|spotlightknowledged|spotlightknowledged[.]importer|spotlightknowledged[.]updater|suggestd|duetexpertd|knowledge-agent|knowledgeconstructiond|AddressBookSourceSync|appstoreagent)( |$)'
pause_bg() { # logfile
  local pid name
  for pid in $(pgrep -f "$BG_PATTERN" 2>/dev/null); do
    if [ "$(ps -o stat= -p "$pid" 2>/dev/null | cut -c1)" != "T" ]; then
      name="$(ps -o comm= -p "$pid" 2>/dev/null)"
      name="${name##*/}"
      if kill -STOP "$pid" 2>/dev/null; then
        printf '%s paused %s %s\n' "$(date -u +%H:%M:%S)" "$pid" "$name" >> "$1"
      fi
    fi
  done
}
resume_bg() { pkill -CONT -f "$BG_PATTERN" 2>/dev/null || true; }
bg_early_log="$(mktemp "${TMPDIR:-/tmp}/kpi-bg.XXXXXX")"
pause_early() { pause_bg "$bg_early_log"; bg_paused=1; }

# Wi-Fi is switched off for the run itself, after the image has been built (the
# build needs the internet; the measurement does not) and switched back on when
# the script ends, on every exit path. It is restored only if it was on before.
# While it is off nothing can sync, update or download during a measurement
# (decision 11 Oct). It needs no admin rights.
wifi_dev=""
wifi_cut=0
bg_paused=0
wifi_off() { # logfile
  local dev out
  dev="$(networksetup -listallhardwareports | awk '/Hardware Port: Wi-Fi/{getline; print $2; exit}')"
  [ -n "$dev" ] || { note "no Wi-Fi device found; nothing to switch off"; return 0; }
  out="$(networksetup -getairportpower "$dev" 2>&1 || true)"
  if [[ "$out" == *": On" ]]; then
    wifi_dev="$dev"
    if networksetup -setairportpower "$dev" off >/dev/null 2>&1; then
      wifi_cut=1
      printf '%s Wi-Fi (%s) switched off\n' "$(date -u +%H:%M:%S)" "$dev" >> "$1"
      note "Wi-Fi switched off for this run (it is switched back on when the script ends)"
    else
      note "warning: could not switch Wi-Fi off; the run continues with Wi-Fi on"
    fi
  else
    note "Wi-Fi was already off; it stays off"
  fi
}
wifi_restore() {
  if [ "$wifi_cut" = 1 ]; then
    wifi_cut=0
    if networksetup -setairportpower "$wifi_dev" on >/dev/null 2>&1; then
      note "Wi-Fi switched back on"
    else
      note "WARNING: could not switch Wi-Fi back on; run: networksetup -setairportpower $wifi_dev on"
    fi
  fi
}

# Everything this script changes temporarily is undone here, on every exit path.
cleanup() {
  if [ "$bg_paused" = 1 ]; then resume_bg; bg_paused=0; fi
  wifi_restore
  rm -f "$bg_early_log" 2>/dev/null || true
}
trap cleanup EXIT
trap 'cleanup; exit 129' HUP
trap 'cleanup; exit 143' TERM

# run_session RID: one start or resume, from session.json to the batch's exit.
# Sets session_code to the batch's exit code.
session_code=""
extending=0       # 1 while extending a finished run (see the extend command)
extend_folder=""  # results folder of the run being extended
run_session() {
  local rid="$1" tag persist n work offs offset unc refs p e monpid delpid cafpid started
  tag="$(docker inspect -f '{{.Config.Image}}' "kpi-run-$rid")"
  persist="$(persist_dir "$rid")"
  mkdir -p "$persist"
  n=$(( $(find "$persist" -maxdepth 1 -name 'session-s*.json' | wc -l) + 1 ))
  work="$(mktemp -d "${TMPDIR:-/tmp}/kpi-mon.XXXXXX")"
  if [ "${#problems[@]}" -gt 0 ]; then printf '%s\n' "${problems[@]}" > "$persist/preflight-s$n.txt"; else : > "$persist/preflight-s$n.txt"; fi
  [ "$force" = 1 ] && echo "forced" >> "$persist/preflight-s$n.txt"

  # The host monitor starts first, so it is warmed up (top's first samples are
  # discarded) and delivering before the batch's first block.
  : > "$work/empty.ok"
  "$repo/scripts/monitor-mac.sh" "$work" "$PERIOD_S" "$ROWS_PER_CHUNK" "$$" >/dev/null &
  monpid=$!
  cat "$bg_early_log" >> "$persist/bg-paused-s$n.txt" 2>/dev/null || true
  : > "$bg_early_log"
  pause_bg "$persist/bg-paused-s$n.txt"
  bg_paused=1
  wifi_off "$persist/wifi-s$n.txt"
  note "paused macOS background analysis services for this run (resumed automatically at the end)"
  ( while :; do deliver_chunks "$work" "$rid" "$n"; pause_bg "$persist/bg-paused-s$n.txt"; sleep 10; done ) >/dev/null 2>&1 &
  delpid=$!
  disown "$delpid"

  offs="$(measure_offset "$tag")"
  offset="${offs% *}"; unc="${offs#* }"
  note "host-to-container clock offset ${offset} ns (+/- ${unc} ns)"
  refs="$(native_refs "$work" "$persist" "$n")"
  p="${refs% *}"; e="${refs#* }"
  note "native canary references: P-core ${p}/s, E-core ${e}/s"
  container_cpuset="$(docker inspect -f '{{.HostConfig.CpusetCpus}}' "kpi-run-$rid")"
  write_session "$persist/session-s$n.json" "$offset" "$unc" "$p" "$e" "$cooling"
  docker cp "$persist/session-s$n.json" "kpi-run-$rid:/out/session.json" >/dev/null

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
  resume_bg
  bg_paused=0
  note "resumed the paused macOS background services"
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
  if [ "$extending" != 1 ]; then [ ! -e "$dest" ] || die "$dest already exists"; fi
  rm -rf "$stage"; mkdir -p "$stage"
  docker cp "kpi-run-$rid:/out/." "$stage/" >/dev/null
  if [ "$extending" = 1 ]; then
    { cat "$extend_folder/console.txt" 2>/dev/null || true
      printf '\n=== extension session (blocks without a clean attempt) ===\n'
      docker logs "kpi-run-$rid" 2>&1 || true; } > "$stage/console.txt"
  else
    docker logs "kpi-run-$rid" > "$stage/console.txt" 2>&1 || true
  fi
  mkdir -p "$stage/launcher"
  cp "$persist"/* "$stage/launcher/"
  ( cd "$stage" && shasum -a 256 -c --quiet SHA256SUMS ) || die "SHA256SUMS verification failed in scratch/staging/$rid (container kept)"
  privacy_scan "$stage" || die "privacy scan failed: nothing was moved into results/ (scratch/staging/$rid; container kept)"
  mkdir -p "$(dirname "$dest")"
  if [ "$extending" = 1 ]; then
    # The original folder is kept in scratch/backup/ (made before the extension) and in git.
    rm -rf "$dest"
    grep -v -F "| $rid |" "$repo/results/$osdir/INDEX.md" > "$repo/results/$osdir/INDEX.md.new" || true
    mv "$repo/results/$osdir/INDEX.md.new" "$repo/results/$osdir/INDEX.md"
  fi
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
[ -n "$cmd" ] || die "usage: run.sh start|resume|extend|stop|status ..."
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
    cpuset="$(plan_cpuset "$planfile")"
    commit="$(repo_commit)"
    ensure_colima
    build_image "$variant" "$commit" "$rebuild"
    tag="$image_tag"
    # The macOS background services wake up as soon as the Mac is idle (for
    # example during the cool-down) and would fail the quiet-machine check, so
    # they are paused from here on (the session logs them; cleanup resumes them).
    pause_early
    if [ "$built" = 1 ] && [ "$kind" != "smoke" ]; then
      note "cooling down for 120 s after the image build (background services paused)"
      sleep 120
      pause_early
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
      --network none --cpuset-cpus="$cpuset" --memory="$cmem" --memory-swap="$cmem" -e GOMAXPROCS=1 \
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
    ensure_colima
    docker inspect "kpi-run-$rid" >/dev/null 2>&1 || die "no container kpi-run-$rid"
    [ "$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi"}}' "kpi-run-$rid")" = "1" ] || die "kpi-run-$rid is not a KPI container"
    [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid")" = "false" ] || die "kpi-run-$rid is already running"
    kind="$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi.kind"}}' "kpi-run-$rid")"
    pause_early
    preflight "$kind"
    run_session "$rid"
    finish_run "$rid" "$session_code"
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
    [ -n "$blocks" ] || die "extend needs --blocks ID[,ID...] (for example E1.hqc-3.encaps.r3)"
    folder="$repo/results/$osdir/$rid"
    [ -d "$folder" ] || die "no results folder $folder"
    ( cd "$folder" && shasum -a 256 -c --quiet SHA256SUMS ) || die "the SHA256SUMS of $rid does not verify"
    [ "$(json_get "$folder/manifest.json" status)" = "done" ] || die "only a finished run can be extended"
    kind="$(json_get "$folder/manifest.json" plan kind)"
    [ "$kind" = "main" ] || die "only main runs can be extended (this one is $kind)"
    plan="$(json_get "$folder/manifest.json" plan name)"
    cpuset="$(plan_cpuset "$repo/plans/$plan.json")"
    seed="$(json_get "$folder/manifest.json" seed)"
    machine="$(json_get "$folder/manifest.json" machine)"
    variant="$(json_get "$folder/manifest.json" build variant)"
    old_commit="$(json_get "$folder/manifest.json" harness_commit)"
    [ "$machine" = "mac-m1" ] || die "this launcher runs on mac-m1 only"
    commit="$(repo_commit)"
    git -C "$repo" cat-file -e "$old_commit^{commit}" 2>/dev/null || die "the run's harness commit $old_commit is not in this repository"
    # Everything that produces the numbers must be identical. The orchestrator
    # (cmd/kpi-batch) may differ, because it is the part that implements the
    # extension; its changed files are recorded in the run's manifest.
    if ! git -C "$repo" diff --quiet "$old_commit" -- cmd/kpi-prim cmd/kpi-env cmd/kpi-canary-native internal csrc docker plans config go.mod go.sum; then
      die "the measuring sources (cmd/kpi-prim, cmd/kpi-env, cmd/kpi-canary-native, internal, csrc, docker, plans, config, go.mod) changed since $old_commit; the run cannot be extended"
    fi
    orch_changed="$(git -C "$repo" diff --name-only "$old_commit" -- cmd/kpi-batch | tr '\n' ',' | sed 's/,$//')"
    note "measuring sources are identical to those of the run ($old_commit); orchestrator files that differ: ${orch_changed:-none}"
    mkdir -p "$repo/scratch/backup"
    rm -rf "$repo/scratch/backup/$rid"
    cp -R "$folder" "$repo/scratch/backup/$rid"
    note "backup of the finished run: scratch/backup/$rid"
    ensure_colima
    build_image "$variant" "$commit" 0
    tag="$image_tag"
    pause_early
    if [ "$built" = 1 ]; then
      note "cooling down for 120 s after the image build (background services paused)"
      sleep 120
      pause_early
    fi
    preflight "$kind"
    imageid="$(docker image inspect -f '{{.Id}}' "$tag")"
    ! docker inspect "kpi-run-$rid" >/dev/null 2>&1 || die "container kpi-run-$rid already exists"
    docker volume create --label org.qchain.kpi=1 "kpi-out-$rid" >/dev/null
    docker create --init --name "kpi-run-$rid" --stop-timeout 30 \
      --label org.qchain.kpi=1 --label "org.qchain.kpi.run=$rid" --label "org.qchain.kpi.plan=$plan" \
      --label "org.qchain.kpi.kind=$kind" --label "org.qchain.kpi.machine=$machine" \
      --network none --cpuset-cpus="$cpuset" --memory="$cmem" --memory-swap="$cmem" -e GOMAXPROCS=1 \
      --mount "type=volume,src=kpi-out-$rid,dst=/out" \
      "$tag" run --plan "/opt/kpi/plans/$plan.json" --run-id "$rid" --machine "$machine" \
      --seed "$seed" --image-id "$imageid" --extend "$blocks" --allow-rebuild \
      --orchestrator-changed "$orch_changed" >/dev/null
    # The finished run goes back into the volume (without the launcher's own files,
    # which are re-added at the end) and the volume is handed to the unprivileged user.
    prep="$(mktemp -d "${TMPDIR:-/tmp}/kpi-ext.XXXXXX")"
    cp -R "$folder/." "$prep/"
    rm -rf "$prep/launcher" "$prep/console.txt"
    find "$prep" -name .DS_Store -delete   # Finder files are not part of a run
    docker cp "$prep/." "kpi-run-$rid:/out/" >/dev/null
    rm -rf "$prep"
    docker run --rm --network none --user root --entrypoint /bin/chown \
      --mount "type=volume,src=kpi-out-$rid,dst=/out" "$tag" -R kpi:kpi /out
    persist="$(persist_dir "$rid")"
    rm -rf "$persist"; mkdir -p "$persist"
    cp "$folder/launcher/"* "$persist/"
    extending=1
    extend_folder="$folder"
    note "extending run $rid with: $blocks (image $tag)"
    run_session "$rid"
    finish_run "$rid" "$session_code"
    ;;
  stop)
    rid="${1:-}"
    [ -n "$rid" ] || die "stop needs a run id"
    [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid" 2>/dev/null)" = "true" ] || die "kpi-run-$rid is not running"
    tmp="$(mktemp "${TMPDIR:-/tmp}/kpi-stop.XXXXXX")"
    chmod 644 "$tmp"
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
