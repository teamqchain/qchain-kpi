# KPI Benchmarking Harness — Coding Plan (v2, after independent review)

**For:** Maqqavi. **Status:** all library facts verified from primary sources (10 Oct). An independent review found 40 issues, all checked and folded in below. All open decisions were answered by you on 10 Oct:
- repo history: §1c;
- curated-suite variants: **SLH-DSA-SHA2 + standard (unpadded) Falcon**;
- production code: **copied verbatim**;
- E11 grid: **each stage once per setting it depends on**.

## Context
Plan v3.6 (`plan.md` in this folder) fixes *what* to measure: 30 algorithms (25 PQC + 5 classical), experiments E0–E14, 3 machines, quality controls and canary. This plan fixes *how the code is built*: an independent harness in a **private repo `teamqchain/qchain-kpi`** (merged into the public repo as `kpi-benchmarking/` after publication). It measures the 30 algorithms the way QChain's backend and wallet use them, records every sample, flags or repeats noisy measurements, and produces data the paper can be written from. Nothing in the main repo is imported or modified.

## 1. Facts verified in the repo (what we mirror)
| Production fact | Source |
|---|---|
| Builder `golang:1.24-bookworm` (= Go 1.24.13), runtime `debian:bookworm-slim` + `libssl3`, `ENV TZ=Asia/Dubai`, `import _ "time/tzdata"`; liboqs from unpinned `main`, `-DBUILD_SHARED_LIBS=ON -DOQS_BUILD_ONLY_LIB=ON -DOQS_DIST_BUILD=ON -DOQS_MINIMAL_BUILD=…`; pkg-config shim `liboqs-go.pc` | `offchain/Dockerfile`, `config.go:22` |
| liboqs-go `v0.0.0-20260310140033-75451133b94a` (= v0.15.0); `golang.org/x/crypto v0.28.0` | `offchain/go.mod` |
| `pqcSign` / `pqcVerify`: hex-decode → `Init` (re-init every call) → op → hex; `Clean` deferred | `crypto.go:34-74` |
| KEM keygen/encap/decap with hex at every boundary; HKDF-SHA3-256 (`hkdf.New(sha3.New256, ss, nil, info)`); AES-256-GCM, random 12-byte nonce; key wrap under a zero nonce | `kem.go` |
| Envelope v2 (field order `_qc_env,v,kemAlg,aeadAlg,kdf,credId,wraps,fields`), `credId="env-"+32hex`, fields sorted, per field: data key, seal, HKDF `qchain/trackB/v1|<credId>|<key>`, wrap; values via HTML-escaping `json.Marshal`; `_salts` | `envelope.go:95-225` |
| Commitment canonical JSON (sorted, `SetEscapeHTML(false)`, newline trimmed) → SHA3 hex → sign the 64-char hex; salt 16 B → 32 hex; field hash `sha3Hex(salt:key:value)`; golden string in tests | `commitment.go`, `commitment_test.go:21-52` |
| Issuance: `normalizeIssueAttributes` (incl. `parseExpiryDate` → `time.LoadLocation`) → `commitAttributes` → `sealCredentialEnvelope` → (IPFS) → commitment, Dubai `issuedAt` → `signCommitment` → `json.Marshal(fieldHashes)` → (chain) | `credentials.go:140-213` |
| Verification: `parseDisclosedPayload` → `effectiveStatus` (→ `dubaiDate` → uncached `time.LoadLocation` when an expiry exists) → `commitmentHash` → `EqualFold` on the issuer key hex → `pqcVerify` → `verifyDisclosedFields` (incl. `validateDisclosureSalts`) → `pqcVerify(sha3Hex(raw))` | `verification.go:126-158`, `chain.go:279-301` |
| Backend steps around holder actions: activation key-hex checks (`mobile.go:108-123`), retrieval `looksLikeEnvelope` full-JSON scan (`mobile.go:178`), presentation `validatePresentationRequest` (`mobile.go:483`) | `mobile.go` |
| Holder side (the wallet): payload = recursively sorted, compact, non-HTML-escaped JSON `{credentialID:"CRED-%04d", disclosedFields, salts, timestamp:"YYYY-MM-DDTHH:MM:SS.mmmuuu"}`; signs the UTF-8 of its SHA3 hex with re-init + hex-decoded key; envelope decrypted step by step as mirrored by `decryptEnvelope` | `crypto_service.dart:455,588-673,693`, `cmd/keygen/main.go:41-133`, `db.go:113` |
| Ledger record: sorted-key JSON (`DocType … Signature, PublicKey`), written by a tx whose arguments also carry the signature/key hex | `QChaincode.js:82-126,262` |
| Main repo is **public**; its CI fires only on pushes to its `main` | GitHub API, `.github/workflows` |

