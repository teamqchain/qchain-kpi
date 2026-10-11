# Decisions log (harness)

The decisions about the study itself (scope, machines, run rules, thresholds) are in the research plan's decision log, `docs/plan.md` §19. This file records the decisions taken while building and testing the harness. Each one is dated and gives its reason.

| Date | Decision | Why |
|---|---|---|
| 10 Oct | Coding plan approved with the refinements of its §13, and Go 1.24.13 everywhere | production's builder image resolves to Go 1.24.13 |
| 10 Oct | liboqs pinned to the 0.16.0 release; liboqs-go `v0.16.0` | production builds an unpinned `main`; a study needs a fixed version |
| 10 Oct | DHKEM implemented by hand on OpenSSL 3.0 primitives and checked against the 8 RFC 9180 vectors | OpenSSL 3.0 (Debian bookworm) has no HPKE API |
| 10 Oct | Production functions copied verbatim (commit `9026aba`); only the signature algorithm name is parameterised | user decision: measure the real code, with provenance |
| 10 Oct | Input sets generated per (algorithm, repetition) by a separate untimed process | No crypto runs before the cold call. The repetitions include key-to-key variation. |
| 10 Oct | The probe pass uses its own small input set (P = 6) | a cold call plus 5 calls needs 6 inputs; full SLH-DSA-s pools would add minutes per algorithm |
| 10 Oct | Classical adapters perform no cryptographic operation when constructed. The RSA public key size is fixed at 422 bytes (SPKI DER, 3072-bit, e = 65537); the private key length is variable. | Constructing the RSA object used to generate a key, which would have run crypto before the cold call |
| 10 Oct | Raw samples are kept in memory during the timed phase and written afterwards | no file I/O or compression between samples |
| 10 Oct | Keygen correctness: every key's lengths are checked on the spot; a seeded subset (every 50th, at least 3) is fully exercised after the timed phase | approved subset rule. Deferring the full checks keeps their cache effects out of the timed samples. |
| 10 Oct | **Containers pinned to one CPU with no `--cpus` quota** (user decision) | In the Mac smoke run, a quota equal to the pinned CPU made Linux pause a fully busy container in 29 of 100 periods, about 1.8 ms each time. Every pause lands inside a timed sample, and the quota flag would have repeated every block. kpi-batch now refuses trial and main runs if a quota is set; the throttling check stays as a sanity check. |
| 10 Oct | Preflight for trial and main runs also refuses while Microsoft Edge is open (user decision) | Edge is the browser actually in use on the Mac; its tabs use CPU unpredictably |
| 10 Oct | Host-monitor chunks are delivered with `docker cp` every 10 s (a new chunk each minute); session facts arrive as `session.json` before every start or resume | no bind mounts (spaced paths, Colima without mounts), and no process runs inside the measurement container |
| 10 Oct | The Mac monitor skips `top`'s first two samples | the second sample still has empty per-process figures |
| 10 Oct | `pgrep -a` in the launcher's preflight | macOS `pgrep` skips its own ancestors by default, so VS Code was not seen when the launcher ran inside it |
| 10 Oct | Results are verified (`SHA256SUMS`) and privacy-scanned in `scratch/staging/` before they are moved into `results/` | a failed check must never leave files where they could be committed |
| 10 Oct | Mac preflight (trial and main) refuses while automatic macOS updates are on (`AutomaticDownload` or `AutomaticallyInstallMacOSUpdates`). It notes automatic App Store updates. | Night-time inventory: on this Mac both were on, so macOS could download an update and restart during a run and change the OS mid-study. Time Machine has no destination. Spotlight is on; the host monitor catches its activity. |
| 10 Oct | The host monitor and chunk delivery start at the very beginning of a session, before the clock-offset measurement | so the batch's first blocks are covered by warmed-up host samples |
| 10 Oct | `run.sh` starts Colima automatically if it is down (user choice, option 2) | The Colima VM does not survive a Mac sleep, and the Mac sleeps whenever it is idle between runs (sleep log: idle and clamshell sleeps on 10 Oct). The launcher restarts it with the study settings: vz, 4 CPUs, 4 GiB, no mounts. |
| 10 Oct | **macOS background analysis services are paused during every Mac run** (user decision). The services are `mediaanalysisd`, `photoanalysisd`, `spotlightknowledged` (and its importer and updater), `suggestd`, `duetexpertd`, `knowledge-agent` and `knowledgeconstructiond`. They are paused with SIGSTOP at session start (re-checked every 10 s) and resumed with SIGCONT on every exit path; the PIDs and names are logged in `launcher/bg-paused-s<N>.txt`. | Four Mac trials on 10 Oct were dominated by these services. They run whenever the Mac is idle on mains power, which are exactly the test conditions. Switching off Siri, the Spotlight suggestions, the Spotlight app results and Live Text, and pausing Spotlight indexing, did not stop them. In the fourth trial, Photos analysis used up to 2 cores and drove the fanless Mac to thermal pressure level 1: other CPU had a median of 34%, 224 of 357 attempts were flagged and 60 blocks were exhausted. |
| 11 Oct | **The launcher switches Wi-Fi off for the run and back on at the end** (user idea). It switches off after the image build, before the session starts, because the build needs the internet and the measurement does not. It restores Wi-Fi when the script ends on every exit path (normal end, error, Ctrl-C, closed Terminal), and only if Wi-Fi was on before. It is the Mac only: the VM must stay reachable over SSH and Obeid's laptop is operated over AnyDesk (phase 1b decides what, if anything, to do there). | It makes the per-run manual step unnecessary and removes App Store, iCloud and update traffic from the measurement without changing any setting permanently. The 10 Oct trial with Wi-Fi off was the clean one. |
| 11 Oct | The background services are now paused before the post-build cool-down and the quiet-machine check, not only when the session starts (launcher fix). | The first E1 main attempt was refused: "host CPU busy at 14%". During the 120 s cool-down the Mac was idle, so the services woke up and failed the check before the session had paused them. The services paused early are logged in the run's `launcher/bg-paused-s<N>.txt`; the cleanup routine resumes them on every exit, including a refused start. |
| 11 Oct | **A finished run can be extended** (`run.sh extend`, `kpi-batch run --extend --allow-rebuild`): blocks that ended with no clean attempt are run again inside the same run folder, with a fresh budget of 3 attempts (a4, a5, a6). User decision: no separate plan and no separate folder. It is recorded as session 2 (manifest, state, journal, console), and the launcher requires the measuring sources to be identical to the run's harness commit. | The E1 main run had 4 of 450 blocks without a clean attempt (all long operations hit by short macOS bursts: `XProtectRemediator`, `mobileassetd`, `PerfPowerService`, which run as root and cannot be paused). Re-running the whole run (6 h) for 4 blocks was not sensible, and a separate plan would split the data. |
| 11 Oct | `AddressBookSourceSync` added to the paused services. The other noisy processes of the E1 run (`XProtectRemediator*`, `mobileassetd`, `PerfPowerService`, `kernel_task`) run as root or in the kernel, so a normal user cannot pause them; they were 2% of the samples. | A signal to a root process needs admin rights, which the launcher deliberately does not use. |
| 11 Oct | The extension safety check compares the **measuring** sources (`cmd/kpi-prim`, `cmd/kpi-env`, `cmd/kpi-canary-native`, `internal/`, `docker/`, `plans/`, `config/`, `go.mod`, `go.sum`), not `cmd/kpi-batch` (user approved). The `cmd/kpi-batch` files that differ are recorded in the manifest. | The first attempt to extend the E1 run was refused because the extension feature itself changed `cmd/kpi-batch`. The measuring code was identical; only the orchestrator that implements the extension differed. |
| 11 Oct | **Drift-flagged blocks are kept under the approved rule** (user decision, option 1): the original attempt stays the headline, the diagnostic re-run is kept beside it, and the effect is reported. No further diagnostic experiment for now. | 13 of 465 headline blocks of the Mac E1 and E10 runs were 5% to 54% slower than their cell reference, with no sign of outside interference. Changing the headline rule after the runs would bias the results. See the record below. |

