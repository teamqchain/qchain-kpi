# qchain-kpi: performance benchmarks of post-quantum cryptography for QChain

This repository holds the harness and the results of the QChain KPI study. It measures the 30 algorithms QChain could use, the way QChain's backend and wallet use them, on three machines:
- 25 post-quantum algorithms from liboqs 0.16.0;
- 5 classical baselines through OpenSSL 3.0.

It records every sample, flags noisy measurements and repeats them, and produces data the paper is written from.

- **Research plan** (what is measured and why): [docs/plan.md](docs/plan.md)
- **Coding plan** (how the harness is built): [docs/coding-plan.md](docs/coding-plan.md)
- **What the code does when it measures:** [docs/methodology.md](docs/methodology.md)
- **Every output file and column:** [docs/data-dictionary.md](docs/data-dictionary.md)
- **Where every input comes from** (QChain code copied verbatim, pinned libraries, test vectors): [docs/provenance.md](docs/provenance.md)
- **Decisions taken while building the harness:** [docs/decisions-log.md](docs/decisions-log.md)

**Status (25 Oct 2026).** Every Mac experiment is done and committed (E0 to E14). The VM's and the Windows laptop's launchers (`scripts/run-linux.sh`, `scripts/run-windows.ps1`) and their plans (`plans/vm/`, `plans/windows/`, the scope of 21 Oct in `docs/methodology.md` section 9) are coded; their runs come next.

