#!/bin/bash
# QChain KPI launcher for the University VM (Ubuntu; Docker runs on the VM's own
# kernel). The Mac has scripts/run-mac.sh and Obeid's Windows laptop
# scripts/run-windows.ps1 (one launcher per machine).
#
#   scripts/run-linux.sh start --plan NAME --machine vm-x86v2 [--variant b1-prod|b4-noossl]
#                              [--seed N] [--rebuild] [--force] [--foreground]
#   scripts/run-linux.sh resume RUN_ID [--force] [--foreground]
#   scripts/run-linux.sh extend RUN_ID --blocks ID[,ID...] [--force] [--foreground]
#   scripts/run-linux.sh stop RUN_ID        # graceful: stops after the current block
#   scripts/run-linux.sh status
#
# The launcher only prepares and supervises; all measurement and
# orchestration logic is Go inside the image (coding plan §2, §9), built from
# the same recipe as on the Mac. The container is pinned to vCPU 1 with
# --cpuset-cpus (E12 plans: the CPUs the plan names) and has NO CPU-time quota
# (--cpus; decision of 10 Oct). It:
#   - detaches from the terminal (start, resume, extend): it restarts itself in
#     a tmux session (with setsid and nohup when tmux is missing), under
#     systemd-inhibit so the VM neither sleeps nor idles during the batch, and
#     pinned to vCPU 0 with its monitor; the SSH, VS Code or AnyDesk session can
#     then be closed. --foreground keeps it in this terminal (still inhibited
#     and pinned);
#   - refuses to start on a machine that is not ready (trial and main plans;
#     smoke plans only warn), unless --force is given (recorded): no container
#     running (QChain's Fabric network, qchain-api and qchain-web-gateway are
#     stopped for the study), MySQL, IPFS and the GitHub Actions runner
#     stopped, VS Code closed (a detached trial or main run waits up to 10
#     minutes for it), the machine quiet, 10 GB free; Docker must use cgroup v2;
#   - builds the image from a clean commit with the buildx builder "kpi" and
#     labels it with that commit. The VM's liboqs builds are b1-prod and, for
#     E4, b4-noossl (decision of 21 Oct);
#   - creates the run's container and named volume, measures the host-to-
#     container clock offset and delivers it as session.json;
#   - runs the host monitor (scripts/monitor-linux.sh) and delivers its chunks
#     into the volume every 10 s, and attaches to the batch;
#   - when the batch is done: copies the results out, verifies SHA256SUMS,
#     runs the privacy scan, adds the INDEX row, and removes our container and
#     volume (only ours, found by name and label). Pushing stays manual.
#
# Paths may contain spaces: every expansion is quoted.
set -euo pipefail

LAUNCHER="run-linux.sh 1"
MACHINE=vm-x86v2
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
self="$repo/scripts/run-linux.sh"
PERIOD_S=5        # host monitor sampling period
ROWS_PER_CHUNK=12 # one chunk per minute
osdir=linux
runtime=docker
cmem=4g
plandir=plans/vm
# The launcher's notes per run (copied into the run's launcher/ folder at the
# end) and the logs of detached launchers (never copied: they name paths).
state_root="${XDG_STATE_HOME:-$HOME/.local/state}/qchain-kpi"

die() { echo "run-linux.sh: $*" >&2; exit 1; }
note() { echo "run-linux.sh: $*" >&2; }

[ "$(uname -s)" = Linux ] || die "this is the VM's launcher; the Mac has scripts/run-mac.sh, Windows scripts/run-windows.ps1"

now_ns() { date +%s%N; }

# json_get FILE KEY...: prints a value from a JSON file.
json_get() {
  python3 -I - "$@" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    d = json.load(f)
for k in sys.argv[2:]:
    d = d.get(k) if isinstance(d, dict) else None
print("" if d is None else d, end="")
PY
}

