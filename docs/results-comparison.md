# Results: the three machines compared

Analysis of all main runs of the Mac, the university VM and the Windows laptop, written on 10 November 2026. It answers the six research questions of [the plan](plan.md#1-purpose-and-research-questions) in turn. The laptop alone is analysed in [results-windows.md](results-windows.md). Every number can be regenerated with `python3 analysis/run_all.py`; analysis tables are in `results/combined/analysis_tables/` (`T0` to `T14`), presentation tables in `results/combined/tables/` and figures in `results/combined/figures/` (`F0` to `F17`); each machine's own are in `results/<machine>/`.

| | Mac (`mac-m1`) | VM (`vm-x86v2`) | Laptop (`win-ryzen`) |
|---|---|---|---|
| Processor | Apple M1, ARM, NEON | QEMU virtual x86 CPU, **no AVX2**, 2 vCPUs | AMD Ryzen 5 7520U, x86, **AVX2**, 4 cores, 8 threads |
| Runtime | Docker in Colima (VZ) | Docker | Docker on WSL2 |
| Main runs | 14 plans, all experiments | 7 plans | 8 plans |
| Attempts (non-diagnostic) | 11,507 | 5,997 | 6,672 |
| Flagged for an external cause | 5.0 % | 10.1 % | 11.5 % |
| Blocks without a clean attempt | 0 | 0 | 17 |

## Summary

1. **The post-quantum schemes QChain uses are as cheap as the classical ones, or cheaper, on hardware with the optimised code.** ML-DSA-44 verifies faster than ECDSA P-256 on the Mac (28 against 55 µs) and on the laptop (32 against 59 µs); ML-KEM-768 encapsulates 3.8 to 4.8 times faster than X25519 on both. On the VM, which lacks AVX2, ML-DSA-44 verification is 79 µs against 78 µs for ECDSA, and signing is 2 to 7 times slower than the classical schemes on every machine.
2. **The hardware effect is large and depends on the family.** Against the Mac, the laptop is 1.1 times slower for ML-DSA and 1.4 times for ML-KEM; the VM is 2.6 and 2.8 times slower. Hash-based SLH-DSA (SHA-2) and HQC are portable C and hardly depend on the processor (VM against laptop 1.05 and 1.12). Falcon signing is 17 times slower on the VM than on the laptop, 25 times slower than on the Mac.
3. **The production build already reaches full speed; the portable build does not.** A native build changes lattice schemes by at most 2 % on every machine, while a portable build makes ML-DSA 1.4 times (Mac) and 2.35 times (laptop) slower.
4. **The whole credential lifecycle of the production suite costs 0.69 ms (Mac), 0.96 ms (laptop) and 1.32 ms (VM)**, which is 0.86, 0.78 and 1.13 times the classical suite. The post-quantum suite is cheaper than the classical one on both machines with the optimised code and 13 % dearer on the VM.
5. **The integration pattern, not the cryptography, is the avoidable cost.** The deployed code adds 6 to 13 % to ML-DSA calls and 14 to 28 % to ML-KEM calls (hex conversion), and up to 2.2 times to classical calls (key import per call). The Go binding itself adds a median of 0.3 %.
6. **Signing identity does not matter for speed:** issuer-side and holder-side ML-DSA signing are statistically equivalent on all three machines (9 of 9 tests).
7. **The storage cost is the real price.** A credential's `issueCredential` transaction is 22.3 KB with ML-DSA-44 against 7.9 KB with ECDSA, 2.8 times as much. A block holds 10 such transactions.
8. **Flagging rules hardly change any result.** Over 5,030 cells, headline and all-attempts medians differ by 0.0 % in the median and by at most 5.5 %.

## 1. Data quality across the machines

Table `T0_run_quality.csv`, figure F0. All runs follow the same quality rules (steal, other CPU, heat, canary, drift) with thresholds calibrated on each machine's own trial.

| | Mac | VM | Laptop |
|---|---|---|---|
| Share of attempts flagged, E1 | 9.6 % | 19.6 % | 20.6 % |
| Share of attempts flagged, E11 | 3.8 % | 14.4 % | 9.1 % |
| Main cause (attempts flagged) | `other` (559), `canary` (22) | `canary` (565), `other` (51) | `other` (638), `canary` (230) |
| Heat or throttling | none | not measurable | 3 attempts, E12 only |
| Plans that needed rerun sessions | 5 (E3 twice) | 2 | 5 (8 extension sessions) |
| Canary range (ratio to reference) | 0.96 to 1.00 | 0.94 to 1.06 | 0.86 to 1.01 |

The VM's spread is wider because its neighbours cannot be controlled; the VM's thresholds are loose by design and its noise is a reported finding. The Mac is the quietest machine. The laptop is a shared working computer and shows it, but its flagged share is no higher than the VM's.

**Sensitivity analysis** (table `T14`): median absolute difference between the headline and all-attempts cell medians 0.0 % on every machine and plan; 95th percentile 0.03 % to 1.4 %; largest 0.24 % (Mac, E1) to 5.5 % (VM, E2). The 17 laptop blocks without a clean attempt (16 in E12 at four workers, 1 in E2) are discussed in [results-windows.md](results-windows.md#2-data-quality). Excluding them changes no conclusion.

## 2. RQ1: what the primitives cost

Median latency in microseconds, one call (table `T2_E1_primitive_latency.csv`, figures F1 and F2, five repetitions, bootstrap intervals in the table):

| Operation | Mac | Laptop | VM |
|---|---|---|---|
| ML-DSA-44 key generation | 30.2 | 32.0 | 77.3 |
| ML-DSA-44 sign | 60.2 | 79.2 | 206.0 |
| ML-DSA-44 verify | 28.4 | 32.0 | 78.7 |
| ML-DSA-87 sign | 139.6 | 153.4 | 415.2 |
| Falcon-512 key generation | 4,802 | 5,620 | 11,581 |
| Falcon-512 sign | 145.2 | 210.3 | 3,667 |
| Falcon-512 verify | 21.7 | 41.0 | 36.2 |
| SLH-DSA-SHA2-128f sign | 20,748 | 20,817 | 22,651 |
| SLH-DSA-SHA2-128s sign | 431,348 | 429,945 | 470,114 |
| ML-KEM-768 key generation / encapsulate / decapsulate | 13.3 / 14.0 / 16.1 | 20.3 / 20.6 / 23.7 | 37.1 / 38.9 / 40.3 |
| HQC-1 encapsulate | 1,791 | 2,165 | 2,479 |
| ECDSA P-256 sign / verify | 19.9 / 54.5 | 24.4 / 58.9 | 30.9 / 78.4 |
| Ed25519 sign / verify | 30.3 / 81.6 | 38.1 / 101.4 | 40.3 / 101.9 |
| ECDH on P-256 encapsulate (DHKEM) | 110.3 | 130.7 | 138.3 |
| X25519 encapsulate (DHKEM) | 67.4 | 77.8 | 90.3 |

**Sizes** (E8, the same on every machine; bytes): ML-DSA-44 public key 1,312, secret key 2,560, signature 2,420; ML-DSA-65 1,952 / 4,032 / 3,309; ML-DSA-87 2,592 / 4,896 / 4,627; Falcon-512 897 / 1,281 / 647 to 663 (666 padded); Falcon-1024 1,793 / 2,305 / 1,256 to 1,281 (1,280 padded); SLH-DSA-SHA2-128s 32 / 64 / 7,856, 128f 17,088, 256s 29,792, 256f 49,856; ECDSA P-256 65 / 32 / 69 to 72; Ed25519 32 / 32 / 64. KEM ciphertexts: ML-KEM-512 / 768 / 1024 768 / 1,088 / 1,568; HQC-1 / 3 / 5 4,433 / 8,978 / 14,421; X25519 32. All ML-DSA and ML-KEM sizes match FIPS 204 and 203. Figure F4 plots size against speed: on the post-quantum Pareto frontier are Falcon-512 (smallest) and ML-DSA-44 (fastest) on every machine.

**Findings.**
- **The classical schemes are cheaper to sign and to generate keys; the lattice schemes are cheaper to verify and to exchange keys.** ML-DSA-44 signing costs 3.0 times ECDSA on the Mac, 3.2 times on the laptop and 6.7 times on the VM.
- **ML-DSA signing has a heavy tail on every machine** (99th percentile 2.8 to 4.1 times the median) from its rejection sampling; the other lattice operations have a 99th percentile under 2.8 times the median (figure F3).
- **SLH-DSA and HQC are the slow families** (signing 21 ms to 1.5 s; HQC-5 decapsulation 18 to 27 ms), but their speed hardly depends on the machine.
- **A fresh process pays a one-off cost on its first randomised call** (key generation, signing, encapsulation): about 0.9 ms more than the warm median on the Mac and 1.7 to 1.9 ms on the laptop and the VM. Its first verification or decapsulation costs only 0.03 to 0.14 ms more. See [results-windows.md](results-windows.md#3-primitive-latency-e1).

## 3. RQ4: sensitivity to hardware, runtime and build

**Machine against machine** (ratio of E1 medians per family, table `T2b_E1_machine_ratios.csv`, figure F9; the last column counts cells where all five repetitions of one machine are slower than all five of the other):

| Family | Laptop / Mac | VM / Mac | VM / laptop |
|---|---|---|---|
| ML-DSA | 1.10 | 2.56 | 2.42 |
| Falcon | 1.44 | 2.31 (signing 25) | 2.07 (signing 17, verification 0.88) |
| SLH-DSA | 1.22 (SHA-2: 1.01, SHAKE: 1.4 to 1.6) | 1.28 | 1.05 |
| ML-KEM | 1.44 | 2.75 | 1.83 |
| HQC | 1.34 | 1.50 | 1.12 |
| Classical | 1.22 | 1.34 | 1.08 |
| Cells where all repetitions are ordered | 83 of 90 | 89 of 90 | 83 of 90 |

- **The AVX2 effect (laptop against VM) is 2.4 times for ML-DSA, 1.8 times for ML-KEM and 17 times for Falcon signing**, which uses floating-point vector code. Falcon verification is the exception: it is slightly faster on the VM (36 against 41 µs).
- **SLH-DSA with SHAKE is 1.4 to 1.6 times faster on the Mac** than on the laptop (ARM has instructions for the Keccak permutation); SLH-DSA with SHA-2 is identical.
- **The classical algorithms vary little between machines** (family medians 1.08 to 1.34); SLH-DSA with SHA-2 varies least.

**Runtime: container against native macOS (E5, Mac, table `T9_E5_native_vs_container.csv`).** Native medians are 0.97 to 1.09 of the container's by family (ML-DSA 1.03, ML-KEM 0.99, Falcon 1.09, HQC 0.97, SLH-DSA 0.97, classical 0.98): **running in Docker on a Mac costs nothing measurable.**

**Build (E4, table `T7_E4_build_sensitivity.csv`, figure F9b).**

| Family | Mac portable | Mac native | Laptop portable | Laptop native | VM without OpenSSL |
|---|---|---|---|---|---|
| ML-DSA | 1.38 | 1.00 | 2.35 | 0.99 | 1.00 |
| ML-KEM | 1.45 | 1.01 | 1.76 | 0.98 | 0.96 |
| Falcon | 2.20 (signing up to 27) | 1.00 | 2.12 (signing up to 20) | 1.00 | 1.00 |
| SLH-DSA | 1.00 | 1.00 | 1.00 | 0.92 (SHAKE 0.73 to 0.84) | 0.99 |
| HQC | 1.00 | 1.00 | 1.00 | 0.97 | 1.01 |
| Classical (control) | 1.00 | 1.00 | 1.00 | 1.00 | 1.00 |

The production build (runtime dispatch) is within 2 % of a native build for the lattice schemes on both machines, so there is nothing to gain from compiling for the deployment machine, and everything to lose from a portable build.

## 4. RQ3: what the integration adds

**The Go binding (E3, Mac, table `T8_E3_integration_layers.csv`, figure F8).** The ratio of the Go binding's call (reused key) to the C library's own call has a median of 1.003 over the 46 operations that take under a millisecond (at most 1.024), and between 1.000 and 1.006 for every family. An empty call across the boundary costs about 28 ns; for the fast operations the boundary costs a median of 84 ns and the binding's own work a median of 63 ns, at most 500 ns, which is at the level of the timer's resolution (41 ns). The Go standard library's classical operations cost between 0.4 and 2.4 times OpenSSL's, depending on the algorithm, so the choice of classical implementation matters more than the binding.

**The deployed pattern (E2, all machines, table `T3_E2_integration_overhead.csv`).** Production against reused key, median over the cells of each family:

| Family | Mac | Laptop | VM |
|---|---|---|---|
| ML-DSA | 1.116 | 1.129 | 1.059 |
| ML-KEM | 1.257 | 1.279 | 1.137 |
| Falcon | 1.020 | 1.021 | 1.005 |
| HQC | 1.007 | 1.005 | 1.003 |
| SLH-DSA (Mac only) | 1.002 | n/a | n/a |
| Classical | 1.136 | 1.200 | 1.154 |

For the lattice schemes the extra cost is the conversion of keys, signatures and ciphertexts to and from hex strings (about 6 µs on a 79 µs ML-DSA-44 signing on the laptop). Re-initialising the library object on every call costs 0.2 to 0.55 µs, so the call pattern is not the problem. For ECDSA and X25519 the dominant extra cost is parsing the key again on every call (14 to 38 µs, up to 2.2 times for signing). The VM shows the smallest relative overhead, presumably because the same conversion work is spread over slower calls.

## 5. RQ2: the credential lifecycle

Sum of the five stages for 10 attributes of 16 bytes with 5 disclosed (E11, table `T3b_E11_lifecycle_per_suite.csv`, figures F5 and F6). The Mac measured all 176 signature and KEM pairs; the VM and the laptop the 16 curated suites.

| Suite | Mac, ms | Laptop, ms | VM, ms |
|---|---|---|---|
| ML-DSA-44 + ML-KEM-768 (production) | 0.69 | 0.96 | 1.32 |
| ML-DSA-65 + ML-KEM-768 | 0.86 | 1.17 | 1.82 |
| ML-DSA-87 + ML-KEM-1024 | 1.16 | 1.49 | 2.27 |
| ECDSA P-256 + ECDH P-256 | 0.81 | 1.23 | 1.18 |
| Ed25519 + X25519 | 0.79 | 1.16 | 1.14 |
| Falcon-512 + ML-KEM-512 | 5.53 | 6.72 | 19.19 |
| SLH-DSA-SHA2-128f + ML-KEM-512 | 45.59 | 46.09 | 49.29 |

- **The production suite is 14 % (Mac) and 22 % (laptop) cheaper than the classical suite and 13 % dearer on the VM.** The ordering flips with the processor's vector support.
- **The cost of a suite is set by its signature scheme.** In the Mac's 176-pair heatmap (F6) the KEM changes the total by at most a factor of 1.3 for the lattice and classical KEMs; HQC multiplies it by 8 to 52 (HQC-1 to HQC-5) in the cheapest rows. Against ECDSA with X25519, SLH-DSA suites cost 65 to 3,000 times as much and Falcon suites 8 to 74 times, because the holder generates a Falcon key at activation (5 to 15 ms).
- **Agreement with progress report 2.** The report quoted the Mac's production suite as 0.61 ms for the four stages without activation, 0.69 ms with activation and 0.75 ms chained, against 0.72 to 0.75 ms for the classical elliptic-curve suites (four stages). This table includes activation and gives the same figures (0.69 ms; ECDSA and Ed25519 suites 0.81 and 0.79 ms with activation).
- **A chained run is 9 % (Mac), 17 % (laptop) and 13 % (VM) dearer than the sum of its stages.**
- **Issuance and verification grow linearly with the number of attributes** (about 10 µs per attribute on the Mac, 14 to 15 µs on the others) and with value size; 64 attributes of 4 KiB cost 4.8 to 5.5 times 64 attributes of 16 bytes (figure F7).

## 6. RQ5: storage

**Ledger bytes (E13, Mac; a model calibrated on a block captured from the live network and validated against it on 60 transactions, 61 blocks, 71 IPFS objects and 238 certificates; table `T5_E13_ledger_issue_credential.csv`, figure F11).** Bytes of one `issueCredential` transaction with 10 attributes of 16 bytes, on the network as captured (Fabric's own identities ECDSA; only the credential's signature changes with the scheme):

| Issuer's signature scheme | Bytes per transaction | Transactions per block |
|---|---|---|
| ECDSA P-256 | 7,911 | 10 |
| Falcon-512 | 13,575 | 10 |
| ML-DSA-44 (production) | 22,299 | 10 |
| ML-DSA-65 | 28,415 | 10 |
| ML-DSA-87 | 36,248 | 10 |
| SLH-DSA-SHA2-128f | 75,863 | 6 |
| SLH-DSA-SHA2-256f | 207,064 | 2 |

(Progress report 2 quoted 21,529 bytes for ML-DSA-44 and about 7,100 for ECDSA: the same model at 5 attributes; the table uses 10.) At 22.3 KB a transaction, **a million credentials need about 22 GB per peer** with the production scheme (both peers store the whole ledger); a block closes at the transaction count (10) or at 512 KB, whichever is first, and ML-DSA-44 fits ten. The signature scheme sets the size: ML-DSA-44 makes a credential transaction 2.8 times the ECDSA one, SLH-DSA-256f 26 times.

**IPFS (E13, table `T5b_E13_ipfs_envelope_bytes.csv`).** The encrypted envelope for 10 attributes of 16 bytes is 5,948 bytes with ML-KEM-768 (5,308 and 6,909 for 512 and 1024), 3,838 with X25519, 3,902 with ECDH on P-256, and 12,633 / 21,723 / 32,609 with HQC-1 / 3 / 5. It grows by about 340 bytes per attribute for 16-byte values (24,092 bytes for 64 attributes) and by the values themselves: 64 attributes of 4 KiB take 546 KB.

## 7. RQ6: issuer and holder signing

TOST on the paired per-repetition log-ratio, 90 % interval inside the machine's margin (table `T6_E10_issuer_holder_equivalence.csv`):

| Machine | ML-DSA-44 | ML-DSA-65 | ML-DSA-87 |
|---|---|---|---|
| Mac | -0.15 % (margin 2.00 %) | -0.11 % (2.09 %) | +0.08 % (2.00 %) |
| VM | -0.30 % (6.99 %) | -0.15 % (3.32 %) | -0.57 % (21.67 %) |
| Laptop | -1.04 % (6.09 %) | +0.20 % (2.00 %) | +0.33 % (2.00 %) |

**All nine tests show equivalence.** The margins are the machine's own (twice the A/A noise floor, at least 2 %), so they are widest on the noisy VM (21.7 % for ML-DSA-87); the issuer and holder arms differ by under 1.1 % on every machine in every set.

## 8. Other results

- **Memory (E9, `T4`, F10):** heap figures are identical on all machines, stack differs by at most 3 %. A signing call uses 49 KiB of stack for ML-DSA-44, 111 KiB for ML-DSA-87, 44 KiB for Falcon-512, 4.4 KiB for SLH-DSA and about 3 KiB for ECDSA; HQC-1 decapsulation 64 KiB, HQC-5 about 200 KiB.
- **Concurrency (E12, `T12`, F12):** production suite issuance, calls per second at 1 / 2 / 4 workers: Mac 4,487 / 8,414 / 12,730; laptop 2,629 / 4,641 / 7,607 (9,599 at 8); VM 1,996 / 3,000 at two workers. Efficiency at four workers is 0.71 to 0.78 on the Mac and the laptop; at two workers the VM reaches 0.75 for issuance and 0.86 for verification. The 99th percentile grows 2 to 5 times between one and the highest worker count.
- **Message size (E6, Mac, `T10`, F13):** ML-DSA-44 signing takes 59.8 µs for 64 bytes, 62.5 µs for 1 KiB, 228 µs for 100 KiB and 1.79 ms for 1 MiB (585 MB/s); verification 604 MB/s at 1 MiB; Ed25519 703 MB/s. QChain signs a 64-character hex hash, so message size does not matter for it.
- **Rejection timing (E7, Mac, `T11`, F13):** rejecting a bit-flipped signature, a wrong message or a tampered ciphertext takes as long as accepting a valid one in 51 of 52 comparisons (within 3.1 %): signature schemes verify in constant time with respect to the failure, and ML-KEM and HQC decapsulation reject implicitly. The exception is ECDH on P-256, which refuses a malformed point after 0.14 of the normal time (the check is on a public value and is by design).
- **Fabric (E14, Mac; QChain's Fabric network rebuilt in Docker with ECDSA identities and one CouchDB shared by both peers, payloads sized like post-quantum credentials; `T13`):** a transaction sent alone takes 2.05 to 2.07 s whatever its size (a 4 KB payload with ML-DSA-44, 50 KB with SLH-DSA-256f), because the orderer cuts a lone transaction only at its 2-second batch timeout; endorsement takes 13 to 23 ms. Bursts of 40 transactions released at once complete at a median of 164 ms (tiny payload), 195 ms (ML-DSA-44 sized) and 705 ms (SLH-DSA-256f sized); 410 of the 1,800 burst transactions were marked MVCC_READ_CONFLICT because both peers of that network shared one CouchDB. The main run has 6,360 transactions. **Fabric's own latency is two to three orders of magnitude above the cryptography's.**

## 9. Limits and threats to validity

- **Three machines, one of each kind.** No conclusion about "x86" or "ARM" in general; the VM is a virtualised x86 server whose virtual CPU has no AVX2, and the laptop is a 15 W part.
- **Pinning is to a virtual CPU on two machines.** The VM's and the laptop's containers are pinned to one vCPU, and the Mac's to one of its virtual CPUs; the host scheduler chooses the physical thread, and the Mac's performance and efficiency cores cannot be told apart from inside the container (the canary classes them).
- **Noise levels differ.** The Mac is quietest; the VM has neighbours; the laptop is a shared working computer. Quality rules and thresholds are per machine, so each machine's flagged share is not comparable one to one.
- **Seventeen laptop blocks without a clean attempt**; the sensitivity analysis shows the effect is small.
- **Experiments run on one machine only** (E3, E5, E6, E7, E8, E13, E14 on the Mac) are by design: their question is a ratio or a size that does not depend on the processor ([methodology section 9](methodology.md)). E5 says native macOS equals the container on the Mac, not on other machines.
- **E13 is a model**, calibrated on one captured block and validated against 60 transactions of it.
- **E14 measured one Fabric configuration** (ECDSA identities, one CouchDB shared by both peers, one organisation pair); other configurations would give other latencies.
- **Frequency scaling and thermal state** are recorded but not controlled on the Mac and the laptop; the canary, the monitor and the drift rule detect them.
- **Multiple comparisons.** With five repetitions per machine the exact Mann-Whitney test cannot give a two-sided p-value below 0.008, so a Holm correction over dozens of cells has no power. We report ratios with bootstrap intervals over the repetitions and Cliff's delta (`T2b`): in 83 to 89 of 90 cells every repetition of one machine is slower than every repetition of the other (delta of exactly 1).

## 10. Answers to the research questions

| RQ | Answer |
|---|---|
| RQ1 primitive cost | ML-DSA-44: sign 60 to 206 µs, verify 28 to 79 µs, keys 1.3 and 2.6 KB, signature 2.4 KB, depending on the machine; ML-KEM-768 encapsulation 14 to 39 µs, ciphertext 1,088 bytes. Lattice schemes verify and exchange keys faster than the classical ones on AVX2 or ARM hardware and sign 2 to 7 times slower. Falcon has the smallest signature but the slowest key generation (5 to 12 ms); SLH-DSA signing and HQC are roughly 200 to 25,000 times slower than their lattice counterparts. |
| RQ2 lifecycle | The production suite's full lifecycle costs 0.69 to 1.32 ms, between 0.78 and 1.13 times the classical suite; cost is set by the signature scheme and grows linearly with attributes. |
| RQ3 integration | The Go binding adds a median of 0.3 %; the deployed hex-string pattern adds 6 to 13 % to ML-DSA and 14 to 28 % to ML-KEM; classical calls pay up to 2.2 times for key import. |
| RQ4 hardware | The optimised x86 (AVX2) and ARM code is 1.8 to 2.8 times faster than the VM's code without AVX2 for ML-DSA and ML-KEM and 17 to 25 times for Falcon signing; SLH-DSA (SHA-2) and HQC do not depend on the processor. Docker costs nothing on a Mac; a native build gains nothing; a portable build costs 1.4 to 2.35 times. |
| RQ5 storage | An `issueCredential` transaction is 22.3 KB with ML-DSA-44 (7.9 KB with ECDSA), 10 per block, about 22 GB per million credentials per peer; the IPFS envelope is 5.9 KB for 10 attributes with ML-KEM-768. |
| RQ6 equivalence | Issuer-side and holder-side ML-DSA signing are equivalent in all nine tests. |

## Open items for the final report

- **The two reruns' conclusions** are final: no limit changes, no further runs; the 17 laptop blocks stay as described.
- **Figures and tables for the paper** are in `results/combined/` (26 figures in PDF and PNG, presentation tables for every experiment; [index](../results/combined/README.md)).
