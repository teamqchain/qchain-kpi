# History of the study

How the KPI study went, from the first plan to the analysis, in the order things happened. It is written for someone who joins later, a reviewer, or an assistant that has to pick the work up. Every date is 2026; times are UAE local time (UTC+4) unless a run id is quoted (run ids are UTC). The *why* of each technical decision is in the [decisions log](decisions-log.md); the plan's own log is [plan.md §19](plan.md#19-decision-log).

---

## At a glance

```
 Sep 27 ─ Oct 10   planning, literature, machine checks          docs/plan.md v3.6, docs/coding-plan.md
 Oct 10 ─ Oct 22   harness built and every experiment run         Mac: E0 to E14 (results/mac)
 Oct 21            scope per machine decided                      methodology §9
 Oct 25            VM and Windows launchers and plans             plans/vm, plans/windows
 Oct 25 ─ Nov 01   VM runs, analysis, VM restored                 results/linux
 Oct 31            progress report 2 delivered (Mac results)
 Nov 06 ─ Nov 10   Windows laptop: set-up, trial, runs, reruns    results/windows
 Nov 10            all three machines reverted; analysis          analysis/, results/combined, docs/results-*.md
 Nov 11            documentation completed
 Nov 12            figures and presentation tables for the paper    results/combined, results/<machine>/
```

---

## 1. Planning (27 September to 10 October)