**22 Oct:** Mac results: the trial, E1 to E13 (all committed; E4 with `b2-portable` and `b3-native`, E5 natively). E14 (Fabric latency against payload size, on QChain's network rebuilt on the Mac) is coded.

**19 Oct:** Mac results: the trial, E1, E10, E2, E3, E6, E7, E8, E9, E11, E12 and E13 (all committed). E4 (liboqs build variants `b2-portable` and `b3-native`) and E5 (native macOS, outside Docker) are coded.

**16 Oct:** Mac results: the trial, E1, E10, E2, E6, E7, E8, E9, E11, E12 and E13 (all committed). E3 (integration-pattern cost) is coded.

**14 Oct:** Mac results: the trial, E1, E10, E2, E8, E11 and E13 (all committed). E6 and E7 are coded.

**13 Oct:** Mac results: the trial, E1, E10, E2, E8 and E11 (all committed). E13 is coded: its size model is calibrated on a read-only capture of the live ledger and IPFS store and reproduces it to the byte.

**Earlier status (11 Oct 2026): Mac-first.**
- The harness runs E0 (environment), E1 (primitive latency), E2 (integration cost: reuse, reinit, production, and their components), E8 (sizes and their distributions), E10 (issuer/holder equivalence for ML-DSA), the probe passes and the canary.
- The macOS launcher works end to end. Mac results so far: the trial, E1, E10 and E2.
- Decision of 11 Oct: every remaining experiment is coded for the Mac first (E2, E8, E13's model, E11, E3, E9, E4, E5, E6, E7, E12), each with unit tests and a Mac smoke run, then run one by one. The Windows and Linux launchers wait until those machines are available.

## How it works

All measurement and orchestration logic is Go inside one Docker image. The image is built natively on each machine, never emulated: arm64 on the M1 Mac, amd64 on the Windows laptop and the VM.

| Part | Role |
|---|---|
| `docker/Dockerfile` | Builds liboqs 0.16.0 for one variant (`b1-prod` by default) and runs its correctness gate. Then runs `go vet` and `go test` of the whole harness, then builds the binaries. Any failure fails the build. |
| `kpi-batch` | The image's entry point. It runs a plan as a resumable batch: shuffled repetitions, a canary every 10 minutes, quality flags, repeats of flagged blocks, checksums. |
| `kpi-prim` | One fresh process per block (prep, E1, E10, probe, canary, and every later experiment). |
| `kpi-env` | E0, the environment capture. |
| `scripts/run-mac.sh` | The Mac's host launcher (one per machine: the VM has `scripts/run-linux.sh`, Obeid's laptop `scripts/run-windows.ps1`). It runs preflight checks, builds the image, runs the host monitor, keeps the machine awake, copies the results out, verifies them and scans them for identifying text. |
| `scripts/run-linux.sh`, `scripts/monitor-linux.sh` | The VM's launcher and host monitor: the same steps, with the VM's checks (QChain's services and the GitHub runner stopped, VS Code closed). The launcher detaches into tmux under `systemd-inhibit`, so the SSH, VS Code or AnyDesk session can be closed. |
| `scripts/run-windows.ps1`, `scripts/monitor-windows.ps1` | Obeid's laptop (Windows PowerShell 5.1, Docker Desktop on WSL2): the same steps, with the laptop's checks (mains power, power mode, Obeid's containers stopped). The run stays in its PowerShell window; Ctrl-C asks for a stop after the current block. |
| `scripts/e5-setup-mac.sh`, `scripts/e5-run-mac.sh` | E5 only: the native macOS toolchain (liboqs and OpenSSL built with Apple clang into `/Users/Shared/qchain-kpi-native`, nothing installed system-wide) and the native launcher, which runs `kpi-batch` directly on macOS with `run-mac.sh`'s own preflight, monitor, pause, Wi-Fi and finishing steps. |

Each run uses its own container and its own Docker volume:
- The container is pinned to one CPU (E12 plans: the CPUs they name), with no CPU-time quota, no network, and memory limited with no swap.
- Results leave the container only with `docker cp`, after the batch has finished.

## Running

Requirements on the Mac:
- Colima: `colima start --vm-type vz --cpus 4 --memory 4 --mount none`;
- Docker CLI with `docker-buildx`, and a builder: `docker buildx create --name kpi --driver docker-container`;
- Homebrew `go@1.24` (Go 1.24.13);
- `shellcheck` (for development).

```bash
scripts/dev.sh                     # build the test stage: liboqs gate + go vet + go test
scripts/run-mac.sh start --plan smoke --machine mac-m1          # pipeline test (tiny budgets)
scripts/run-mac.sh start --plan trial --machine mac-m1 --cooling "USB fan cooling pad"
scripts/run-mac.sh stop  <run-id>      # graceful: finishes the current block, then stops
scripts/run-mac.sh resume <run-id>     # continue a stopped or interrupted batch
scripts/run-mac.sh extend <run-id> --blocks E1.alg.op.rN[,...]   # run blocks that ended without a clean attempt again, inside the finished run
scripts/run-mac.sh status
scripts/run-mac.sh start --plan e4 --machine mac-m1 --variant b2-portable   # E4: another liboqs build
```

E5 runs natively on macOS (Colima stopped):

```bash
scripts/e5-setup-mac.sh                       # once: the native toolchain (liboqs gate, OpenSSL tests)
scripts/e5-run-mac.sh build                   # optional: native go vet, go test and binaries for the current commit
scripts/e5-run-mac.sh start --plan smoke-e5   # then e5; stop, resume, extend and status as run-mac.sh
```

E14 runs against QChain's Fabric network rebuilt in Colima (`fabric/`; `docs/methodology.md`, E14):

```bash
scripts/e14-network-mac.sh setup                  # once, with Wi-Fi on: images, certificates, channel, chaincode
scripts/e14-network-mac.sh start|stop|status      # after a Mac sleep the run launcher starts it by itself
scripts/e14-run-mac.sh check                      # read-only: network up, preflight, network facts
scripts/e14-run-mac.sh start --plan smoke-e14     # then e14; stop, resume and status as run-mac.sh
```

**Trial and main plans refuse to start unless the machine is ready:**
- on mains power;
- thermal pressure nominal, Low Power Mode off;
- no other containers running;
- Teams, Chrome, Edge, VS Code and OneDrive closed;
- automatic macOS updates off (they could restart the Mac mid-run);
- (done by the launcher) macOS background analysis services (Photos analysis, Spotlight AI indexing, suggestions) are paused for the run and resumed at the end;
- (done by the launcher) Wi-Fi is switched off once the image is built and switched back on when the script ends (only if it was on before);
- the host CPU quiet;
- at least 10 GB free;
- the harness committed.

`--force` overrides this, and the override is recorded.

**Start trial and main runs from Terminal, not from VS Code.** When a batch finishes, the launcher:
1. moves the folder into `results/<os>/<run-id>/`;
2. adds a row to `results/<os>/INDEX.md`;
3. prints the `git` commands to commit it.

Pushing is always manual.

**Plans** (one folder per machine, listed in `plans/README.md`). The Mac's, in `plans/mac/`:
- `smoke`: all algorithms, tiny budgets;
- `smoke-quick`: a few algorithms;
- `trial`: calibration;
- `trial-e2`: add-on calibration for E2 (batch sizes of the new arms only), once per machine before its first E2 run;
- `trial-e11`: add-on calibration for E11 (fast/slow classification of the 991 cells), once per machine before its first E11 run;
- `trial-e6`, `trial-e7`: add-on calibrations for E6 (classification and batch sizes of the 220 cells) and E7 (batch sizes of its arms), once per machine before their main runs;
- `smoke-e2`, `smoke-e6`, `smoke-e7`, `smoke-e3`, `smoke-e4`, `smoke-e5`, `smoke-e8`, `smoke-e9`, `smoke-e11`, `smoke-e12`, `smoke-e13`: pipeline tests;
- `e1`, `e10`, `e2`, `e3`, `e4`, `e5`, `e6`, `e7`, `e8`, `e9`, `e11`, `e12`, `e13`: main runs. `e4` and `smoke-e4` need a `--variant` other than `b1-prod`; `e5` and `smoke-e5` run only with `scripts/e5-run-mac.sh`. `e12` and `smoke-e12` give the container 4 CPUs (`cpuset`); every other plan pins it to one.

A machine's first main run needs its calibrated section in `config/thresholds.json`. The trial writes `calibration.json`, which is reviewed, merged and committed before any main run on that machine.

### The VM and Obeid's laptop

Set-up, once per machine: a clone of this repository (on the VM next to the QChain checkout, never inside it), and the image builder `docker buildx create --name kpi --driver docker-container`. Plans are in `plans/vm/` and `plans/windows/` (`plans/README.md`).

```bash
# VM: start, resume and extend restart themselves in a tmux session (setsid and nohup without
# tmux), under systemd-inhibit and pinned to vCPU 0; the terminal can then be closed
scripts/run-linux.sh start --plan smoke --machine vm-x86v2
tmux attach -t kpi-<stamp>              # watch it; leave with Ctrl-B, then D
scripts/run-linux.sh start --plan e4 --machine vm-x86v2 --variant b4-noossl   # E4 on the VM: b4 only
scripts/run-linux.sh stop|resume|extend|status ...    # as on the Mac; --foreground stays in the terminal
scripts/test-run-linux.sh               # on the Mac: the launcher against a fake docker (about 8 minutes)
```

```powershell
# Obeid's laptop, in a PowerShell window that stays open for the run (AnyDesk can disconnect)
powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 start --plan smoke --machine win-ryzen
powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 start --plan e4 --machine win-ryzen --variant b2-portable   # and b3-native
powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 stop|resume|extend|status ...
```

**Trial and main plans refuse to start unless the machine is ready** (`--force` overrides, recorded):
- **VM:** no container running (QChain's Fabric network, `qchain-api` and `qchain-web-gateway` are stopped for the study); MySQL, IPFS and the GitHub Actions runner stopped; VS Code closed (a detached run waits up to 10 minutes for it to close); the machine quiet over 6 s (its steal is recorded); 10 GB free; Docker with cgroup v2.
- **Windows:** on mains power; Energy Saver off; power mode "Best performance" (the laptop's own setting); no container running (Obeid's are stopped for the run); no window of VS Code, Teams, Edge or Chrome; the laptop quiet over 6 s; 10 GB free; Docker with cgroup v2. The Windows Update pause date is shown and recorded.
- **Both:** the harness committed; only the machine's liboqs builds (VM `b1-prod`, `b4-noossl`; Windows `b1-prod`, `b2-portable`, `b3-native`), and a build other than `b1-prod` only for E4 plans.

## Layout

```
cmd/        kpi-batch kpi-prim kpi-env kpi-canary-native
internal/   algs pqc classical qchain scheme timing inputs proto plan quality stats record canary procsetup buildinfo memprobe conc l0 gostd
csrc/       kpi_malloc.c (E9's C-heap counter, preloaded into E9's blocks only)
docker/     Dockerfile and the liboqs build and gate scripts
plans/      run plans (JSON), one folder per machine: mac/ vm/ windows/; README.md lists them
            (code: each experiment in its own eN.go file in cmd/kpi-prim, cmd/kpi-batch, internal/proto, internal/plan, internal/inputs)
config/     thresholds.json (one calibrated section per machine)
scripts/    run-mac.sh monitor-mac.sh run-linux.sh monitor-linux.sh test-run-linux.sh run-windows.ps1 monitor-windows.ps1
            dev.sh check-sync.sh e5-setup-mac.sh e5-run-mac.sh e5-gate-hashlib-mac.py
            e13-*.py (ledger capture tools) e14-network-mac.sh e14-run-mac.sh
fabric/     E14: network/ (QChain's network files, copied, and the channel and chaincode scripts), chaincode/ (kpipayload),
            client/ (the Go client, its own module), plans/, Dockerfile
testdata/   RFC 9180 vectors; fabric/ and fabric-blockgen/ (a synthetic block file for the E13 tools' test);
            launcher-linux/ (the fake docker of scripts/test-run-linux.sh)
docs/       plan, coding plan, methodology, data dictionary, provenance, decisions
results/    mac/ windows/ linux/: MACHINE.md, INDEX.md and one folder per run
```

## Renamed files (25 Oct 2026)

The Mac's runs were made before these renames. Their run folders (`console.txt`, the launcher files), the plan
descriptions, the planning snapshots `docs/plan.md` and `docs/coding-plan.md`, and the decisions log's earlier rows
use the old names. Nothing else changed: the plan files moved byte for byte, and
`scripts/e13-block-sizes.py` is byte-identical to the `fabric-block-sizes.py` that ran on the VM
(SHA-256 `b29348b9...`, `internal/ledger/vmcapture/README.md`).

| Old name | New name |
|---|---|
| `plans/*.json` | `plans/mac/*.json` (one folder per machine: `plans/README.md`) |
| `scripts/run.sh` | `scripts/run-mac.sh` |
| `scripts/native-mac.sh` | `scripts/e5-run-mac.sh` |
| `scripts/native-setup.sh` | `scripts/e5-setup-mac.sh` |
| `scripts/native-gate-hashlib.py` | `scripts/e5-gate-hashlib-mac.py` |
| `scripts/e14-run.sh` | `scripts/e14-run-mac.sh` |
| `scripts/e14-network.sh` | `scripts/e14-network-mac.sh` |
| `scripts/fabric-block-sizes.py` | `scripts/e13-block-sizes.py` |
| `scripts/fabric-sizes-detail.py` | `scripts/e13-sizes-detail.py` |
| `scripts/fabric-capture-export.py` | `scripts/e13-capture-export.py` |
| `scripts/fabric-ipfs-link.py` | `scripts/e13-ipfs-link.py` |
| `scripts/test-fabric-block-sizes.py` | `scripts/e13-test-block-sizes.py` |

Naming rule: a script tied to one experiment starts with it (`eN-`), and one tied to one machine ends with it
(`-mac`, `-linux`, `-windows`): `eN-<purpose>-<machine>`.

## Rules

- **Privacy.** Run folders hold only allow-listed facts. Before a folder enters `results/`, every file is scanned for the machine's host name, user name, home path, IP and MAC addresses and serial number (and the AnyDesk ID on Windows). Credentials never enter any file.
- **Other people's machines.** On Obeid's laptop only `kpi-*` images, containers and volumes are ever removed (never `docker system prune`). On the VM, the QChain checkout is never touched.
- **Verbatim copies.** The QChain code in `internal/qchain/*_prod.go` is copied verbatim from the main repository. Do not edit it there (see `docs/provenance.md`).