## Record of the Mac trials of 10 Oct 2026 (issue and fix)

Five Mac trials were run on 10 Oct. Only the last one is kept (`results/mac/20261010-1819_...`); the other four were deleted on 11 Oct (user decision), because they were diagnostic runs and not results. This record keeps what they showed.

**Problem.** macOS background analysis services (the Spotlight AI "knowledge" indexer `spotlightknowledged` and Photos analysis `mediaanalysisd`, plus smaller ones) ran whenever the Mac was idle on mains power, which are exactly the test conditions. They used up to about two of the eight cores. The "other CPU" quality rule (limit 5%) therefore flagged blocks, and flagged blocks are repeated up to twice.

| Trial | What was different | Flagged attempts | Notes |
|---|---|---|---|
| 1 (run id 1301) | first trial, VS Code and the display in use | 76 of 264 | the monitor could not yet name the process |
| 2 (run id 1404) | hands-off, monitor names the process | 227 of 356 | median other CPU high; `spotlightknowledged` in about 85% of the noisy samples |
| 3 (run id 1526) | Spotlight indexing off (`mdutil`) | 149 of 317 | 25 blocks without a clean attempt; not enough |
| 4 (run id 1700) | Siri, Spotlight suggestions and app results, Live Text switched off | 224 of 357 | median other CPU 34%, thermal pressure on 81 samples (fanless Mac), 60 blocks without a clean attempt, over an hour |
| 5 (run id 1819, kept) | the launcher pauses the services and Wi-Fi is off | 1 of 194 | median other CPU 1.1%, no thermal pressure, no block without a clean attempt, about 25 minutes |

