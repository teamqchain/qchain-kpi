# Results: the Windows laptop (`win-ryzen`)

Complete analysis of the laptop's main runs, written on 10 November 2026 from commit `ba0990e` of the results (7 and 8 November, with two rerun rounds on 10 November). Every number below can be regenerated with `python3 analysis/run_all.py`; the analysis tables are in `results/combined/analysis_tables/`, the presentation tables in `results/combined/tables/`, the figures in `results/combined/figures/`, and the laptop's own bar charts and tables in [`results/windows/`](../results/windows/README.md). The comparison with the Mac and the VM is in [results-comparison.md](results-comparison.md).

**The machine:** an HP laptop with an AMD Ryzen 5 7520U (4 cores, 8 threads, 15 W, AVX2), 8 GB of memory, Windows 11, measured inside Docker Desktop on WSL2. The container is pinned to one WSL2 virtual CPU, with no CPU quota, no swap and no network ([machine description](../results/windows/MACHINE.md)). The scope follows [methodology section 9](methodology.md): E1, E2 (without SLH-DSA), E4 (portable and native builds), E9, E10, E11 (the 16 curated suites) and E12 (1, 2, 4 and 8 workers).

## Summary

1. **ML-DSA-44 on this laptop signs in 79 µs and verifies in 32 µs.** Verification is faster than ECDSA P-256 (59 µs) and Ed25519 (101 µs); signing is 3.2 times slower than ECDSA. ML-KEM-768 encapsulates in 21 µs, against 78 µs for X25519 and 131 µs for ECDH on P-256.
2. **The slow cases are Falcon key generation (5.6 ms for Falcon-512), SLH-DSA signing (21 ms to 1.5 s) and HQC (2 to 24 ms).** Falcon signing is fast on this machine (210 µs) because it has AVX2.
3. **The whole credential lifecycle with the production suite (ML-DSA-44 and ML-KEM-768) costs 0.96 ms, 22 % less than the classical suite (ECDSA P-256 and ECDH on P-256, 1.23 ms).** The classical suite is slower mainly because the deployed pattern imports its key again on every call (24 µs per call).
4. **Equivalence of issuer-side and holder-side signing (E10) holds for all three ML-DSA parameter sets**, with differences between -1.0 % and +0.3 % against margins of 2 % to 6.1 %.
5. **The deployed integration pattern adds 13 % to ML-DSA calls and 28 % to ML-KEM calls**, almost all of it hex conversion of keys, signatures and ciphertexts. It adds 2 % to Falcon and 0.5 % to HQC.
6. **Throughput scales to 2.9 times at four workers and 3.7 times at eight** (production suite issuance); the eight-worker point adds only 26 % over four because the machine has four physical cores.
7. **Data quality is good except for one corner.** 17 of the 5,920 blocks have no clean attempt: 16 E12 blocks at four workers and one long E2 block. They are reported through the pre-declared sensitivity analysis (section 2), which shows that flagged attempts change a typical result by 0.0 % and at most 2.3 %.

## 1. What was run

| Plan | Build | Blocks | Attempts | Flagged | Rerun sessions | Blocks without a clean attempt |
|---|---|---|---|---|---|---|
| E1 primitive latency | production (b1) | 450 | 567 | 20.6 % | 2 | 0 |
| E10 issuer and holder | b1 | 15 | 16 | 6.2 % | 0 | 0 |
| E9 memory | b1 | 1,350 | 1,350 | 0.0 % | 0 | 0 |
| E12 concurrency | b1 | 160 | 331 | 56.5 % | 2 | 16 |
| E2 integration cost (no SLH-DSA) | b1 | 540 | 631 | 14.6 % | 2 | 1 |
| E11 lifecycle (16 curated suites) | b1 | 2,505 | 2,756 | 9.1 % | 1 | 0 |
| E4 build sensitivity | portable (b2) | 450 | 531 | 15.3 % | 1 | 0 |
| E4 build sensitivity | native (b3) | 450 | 490 | 8.2 % | 0 | 0 |

