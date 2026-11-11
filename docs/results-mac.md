# Results: the Mac (`mac-m1`)

> **Where this fits.** One of three per-machine result documents ([Mac](results-mac.md) · [VM](results-vm.md) · [Windows laptop](results-windows.md)); the comparison of the three is [results-comparison.md](results-comparison.md). All numbers come from `python3 analysis/run_all.py`: tables in `../results/combined/tables/`, figures in `../results/combined/figures/`.

The Mac was the first machine of the study and the only one that ran **every** experiment, E0 to E14. Its runs were made between 10 and 22 October 2026 (run ids are in UTC). It is also the machine whose numbers were reported in progress report 2 (31 October 2026); this document gives the same numbers, plus the parts the report did not have room for.

---

## The machine

| | |
|---|---|
| Model | MacBook Air (M1, 2020), fanless, on mains power, on a cooling pad, lid open |
| Processor | Apple M1: 4 performance and 4 efficiency cores; NEON, AES, SHA-2 and **SHA-3** instructions |
| Memory | 8 GB |
| Operating system | macOS 27.0.1, not updated during the study |
| Container runtime | Colima 0.10.3 with Apple's Virtualization framework: a Linux VM with 4 CPUs and 4 GiB; Docker Engine 29 inside |
| Container | pinned to one virtual CPU (`--cpuset-cpus=1`, `GOMAXPROCS=1`), no CPU quota, 3 GB without swap, no network |
| Timer | `arch_sys_counter`, effective resolution 41 ns, `time.Now` overhead about 42 ns |
| Native runs (E5) | the same harness built with Apple clang into `/Users/Shared/qchain-kpi-native`, outside Docker |

Full description: [`results/mac/MACHINE.md`](../results/mac/MACHINE.md). Two things matter when reading the Mac's numbers:
- **Core type.** macOS cannot pin a thread to a performance core, and the virtual machine hides the core type. Every canary is therefore classed against native references measured at the start of each session (a performance core does about 2.5 M SHA3-256 hashes per second, an efficiency core about 0.58 M). A block measured next to an efficiency-core canary is flagged and repeated.
- **Quiet machine.** During every Mac run the launcher paused macOS's background analysis services, switched Wi-Fi off after the image build and kept the machine awake ([decisions log](decisions-log.md), 10 and 11 October). These steps were reverted after each run, and the machine's settings were restored for good on 23 October and its tools removed on 10 November.

---

## The runs

| Plan | Run (UTC) | Blocks | Attempts | Flagged | Extension sessions | Canary range |
|---|---|---|---|---|---|---|
| E1 primitive latency | `20261010-2057` | 450 | 498 | 9.6 % | 1 | 0.971 to 1.000 |
| E10 issuer and holder | `20261011-0402` | 15 | 15 | 0 % | 0 | 0.982 to 1.000 |
| E2 integration cost (all 30 algorithms) | `20261011-0514` | 900 | 1,002 | 10.2 % | 1 | 0.960 to 1.001 |
| E8 sizes | `20261012-0309` | 60 | 60 | not flagged | 0 | 0.982 to 1.000 |
| E11 lifecycle (all 176 pairs) | `20261012-0605` | 4,955 | 5,152 | 3.8 % | 1 | 0.981 to 1.002 |
| E13 ledger and IPFS model | `20261013-1252` | 30 | 30 | not flagged | 0 | 0.986 to 1.000 |
| E6 message size | `20261013-2220` | 1,100 | 1,197 | 8.1 % | 1 | 0.980 to 1.001 |
| E7 rejection timing | `20261014-1822` | 150 | 161 | 6.8 % | 0 | 0.977 to 1.000 |
| E9 memory | `20261015-0300` | 1,350 | 1,350 | not flagged | 0 | 0.979 to 1.000 |
| E12 concurrency (1, 2, 4 workers) | `20261015-0837` | 120 | 125 | 4.0 % | 0 | 0.981 to 1.000 |
| E3 integration layers | `20261016-1904` | 455 | 514 | 11.5 % | 2 | 0.980 to 1.001 |
| E4 build: portable (b2) | `20261018-2205` | 450 | 481 | 6.4 % | 0 | 0.981 to 1.001 |
| E4 build: native (b3) | `20261019-1901` | 450 | 465 | 3.2 % | 0 | 0.982 to 1.001 |
| E5 native macOS | `20261020-1525` | 450 | 457 | 1.5 % | 0 | 0.980 to 1.000 |
| E14 Fabric latency | `20261021-2135` | 93 | 93 | recorded, not repeated | 0 | n/a |

**Quality.** 11,507 attempts in the timed and size plans, 5.0 % flagged for an external cause: 559 for other CPU use, 22 for the canary, none for heat (the Mac reported no thermal pressure in any run) and none for steal. Every block has a headline attempt; the longest case was one E3 block that needed nine attempts over two extension sessions. The headline and all-attempts medians differ by at most 0.24 % in E1 and 1.65 % in E11 (table `T14`). One E2 attempt is recorded twice (an extension bug fixed on 12 October; kept as it is by decision, see the decisions log).

---

## 1. Primitives (E1, E8)

