# Results: the virtualised x86 server (`vm-x86v2`)

> **Where this fits.** One of three per-machine result documents ([Mac](results-mac.md) · [VM](results-vm.md) · [Windows laptop](results-windows.md)); the comparison of the three is [results-comparison.md](results-comparison.md). All numbers come from `python3 analysis/run_all.py`: analysis tables in `../results/combined/analysis_tables/` (T0–T14), presentation tables in `../results/combined/tables/`, figures in `../results/combined/figures/`, and this machine's own bar charts and tables in [`../results/linux/`](../results/linux/README.md).

> ⚠️ **How to describe this machine.** In the paper and in reports it is *a tested environment: a virtualised x86 server*. It is never described as QChain's deployment or production environment ([`results/linux/MACHINE.md`](../results/linux/MACHINE.md)).

The VM ran the reduced scope decided on 21 October 2026 ([methodology section 9](methodology.md#9-which-experiments-run-on-which-machine-decided-21-oct-2026)): E1, E2 without SLH-DSA, E4 without OpenSSL, E9, E10, E11 on the 16 curated suites and E12 with one and two workers. Its main runs were made between 26 and 31 October 2026; QChain's services on the same VM were stopped for that week and restarted on 1 November.

---

## The machine

| | |
|---|---|
| Virtual CPU | Proxmox type `x86-64-v2-AES` ("QEMU Virtual CPU version 2.5+"): AES and SSE4.2, but **no AVX, AVX2, BMI2, PCLMULQDQ or AVX-512**, so liboqs runs its portable C code |
| vCPUs | 2, unpinned on the host and time-shared with other virtual machines |
| Host | Intel Xeon Silver 4510 under Proxmox VE 8 (KVM/QEMU) |
| Memory | 30 GiB |
| Operating system | Ubuntu 24.04, kernel 7.0 |
| Container | pinned to vCPU 1 (`--cpuset-cpus=1`, `GOMAXPROCS=1`), no CPU quota, 4 GB without swap, no network; the host monitor runs on vCPU 0 |
| Timer | `kvm-clock`, effective resolution 54 ns, overhead 62 ns; no PMU, no frequency or temperature readings |

**What this machine adds to the study:** it is the only machine whose processor has no AVX2, so its lattice schemes run the portable code paths, and the only one with neighbours the study cannot control. Steal time is measured for every block, and CPU sharing is treated as a studied factor.

---

## The runs

| Plan | Run (UTC) | Blocks | Attempts | Flagged | Extension | Canary range |
|---|---|---|---|---|---|---|
| E1 primitive latency | `20261026-0852` | 450 | 560 | 19.6 % | none | 0.945 to 1.046 |
| E10 issuer and holder | `20261026-1602` | 15 | 15 | 0 % | none | 0.956 to 0.964 |
| E9 memory | `20261026-1608` | 1,350 | 1,350 | not flagged | none | 0.944 to 1.022 |
| E12 concurrency (1, 2 workers) | `20261026-1732` | 80 | 121 | 33.9 % | 3 blocks | 0.944 to 1.061 |
| E2 integration cost (no SLH-DSA) | `20261030-1946` | 540 | 573 | 5.8 % | none | 0.958 to 1.056 |
| E11 lifecycle (16 curated suites) | `20261031-0244` | 2,505 | 2,925 | 14.4 % | 4 blocks | 0.952 to 1.065 |
| E4 build without OpenSSL (b4) | `20261031-1140` | 450 | 453 | 0.7 % | none | 0.957 to 1.040 |

**Before the main runs:** a smoke run on 25 October with QChain still live; the trial on 26 October 02:32 UTC (17 canaries between 2.31 and 2.45 million hashes per second, steal 0); the two add-on trials at 05:51 and 05:53. Because the first canary of a batch was too random on this machine, the user chose (26 October) **a fixed canary reference, the median canary of the trial**, instead of each batch's first canary; the laptop later used the same rule. Thresholds: steal 2 %, other CPU 5 %, a loose drift limit (the VM's noise is a reported finding) and wide E10 margins (7.0, 3.3 and 21.7 % for ML-DSA-44, 65 and 87).

**Quality.** 5,997 attempts, 10.1 % flagged: 565 for the canary, 51 for other CPU and 1 for steal. Every block ended with a headline attempt; seven blocks needed an extension session. Checked on 1 November: **the canary-flagged attempts were not slower than the clean attempt of the same block** (median ratio 0.994 to 1.012), so the repeats were conservative and the headline is not biased. Day-to-day variation reached about 8 % on the classical control cells between 26 and 31 October. Headline and all-attempts medians differ by at most 3.9 % in E1, 5.5 % in E2 and 4.5 % in E11 (table `T14`), the largest of the three machines.

---

## 1. Primitives (E1)

Median microseconds per call (table `T2`; as bars: figures F1a–F1c and F2a–F2c, and the VM's own in `results/linux/figures/`):

| Algorithm | Key generation | Sign / encapsulate | Verify / decapsulate |
|---|---|---|---|
| ML-DSA-44 | 77.3 | 206.0 | 78.7 |
| ML-DSA-65 | 132.3 | 321.0 | 108.2 |
| ML-DSA-87 | 183.2 | 415.2 | 183.5 |
| Falcon-512 | 11,581 | 3,667 | 36.2 |
| Falcon-1024 | 32,969 | 7,911 | 78.3 |
| SLH-DSA-SHA2-128f | 981 | 22,651 | 1,394 |
| SLH-DSA-SHA2-128s | 61,058 | 470,114 | 471 |
| ML-KEM-512 / 768 / 1024 | 24.0 / 37.1 / 55.4 | 26.1 / 38.9 / 53.7 | 26.2 / 40.3 / 58.6 |
| HQC-1 / 3 / 5 | 1,231 / 3,659 / 8,854 | 2,479 / 7,270 / 17,703 | 3,730 / 10,942 / 26,783 |
| ECDSA P-256 | 40.8 | 30.9 | 78.4 |
| Ed25519 | 46.7 | 40.3 | 101.9 |
| RSA-3072 (PSS) | 510,364 | 2,176 | 53.3 |
| ECDH P-256 / X25519 (DHKEM) | 38.5 / 42.6 | 138.3 / 90.3 | 125.6 / 45.1 |

- **Without AVX2 the lattice schemes lose their advantage.** ML-DSA is 2.6 times slower than on the Mac and 2.4 times slower than on the laptop (family medians); ML-DSA-44 verification (78.7 µs) only ties ECDSA's (78.4 µs). ML-KEM-768 still encapsulates 2.3 times faster than X25519.
- **Falcon signing is the outlier: 3.7 ms**, 25 times the Mac and 17 times the laptop. Falcon signs with floating-point arithmetic, which the portable code does without a vector unit; the Mac's portable build (E4) shows the same factor. Falcon verification has no such penalty (36 µs, faster than on the laptop).
- **SLH-DSA with SHA-2 and HQC are almost unaffected** (1.05 to 1.12 times the laptop): their code is portable C on every machine.
- **A first call in a fresh process** costs 1.7 to 1.9 ms more than a warm call for the randomised operations.

---

## 2. The deployed pattern (E2)

Production against a reused key, median over each family (table `T3`): ML-DSA 1.059, ML-KEM 1.137, Falcon 1.005, HQC 1.003, classical 1.154. The relative overhead is the smallest of the three machines, presumably because the same hex conversion is spread over slower calls; re-initialising the library object per call costs 0 to 7 %, as on the Mac.

---

## 3. The lifecycle (E11)

Sum of the five stages at 10 attributes, 5 disclosed (table `T3b`, figures F5a and the VM's F5b), milliseconds:

| Suite | Activation | Issuance | Retrieval | Presentation | Verification | **Total** | Against ECDSA + ECDH P-256 |
|---|---|---|---|---|---|---|---|
| ML-DSA-44 + ML-KEM-768 (production) | 0.16 | 0.42 | 0.19 | 0.28 | 0.26 | **1.32** | 1.13 |
| ML-DSA-65 + ML-KEM-768 | 0.26 | 0.55 | 0.19 | 0.40 | 0.42 | **1.82** | 1.55 |
| ML-DSA-87 + ML-KEM-1024 | 0.32 | 0.66 | 0.22 | 0.52 | 0.54 | **2.27** | 1.92 |
| ECDSA P-256 + ECDH P-256 | 0.09 | 0.43 | 0.32 | 0.09 | 0.25 | **1.18** | 1.00 |
| Ed25519 + X25519 | 0.10 | 0.38 | 0.22 | 0.13 | 0.31 | **1.14** | 0.98 |
| Falcon-512 + ML-KEM-512 | 11.37 | 3.85 | 0.17 | 3.69 | 0.16 | **19.19** | 16.3 |
| SLH-DSA-SHA2-128f + ML-KEM-512 | 1.01 | 22.76 | 0.17 | 22.64 | 3.01 | **49.29** | 42.0 |

- **On this machine the post-quantum production suite costs 13 % more than the classical suite**, the reverse of the Mac (0.86) and the laptop (0.78). Without activation (the four stages per credential) it is 1.16 ms against 1.04 to 1.08 ms for the classical suites.
- **A chained run costs 13 % more** than the sum of its stages (1.47 ms for the production suite).
- **Falcon's cost moves into signing as well as activation** (3.9 ms in issuance and 3.7 ms in presentation), because signing is so slow here.

---

## 4. Concurrency (E12)

The VM has two vCPUs, so E12 ran with one and two workers (table `T12`). Production suite: issuance 1,996 calls per second with one worker and 3,000 with two (1.50 times); verification 3,717 and 6,381 (1.72 times). The fast stages scale only 1.5 to 1.7 times, most likely because two workers then use the whole VM, including the vCPU the monitor and the system run on; the heavy SLH-DSA stages scale 1.9 to 2.0 times. The 99th percentile of issuance rises from 1.14 ms to 2.70 ms.

---

## 5. Equivalence, memory and build (E10, E9, E4)

- **E10** (table `T6`, figure F15): holder against issuer -0.30 % (ML-DSA-44), -0.15 % (65), -0.57 % (87). All three are equivalent within the VM's own margins, and their 90 % intervals (all within ±1.7 %) would also pass the Mac's 2 % margin.
- **E9** (table `T4`): the heap figures equal the Mac's and the laptop's in every cell; the stack differs by at most 3 %; the resident memory is about 10 % higher than on the Mac.
- **E4, liboqs without OpenSSL** (table `T7`): only the operations that draw random numbers change (ML-KEM key generation and encapsulation 5 to 11 % faster, probably liboqs's own random-number generator against OpenSSL's), every other family stays within 1 %; since the classical controls also vary by up to 8 % between days, this is reported with caution.

---

## Limits of the VM's results

- **Shared hardware.** The host runs other virtual machines; steal time stayed near zero (one attempt flagged), but neighbours can change the frequency and the caches without showing up as steal.
- **No temperature, frequency or performance counters** inside the VM; the canary and the drift rule are the only checks of the CPU's speed.
- **Two vCPUs** limit E12 to two workers.
- **The live system was stopped** for the measurements; nothing ran beside the benchmark except the operating system and the monitor.

---

## Files

- Raw data: [`results/linux/`](../results/linux/) (7 main runs, the smoke runs, the trial and the add-on trials; [`INDEX.md`](../results/linux/INDEX.md)).
- Tables and figures: [`results/combined/`](../results/combined/README.md); this machine's own: [`results/linux/`](../results/linux/README.md).