The main series ran from Saturday 7 November 11:15 to Monday 9 November 05:44 (42.5 hours) under the thresholds calibrated from the laptop's own trial and add-on trials. Two rerun rounds (10 November, 01:16 to 03:23 and 06:39 to 07:50) gave 32 blocks without a clean attempt up to three more attempts each, then 18 blocks up to three more. Table `T0_run_quality.csv` lists every run. Every run folder verifies against its checksums and was produced by harness commit `7999c37`, with the thresholds committed in `8fa5158` and `7999c37`.

## 2. Data quality

**Totals.** 6,672 attempts, of which 11.5 % were flagged for an external cause (5.0 % on the Mac, 10.1 % on the VM). Almost all flags are `other`: background CPU use above the 5 % limit of the laptop's thresholds. There was no steal time, no CPU quota throttling and no battery use. The laptop's own temperature stayed between 36 and 85 °C in every run except E12 (below).

**Flagged attempts are mostly short blocks.** All the flagged probe blocks of the two add-on trials lasted under a second, where a moment of other work is a large share of the block. In the main runs the share of flagged attempts rises with block length: 3 to 6 % for blocks under a minute, 13 to 40 % for blocks of one to five minutes, and 55 % for the E2 blocks that last over five minutes (HQC-1 encapsulation).

**Canary.** The reference rate was fixed from the trial's median (2,275,149 hashes per second, limit 3 %). 8 of the 275 canary readings of the main runs are more than 3 % below it (the lowest are 0.858 and 0.866 of the reference; none is above it). The quality rules flag the blocks around such a reading and repeat them; the cause of an individual dip is not always known, and one coincides with a remote-desktop session. The effect on results is covered by the sensitivity analysis.

**E12 at four workers.** 153 of 177 attempts at four workers were flagged (86 %), against 4 of 44 at one worker, 11 of 51 at two and 19 of 59 at eight. The 144 attempts of the 16 blocks without a clean attempt have a median combined other-CPU reading of 6.4 % against the 5 % limit, and 100 of them are under 7 %; the clean attempts at four workers already sit at about 4.5 %. None carries a heat or throttling flag. The cause is therefore the load itself, not an outside program: four busy workers on four of the virtual machine's eight threads put the measure close to the limit. This is a hypothesis about the accounting, not a proven cause. With the thresholds frozen, repeating the blocks could not remove it.

**Thermal.** Only E12 reached the machine's limits: 99 °C and a lowest performance limit of 88 % in 3 of 1,706 monitor rows, and 3 attempts flagged `thermal` (two at two workers, one at eight). The other runs stayed at or below 85 °C with the performance limit at 100 %.

**Blocks without a clean attempt (17).** The strict rule leaves these without a headline result, and they stay that way by decision of 10 November (no change of limits, no further reruns):

| Block group | Blocks | Attempts each | Where the numbers come from |
|---|---|---|---|
| E12, 4 workers, SLH-DSA-128f + ML-KEM-512, issuance and verification, repetitions 1 to 5 | 10 | 9 | sensitivity analysis |
| E12, 4 workers, Falcon-512 + ML-KEM-512, issuance, repetitions 1 to 5 | 5 | 9 | sensitivity analysis |
| E12, 4 workers, ECDSA P-256 + ECDH P-256, issuance, repetition 2 | 1 | 9 | sensitivity analysis |
| E2, HQC-1 encapsulation, repetition 4 | 1 | 9 | the other four repetitions (headline) |

**Sensitivity analysis (pre-declared, table `T14_sensitivity_headline_vs_all_attempts.csv`).** For every cell of E1, E2, E3, E6 and E11 on the machines that ran them (5,030 cells), we compared the median over the headline attempts with the median over all attempts. The median difference is 0.0 %, the 95th percentile at most 1.4 % and the largest difference 5.5 % (VM, E2). On the laptop the largest difference is 2.3 % (E2) and for E1 0.34 %. For the E12 cells at four workers, the all-attempts throughput and the throughput over attempts under 8 % other CPU agree with each other (for example 142.1 and 142.2 calls per second for SLH-DSA issuance, 1,068.9 and 1,067.4 for verification), and with the clean cells at one, two and eight workers the scaling curve is smooth. The paper reports these cells as sensitivity results and states that the machine is at its noise limit at four workers.

## 3. Primitive latency (E1)

