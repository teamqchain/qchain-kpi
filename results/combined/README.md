<p align="center">
  <img src="../../.github/assets/qchain_logo.jpeg" width="120" alt="QChain logo" />
</p>

<h1 align="center">Combined results</h1>

<p align="center">
  <b>The tables and figures that combine the runs of the Mac, the virtualised x86 server and the Windows laptop.</b><br/>
  <a href="../../README.md">back to the main README</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/analysis%20tables-20%20CSV-0A66C2" alt="Analysis tables" />
  <img src="https://img.shields.io/badge/presentation%20tables-33%20CSV%20%2B%20Markdown-2EA44F" alt="Presentation tables" />
  <img src="https://img.shields.io/badge/figures-26%20PDF%20%2B%20PNG-6E40C9" alt="Figures" />
  <img src="https://img.shields.io/badge/generated-never%20edit%20by%20hand-red" alt="Generated" />
</p>

---

> ⚠️ **Generated, never edited by hand.** `python3 analysis/run_all.py` rewrites everything here from `results/mac`,
> `results/linux` and `results/windows`, and two runs give identical files. How each number is made: [analysis/README.md](../../analysis/README.md).

**What the numbers say:** [results-comparison.md](../../docs/results-comparison.md) (the three machines against the six research questions), and per machine [results-mac.md](../../docs/results-mac.md), [results-vm.md](../../docs/results-vm.md) and [results-windows.md](../../docs/results-windows.md).

**Machines:** `mac-m1` (Apple M1), `win-ryzen` (the AMD Ryzen 5 7520U laptop), `vm-x86v2` (the virtualised x86 server without AVX2). In every figure the Mac is blue, the laptop green and the VM orange, always in that order.

## Folders

| Folder | What it holds | Use it for |
|---|---|---|
| [`analysis_tables/`](analysis_tables/) | **T0 to T14**: every number at full precision, one row per cell, with intervals and statistics (CSV) | re-analysis, checking a number, the data behind a figure |
| [`tables/`](tables/README.md) | **presentation tables** for every experiment, the three machines side by side (CSV to paste into Word, Markdown to read) | the paper, the reports, slides |
| [`figures/`](figures/) | **F0 to F17**: every figure as PDF (for the paper) and PNG (for reports) | the paper, the reports, slides |
| `results/<machine>/figures/`, `results/<machine>/tables/` | one machine's own bar charts and tables, with every statistic of each cell ([Mac](../mac/README.md) · [VM](../linux/README.md) · [laptop](../windows/README.md)) | describing one machine |

---

## Every experiment, and where to find it