**What did not help:** switching off Siri, Spotlight suggestions and results, Live Text and Spotlight file indexing. Those switches stop features, not the background work already queued (about 305,000 items had been processed and thousands were still waiting).

**Fix:** `scripts/run.sh` pauses the services during every Mac run and resumes them at the end, and switches Wi-Fi off for the run after the image build.

**Side finding:** for 25 cells measured both in noisy and in clean attempts, the block medians were identical (median difference 0.000%, range -0.6% to +1.8%; spread among clean attempts 0.9%). The background load cost time and flags, not accuracy, because the benchmark is pinned to one core. The 5% rule was kept anyway (a relaxed rule was not approved).

## Record of the drift-flagged blocks (Mac E1 and E10, 11 Oct 2026)

**What was seen.** 13 of 465 headline blocks (12 of 450 in E1, 1 of 15 in E10) were flagged for drift. Their medians were 5.3, 5.6, 6.8, 7.1, 7.4, 7.8, 14.6, 18.4, 27.5, 36.9, 38.6, 48.5 and 53.9% above the cell reference (the median of the first attempts of the 5 repetitions).

- **The whole block was slow, evenly.** Split into ten parts in sample order, every part was slow by the same amount; in most of them even the fastest sample was slower than the normal median. The whole process ran slow from start to end, rather than being interrupted.
- **Nothing outside explains it.** Host other CPU was about 1%, thermal pressure 0, steal 0, the canary P-like and in range.
- **It was limited to that process.** The neighbouring blocks were normal, and 12 of the 13 diagnostic re-runs were normal (the 13th was +8%). Twice, two consecutive processes were slow (01:20 to 01:22 and 04:02).
- **In E10 all three arms slowed equally (+27.5%),** so the within-block issuer-versus-holder comparison is unaffected.
- **The clean trial** had 4 blocks at +5% to +7% and none larger.

**A separate, smaller effect.** The pinned CPU ran at two speed levels about 1.9% apart, also visible in the canary (ratios 0.974 and 0.992). It explains the ordinary spread of 1% to 3% (57 of 450 E1 blocks) and stays inside the 4.16% drift limit.

**Likely cause (not proven).** Most likely a per-process effect: each block is a fresh process with randomised addresses and different physical pages, and some layouts make the same code consistently slower for the life of that process. This is a known benchmarking effect (memory-layout bias; to be verified and cited in the paper). A lower host clock for the VM CPU for a minute or two is the alternative; it would explain the two consecutive slow pairs, but not why every change coincided with a block boundary. The runs on the Windows laptop and the VM will show whether the effect is specific to the Mac.

**Impact.** Cell results use the median across the 5 repetitions, which one slow repetition does not move. The bootstrap intervals resampled over repetitions (plan section 9) widen for the affected cells, which correctly reflects the variation between processes. The planned sensitivity analysis reports the results with and without the drift-flagged attempts.