Median microseconds per call (table `T2`), sizes in bytes (table `T1`):

| Algorithm | Key generation | Sign / encapsulate | Verify / decapsulate | Public key | Signature / ciphertext |
|---|---|---|---|---|---|
| ML-DSA-44 | 30.2 | 60.2 | 28.4 | 1,312 | 2,420 |
| ML-DSA-65 | 53.0 | 95.5 | 47.6 | 1,952 | 3,309 |
| ML-DSA-87 | 90.9 | 139.6 | 83.5 | 2,592 | 4,627 |
| Falcon-512 | 4,802 | 145.2 | 21.7 | 897 | 647 to 663 |
| Falcon-1024 | 14,858 | 290.4 | 42.6 | 1,793 | 1,256 to 1,281 |
| SLH-DSA-SHA2-128f | 888 | 20,748 | 1,252 | 32 | 17,088 |
| SLH-DSA-SHA2-128s | 56,810 | 431,348 | 428 | 32 | 7,856 |
| SLH-DSA-SHAKE-128f | 1,120 | 26,152 | 1,558 | 32 | 17,088 |
| ML-KEM-512 / 768 / 1024 | 8.3 / 13.3 / 20.2 | 9.1 / 14.0 / 21.0 | 10.1 / 16.1 / 23.7 | 800 / 1,184 / 1,568 | 768 / 1,088 / 1,568 |
| HQC-1 / 3 / 5 | 894 / 2,345 / 5,922 | 1,791 / 4,668 / 11,813 | 2,728 / 7,045 / 17,809 | 2,241 / 4,514 / 7,237 | 4,433 / 8,978 / 14,421 |
| ECDSA P-256 | 25.9 | 19.9 | 54.5 | 65 | 69 to 72 |
| Ed25519 | 32.1 | 30.3 | 81.6 | 32 | 64 |
| RSA-3072 (PSS) | 401,964 | 1,605 | 33.0 | 422 | 384 |
| ECDH P-256 / X25519 (DHKEM) | 26.0 / 31.9 | 110.3 / 67.4 | 99.5 / 37.5 | 65 / 32 | 65 / 32 |

- **The M1 is the fastest of the three machines for every lattice scheme** (the laptop is 1.10 times slower for ML-DSA, 1.44 times for ML-KEM; table `T2b`).
- **SLH-DSA with SHAKE is 1.4 to 1.6 times faster here than on the laptop**, because the M1 has SHA-3 instructions; SLH-DSA with SHA-2 runs at the same speed on every machine.
- **A first call in a fresh process** (key generation, signing, encapsulation) costs about 0.9 ms more than a warm call, half the laptop's and the VM's one-off cost.
- **ML-DSA signing's 99th percentile is 2.8 to 3.6 times its median**, the rejection-sampling tail; everything else in the lattice families stays under 2.2 times.

---

## 2. Integration layers and the deployed pattern (E3, E2)

- **E3, the Go binding** (table `T8`, figure F8): over the 46 operations that take under a millisecond, the Go binding's call costs a median of 1.003 times the C library's own call (at most 1.024). An empty call across the Go-to-C boundary takes about 28 ns; the binding's own work is a median of 63 ns. **The binding is not where the cost is.**
- **E3, Go's own library against OpenSSL** for the classical algorithms: 0.4 to 2.4 times depending on the algorithm (Go's ECDSA key generation is about half OpenSSL's cost).
- **E2, the deployed pattern** (table `T3`; the Mac also ran SLH-DSA): production against a reused key adds 11.6 % to ML-DSA, 25.7 % to ML-KEM, 2.0 % to Falcon, 0.7 % to HQC, 0.2 % to SLH-DSA and 13.6 % to the classical schemes. For ML-DSA-44 signing the hex conversion of key and signature costs 5 µs of the 67 µs production call; ECDSA's key parsing costs 14 µs a call.

---

## 3. The lifecycle (E11): every signature with every KEM

The Mac measured all **176 pairs** of the 22 signature schemes and 8 KEMs at 10 attributes with 5 disclosed, plus the full grid of the 16 curated suites (991 cells; table `T3b`, heatmap F6).

- **The production suite (ML-DSA-44 + ML-KEM-768) costs 0.69 ms**, 0.86 times the classical ECDSA P-256 with ECDH P-256 (0.81 ms) and 0.87 times Ed25519 with X25519 (0.79 ms). Without activation (the four stages QChain repeats per credential) it is 0.61 ms, the figure in progress report 2; chained, 0.75 ms.
- **The cheapest pairs on the Mac** are ECDSA P-256 with ML-KEM-512 (0.58 ms), ECDSA P-256 with ML-KEM-768 (0.62 ms), ML-DSA-44 with ML-KEM-512 (0.66 ms) and the production suite (0.69 ms): **ML-KEM is cheaper than either classical key exchange in every suite**.
- **The signature scheme sets the cost.** Against ECDSA with X25519, ML-DSA rows cost 0.9 to 1.9 times, Falcon rows 8 to 74 times (key generation at activation), SLH-DSA rows 65 to 3,000 times, RSA 600 to 720 times; in the cheapest rows HQC multiplies the cost by 8.6 (HQC-1) to 52 (HQC-5).
- **Scaling.** Issuance grows by about 10 µs per extra attribute; 64 attributes of 4 KiB cost 5.5 times 64 attributes of 16 bytes.