## 1b. Library and platform facts (primary sources)
- **liboqs 0.16.0** (tag `0.16.0`, commit `5a1a854b0dc9…`):
  - names: `ML-DSA-44/65/87`, `Falcon-512/1024`, `Falcon-padded-512/1024`, `SLH_DSA_PURE_{SHA2,SHAKE}_{128,192,256}{S,F}`, `ML-KEM-512/768/1024`, `HQC-1/3/5` (HQC enabled by default again).
  - Minimal-build tokens (`alg_support.cmake`): `KEM_ml_kem_{512,768,1024};KEM_hqc_{1,3,5};SIG_ml_dsa_{44,65,87};SIG_falcon_{512,1024};SIG_falcon_padded_{512,1024};SIG_slh_dsa_pure_{sha2,shake}_{128,192,256}{s,f}`. The 22 non-HQC tokens equal liboqs's own `STD` list.
- **Implementations:**
  - ML-KEM and ML-DSA: portable / AVX2 / aarch64, chosen at runtime in distributable builds; Falcon: portable / AVX2 / aarch64.
  - **SLH-DSA (own SHA-2/Keccak) and HQC: portable C only.** "Portable" still means the compiler auto-vectorises with the baseline instructions (SSE2 on x86-64, NEON on arm64), and B3 recompiles with `-march/-mcpu=native`. So "same code everywhere" holds only for B1/B2 and only for hand-written SIMD.
- **Build switches:**
  - Portable: `-DOQS_DIST_BUILD=OFF -DOQS_OPT_TARGET=generic` (+ `OQS_USE_ARM_{NEON,AES,SHA2,SHA3}_INSTRUCTIONS=OFF` on arm64); native: `OQS_OPT_TARGET=auto`.
  - With OpenSSL on, liboqs's RNG = OpenSSL `RAND_bytes`.
  - With `DIST_BUILD=OFF`, `OQS_CPU_has_extension` returns 0, so `oqsconfig.h` is recorded.
  - Tests are removed by `OQS_BUILD_ONLY_LIB=ON`, so our gate build keeps it OFF.
- **liboqs-go `v0.16.0`** (proper Go tag). Traps the code handles:
  - `Init` keeps *our* key slice and `Clean` zeroes it; `ExportSecretKey` aliases the internal slice. So we always copy, and a test proves keys survive `Clean`.
  - `Verify` returns `false,nil` for a bad signature but an error for a wrong length.
  - `Init` leaks **two** C strings plus two Go strings per call (as in production; matters for E9).
- **Go 1.24.13 everywhere:**
  - `golang:1.24.13-bookworm` (amd64 + arm64); Homebrew `go@1.24` = 1.24.13 on the Mac (keg-only).
  - Go 1.24's `crypto/rand` uses the fast getrandom vDSO only on Linux ≥ 6.11 (WSL2 6.18: yes; Colima/VM: probably not). So the kernel is recorded and `crypto/rand` cost is micro-benchmarked in E0.
- **OpenSSL 3.0.22** (bookworm) has no HPKE API, so DHKEM is implemented by hand per RFC 9180 and checked against **8 official vectors** (4 each for kem_id 16 and 32; source file SHA-256 `61fc662f…`).
- **Colima 0.10.3:** `colima start --vm-type vz --cpus 4 --memory 4`. Mounts are not needed any more (see §2).
- **Mac thermal pressure** without admin rights: `notifyutil -g com.apple.system.thermalpressurelevel` (instant; 0 = nominal). `pmset -g therm` only reports recorded warnings.
- **GitHub:** `teamqchain` was created on 2026-10-03; new organisations restrict third-party (OAuth) apps by default, which can block Git Credential Manager / VS Code logins on the private repo (§11 step 0).