# plan_has_e4 PLANFILE: whether the plan has an E4 item.
plan_has_e4() {
  python3 -I - "$1" <<'PY'
import json, sys
with open(sys.argv[1]) as f:
    sys.exit(0 if any(i.get("exp") == "E4" for i in json.load(f)["items"]) else 1)
PY
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

problems=()
preflight_notes=()
problem() { problems+=("$1"); }

free_space_check() { # dir label
  local free_kb
  free_kb="$(df -Pk "$1" | awk 'NR==2{print $4}')"
  [ "${free_kb:-0}" -ge 10485760 ] || problem "less than 10 GB free on the disk of $2"
}

# docker_ready: Docker reachable, with cgroup v2 (the harness reads
# memory.max, cpu.stat and memory.peak).
docker_ready() {
  docker info >/dev/null 2>&1 || die "Docker is not reachable (is the docker service running, and may this user use it?)"
  [ "$(docker info -f '{{.CgroupVersion}}')" = 2 ] || die "Docker uses cgroup v1 here; the harness reads cgroup v2 files and cannot run on this machine as built"
}

# machine_load SECONDS: "busy_pct steal_pct" of the whole machine over that time.
machine_load() {
  local a b db dt ds
  read -r -a a < /proc/stat
  sleep "$1"
  read -r -a b < /proc/stat
  db=$(((b[1] + b[2] + b[3] + b[6] + b[7]) - (a[1] + a[2] + a[3] + a[6] + a[7])))
  dt=$(((b[1] + b[2] + b[3] + b[4] + b[5] + b[6] + b[7] + b[8]) - (a[1] + a[2] + a[3] + a[4] + a[5] + a[6] + a[7] + a[8])))
  ds=$((b[8] - a[8]))
  if [ "$dt" -le 0 ]; then echo "0 0"; return 0; fi
  echo "$(((100 * db + dt / 2) / dt)) $(((100 * ds + dt / 2) / dt))"
}

vscode_running() { pgrep -x code >/dev/null || pgrep -f '[.]vscode-server/' >/dev/null; }

# wait_vscode: a detached trial or main run gives VS Code up to 10 minutes to
# close (the command may have been typed into its terminal). The preflight
# then reports it if it is still running.
wait_vscode() {
  local i
  vscode_running || return 0
  note "waiting up to 10 minutes for VS Code to close (close its windows now)"
  for i in $(seq 40); do
    sleep 15
    if ! vscode_running; then
      note "VS Code has closed; letting the machine settle for 30 s"
      sleep 30
      return 0
    fi
  done
  : "$i"
}

preflight() { # kind
  local kind="$1" n load busy steal
  problems=()
  preflight_notes=()
  docker_ready
  free_space_check "$repo" "the repository"
  free_space_check "$(docker info -f '{{.DockerRootDir}}')" "Docker's data"
  n="$(docker ps -q | wc -l | tr -d ' ')"
  [ "$n" = "0" ] || problem "$n container(s) running; none may run during a batch (QChain's are stopped for the study)"
  if pgrep -x mysqld >/dev/null; then problem "MySQL is running (QChain's services are stopped for the study)"; fi
  if pgrep -x ipfs >/dev/null; then problem "IPFS is running (QChain's services are stopped for the study)"; fi
  if pgrep -x 'Runner.Listener|Runner.Worker' >/dev/null; then problem "the GitHub Actions runner is running (it is stopped for the study)"; fi
  if vscode_running; then problem "VS Code is running (close it during trial and main runs; its server on the VM ends a few minutes after the last window)"; fi
  load="$(machine_load 6)"
  busy="${load% *}"
  steal="${load#* }"
  [ "$busy" -le 10 ] || problem "the machine was ${busy}% busy over 6 s (not quiet)"
  if [ "$steal" -gt 2 ]; then note "warning: the hypervisor took ${steal}% of the CPU time over 6 s (steal); blocks above 2% are repeated"; fi
  preflight_notes+=("machine busy ${busy}%, steal ${steal}% over 6 s before the session")
  preflight_verdict "$kind"
}

# preflight_verdict KIND: reports the problems found; refuses trial and main
# runs with problems unless --force was given.
preflight_verdict() {
  local kind="$1" p
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

# variant_check VARIANT PLANFILE: the VM's liboqs builds (21 Oct), and a
# non-production build only for E4.
variant_check() {
  case "$1" in
    b1-prod | b4-noossl) ;;
    *) die "the VM runs the b1-prod build and, for E4, b4-noossl (decision of 21 Oct); got $1" ;;
  esac
  if [ "$1" != b1-prod ] && ! plan_has_e4 "$2"; then
    die "--variant $1 is for E4 plans (e4, smoke-e4); $(basename "$2" .json) has no E4 item"
  fi
}

