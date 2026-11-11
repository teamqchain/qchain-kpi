<p align="center">
  <img src="../../.github/assets/qchain_logo.jpeg" width="120" alt="QChain logo" />
</p>

<h1 align="center">Combined results</h1>

<p align="center">
  <b>The tables and figures that combine the runs of the Mac, the virtualised x86 server and the Windows laptop.</b><br/>
  <a href="../../README.md">back to the main README</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/tables-19%20CSV-0A66C2" alt="Tables" />
  <img src="https://img.shields.io/badge/figures-15%20PDF%20%2B%20PNG-6E40C9" alt="Figures" />
  <img src="https://img.shields.io/badge/generated-never%20edit%20by%20hand-red" alt="Generated" />
</p>

---

> ⚠️ **Generated, never edited by hand.** `python3 analysis/run_all.py` rewrites everything here from `results/mac`,
> `results/linux` and `results/windows`. How each number is made: [analysis/README.md](../../analysis/README.md).

**What the numbers say:** [results-comparison.md](../../docs/results-comparison.md) (the three machines against the six research questions), and per machine [results-mac.md](../../docs/results-mac.md), [results-vm.md](../../docs/results-vm.md) and [results-windows.md](../../docs/results-windows.md).

**Machine ids** in every table: `mac-m1` (Apple M1), `vm-x86v2` (the virtualised x86 server without AVX2), `win-ryzen` (the AMD Ryzen 5 7520U laptop).

---

## `tables/` (CSV)

| Table | Content | Machines |
|---|---|---|
| `T0_run_quality.csv` | blocks, attempts, flagged share and causes, blocks without a headline, canary range, temperature, per run | all |
| `T1_E8_sizes.csv` | key, signature and ciphertext sizes of the 30 algorithms | Mac (sizes do not depend on the machine) |
| `T2_E1_primitive_latency.csv` | median, bootstrap interval, p99, mean, CoV, first call, per cell | all |
| `T2b_E1_machine_ratios.csv` | ratio of medians between machines, with interval and Cliff's delta | all pairs |
| `T3_E2_integration_overhead.csv` | reuse, reinit and production arms, their ratios and components | all (SLH-DSA on the Mac only) |
| `T3b_E11_lifecycle_per_suite.csv` | the five lifecycle stages and their sum per suite | all (176 pairs on the Mac, 16 suites elsewhere) |
| `T3c_E11_scaling_production_suite.csv` | issuance and verification against attributes and value size | all |
| `T4_E9_memory.csv` | stack, C heap, Go heap and process memory per cell | all |
| `T5_E13_ledger_issue_credential.csv` | ledger bytes per `issueCredential`, as captured and projected with ML-DSA-44 Fabric identities | Mac (model) |
| `T5b_E13_ipfs_envelope_bytes.csv` | IPFS envelope bytes against attributes and value size | Mac (model) |
| `T6_E10_issuer_holder_equivalence.csv` | TOST per machine and ML-DSA set | all |
| `T7_E4_build_sensitivity.csv` | build variant against the production build, per cell | Mac and laptop (b2, b3), VM (b4) |
| `T8_E3_integration_layers.csv` | C, Go-to-C, Go binding and Go standard library per cell | Mac |
| `T9_E5_native_vs_container.csv` | native macOS against the container | Mac |
| `T10_E6_message_size.csv` | sign and verify against message size | Mac |
| `T11_E7_rejection_timing.csv` | rejection against acceptance time | Mac |
| `T12_E12_concurrency.csv` | throughput, speed-up, efficiency and latency against workers, with sensitivity columns | all |
| `T13_E14_fabric_latency.csv` | Fabric latency per payload point and mode (sequential, bursts of 40) from the main run `20261021-2135`: total p50 and p99, endorsement and commit p50, envelope bytes, MVCC conflicts | Mac |
| `T14_sensitivity_headline_vs_all_attempts.csv` | difference between the headline and all-attempts medians of every cell (5,030 cells) | all |

---

## `figures/` (PDF and PNG)

PDF for the paper, PNG for reports and slides. Colours are Okabe-Ito (colour-blind safe), one colour per machine.

| Figure | Content |
|---|---|
| `F0_run_quality` | share of flagged attempts per run |
| `F1_signature_latency`, `F2_kem_latency` | every operation of every algorithm on the three machines (log scale; median to p99) |
| `F3_latency_tails` | complementary ECDF of individual calls (ML-DSA-44 signing, Falcon-512 signing, ML-KEM-768 decapsulation) |
| `F4_size_vs_speed` | public key plus signature against sign plus verify, with Pareto frontiers |
| `F5_lifecycle_per_suite` | lifecycle cost per suite and the share of each stage |
| `F6_pair_heatmap` | lifecycle cost of all 176 signature and KEM pairs (Mac) |
| `F7_envelope_scaling` | issuance and verification time and IPFS envelope size against attributes and value size |
| `F8_integration_cost` | the Go binding (E3) and the deployed pattern (E2) against the library call |
| `F9_platform_comparison`, `F9b_build_sensitivity` | machine ratios by family; build variants against production |
| `F10_memory_stack` | stack high-water mark per call |
| `F11_ledger_footprint` | bytes per credential, credentials per block, GB per million credentials |
| `F12_concurrency_scaling` | throughput speed-up against workers |
| `F13_negative_path_and_message_size` | rejection timing (E7) and message-size scaling (E6) |
