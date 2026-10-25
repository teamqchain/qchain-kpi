#!/bin/bash
# Test of scripts/run-linux.sh against the fake docker, as an ordinary user in a
# throwaway Ubuntu container. Prints PASS or FAIL per check.
set -uo pipefail
R="$HOME/Desktop/QChain/qchain-kpi"
L="$R/scripts/run-linux.sh"
ST="$HOME/.local/state/qchain-kpi"
fails=0
pass() { echo "PASS $*"; }
fail() { echo "FAIL $*"; fails=$((fails + 1)); }
check() { # description command... (returns 1 on failure, so a log can follow)
  local d="$1"
  shift
  if "$@" >/dev/null 2>&1; then pass "$d"; else fail "$d"; return 1; fi
}
expect_fail() { # description pattern command...
  local d="$1" pat="$2" out code
  shift 2
  out="$("$@" 2>&1)"
  code=$?
  if [ "$code" != 0 ] && grep -q -- "$pat" <<< "$out"; then pass "$d"; else fail "$d (exit $code): $out"; fi
}
rid_of() { sed -n 's/.*done: results\/linux\/\([^ ]*\) .*/\1/p' <<< "$1" | tail -1; }
# newest_run: the run id of the newest container the fake docker holds.
newest_run() {
  local d last=""
  for d in "$FAKE"/containers/kpi-run-*; do
    [ -e "$d" ] && last="$d"
  done
  [ -z "$last" ] || basename "$last" | sed 's/^kpi-run-//'
}
# Run ids have minute resolution, so a new run starts in a new minute.
last_min=""
fresh_minute() { while [ "$(date -u +%Y%m%d-%H%M)" = "$last_min" ]; do sleep 1; done; last_min="$(date -u +%Y%m%d-%H%M)"; }
show() { echo "---- tail of $1"; tail -n 25 "$1"; echo "----"; }

cd "$R" || exit 1
git init -q && git add -A && git -c user.name=t -c user.email=t@example.invalid commit -q -m test || exit 1
commit="$(git log -1 --format=%H)"

echo "== refusals"
expect_fail "no command" "usage" "$L"
expect_fail "start without --machine" "needs --plan and --machine" "$L" start --plan smoke
expect_fail "another machine's id" "runs on vm-x86v2 only" "$L" start --plan smoke --machine mac-m1
expect_fail "unknown plan" "no plan plans/vm/nope.json" "$L" start --plan nope --machine vm-x86v2
expect_fail "a Mac-only Mac plan name" "no plan plans/vm/e3.json" "$L" start --plan e3 --machine vm-x86v2
expect_fail "b2-portable refused on the VM" "the VM runs the b1-prod build" "$L" start --plan e4 --machine vm-x86v2 --variant b2-portable
expect_fail "b4-noossl for a plan without E4" "is for E4 plans" "$L" start --plan smoke --machine vm-x86v2 --variant b4-noossl
expect_fail "cgroup v1" "cgroup v1" env FAKE_CGROUP=1 "$L" start --plan smoke --machine vm-x86v2
expect_fail "no buildx builder" "docker buildx create --name kpi" env FAKE_NO_BUILDER=1 "$L" start --plan smoke --machine vm-x86v2
echo x >> README.md
expect_fail "uncommitted changes" "uncommitted changes" "$L" start --plan smoke --machine vm-x86v2
git checkout -q README.md
expect_fail "image build failure" "the image build failed" env FAKE_BUILD_FAIL=1 "$L" start --plan smoke --machine vm-x86v2 --foreground
check "refused starts created no container" test -z "$(ls "$FAKE/containers")"
check "the builder was stopped after the failed build" grep -q "docker buildx stop kpi" "$FAKE/calls.log"

