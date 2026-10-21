# QChain PQC KPI Benchmarking: Consolidated Plan

**Version 3.6, 10 October 2026.** Status: **coding plan approved (10 Oct); coding in progress.** The implementation design is in `coding-plan.md` (same folder); where the two differ, the coding plan's §13 refinements apply. Code and results live in the private repo `teamqchain/qchain-kpi` (not a branch of the public repo). Machine specifications: §6 and `memory/qchain_kpi_test_machines.md`.

Gantt task "PQC KPI Benchmarking": Maqqavi and Nihal, 27 Sep to 24 Oct. Co-authors of the resulting paper: all four team members.
**This copy lives in the harness repo as `docs/plan.md`; it is the planning snapshot and the decision log (§19). Where it disagrees with `docs/methodology.md`, `docs/data-dictionary.md` or `docs/decisions-log.md`, those describe what the code does and win.** Sections 11–13 and 15–16 were written before the private repo existed: the layout is now the one in `README.md`, results are committed from each machine's clone of `qchain-kpi` (not a `kpi-benchmarking` branch of the public repo), and the container is pinned with `--cpuset-cpus` only, with no `--cpus` quota. The literature PDFs mentioned in §17 stay outside the repo.

---

## 1. Purpose and research questions

We benchmark the NIST-approved and NIST-selected post-quantum algorithms in liboqs against KPIs that reflect QChain's **current** application-layer credential flow. The results are intended for a conference or journal paper. There are two tracks:

- **Signature track:** keygen, sign and verify.
- **KEM track:** keygen, encapsulate and decapsulate, both alone and inside QChain's KEM plus AES-GCM envelope.

| RQ | Question |
|---|---|
| RQ1 | What do the primitives cost (latency distribution, size, memory) for every final and selected NIST PQC algorithm, compared with classical baselines? |
| RQ2 | What does a full **encrypted** credential lifecycle cost (issue, retrieve, present, verify) for each algorithm suite and attribute count? |
| RQ3 | How much does the deployed integration pattern add on top of the raw C library (Go/cgo, re-initialising per call, hex key import, Go allocations)? |
| RQ4 | How sensitive are results to the hardware (virtualised x86 without AVX2, x86 with AVX2, ARM), the runtime (Docker vs native) and the build flags? |
| RQ5 | What are the storage consequences: ledger bytes per credential, credentials per Fabric block, IPFS envelope size? |
| RQ6 | Are issuer-side and holder-side ML-DSA signing statistically equivalent? |

## 2. Hard constraints

- **Independent.** A new top-level folder `kpi-benchmarking/` with its own Dockerfile, `go.mod`, tests and scripts. It imports nothing from `offchain/`, `UI_App/`, `UI_WebApp/`, `shared/` or the chaincode.
- **Never touches the live system.** No use of the VM's Fabric network, MySQL, IPFS, backend or frontends. Containers run with `--network none`. During the study the QChain services on the VM are stopped (§13.1).
- **Application layer only.** PQC inside Fabric's identity layer (Obeid's MSP track) is out of scope.
- **PQC from liboqs only.**
- **Phone and wallet timing are out of scope entirely**, not even listed as future work.
- **Nothing is assumed.** Open points are asked (§15).

## 3. The QChain flow being mirrored (verified in code, 4 Oct 2026)

| Step | Who | Operations (N = number of attributes, k = number disclosed) |
|---|---|---|
| Wallet activation (once per holder) | Holder | 1 ML-KEM-768 keygen + 1 ML-DSA-44 keygen |
| Issuance | Issuer (backend) | N+1 random 16-byte salts; N salted field hashes `SHA3-256(salt:key:value)`; 1 ML-KEM encapsulation; N+1 × (HKDF-SHA3-256 + AES-256-GCM seal + AES-GCM key wrap); canonical commitment JSON → SHA3-256 → **ML-DSA-44 sign of the 64-char hex string** |
| Retrieval | Holder | 1 ML-KEM decapsulation; N+1 × (HKDF + unwrap + AES-GCM open) |
| Presentation | Holder | canonical payload JSON → SHA3-256 → **ML-DSA-44 sign of the 64-char hex string** |
| Verification | Verifier (backend) | recompute commitment (1 SHA3) + **ML-DSA verify (issuer)** + k field-hash recomputations + payload hash (1 SHA3) + **ML-DSA verify (holder)**; reads the chain only, never IPFS |
| Expiry update | Issuer | 1 SHA3 + 1 sign |

**Byte-format details:**
- **Attributes:** string-to-string only, at most 64 fields. Reserved names are rejected, and `expiryDate` is lifted into on-chain metadata.
- **Envelope:**
  - one encapsulation per credential, to a single recipient ("holder");
  - per field, HKDF info string `qchain/trackB/v1|<credId>|<field>`;
  - a fresh AES-256-GCM data key and a random 12-byte nonce per field;
  - the data key wrapped with AES-GCM under a zero nonce, giving 76 B overhead per field;
  - a reserved `_salts` field.
- **Commitment:** `{v:2, holderID, credentialType, issuedAt, issuerOrgID, expiryDate, cid, fieldHashes}`.
- **Presentation payload:** `{credentialID, disclosedFields, salts, timestamp}`.
- **Ledger record:** stores the signature hex, the issuer public key hex (repeated in every credential), the field hashes, the CID and metadata. Hex encoding doubles every size. The chaincode performs **no** PQC crypto.

**Production build facts:**
- **Backend image:**
  - liboqs is cloned from `main` unpinned, with `OQS_DIST_BUILD=ON` and `OQS_MINIMAL_BUILD` (ML-DSA, ML-KEM, Kyber);
  - OpenSSL is present;
  - Go 1.24 on Debian bookworm;
  - liboqs-go is pinned at v0.15.0 (commit 75451133).
- **Backend code:**
  - `pqcSign` and `pqcVerify` re-initialise the signer on every call;
  - symmetric crypto uses Go's `crypto/aes` + `cipher` GCM, `x/crypto/hkdf` and `x/crypto/sha3`.

## 4. Algorithms (30)

| Track | Tier A: final standard | Tier B: NIST-selected, pre-standard | Classical baselines |
|---|---|---|---|
| Signatures (22) | ML-DSA-44/65/87 (FIPS 204); SLH-DSA {SHA2, SHAKE} × {128, 192, 256} × {s, f}, the 12 pure sets (FIPS 205) | Falcon-512, Falcon-1024, Falcon-padded-512, Falcon-padded-1024 (liboqs implements the 2021 spec, **not** FN-DSA) | ECDSA P-256, Ed25519, RSA-3072 (RSA-PSS with SHA-256) |
| KEMs (8) | ML-KEM-512/768/1024 (FIPS 203) | HQC-1/3/5 (spec 2025-08-22, reference implementation only) | DHKEM(X25519) and DHKEM(P-256), per RFC 9180 §4.1 |