Median latency of one call, microseconds (table `T2_E1_primitive_latency.csv`; as bars: figures F1a–F1c, F2a–F2c and the laptop's own in `results/windows/figures/`; 5 repetitions each; the 95 % bootstrap intervals over the repetitions are in the table and are under 4 % of the median in 87 of the 90 cells; the exceptions are Falcon-1024 verification, where one repetition ran 1.5 times slower, RSA key generation, whose time is random, and X25519 encapsulation at 4.6 %). Figure F1 and F2 show all 30 algorithms.

| Algorithm | Key generation | Sign or encapsulate | Verify or decapsulate |
|---|---|---|---|
| ML-DSA-44 | 32.0 | 79.2 | 32.0 |
| ML-DSA-65 | 55.0 | 125.1 | 53.1 |
| ML-DSA-87 | 85.2 | 153.4 | 81.3 |
| Falcon-512 (padded: same within 1 %) | 5,620 | 210.3 | 41.0 |
| Falcon-1024 | 15,838 | 416.9 | 81.1 |
| SLH-DSA-SHA2-128f | 890 | 20,817 | 1,264 |
| SLH-DSA-SHA2-128s | 55,977 | 429,945 | 438 |
| SLH-DSA-SHA2-256f | 3,462 | 71,459 | 1,910 |
| ML-KEM-512 / 768 / 1024 | 11.9 / 20.3 / 28.5 | 12.8 / 20.6 / 29.4 | 14.5 / 23.7 / 34.0 |
| HQC-1 / 3 / 5 | 1,078 / 3,275 / 7,963 | 2,165 / 6,512 / 15,897 | 3,301 / 9,799 / 23,902 |
| ECDSA P-256 | 38.4 | 24.4 | 58.9 |
| Ed25519 | 41.7 | 38.1 | 101.4 |
| RSA-3072 (PSS) | 523,698 | 1,527 | 36.9 |
| ECDH P-256 / X25519 | 37.5 / 39.9 | 130.7 / 77.8 | 116.7 / 39.2 |

**What stands out.**
- **Lattice schemes beat classical ones on verification and key exchange.** ML-DSA-44 verifies 1.8 times faster than ECDSA and 3.2 times faster than Ed25519. ML-KEM-768 encapsulates 3.8 times faster than X25519. The price is on the signing side: ML-DSA signs 2 to 3 times slower than the classical schemes.
- **ML-DSA signing has a heavy tail.** The 99th percentile is 3.2 to 3.9 times the median and the coefficient of variation is 0.5 to 0.6, because the algorithm repeats its attempt until a signature passes the rejection test. Verification and key generation are stable (99th percentile 1.7 to 1.9 times the median).
- **A fresh process pays a one-off cost of about 1.7 to 1.8 ms on its first randomised call.** The first call of key generation, signing and encapsulation takes a median of 1.7 to 1.8 ms more than the warm median for the post-quantum schemes (ML-DSA-44 signing 1.94 ms in total, ML-KEM-512 key generation 1.76 ms), and 0.7 to 0.9 ms for the classical ones. The first call of verification and decapsulation costs only 0.06 to 0.15 ms more. The randomised operations probably include the start-up of the random-number generator; that is an inference, not a measurement. A design that starts a new process per request pays this every time.
- **SLH-DSA is slow for signing and key generation.** Verification takes 0.4 to 3.5 ms. The "f" variants sign in 21 to 125 ms and generate keys in 0.9 to 6 ms; the "s" variants sign in 0.43 to 1.5 s and generate keys in 55 to 166 ms.
- **Falcon key generation is the single slowest lattice operation** (5.6 ms for Falcon-512), 175 times the cost of ML-DSA-44 key generation. Padded Falcon costs the same as standard Falcon.

## 4. Issuer and holder signing (E10)

| ML-DSA set | Issuer | Holder | Holder against issuer | 90 % interval | Margin | Equivalent |
|---|---|---|---|---|---|---|
| 44 | 79.5 µs | 78.2 µs | -1.04 % | -2.90 % to +0.86 % | 6.09 % | yes |
| 65 | 126.2 µs | 126.6 µs | +0.20 % | -0.25 % to +0.65 % | 2.00 % | yes |
| 87 | 154.0 µs | 153.9 µs | +0.33 % | -0.05 % to +0.71 % | 2.00 % | yes |

The margins come from the laptop's trial (twice the upper bound of the A/A floor, at least 2 %); ML-DSA-44 has a wider margin because its A/A floor was 3.0 %. The A/A comparison (issuer against issuer again) gives -0.29 % to +0.16 %, so the instrument's own noise is smaller than every margin except the one for ML-DSA-44. Table `T6_E10_issuer_holder_equivalence.csv`, figure F15.

## 5. What the deployed integration adds (E2)

Production pattern against a reused key, median over the cells of each family (table `T3_E2_integration_overhead.csv`):

| Family | Production / reused | Range over cells |
|---|---|---|
| ML-DSA | 1.129 | 1.102 to 1.211 |
| ML-KEM | 1.279 | 1.198 to 1.404 |
| Falcon | 1.021 | 1.000 to 1.114 |
| HQC | 1.005 | 1.002 to 1.012 |
| Classical (ECDSA, Ed25519, RSA, DHKEM) | 1.200 | 0.991 to 4.867 |

- **For lattice schemes the extra cost is hex conversion.** ML-DSA-44 signing needs 6.2 µs of conversion on a 79 µs call (key decoded from hex, signature encoded to hex). Re-initialising the library object on every call costs only 0.35 µs, so the re-initialisation that the deployed code does is not the problem for the post-quantum schemes.
- **For the classical schemes the extra cost is key import.** ECDSA signing takes 24.6 µs with a held key and 53.8 µs when the key is parsed again on every call (2.2 times); verification 1.5 times. Ed25519 and X25519 behave similarly. This is why the post-quantum production suite is cheaper than the classical one in the lifecycle (next section).

## 6. The credential lifecycle (E11)

Sum of the five stages for 10 attributes of 16 bytes with 5 disclosed (table `T3b_E11_lifecycle_per_suite.csv`, figure F5). Milliseconds, with the ratio to the classical suite ECDSA P-256 and ECDH on P-256:

| Suite | Activation | Issuance | Retrieval | Presentation | Verification | **Total** | Ratio |
|---|---|---|---|---|---|---|---|
| ML-DSA-44 + ML-KEM-768 (production) | 0.10 | 0.32 | 0.19 | 0.15 | 0.19 | **0.96** | 0.78 |
| ML-DSA-65 + ML-KEM-768 | 0.14 | 0.38 | 0.19 | 0.21 | 0.25 | **1.17** | 0.96 |
| ML-DSA-87 + ML-KEM-1024 | 0.21 | 0.42 | 0.21 | 0.26 | 0.38 | **1.49** | 1.21 |
| ECDSA P-256 + ECDH P-256 | 0.09 | 0.47 | 0.33 | 0.09 | 0.25 | **1.23** | 1.00 |
| Ed25519 + X25519 | 0.09 | 0.40 | 0.23 | 0.12 | 0.32 | **1.16** | 0.94 |
| Falcon-512 + ML-KEM-512 | 5.70 | 0.42 | 0.17 | 0.25 | 0.18 | **6.72** | 5.47 |
| SLH-DSA-SHA2-128f + ML-KEM-512 | 0.92 | 21.15 | 0.17 | 21.09 | 2.81 | **46.09** | 37.5 |

- **A full chained run costs 17 % more than the sum of its stages** (1.12 ms against 0.96 ms for the production suite), because the stages share caches and garbage collection in a chain.
- **The production suite scales linearly with attributes:** each extra attribute costs about 14.6 µs in issuance (16-byte values); 64 attributes of 4 KiB take 4.8 times as long as 64 attributes of 16 bytes (figure F7).
- **Activation is 85 % of the Falcon-512 lifecycle and 99 % of the RSA-3072 lifecycle**, because the holder generates a key pair there. SLH-DSA is dominated by signing, in issuance and in presentation.

## 7. Concurrency (E12)

Production suite, 10 attributes (table `T12_E12_concurrency.csv`, figure F12):

| Workers | Issuance, calls per second | Speed-up | Verification, calls per second | Speed-up | 99th percentile, issuance |
|---|---|---|---|---|---|
| 1 | 2,629 | 1.00 | 4,986 | 1.00 | 0.93 ms |
| 2 | 4,641 | 1.76 | 8,835 | 1.77 | 1.24 ms |
| 4 | 7,607 | 2.89 | 15,553 | 3.12 | 1.53 ms |
| 8 | 9,599 | 3.65 | 17,587 | 3.53 | 4.29 ms |

Scaling is close to ideal at two workers (efficiency 0.88), good at four (0.72 to 0.78) and flat beyond the four physical cores (the eighth to the fourth worker adds 26 % for issuance and 13 % for verification) while the 99th percentile latency rises 2.8 times. SLH-DSA, whose calls are long, scales better at eight workers (4.1 times) but with a 99th percentile of 61 ms. The four-worker cells of SLH-DSA, Falcon-512 and ECDSA issuance come from the sensitivity analysis (section 2).

## 8. Memory (E9)

Memory is a property of the algorithm. The Go heap and C heap figures are identical on all three machines and the stack differs by at most 3 % (table `T4_E9_memory.csv`, figure F10). On the laptop one call uses this much stack: ML-DSA-44 signing 49.3 KiB, ML-KEM-768 decapsulation 18.0 KiB, Falcon-512 signing 44.0 KiB, HQC-1 decapsulation 64.3 KiB, SLH-DSA-128f signing 4.4 KiB, ECDSA signing 3.1 KiB. The Go side allocates 15.3 KB per ML-DSA-44 signing and 100.6 KB per SLH-DSA-128f signing in the production route.

## 9. Build sensitivity (E4)

Variant against the production build, median over the cells of the family (table `T7_E4_build_sensitivity.csv`, figure F9b):

| Family | Portable build (b2) | Native build (b3) |
|---|---|---|
| ML-DSA | 2.35 | 0.99 |
| ML-KEM | 1.76 | 0.98 |
| Falcon | 2.12 (signing 18 to 20 times) | 1.00 |
| SLH-DSA | 1.00 | 0.92 (SHAKE variants 0.73 to 0.84, SHA-2 variants 1.01 to 1.06) |
| HQC | 1.00 | 0.97 |
| classical (control) | 1.00 | 1.00 |

- **The production build already reaches full speed.** It selects the AVX2 code at run time, so compiling for the native processor changes lattice schemes by at most 2 %.
- **Without the optimised code ML-DSA is 2.35 times slower and Falcon signing 18 to 20 times slower.** The classical controls do not move, so the effect is the library's, not the measurement's.
- **SLH-DSA with SHAKE gains 16 to 27 % from a native build** (probably because the compiler uses processor-specific instructions for the Keccak permutation; not verified), while the SHA-2 variants are 1 to 6 % slower.

## 10. Limits of the laptop's results

- **One virtual CPU, not one core.** The container is pinned to one WSL2 virtual CPU; Windows decides which physical thread runs it, and sibling threads share a core. The frequency can change within a run (the "proc_perf" reading in the monitor was 107 to 149 % of the base clock in the trial and add-on runs). This adds spread, not bias, and the canary and the drift rule are designed to expose it.
- **A shared machine.** The laptop is Obeid's working computer: Windows services, Defender and his applications run in the background. We quit what we could and the monitor measured the rest, but this is a noisier host than the Mac or the VM, and it shows in the flagged share.
- **Four-worker E12 cells** are outside the strict rule (section 2).
- **Experiments not run here, by design:** E3, E5, E6, E7, E8, E13 and E14 (see [methodology section 9](methodology.md)); their results are Mac-only and are in the comparison document.
- **Windows inside a virtual machine:** the container runs Linux inside WSL2. A native Windows build of the harness was not part of the scope.

## Files

- Tables: `results/combined/analysis_tables/` (`T0`, `T2` to `T14`), `results/combined/tables/` (presentation tables) and the laptop's own in `results/windows/tables/`.
- Figures: `results/combined/figures/` (`F0` to `F17`, PDF and PNG) and the laptop's own bar charts in `results/windows/figures/`.
- Code: `analysis/` (`run_all.py` regenerates everything; see [analysis/README.md](../analysis/README.md)).
- Raw data: `results/windows/` (8 main runs, 7 smoke runs, the trial and the two add-on trials).
