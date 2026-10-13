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

**Status (14 Oct 2026): Mac-first.** Mac results: the trial, E1, E10, E2, E8, E11 and E13 (all committed). E6 and E7 are coded.

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
| `kpi-prim` | One fresh process per block (prep, E1, E10, probe, canary). |
| `kpi-env` | E0, the environment capture. |
| `scripts/run.sh` | The host launcher (macOS now; Linux and the Windows `run.ps1` in phase 1b). It runs preflight checks, builds the image, runs the host monitor, keeps the machine awake, copies the results out, verifies them and scans them for identifying text. |

Each run uses its own container and its own Docker volume:
- The container is pinned to one CPU, with no CPU-time quota, no network, and memory limited with no swap.
- Results leave the container only with `docker cp`, after the batch has finished.

## Running

Requirements on the Mac:
- Colima: `colima start --vm-type vz --cpus 4 --memory 4 --mount none`;
- Docker CLI with `docker-buildx`, and a builder: `docker buildx create --name kpi --driver docker-container`;
- Homebrew `go@1.24` (Go 1.24.13);
- `shellcheck` (for development).

```bash
scripts/dev.sh                     # build the test stage: liboqs gate + go vet + go test
scripts/run.sh start --plan smoke --machine mac-m1          # pipeline test (tiny budgets)
scripts/run.sh start --plan trial --machine mac-m1 --cooling "USB fan cooling pad"
scripts/run.sh stop  <run-id>      # graceful: finishes the current block, then stops
scripts/run.sh resume <run-id>     # continue a stopped or interrupted batch
scripts/run.sh extend <run-id> --blocks E1.alg.op.rN[,...]   # run blocks that ended without a clean attempt again, inside the finished run
scripts/run.sh status
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

**Plans (`plans/`):**
- `smoke`: all algorithms, tiny budgets;
- `smoke-quick`: a few algorithms;
- `trial`: calibration;
- `trial-e2`: add-on calibration for E2 (batch sizes of the new arms only), once per machine before its first E2 run;
- `trial-e11`: add-on calibration for E11 (fast/slow classification of the 991 cells), once per machine before its first E11 run;
- `trial-e6`, `trial-e7`: add-on calibrations for E6 (classification and batch sizes of the 220 cells) and E7 (batch sizes of its arms), once per machine before their main runs;
- `smoke-e2`, `smoke-e6`, `smoke-e7`, `smoke-e8`, `smoke-e11`, `smoke-e13`: pipeline tests;
- `e1`, `e10`, `e2`, `e6`, `e7`, `e8`, `e11`, `e13`: main runs.

A machine's first main run needs its calibrated section in `config/thresholds.json`. The trial writes `calibration.json`, which is reviewed, merged and committed before any main run on that machine.

## Layout

```
cmd/        kpi-batch kpi-prim kpi-env kpi-canary-native
internal/   algs pqc classical qchain scheme timing inputs proto plan quality stats record canary procsetup buildinfo
docker/     Dockerfile and the liboqs build and gate scripts
plans/      run plans (JSON)
            (code: each experiment in its own eN.go file in cmd/kpi-prim, cmd/kpi-batch, internal/proto, internal/plan, internal/inputs)
config/     thresholds.json (one calibrated section per machine)
scripts/    run.sh monitor-mac.sh dev.sh
testdata/   RFC 9180 vectors
docs/       plan, coding plan, methodology, data dictionary, provenance, decisions
results/    mac/ windows/ linux/: MACHINE.md, INDEX.md and one folder per run
```

## Rules

- **Privacy.** Run folders hold only allow-listed facts. Before a folder enters `results/`, every file is scanned for the machine's host name, user name, home path, IP and MAC addresses and serial number (and the AnyDesk ID on Windows). Credentials never enter any file.
- **Other people's machines.** On Obeid's laptop only `kpi-*` images, containers and volumes are ever removed (never `docker system prune`). On the VM, the QChain checkout is never touched.
- **Verbatim copies.** The QChain code in `internal/qchain/*_prod.go` is copied verbatim from the main repository. Do not edit it there (see `docs/provenance.md`).