echo "== foreground smoke run (70 s, one monitor chunk delivered during the batch)"
fresh_minute
out="$(env FAKE_BATCH_S=70 "$L" start --plan smoke --machine vm-x86v2 --foreground 2>&1)"
code=$?
echo "$out" > /tmp/run-smoke.log
rid="$(rid_of "$out")"
d="$R/results/linux/$rid"
check "smoke run finished (exit $code, $rid)" test "$code" = 0 -a -n "$rid" -a -f "$d/manifest.json" || show /tmp/run-smoke.log
check "systemd-inhibit refusal reported" grep -q "systemd-inhibit was refused" <<< "$out"
check "run id format" grep -E -q '^[0-9]{8}-[0-9]{4}_vm-x86v2_docker_b1-prod$' <<< "$rid"
check "image built once with the harness commit" grep -q "buildx build --builder kpi --load --progress=plain --build-arg VARIANT=b1-prod --build-arg HARNESS_COMMIT=$commit -t kpi-bench:b1-prod-${commit:0:12}" "$FAKE/calls.log"
check "container flags" grep -q -- "create --init --name kpi-run-$rid --stop-timeout 30 .* --network none --cpuset-cpus=1 --memory=4g --memory-swap=4g -e GOMAXPROCS=1 --mount type=volume,src=kpi-out-$rid,dst=/out kpi-bench:b1-prod-${commit:0:12} run --plan /opt/kpi/plans/vm/smoke.json --run-id $rid --machine vm-x86v2 --seed [0-9]* --image-id sha256:" "$FAKE/calls.log"
check "clock containers carry our label" grep -q -- "run --rm --network none --label org.qchain.kpi=1 --entrypoint /opt/kpi/bin/kpi-batch kpi-bench:b1-prod-${commit:0:12} clock" "$FAKE/calls.log"
check "session.json valid" python3 -I -c 'import json,sys; json.load(open(sys.argv[1]))' "$d/launcher/session-s1.json"
check "session.json facts" python3 -I - "$d/launcher/session-s1.json" <<'PY'
import json, os, sys
s = json.load(open(sys.argv[1]))
allowed = {"runtime", "host_os", "host_cpu", "host_mem_gb", "docker", "colima", "lima", "wsl", "docker_desktop", "vm_cpus", "vm_mem_gb",
           "container_mem", "container_cpuset", "power", "battery_pct", "cooling", "launcher", "native_go", "low_power_mode", "thermal_at_start"}
h = s["host"]
assert set(h) <= allowed, set(h) - allowed
assert all(len(v) <= 120 and "\n" not in v for v in h.values())
assert h["runtime"] == "docker" and h["container_mem"] == "4g" and h["container_cpuset"] == "1" and h["launcher"] == "run-linux.sh 1"
assert h["host_os"].startswith("Ubuntu") and "(kernel " in h["host_os"] and h["docker"] == "29.3.1"
assert s["host_logical_cpus"] == os.cpu_count() and s["host_monitor_period_s"] == 5 and s["host_offset_unc_ns"] > 0
assert s["native_p_rate"] == 0 and s["native_e_rate"] == 0
PY
check "sleep state recorded" grep -q "not blocked" "$d/launcher/sleep-s1.txt"
check "preflight notes recorded" grep -q "^note: machine busy" "$d/launcher/preflight-s1.txt"
check "clock offsets at start and end" grep -q "^end -\?[0-9]" "$d/launcher/clock-offset-s1.txt"
check "monitor rows (>= 12, no CPU/thermal/power, no alert)" python3 -I - "$d/launcher/host-monitor-s1.csv" <<'PY'
import csv, sys
rows = list(csv.reader(open(sys.argv[1])))
assert rows[0] == ["utc_ns", "other_pct", "throttle", "on_ac", "alert", "detail"], rows[0]
data = rows[1:]
assert len(data) >= 12, len(data)
for r in data:
    assert len(r) == 6 and r[1:4] == ["-1", "-1", "-1"] and r[4] == "0" and "found=none" in r[5], r
    int(r[0])
PY
check "a chunk was delivered during the batch" test -f "$d/monitor/host-s1-000001.csv" -a -f "$d/monitor/host-s1-000001.ok"
check "INDEX row" grep -q "^| $rid | smoke | smoke | done | 3 | 0 | ${commit:0:12} |" "$R/results/linux/INDEX.md"
check "container and volume removed" test ! -e "$FAKE/containers/kpi-run-$rid" -a ! -e "$FAKE/volumes/kpi-out-$rid"
check "launcher notes folder removed" test ! -e "$ST/$rid"
check "monitor ended" bash -c '! pgrep -f "[m]onitor-linux.sh"'
check "no launcher log in the results" bash -c "! ls '$d/launcher' | grep -q launcher-"
check "no user name or home in the results" bash -c "! grep -r -i -q -e '$(id -un)' -e '$HOME' '$d'"

