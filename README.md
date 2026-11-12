<p align="center">
  <img src=".github/assets/qchain_logo.jpeg" width="220" alt="QChain logo" />
</p>

<h1 align="center">qchain-kpi</h1>

<p align="center">
  <b>Post-quantum performance benchmarks for QChain: 30 algorithms, 15 experiments, three machines.</b><br/>
  A measurement harness that times every NIST post-quantum algorithm in liboqs the way QChain's backend and wallet use them, records every sample, controls the noise, and turns the runs into the tables and figures of the study.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24.13-00ADD8" alt="Go" />
  <img src="https://img.shields.io/badge/liboqs-0.16.0-6E40C9" alt="liboqs" />
  <img src="https://img.shields.io/badge/OpenSSL-3.0-721412" alt="OpenSSL" />
  <img src="https://img.shields.io/badge/Algorithms-25%20PQ%20%2B%205%20classical-0A66C2" alt="Algorithms" />
  <img src="https://img.shields.io/badge/Machines-ARM%20%C2%B7%20x86%20AVX2%20%C2%B7%20x86%20VM-2F3134" alt="Machines" />
  <img src="https://img.shields.io/badge/Status-measurements%20complete-2EA44F" alt="Status" />
  <img src="https://img.shields.io/badge/License-Proprietary-red" alt="License" />
</p>

---

## 📊 Status