**Excluded, with reasons:**
- Legacy Kyber, Dilithium and SPHINCS+: superseded by the FIPS versions.
- MAYO, CROSS, UOV, SNOVA and MQOM: not NIST-selected.
- BIKE, Classic McEliece, FrodoKEM, NTRU and sntrup: not selected.
- The 144 pre-hash SLH-DSA variants: out of scope.
- Stateful LMS/XMSS: they need state management.

Source for the standards status: the NIST CSRC page (5 Aug 2026), where only FIPS 203/204/205 are final, and the liboqs 0.16.0 documentation.

## 5. Measurement layers

| Layer | What it calls | Role |
|---|---|---|
| **L1: Go via cgo (primary)** | PQC through liboqs-go v0.16.0; classical through OpenSSL libcrypto via cgo, so both cross the same Go-to-C boundary | All headline results. Two modes: **reuse** (one signer object) and **re-init per call** (as the backend does) |
| L0: raw C | liboqs and OpenSSL called directly from a small C helper | Integration-pattern cost (L1 − L0); stack, heap and memory measurements |
| L2: Go standard library | `crypto/ecdsa`, `ed25519`, `rsa`, `ecdh` | Secondary classical reference |

## 6. Platforms, runtimes and builds

| Results folder | Machine | Known specs | Runtime |
|---|---|---|---|
| `results/linux/` | UoS VM (host-vm4), described in the paper only as a *tested environment*: a virtualised x86 server | **Guest:** Proxmox CPU type **`x86-64-v2-AES`** (Proxmox VE 8's default; reports itself as "QEMU Virtual CPU version 2.5+"), KVM, **1 socket × 2 cores = 2 vCPUs, unpinned**, 30 GiB RAM. Flags include aes and sse4.2 but **no AVX, AVX2, BMI2, PCLMULQDQ or AVX-512**; no constant TSC, no PMU; clocksource kvm-clock; no cpufreq, no RAPL. Docker 29.3.1. The Ubuntu release, kernel and free disk are captured by E0. **Host (Omnia, 7 Oct):** Supermicro X13 CloudDC 2U (X13DDW-A), Intel Xeon Silver 4510 (Emerald Rapids, 12 cores / 24 threads per socket, 2.4 GHz base, up to 4.1 GHz turbo, 30 MB L3, 150 W), DDR5-5600 ECC, NVMe PCIe 4.0, 2× NVIDIA L40S (not attached to the VM), Proxmox VE 8.x on KVM/QEMU. **The vCPUs are time-shared with other VMs**: at most 2 run at once, and they may float across the host's 24 threads. **QChain services are stopped during the study** | Docker (Linux, so effectively native); `--network none --cpuset-cpus=1 --memory=4g` (no `--cpus` quota; 10 Oct) |
| `results/mac/` | MacBook Air (M1, 2020, MacBookAir10,1), **fanless**, run on a USB fan **cooling pad** | Apple M1: 4 performance + 4 efficiency cores, 8 GB RAM; NEON, AES, SHA-2/SHA-3 and PMULL extensions; macOS 27.0.1 (26A434); Apple clang 21. Docker, Colima, Go and CMake are **not installed yet** (installed just before testing) | **Colima** with Apple's native virtualisation (`--vm-type vz`): VM with 4 CPUs and 4 GB; containers `--memory=3g --cpuset-cpus=1` (no `--cpus` quota). Fallback: Docker Desktop with the same limits. Native macOS is secondary |
| `results/windows/` | Obeid's own laptop (not a VM), accessed by Maqqavi over AnyDesk | **Checked 7–8 Oct.** HP Laptop 15-fc0xxx; **AMD Ryzen 5 7520U** (Zen 2, 4 cores / 8 threads, 2.8 GHz base, 2 MB L2, 4 MB L3, all cores the same type); **AVX and AVX2 yes, no AVX-512**; 8 GB LPDDR5-5500 (7.2 GB usable); Windows 11 Home 25H2 (build 26200); VBS and memory integrity on (recorded, not changed); Windows Defender plus McAfee. WSL 2.7.12 (kernel 6.18), Docker 29.7.2 on WSL2 with 8 CPUs and **3.45 GB of memory** (WSL default, no .wslconfig), so the **container memory cap is 3 GB here**. 272 GB free disk. Whether the laptop has a cooling fan is **unconfirmed**; thermals are logged | Docker Desktop via WSL2 (primary), containers `--memory=3g --cpuset-cpus=1` (no `--cpus` quota); native Windows optional |

**Why Docker is primary.** On a Mac or Windows machine, Docker runs a Linux VM. The host operating system is therefore irrelevant for the primary results; what differs is the hardware. Every machine runs the *same* compiler, libraries, flags and source build, so differences come from the hardware: ARM with NEON, x86 with AVX2, or virtualised x86 without AVX2. Native runs add the real-OS dimension and measure Docker's own overhead. Images are always built for the machine's own architecture (arm64 on the M1, amd64 elsewhere), never emulated.

**Mixed core types.** The M1 has performance and efficiency cores, and macOS cannot pin threads to cores. Native runs set the "user-interactive" thread-priority hint to favour performance cores. Under Docker (Colima or Docker Desktop) the core type cannot be controlled, so it is recorded and reported as a limitation. The Windows laptop's Ryzen 5 7520U has four identical cores, so this limitation does not apply to it.

| Build | liboqs 0.16.0 flags | Runs on |
|---|---|---|
| **B1: production-like (primary)** | OpenSSL on, `OQS_DIST_BUILD=ON`, all algorithms in §4 | All three |
| B2: portable | generic target, no optimised code paths | All three; separates the instruction-set effect from the machine |
| B3: native-optimised | `OQS_DIST_BUILD=OFF`, tuned for the build machine | Mac and Windows (identical to B2 on the VM, which has no AVX2) |
| B4: no OpenSSL | OpenSSL off | VM only |

## 7. Experiments

| ID | Experiment | Scope |
|---|---|---|
| E0 | Environment capture; timer resolution and overhead; correctness gates | Every run |
| E1 | Primitive latency (keygen, sign/verify, encaps/decaps) | 30 algorithms, L1 reuse |
| E2 | Re-init per call vs reuse, plus hex key-import cost | 30 algorithms |
| E3 | Integration-pattern cost: L0 vs L1 (plus L2 for classical) | 30 algorithms |
| E4 | Build sensitivity, B1–B4 | per §6 |
| E5 | Platform and runtime comparison | E1 on every machine and runtime |
| E6 | Message-size sweep: 64 B, 1 KB, 10 KB, 100 KB, 1 MB | Signatures |
| E7 | Negative paths: corrupted signature, wrong message, tampered ciphertext (timing of rejection) | All |
| E8 | Sizes, including Falcon's real signature-size distribution and QChain hex/JSON encoded sizes | All |
| E9 | Memory: stack high-water mark, C heap, Go heap per operation, peak process memory | All, every machine |
| E10 | ML-DSA equivalence test: the issuer's commitment message vs the holder's payload message. TOST equivalence test. Margin = max(2% of the median, 2× the A/A noise floor measured in the trial run); ±5% is also reported. Fixed before the main runs | ML-DSA-44/65/87 |
| E11 | Lifecycle composites: activation, issuance, retrieval, presentation, verification. N ∈ {1, 5, 10, 20, 64}; k ∈ {1, N/2, N}; per-stage breakdown | 16 curated suites, plus all 176 signature × KEM pairs at N = 10 |
| E12 | Concurrency: throughput and p99 against number of workers, plus container peak memory | Production suite + 3 others; worker range depends on each machine. On the VM: one test of 15 minutes or less using both vCPUs (treated as approved); every other VM experiment uses 1 vCPU |
| E13 | Ledger and IPFS footprint model, using Fabric batching rules and offline transaction sizing | All 22 signature schemes |
| E14 *(on the Mac, coded 22 Oct)* | Fabric latency against payload size on QChain's network rebuilt in Colima, with a synthetic chaincode and QChain-shaped transactions; sequential and burst. Never on the VM. Design below | Payload sizes from E13 (8 signature schemes + the smallest transaction) |

**The 16 curated suites:**
- **Variants (decided 10 Oct):** every SLH-DSA entry below is the **SHA2** variant; Falcon entries are the **standard (unpadded)** variant.
- **Production:** ML-DSA-44 + ML-KEM-768.
- **Category 1–2:**
  - ML-DSA-44 + ML-KEM-512
  - Falcon-512 + ML-KEM-512
  - SLH-DSA-128f + ML-KEM-512
  - SLH-DSA-128s + ML-KEM-512
  - ML-DSA-44 + HQC-1
- **Category 3:**
  - ML-DSA-65 + ML-KEM-768
  - SLH-DSA-192f + ML-KEM-768
  - ML-DSA-65 + HQC-3
- **Category 5:**
  - ML-DSA-87 + ML-KEM-1024
  - Falcon-1024 + ML-KEM-1024
  - SLH-DSA-256f + ML-KEM-1024
  - ML-DSA-87 + HQC-5
- **Classical:**
  - ECDSA + DHKEM(P-256)
  - Ed25519 + DHKEM(X25519)
  - RSA-3072 + DHKEM(P-256)

**E14 design (decided 21–22 Oct; this replaces the separate E14 plan, `docs/e14-plan.md`, which was merged here and deleted):**
- **Question.** E13 computed bytes; E14 measures how long a transaction takes on a real Fabric network and whether QChain's post-quantum payload sizes change it, and checks E13's block model (how many transactions fit in a block). Not measured: post-quantum identities (Fabric 2.5 cannot use them; E13's projection covers that) and chaincode cryptography (QChain's chaincode has none).
- **Decisions (user, 21 Oct):** on the Mac only; mirror the live network; 8 representative schemes plus the smallest transaction; sequential and burst; a synthetic JavaScript chaincode; **run quality recorded, not repeated** (approved with the plan, 22 Oct); **keep the live shared CouchDB** (22 Oct, after phase A found that both peers write every record to the same document, one of them after a revision conflict and a retry).
- **Network.** QChain's `qchain-network` (commit `71a1db6`) in Colima: Government and General organisations, one peer each, one Raft orderer, one CouchDB shared by both peers, three Fabric CAs, TLS, `mychannel`, BatchTimeout 2 s, MaxMessageCount 10, PreferredMaxBytes 512 KB, majority endorsement (both organisations); Fabric 2.5.15, Fabric CA 1.5.15, CouchDB 3.3.2 pinned. QChain's enrolment, environment, channel-profile, CLI and CA files are copied byte for byte; `scripts/e14-network.sh` uses plain `docker run` (no compose on the Mac) with the live names, environment variables and mount points (volume sub-paths); no host ports. The decoded channel configuration matched the live one.
- **Chaincode** `kpipayload` 1.0: QChain's `issueCredential` pattern (role check, holder read, existence check, one sorted-key JSON write, short result) with an N-byte hex payload; same libraries as QChain's chaincode, versions pinned.
- **Client** (`fabric/client`, Go, its own module, built and tested in Docker): QChain's backend way of submitting (fabric-gateway v1.7.0, gRPC v1.68.0, a new connection per transaction, `issuer1` through `peer0.general.uae.com`), phases timed (connect, endorse, submit, commit, total).
- **Payloads** (E13's `issueCredential` envelope, N = 5 attributes of 16 bytes; E13's predicted transactions per block in brackets): smallest transaction about 4.9 KB (10), ECDSA P-256 7,141 B (10), Falcon-512 12,805 B (10), ML-DSA-44 21,529 B (10), ML-DSA-65 27,645 B (10), ML-DSA-87 35,478 B (10), SLH-DSA-SHA2-128s 38,164 B (10), SLH-DSA-SHA2-128f 75,093 B (6), SLH-DSA-SHA2-256f 206,294 B (2). The payload length is calibrated against the committed envelope, within 0.5 %.
- **Protocol.** 9 points x 2 modes x 5 repetitions, seeded order, drift controls at the start, middle and end: sequential blocks of 100 transactions (each waits for its commit), bursts of 40 released at once; warm-ups not recorded; every transaction must commit as valid; blocks read back for transactions per block, bytes and cut reason; one payload per block read back. About 2.8 h. `scripts/e14-run.sh` runs it with run.sh's own checks and finishing steps.
- **Mac changes** (revert checklist): the pinned images, the client image, the network and its volumes and containers in Colima; nothing installed on the Mac itself; Colima stays at 4 CPUs and 4 GiB (the network uses about 0.4 GiB).
- **Risks, resolved in phase A and B:** the shared CouchDB works but conflicts on every write (kept, reported); Colima memory is ample; the peer CLI cannot pass arguments over 128 KB, so large payloads go through the client only; the smallest transaction is about 4.9 KB, not the 0.5 KB first assumed.

**Sampling:**
- **Fast operations (under 1 ms):** 1,000 warm-up runs, then 10,000 timed samples, repeated 5 times in fresh processes.
- **Slow operations:** time-boxed at 60 s per repetition, with a minimum of 30 samples.
- **Composites:** 2,000 samples × 5 repetitions.
- **Order:** algorithms run in shuffled order with a recorded seed.
- **Outliers:** never dropped.
- **Correctness:** every result is checked outside the timed window.

**Correctness gates, before any timing counts:**
- liboqs's own known-answer tests (run during the image build);
- round-trip tests for every adapter;
- RFC 9180 test vectors for the classical KEMs;
- **golden vectors** proving our QChain-format reimplementation is byte-exact. These are generated once from a local scratch copy of the backend's format functions, on the Mac only and never committed (approved).

## 8. KPIs

- **Time (both tracks):**
  - Distribution: mean, median, sd, CoV, p90/95/99/99.9, max, bootstrap 95% confidence intervals.
  - First call vs warm.
  - Re-init vs reuse.
  - Single-core throughput.
  - Multi-worker throughput and p99.
  - Optimised vs portable build speed-up.
- **Signatures:** keygen, sign, verify; invalid-signature rejection; message-size scaling; Falcon size distribution; key import; lifecycle composites; issuer/holder equivalence.
- **KEMs:** keygen, encaps, decaps; tampered-ciphertext decaps timing; envelope cost against N and value size; KEM share of envelope cost; envelope bytes against N.
- **Memory:**
  - **Stack:** high-water mark measured by stack painting in the C helper.
  - **C heap:** bytes and calls per operation, via malloc counting, cross-checked with Valgrind massif on Linux.
  - **Go heap:** bytes and allocations per operation.
  - **Process memory:** peak memory delta in a fresh process. Read with `getrusage` on Linux and macOS, and `PeakWorkingSetSize` on Windows native.
  - **Container memory:** peak under load, from cgroup v2 `memory.peak`.
- **Space:**
  - public key, secret key, signature or ciphertext, and shared-secret sizes;
  - hex/JSON encoded sizes;
  - envelope size;
  - ledger bytes per credential, credentials per block, GB per million credentials;
  - wallet key storage;
  - optionally, liboqs library size per build.
- **Dropped, and reported as limitations:**
  - **Energy:** no RAPL on the VM.
  - **Hardware cycle counts:** no PMU or stable TSC on the VM, and none accessible on the M1.

## 9. Statistics

- Medians and percentiles with bootstrap confidence intervals, resampled over the repetitions (process-level).
- Comparisons with Mann–Whitney U and the Cliff's δ effect size, with Holm correction for multiple comparisons.
- Equivalence (E10) with TOST.
- **Run-quality controls (approved 10 Oct).** Recorded before, during and after every batch on every machine: steal time (VM), non-benchmark CPU use, thermal state, and any running QChain containers (VM). A measurement block is flagged and repeated (up to 2 times; **all attempts are kept** and the flagged fraction is reported) when:
  - VM steal time exceeds **2%** of the block's CPU time;
  - other-process CPU use exceeds **5%** of total capacity;
  - any **heat-throttling** reading appears (Mac: `pmset -g therm`; Windows: the CPU performance % counter and thermal-zone temperature);
  - the block's median differs by more than **3%** from the median of all repeats (Mac and Windows; on the VM the limit stays loose, because its noise is a reported finding);
  - the canary drifts beyond its limit (calibrated in the trial run).
- **Canary (approved 10 Oct).** A tiny fixed benchmark (1–2 s; the operation is chosen in the coding plan) run at the start, middle and end of every batch on all three machines. Drift flags that stretch, so heat and neighbour effects are detectable even on the VM, where temperature cannot be read.
- **Calibration.** The numbers above are starting values. The 30-minute trial run calibrates them and measures the A/A noise floor (the same operation timed twice with identical code). All thresholds, the canary limit and the E10 margin are then fixed and written down **before** the main runs, and never adjusted afterwards.

## 10. Outputs and visualisation

**Figures** (vector PDF for the paper, PNG for reports):
- **F1:** signature latency dot plot, log scale, median with p99 whisker.
- **F2:** the same for KEMs.
- **F3:** ECDF and violin plots of distributions (ML-DSA and Falcon tails).
- **F4:** size against speed scatter with the Pareto frontier.
- **F5:** lifecycle stacked bars per suite.
- **F6:** heatmap of 22 signatures × 8 KEMs, showing slowdown relative to ECDSA + X25519.
- **F7:** envelope time and size against N.
- **F8:** integration-pattern cost (C vs Go reuse vs Go re-init).
- **F9:** platform, runtime and instruction-set comparison.
- **F10:** memory (stack, heap, process memory).
- **F11:** ledger bytes per credential, credentials per block, GB per million credentials, with the 512 KB line.
- **F12:** concurrency scaling.
- **F13:** negative-path timing.

**Tables:**
- **T1:** algorithm inventory.
- **T2:** primitive medians, p99 and confidence intervals.
- **T3:** lifecycle cost per suite.
- **T4:** memory.
- **T5:** ledger footprint.
- **T6:** equivalence result.

**Dashboard:** an interactive HTML page so all four co-authors can explore the data.

## 11. Repository layout (code and results together, on branch `kpi-benchmarking`)

```
kpi-benchmarking/
  README.md          complete standalone documentation (outline below)
  .gitattributes     LF line endings for *.sh, *.go, Dockerfile (Windows clones); *.gz binary
  .gitignore         build artefacts only; results ARE committed
  go.mod  go.sum     own module; liboqs-go v0.16.0 pinned
  cmd/       kpi-env/  kpi-prim/  kpi-flow/  kpi-ledger/
  internal/  algs/  pqc/  classical/  gostd/  qformat/  timing/  record/
  csrc/      kpi_c.c (raw-C layer, stack painting, malloc counting, process memory)
  docker/    Dockerfile (liboqs 0.16.0 pinned; build args select B1–B4)
  scripts/   run.sh (macOS/Linux)  run.ps1 (Windows)  native-mac.sh  [native-windows.ps1 optional]
  testdata/  golden/  rfc9180/
  analysis/  requirements.txt  analyze.py  figures.py  dashboard/
  docs/      plan.md  methodology.md  experiment-matrix.md  qchain-flow.md
             literature.md  references.bib  decisions-log.md
  results/
    linux/    MACHINE.md  INDEX.md  <run-id>/
    mac/      MACHINE.md  INDEX.md  <run-id>/
    windows/  MACHINE.md  INDEX.md  <run-id>/
    combined/ figures/  tables/        (generated on the Mac by analysis/)
```

**Run folders:**
- **Naming:** `<YYYYMMDD-HHMM>_<machine>_<runtime>_<build>`, for example `20261013-2200_uos-vm_docker_b1-prod`.
- **Contents:**
  - `manifest.json`: hardware, CPU flags, OS, liboqs commit and flags, compiler and Go versions, image ID, harness commit, seed, load and steal;
  - `raw/*.csv.gz`: one row per timed sample;
  - `summary/*.csv`;
  - `console.txt`;
  - `SHA256SUMS`.
- **OS detection:** the host-side launcher detects the OS and mounts `results/<os>/`, because inside Docker every machine looks like Linux.
- **Indexes:** there is one `INDEX.md` per OS folder, so the three machines never edit the same file.

**Repo pitfalls (verified):**
- The root `.gitignore` ignores `*.log`, `*.out`, `*.test` and `**/build/`, so result files never use those names.
- GitHub rejects files over 100 MB and warns at 50 MB, so raw data is gzipped and chunked under 50 MB per file. Estimated total: 100–300 MB.
- Paper PDFs are **not** committed, for copyright and size reasons. `docs/literature.md` and `references.bib` carry the content.

**README outline:**
1. Purpose and research questions
2. Relationship to QChain: what is mirrored, isolated and out of scope
3. The flow being modelled
4. Algorithms and exclusions
5. Machines, runtimes and builds
6. Experiments: what, why, how, sample sizes
7. KPIs and how each is measured
8. Statistics
9. Folder layout
10. How to reproduce on each OS
11. Results: location, formats, column dictionary, run indexes
12. Key findings (after analysis)
13. Literature and novelty
14. Limitations and threats to validity
15. Decision log
16. Authors

## 12. How results reach the repo

- **Branch:** `kpi-benchmarking`, created from `main` when coding starts. Each machine writes **only** its own `results/<os>/` folder, so pushes never conflict. Everyone runs `git pull --rebase` before `git push`.
- **Mac:** I run and commit locally; pushes happen when Maqqavi says so.
- **Windows (Obeid's laptop, operated by Maqqavi over AnyDesk):**
  1. Clone the repo and check out `kpi-benchmarking`.
  2. Run `scripts\run.ps1`, then disconnect AnyDesk for the duration of the batch.
  3. Reconnect, commit `results/windows/`.
  4. `git pull --rebase`, then `git push`, using Obeid's GitHub login already set up on the laptop.
- **VM:**
  - A separate clone, `git clone --branch kpi-benchmarking --single-branch …` into `~/kpi-work/`. The live checkout is never touched.
  - Pushes use the contributor GitHub account already configured on the VM (used from VS Code). No new token; the global git config is not modified.
  - Pushes go only to `HEAD:kpi-benchmarking`. Nothing is ever committed in the live checkout.
  - At setup, a read-only check confirms that the existing credentials work from the separate clone.
- **Merge to `main` at the end.**
  - The web-gateway build sends the whole repo as its context. Its Dockerfile copies only `UI_WebApp/`, `shared/` and `nginx.conf`, so the QPortal image is unaffected; the only cost is a larger context transfer.
  - Adding the folder to `.dockerignore` would trigger one QPortal redeploy. This is Maqqavi's decision at merge time.

## 13. Run procedures (each run window)

**Common to all machines**
- A host-side launcher (`run.sh` or `run.ps1`) starts every batch. It checks the preconditions (power, sleep blockers, background load), records the run-quality data of §9, runs the canary at the start, middle and end, and refuses to start when a precondition fails.
- **Sleep suspends every process**, so the launcher blocks sleep for the duration of the batch only, with no permanent setting changes: `caffeinate -i -s` on the Mac and `systemd-inhibit --what=sleep:idle` on the VM. The Windows laptop is already set to never sleep on AC. Staying awake adds no measurement noise by itself.
- All VM and Windows commands are relayed to Maqqavi one step at a time. Results are never assumed.

### 13.1 VM (`results/linux/`)

1. Runs may start at any time (Omnia: no restrictions, no scheduled jobs, no reboots). Only Maqqavi touches the VM during the study.
2. **QChain services are stopped for the whole study**, from right before the first VM run (after coding and the Mac smoke test): the Fabric network (3 CAs, orderer, CouchDB, 2 peers, chaincode containers), `qchain-api`, `qchain-web-gateway`, IPFS, MySQL and the GitHub runner.
   - A read-only inventory comes first; services are then stopped one step at a time.
   - **Stop, never delete**: no `down -v`, no prune, no volume removal. The team has been told.
   - `qchain-fabric.service` restarts Fabric after any VM reboot, so E0 records every running QChain container and flags the run if any is present.
   - For a demo, QChain is started temporarily and the benchmarks pause.
3. **No merges or pushes to `main`** during the study, because the self-hosted runner would deploy and corrupt the measurements. No pulls in the live checkout. Pulling and pushing `kpi-benchmarking` in `~/kpi-work` triggers nothing.
4. Before the first run: a read-only check of free disk (at least 10 GB), the VM's sleep settings and the Docker version. E0 captures the Ubuntu release and kernel.
5. Close VS Code. Start the batch in `tmux` or `nohup`, wrapped in `systemd-inhibit`, then **disconnect AnyDesk** to remove screen-capture load.
6. Container flags: `--network none --cpuset-cpus=1 --memory=4g` (no `--cpus` quota; 10 Oct), with names and tags prefixed `kpi-`. Only the concurrency test (E12, 15 minutes or less) uses both vCPUs.
7. Afterwards: verify checksums, commit `results/linux/`, push from the separate clone.

### 13.2 Mac (`results/mac/`)

1. Charged to 100% and on the charger. The launcher refuses to start on battery.
2. On the USB fan cooling pad (powered from the laptop), on a solid surface, lid open, in the same arrangement every time. The pad is stated in the methodology. `pmset -g therm` is logged, and throttling flags the block.
3. Quit Teams, Chrome, VS Code and OneDrive; check with `top -o cpu` that `fileproviderd` and Spotlight are quiet. The local MySQL may stay running while idle (if the check shows activity, stop it from System Settings → MySQL; never change its auto-start). Low Power Mode off. No macOS update during the study.
4. Colima (`vz`) with 4 CPUs and 4 GB; containers `--memory=3g --cpuset-cpus=1` (no `--cpus` quota). Maqqavi launches the batch from Terminal with a command that is provided just before the first run.

### 13.3 Obeid's laptop (`results/windows/`)

1. Maqqavi messages Obeid before every night's run. The laptop is on (only Obeid can power it on, because Wake-on-LAN is unavailable), plugged in, lid open, on a hard surface, and not used during the batch. None of Obeid's own containers may run.
2. Before the first Windows run: verify that McAfee's scheduled scans are disabled or moved to the morning (Obeid agreed to do this on 9 Oct).
3. Docker Desktop on WSL2; containers `--memory=3g --cpuset-cpus=1` (no `--cpus` quota). **Never** run `docker system prune` or remove images that are not ours (he has Fabric, Ollama, Qdrant, Postgres and agent images); remove only `kpi-*`. `run.ps1` must run under Windows PowerShell 5.1.
4. Thermals: whether the laptop has a fan is unconfirmed. The trial run logs the CPU performance % counter and the thermal-zone temperature, and Obeid is asked whether he hears a fan under load. If throttling appears: hard surface, night-time runs and conditional cool-down pauses.
5. Start the batch, then **disconnect AnyDesk** (the free licence ends sessions after about 15 minutes anyway). Reconnect afterwards, verify, commit `results/windows/` and push with Obeid's login.

## 14. Isolation hazards and mitigations

| Hazard | Mitigation |
|---|---|
| Editing `.dockerignore` triggers a QPortal redeploy | Not edited during the study; decided at merge time |
| Deploy workflows run `docker image prune -f` | It removes only dangling images; all KPI images are tagged `kpi-*` |
| Deploy workflows refuse to run if the live checkout is dirty | We never work in the live checkout; we use the separate clone in `~/kpi-work/` |
| Live Fabric network `docker_fabric_net` and ports 7050, 7051, 9051, 5984, 5001, 8080, 4001, 3000, 8090, 3306 | `--network none`, no ports opened |
| The self-hosted runner builds on the VM after a merge to `main` | The runner is stopped, and nobody merges to `main` during the study (§13.1) |
| CPU noise on the VM | The live services are stopped, so only co-located VMs remain; measured as steal time and by the canary; single-vCPU benchmarks; the two-vCPU test is separate |
| Fabric auto-start (`qchain-fabric.service`) brings QChain back after a VM reboot | E0 records and flags any running QChain container |
| Docker cleanup on Obeid's laptop would delete his own images | No prune; remove only `kpi-*` |
| Memory pressure on the 8 GB Mac distorts timings | Colima instead of Docker Desktop; Teams, Chrome and VS Code closed; containers capped at 3 GB |
| A sleeping computer silently pauses the batch | `caffeinate` on the Mac, `systemd-inhibit` on the VM, never-sleep on the Windows laptop |
| Heat throttling (fanless Mac; Windows fan unconfirmed) | Cooling pad, thermal logging, canary, shuffled order, conditional cool-down pauses |

## 15. Inputs received and what remains

**From Omnia (VM), 6–9 Oct: all received.**

| Question | Answer | Effect |
|---|---|---|
| Run hours | **Any time** ("you are not using the GPU") | No schedule restrictions |
| Scheduled backups, snapshots, updates, reboots | **None** | None to avoid |
| Host and virtualisation details | Xeon Silver 4510 host, Proxmox VE 8.x, KVM/QEMU, **CPU type `x86-64-v2-AES`**, 1 socket × 2 cores, unpinned, **shared with other VMs** | Documented in §6 and the methodology |
| Offer to change the CPU type (restart plus snapshot), or to use a separate physical machine | **Declined, final.** The VM stays unchanged: it gives a third distinct code-path level (NEON / AVX2 / portable), and a default Proxmox setting is practically relevant. A dedicated machine would mostly behave like a faster x86 laptop. She was given the full explanation | No restart, no snapshot, no fourth machine |
| Two-vCPU concurrency test (15 minutes or less) | No explicit reply; **treated as approved** by Maqqavi (the vCPUs belong to the VM and QChain is stopped) | E12 runs on the VM |
| Dashboard access over the lab network | Offered as a fallback | Read-only use only |

**Obeid's laptop: verified over AnyDesk on 7–8 Oct and ready.** Details in §6 and `memory/qchain_kpi_test_machines.md`. Remaining, none of which blocks coding:
- verify the McAfee scan schedule at the next laptop session;
- check for a fan and log thermals in the trial run;
- push access is proven at the first push (Obeid is a repository collaborator).

**From Maqqavi, at test time:** install Colima, the Docker CLI, Go, CMake and Homebrew `openssl@3` on the Mac (approved; only right before testing).

**Fixed by the 30-minute trial run:** the noise thresholds, the canary limit and the E10 margin (§9).

## 16. Schedule (Gantt ends 24 Oct; PR2 due 31 Oct)

| When | What |
|---|---|
| 4–10 Oct | Planning, literature, machine checks and replies from Omnia and Obeid. **Done** |
| 10 Oct | Decisions finalised (v3.5). Coding plan written for review |
| Next | Phase 1 code: scaffold, Dockerfile, algorithm registry, primitive runner, classical adapters, RFC vectors, canary, run-quality logging, launchers |
| Then | Install Colima, Go and CMake on the Mac; smoke-test; push `kpi-benchmarking`; pull on the VM clone and Obeid's laptop |
| Then | 30-minute trial run (calibrates the thresholds, the canary limit and the E10 margin); main runs on all three machines |
| Phase 2 code | C helper and memory measurements, lifecycle composites, build variants, concurrency, ledger model, analysis |
| Run window | The window communicated to Omnia was 8–15 Oct; she said any time, so it may extend. Maqqavi informs Omnia |
| 20–24 Oct | Analysis, figures, tables, dashboard; results-section draft; PR2 preliminary results |
| 21–22 Oct | E14 planned and coded on the Mac (moved forward from November, user 21 Oct); paper writing follows |

**Run order per machine:** Mac first (fully under our control), then the VM, then Windows. A late failure on one machine then leaves two complete result sets.

**Risk.** If time compresses, the priority order is:
1. E1, E2, E8, E11 and E13 on all machines.
2. E3, E9 and E10.
3. E4, E6, E7 and E12.

## 17. Literature and novelty

PDFs are in `~/Desktop/QChain_Project_Context/QChain_KPI_Literature/` (36 files), along with the IEEE Xplore export (87 records). ACM DL and Scopus were blocked by the university login, so an OpenAlex search covering ACM, Springer and Elsevier records replaced them (219 records screened).

| Work | What it covers | Gap that we fill |
|---|---|---|
| Arakawa 2025 (TechRxiv), the closest | 15 signature schemes in SD-JWT verifiable credentials; Rust with liboqs-rust 0.11; one AWS x86 server; 1,000 runs; median and CV | No KEM or encryption, no keygen, holder signature only *assumed* identical, no memory, tails or concurrency, no ledger, one platform, pre-standard SPHINCS+ (SHAKE variants only) |
| Ouadrassi et al. 2026 (IEEE Access) | Application-layer ML-DSA-65 alongside ECDSA on Fabric 2.5.13 (liboqs-python); signature and public key on the ledger; no significant transaction-level slowdown; **blocks 1.43× larger** | One algorithm. Its own future work lists Falcon and SLH-DSA, **off-chain storage with on-chain commitments**, and **confidentiality**, all of which QChain does |
| Schemitt et al. 2025 | 62 signature implementations, blockchain simulation | Signatures only, Python, no memory or tails |
| Commey et al. 2025 | KEMs and signatures, 3 platforms, liboqs 0.8 via Python | Old liboqs; memory only as indicators |
| PQ-FundusChain 2026 | ML-KEM + AES-GCM + ML-DSA with off-chain storage | One suite, no cross-family comparison |
| Fabrizio et al. 2025 (TMA) | Federated identity; OpenSSL speed on a server core; RSA baseline; ML-KEM and HQC | Not credential-centric; no lifecycle composites |
| Computers 15:116 (2026) | Windows 11 CPUs, signatures and KEMs, 10 repetitions, shows integration choices matter | Different algorithms (HAWK, SQIsign, SNOVA, BIKE); a Python TCP boundary |
| Applied Sciences 16:5781 (2026) | C, Go (CIRCL) and Python on a Ryzen and an ESP32; memory, determinism | Different library per language, not one library behind different bindings |
| Kotukh et al. 2026 (OCaml liboqs) | OCaml FFI cost under 10 ns per call; *asserts* about 50 ns for Go/cgo, unmeasured | → our integration-pattern measurement |
| Abbas et al. 2024 (Networking Letters) | Fabric hybrid X.509 certificates via liboqs-go (identity layer) | Identity layer; relevant to Obeid's track, not ours |
| BDEC, Li et al. (ProvSec 2024, LNCS) | Post-quantum *digital learning credentials*, the same domain as QChain | **Paywalled; abstract withheld by the publisher; not read.** Ask the authors for a copy (Zoey Ziyi Li, Hui Cui et al.) |

**Novelty claims:**
- **N1 (headline).** An *encrypted* credential-lifecycle cost model: a KEM + AES-GCM envelope, issuer and holder signatures, and selective disclosure, across security-category-matched suites including classical ones. No prior work combines confidentiality with signing in a credential-lifecycle benchmark.
- **N2.** The complete FIPS 203/204/205 set plus Falcon and HQC, on liboqs 0.16.0 (the mlkem-native and mldsa-native era), with 3 classical signature and 2 classical KEM baselines.
- **N3.** A cross-hardware comparison with an identical software stack, including a virtualised server CPU without AVX2 (Proxmox's default CPU type, a common configuration), plus a native-OS layer.
- **N4.** Full distributions, tail latency and memory (stack, heap, process), not only means.
- **N5.** The measured cost of the deployed integration pattern (re-init per call, hex import, Go allocations) against raw C.
- **N6.** A ledger and IPFS footprint model for 22 signature schemes under Fabric batching, cross-checked against Ouadrassi's measured 1.43×, then calibrated empirically (E14).
- **N7.** Issuer-vs-holder ML-DSA equivalence tested rather than assumed.
- **N8.** The cost of aligning security levels (production mixes ML-DSA category 2 with ML-KEM category 3).

**Risk:** N2 and N4 on their own would look incremental. The paper's strength rests on N1, N3 and N6.

## 18. Limitations and threats to validity

- No energy data (no RAPL on the VM).
- No hardware cycle counts (no PMU or stable TSC on the VM; none accessible on the M1).
- The VM's vCPUs are time-shared with other VMs on the host (steal time and the canary measure it); the QChain services on the VM are stopped during runs.
- Core types cannot be controlled on the M1 under Docker. The Windows laptop has four identical cores.
- The M1 MacBook Air is fanless; runs use an active cooling pad and thermal logging. Whether the Windows laptop has a fan is unconfirmed.
- The Windows laptop runs with Windows Defender, McAfee (scans disabled or moved) and virtualisation-based security enabled; the Mac runs Colima, and Docker Desktop is the fallback.
- WSL's `--cpuset-cpus` applies to virtual CPUs, not physical cores.
- Docker on the Mac and Windows means Linux in a VM; native runs cover the real OS.
- liboqs Falcon is the 2021 specification, not FN-DSA; HQC is the reference implementation only.
- Application layer only; no network effects (E14 later addresses Fabric).
- No phone.
- Production uses unpinned liboqs `main`, while the study pins the 0.16.0 release.

## 19. Decision log

| Date | Decision |
|---|---|
| 4 Oct | Both tiers A and B; all 12 SLH-DSA pure sets; pre-hash variants and LMS/XMSS excluded |
| 4 Oct | Classical baselines: ECDSA P-256, Ed25519, RSA-3072, X25519, ECDH P-256 |
| 4 Oct | Go wrapper (liboqs-go) is the primary layer; raw C only to measure overhead |
| 4 Oct | Platforms: M1, VM, Windows. Energy skipped. Build variants as in §6 |
| 4 Oct | No blockchain in the measured path; analytical ledger model; empirical Fabric calibration later |
| 4 Oct | All four team members are co-authors; venue and deadline undecided |
| 5 Oct | Results committed in the repo under `results/{mac,windows,linux}/`; comprehensive folder README |
| 5 Oct | Separate branch `kpi-benchmarking`, merged to `main` at the end |
| 5 Oct | Phone/wallet timing removed from scope entirely |
| 5 Oct | Golden vectors from a local scratch copy of the backend format functions (Mac only) approved |
| 5 Oct | Docker primary on all three machines; Mac native secondary; Windows native optional |
| 5 Oct | ~~The VM pushes results with a repo-scoped token~~ → superseded: existing VM contributor account, from the separate clone |
| 5 Oct | The VS Code and AnyDesk sessions on the VM are Maqqavi's; VS Code is closed and AnyDesk disconnected during runs |
| 5 Oct | The Windows laptop is Obeid's; Maqqavi operates it over AnyDesk (disconnected during batches); pushes use Obeid's GitHub login; run windows agreed between them |
| 5 Oct | No coding until Omnia replies and Obeid's laptop has been inspected (superseded on 8 Oct) |
| 6 Oct | Omnia: VM runs allowed at any time, no scheduled jobs. She misread our email (she thought the VM's network and programs would be shut down) and offered dashboard access; a clarification was sent. Branch confirmed: work on `kpi-benchmarking`, pushed so the other machines can pull it, merged into `main` only after testing. Laptop checks are done one step at a time; Mac tool installs happen just before testing |
| 7 Oct | Omnia gave the host details (§6) and offered dedicated vCPUs (needs a VM restart) or a separate physical machine |
| 7 Oct | **Decision: exactly 3 machines: M1 Mac (ARM), Obeid's Windows laptop (x86 with AVX2), and the UoS VM unchanged** (shared vCPUs, generic CPU model without AVX2). Omnia's dedicated machine declined. **The plan's experiments stay exactly as they are; no trimming.** The paper type (conference or journal) stays undecided |
| 7 Oct | **Framing rule:** the paper describes the VM only as one *tested environment* (a virtualised x86 server), **never** as QChain's deployment or production environment. The deployment may move in future |
| 8 Oct | **Obeid's laptop verified over AnyDesk; ready, nothing changed** (AnyDesk 9.7 unattended access limited to Maqqavi's ID, no lock on disconnect, never sleeps on AC, updates paused until 1 Nov, admin account, Git login can read the repo, WSL2 and Docker OK). Rules on his laptop: never `docker prune`; remove only `kpi-*`; none of his containers during batches; lid open, plugged in, hard surface |
| 8 Oct | Mac specs recorded (MacBook Air M1, fanless, 4P+4E, 8 GB, macOS 27.0.1). The `openssl` on PATH comes from the user's CVRP conda env, so native builds must use Homebrew `openssl@3` explicitly |
| 8 Oct | **Coding starts without waiting for Omnia's or Obeid's remaining answers**, which affect only scheduling and the paper text. Order: coding plan → code → Mac smoke test → push branch → pull on the VM clone and Obeid's laptop → tests |
| 8 Oct | Obeid agreed to disable McAfee's scheduled scans until the end of October or move them to the morning. Verify at the next laptop session. No remote power-on exists, so Obeid switches the laptop on |
| 9 Oct | **Omnia:** CPU type `x86-64-v2-AES`, 1 socket × 2 cores, unpinned. Offer to change the CPU type **declined, final** (three distinct code-path levels: NEON / AVX2 / portable). She received the full explanation, including that the host OS is irrelevant because everything runs in Docker |
| 9 Oct | **QChain stopped on the VM for the whole study**, from right before the first VM run (stop, never delete; GitHub runner stopped too). No snapshot. Team informed. Only Maqqavi touches the VM; no merges to `main` |
| 9 Oct | Two-vCPU concurrency test treated as approved by Maqqavi; Omnia is not pressed further. Obeid agreed to run nights; Maqqavi messages him before each |
| 10 Oct | **Approved:** (1) Mac runtime Colima (`vz`), VM 4 CPUs / 4 GB, containers 3 GB; fallback Docker Desktop. (2) Mac runs on the charger at 100% and the cooling pad, solid surface, lid open, stated in the paper; `pmset -g therm` logged. (3) Anti-sleep: `caffeinate -i -s` on the Mac, `systemd-inhibit` + tmux on the VM; the Windows laptop already never sleeps. (4) Run-quality starting values (§9). (5) E10 margin max(2%, 2× A/A noise floor), ±5% also reported. (6) The Mac's local MySQL may stay running while idle |
| 10 Oct | **Canary approved:** a fixed micro-benchmark at the start, middle and end of every batch on all three machines (§9). Thresholds, canary limit and margin are fixed after the 30-minute trial, before the main runs |
| 10 Oct | **Repo:** the harness and results go into a new **private repo `teamqchain/qchain-kpi`** (replaces the `kpi-benchmarking` branch of the public repo). Its clones sit next to each machine's main clone: Mac `/Applications/Programs for Fun/qchain-kpi`, VM next to `~/Desktop/QChain/QChain-PQC-blockchain`, Obeid's laptop next to his main clone. Everything may be committed (code, results, full plan). It is merged into the public repo with full history **after the first paper draft is submitted for review**; if the venue is double-blind, after acceptance or as an anonymised copy |
| 10 Oct | **Tools on the Mac installed at the start of coding:** Colima, Docker CLI, docker-buildx, `go@1.24` (= Go 1.24.13, the production version), shellcheck |
| 10 Oct | **Coding plan approved** (`KPI_Coding_Plan.md`). Notable refinements:<br>• one-vCPU pinning on all machines;<br>• canary every 10 min plus a Mac core-type check;<br>• slow E11 composites follow the slow-op rule;<br>• interleaved arms with full budgets;<br>• batching of sub-resolution operations;<br>• GC policy (off in micro-benchmarks, production-like in E11/E12);<br>• attempt policy (external flags repeat; drift-only stays in the headline; sensitivity analysis);<br>• keygen correctness on a subset;<br>• per-machine thresholds;<br>• fixed input pools (P = 256);<br>• results via a Docker named volume + `docker cp`;<br>• E11 value-size sweep;<br>• E13 per-transaction model with a required read-only real-block capture on the VM;<br>• machine order Mac → Obeid → VM |
| 10 Oct | **Container CPU limit: pin only.** Containers are pinned with `--cpuset-cpus=1` and `GOMAXPROCS=1` and have **no `--cpus` quota**. Measured in the Mac smoke run: a quota equal to the pinned CPU made Linux pause a fully busy container in 29 of 100 periods (about 1.8 ms each; 53.6 ms per 10 s). Each pause lands inside a timed sample, and the quota flag would have repeated every block. The quota-throttling check stays as a sanity check (it should always read 0); kpi-batch refuses trial and main runs if a quota is set. **Preflight** also refuses with Microsoft Edge open (in addition to Teams, Chrome, VS Code, OneDrive) |
| 10 Oct | **E11:** curated suites use SLH-DSA-SHA2 and standard Falcon. The harness contains **verbatim copies** of the needed production functions (with file:line + commit provenance; algorithm names parameterised), which replaces the "scratch copy never committed" approach. Each stage is measured **once per setting it depends on** (+ one chained validation run). Count corrected: **25 PQC + 5 classical = 30** algorithms |