## 1c. Where the code lives
- **Private repo `teamqchain/qchain-kpi`** (you create it; collaborators of the main repo get write access). The harness sits at its root (module `qchain-kpi`); later `git subtree add --prefix=kpi-benchmarking` into the public repo.
- **Clones next to each machine's main clone:** Mac `/Applications/Programs for Fun/qchain-kpi`; VM next to `~/Desktop/QChain/QChain-PQC-blockchain` (confirmed at setup); Obeid's laptop next to his main clone (found read-only at setup; if it is inside OneDrive I stop and ask).
- **Spaces in paths:** every bash variable quoted, `set -euo pipefail`, arrays, `-print0`; `shellcheck` on all scripts; PowerShell `-LiteralPath`; the Mac smoke test runs from the spaced path.
- **What may be committed (your 10 Oct answer):** everything, including code, results and the full plan (`docs/plan.md`). The private repo is merged into the public repo **after the paper's first draft has been submitted for review**, when sharing everything is fine.
  - **Caveat:** if the chosen venue uses **double-blind review**, a public repo with the team's names during review can break anonymity. In that case merge after acceptance, or share an anonymised copy.
  - Credentials never enter any file (passwords, PINs, tokens). The automatic per-run scan for hostnames, usernames, home paths, IP/MAC addresses and AnyDesk IDs stays, as basic hygiene.

## 2. Architecture
- **All measurement and orchestration logic is Go inside the Docker image** (same on all 3 machines). Host launchers (`run.sh` macOS/Linux, `run.ps1` Windows PowerShell 5.1) only do preflight, sleep blocking, host monitoring, build/run, copy-out, checksum verification and the index row.
- **Results never pass through a shared folder during a run:**
  - the container writes into a per-run Docker **named volume** `kpi-out-<run-id>`;
  - host-monitor samples are delivered into it every 60 s with `docker cp`, as chunk files plus completion markers, so no process ever runs inside the measurement container;
  - at the end the launcher runs `docker cp` to the quoted host path, verifies checksums, then removes our container and volume (ours only).
  - This avoids Colima mounts of spaced paths, Spotlight indexing, Windows Defender scanning and root-owned files.
- **Block** = one fresh process for one (experiment, algorithm, operation or arm set, parameters, repetition, attempt).
  - The block ID encodes all of these, so file names never collide.
  - Seeds come from (batch seed, block ID).
  - Writes use temp file → fsync → rename, plus a journal row.
- **`kpi-batch`** (orchestrator inside the container):
  - expands a JSON run plan; repetitions rep-major, order shuffled per rep;
  - runs canaries; evaluates quality flags at every checkpoint (~10 min) and at the end, re-inserting externally flagged blocks at seeded random later positions;
  - **resumes** after interruption, but refuses if the image digest, harness commit or thresholds changed, and runs a canary at each resume;
  - honours a STOP file and SIGTERM, runs under `docker run --init`, children get `oom_score_adj=1000`;
  - subcommands `--dry-run`, `estimate` (time and data size), `calibrate`, `verify`.

## 3. Folder layout (private repo root)
```
qchain-kpi/
  README.md  .gitattributes (eol=lf; *.gz binary)  .gitignore (+ scratch/)  .dockerignore
  go.mod go.sum   module qchain-kpi; go 1.24.0; liboqs-go v0.16.0; x/crypto v0.28.0; no toolchain line (tested)
  cmd/       kpi-env kpi-prim kpi-flow kpi-ledger kpi-batch kpi-conc kpi-canary-native(macOS, pure Go)
  internal/  algs pqc classical gostd qchain(verbatim production copies, provenance in docs/provenance.md) timing record quality
  csrc/      kpi_c.c (L0 timing, stack painting)  kpi_malloc.c (LD_PRELOAD heap counter)
  docker/    Dockerfile (targets: oqs, test, runtime; ARG VARIANT=b1-prod|b2-portable|b3-native|b4-noossl)  oqs-gate.sh
  plans/     smoke trial e1 e2 e8 e10 e11 e13 … .json      config/ thresholds.json (one section per machine)
  scripts/   run.sh run.ps1 monitor-mac.sh monitor-linux.sh dev.sh native-mac.sh check-sync.sh
  testdata/  rfc9180/ golden/
  analysis/  (phase 2, Python venv from /usr/bin/python3)
  docs/      methodology  experiment-matrix (incl. classical key formats per arm)  qchain-flow (included/excluded per stage)
             data-dictionary  provenance  decisions-log  plan (copy of the research plan)
  results/   linux/ mac/ windows/ (MACHINE.md, INDEX.md, <run-id>/)  combined/
```