- **The question.** QChain signs credentials with ML-DSA-44 and encrypts them with ML-KEM-768. Earlier benchmarks in the project ([qchain-docs `algo-benchmarking/`](https://github.com/teamqchain/qchain-docs)) timed only ML-DSA in isolation. The study set out to measure every NIST post-quantum algorithm in liboqs the way QChain actually uses it, on more than one kind of machine, with the noise controls a paper needs. The six research questions are in [plan.md §1](plan.md#1-purpose-and-research-questions).
- **4 to 5 October.** Scope fixed: 25 post-quantum algorithms (ML-DSA, Falcon, all 12 SLH-DSA pure sets, ML-KEM, HQC) and 5 classical baselines; Go through liboqs-go as the measured layer, raw C only to measure the overhead; three platforms; no energy measurement; no phone timing; analytical ledger model instead of timing the blockchain.
- **4 October.** Read-only probe of the university VM: two unpinned vCPUs with a generic x86 model that has no AVX2.
- **6 to 9 October.** The VM's administrator confirmed that runs were allowed at any time and gave the host's details; the offer to change the VM's CPU type was declined so that the three machines cover three different code paths. **Decision of 7 October: exactly three machines**, the M1 Mac (ARM), Obeid's Windows laptop (x86 with AVX2) and the VM unchanged. **Framing rule of 7 October:** the paper calls the VM a tested environment, a virtualised x86 server, and never QChain's production environment. Obeid's laptop was inspected over AnyDesk on 8 October and nothing was changed.
- **10 October.** Research plan version 3.6 and the coding plan approved. The repository `teamqchain/qchain-kpi` was created (private). Decisions of that day include: Go 1.24.13 and liboqs 0.16.0 pinned everywhere, DHKEM written by hand on OpenSSL and checked against the RFC 9180 vectors, QChain's production functions copied verbatim from the then main repository, containers pinned to one CPU with no CPU quota, a canary benchmark every 10 minutes, thresholds calibrated from each machine's own trial and then frozen.

---

## 2. The Mac (10 to 22 October)

- **10 October: first trials, first problem.** The first Mac trials were noisy. Background analysis services of macOS (photo and media analysis, Spotlight's knowledge daemon and others) woke up during runs. Fix: the launcher pauses them during every run and resumes them after; from 11 October it also switches Wi-Fi off after the image build. The clean trial of 11 October calibrated the Mac's thresholds (drift 4.16 %, canary 3 %, E10 margins 2.0 to 2.09 %).
- **10 to 11 October: E1 and E10.** Four E1 blocks had no clean attempt, so the user decided that a finished run can be **extended**: only those blocks are run again inside the same run folder (`extend`), never as a separate rerun. This became a standing feature, tested for every new experiment.
- **11 October: drift rule.** Some blocks were slower than the rest of their cell with no external cause recorded (a per-process effect). The user decided that the original attempt stays the headline and a diagnostic re-run is kept beside it, reported in a sensitivity analysis.
- **11 October: Mac first.** Every remaining experiment was coded and run on the Mac before any other machine, each with unit tests and a smoke run.
- **11 to 21 October: the experiments, one by one.**

  | Date | Experiment | Run |
  |---|---|---|
  | 11 to 12 Oct | E2 integration cost (one attempt recorded twice after an extension bug, fixed; kept by decision) | `20261011-0514` |
  | 12 Oct | E8 sizes | `20261012-0309` |
  | 12 to 13 Oct | E11 lifecycle, all 176 pairs (26 hours) | `20261012-0605` |
  | 13 Oct | E13 ledger model, after a read-only capture of a block and the IPFS store on the live VM | `20261013-1252` |
  | 13 to 14 Oct | E6 message size | `20261013-2220` |
  | 14 Oct | E7 rejection timing | `20261014-1822` |
  | 15 Oct | E9 memory, E12 concurrency | `20261015-0300`, `20261015-0837` |
  | 16 to 17 Oct | E3 integration layers | `20261016-1904` |
  | 18 to 19 Oct | E4 portable and native builds | `20261018-2205`, `20261019-1901` |
  | 20 Oct | E5 native macOS (built outside Docker) | `20261020-1525` |
  | 21 to 22 Oct | E14 Fabric latency on QChain's network rebuilt in Colima | `20261021-2135` |

- **21 October: scope per machine.** To finish in time, experiments whose answer does not depend on the processor (E3, E5, E6, E7, E8, E13, E14) stayed Mac-only; the VM and the laptop run E1, E2 without SLH-DSA, E4, E9, E10, E11 on 16 curated suites and E12 ([methodology §9](methodology.md#9-which-experiments-run-on-which-machine-decided-21-oct-2026)).
- **22 October: a finding about QChain itself.** E14 showed that QChain's two peers shared one CouchDB, so bursts of transactions met MVCC read conflicts (410 of 1,800 burst transactions). The user chose to record them rather than change the network.
- **23 October.** The Mac's own settings (Spotlight, Siri, Live Text, automatic updates) were restored; Colima, Go and the native toolchain stayed until the end of the study.

---

## 3. The university VM (25 October to 1 November)

- **25 October.** The VM and Windows launchers and their plans were written and tested on the Mac in throwaway containers; the repository stays private until the paper is published. First VM smoke run with QChain still live: the pipeline worked; a monitor bug was fixed.
- **26 October.** QChain's services, IPFS, MySQL and the GitHub runner were stopped (nothing deleted). The trial calibrated the VM's thresholds. Its first canary of each batch was too random, so the user chose **a fixed canary reference, the median canary of the trial**. The add-on trials for E2 and E11 followed. **Series 1:** E1, E10, E9, E12 (one and two workers; three blocks extended).
- **30 to 31 October. Series 2:** E2 without SLH-DSA, E11 on the curated suites (four blocks extended), E4 without OpenSSL.
- **31 October.** Progress report 2 delivered with the Mac's results.
- **1 November.** The VM's runs were analysed (no rerun needed; the canary-flagged attempts were not slower than the clean ones). QChain was restarted and verified, and the VM was reverted: images, builder, clone and logs removed.

---

## 4. The Windows laptop (6 to 10 November)

The laptop is Obeid's own computer, reached over AnyDesk with a free licence (sessions of about 15 minutes). Every command was relayed one step at a time, read-only first, and every change was written into a revert list before it was made.

- **6 November, evening.** Read-only inspection. A launcher bug was fixed before its first run (a missing manifest field would have stopped every run after it had finished), and a series script was added so that eight plans could run unattended. Noise decisions: Windows Update paused for a week, Obeid's Ubuntu distribution stopped, background apps quit, the clone at `C:\qchain-kpi`, Defender and maintenance left alone.
- **6 to 7 November, night.** Smoke runs of every plan; the first push from the laptop.
- **7 November.** The trial (`20261106-2212`, accepted: 3 of 106 main attempts flagged, no heat flags). Found while calibrating: `kpi-batch calibrate` built on macOS ignores the host monitor when it re-evaluates flags, so the laptop's thresholds were computed with a Linux-rule build (no code change; the VM's thresholds were checked and are not affected). The fixed canary reference was adopted, as on the VM. The add-on trials followed.
- **7 November 11:15 to 9 November 05:44. Main series** (42.5 hours): E1, E10, E9, E12 (one to eight workers), E2 without SLH-DSA, E11 on the curated suites, E4 portable and native. A paste of old window text into PowerShell on the 7th ran nothing harmful (checked line by line).
- **9 November.** 32 blocks had no clean attempt. Obeid needed the laptop for the day, so only a restart was done (nothing deleted).
- **10 November, night.** After a full read-only check of everything running, two rerun rounds: 32 blocks, of which 14 resolved; then the remaining 18, of which one resolved. **The user decided to keep the last 17 blocks as they are** (16 E12 blocks at four workers, where the measured load itself sits at the 5 % limit, and one long E2 block): no change of limits after the fact, no further reruns, results from the pre-declared sensitivity analysis.
- **10 November.** The laptop was reverted step by step (images and builder, clone, Docker's Ubuntu integration, Obeid's `qchain-api` container, Windows Update, PowerShell history, a final restart) and handed back. Only Docker's disk file stayed 6 GB larger, by the user's choice.

---

## 5. Revert, analysis and documentation (10 to 11 November)

- **10 November: everything reverted.** The VM's last leftovers (the E13 capture folder, three logs and a builder image) and the Mac's tools (Colima's virtual machine, Docker, Go, the native toolchain and caches) were removed. The test environments of all three machines are back to their state before the study. No run can be repeated without a new set-up.
- **10 November: analysis.** `analysis/` was written and run; 19 tables and 15 figures in `results/combined/`. Two figure defects and several mistakes in the first drafts of the results documents were found by checking every claim against the data and were fixed.
- **11 November: documentation.** The results of the Mac and the VM got their own documents; the E14 table was corrected to use the main run (6,360 transactions) instead of the smoke run; the figures were made reproducible (two runs of the analysis now give byte-identical files); this history, a glossary and a documentation guide were added; every README was rewritten in the style of the team's other repositories.

- **12 November: figures and tables for the paper.** Bar charts in the style of the project poster were added for every E1 operation and the lifecycle, for the three machines together and for each machine alone, with new figures for E8, E10, E5 and E14; E7 was redrawn with the ratio on the x-axis; every log axis now says "(log scale)" and is labelled in plain units; the machine colours were made exclusive; presentation tables (CSV and Markdown) were added for every experiment, and the analysis tables moved to `results/combined/analysis_tables/`. Two gaps found on the way were closed: the E14 run was missing from the run-quality table, and the native builds (E4, b3) were in no figure.

---

## Where the measured code lives now

The KPI study measured QChain's code as it was on 10 October (commit `9026aba` of the then single repository). On 9 and 10 November that repository was split into eight (`qchain-backend`, `qchain-network`, ...). [provenance.md](provenance.md) maps the copied code from `9026aba` to its new home (`qchain-backend` commit `d2c0629`, byte-identical).
