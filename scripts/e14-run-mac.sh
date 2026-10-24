#!/bin/bash
# E14 launcher (macOS with Colima): runs the E14 client against the Fabric
# network of scripts/e14-network-mac.sh (design: docs/plan.md, "E14 design"; method: docs/methodology.md, E14).
#
#   scripts/e14-run-mac.sh start --plan smoke-e14|e14 [--seed N] [--cooling TEXT] [--force]
#   scripts/e14-run-mac.sh resume RUN_ID [--cooling TEXT] [--force]
#   scripts/e14-run-mac.sh stop RUN_ID        # graceful: stops after the current block
#   scripts/e14-run-mac.sh status
#   scripts/e14-run-mac.sh check         # read-only: network up, preflight (reported only), network.json
#
# It does what run-mac.sh does, with run-mac.sh's own functions (loaded from its
# "shared" blocks): the quiet-machine preflight (main plans refuse, smoke
# plans record), the background-service pause, Wi-Fi off for the run, the host
# monitor delivered into the run volume, the clock offset and the native
# canary references, and at the end the copy-out, SHA256SUMS verification,
# privacy scan, the move into results/mac/ and the INDEX row. Differences:
#   - the client image (fabric/Dockerfile) is built from the clean commit and
#     tagged kpi-fabric-client:<commit>;
#   - the run container joins the network docker_fabric_net (it talks to the
#     peers) and reads the certificates from kpi-fabric-material (read-only);
#     it is not pinned to one CPU and has no memory cap, like the network's
#     containers and QChain's backend;
#   - the network must be up (started here if Colima was restarted) and no
#     other container may run; network.json records its images and chaincode.
# Paths may contain spaces: every expansion is quoted.
# shellcheck disable=SC2154,SC2034 # osdir, runtime, bg_* and problems belong to run-mac.sh's shared blocks, loaded below
set -euo pipefail

LAUNCHER="e14-run-mac.sh 1"
LAUNCHER_CMD=scripts/e14-run-mac.sh
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
PERIOD_S=5        # host monitor sampling period
ROWS_PER_CHUNK=12 # one chunk per minute
GO=/opt/homebrew/opt/go@1.24/bin/go
MACHINE=mac-m1
VARIANT=fabric2.5.15
NET=docker_fabric_net
VOL=kpi-fabric-material
NETWORK=(ca.government.uae.com ca.general.uae.com ca.orderer.example.com couchdb0
         orderer0.orderer.example.com peer0.government.uae.com peer0.general.uae.com)

# run-mac.sh's shared functions (eval: macOS's bash 3.2 cannot source a process substitution).
shared="$(awk '/^# >>> shared with scripts\/(e5-run-mac|e14-run-mac)\.sh/{f=1; next} /^# <<< shared with scripts\/(e5-run-mac|e14-run-mac)\.sh/{f=0} f' "$repo/scripts/run-mac.sh")"
eval "$shared"
for fn in repo_commit preflight_host preflight_verdict free_space_check native_refs pause_bg resume_bg wifi_off cleanup \
          persist_dir privacy_scan index_row json_get json_str ensure_colima deliver_chunks finish_run; do
  declare -F "$fn" >/dev/null || { echo "e14-run-mac.sh: $fn was not loaded from run-mac.sh's shared blocks" >&2; exit 1; }
done
die() { echo "e14-run-mac.sh: $*" >&2; exit 1; }
note() { echo "e14-run-mac.sh: $*" >&2; }
extending=0       # finish_run's extension branch is not used by E14
extend_folder=""
force=0
cooling="unstated"

# ------------------------------------------------------------------ checks

ours() { [ "$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi.e14"}}' "$1" 2>/dev/null)" = "1" ]; }
running() { [ "$(docker inspect -f '{{.State.Running}}' "$1" 2>/dev/null)" = "true" ]; }

network_up() {
  local c down=0
  ensure_colima
  for c in "${NETWORK[@]}"; do
    docker inspect "$c" >/dev/null 2>&1 || die "no container $c: set the network up first (scripts/e14-network-mac.sh setup)"
    ours "$c" || die "$c is not E14's"
    running "$c" || down=1
  done
  if [ "$down" = 1 ]; then
    note "the E14 network is stopped (Colima or the Mac restarted); starting it"
    "$repo/scripts/e14-network-mac.sh" start
    sleep 10
  fi
  for c in "${NETWORK[@]}"; do running "$c" || die "$c did not start"; done
}