| Experiment | Machines | Analysis table | Presentation tables | Figures here | Per machine |
|---|---|---|---|---|---|
| E0 environment | all | (the run folders' `env/`) | E00 | – | E00 |
| Run quality | all | T0 | Q1 | F0 | Q1 |
| E1 primitive latency | all | T2, T2b | E01a to E01g | F1, F1a–F1c, F2, F2a–F2c, F3, F4, F9 | E01a, E01b; F1a–F1c, F2a–F2c |
| E2 integration cost | all (SLH-DSA on the Mac only) | T3 | E02a, E02b | F8 (right) | E02 |
| E3 integration layers | Mac | T8 | E03 | F8 (left) | – |
| E4 build sensitivity | all (each its own variants) | T7 | E04a, E04b | F9b | E04 |
| E5 native macOS | Mac | T9 | E05a, E05b | F16 | – |
| E6 message size | Mac | T10 | E06 | F13 (right) | – |
| E7 rejection timing | Mac | T11 | E07 | F13 (left) | – |
| E8 sizes | Mac (sizes do not depend on the machine) | T1 | E08 | F14, F4 | – |
| E9 memory | all | T4 | E09 | F10 | E09 |
| E10 issuer and holder | all | T6 | E10 | F15 | E10 |
| E11 lifecycle | all (all 176 pairs on the Mac) | T3b, T3c | E11a to E11d | F5, F5a, F6, F7 (left) | E11a, E11c; F5a, F5b |
| E12 concurrency | all | T12 | E12 | F12 | E12 |
| E13 ledger and IPFS bytes | Mac (a model) | T5, T5b, T5c | E13a to E13e | F11, F7 (right) | – |
| E14 Fabric latency | Mac | T13 | E14 | F17 | – |
| Sensitivity analysis | all | T14 | Q2 | – | – |

---

## `analysis_tables/` (CSV)

| Table | Content | Machines |
|---|---|---|
| `T0_run_quality.csv` | blocks, attempts, flagged share and causes, blocks without a headline, canary range, temperature, per run (E14: recorded once, not flagged) | all |
| `T1_E8_sizes.csv` | key, signature and ciphertext sizes of the 30 algorithms | Mac (sizes do not depend on the machine) |
| `T2_E1_primitive_latency.csv` | median, bootstrap interval, p99, mean, CoV, first call, per cell | all |
| `T2b_E1_machine_ratios.csv` | ratio of medians between machines, with interval and Cliff's delta | all pairs |
| `T3_E2_integration_overhead.csv` | reuse, reinit and production arms, their ratios and components | all (SLH-DSA on the Mac only) |
| `T3b_E11_lifecycle_per_suite.csv` | the five lifecycle stages and their sum per suite | all (176 pairs on the Mac, 16 suites elsewhere) |
| `T3c_E11_scaling_production_suite.csv` | issuance and verification against attributes and value size | all |
| `T4_E9_memory.csv` | stack, C heap, Go heap and process memory per cell | all |
| `T5_E13_ledger_issue_credential.csv` | ledger bytes per `issueCredential` (1, 5, 10, 20, 64 attributes) on the network as captured (`measured`), and three hypothetical encodings (`whatif-*`) | Mac (model) |
| `T5b_E13_ipfs_envelope_bytes.csv` | IPFS envelope bytes against attributes and value size | Mac (model) |
| `T5c_E13_ledger_every_transaction.csv` | ledger bytes of every QChain transaction (register, bind keys per KEM, issue, update, suspend, restore, revoke) | Mac (model) |
| `T6_E10_issuer_holder_equivalence.csv` | TOST per machine and ML-DSA set | all |
| `T7_E4_build_sensitivity.csv` | build variant against the production build, per cell | Mac and laptop (b2, b3), VM (b4) |
| `T8_E3_integration_layers.csv` | C, Go-to-C, Go binding and Go standard library per cell | Mac |
| `T9_E5_native_vs_container.csv` | native macOS against the container | Mac |
| `T10_E6_message_size.csv` | sign and verify against message size | Mac |
| `T11_E7_rejection_timing.csv` | rejection against acceptance time | Mac |
| `T12_E12_concurrency.csv` | throughput, speed-up, efficiency and latency against workers, with sensitivity columns | all |
| `T13_E14_fabric_latency.csv` | Fabric latency per payload point and mode (sequential, bursts of 40) from the main run `20261021-2135`: total p50 and p99, endorsement and commit p50, envelope bytes, MVCC conflicts | Mac |
| `T14_sensitivity_headline_vs_all_attempts.csv` | difference between the headline and all-attempts medians of every cell (5,030 cells) | all |

## `tables/` (CSV + Markdown)

33 presentation tables, listed with their titles in [tables/README.md](tables/README.md): Q1 and Q2 (run quality, sensitivity), E00 (environment), and E01a to E14 (one or more per experiment).

---

## `figures/` (PDF and PNG)

PDF for the paper, PNG for reports and slides. Every logarithmic axis says "(log scale)" and is labelled in plain units (10 µs, 1 ms, 1 KB, 2×); every bar carries its value. The machines keep their colours (Mac blue, laptop green, VM orange); algorithm families and lifecycle stages use other colours, so a colour never means two things.

| Figure | Content |
|---|---|
| `F0_run_quality` | share of flagged attempts per run, every main run of the three machines |
| `F1_signature_latency`, `F2_kem_latency` | every operation of every algorithm on the three machines (median and a line to p99) |
| `F1a`, `F1b`, `F1c` | signature schemes, **key generation / signing / verification**: median per call, one bar per machine |
| `F2a`, `F2b`, `F2c` | key-encapsulation mechanisms, **key generation / encapsulation / decapsulation**: median per call, one bar per machine |
| `F3_latency_tails` | complementary distribution of individual calls (ML-DSA-44 signing, Falcon-512 signing, ML-KEM-768 decapsulation) |
| `F4_size_vs_speed` | public key plus signature against sign plus verify, with Pareto frontiers |
| `F5_lifecycle_per_suite` | lifecycle cost per suite (median, 95 % interval) and the share of each stage (laptop) |
| `F5a_lifecycle_cost` | lifecycle cost of the 16 curated suites, one bar per machine |
| `F6_pair_heatmap` | lifecycle cost of all 176 signature and KEM pairs (Mac) |
| `F7_envelope_scaling` | issuance and verification time and IPFS envelope size against attributes and value size |
| `F8_integration_cost` | the Go binding (E3) and the deployed pattern (E2) against the library call |
| `F9_platform_comparison` | how much slower one machine is than another, by family |
| `F9b_build_sensitivity` | every liboqs build variant against the production build: portable and native on the Mac and the laptop, without OpenSSL on the VM |
| `F10_memory_stack` | stack high-water mark per call |
| `F11_ledger_footprint` | bytes per credential, credentials per block, GB per million credentials |
| `F12_concurrency_scaling` | throughput speed-up against workers |
| `F13_negative_path_and_message_size` | rejection timing by family: the range of the tests and their median, ratio on the x-axis, zoomed on 1× (E7); signing and verification against message size (E6) |
| `F14_key_and_signature_sizes` | public keys, signatures and ciphertexts, bytes, each algorithm in its family colour (darker: public key; lighter: signature or ciphertext) (E8) |
| `F15_issuer_holder_equivalence` | holder-side against issuer-side signing with the 90 % interval and each machine's margin (E10) |
| `F16_native_vs_container` | native macOS against the container, by family (E5) |
| `F17_fabric_latency` | Fabric latency per payload size, one at a time and in bursts, and the MVCC conflicts of the bursts (E14) |