| | |
|---|---|
| **Measurements** | complete on all three machines: the Mac (10 to 22 Oct 2026), the virtualised x86 server (26 to 31 Oct) and the Windows laptop (7 to 10 Nov) |
| **Analysis** | complete (10 Nov 2026; figures and tables for the paper 12 Nov): 20 analysis tables, 33 presentation tables covering every experiment and 26 figures in [`results/combined/`](results/combined/README.md), each machine's own charts and tables in `results/<machine>/`, written up in [`docs/`](docs/README.md) |
| **Data quality** | every block has a clean headline attempt except 17 on the laptop (16 E12 blocks at four workers, one E2 block), reported through the pre-declared sensitivity analysis; no limit was changed after the runs |
| **Test machines** | all three returned to their state before the study (10 Nov 2026); a new run needs the set-up under [Running the harness](#running-the-harness) |
| **Next** | the paper and the project's final report |

> ⚠️ **Private until publication.** This repository becomes public when the paper is published. Credentials never enter
> any file, and every run folder passed an automatic privacy scan before it was committed.

---

## Overview

QChain issues academic credentials on a Hyperledger Fabric blockchain. Every credential is signed with **ML-DSA-44** and its contents are encrypted to the holder with **ML-KEM-768**. Choosing those two algorithms raises questions that one benchmark cannot answer: what does each post-quantum algorithm cost on real hardware, what does QChain's own code add on top of the cryptographic library, how much does the processor matter, how big does the ledger get, and is the post-quantum system slower than a classical one would be?

This repository answers them. It measures **30 algorithms** (25 post-quantum from liboqs 0.16.0 and 5 classical baselines through OpenSSL 3.0) in **15 experiments**, on **three machines** that exercise three different code paths: an Apple M1 (ARM with NEON), an AMD Ryzen laptop (x86 with AVX2) and a virtualised x86 server without AVX2. QChain's production functions are copied verbatim into the harness, so the measured code is the deployed code.

The six research questions ([plan §1](docs/plan.md#1-purpose-and-research-questions)) and their answers ([details](docs/results-comparison.md#10-answers-to-the-research-questions)):

| | Question | Answer in short |
|---|---|---|
| **RQ1** | What do the primitives cost? | ML-DSA-44 signs in 60 to 206 µs and verifies in 28 to 79 µs. It verifies faster than ECDSA on ARM and AVX2 hardware but signs 3 to 7 times slower. |
| **RQ2** | What does the whole credential lifecycle cost? | 0.69 ms (Mac), 0.96 ms (laptop), 1.32 ms (VM) for the production suite: 0.86, 0.78 and 1.13 times the classical suite. |
| **RQ3** | What does QChain's integration add? | The Go binding a median of 0.3 %; the deployed hex-string pattern 6 to 13 % (ML-DSA) and 14 to 28 % (ML-KEM). |
| **RQ4** | How much do hardware, runtime and build matter? | Hardware a lot (no AVX2: 2.6× for ML-DSA, 25× for Falcon signing); Docker and a native build nothing measurable; a portable build 1.4 to 2.4× for ML-DSA and up to 27× for Falcon signing. |
| **RQ5** | What does it cost in storage? | An ML-DSA-44 credential transaction is 22.3 KB on the ledger (7.9 KB with ECDSA), ten to a block; about 22 GB per million credentials per peer. |
| **RQ6** | Are issuer-side and holder-side signing equivalent? | Yes, in all nine tests (three ML-DSA sets on three machines). |

---

## Key results

| Operation (median, one call) | Mac M1 | Laptop (Ryzen, AVX2) | VM (x86, no AVX2) |
|---|---|---|---|
| ML-DSA-44 sign / verify | 60 / 28 µs | 79 / 32 µs | 206 / 79 µs |
| ECDSA P-256 sign / verify | 20 / 55 µs | 24 / 59 µs | 31 / 78 µs |
| ML-KEM-768 encapsulate | 14 µs | 21 µs | 39 µs |
| X25519 encapsulate (DHKEM) | 67 µs | 78 µs | 90 µs |
| Falcon-512 sign | 0.15 ms | 0.21 ms | 3.67 ms |
| SLH-DSA-SHA2-128f sign | 20.7 ms | 20.8 ms | 22.7 ms |
| **Credential lifecycle, production suite** | **0.69 ms** | **0.96 ms** | **1.32 ms** |
| Credential lifecycle, ECDSA + ECDH P-256 | 0.81 ms | 1.23 ms | 1.18 ms |

- **The post-quantum production suite is cheaper than the classical one on hardware with the optimised code**, and 13 % dearer on a processor without AVX2.
- **The cost of a suite is set by its signature scheme.** Falcon pays at key generation (5 to 33 ms), SLH-DSA at signing (21 ms to 1.5 s); on the Mac, HQC multiplies the cheapest suites' cost by 8.6 to 52.
- **The avoidable cost is the integration pattern**, not the library: hex conversion for the lattice schemes, key import on every call for the classical ones.
- **Storage is the real price:** an ML-DSA-44 credential transaction is 22.3 KB on the ledger against 7.9 KB with ECDSA, 2.8 times as much; a block holds ten of them.
- **The noise controls change almost nothing:** over 5,030 cells, the headline and all-attempts medians differ by 0.0 % in the median and by at most 5.5 %.

Full results: [Mac](docs/results-mac.md) · [VM](docs/results-vm.md) · [Windows laptop](docs/results-windows.md) · [**the three compared**](docs/results-comparison.md).

---

## Architecture

```
  host (one launcher per machine)               Docker image  kpi-bench:<variant>-<harness commit>
 ┌───────────────────────────────┐           ┌──────────────────────────────────────────────────────┐
 │ scripts/run-mac.sh            │   build   │ liboqs 0.16.0  b1 prod · b2 portable · b3 native ·    │
 │ scripts/run-linux.sh          │ ────────▶ │                b4 without OpenSSL                     │
 │ scripts/run-windows.ps1       │           │   + its correctness gate (KAT and ACVP vectors)       │
 │  · preflight: a quiet machine │           │ go vet + go test of the whole harness                 │
 │  · host monitor every 5 s     │    run    │                                                       │
 │  · keep awake, one CPU, no net│ ────────▶ │ kpi-batch ── the plan's blocks, shuffled ──┐          │
 │  · copy out, verify, privacy  │           │   canary every 10 min, quality flags,      │          │
 └───────────────┬───────────────┘           │   repeats, checkpoints, resume, extend     ▼          │
                 │                           │ kpi-prim ── one fresh process per block:              │
                 │                           │   E1…E13, QChain's production code copied verbatim    │
                 ▼                           └──────────────────────────────────────────────────────┘
   results/<machine>/<run-id>/    manifest · blocks.csv · quality.csv · canary.csv · raw/*.csv.gz · SHA256SUMS
                 │
                 ▼
   analysis/run_all.py  ──▶  results/combined/analysis_tables (T0–T14) · tables (presentation) · figures (F0–F17)
                        ──▶  results/<machine>/figures · tables (one machine's bar charts and tables)
                 │
                 ▼
   docs/results-mac.md · results-vm.md · results-windows.md · results-comparison.md
```

| Part | Role |
|---|---|
| `docker/Dockerfile` | Builds liboqs 0.16.0 for one variant (`b1-prod` by default) and runs its correctness gate, then `go vet` and `go test` of the whole harness, then the binaries. Any failure fails the build. |
| `kpi-batch` | The image's entry point. Runs a plan as a resumable batch: shuffled repetitions, a canary every 10 minutes, quality flags, repeats of flagged blocks, checksums. |
| `kpi-prim` | One fresh process per block (preparation, every experiment, the probe and the canary). |
| `kpi-env` | E0, the capture of the environment. |
| `scripts/run-mac.sh`, `monitor-mac.sh` | The Mac's launcher: preflight checks, the image build, the host monitor, keeping the machine awake, copying the results out, verifying them and scanning them for identifying text. |
| `scripts/run-linux.sh`, `monitor-linux.sh` | The VM's launcher and monitor: the same steps with the VM's checks. Detaches into tmux under `systemd-inhibit`, so the SSH, VS Code or AnyDesk session can be closed. |
| `scripts/run-windows.ps1`, `monitor-windows.ps1` | The Windows laptop (PowerShell 5.1, Docker Desktop on WSL2): the same steps with the laptop's checks. The run stays in its window; Ctrl-C asks for a stop after the current block. |
| `scripts/e5-setup-mac.sh`, `e5-run-mac.sh` | E5 only: liboqs and OpenSSL built with Apple clang into `/Users/Shared/qchain-kpi-native` (nothing installed system-wide) and a native launcher with `run-mac.sh`'s own checks. |
| `analysis/run_all.py` | Turns every run folder into the analysis tables, presentation tables and figures of `results/combined/`, and each machine's own in `results/<machine>/`. |

Each run uses its own container and its own Docker volume. The container is pinned to one CPU (E12: the CPUs its plan names), with no CPU-time quota, no network and memory without swap. Results leave the container only by `docker cp`, after the batch has finished. **Quality control:** a block disturbed by an external cause (steal, other CPU use, heat, a slow canary) is flagged and repeated, and only its first clean attempt, the **headline**, enters the results. Each machine's limits are calibrated from its own trial and frozen before its main runs.

---

## What is measured

**Algorithms (30)**

| Family | Parameter sets | Library |
|---|---|---|
| ML-DSA (FIPS 204) | 44, 65, 87 | liboqs |
| Falcon | 512, 1024, padded 512, padded 1024 | liboqs |
| SLH-DSA (FIPS 205), pure | SHA-2 and SHAKE, 128/192/256, s and f (12 sets) | liboqs |
| ML-KEM (FIPS 203) | 512, 768, 1024 | liboqs |
| HQC | 1, 3, 5 | liboqs |
| Classical baselines | ECDSA P-256, Ed25519, RSA-3072 (PSS), DHKEM with P-256 and with X25519 (RFC 9180) | OpenSSL 3.0 |

**Experiments (15)** and where they ran ([methodology §9](docs/methodology.md#9-which-experiments-run-on-which-machine-decided-21-oct-2026))

| | Experiment | Mac | VM | Laptop |
|---|---|:-:|:-:|:-:|
| E0 | environment and correctness (every run) | ✓ | ✓ | ✓ |
| E1 | primitive latency: every operation of the 30 algorithms | ✓ | ✓ | ✓ |
| E2 | integration cost: reused key, re-initialised object, QChain's production pattern, and their components | ✓ | without SLH-DSA | without SLH-DSA |
| E3 | integration layers: C library, Go-to-C boundary, Go binding, Go's own library | ✓ | | |
| E4 | liboqs build sensitivity | portable, native | without OpenSSL | portable, native |
| E5 | native macOS against the container | ✓ | | |
| E6 | message size, 64 B to 1 MiB | ✓ | | |
| E7 | rejection timing (bad signature, wrong message, tampered ciphertext) | ✓ | | |
| E8 | sizes of keys, signatures and ciphertexts | ✓ | | |
| E9 | memory: stack, C heap, Go heap, resident set | ✓ | ✓ | ✓ |
| E10 | issuer-side against holder-side ML-DSA signing (equivalence) | ✓ | ✓ | ✓ |
| E11 | credential lifecycle: activation, issuance, retrieval, presentation, verification | 176 pairs | 16 suites | 16 suites |
| E12 | concurrency | 1, 2, 4 workers | 1, 2 | 1, 2, 4, 8 |
| E13 | ledger and IPFS bytes: a model calibrated on a captured block | ✓ | | |
| E14 | Fabric latency against payload size, on QChain's network rebuilt in Docker | ✓ | | |

---

## Machines

| | Mac (`mac-m1`) | VM (`vm-x86v2`) | Laptop (`win-ryzen`) |
|---|---|---|---|
| Processor | Apple M1, 4P + 4E cores, NEON, SHA-3 | QEMU virtual x86 (`x86-64-v2-AES`), **no AVX2**, 2 vCPUs shared with other VMs | AMD Ryzen 5 7520U, 4 cores / 8 threads, **AVX2**, 15 W |
| Memory | 8 GB | 30 GiB | 8 GB |
| Runtime | Docker in Colima (Apple Virtualization) | Docker on Ubuntu 24.04 | Docker Desktop on WSL2, Windows 11 |
| Main runs | 15 | 7 | 8, plus two rerun rounds |
| Attempts / flagged | 11,507 / 5.0 % | 5,997 / 10.1 % | 6,672 / 11.5 % |
| Description | [MACHINE.md](results/mac/MACHINE.md) | [MACHINE.md](results/linux/MACHINE.md) | [MACHINE.md](results/windows/MACHINE.md) |

> In the paper the VM is described only as *a tested environment: a virtualised x86 server*, never as QChain's deployment.

---

## Technology stack

| Layer | Technology |
|---|---|
| Post-quantum library | liboqs 0.16.0 (commit `5a1a854b`), liboqs-go v0.16.0 |
| Classical library | OpenSSL 3.0 (Debian bookworm) through cgo; DHKEM written to RFC 9180 and checked against its vectors |
| Harness | Go 1.24.13; small C helpers for the raw-library layer, stack painting and heap counting |
| Images | Docker; `golang:1.24.13-bookworm` and `debian:bookworm-slim` pinned by digest; apt from a fixed Debian snapshot |
| Launchers | Bash (macOS, Linux), Windows PowerShell 5.1 |
| E14 | Hyperledger Fabric 2.5.15 in Docker, a Go Fabric Gateway client, the `kpipayload` chaincode |
| Analysis | Python 3 with numpy, scipy and matplotlib |

---

## Repository structure

```
qchain-kpi/
├── README.md                this file
├── LICENSE
├── cmd/                     kpi-batch (orchestrator) · kpi-prim (one block) · kpi-env (E0) · kpi-canary-native
├── internal/                algs pqc classical qchain scheme timing inputs proto plan quality stats record
│   │                        canary procsetup buildinfo memprobe conc l0 gostd flow cells ledger
│   ├── qchain/              QChain's production functions, copied verbatim (docs/provenance.md)
│   └── ledger/vmcapture/    the E13 capture of a real block and IPFS store (numbers only)
├── csrc/                    kpi_malloc.c: E9's C-heap counter, preloaded into E9's blocks only
├── docker/                  Dockerfile, the liboqs build and its correctness gate, apt set-up
├── config/thresholds.json   each machine's calibrated limits (frozen before its main runs)
├── plans/                   run plans (JSON), one folder per machine: mac/ vm/ windows/
├── scripts/                 launchers, host monitors, series scripts, E5/E13/E14 tools, check-sync.sh
├── fabric/                  E14: QChain's network files copied, the kpipayload chaincode, the Go client
├── testdata/                RFC 9180 vectors, a synthetic block file, the fake docker for launcher tests
├── analysis/                the analysis code: python3 analysis/run_all.py
├── results/
│   ├── mac/  linux/  windows/    README.md, MACHINE.md, INDEX.md, one folder per run (every sample, gzipped),
│   │                             and figures/ and tables/: that machine's own bar charts and tables
│   └── combined/                 analysis_tables/ (T0–T14), tables/ (presentation), figures/ (F0–F17)
└── docs/                    plan, methodology, data dictionary, provenance, decisions, history,
                             glossary and the four results documents (start at docs/README.md)
```

Each experiment's code sits in its own `eN.go` file in `cmd/kpi-prim`, `cmd/kpi-batch`, `internal/proto`, `internal/plan` and `internal/inputs`.

---

## Reproducing the analysis

```bash
git clone https://github.com/teamqchain/qchain-kpi.git && cd qchain-kpi
python3 analysis/run_all.py        # about one minute; needs numpy, scipy and matplotlib only
```

It reads every run folder in `results/` and `config/thresholds.json` and rewrites `results/combined/` and the `figures/` and `tables/` folders of each machine; two runs give identical files. A cell's value is the median over its five repetitions of each repetition's median (headline attempts only), with a 95 % bootstrap interval over the repetitions; machines are compared by ratios, intervals and Cliff's delta, and E10's equivalence by TOST. Details: [analysis/README.md](analysis/README.md) and [methodology §10](docs/methodology.md#10-analysis-after-the-runs-10-nov-2026).

To check a run's integrity: `cd results/<machine>/<run-id> && shasum -a 256 -c SHA256SUMS`.

---

## Running the harness

> ⚠️ **The study's runs are finished and the test machines have been reverted.** These are the commands the runs were
> made with, for anyone who repeats or extends the study. A new machine needs its own trial and calibration first:
> the trial writes `calibration.json`, which is reviewed, merged into `config/thresholds.json` and committed before
> that machine's first main run.

<details>
<summary><b>On the Mac</b> (Colima, Docker, Go 1.24)</summary>

Requirements:
- Colima: `colima start --vm-type vz --cpus 4 --memory 4 --mount none`;
- the Docker CLI with `docker-buildx`, and a builder: `docker buildx create --name kpi --driver docker-container`;
- Homebrew `go@1.24` (Go 1.24.13);
- `shellcheck`, for development.

```bash
scripts/dev.sh                                                    # the test stage: liboqs gate + go vet + go test
scripts/run-mac.sh start --plan smoke --machine mac-m1            # pipeline test (tiny budgets)
scripts/run-mac.sh start --plan trial --machine mac-m1 --cooling "USB fan cooling pad"
scripts/run-mac.sh stop   <run-id>                                # graceful: finishes the current block, then stops
scripts/run-mac.sh resume <run-id>                                # continue a stopped or interrupted batch
scripts/run-mac.sh extend <run-id> --blocks E1.alg.op.rN[,...]    # blocks without a clean attempt, again, inside the run
scripts/run-mac.sh status
scripts/run-mac.sh start --plan e4 --machine mac-m1 --variant b2-portable   # E4: another liboqs build
```

E5 runs natively on macOS, with Colima stopped:

```bash
scripts/e5-setup-mac.sh                       # once: the native toolchain (liboqs gate, OpenSSL tests)
scripts/e5-run-mac.sh build                   # optional: native go vet, go test and binaries for this commit
scripts/e5-run-mac.sh start --plan smoke-e5   # then e5; stop, resume, extend and status as run-mac.sh
```

E14 runs against QChain's Fabric network rebuilt in Colima (`fabric/`; [methodology](docs/methodology.md), E14):

```bash
scripts/e14-network-mac.sh setup              # once, with Wi-Fi on: images, certificates, channel, chaincode
scripts/e14-network-mac.sh start|stop|status  # after a Mac sleep the run launcher starts it by itself
scripts/e14-run-mac.sh check                  # read-only: network up, preflight, network facts
scripts/e14-run-mac.sh start --plan smoke-e14 # then e14; stop, resume and status as run-mac.sh
```

**Trial and main plans refuse to start unless the Mac is ready** (`--force` overrides, and the override is recorded):
- on mains power; thermal pressure nominal; Low Power Mode off;
- no other container running;
- Teams, Chrome, Edge, VS Code and OneDrive closed;
- automatic macOS updates off (they could restart the Mac mid-run);
- the host CPU quiet; at least 10 GB free; the harness committed;
- done by the launcher: macOS's background analysis services (Photos analysis, Spotlight indexing, suggestions) are paused for the run and resumed at the end; Wi-Fi is switched off once the image is built and back on when the script ends (only if it was on).

Start trial and main runs from Terminal, not from VS Code.
</details>

<details>
<summary><b>On a Linux server</b> (the VM)</summary>

Set-up once: a clone of this repository next to (never inside) any other project's checkout, and the builder `docker buildx create --name kpi --driver docker-container`. Plans are in `plans/vm/`.

```bash
# start, resume and extend restart themselves in a tmux session (setsid and nohup without tmux),
# under systemd-inhibit and pinned to vCPU 0; the terminal can then be closed
scripts/run-linux.sh start --plan smoke --machine vm-x86v2
tmux attach -t kpi-<stamp>                                   # watch it; leave with Ctrl-B, then D
scripts/run-linux.sh start --plan e4 --machine vm-x86v2 --variant b4-noossl   # E4 on the VM: b4 only
scripts/run-linux.sh stop|resume|extend|status ...           # as on the Mac; --foreground stays in the terminal
scripts/run-linux-series.sh e1 e2-noslh e4@b4-noossl         # several plans one after another, unattended
scripts/run-linux-series.sh status|stop                      # is a series running; stop it after the current block
scripts/test-run-linux.sh                                    # on the Mac: launcher and series against a fake docker (~15 min)
```

**Preflight:** no container running (QChain's Fabric network, `qchain-api` and `qchain-web-gateway` were stopped for the study); MySQL, IPFS and the GitHub Actions runner stopped; VS Code closed (a detached run waits up to 10 minutes for it to close); the machine quiet over 6 s (its steal is recorded); 10 GB free; Docker with cgroup v2; the harness committed; only the builds `b1-prod` and `b4-noossl`, and `b4` only for E4.

**A series** runs plans one after another and detaches like the launcher; its log is `~/.local/state/qchain-kpi/series.log`. It refuses a second series (a lock), waits 65 s between plans (run ids have minute resolution), tries a refused preflight again up to 6 times 5 minutes apart (a short burst of system work must not end a days-long series), and stops at the first plan that fails or whose batch stopped (exit 5: resume it with `run-linux.sh resume`, then start a series with the remaining plans). Do not open VS Code while a series runs: its server process makes the next preflight refuse.
</details>

<details>
<summary><b>On Windows</b> (Docker Desktop on WSL2, Windows PowerShell 5.1)</summary>

Set-up once: a clone of this repository and the builder `docker buildx create --name kpi --driver docker-container`. Plans are in `plans/windows/`. The run stays in its PowerShell window, which must remain open (a remote-desktop session can disconnect).

```powershell
powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 start --plan smoke --machine win-ryzen
powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 start --plan e4 --machine win-ryzen --variant b2-portable   # and b3-native
powershell -ExecutionPolicy Bypass -File scripts\run-windows.ps1 stop|resume|extend|status ...
powershell -ExecutionPolicy Bypass -File scripts\run-windows-series.ps1 --delay 120 e1 e10 e4@b2-portable
powershell -ExecutionPolicy Bypass -File scripts\run-windows-series.ps1 status|stop   # from a second window
```

**Preflight:** on mains power; Energy Saver off; power mode "Best performance"; no container running (the owner's are stopped for the run); no window of VS Code, Teams, Edge or Chrome; the laptop quiet over 6 s; 10 GB free; Docker with cgroup v2; the harness committed; only the builds `b1-prod`, `b2-portable`, `b3-native`, and the last two only for E4. The Windows Update pause date is shown and recorded.

**A series** follows the Linux rules (a lock, 65 s between plans, 6 retries of a refused preflight, stops at the first failure) but stays in its window; `--delay` gives time to disconnect the remote session first. Its log, `%LOCALAPPDATA%\qchain-kpi\series-<stamp>.log`, holds only the series' own lines; each batch's output stays in the window and in the run's `console.txt`.
</details>

**After a batch** the launcher verifies the checksums, scans every file for private data, moves the folder into `results/<machine>/<run-id>/`, adds a row to that machine's `INDEX.md` and prints the `git` commands to commit it. Pushing is always manual.

**Plans** are listed in [plans/README.md](plans/README.md): `smoke*` (pipeline tests), `trial` (calibration), `trial-e2`, `trial-e6`, `trial-e7`, `trial-e11` (add-on calibrations, once per machine before that experiment's first main run), and the main plans `e1` to `e14` (`e2-noslh`, `e11-curated`, `e12-vm`, `e12-windows` for the reduced scope). `e4` needs a `--variant` other than `b1-prod`; `e5` runs only through `scripts/e5-run-mac.sh`; `e12` plans give the container several CPUs, every other plan one.

---

## Recent updates

- **12 November 2026: figures and tables for the paper.** Bar charts for every E1 operation and the lifecycle, on the three machines together and on each machine alone; new figures for E8 sizes, E10 equivalence, E5 native builds and E14 Fabric latency; E7 redrawn; every log axis labelled "(log scale)" with ticks in plain units; presentation tables (CSV + Markdown) for every experiment; the analysis tables moved to `results/combined/analysis_tables/`.
- **11 November 2026: documentation completed.** Results documents for the Mac and the VM, a history of the study, a glossary and a documentation guide; every README rewritten; the E14 table corrected to use the main run instead of the smoke run.
- **10 November 2026: analysis and revert.** `analysis/` written; tables and figures in `results/combined/`; the laptop's reruns closed (17 blocks reported through the sensitivity analysis); all three test machines returned to their original state.
- **7 to 9 November 2026:** the Windows laptop's main series (42.5 hours), after its trial and add-on trials on 7 November.
- **26 to 31 October 2026:** the VM's runs, analysed on 1 November (no rerun needed).
- **25 October 2026:** the VM and Windows launchers and per-machine plans; scripts renamed (below).
- **10 to 22 October 2026:** every experiment on the Mac, E0 to E14.

The whole story, with the reasons for each turn, is in [docs/history.md](docs/history.md).

---

## Roadmap

- [x] Research plan, harness and correctness gates (October 2026)
- [x] Mac: E0 to E14
- [x] VM and Windows laptop: the reduced scope
- [x] Analysis: tables, figures, results documents
- [x] Figures and presentation tables for the paper, per experiment and per machine
- [x] Test machines reverted
- [ ] Paper draft and the project's final report
- [ ] Interactive dashboard for the co-authors ([plan §10](docs/plan.md#10-outputs-and-visualisation))
- [ ] Public release with the paper

---

## Documentation

| Document | What it covers |
|---|---|
| [docs/README.md](docs/README.md) | **start here**: reading order and conventions |
| [docs/history.md](docs/history.md) | what happened, in order, and why |
| [docs/results-comparison.md](docs/results-comparison.md) | the three machines against the six research questions |
| [docs/results-mac.md](docs/results-mac.md) · [results-vm.md](docs/results-vm.md) · [results-windows.md](docs/results-windows.md) | each machine's results |
| [docs/methodology.md](docs/methodology.md) | what the code measures, quality control, calibration, scope, analysis |
| [docs/glossary.md](docs/glossary.md) | the study's vocabulary |
| [docs/data-dictionary.md](docs/data-dictionary.md) | every file and column of a run folder |
| [docs/provenance.md](docs/provenance.md) | where every copied line of code and every test vector comes from |
| [docs/decisions-log.md](docs/decisions-log.md) | dated technical decisions and their reasons |
| [docs/plan.md](docs/plan.md) · [docs/coding-plan.md](docs/coding-plan.md) | the research plan and the harness design (snapshots of 10 Oct 2026) |
| [analysis/README.md](analysis/README.md) · [results/combined/README.md](results/combined/README.md) | how the numbers are made, and where every experiment's tables and figures are |
| [results/combined/tables/README.md](results/combined/tables/README.md) | the presentation tables, every experiment (CSV to paste, Markdown to read) |
| [results/mac/README.md](results/mac/README.md) · [linux](results/linux/README.md) · [windows](results/windows/README.md) | each machine's runs, figures and tables |
| [plans/README.md](plans/README.md) | every run plan, per machine |

---

## Conventions and rules

- **Privacy.** Run folders hold only allow-listed facts. Before a folder enters `results/`, every file is scanned for the machine's host name, user name, home path, IP and MAC addresses and serial number (and the AnyDesk ID on Windows). Credentials never enter any file.
- **Other people's machines.** On a borrowed machine only `kpi-*` images, containers and volumes are ever removed, and never with `docker system prune`; another project's checkout is never touched. Every change to a test machine is written into a revert list before it is made.
- **Verbatim copies.** The QChain code in `internal/qchain/*_prod.go` is copied verbatim from QChain's backend. Do not edit it here ([provenance](docs/provenance.md); `scripts/check-sync.sh` compares the copies).
- **Frozen thresholds.** A machine's section of `config/thresholds.json` is not changed after its first main run.
- **Script names.** A script tied to one experiment starts with it (`eN-`), one tied to one machine ends with it (`-mac`, `-linux`, `-windows`): `eN-<purpose>-<machine>`.

<details>
<summary><b>Renamed files (25 Oct 2026)</b></summary>

The Mac's runs were made before these renames. Their run folders (`console.txt`, the launcher files), the plan descriptions, the planning snapshots `docs/plan.md` and `docs/coding-plan.md`, and the decisions log's earlier rows use the old names. Nothing else changed: the plan files moved byte for byte, and `scripts/e13-block-sizes.py` is byte-identical to the `fabric-block-sizes.py` that ran on the VM (SHA-256 `b29348b9...`, [vmcapture README](internal/ledger/vmcapture/README.md)).

| Old name | New name |
|---|---|
| `plans/*.json` | `plans/mac/*.json` |
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
</details>

---

## Related repositories

| Repository | What it holds |
|------------|---------------|
| [qchain-portal](https://github.com/teamqchain/qchain-portal) | QPortal, the web portal for issuers, verifiers and IT admins |
| [qchain-wallet](https://github.com/teamqchain/qchain-wallet) | QWallet, the holder's mobile app |
| [qchain-shared](https://github.com/teamqchain/qchain-shared) | the shared Flutter package: certificate template and viewer |
| [qchain-backend](https://github.com/teamqchain/qchain-backend) | the Go API whose production functions this harness copies |
| [qchain-network](https://github.com/teamqchain/qchain-network) | the Hyperledger Fabric network and the chaincode |
| [qchain-docs](https://github.com/teamqchain/qchain-docs) | reports, the setup guide and the earlier ML-DSA benchmarks this study builds on |
| [qchain-kpi](https://github.com/teamqchain/qchain-kpi) | this repository |
| [teamqchain.github.io](https://github.com/teamqchain/teamqchain.github.io) | the project website |

Some of these repositories are private.

---

## Team

A University of Sharjah capstone project, supervised by **Dr. Manar Abu Talib** and **Dr. Sohail Abbas**.

| Member | Role |
|--------|------|
| Mohammed Bin Ali Maqqavi ([@M0hd-Maqqavi](https://github.com/M0hd-Maqqavi)) | Project Manager |
| Mohammed Abdul Haris ([@mohammed-ah-14](https://github.com/mohammed-ah-14)) | Technical Lead |
| Mohammed Nihal ([@nihvp](https://github.com/nihvp)) | Quality Assurance Lead |
| Mohammed Obeid ([@MohammedObeid88](https://github.com/MohammedObeid88)) | Blockchain Specialist |

---

## License

**Proprietary — all rights reserved.** © 2026 QChain. This source code and its data are proprietary and confidential;
unauthorized copying, modification, distribution, or use is prohibited without prior written permission.
See [LICENSE](LICENSE).

---

## Disclaimer

This is an academic research project. The measurements describe three specific machines under controlled conditions
and QChain's code of 10 October 2026; they are not a certification of any algorithm or product. The virtualised x86
server is a tested environment only. E13 is a model calibrated on one captured block, and E14 timed QChain's Fabric network as it was
configured for the study (ECDSA Fabric identities, one CouchDB shared by both peers); both are labelled as such
wherever they are used.