echo "== detached start in tmux (E12 smoke, CPUs 0-1)"
fresh_minute
out="$(env FAKE_BATCH_S=15 "$L" start --plan smoke-e12-vm --machine vm-x86v2 2>&1)"
code=$?
check "returns at once, in tmux (exit $code)" grep -q "started in the tmux session kpi-" <<< "$out"
log="$(sed -n 's/.*; log: \(.*\)$/\1/p' <<< "$out")"
for _ in $(seq 120); do grep -q "done: results/linux/" "$log" 2>/dev/null && break; sleep 1; done
rid2="$(rid_of "$(cat "$log")")"
check "detached run finished ($rid2)" test -n "$rid2" -a -f "$R/results/linux/$rid2/manifest.json"
check "E12 container on CPUs 0-1" grep -q -- "--name kpi-run-$rid2 .*--cpuset-cpus=0-1 " "$FAKE/calls.log"
check "session.json names CPUs 0-1" grep -q '"container_cpuset": "0-1"' "$R/results/linux/$rid2/launcher/session-s1.json"
check "launcher pinned to vCPU 0" grep -q "taskset -c 0" "$ST"/launcher-*.sh
pane=""
for _ in $(seq 10); do
  pane="$(tmux capture-pane -p -t "$(tmux ls -F '#S' | head -1)" 2>&1)"
  grep -q "has finished" <<< "$pane" && break
  sleep 1
done
check "the tmux window stays open with the final message" grep -q "has finished" <<< "$pane" || { echo "---- pane"; echo "$pane"; tmux ls; echo "----"; }
tmux kill-server 2>/dev/null

echo "== stop and resume (trial)"
fresh_minute
(sleep 20; "$L" stop "$(newest_run)") > /tmp/stop.log 2>&1 &
out="$(env FAKE_BATCH_S=60 "$L" start --plan trial --machine vm-x86v2 --foreground 2>&1)"
code=$?
echo "$out" > /tmp/run-trial.log
wait
rid3="$(newest_run)"
check "stop requested" grep -q "requested a graceful stop of $rid3" /tmp/stop.log
check "stopped run reported resumable (exit $code)" grep -q "batch stopped and can be resumed: scripts/run-linux.sh resume $rid3" <<< "$out"
check "trial preflight strict and passed" bash -c "! grep -q 'preflight failed' <<< \"\$1\"" _ "$out"
check "container kept, exit 5" test "$(cat "$FAKE/containers/kpi-run-$rid3/exitcode")" = 5
out="$(env FAKE_BATCH_S=5 "$L" resume "$rid3" --foreground 2>&1)"
code=$?
echo "$out" > /tmp/run-resume.log
check "resumed run finished (exit $code)" test -n "$rid3" -a "$(rid_of "$out")" = "$rid3" || show /tmp/run-resume.log
check "two sessions recorded" test -f "$R/results/linux/$rid3/launcher/session-s2.json" -a -f "$R/results/linux/$rid3/launcher/preflight-s2.txt" -a -f "$R/results/linux/$rid3/launcher/host-monitor-s2.csv"
check "manifest says 2 sessions" grep -q '"sessions":2' "$R/results/linux/$rid3/manifest.json"
expect_fail "resume of a finished run" "no container kpi-run-$rid3" "$L" resume "$rid3"

