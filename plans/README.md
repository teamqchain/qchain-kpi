# Run plans, one folder per machine

Each machine runs the plans in its own folder, and its launcher reads only that folder:

| Machine | Folder | Launcher |
|---|---|---|
| M1 MacBook Air (`mac-m1`) | `plans/mac/` | `scripts/run-mac.sh` (E5: `scripts/e5-run-mac.sh`) |
| UoS VM (`vm-x86v2`) | `plans/vm/` | `scripts/run-linux.sh` |
| Obeid's laptop (`win-ryzen`) | `plans/windows/` | `scripts/run-windows.ps1` |

Rules:
- **The experiment code is the same on every machine.** Plans only choose what to run (algorithms, suites,
  workers, repetitions), so a difference between machines comes from the machine, not the code.
- **A plan that is the same on several machines is copied into each folder**, and a test keeps the copies
  byte-identical to the Mac's original. A plan that differs gets its own file in that machine's folder.
- **Plans used for a machine's runs are never edited or replaced.** Every run also stores its full plan in its
  `manifest.json`.
- A trial estimates the main plans **of its own folder** (the folder of the trial plan).
- E14's plans live with its Fabric client in `fabric/plans/` (Mac only).

Which experiments run where was decided on 21 Oct (`docs/methodology.md`, section 9).

## Mac (`plans/mac/`): every Mac run so far

| Plans | Purpose |
|---|---|
| `smoke`, `smoke-quick` | pipeline tests: all algorithms with tiny budgets, or a few |
| `trial` | calibration of the machine's thresholds and fast/slow classes |
| `trial-e2`, `trial-e6`, `trial-e7`, `trial-e11` | add-on calibrations before the first E2, E6, E7 and E11 runs |
| `smoke-e2`, `smoke-e3`, `smoke-e4`, `smoke-e5`, `smoke-e6`, `smoke-e7`, `smoke-e8`, `smoke-e9`, `smoke-e11`, `smoke-e12`, `smoke-e13` | pipeline tests of each experiment |
| `e1`, `e2`, `e3`, `e4`, `e5`, `e6`, `e7`, `e8`, `e9`, `e10`, `e11`, `e12`, `e13` | main runs |

Notes: `e4` and `smoke-e4` need a `--variant` other than `b1-prod`; `e5` and `smoke-e5` run only natively with
`scripts/e5-run-mac.sh`; `e12` and `smoke-e12` give the container 4 CPUs (`cpuset`), every other plan pins it to one.
Some plan descriptions still name `run.sh`, the Mac launcher's name when they ran (now `run-mac.sh`); the files are
kept byte for byte.