# builder_check: the image builder of the study (created once at set-up, removed after the study).
builder_check() {
  docker buildx version >/dev/null 2>&1 || die "docker buildx is not installed"
  docker buildx inspect kpi >/dev/null 2>&1 ||
    die "no buildx builder 'kpi' on this machine; create it once with: docker buildx create --name kpi --driver docker-container"
}

built=0
build_image() { # variant commit rebuild -> sets image_tag (and built=1 if it built)
  local variant="$1" commit="$2" rebuild="$3" tag label
  tag="kpi-bench:${variant}-${commit:0:12}"
  if [ "$rebuild" = 1 ] || ! docker image inspect "$tag" >/dev/null 2>&1; then
    builder_check
    built=1
    building=1
    note "building $tag (liboqs gate and harness tests run inside the build)"
    if ! docker buildx build --builder kpi --load --progress=plain \
      --build-arg "VARIANT=$variant" --build-arg "HARNESS_COMMIT=$commit" \
      -t "$tag" -f "$repo/docker/Dockerfile" "$repo" >&2; then
      docker buildx stop kpi >/dev/null 2>&1 || true
      building=0
      die "the image build failed (see above)"
    fi
    # The builder container is stopped so it never runs alongside a measurement.
    docker buildx stop kpi >/dev/null 2>&1 || true
    building=0
  fi
  label="$(docker image inspect -f '{{index .Config.Labels "org.qchain.kpi.harness-commit"}}' "$tag")"
  [ "$label" = "$commit" ] || die "image $tag is labelled with commit $label, not $commit"
  image_tag="$tag"
}

# relaunch ARGS...: runs this command again under systemd-inhibit (sleep and
# idle blocked) and pinned to vCPU 0: in a tmux session (setsid and nohup
# without tmux) that outlives the terminal, or here with --foreground.
relaunch() {
  local inhibit=() sleep_state stamp wrapper log name line
  [ "${KPI_RELAUNCHED:-0}" = 1 ] && return 0
  if systemd-inhibit --what=sleep:idle --who=qchain-kpi --why=check --mode=block true >/dev/null 2>&1; then
    inhibit=(systemd-inhibit --what=sleep:idle --who=qchain-kpi "--why=QChain KPI benchmark run" --mode=block)
    sleep_state="sleep and idle blocked by systemd-inhibit"
  else
    note "warning: systemd-inhibit was refused; sleep and idle are not blocked during this run (the VM does not sleep on its own)"
    sleep_state="sleep and idle not blocked (systemd-inhibit was refused)"
  fi
  if [ "$foreground" = 1 ]; then
    exec env KPI_RELAUNCHED=1 "KPI_SLEEP=$sleep_state" "${inhibit[@]}" taskset -c 0 "$self" "$@"
  fi
  mkdir -p "$state_root"
  stamp="$(date -u +%Y%m%d-%H%M%S)"
  log="$state_root/launcher-$stamp.log"
  wrapper="$state_root/launcher-$stamp.sh"
  line="$(printf '%q ' env KPI_RELAUNCHED=1 "KPI_SLEEP=$sleep_state" "${inhibit[@]}" taskset -c 0 "$self" "$@")"
  {
    echo '#!/bin/bash'
    # tee ignores Ctrl-C, so the launcher can still finish after an interrupt.
    printf '%s 2>&1 | tee -i -a %q\n' "$line" "$log"
    printf 'echo; echo %q\nread -r _\n' "run-linux.sh has finished (log: $log). Press Enter to close this window."
  } > "$wrapper"
  if command -v tmux >/dev/null 2>&1; then
    name="kpi-$stamp"
    tmux new-session -d -s "$name" "bash $(printf '%q' "$wrapper")"
    note "started in the tmux session $name; watch it with: tmux attach -t $name (leave it running with Ctrl-B, then D)"
  else
    setsid nohup bash "$wrapper" >/dev/null 2>&1 < /dev/null &
    note "started in the background (tmux is not installed); follow it with: tail -f $log"
  fi
  note "closing this terminal, VS Code or AnyDesk does not stop it; log: $log"
  exit 0
}

# ----------------------------------------------------------------- session