## 4. Components
| Package / binary | Responsibility |
|---|---|
| `internal/algs` | 30-algorithm registry (names §1b, NIST category, tier). Gate at start: all 25 liboqs names enabled, `LiboqsVersion()=="0.16.0"`, `runtime.Version()=="go1.24.13"` |
| `internal/pqc` | liboqs-go arms `reuse`, `reinit` (fresh key copy per call, made untimed), `prod` (exact production bodies) |
| `internal/classical` | OpenSSL 3.0 via cgo, one C wrapper per operation, OpenSSL parameter arrays built in C, pre-fetched digests, error queue cleared. Keys: ECDSA sk raw 32 B / pk SEC1 65 B; Ed25519 and X25519 raw; RSA-3072 PKCS#8 / SPKI DER; P-256 DHKEM sk 32 / pk 65. Arms: `reuse` keeps the key object; `reinit` decodes key bytes per call; `prod` adds hex. DHKEM: suite_id `"KEM"‖I2OSP(kem_id,2)`, `I2OSP(L,2)` in LabeledExpand, P-256 point validation, X25519 all-zero check, injected-ephemeral path for vector tests only |
| `internal/qchain` | **Verbatim copies** (your decision) of the production functions in §1:<br>• from `offchain/`: `commitment.go`, `envelope.go`, `kem.go`, the `crypto.go` signing helpers, `verification.go` plus `effectiveStatus`/`dubaiDate`/`mustLoadLocation`, the activation/retrieval/presentation request checks;<br>• from the keygen tool: `decryptEnvelope`, the wallet mirror;<br>• plus a Go mirror of the wallet's payload builder.<br>Each function records its file:line and main-repo commit `9026aba`. **The only edit:** algorithm names become parameters, and every edit is listed in `docs/provenance.md`. `scripts/check-sync.sh` compares the copies with the main repo. Synthetic attributes are ASCII-only, so Go and Dart encode identically |
| `internal/timing` | Sampling rules of §6; `LockOSThread`; batching; interleaving; GC policy |
| `internal/record` | Compact per-block files (`seq` implicit, arm index, `ns`, plus `ok`/`bytes` only where they vary), block summaries, manifests from an **allow-list** of fields |
| `internal/quality` | Busy = user+nice+system+irq+softirq; steal for the CPU in `cpuset.cpus.effective`; own `cpu.stat` incl. `nr_throttled`; host chunks; flag rules |
| `cmd/kpi-env` | E0: CPU model/flags, kernel, mitigations, glibc, gcc, Go, liboqs version + CMake flags + `oqsconfig.h` + `OQS_CPU_has_extension`, OpenSSL version, cgroup version/limits, `memory.peak` availability, effective timer resolution + overhead, `crypto/rand` micro-benchmark, enabled algorithms |
| `cmd/kpi-prim`, `kpi-flow`, `kpi-ledger`, `kpi-conc` | E1/E2/E6/E7/E8/E10/canary; E11; E13; E12 |
| `cmd/kpi-canary-native` | Mac only: the same canary natively at user-interactive and background priority once per session, giving P-core and E-core reference speeds |
| `csrc/` | E3 L0 arms (pure C timed region, called through cgo so L0/L1 interleave) and E9 memory binary |

## 5. Docker image (built natively per machine)
- **Pinned inputs:**
  - `golang:1.24.13-bookworm@sha256:<index digest>` and `debian:bookworm-slim@sha256:<index digest>` (digests recorded at first build);
  - apt from `snapshot.debian.org` at a fixed date. If unreachable, the build falls back to live mirrors and this is flagged in the manifest. `dpkg -l` is saved;
  - liboqs tag `0.16.0`, commit asserted;
  - ACVP vector files downloaded by liboqs's tests are checked against pinned SHA-256s.
- **Variants** (all `-GNinja -DBUILD_SHARED_LIBS=ON -DOQS_BUILD_ONLY_LIB=OFF -DOQS_MINIMAL_BUILD=<25 tokens>`):
  - B1 `-DOQS_DIST_BUILD=ON` (production);
  - B2 portable (§1b);
  - B3 `-DOQS_DIST_BUILD=OFF -DOQS_OPT_TARGET=auto`, recording what `-march/-mcpu=native` resolves to;
  - B4 B1 + `-DOQS_USE_OPENSSL=OFF`.
- **liboqs gate:** pytest `tests/test_cmdline.py tests/test_kat.py tests/test_acvp_vectors.py`, `OQS_BUILD_DIR` set, `--numprocesses auto`, `SKIP_TESTS=examples`, plus "all 25 enabled". Any failure fails the build. Outputs are saved to `/opt/kpi/build-info/`.
- **`test` target:** `go vet` + `go test ./...` (§14). **`scripts/dev.sh`** runs this target, so the Mac dev loop needs no native liboqs/OpenSSL.
- **`runtime` target:** bookworm-slim + `libssl3` + `ENV TZ=Asia/Dubai` (production parity) + `valgrind` (E9 only); non-root user owning `/out`. Images are labelled with variant, liboqs commit and harness commit; only our labelled images are ever removed.
- **Builds use a dedicated `docker buildx` builder `kpi`**, stopped after each build and removed at the end of the study, so Obeid's build cache is never touched. The Mac also needs `docker-buildx`.

