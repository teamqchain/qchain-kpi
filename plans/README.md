<p align="center">
  <img src="../.github/assets/qchain_logo.jpeg" width="120" alt="QChain logo" />
</p>

<h1 align="center">Run plans</h1>

<p align="center">
  <b>What each machine runs: one folder of JSON plans per machine.</b><br/>
  <a href="../README.md">back to the main README</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/mac-every%20experiment-2F3134" alt="Mac" />
  <img src="https://img.shields.io/badge/vm%20%C2%B7%20windows-reduced%20scope%20of%2021%20Oct-2F3134" alt="VM and Windows" />
  <img src="https://img.shields.io/badge/used%20plans-never%20edited-red" alt="Never edited" />
</p>

---

## Folders

Each machine runs the plans in its own folder, and its launcher reads only that folder.

| Machine | Folder | Launcher |
|---|---|---|
| Apple M1 MacBook Air (`mac-m1`) | `plans/mac/` | `scripts/run-mac.sh` (E5: `scripts/e5-run-mac.sh`) |
| University VM, a virtualised x86 server (`vm-x86v2`) | `plans/vm/` | `scripts/run-linux.sh` |
| Obeid's Windows laptop (`win-ryzen`) | `plans/windows/` | `scripts/run-windows.ps1` |
| E14 (Mac only) | `fabric/plans/` | `scripts/e14-run-mac.sh` |

Which experiments run where was decided on 21 October 2026 ([methodology §9](../docs/methodology.md#9-which-experiments-run-on-which-machine-decided-21-oct-2026)); `internal/plan/machines_test.go` checks the folders against it.

---

## Rules

- **The experiment code is the same on every machine.** Plans only choose what to run (algorithms, suites, workers, repetitions), so a difference between machines comes from the machine, not the code.
- **A plan that is the same on several machines is copied into each folder.** The same file name in several folders always means a byte-identical copy; a plan whose coverage differs gets its own name (`e2-noslh`, `e11-curated`, `e12-vm`). A plan that names container CPUs (`cpuset`, E12) is named for its machine.
- **Plans used for a machine's runs are never edited or replaced.** Every run also stores its full plan in its `manifest.json`, and a test checks every Mac plan, and the blocks it expands to, against the version that ran.
- **A trial estimates the main plans of its own folder** (the folder of the trial plan).

---

## Mac (`plans/mac/`): every Mac run

| Plans | Purpose |
|---|---|
| `smoke`, `smoke-quick` | pipeline tests: all algorithms with tiny budgets, or a few |
| `trial` | calibration of the machine's thresholds and fast/slow classes |
| `trial-e2`, `trial-e6`, `trial-e7`, `trial-e11` | add-on calibrations before the first E2, E6, E7 and E11 runs |
| `smoke-e2`, `smoke-e3`, `smoke-e4`, `smoke-e5`, `smoke-e6`, `smoke-e7`, `smoke-e8`, `smoke-e9`, `smoke-e11`, `smoke-e12`, `smoke-e13` | pipeline tests of each experiment |
| `e1`, `e2`, `e3`, `e4`, `e5`, `e6`, `e7`, `e8`, `e9`, `e10`, `e11`, `e12`, `e13` | main runs |

- `e4` and `smoke-e4` need a `--variant` other than `b1-prod`.
- `e5` and `smoke-e5` run only natively, with `scripts/e5-run-mac.sh`.
- `e12` and `smoke-e12` give the container 4 CPUs (`cpuset`); every other plan pins it to one.
- Some plan descriptions still name `run.sh`, the Mac launcher's name when they ran (now `run-mac.sh`); the files are kept byte for byte.

---

## VM (`plans/vm/`) and Windows (`plans/windows/`)

Both folders hold the same 17 plans, except E12's, which name each machine's CPUs.

| Plans | Purpose |
|---|---|
| `smoke`, `smoke-quick` | pipeline tests (copies of the Mac's) |
| `trial` | calibration of the machine's thresholds and fast/slow classes (copy of the Mac's) |
| `trial-e2-noslh`, `trial-e11-curated` | add-on calibrations before the first `e2-noslh` and `e11-curated` runs |
| `smoke-e2-noslh`, `smoke-e4`, `smoke-e9`, `smoke-e11-curated`, `smoke-e12-vm` or `smoke-e12-windows` | pipeline tests of each experiment |
| `e1`, `e4`, `e9`, `e10` | main runs, copies of the Mac's (450, 450, 1,350 and 15 blocks) |
| `e2-noslh` | E2 and E2c for the 18 algorithms outside SLH-DSA: 108 cells, 540 blocks |
| `e11-curated` | E11 on the 16 curated suites: 501 of the grid's 991 cells, 2,505 blocks |
| `e12-vm` | E12 with W = 1 and 2 on both vCPUs (`cpuset` 0-1): 80 blocks |
| `e12-windows` | E12 with W = 1, 2, 4 and 8 on the 8 CPUs of Docker's WSL2 VM (`cpuset` 0-7): 160 blocks |

- Every main plan needs the machine's `trial` first; `e2-noslh` and `e11-curated` also need their add-on trial, which probes exactly their cells.
- `e11-curated` keeps every cell of the 16 curated suites at every N and k, the production suite's 256- and 4096-byte value cells and the 13 chained runs. It drops the 490 cells that exist only for the all-pairs sweep (N = 10, k = 5), which ran on the Mac: 160 activations, 160 issuances, 10 presentations and 160 verifications.
- `e4` and `smoke-e4` run on the machine's own liboqs builds: `--variant b4-noossl` on the VM; `b2-portable` and `b3-native` on Windows. The launchers refuse other builds, and any build but `b1-prod` for plans without E4.
- The descriptions of `e4` and `smoke-e4` name `run.sh`; on these machines the launchers are `scripts/run-linux.sh` and `scripts/run-windows.ps1`, with the same options.

---

## What actually ran

| Machine | Main plans run | Where the runs are |
|---|---|---|
| Mac | all 13 above, plus `e14` (Fabric) | [results/mac/INDEX.md](../results/mac/INDEX.md) |
| VM | `e1`, `e10`, `e9`, `e12-vm`, `e2-noslh`, `e11-curated`, `e4@b4-noossl` | [results/linux/INDEX.md](../results/linux/INDEX.md) |
| Windows | `e1`, `e10`, `e9`, `e12-windows`, `e2-noslh`, `e11-curated`, `e4@b2-portable`, `e4@b3-native` | [results/windows/INDEX.md](../results/windows/INDEX.md) |