measure_offset() { # tag -> "offset_ns uncertainty_ns" (host clock minus container clock)
  local tag="$1" i h0 h1 c best_off=0 best_unc=-1 off unc
  for i in 1 2 3; do
    h0="$(now_ns)"
    c="$(docker run --rm --network none --label org.qchain.kpi=1 --entrypoint /opt/kpi/bin/kpi-batch "$tag" clock)"
    h1="$(now_ns)"
    off=$((h0 / 2 + h1 / 2 - c))
    unc=$(((h1 - h0) / 2))
    if [ "$best_unc" -lt 0 ] || [ "$unc" -lt "$best_unc" ]; then best_off=$off; best_unc=$unc; fi
    : "$i"
  done
  printf '%s %s\n' "$best_off" "$best_unc"
}

write_session() { # file offset unc
  local f="$1" ncpu mem os kernel cpu docker_v
  ncpu="$(getconf _NPROCESSORS_ONLN)"
  mem=$(($(awk '/^MemTotal:/{print $2}' /proc/meminfo) / 1048576))
  # shellcheck source=/dev/null
  os="$(. /etc/os-release && printf '%s' "${PRETTY_NAME:-Linux}")"
  kernel="$(uname -r)"
  cpu="$(awk -F': ' '/^model name/{print $2; exit}' /proc/cpuinfo)"
  docker_v="$(docker version --format '{{.Server.Version}}')"
  {
    printf '{\n'
    printf '  "host_offset_ns": %s,\n  "host_offset_unc_ns": %s,\n' "$2" "$3"
    printf '  "native_p_rate": 0,\n  "native_e_rate": 0,\n'
    printf '  "host_logical_cpus": %s,\n  "host_monitor_period_s": %s,\n' "$ncpu" "$PERIOD_S"
    printf '  "host": {\n'
    printf '    "runtime": %s,\n' "$(json_str "$runtime")"
    printf '    "host_os": %s,\n' "$(json_str "$os (kernel $kernel)")"
    printf '    "host_cpu": %s,\n' "$(json_str "$cpu")"
    printf '    "host_mem_gb": %s,\n' "$(json_str "$mem")"
    printf '    "docker": %s,\n' "$(json_str "$docker_v")"
    printf '    "container_mem": %s,\n    "container_cpuset": %s,\n' "$(json_str "$cmem")" "$(json_str "${container_cpuset:-1}")"
    printf '    "launcher": %s\n' "$(json_str "$LAUNCHER")"
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

persist_dir() { printf '%s\n' "$state_root/$1"; }

# The monitor and the delivery loop end with the launcher on every exit path,
# and an interrupted image build does not leave the builder running.
monpid=""
delpid=""
building=0
cleanup() {
  if [ -n "$delpid" ]; then kill "$delpid" 2>/dev/null || true; fi
  if [ -n "$monpid" ]; then kill "$monpid" 2>/dev/null || true; fi
  if [ "$building" = 1 ]; then docker buildx stop kpi >/dev/null 2>&1 || true; fi
}
trap cleanup EXIT
trap 'cleanup; exit 129' HUP
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

# run_session RID: one start or resume, from session.json to the batch's exit.
# Sets session_code to the batch's exit code.
session_code=""
extending=0      # 1 while extending a finished run (see the extend command)
extend_folder="" # results folder of the run being extended
run_session() {
  local rid="$1" tag persist n work offs offset unc started
  tag="$(docker inspect -f '{{.Config.Image}}' "kpi-run-$rid")"
  persist="$(persist_dir "$rid")"
  mkdir -p "$persist"
  n=$(($(find "$persist" -maxdepth 1 -name 'session-s*.json' | wc -l) + 1))
  work="$(mktemp -d "${TMPDIR:-/tmp}/kpi-mon.XXXXXX")"
  {
    if [ "${#problems[@]}" -gt 0 ]; then printf '%s\n' "${problems[@]}"; fi
    if [ "$force" = 1 ]; then echo "forced"; fi
    if [ "${#preflight_notes[@]}" -gt 0 ]; then printf 'note: %s\n' "${preflight_notes[@]}"; fi
  } > "$persist/preflight-s$n.txt"
  printf '%s\n' "${KPI_SLEEP:-unknown}" > "$persist/sleep-s$n.txt"

  # The host monitor starts first, so it is delivering before the batch's first block.
  : > "$work/empty.ok"
  taskset -c 0 nice -n 19 "$repo/scripts/monitor-linux.sh" "$work" "$PERIOD_S" "$ROWS_PER_CHUNK" "$$" >/dev/null &
  monpid=$!
  (while kill -0 "$$" 2>/dev/null; do
    deliver_chunks "$work" "$rid" "$n"
    sleep 10
  done) >/dev/null 2>&1 &
  delpid=$!
  disown "$delpid"

  offs="$(measure_offset "$tag")"
  offset="${offs% *}"
  unc="${offs#* }"
  note "host-to-container clock offset ${offset} ns (+/- ${unc} ns)"
  container_cpuset="$(docker inspect -f '{{.HostConfig.CpusetCpus}}' "kpi-run-$rid")"
  write_session "$persist/session-s$n.json" "$offset" "$unc"
  docker cp "$persist/session-s$n.json" "kpi-run-$rid:/out/session.json" >/dev/null

  trap 'note "interrupt: the batch discards its current block and stops; resume with: scripts/run-linux.sh resume '"$rid"'"' INT
  note "starting batch $rid (session $n); stop gracefully with: scripts/run-linux.sh stop $rid"
  started="$(docker inspect -f '{{.State.StartedAt}}' "kpi-run-$rid")"
  # -a attaches the batch's output and forwards signals (Ctrl-C) to it.
  docker start -a "kpi-run-$rid" >&2 || true
  while [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid")" = "true" ]; do sleep 2; done
  trap 'cleanup; exit 130' INT
  if [ "$(docker inspect -f '{{.State.StartedAt}}' "kpi-run-$rid")" = "$started" ]; then
    session_code=start-failed
  else
    session_code="$(docker inspect -f '{{.State.ExitCode}}' "kpi-run-$rid")"
  fi

  kill "$delpid" 2>/dev/null || true
  delpid=""
  kill "$monpid" 2>/dev/null || true
  wait "$monpid" 2>/dev/null || true
  monpid=""
  deliver_chunks "$work" "$rid" "$n"
  cp "$work/host-all.csv" "$persist/host-monitor-s$n.csv" 2>/dev/null || true
  offs="$(measure_offset "$tag")"
  printf 'start %s %s\nend %s\n' "$offset" "$unc" "$offs" > "$persist/clock-offset-s$n.txt"
  rm -rf "$work"
}

# ----------------------------------------------------------------- finish

privacy_patterns() {
  {
    hostname
    hostname -s 2>/dev/null || true
    hostname -f 2>/dev/null || true
    id -un
    getent passwd "$(id -un)" 2>/dev/null | cut -d: -f5 | cut -d, -f1 || true
    printf '%s\n' "$HOME"
    ip -o addr 2>/dev/null | awk '{split($4, a, "/"); print a[1]}' | grep -v -e '^127\.' -e '^::1$' || true
    cat /sys/class/net/*/address 2>/dev/null | grep -v '^00:00:00:00:00:00$' || true
    cat /etc/machine-id 2>/dev/null || true
    # Only a numeric ID: any message printed instead would match ordinary words.
    timeout 5 anydesk --get-id 2>/dev/null | grep -E -x '[0-9]{9,10}' || true
  } | grep -v -i -x -e 'localhost' -e 'localhost[.]localdomain' | awk 'length($0) >= 4' | sort -u
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
      note "batch stopped and can be resumed: scripts/run-linux.sh resume $rid"
      return 0 ;;
    *)
      stage="$repo/scratch/failed/$rid"
      rm -rf "$stage"
      mkdir -p "$stage"
      docker cp "kpi-run-$rid:/out/." "$stage/" >/dev/null
      docker logs "kpi-run-$rid" > "$stage/console.txt" 2>&1 || true
      note "batch ended with exit code $code; its files were copied to scratch/failed/$rid for inspection"
      note "the container kpi-run-$rid and volume kpi-out-$rid were kept"
      return 1 ;;
  esac
  stage="$repo/scratch/staging/$rid"
  dest="$repo/results/$osdir/$rid"
  if [ "$extending" != 1 ]; then [ ! -e "$dest" ] || die "$dest already exists"; fi
  rm -rf "$stage"
  mkdir -p "$stage"
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
  (cd "$stage" && sha256sum -c --quiet SHA256SUMS) || die "SHA256SUMS verification failed in scratch/staging/$rid (container kept)"
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

# create_container RID PLAN KIND CPUSET TAG SEED IMAGEID [EXTRA...]: the batch's
# container and named volume (extra arguments go to kpi-batch run).
create_container() {
  local rid="$1" plan="$2" kind="$3" cpuset="$4" tag="$5" seed="$6" imageid="$7"
  shift 7
  docker volume create --label org.qchain.kpi=1 "kpi-out-$rid" >/dev/null
  docker create --init --name "kpi-run-$rid" --stop-timeout 30 \
    --label org.qchain.kpi=1 --label "org.qchain.kpi.run=$rid" --label "org.qchain.kpi.plan=$plan" \
    --label "org.qchain.kpi.kind=$kind" --label "org.qchain.kpi.machine=$MACHINE" \
    --network none --cpuset-cpus="$cpuset" --memory="$cmem" --memory-swap="$cmem" -e GOMAXPROCS=1 \
    --mount "type=volume,src=kpi-out-$rid,dst=/out" \
    "$tag" run --plan "/opt/kpi/$plandir/$plan.json" --run-id "$rid" --machine "$MACHINE" \
    --seed "$seed" --image-id "$imageid" "$@" >/dev/null
}

# ------------------------------------------------------------------- main

cmd="${1:-}"
[ -n "$cmd" ] || die "usage: run-linux.sh start|resume|extend|stop|status ..."
shift
args=("$@")
force=0
foreground=0

case "$cmd" in
  start)
    plan="" machine="" variant="b1-prod" seed="" rebuild=0
    while [ $# -gt 0 ]; do
      case "$1" in
        --plan) plan="$2"; shift 2 ;;
        --machine) machine="$2"; shift 2 ;;
        --variant) variant="$2"; shift 2 ;;
        --seed) seed="$2"; shift 2 ;;
        --rebuild) rebuild=1; shift ;;
        --force) force=1; shift ;;
        --foreground) foreground=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    [ -n "$plan" ] && [ -n "$machine" ] || die "start needs --plan and --machine"
    [ "$machine" = "$MACHINE" ] || die "this launcher runs on $MACHINE only (got $machine)"
    planfile="$repo/$plandir/$plan.json"
    [ -f "$planfile" ] || die "no plan $plandir/$plan.json"
    variant_check "$variant" "$planfile"
    kind="$(json_get "$planfile" kind)"
    cpuset="$(plan_cpuset "$planfile")"
    commit="$(repo_commit)"
    docker_ready
    if [ "$rebuild" = 1 ] || ! docker image inspect "kpi-bench:${variant}-${commit:0:12}" >/dev/null 2>&1; then builder_check; fi
    relaunch start "${args[@]}"
    build_image "$variant" "$commit" "$rebuild"
    tag="$image_tag"
    if [ "$built" = 1 ] && [ "$kind" != "smoke" ]; then
      note "cooling down for 120 s after the image build"
      sleep 120
    fi
    if [ "$kind" != "smoke" ] && [ "$foreground" = 0 ]; then wait_vscode; fi
    preflight "$kind"
    imageid="$(docker image inspect -f '{{.Id}}' "$tag")"
    [ -n "$seed" ] || seed="$(od -An -N8 -tu8 /dev/urandom | tr -d ' \n')"
    rid="$(date -u +%Y%m%d-%H%M)_${machine}_${runtime}_${variant}"
    [ ! -e "$repo/results/$osdir/$rid" ] || die "results/$osdir/$rid already exists"
    create_container "$rid" "$plan" "$kind" "$cpuset" "$tag" "$seed" "$imageid"
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
        --force) force=1; shift ;;
        --foreground) foreground=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    docker_ready
    docker inspect "kpi-run-$rid" >/dev/null 2>&1 || die "no container kpi-run-$rid"
    [ "$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi"}}' "kpi-run-$rid")" = "1" ] || die "kpi-run-$rid is not a KPI container"
    [ "$(docker inspect -f '{{.State.Running}}' "kpi-run-$rid")" = "false" ] || die "kpi-run-$rid is already running"
    kind="$(docker inspect -f '{{index .Config.Labels "org.qchain.kpi.kind"}}' "kpi-run-$rid")"
    relaunch resume "${args[@]}"
    if [ "$kind" != "smoke" ] && [ "$foreground" = 0 ]; then wait_vscode; fi
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
        --force) force=1; shift ;;
        --foreground) foreground=1; shift ;;
        *) die "unknown option $1" ;;
      esac
    done
    [ -n "$blocks" ] || die "extend needs --blocks ID[,ID...] (for example E1.hqc-3.encaps.r3)"
    folder="$repo/results/$osdir/$rid"
    [ -d "$folder" ] || die "no results folder $folder"
    (cd "$folder" && sha256sum -c --quiet SHA256SUMS) || die "the SHA256SUMS of $rid does not verify"
    [ "$(json_get "$folder/manifest.json" status)" = "done" ] || die "only a finished run can be extended"
    kind="$(json_get "$folder/manifest.json" plan kind)"
    [ "$kind" = "main" ] || die "only main runs can be extended (this one is $kind)"
    plan="$(json_get "$folder/manifest.json" plan name)"
    cpuset="$(plan_cpuset "$repo/$plandir/$plan.json")"
    seed="$(json_get "$folder/manifest.json" seed)"
    machine="$(json_get "$folder/manifest.json" machine)"
    variant="$(json_get "$folder/manifest.json" build variant)"
    old_commit="$(json_get "$folder/manifest.json" harness_commit)"
    [ "$machine" = "$MACHINE" ] || die "this launcher runs on $MACHINE only"
    commit="$(repo_commit)"
    git -C "$repo" cat-file -e "$old_commit^{commit}" 2>/dev/null || die "the run's harness commit $old_commit is not in this repository"
    # Everything that produces the numbers must be identical. The orchestrator
    # (cmd/kpi-batch) may differ, because it is the part that implements the
    # extension; its changed files are recorded in the run's manifest.
    if ! git -C "$repo" diff --quiet "$old_commit" -- cmd/kpi-prim cmd/kpi-env cmd/kpi-canary-native internal csrc docker "$plandir" config go.mod go.sum; then
      die "the measuring sources (cmd/kpi-prim, cmd/kpi-env, cmd/kpi-canary-native, internal, csrc, docker, $plandir, config, go.mod) changed since $old_commit; the run cannot be extended"
    fi
    docker_ready
    ! docker inspect "kpi-run-$rid" >/dev/null 2>&1 || die "container kpi-run-$rid already exists"
    docker image inspect "kpi-bench:${variant}-${commit:0:12}" >/dev/null 2>&1 || builder_check
    relaunch extend "${args[@]}"
    orch_changed="$(git -C "$repo" diff --name-only "$old_commit" -- cmd/kpi-batch | tr '\n' ',' | sed 's/,$//')"
    note "measuring sources are identical to those of the run ($old_commit); orchestrator files that differ: ${orch_changed:-none}"
    mkdir -p "$repo/scratch/backup"
    rm -rf "$repo/scratch/backup/$rid"
    cp -R "$folder" "$repo/scratch/backup/$rid"
    note "backup of the finished run: scratch/backup/$rid"
    build_image "$variant" "$commit" 0
    tag="$image_tag"
    if [ "$built" = 1 ]; then
      note "cooling down for 120 s after the image build"
      sleep 120
    fi
    if [ "$foreground" = 0 ]; then wait_vscode; fi
    preflight "$kind"
    imageid="$(docker image inspect -f '{{.Id}}' "$tag")"
    create_container "$rid" "$plan" "$kind" "$cpuset" "$tag" "$seed" "$imageid" \
      --extend "$blocks" --allow-rebuild --orchestrator-changed "$orch_changed"
    # The finished run goes back into the volume (without the launcher's own files,
    # which are re-added at the end) and the volume is handed to the unprivileged user.
    prep="$(mktemp -d "${TMPDIR:-/tmp}/kpi-ext.XXXXXX")"
    cp -R "$folder/." "$prep/"
    rm -rf "$prep/launcher" "$prep/console.txt"
    docker cp "$prep/." "kpi-run-$rid:/out/" >/dev/null
    rm -rf "$prep"
    docker run --rm --network none --label org.qchain.kpi=1 --user root --entrypoint /bin/chown \
      --mount "type=volume,src=kpi-out-$rid,dst=/out" "$tag" -R kpi:kpi /out
    persist="$(persist_dir "$rid")"
    rm -rf "$persist"
    mkdir -p "$persist"
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