## 6. Sampling rules
- **Inputs:**
  - a separate untimed "prep" process makes keys and fixed pools of **P = 256** messages/signatures/ciphertexts per (algorithm, rep), cycled during sampling; P is recorded;
  - messages are fresh 64-char hex strings (what QChain signs);
  - crypto randomness stays real.
- **Cold:** the first call of an operation in a fresh process that has run no other crypto operation. E1 runs one process per operation. In interleaved blocks the first call per arm is labelled `first`, not `cold`.
- **Classification fast / slow:** probe median < 1 ms (from the trial's probe pass) → fast. **Frozen per machine × algorithm × operation** in `thresholds.json`, so it can't flip between reps.
- **Fast:** 1,000 warm-up + 10,000 timed samples.
- **Slow:** time box ≥ 60 s **and** ≥ 30 samples, **no cap**.
- **Batching:** operations whose median is below 100× the effective timer resolution (e.g. sub-µs hex decode, `Clean`) are timed in fixed batches of B calls per sample; B is recorded.
- **Interleaved arms** (E2, E3, E7, E10): arms alternate in seeded random order inside one block, and **every arm keeps the full approved budget** (10,000 samples, or 60 s and ≥ 30). That means no reduction, while drift is cancelled.
- **Composites (E11):** 20 untimed warm-ups, then 2,000 samples; composites whose probe median ≥ 30 ms follow the slow rule (§13).
- **GC policy (§13):**
  - micro-benchmarks (E1–E3, E6–E8, E10): GC off during timed loops (`SetGCPercent(-1)` + memory limit), with an untimed `runtime.GC()` before each loop and every 1,000 samples; bytes and allocations per op and GC counts are reported separately;
  - E11/E12: default GOGC (production-like).
- **Correctness (untimed):**
  - every signature verified, every encapsulation decapsulated and compared, every verify result checked;
  - keygen: every key's lengths checked, and a seeded subset (every 50th, ≥ 3 per block) fully exercised (sign/verify or encaps/decaps), because a full check of SLH-DSA-s keys would take hours.
- **Timer:** Go monotonic `time.Now/Since`; L0 `clock_gettime(CLOCK_MONOTONIC)`; resolution and overhead reported, never subtracted.

## 7. Experiments → code → phase
| Exp | Implementation | Phase |
|---|---|---|
| E0 | `kpi-env` every run + launcher host info (allow-listed) | 1a |
| E1 | `kpi-prim` L1 `reuse`, one process per operation | 1a |
| E10 | ML-DSA-44/65/87, `reuse`, three arms:<br>• issuer: commitment hash hex, issuer key;<br>• holder: payload hash hex, holder key;<br>• issuer-again: same key, separate pool — this arm gives the A/A floor.<br>Each arm has its own message pool. TOST on the paired per-rep log-ratio of medians (n = 5, 90 % t-interval inside ±margin), per machine × parameter set; "all equivalent" needs all 9 to pass | 1a |
| canary | SHA3-256 throughput, 15 × 100 ms windows (pure Go, independent of liboqs build) | 1a |
| E2 | arms `reuse`/`reinit`/`prod` + component arms (hex decode, `Init`, `Clean`, batched); classical arms per §4 | 1b |
| E8 | sizes incl. hex/JSON; Falcon and ECDSA-DER signature-length distributions | 1b |
| E11 | `kpi-flow`. Stages and the actor of each step:<br>• activation: holder keygens + backend key checks;<br>• issuance: backend;<br>• retrieval: backend envelope check + holder decrypt;<br>• presentation: holder sign + backend request check;<br>• verification: backend, incl. decoding the chain record as its own sub-step, counted in the total and reported separately.<br>IPFS, Fabric, MySQL and HTTP are excluded.<br>**Grid:** 16 curated suites × N∈{1,5,10,20,64} × k∈{1,⌈N/2⌉,N}; all 176 pairs at N=10, k=5; value sizes {16, 256, 4096} B for the production suite.<br>**Curated-suite variants (your decision):** SLH-DSA-**SHA2**-128f/128s/192f/256f and **standard** Falcon-512/1024.<br>**Defaults:** names `field01…`, 16-char ASCII values, `expiryDate` set.<br>**Measurement (your decision):** each stage once per setting it depends on (issuance: suite, N, value size; retrieval: KEM, N; presentation and verification: signature, N, k; activation: suite), with the approved samples and 5 reps. Cell totals = sum of stage results, plus one fully chained run of the production suite to validate the sum. | 1c |
| E13 | **bytes per transaction**:<br>• proposal arguments + written record + creator/endorser certificates and signatures + `Issuer` ID length;<br>• the holder-key binding transaction;<br>• credentials per block from `configtx.yaml`;<br>• GB per million;<br>• IPFS bytes vs N.<br>Calibrated on a **real block and record captured read-only on the VM before QChain stops** | 1c |
| E6, E7 | E6 size sweep. E7 corrupts signatures by bit-flip at the same length. KEM tampering semantics differ: ML-KEM/HQC return a different key (checked), P-256 errors, X25519 gives a different secret | 2 |
| E3, E9, E4, E5, E12 | L0 arms; memory (Go heap, C heap via `LD_PRELOAD`, stack painting, peak RSS, `memory.peak`); B2–B4 images; native Mac (compiler confound Apple clang vs gcc 12 stated or avoided with Homebrew `gcc@12`); concurrency VM {1,2} ≤ 15 min, Mac {1,2,4}, Windows {1,2,4,8} | 2 |

## 8. Run quality, canary, calibration
- **Host monitors** (5 s; host clock in UTC epoch-ns; offset to the container measured at start and end):
  - **Mac:**
    - one `top -l 0 -s 5` stream; other CPU = (host busy − Virtualization VM process − top/monitor) + (VM busy − container `usage_usec`), divided by 8 logical CPUs;
    - `notifyutil` thermal pressure, `pmset -g batt`, Low Power Mode;
    - a **core-type check:** each in-VM canary is classed P-like or E-like against the native references.
  - **Windows:**
    - `% Processor Time`, `% Performance Limit` + `Performance Limit Flags`, thermal zone, and the `vmmemWSL`/`vmmem` process (excluded from other CPU);
    - the counter set is confirmed read-only in the first session, and a missing counter is recorded as "n/a".
  - **VM:** the monitor runs under `taskset -c 0 nice -n 19` (the benchmark is pinned to vCPU 1); containers checked every 60 s; steal support checked once read-only (if steal is never reported, the canary carries that role).
- **Flags (per machine, `thresholds.json`):**
  - steal > 2 %;
  - other CPU > 5 %;
  - throttling (Mac pressure ≥ 1; Windows `% Performance Limit` < 100);
  - CPU-quota throttling `nr_throttled` > 0;
  - E-like core on the Mac;
  - canary outside C_m;
  - drift: |block median ÷ reference − 1| > max(D_m, 3 × bootstrap SE of the block median).
- **Formulas, fixed in code before any trial:**
  - D_m = max(3 %, 2 × d95), where d95 = 95th percentile of rep-to-rep median deviation in that machine's trial. This gives the "loose" VM limit as a number.
  - C_m = max(3 %, 3 × robust SD of trial canary deviations).
  - A/A floor F = upper 90 % bound of |issuer vs issuer-again log-ratio|.
  - E10 margin = max(2 %, 2F) per machine × parameter set.
- **Attempt policy (§13):**
  - **Reference** = median of the 5 first attempts, frozen.
  - **External-cause flags** (steal, other, throttling, quota, core type, canary) → repeat (≤ 2), re-inserted at a later random position; the headline uses the first attempt without an external flag.
  - **Drift-only flags** → one diagnostic re-run, but the original stays in the headline (re-running until it agrees would make the confidence intervals falsely narrow).
  - **Reporting:** an all-attempts sensitivity analysis.
- **Trial (≈ 30 min, at the same clock time as the main runs where possible):**
  - E0; Mac core references; canary every 5 min;
  - a **probe pass** (cold + 5 calls for all 30 algorithms × operations; feeds `estimate` and freezes fast/slow);
  - E10 arms for all three ML-DSA sets;
  - 5 reps of fast operations (ML-DSA-44, ML-KEM-768, Ed25519, X25519, Falcon-512 sign/verify) and 3 reps of SLH-DSA-SHA2-128f and HQC-1.
  - Then `kpi-batch calibrate` writes **that machine's** section of `thresholds.json`, committed before that machine's first main run.
- **Night-time background inventory** (read-only preflight, recorded): Windows Automatic Maintenance time and Defender schedule; macOS Time Machine/Spotlight; VM systemd timers (apt-daily, unattended-upgrades, snapd). If any falls inside run windows, I bring options to you / Obeid / Omnia.

## 9. Launchers and preflight (refuse on failure; `--force` is recorded)
- **Container:** `docker run --init --name kpi-run-<id> --network none --cpuset-cpus=<1> --memory=<4g VM | 3g Mac/Win> --memory-swap=<same> -e GOMAXPROCS=1 --mount type=volume,src=kpi-out-<id>,dst=/out kpi-bench:<variant>-<commit>` (E12 alone uses more CPUs).
- **Every machine:**
  - harness commit = `git log -1 --format=%H -- . ':!results'`, must match the image label;
  - `git status --porcelain -- . ':!results'` must be clean;
  - ≥ 10 GB free;
  - no running container except ours (non-ours are counted, not named, except QChain names on the VM).
- **Strictness:** `smoke` records but does not enforce the quiet-machine checks; `trial`/`main` enforce all. Mac trial/main are started by you from Terminal with VS Code closed.
- **Mac:** Colima `vz` 4 CPU/4 GB (the launcher starts it if it is down); on AC (refuse on battery; warn < 98 %); **(added 10-11 Oct) the launcher pauses the macOS background analysis services and switches Wi-Fi off for the run after the image build, and restores both when the script ends**; thermal pressure 0; no Teams/Chrome/VS Code/OneDrive; Spotlight/`fileproviderd` quiet; Low Power Mode off; `caffeinate -i -s`.
- **VM:** GitHub runner, IPFS, MySQL inactive; steal baseline; self-relaunch in `tmux` (or `setsid nohup`) under `systemd-inhibit --what=sleep:idle`.
- **Windows (PS 5.1, ASCII-only):** `powershell -ExecutionPolicy Bypass -File …`; on AC; `SetThreadExecutionState` for the run only; files written with BOM-less UTF-8 + LF via `[IO.File]`; `$LASTEXITCODE` checked; no `2>&1` on Docker; no PS7 syntax.
- **After:** copy-out, checksum verification, INDEX row, `docker rm` + `docker volume rm` (ours), print commit/push commands. Pushes stay manual and happen after each night. On the VM, push from VS Code's terminal, since its GitHub login works only there.

## 10. Data and privacy
- **Run folder** `results/<os>/<UTC yyyymmdd-hhmm>_<machine>_<runtime>_<variant>/`: `manifest.json`, `blocks.csv`, `journal.csv`, `raw/<block-id>.csv.gz`, `monitor/`, `console.txt`, `SHA256SUMS`.
- **Machine IDs:** `mac-m1`, `win-ryzen`, `vm-x86v2`.
- **Privacy:**
  - manifests come from an allow-list (e.g. `docker info` reduced to version/CPUs/memory/kernel/cgroup);
  - repo-relative paths only;
  - at the end of every run, **all output files are scanned for that machine's actual hostname, username, home path, IP/MAC addresses and AnyDesk ID** (computed at run time), and the run fails if any appears.
- **Size:**
  - compact rows; `estimate` reports projected GB after the trial;
  - my estimate is ~0.2–0.35 GB per machine (gzipped), so about 1 GB in total; no file exceeds 50 MB; pushed nightly.
  - If the projection exceeds ~1.5 GB, I bring you options before running.

## 11. Order of work
0. **You:**
   - create `teamqchain/qchain-kpi` (private, empty) and give the collaborators write access;
   - **check Organisation → Settings → Third-party access → OAuth app policy**, and approve "Git Credential Manager" and "Visual Studio Code" if apps are restricted. Otherwise Obeid's and the VM's logins can't reach the private repo.
   - **Me:** right after approval, update memory (both copies) and plan v3.6; install `colima docker docker-buildx go@1.24 shellcheck`; clone the repo; then start coding.
1. **Phase 1a, the minimum for the Mac trial:**
   - code: scaffold, Dockerfile (B1 + `test` target) + liboqs gate, `algs`, `pqc`, `classical` + RFC vectors, `qchain` (enough for E10 messages), `timing`, `record`, `quality`, `kpi-env`, `kpi-prim` (E1, E10, canary, probe), `kpi-batch` (incl. `calibrate`, `estimate`), `kpi-canary-native`, `run.sh` + Mac monitor; plans `smoke`, `trial`, `e1`.
   - Then: Mac smoke run → Mac trial → calibrate → commit the Mac thresholds → **Mac main runs begin** (E1 first).
2. **Phase 1b:** E2, E8, `run.ps1` + Windows monitor, Linux monitor + VM preflight → push → Obeid's laptop (McAfee + counters + access check, clone, build, smoke, trial, main nights).
3. **Phase 1c:** E11 (+ golden cross-check) and E13. On every machine **E11 runs before any priority-2/3 experiment**. VM: read-only inventory, **real block/record capture**, stop QChain, clone, build, smoke, trial, main runs (24/7).
4. **Phase 2:** E3, E4, E5, E6, E7, E9, E12, analysis/figures/dashboard, README complete.

## 12. Runtime budget (rough; exact numbers come from `estimate` after each trial)
- **Per machine:**
  - **Priority 1:** E1 4 h + E2 9.5 h + E8 0.2 h + E11 ~25 h (with de-duplication; ~2× without) + E13 compute = **~39 h**.
  - **Priority 2:** E3 8 h + E9 1 h + E10 0.5 h = ~9.5 h.
  - **Priority 3:** E4 8.5 h + E6 14 h + E7 3.5 h + E12 1 h = ~27 h.
  - E5 adds ~4 h on the Mac.
  - **≈ 75–80 h per machine.**
- **Capacity:**
  - the VM runs 24/7, so ~3.5 days;
  - the laptops run nightly (~8–9 h), so ~9 nights each, priority 1 within the first ~5 nights.
- **Nothing is dropped.** If the schedule can't reach your deadline, I come back to you with the exact numbers.

## 13. Changes to approved rules (please confirm by approving this plan)
1. **Pin to one vCPU on Mac and Windows too** (`--cpuset-cpus`, `GOMAXPROCS=1`); quota throttling is recorded and flagged. **Amended 10 Oct (user decision): no `--cpus` quota at all.** A quota equal to the pinned CPU throttled a fully busy container in 29 of 100 periods (~1.8 ms each), so it was dropped; the throttling check remains as a sanity check, and kpi-batch refuses trial/main runs if a quota is set.
2. **Canary every 10 min** + start/end; Mac core-type check.
3. **E11 composites with slow operations follow the slow-op rule** (60 s, ≥ 30), instead of 2,000 samples. Otherwise one SLH-DSA-s block would take 30–100 min.
4. **Interleaved arms**, each with the full approved budget (no reduction).
5. **Sub-resolution operations timed in batches.**
6. **GC policy:** off during micro-benchmark timed loops, with allocations reported separately; production-like in E11/E12.
7. **Attempt policy:** external flags repeat; drift-only stays in the headline; plus a sensitivity analysis.
8. **Keygen correctness on a subset** (all key lengths checked); everything else checked fully.
9. **Fast/slow frozen from the trial; thresholds per machine**, each committed after its own trial; formulas fixed in code now.
10. **Fixed input pools** (P = 256, cycled).
11. **Results via a named volume + `docker cp`** (no shared folders), no swap, resumable batches.
12. **E11 additions:** value-size sweep (delivers the §8 KPI), `expiryDate` present, chain-record decoding as a reported sub-step.
13. **E13 becomes a per-transaction model** with a **required** read-only real-block capture on the VM before QChain stops.
14. **Machine order Mac → Obeid → VM** (your 9 Oct order).

## 14. Verification
- **Image build:** liboqs gate (correctness, KAT, ACVP) for all 25 PQC; `go vet`; `go test`:
  - RFC 9180 (8 vectors);
  - backend golden commitment string; envelope/payload round trips; Go-vs-Dart encoding agreement on the ASCII alphabet;
  - all 30 adapters round-trip; keys survive `Clean`;
  - no toolchain line; ASCII-only `.ps1`;
  - the manifest allow-list; plan expansion, block IDs and seeds deterministic; resume refusal; flag logic on synthetic data.
- **Golden cross-check:**
  - `scripts/check-sync.sh` proves the copied functions still match the main repo, apart from the listed parameter edits;
  - fixtures (committed, with provenance) prove the parameterised copies give byte-identical output for ML-DSA-44 + ML-KEM-768;
  - the production suite's chained run checks that the stage sums are valid.
- **Smoke run:** all files present, `verify` passes, INDEX row, privacy scan clean.
- **Trial run:** every signal present (steal on VM, thermal on Mac/Windows, core type on Mac), canary stable, A/A floor computed, `estimate` produced.
- **Every main run:** preflight passes; `verify` before commit.