---

## 4. Native macOS against the container (E5)

All 90 E1 cells were measured again natively (table `T9`). The native medians are 0.97 to 1.09 times the container's by family (ML-DSA 1.03, ML-KEM 0.99, Falcon 1.09, HQC 0.97, SLH-DSA 0.97, classical 0.98). **Docker in Colima costs nothing measurable on the Mac.** The small differences in both directions probably come from the compilers (Apple clang natively, gcc 12 in the image) and from the core the scheduler chose.

---

## 5. Build sensitivity (E4)

| Family | Portable build (b2) | Native build (b3) |
|---|---|---|
| ML-DSA | 1.38 | 1.00 |
| ML-KEM | 1.45 | 1.01 |
| Falcon | 2.20 (signing 25 to 27) | 1.00 |
| SLH-DSA | 1.00 | 1.00 |
| HQC | 1.00 (0.86 for HQC-1) | 1.00 |
| Classical (control) | 1.00 | 1.00 |

The production build already selects the NEON code at run time; a native build adds nothing, a portable build costs up to 27 times for Falcon signing (table `T7`).

---

## 6. Message size, rejection timing and memory (E6, E7, E9)

- **E6** (table `T10`): ML-DSA-44 signing takes 59.8 µs for 64 bytes, 62.5 µs for 1 KiB, 77.5 µs for 10 KiB, 228 µs for 100 KiB and 1.79 ms for 1 MiB (585 MB/s); verification reaches 604 MB/s and Ed25519 703 MB/s. QChain signs a 64-character hash, so the message size never matters for it.
- **E7** (table `T11`): rejecting a bit-flipped signature, a wrong message or a tampered ciphertext takes the same time as accepting a valid one in 51 of 52 comparisons (within 3.1 %). The exception, ECDH on P-256, refuses a malformed public point after 0.14 of the normal time, by design.
- **E9** (table `T4`): stack per signing call 49 KiB for ML-DSA-44, 111 KiB for ML-DSA-87, 42.6 KiB for Falcon-512, 4.4 KiB for SLH-DSA, about 2.5 KiB for ECDSA; HQC-1 decapsulation 64 KiB. The heap figures are the same as on the other two machines.

---

## 7. Concurrency (E12)

Production suite, calls per second (table `T12`): issuance 4,487 with one worker, 8,414 with two, 12,730 with four (2.84 times, efficiency 0.71); verification 6,427, 12,076 and 18,994 (2.96 times). The 99th percentile of issuance rises from 0.39 ms to 1.94 ms at four workers. SLH-DSA, whose calls are long, scales almost ideally (3.6 to 3.7 times at four).

---

## 8. Storage (E13) and Fabric (E14)

- **E13** is a byte-exact model of QChain's ledger and IPFS objects, calibrated on a block and IPFS store captured read-only from the live system on 13 October 2026 (`internal/ledger/vmcapture/`). One `issueCredential` with 10 attributes: 22,299 bytes with ML-DSA-44 as captured (Fabric's identities then ECDSA), 48,825 bytes projected with Fabric's identities on ML-DSA-44 (the configuration deployed on 7 November); 7,911 bytes with ECDSA; 10 transactions per block. Tables `T5`, `T5b`, figure F11; discussed in [results-comparison.md](results-comparison.md#6-rq5-storage).
- **E14** rebuilt QChain's classical Fabric network in Colima and timed 6,360 transactions shaped like post-quantum credentials (table `T13`). A transaction sent alone takes 2.05 to 2.07 s whatever its payload (1 byte to 50 KB), because the orderer cuts a lone transaction at its 2-second batch timeout; endorsement takes 13 to 23 ms. Bursts of 40 complete at a median of 164 ms (tiny payload), 195 ms (ML-DSA-44 sized) and 705 ms (SLH-DSA-256f sized). **410 of the 1,800 burst transactions were marked MVCC_READ_CONFLICT**, because both peers of that network shared one CouchDB ([methodology](methodology.md), E14). The post-quantum network deployed on 7 November uses a separate goleveldb state database per peer.

---

## Limits of the Mac's results

- **One virtual CPU on an asymmetric processor.** The scheduler, not the harness, chose the core; the canary detected efficiency-core blocks, which were repeated.
- **Fanless.** Thermal pressure was logged every 5 seconds and never rose above nominal, but the cooling pad was part of the set-up.
- **A personal laptop.** Background services were paused and apps quit for every run; the remaining activity is what the 5 % other-CPU rule measures.
- **E5 compares compilers too** (Apple clang against gcc 12), not only the container.

---

## Files

- Raw data: [`results/mac/`](../results/mac/) (15 main runs, plus smoke runs, the trial and the add-on trials; [`INDEX.md`](../results/mac/INDEX.md)).
- Tables and figures: [`results/combined/`](../results/combined/README.md).
- Earlier write-up of these numbers: progress report 2 (31 October 2026, chapters 6 and 7), in the project's report repository.