preflight_e14() { # kind
  local kind="$1" out name others=0
  problems=()
  docker info >/dev/null 2>&1 || die "Docker is not reachable (is Colima running?)"
  [ "$(docker context show)" = "colima" ] || problem "docker context is not colima"
  out="$(colima status 2>&1 || true)"
  [[ "$out" == *"Virtualization.Framework"* ]] || problem "Colima is not running with --vm-type vz"
  colima list 2>/dev/null | awk 'NR==2{ if ($4 != "4" || $5 != "4GiB") exit 1 }' || problem "Colima VM is not 4 CPUs / 4 GiB"
  free_space_check "$repo" "the repository"
  # Only the network and its two chaincode containers may run.
  while read -r name; do
    [ -n "$name" ] || continue
    case "$name" in
      dev-peer0.government.uae.com-kpipayload_*|dev-peer0.general.uae.com-kpipayload_*) ;;
      *) ours "$name" || others=$((others + 1)) ;;
    esac
  done < <(docker ps --format '{{.Names}}')
  [ "$others" = 0 ] || problem "$others container(s) running besides the E14 network; none may run during E14"
  preflight_host
  preflight_verdict "$kind"
}

build_client() { # commit -> sets image_tag
  local commit="$1" tag label
  tag="kpi-fabric-client:${commit:0:12}"
  built=0
  if ! docker image inspect "$tag" >/dev/null 2>&1; then
    built=1
    note "building $tag (go vet and the client's tests run inside the build)"
    docker buildx build --builder kpi --load --progress=plain --build-arg "HARNESS_COMMIT=$commit" \
      -t "$tag" -f "$repo/fabric/Dockerfile" "$repo/fabric" >&2
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
    c="$(docker run --rm --network none "$tag" clock)"
    h1="$(now_ns)"
    off=$(( h0 / 2 + h1 / 2 - c ))
    unc=$(( (h1 - h0) / 2 ))
    if [ "$best_unc" -lt 0 ] || [ "$unc" -lt "$best_unc" ]; then best_off=$off; best_unc=$unc; fi
    : "$i"
  done
  printf '%s %s\n' "$best_off" "$best_unc"
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
    printf '    "container_mem": "no limit",\n    "container_cpuset": "all (not pinned)",\n'
    printf '    "power": %s,\n    "battery_pct": %s,\n' "$(json_str "$power")" "$(json_str "$batt")"
    printf '    "cooling": %s,\n' "$(json_str "$6")"
    printf '    "low_power_mode": %s,\n    "thermal_at_start": %s,\n' "$(json_str "$lpm")" "$(json_str "$therm")"
    printf '    "launcher": %s,\n    "native_go": %s\n' "$(json_str "$LAUNCHER")" "$(json_str "$go_v")"
    printf '  }\n}\n'
  } > "$f"
  json_get "$f" host runtime >/dev/null || die "session.json is not valid JSON"
}

# write_network FILE: the network's images (with digests), versions and chaincode.
write_network() {
  local f="$1" c img digest first=1 cc
  cc="$(docker ps --format '{{.Names}}' | sed -n 's/^dev-peer0\.general\.uae\.com-\(kpipayload_.*\)$/\1/p' | head -n 1)"
  {
    printf '{\n  "fabric_peer": %s,\n' "$(json_str "$(docker exec peer0.general.uae.com peer version 2>/dev/null | awk '/Version:/{print $2; exit}')")"
    printf '  "chaincode_package": %s,\n' "$(json_str "$cc")"
    printf '  "topology": "QChain qchain-network (commit 71a1db6): 2 organisations, 2 peers sharing one CouchDB, 1 etcdraft orderer, 3 CAs, TLS; BatchTimeout 2s, MaxMessageCount 10, PreferredMaxBytes 512 KB",\n'
    printf '  "containers": {'
    for c in "${NETWORK[@]}"; do
      img="$(docker inspect -f '{{.Config.Image}}' "$c")"
      digest="$(docker image inspect -f '{{index .RepoDigests 0}}' "$img" 2>/dev/null | cut -d@ -f2)"
      [ "$first" = 1 ] || printf ','
      first=0
      printf '\n    %s: {"image": %s, "digest": %s}' "$(json_str "$c")" "$(json_str "$img")" "$(json_str "$digest")"
    done
    printf '\n  }\n}\n'
  } > "$f"
  json_get "$f" fabric_peer >/dev/null || die "network.json is not valid JSON"
}

