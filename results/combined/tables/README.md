<p align="center">
  <img src="../../../.github/assets/qchain_logo.jpeg" width="120" alt="QChain logo" />
</p>

<h1 align="center">Paper tables</h1>

<p align="center">
  <b>Every experiment as a table to read or paste: the three machines side by side, and the Mac-only experiments.</b><br/>
  <a href="../README.md">back to the combined results</a> · <a href="../../../README.md">main README</a>
</p>

---

> ⚠️ **Generated, never edited by hand.** `python3 analysis/paper_tables.py` (run by `analysis/run_all.py`) writes every file here.
> Each table exists twice: `.csv` (numbers only, to paste into Word or a spreadsheet) and `.md` (formatted, readable on GitHub).
> Full precision is in [`../analysis_tables/`](../analysis_tables/); one machine's tables with every statistic are in `results/<machine>/tables/`.

| Table | Content |
|---|---|
| [`E00_environment`](E00_environment.md) · [csv](E00_environment.csv) | Environment of the measurements (E0) |
| [`E01a_signature_latency`](E01a_signature_latency.md) · [csv](E01a_signature_latency.csv) | Signature schemes: median latency per call (µs) |
| [`E01b_kem_latency`](E01b_kem_latency.md) · [csv](E01b_kem_latency.csv) | Key-encapsulation mechanisms: median latency per call (µs) |
| [`E01c_tail_latency`](E01c_tail_latency.md) · [csv](E01c_tail_latency.csv) | 99th-percentile latency per call (µs), and p99 against the median |
| [`E01d_first_call`](E01d_first_call.md) · [csv](E01d_first_call.csv) | The first call in a fresh process (µs) |
| [`E01e_machine_ratios`](E01e_machine_ratios.md) · [csv](E01e_machine_ratios.csv) | How much slower one machine is than another, by family (E1) |
| [`E01f_latency_tails`](E01f_latency_tails.md) · [csv](E01f_latency_tails.csv) | Latency tails of one repetition (µs), the data of figure F3 |
| [`E01g_size_vs_speed`](E01g_size_vs_speed.md) · [csv](E01g_size_vs_speed.csv) | Size against speed for the signature schemes, the data of figure F4 |
| [`E02a_integration_by_family`](E02a_integration_by_family.md) · [csv](E02a_integration_by_family.csv) | What QChain's deployed pattern adds, by family (E2) |
| [`E02b_integration_per_operation`](E02b_integration_per_operation.md) · [csv](E02b_integration_per_operation.csv) | Reused key against the production pattern, every operation (µs) |
| [`E03_integration_layers`](E03_integration_layers.md) · [csv](E03_integration_layers.csv) | Integration layers on the Mac (E3): C library, Go-to-C call, Go binding, Go's own library |
| [`E04a_build_by_family`](E04a_build_by_family.md) · [csv](E04a_build_by_family.csv) | liboqs build variants against the production build, by family (E4) |
| [`E04b_build_per_operation`](E04b_build_per_operation.md) · [csv](E04b_build_per_operation.csv) | liboqs build variants against the production build, every operation (E4) |
| [`E05a_native_by_family`](E05a_native_by_family.md) · [csv](E05a_native_by_family.csv) | Native macOS against the Linux container on the Mac, by family (E5) |
| [`E05b_native_per_operation`](E05b_native_per_operation.md) · [csv](E05b_native_per_operation.csv) | Native macOS against the container, every operation (µs) |
| [`E06_message_size`](E06_message_size.md) · [csv](E06_message_size.csv) | Signing and verification time against message size on the Mac (µs) |
| [`E07_rejection_timing`](E07_rejection_timing.md) · [csv](E07_rejection_timing.csv) | Time to reject an invalid input against the time to accept a valid one, Mac (E7) |
| [`E08_sizes`](E08_sizes.md) · [csv](E08_sizes.csv) | Sizes of keys, signatures, ciphertexts and shared secrets (bytes) |
| [`E09_memory`](E09_memory.md) · [csv](E09_memory.csv) | Memory per call (E9) |
| [`E10_issuer_holder`](E10_issuer_holder.md) · [csv](E10_issuer_holder.csv) | Issuer-side against holder-side ML-DSA signing: equivalence test |
| [`E11a_lifecycle_total`](E11a_lifecycle_total.md) · [csv](E11a_lifecycle_total.csv) | Credential lifecycle cost of the 16 curated suites (ms) |
| [`E11b_lifecycle_stages`](E11b_lifecycle_stages.md) · [csv](E11b_lifecycle_stages.csv) | Credential lifecycle by stage, 16 curated suites, three machines (ms) |
| [`E11c_scaling`](E11c_scaling.md) · [csv](E11c_scaling.csv) | Production suite (ML-DSA-44 + ML-KEM-768): issuance and verification against attributes and value size (µs) |
| [`E11d_all_pairs`](E11d_all_pairs.md) · [csv](E11d_all_pairs.csv) | Lifecycle cost of every signature × KEM pair on the Mac (ms), the data of figure F6 |
| [`E12_concurrency`](E12_concurrency.md) · [csv](E12_concurrency.csv) | Concurrency: throughput against the number of workers |
| [`E13a_ledger_issue_credential`](E13a_ledger_issue_credential.md) · [csv](E13a_ledger_issue_credential.csv) | Ledger bytes of one issueCredential transaction, by the issuer's signature scheme (E13) |
| [`E13b_ledger_every_transaction`](E13b_ledger_every_transaction.md) · [csv](E13b_ledger_every_transaction.csv) | Ledger bytes of every QChain transaction (E13) |
| [`E13c_ipfs_envelope`](E13c_ipfs_envelope.md) · [csv](E13c_ipfs_envelope.csv) | Size of the encrypted credential envelope on IPFS (bytes) |
| [`E13d_encoding_whatif`](E13d_encoding_whatif.md) · [csv](E13d_encoding_whatif.csv) | How much of a credential transaction is encoding: three hypothetical encodings (E13, n = 10) |
| [`E13e_holder_key_binding`](E13e_holder_key_binding.md) · [csv](E13e_holder_key_binding.csv) | Binding the holder's keys: ledger bytes by KEM (holder signature key ML-DSA-44) |
| [`E14_fabric_latency`](E14_fabric_latency.md) · [csv](E14_fabric_latency.csv) | Fabric transaction latency against payload size, Mac (ms) |
| [`Q1_run_quality`](Q1_run_quality.md) · [csv](Q1_run_quality.csv) | Run quality of every main run |
| [`Q2_sensitivity`](Q2_sensitivity.md) · [csv](Q2_sensitivity.csv) | Sensitivity analysis: headline attempts against all attempts |