echo "== extend (main run)"
fresh_minute
out="$(env FAKE_BATCH_S=5 "$L" start --plan e10 --machine vm-x86v2 --foreground 2>&1)"
echo "$out" > /tmp/run-main.log
rid4="$(rid_of "$out")"
check "main run finished ($rid4)" test -n "$rid4" || show /tmp/run-main.log
out="$(env FAKE_BATCH_S=5 "$L" extend "$rid4" --blocks E10.ml-dsa-44.sign.r1 --foreground 2>&1)"
code=$?
echo "$out" > /tmp/run-extend.log
check "extension finished (exit $code)" test -n "$rid4" -a "$(rid_of "$out")" = "$rid4" || show /tmp/run-extend.log
check "measuring sources compared" grep -q "measuring sources are identical" <<< "$out"
check "batch got the extension arguments" grep -q -- "--extend E10.ml-dsa-44.sign.r1 --allow-rebuild --orchestrator-changed" "$FAKE/calls.log"
check "volume handed to the batch user" grep -q -- "--entrypoint /bin/chown --mount type=volume,src=kpi-out-$rid4,dst=/out" "$FAKE/calls.log"
check "manifest of the extended run" grep -q '"extended":true' "$R/results/linux/$rid4/manifest.json"
check "session 2 in the launcher notes" test -f "$R/results/linux/$rid4/launcher/session-s2.json" -a -f "$R/results/linux/$rid4/launcher/session-s1.json"
check "console keeps both sessions" grep -q "=== extension session" "$R/results/linux/$rid4/console.txt"
check "one INDEX row for the run" test "$(grep -c "^| $rid4 |" "$R/results/linux/INDEX.md")" = 1
check "backup of the finished run" test -f "$R/scratch/backup/$rid4/manifest.json"
expect_fail "extend of a smoke run" "only main runs can be extended" "$L" extend "$rid" --blocks x
expect_fail "extend without blocks" "extend needs --blocks" "$L" extend "$rid4"

echo "== privacy scan catches the host name"
fresh_minute
out="$(env FAKE_BATCH_S=3 FAKE_LEAK=1 "$L" start --plan smoke --machine vm-x86v2 --foreground 2>&1)"
code=$?
echo "$out" > /tmp/run-leak.log
rid5="$(newest_run)"
check "scan failed the run (exit $code)" grep -q "privacy scan: identifying text in blocks.csv" <<< "$out" || show /tmp/run-leak.log
check "nothing moved into results" test ! -e "$R/results/linux/$rid5" -a -d "$R/scratch/staging/$rid5"
check "container kept for inspection" test -d "$FAKE/containers/kpi-run-$rid5"
check "status lists it" bash -c "'$L' status | grep -q kpi-run-$rid5"
expect_fail "stop of a run that is not running" "is not running" "$L" stop "$rid5"

echo "== the monitor's alert for containers that are not ours"
# The first live check on the VM showed that a bare empty label line was dropped:
# 11 running QChain containers were not counted.
monrun() { # foreign-count dir
  rm -rf "$2"
  env FAKE_FOREIGN="$1" "$R/scripts/monitor-linux.sh" "$2" 1 2 $$ > /dev/null 2>&1 &
  local m=$!
  sleep 4
  kill "$m" 2>/dev/null
  wait "$m" 2>/dev/null
  true
}
monrun 3 /tmp/mon-foreign
check "foreign containers counted" grep -q 'containers_not_ours=3' /tmp/mon-foreign/host-all.csv
check "foreign containers raise the alert" grep -q ',-1,-1,-1,1,.*containers_not_ours=3' /tmp/mon-foreign/host-all.csv
monrun 0 /tmp/mon-none
check "no foreign container: no alert" grep -q ',-1,-1,-1,0,.*found=none' /tmp/mon-none/host-all.csv

echo "== Ctrl-C during an image build stops the builder"
from=$(($(wc -l < "$FAKE/calls.log") + 1))
# Job control, so the background launcher keeps SIGINT and has its own process group.
set -m
env FAKE_BUILD_S=20 "$L" start --plan smoke --machine vm-x86v2 --rebuild --foreground > /tmp/run-int.log 2>&1 &
pid=$!
sleep 6
kill -INT -- "-$pid"
wait "$pid"
code=$?
set +m
check "interrupted launcher exits with 130 (got $code)" test "$code" = 130 || show /tmp/run-int.log
check "builder stopped after the interrupted build" bash -c "tail -n +$from '$FAKE/calls.log' | sed -n '/buildx build/,\$p' | grep -q 'buildx stop kpi'"
check "no container created" test -z "$(find "$FAKE/containers" -maxdepth 1 -newer /tmp/run-int.log -name 'kpi-run-*')"

echo
if [ "$fails" = 0 ]; then echo "ALL PASS"; else echo "$fails FAILED"; fi
exit "$fails"