session_code=""
run_session() { # rid
  local rid="$1" tag persist n work offs offset unc refs p e monpid delpid cafpid started
  tag="$(docker inspect -f '{{.Config.Image}}' "kpi-run-$rid")"
  persist="$(persist_dir "$rid")"
  mkdir -p "$persist"
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
  ( while :; do deliver_chunks "$work" "$rid" "$n"; pause_bg "$persist/bg-paused-s$n.txt"; sleep 10; done ) >/dev/null 2>&1 &
  delpid=$!
  disown "$delpid"

  offs="$(measure_offset "$tag")"
  offset="${offs% *}"; unc="${offs#* }"
  note "host-to-container clock offset ${offset} ns (+/- ${unc} ns)"
  refs="$(native_refs "$work" "$persist" "$n")"
  p="${refs% *}"; e="${refs#* }"
  note "native canary references: P-core ${p}/s, E-core ${e}/s"
  write_session "$persist/session-s$n.json" "$offset" "$unc" "$p" "$e" "$cooling"
  write_network "$persist/network-s$n.json"
  docker cp "$persist/session-s$n.json" "kpi-run-$rid:/out/session.json" >/dev/null
  docker cp "$persist/network-s$n.json" "kpi-run-$rid:/out/network.json" >/dev/null

  caffeinate -i -s -w "$$" >/dev/null 2>&1 &
  cafpid=$!
  disown "$cafpid"

  trap 'note "interrupt: the run discards its current block and stops; resume with: scripts/e14-run-mac.sh resume '"$rid"'"' INT
  note "starting E14 run $rid (session $n); stop gracefully with: scripts/e14-run-mac.sh stop $rid"
  started="$(docker inspect -f '{{.State.StartedAt}}' "kpi-run-$rid")"
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

# ------------------------------------------------------------------- main

cmd="${1:-}"
[ -n "$cmd" ] || die "usage: e14-run-mac.sh start|resume|stop|status|check ..."
shift

case "$cmd" in
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
    [ -n "$plan" ] || die "start needs --plan (smoke-e14 or e14)"
    planfile="$repo/fabric/plans/$plan.json"
    [ -f "$planfile" ] || die "no plan fabric/plans/$plan.json"
    kind="$(json_get "$planfile" kind)"
    commit="$(repo_commit)"
    network_up
    build_client "$commit"
    tag="$image_tag"
    pause_early
    if [ "$built" = 1 ] && [ "$kind" != "smoke" ]; then
      note "cooling down for 120 s after the image build (background services paused)"
      sleep 120
      pause_early
    fi
    preflight_e14 "$kind"
    imageid="$(docker image inspect -f '{{.Id}}' "$tag")"
    [ -n "$seed" ] || seed="$(od -An -N8 -tu8 /dev/urandom | tr -d ' \n')"
    rid="$(date -u +%Y%m%d-%H%M)_${MACHINE}_${runtime}_${VARIANT}"
    [ ! -e "$repo/results/$osdir/$rid" ] || die "results/$osdir/$rid already exists"
    docker volume create --label org.qchain.kpi=1 --label org.qchain.kpi.e14=1 "kpi-out-$rid" >/dev/null
    docker create --init --name "kpi-run-$rid" --stop-timeout 30 \
      --label org.qchain.kpi=1 --label org.qchain.kpi.e14=1 --label "org.qchain.kpi.run=$rid" \
      --label "org.qchain.kpi.plan=$plan" --label "org.qchain.kpi.kind=$kind" --label "org.qchain.kpi.machine=$MACHINE" \
      --network "$NET" --mount "type=volume,src=$VOL,dst=/kpi,readonly" \
      --mount "type=volume,src=kpi-out-$rid,dst=/out" \
      "$tag" run --plan "/opt/kpi/plans/$plan.json" --run-id "$rid" --machine "$MACHINE" \
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
    network_up
    docker inspect "kpi-run-$rid" >/dev/null 2>&1 || die "no container kpi-run-$rid"
    ours "kpi-run-$rid" || die "kpi-run-$rid is not an E14 container"
    [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid")" = "false" ] || die "kpi-run-$rid is already running"
    kind="$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi.kind"}}' "kpi-run-$rid")"
    pause_early
    preflight_e14 "$kind"
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
  check)
    network_up
    preflight_e14 smoke
    tmp="$(mktemp "${TMPDIR:-/tmp}/kpi-net.XXXXXX")"
    write_network "$tmp"
    cat "$tmp"
    rm -f "$tmp"
    note "check done (nothing was changed; problems above would refuse a main run)"
    ;;
  status)
    docker ps -a --filter label=org.qchain.kpi.e14=1 --format 'table {{.Names}}\t{{.Status}}\t{{.Label "org.qchain.kpi.plan"}}'
    ;;
  *)
    die "unknown command $cmd"
    ;;
esac
