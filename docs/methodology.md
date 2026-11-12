# Methodology (as implemented)

This file describes what the harness does when it measures. It is written from the code. The research plan is `docs/plan.md` and the coding plan is `docs/coding-plan.md`. Any difference between this file and the code is a bug in this file.

## 1. What is measured

**Algorithms (30)** are registered in `internal/algs`:
- **Post-quantum (25, from liboqs 0.16.0):**
  - signatures: ML-DSA-44/65/87, SLH-DSA (pure) SHA2/SHAKE-128/192/256 s/f, Falcon-512/1024, Falcon-padded-512/1024;
  - KEMs: ML-KEM-512/768/1024 and HQC-1/3/5.
- **Classical (5, OpenSSL 3.0 through cgo):** ECDSA P-256 (SHA-256, DER signatures), Ed25519, RSA-3072-PSS (SHA-256, salt length = digest length), and DHKEM(P-256) and DHKEM(X25519), both with HKDF-SHA256 as in RFC 9180.

**Layer L1.** Both families are called from Go through cgo, so they cross the same Go-to-C boundary. The E1 arm is **reuse**: each library is used the way its API is meant to be used repeatedly.
- For liboqs, one `Signature` / `KeyEncapsulation` object is initialised once with its key.
- For OpenSSL, one parsed `EVP_PKEY` is kept.
- Only the operation itself is inside the timed region.
- liboqs's verify API takes the raw public key on every call. OpenSSL verifies against the parsed key object.

**Operations.**
- **Signatures:** `keygen`, `sign` and `verify`. Messages are fresh 64-character lowercase hex strings, which is what QChain signs.
- **KEMs:** `keygen`, `encaps` and `decaps`.

## 2. Processes and isolation

- **One block, one process.**
  - A block is one fresh `kpi-prim` process for one (experiment, algorithm, operation, repetition, attempt).
  - In E1 each operation has its own process, so the first call is a true *cold* call: no other cryptographic operation ran in the process before it.
  - In E10, three arms share the process. Their first calls are labelled *first*, not cold.
- **Container.** The container is pinned to one CPU of the container's kernel (`--cpuset-cpus=1`, `GOMAXPROCS=1`) and has **no CPU-time quota**. A quota equal to the pinned CPU (`--cpus=1`) made Linux pause a fully busy container in 29 of 100 scheduling periods, for about 1.8 ms each time (measured 10 Oct). Every such pause would land inside a timed sample. Further settings:
  - no network;
  - memory limited with no swap (3 GB on the Mac and Windows, 4 GB on the VM);
  - an unprivileged user;
  - `TZ=Asia/Dubai`, as in production.
  - The orchestrator refuses trial and main plans if any of this is not so.
- **Host background services (Mac).** The macOS background analysis services (Photos analysis, the Spotlight AI indexer and the suggestion services) are paused with SIGSTOP for the duration of every run and resumed afterwards. They run whenever the Mac is idle on mains power and otherwise used up to two cores and caused thermal pressure. The paused services are logged with every run.
- **Network (Mac).** After the image has been built (the build needs the internet), the launcher switches Wi-Fi off for the run and switches it back on when the script ends on any exit path (only if it was on before). Nothing can then sync, update or download during a measurement. The container itself never has a network. The VM stays on the network (it is controlled over SSH) and the Windows laptop is controlled over AnyDesk, so this applies to the Mac only.
- **Thread and memory settings.** The measuring goroutine is locked to its OS thread. Measuring processes raise their `oom_score_adj` to the maximum, so the kernel would kill a measuring process, never the orchestrator.
- **Input sets** are made by a separate, untimed `kpi-prim prep` process, once per (algorithm, repetition):
  - **Contents:** keys from the libraries' own randomness, plus a fixed pool of P messages and their signatures, or P ciphertexts and their shared secrets.
  - **P:** 256, or 64 for the SLH-DSA "s" sets.
  - **Cycling:** call *i* uses pool entry *i* mod P.
  - **Per repetition:** each repetition has its own keys and messages, so the spread between repetitions includes key-to-key variation.
  - **Probe pass:** uses a separate small set of P = 6.
  - **Location:** input sets live only inside the container, in `/tmp`. Each block records the fingerprint of its key.

## 3. Sampling (`internal/timing`)

1. **First call.** It is timed and recorded separately (`cold:` or `first:` in the raw file).
2. **Warm-up.** Untimed calls: 1,000 for fast operations, 3 for slow ones.
3. **Fast operations (probe median < 1 ms).** 10,000 timed samples per arm.
4. **Slow operations.** Sampling continues until the arm's accumulated measured time is at least 60 s **and** it has at least 30 samples. There is no cap.
5. **Classification** (fast or slow) is frozen per machine × algorithm × operation from that machine's trial, so it cannot flip between repetitions. E10 uses the classification of `sign`.
6. **Interleaving.** Several arms in one block (E10) run in a seeded random order in every round, and each arm keeps the full budget.
7. **Batching.** An operation whose probe median is below 100 × the effective timer resolution is timed in batches of B calls per sample, with B = ⌈100 × resolution ÷ median⌉. The raw file records the batch total. For the operations of E1 and E10 this is expected to be B = 1 everywhere.
8. **Garbage collection.** Automatic collection is off (`SetGCPercent(-1)`, with a 1 GiB soft limit as a safety net only). Collections are forced outside the timed region: before sampling and every 1,000 samples. Collections the runtime starts by itself are counted (expected 0). Go heap bytes and allocations per call are measured separately after the timed phase.
9. **Correctness, outside the timed region:**
   - every signature is verified;
   - every encapsulation is decapsulated and compared;
   - every verification and decapsulation result is checked;
   - key generation checks the lengths of every key and fully exercises a seeded subset of keys (every 50th, at least 3 per block).
   - Any failure invalidates the block and stops the batch.
10. **Timer.** Go's monotonic clock (`time.Now`/`time.Since`). E0 records the effective resolution and the overhead of a clock read. Neither is subtracted.
11. **Memory and size.** Raw samples are kept in memory during the timed phase and written to a gzip file afterwards, so no file I/O or compression happens between samples.

The **smoke** profile (pipeline tests only) shrinks the budgets: fast is 10 warm-up + 200 samples; slow is 1 s and at least 3 samples.

## 4. Experiments

- **E0 (every session)** is `kpi-env`: CPU, kernel, mitigations, clock source, timer resolution and overhead, cgroup limits, versions, the liboqs build record and runtime CPU-feature detection, a `crypto/rand` benchmark, and the algorithm sizes.
- **E1:** every algorithm and operation with the reuse arm, 5 repetitions.
- **E10:** ML-DSA-44/65/87, three interleaved signing arms in one process, each with its own signer object and its own message pool:
  - **issuer:** commitment hashes (N = 10 attributes), signed with the issuer key;
  - **holder:** presentation-payload hashes (N = 10, k = 5 disclosed), signed with a separate holder key;
  - **issuer-again:** a second, independent pool of commitment hashes, signed with the issuer key. This is the A/A arm: it differs from *issuer* only in its messages.
  - **Equivalence** is tested offline with TOST on the paired per-repetition log-ratio of medians: n = 5, and the 90 % t-interval must lie inside ± the machine's margin.
- **E2 (integration cost):** for every algorithm and operation, two blocks per repetition, 5 repetitions:
  - **E2 block:** three interleaved arms in one process, each with the full approved budget:
    - **reuse:** the library's long-lived object, the same code as E1;
    - **reinit:** a fresh library object for every call: create it, run the operation, destroy it. For liboqs this is `Init`, the operation and `Clean`; for OpenSSL, loading the key into a key object, the operation and freeing the object. liboqs-go keeps the secret-key slice given to `Init` and `Clean` zeroes it, so every `Init` gets its own copy, restored outside the timed region. After key generation the secret key is copied before `Clean` (production does the same implicitly by hex-encoding it first);
    - **prod:** QChain's production function itself, hex strings in and out. For the post-quantum algorithms these are the backend's functions copied verbatim (`pqcSign`, `pqcVerify`, `kemGenerateKeypair`, `kemEncap`, `kemDecap`), with only the algorithm name as a parameter; signature key generation mirrors the keygen tool's inline steps (see `docs/provenance.md`). The classical algorithms have no production code, so their prod arm applies the same pattern to OpenSSL: hex-decode, fresh key object, operation, hex-encode.
    - A classical key generation already creates a new key object on every call, so its reuse and reinit arms would be the same code; it has only reuse and prod.
  - **E2c block (components):** the steps that separate prod from reuse, each on its own: hex decoding of the inputs (`hex-decode-sk`, `-pk`, `-sig`, `-ct`), creating the library object (`init`), destroying it (`clean`), and hex encoding of the outputs (`hex-encode-pk`, `-sk`, `-sig`, `-ct`, `-ss`). Which components an operation has is fixed in `internal/proto/e2.go`. E2c blocks are always fast mode; components shorter than 100× the timer resolution are timed in batches. `init` and `clean` are timed separately: for `init` the objects created during a sample are destroyed outside the timed region, for `clean` they are created beforehand.
  - **Batch sizes** for the new arms come from an add-on trial (`trial-e2` in the machine's plan folder, see section 6).
- **E8 (sizes; nothing is timed):** for every algorithm, one block for key generation (public and secret key) and one for sign (signature) or encaps (ciphertext and shared secret), one repetition. A block produces up to 1,000 keys or 10,000 outputs (at least 100; it stops after 60 s once the minimum is reached) and records the length of every item, so the real distributions of unpadded Falcon and ECDSA (DER) signatures and of RSA PKCS#8 private keys are captured; every other length is fixed and is confirmed on every item. Every output is checked: signatures are verified, ciphertexts decapsulated, and every 50th key fully exercised. QChain stores keys, signatures, ciphertexts and shared secrets as lowercase hex strings in JSON, so an item of n bytes takes 2n hex characters (2n + 2 bytes as a JSON string); the analysis derives these from the recorded lengths. Credential-level sizes (records, envelopes, payloads) belong to E13 and E11. Because sizes do not depend on machine noise, E8 blocks get no run-quality flags, are never repeated, and their first attempt is the result.
- **E11 (lifecycle composites):** the credential lifecycle through QChain's copied production code (`internal/qchain`, checked by `scripts/check-sync.sh`), stage by stage, each stage measured once per setting it depends on (`internal/cells`):
  - **activation** (signature, KEM): the wallet generates both key pairs and sends the public halves; the backend decodes the request and checks the keys (lengths of the measured suite, decision B);
  - **issuance** (signature, KEM, N, V): the issuance handler's computation: attributes, salts and field hashes, the per-field envelope encrypted to the holder, the commitment with Dubai `issuedAt`, its signature, the field-hash JSON;
  - **retrieval** (KEM, N, V): the backend's envelope check and the wallet's decryption of every field;
  - **presentation** (signature, k, V): the wallet's canonical payload, its SHA3 hash and signature, the request; the backend's request check;
  - **verification** (signature, KEM, N, k, V): decoding the chain's response (the chaincode's exact JSON; it carries the holder's KEM public key, hence the KEM dependence) and `verifyPresentation`; the decoding alone is a second, interleaved arm (`chain-decode`), reported separately;
  - **chained** (production suite, N, k): the five stages back to back on their own prepared inputs, to check that the stage results add up.

  Excluded (decision C): network, MySQL, IPFS and Fabric calls and HTTP response writing; the IPFS CID is passed in and the chain record is prepared in advance. Classical suites run the same copied code through OpenSSL twins (decision A). Grid: the 16 curated suites at every N in {1, 5, 10, 20, 64} and k in {1, ceil(N/2), N}; **all 176 signature x KEM pairs at N = 10, k = 5, measured whole**; the production suite with 256- and 4096-byte values; 991 cells, 5 repetitions. Budgets: 20 warm-up calls and 2,000 samples, or at least 60 s and 30 samples for cells whose probe median is at least 30 ms (the classification comes from the add-on trial `trial-e11`). Go's garbage collector runs normally (production-like); automatic cycles are recorded, not flagged. Each block prepares its own inputs (keys, pools of 32 credentials, 8 for verification with SLH-DSA 's' sets) before timing and records that time (`prep_ns`). Correctness: every commitment and holder signature and every verification outcome is checked, every decryption compared; a seeded subset of issued envelopes (every 50th, at least 3 per block) is fully decrypted after the timed phase.
- **E6 (message-size sweep):** sign and verify time of the 22 signature schemes over messages of 64 B, 1 KiB, 10 KiB, 100 KiB and 1 MiB, one process per (scheme, operation, size), with the library's reuse objects as in E1 (so the 64-byte cells line up with E1), 5 repetitions, the approved fast/slow budgets. Messages are random bytes from a seeded ChaCha8 generator (the content does not affect timing; QChain itself signs 64-character hex strings). Each (scheme, repetition) has one key; its prep holds, per size, a pool of min(P, max(8, 64 MiB / size)) messages (1 MiB: 64; the slow SLH-DSA sets: 8), regenerated from a stored seed in the measuring process (not stored: 1 MiB messages would make every prep file hundreds of megabytes), and valid signatures over them for the verify blocks. Every signature is verified and every verification checked, outside the timed region. The classification and batch sizes of the 220 E6 cells come from the add-on trial `trial-e6` (probe pass of every cell; same rules as the first trial).
- **E7 (rejection timing):** one process per algorithm, arms interleaved, each with the full approved budget. Signature schemes: verify with a valid signature (`valid`), with the same signature with one bit flipped (`badsig`), and with a valid signature over a different message of the same length (`wrongmsg`, the next pool entry's message). KEMs: decaps of a valid ciphertext (`valid`) and of the same ciphertext with one bit flipped (`tampered`). The flipped bit is at a seeded random position, a new position for each of the P pool entries, and the invalid inputs are made before timing. How each family rejects differs and is recorded per sample: verify returns false; the library returns an error (for example an ECDSA signature whose DER encoding the flip broke, or a P-256 ciphertext whose point is no longer on the curve); or decaps returns a different secret (ML-KEM's and HQC's implicit rejection, X25519). Every call is checked inside the call (a few nanoseconds), so batched samples are fully checked: valid input must be accepted and invalid input rejected, or the block fails. The classification is the operation's (E1's verify or decaps, frozen by the first trial); the batch sizes of the E7 arms come from the add-on trial `trial-e7`.
- **E3 (integration-pattern cost):** every operation of every algorithm, one block per (algorithm, operation), 5 repetitions, the layers interleaved in seeded random order, each with the full approved budget and E1's frozen fast/slow classification:
  - **l0:** the C library called directly (liboqs's C API; this harness's OpenSSL functions with the key object kept), timed **inside C** with `clock_gettime(CLOCK_MONOTONIC)`, the clock Go's own timer reads (`internal/l0`). The C jobs are E9's (`MemJob`), with the input pool copied into C memory once, so a timed call only selects a pointer.
  - **l0g:** the same C call timed from Go around the cgo call. **l0g - l0 is the cost of the Go-to-C boundary.**
  - **l1:** E1's reuse arm. **l1 - l0g is the Go binding's own work** (liboqs-go's buffers and copies, or this harness's cgo wrappers).
  - **l2** (the 5 classical algorithms): Go's standard library (`internal/gostd`: `crypto/ecdsa`, `ed25519`, `rsa`, `ecdh`, `hkdf`), the same key formats, key objects parsed once. DHKEM is written from RFC 9180 section 4.1 (Go 1.24 has no HPKE package) and reproduces the 8 official vectors. Every l2 result is checked with OpenSSL (its signatures verified, its ciphertexts decapsulated, its keys used), so both libraries are shown to compute the same scheme on every call.
  - **Null block** (one per repetition, cell `none/null`): a cgo call to a C function that does nothing (batches of 1,000) and two C clock readings with nothing between them (the C timer's floor). Always fast mode.
  - The timing engine accepts **self-timed arms** for l0: the arm reports the duration measured in C, which the engine records in place of its own reading; warm-up calls go through the untimed path. Every other experiment's arms are timed as before.
- **E9 (memory; nothing is timed):** every algorithm and operation, three blocks (processes) each, 5 repetitions:
  - **Routes.** `c`: the C library called directly (liboqs's C API; this harness's OpenSSL functions, with the key object kept as in the reuse arm) on a dedicated thread with an 8 MiB stack (glibc's default thread stack, which the threads running QChain's cgo calls also get) and a guard page. `reuse`: E1's arm. `prod`: E2's prod arm (QChain's production pattern, hex in and out, a fresh liboqs object per call). The Go routes are called from the block's goroutine locked to one OS thread.
  - **Stack high-water mark** (`c` only): before every call the thread paints its stack with a 64-bit pattern; afterwards the deepest word that changed gives the depth. A fixed helper frame makes even an empty job measure a stable baseline, which is subtracted. A test checks the method: a 64 KiB local array measures 64 KiB plus a few bytes.
  - **C heap per call:** `csrc/kpi_malloc.c`, loaded with `LD_PRELOAD` into E9's blocks only, wraps glibc's allocation functions (glibc 2.36 has no malloc hooks) and counts per thread: allocations, frees, bytes requested, usable bytes allocated (`malloc_usable_size`), live bytes and their peak. Per call: the peak above the start, bytes allocated, allocations, allocations minus frees (`heap-unfreed`, blocks left allocated) and live bytes after minus before (`heap-retained`, in usable bytes, so glibc's block rounding shows: 24 to 48 bytes for a short string). Because the counters are per thread, the Go runtime's other threads never enter a measurement. Tests check exact counts for known allocations.
  - **Go heap per call** (`reuse`, `prod`): bytes and objects from `runtime.MemStats` around the call (as E1 and E2 report them).
  - **Peak resident memory per block:** `VmHWM` is reset after setup (`/proc/self/clear_refs`) and read after the calls; reported as growth over the resident set before the calls and as the peak itself. Linux 6.2 and later read these from per-CPU counters approximately (up to 32 pages per CPU the process ran on; the container is pinned to one CPU), so growth is precise to about 128 KiB; the heap figures are exact. On the `c` route the painted 8 MiB stack is resident (painted before the reset), so its peak includes it.
  - **Budget:** 100 calls, at least 10, stopping after 20 s once the minimum is reached (slow SLH-DSA signing and RSA key generation). Call 1 is reported on its own (`first`: it includes one-time set-up such as OpenSSL's per-thread random generator); the statistics are over the calls after it. Calls cycle through a pool of 32 inputs per (algorithm, repetition), 16 for the SLH-DSA "s" sets. Every call is checked as in E1 (signatures verified, encapsulations decapsulated, every 50th key exercised).
  - **GC:** Go's default (production-like).
  - The run-wide cgroup `memory.peak` (the whole container; Linux 6.8 cannot reset it) is recorded in the manifest of every run.
  - **Valgrind is not used** (it was planned for E9): it does not run Go programs reliably and may not support liboqs's vector instructions; the counter and the stack measurement are checked by tests instead.
- **E12 (concurrency):** throughput and latency of QChain's backend under W concurrent workers, 5 repetitions.
  - **Workload:** E11's copied production code, closed loop: W workers each call a backend stage back to back, each with its own inputs prepared by `internal/flow` (nothing shared), in one process with `GOMAXPROCS = W` and Go's default GC (production-like). Stages: **verification** (N = 10, k = 5) and **issuance** (N = 10), 16-byte values. Suites (user, 15 Oct): ML-DSA-44 + ML-KEM-768 (production), Falcon-512 + ML-KEM-512, SLH-DSA-SHA2-128f + ML-KEM-512, ECDSA P-256 + DHKEM(P-256). Workers on the Mac: 1, 2 and 4.
  - **CPUs:** E12 plans name the container's CPUs (`cpuset`, Mac `0-3`: all 4 of the VM); every other plan stays pinned to CPU 1. With 4 workers nothing is left for the VM's kernel and Docker daemon, and the VM's CPUs may run on the M1's efficiency cores, so the 4-worker results are labelled as the saturated case.
  - **Timing** (`internal/conc`): all workers start together; 5 s of warm-up, then a 30 s window. A call counts when it completes inside the window; no call starts after it. Throughput = calls counted ÷ 30 s (and per second of the window). Latency = each counted call's start to end, in a log-linear histogram (64 buckets per power of two; quantiles are bucket midpoints, within 0.8 %; minimum, maximum and mean exact).
  - **Checks:** every warm-up call and every 64th window call of each worker is checked as in E11 (the checks take worker time, as they would in production); issuance envelopes of a seeded subset are decrypted afterwards.
  - **Memory:** the process's peak resident set over the run (as in E9, reset after the workers' preparation), and the run's cgroup `memory.peak` in the manifest.
  - **Quality:** the external-cause flags and repeats of the timed experiments apply; there is no drift rule (no fast/slow classification, no trial).
  - **Concurrency safety:** the OpenSSL twins create a key-object slot per call and their public-key table is locked (the liboqs paths already create a fresh object per call); a test runs both stages of the production and classical suites from several goroutines under the race detector in every image build (it finds 12 data races in the previous twins).
- **E13 (ledger and IPFS bytes; nothing is timed):** one block per algorithm, one repetition.
  - **Size model** (`internal/ledger`, no cryptography): Fabric v2.5's protobuf framing of an endorser transaction, the peer's block-file layout (`common/ledger/blkstorage/block_serialization.go`), the orderer's block metadata and block cutting (MaxMessageCount 10, PreferredMaxBytes 512 KB from QChain's `configtx.yaml`), X.509 DER and PEM sizes, and IPFS dag-pb/UnixFS storage (Kubo defaults: CIDv0, 256 KiB chunks). It is **calibrated on a read-only capture of the live QChain ledger and IPFS store** (13 Oct 2026, `internal/ledger/vmcapture`): it reproduces, to the byte, 60 of 60 endorser transactions from their parts, 57 of 57 QChain transactions from their keys and record lengths, all 61 blocks, 71 IPFS objects and 238 certificates. `ledger.Validate` repeats this check at the start of every E13 block and the block fails on any difference.
  - **Ledger block** (each of the 22 signature schemes): QChain's copied code issues a real credential for N = 1, 5, 10, 20 and 64 attributes of 16 bytes (E11's credential; every signature verified), and the records are built as `QChaincode.js` stores them (sorted keys, compact JSON). Every chaincode call is placed in Fabric blocks: `registerHolder`, `bindHolderKeys` for each of the 8 KEMs (the holder's signature key is the scheme measured), `issueCredential`, and the state changes `updateExpiry`, `suspendCredential`, `restoreCredential` and `revokeCredential` (each rewrites the whole record). Unpadded Falcon and ECDSA signatures vary in length: 1,000 signatures are sampled and verified, and every quantity is reported as min, median and max over them. Reads are placed at block height 500,000 (a read version costs 4 bytes there).
  - **Per transaction** the block reports the envelope, the message the orderer counts, the arguments, the stored record, the block it fills alone (`frame_1`, as observed: one transaction per block) and, when transactions arrive faster than the 2 s batch timeout, how many fit in a block (`per_block_max`) and each one's share of it (`share_max`). Ledger bytes per million credentials = `frame_1` or `share_max` × 10⁶, per peer (both peers store the whole ledger).
  - **Scenarios.** `measured`: QChain's network as captured (ECDSA P-256 certificates and signatures, two endorsements); this is the scenario every result reports. `whatif-base64`, `whatif-keyref`, `whatif-both`: hypothetical QChain encodings (signature and keys in base64 instead of hex; the issuer's public key stored as a 64-hex fingerprint instead of in full), to quantify how much of each transaction is hex encoding and repetition. The analysis reads these scenarios only.
  - **IPFS block** (each of the 8 KEMs): real envelopes sealed by QChain's code for N = 1, 5, 10, 20, 64 and values of 16, 256 and 4,096 bytes, each opened with the wallet's code; two per setting, whose lengths must agree. Reported: file bytes, stored bytes (IPFS blocks), number of blocks and bytes on disk (flatfs, 4 KiB file-system blocks).
  - **Differences from real credentials, stated:** E11's credential type (39 characters; real 20–57) and attribute names (`field01`…, 7 characters; real about 10), an expiry date always set (real: mostly empty), and the Issuer field as a placeholder of the captured length and shape (116 characters). Each moves a credential transaction by tens of bytes; the block test checks that the production suite lands within the real credentials' range.
- **E14 (Fabric latency against payload size):** QChain's Fabric network rebuilt on the Mac and timed with QChain-shaped transactions (`scripts/e14-network-mac.sh`, `scripts/e14-run-mac.sh`, `fabric/`).
  - **Network:** a copy of QChain's `qchain-network` (commit `71a1db6`) in Docker inside Colima: two organisations (Government and General), one peer each, one Raft orderer, **one CouchDB shared by both peers** (as live), three Fabric CAs, TLS, channel `mychannel` with the live block settings (BatchTimeout 2 s, MaxMessageCount 10, PreferredMaxBytes 512 KB) and endorsement policy (a majority of the two organisations, so both endorse). Fabric 2.5.15 (the live peer's version), Fabric CA 1.5.15 and CouchDB 3.3.2, pinned by digest. QChain's own enrolment and environment scripts, channel profile, peer CLI configuration and CA configurations are copied byte for byte; the containers keep the live names, environment variables and mount points. The decoded channel configuration was checked against the live one. Differences, outside the measured path: no host ports are published (the client runs in the same Docker network) and the files live in a Docker volume (Colima has no host mounts).
  - **Chaincode** (`kpipayload`, JavaScript like QChain's, the same libraries and versions): what `issueCredential` does on the ledger, without its business fields: the `role=issuer` check, one read of a holder record, an existence check of the new key, one write of a sorted-key JSON record holding the payload as hex (as QChain stores keys and signatures), a short JSON result. No cryptography (QChain's chaincode has none).
  - **Client** (`fabric/client`, Go): as QChain's backend (`offchain/fabric.go`, `chain.go`): fabric-gateway v1.7.0 and gRPC v1.68.0, the `issuer1` identity of the General organisation through `peer0.general.uae.com`, the backend's timeouts, and **a new gRPC connection and Gateway for every transaction** (the backend's `withContract`). Each transaction is timed in phases: connect (gRPC dials lazily, so this is the client set-up), endorse (including the TLS handshake on first use and the endorsement by both organisations), submit (until the orderer accepted it) and commit (until the peer reported the commit status); total = start of the call to the commit status.
  - **Payloads:** the envelope of a committed transaction is calibrated (payload length solved from probe transactions, then refined) to E13's envelope of QChain's `issueCredential` (N = 5 attributes of 16 bytes) for 8 signature schemes: ECDSA P-256 7,141 B, Falcon-512 12,805 B, ML-DSA-44 21,529 B, ML-DSA-65 27,645 B, ML-DSA-87 35,478 B, SLH-DSA-SHA2-128s 38,164 B, SLH-DSA-SHA2-128f 75,093 B, SLH-DSA-SHA2-256f 206,294 B, each within 0.5 %; plus the smallest transaction (a 1-byte payload, about 4.9 KB: three certificates and three signatures). The payload is seeded random hex; transaction keys have a fixed length so that they do not move the sizes.
  - **Modes:** *sequential*: 100 transactions one after the other, each waiting for its commit; alone in the pipeline a transaction waits for the 2 s batch timer. *Burst*: 40 transactions released at once, each with its own connection; blocks close by count, by size or by the timer. Every block of transactions is read back from the peer (`qscc`): transactions per block, envelope bytes, the bytes the orderer counts (payload + signature), the block's bytes and the inferred cut reason (count = 10 transactions; size = one more of the next transaction would pass 512 KB; otherwise timeout). This checks E13's block model (for example 2 transactions per block at 206 KB).
  - **Order and budget:** 9 points x 2 modes x 5 repetitions in a seeded shuffled order, drift-control blocks (20 smallest transactions, sequential) at the start, the middle and the end; 20 warm-up transactions per session and 2 before each block, not recorded. About 2.8 h.
  - **Correctness:** every transaction must commit as VALID, **or** be flagged `MVCC_READ_CONFLICT` (code 11) with its record read back correctly (see the finding below); any other code stops the client. Every transaction is found in its ledger block, the Gateway's commit status must equal the block's flag, and one payload per block (seeded) is read back and compared byte for byte. Read-backs are made after the block, outside the timed region.
  - **Run quality: recorded, not repeated** (as E8, E9 and E13; decision of 21 Oct): latency here is dominated by the 2 s batch timer and the commit path, so blocks are not repeated for host load. The host monitor, the canary (start, every 15 min, end) and the drift-control blocks are kept for the analysis, which says what it excludes. The usual Mac conditions and launcher steps apply (preflight, services paused, Wi-Fi off, no other containers). The network's and the client's containers are not pinned to one CPU and have no memory cap (as production's).
  - **Finding (22 Oct):** because both peers share one CouchDB, (1) every committed write is applied twice to the same document, the second peer meeting a revision conflict and retrying ("Document update conflict ... Retrying update"); and (2) **the peer that validates a block second finds the records the first peer already wrote, so QChain's `_exists` read of a new key (absent at endorsement) now conflicts: it flags the transactions `MVCC_READ_CONFLICT` in its own copy of the ledger and skips their writes. The records are in the state all the same** (read back on both peers), and the two peers' validation flags differ (in the smoke run 7 transactions were flagged only on the General peer, 6 only on the Government peer, none on both). In the first smoke run it hit the largest-payload burst; in the main run (22 Oct, `20261021-2135`) it was a race, not a payload effect: **410 of the 1,800 burst transactions (22.8 %) were flagged, always in whole ledger blocks (41 of the 275 burst blocks, never part of a block); 41 of the 140 blocks of 10 transactions (29 %) but none of the 35 blocks of the 75 KB point and none of the 100 blocks of the 206 KB point; and none of the 4,560 transactions submitted one at a time** (sequential blocks and drift controls, where each block holds one transaction). QChain's backend submits through the General peer and its chaincode has the same `_exists` check, so a credential write could return a spurious conflict error although it took effect when several blocks follow each other closely and the General peer validates second; one issuance at a time was never flagged here. How often it happens on the live network is not known (not measured; the live network is not touched). The first smoke run (`20261021-2108`) stopped on such a transaction. E14 measures the network as deployed (user decisions of 21 and 22 Oct): the client **records** the code of every transaction and counts the conflicts per block (`mvcc_conflicts`), so their rate by payload size and mode is a result.
- **E4 (liboqs build sensitivity):** E1's cells (every operation of every algorithm, the reuse arm, one process per operation), 5 repetitions, measured on an image whose liboqs is built differently, chosen with the launcher's `start --plan e4 --variant ...`. Every variant passes the same liboqs correctness gate in its image build. The baseline is E1 on `b1-prod` (same harness, same machine), and `kpi-batch` refuses a trial or main E4 plan on a `b1-prod` image.
  - **b2-portable:** `OQS_DIST_BUILD=OFF`, `OQS_OPT_TARGET=generic`, and on arm64 liboqs's NEON, AES, SHA-2 and SHA-3 instruction options off. No hand-written vector code; the compiler may still auto-vectorise with the architecture's baseline instructions (NEON on arm64).
  - **b3-native:** `OQS_DIST_BUILD=OFF`, `OQS_OPT_TARGET=auto` (`-mcpu=native` on arm64, `-march=native` on x86-64), no runtime dispatch. What "native" resolved to on the build machine is recorded (`gcc-native-target.txt`); inside the Mac's VM the CPU model is partly hidden, so it is read from that record, not assumed.
  - **b4-noossl** (the VM, later): `b1-prod` with `OQS_USE_OPENSSL=OFF` (liboqs's own SHA-2/SHA-3 and the system random source).
  - **Controls:** the 5 classical algorithms use OpenSSL, which no variant changes. A difference in their cells beyond the run-to-run spread would point to something other than the liboqs build.
  - The fast/slow classification is E1's (frozen by the trial), so every E4 cell gets the same budget rule as its E1 cell even if a variant is slower.
- **E5 (native macOS):** E1's cells, 5 repetitions, run directly on macOS, outside Docker and the VM (`scripts/e5-run-mac.sh`). Compared with E1 in the container on the same Mac.
  - **Toolchain** (`scripts/e5-setup-mac.sh`, everything in one folder, `/Users/Shared/qchain-kpi-native`, nothing installed system-wide): liboqs 0.16.0, same commit, same 25 algorithms and `b1-prod` flags, built with **Apple clang** (the container: gcc 12); OpenSSL **3.0.22**, the container's version, from the official tarball (SHA-256 pinned) with Debian's configure options for arm64 (OpenSSL's macOS target compiles with `-O3`; Debian's package with gcc's `-O2` and hardening flags); Go 1.24.13, the same. liboqs passes the same correctness gate as in the image, OpenSSL its own test suite, and the harness `go vet` and `go test` natively before every native build.
  - **What differs from the container, all at once:** compiler (Apple clang versus gcc 12), C library (macOS libSystem versus glibc 2.36), kernel and scheduler (macOS versus Linux 6.8 in the VM), no virtualisation, no CPU pinning (macOS has none; `GOMAXPROCS = 1`, and `kpi-prim` raises itself to the user-interactive QoS class, which keeps it on the performance cores), no memory limit. E5 therefore measures the **combined** effect of the container platform and toolchain, not any single factor.
  - **Quality:** the same launcher steps as `run-mac.sh` (background services paused, Wi-Fi off, the same host monitor, the native canary references and the core-type check), plus Colima must be stopped. Other CPU is the block's own reading: every CPU's busy time minus the block's process, over exactly the block's interval. The host monitor still decides thermal pressure, power and Low Power Mode, and its other-CPU reading is recorded (`host_max_other_pct`) but not used: every block is a new process, and the monitor (macOS `top`, which excludes the benchmark by process name) cannot attribute a process's CPU in the samples where it starts or ends, so those samples count the block's own core as other activity (first E5 main run, 20 Oct; decisions log). macOS has no steal or quota counters. The drift rule applies, with references from E5's own first attempts. Environment: `TZ=Asia/Dubai` and `GOMAXPROCS=1`, as in the container.
  - Natively only E5 (and its probe pass) may run, and E5 never runs in the container: every other experiment's rules assume the container.
- **Probe pass (trial):** for every algorithm and operation, a cold call plus 5 calls. It freezes the classification and feeds the runtime estimate. The **probe-E2 pass** (add-on trial) does the same for every E2 and E2c arm: a first call plus 5 calls per arm, unbatched.
- **Canary:** SHA3-256 throughput over 15 windows of 100 ms, in pure Go (independent of the liboqs build). It runs at the start of every session, every 10 minutes (5 in the trial) and at the end.

## 5. Quality control (`cmd/kpi-batch/verdict.go`)

Each attempt gets external-cause flags:

| Flag | Rule |
|---|---|
| `steal` | hypervisor steal on the pinned CPU > 2 % of its time (VM) |
| `other` | host other CPU (max over the block's host samples) + container-kernel other CPU × (its CPUs ÷ host CPUs) > 5 % of host capacity |
| `quota` | CPU-quota throttling of the container (`nr_throttled` > 0) |
| `thermal` | Mac: thermal pressure above nominal. Windows: the processor's `% Performance Limit`, or a thermal zone's `% Passive Limit`, below 100. The VM reports none. |
| `battery`, `alert` | on battery (Mac, Windows); alert: Low Power Mode on (Mac); Energy Saver on or a container that is not ours (Windows); a container that is not ours, MySQL, IPFS or the GitHub Actions runner (VM) |
| `canary` | the canary just before or just after the block deviates by more than C_m from the batch's first canary (the Mac) or, on a machine whose thresholds have `canary_ref_rate` (the VM and the Windows laptop), from that fixed reference: the median canary of the machine's trial |
| `core` | Mac: that canary ran at efficiency-core speed. It is compared with the native references measured at the start of every session: the plain run gives the P-core reference, `taskpolicy -b` the E-core reference, and the threshold is their geometric mean. |
| `host_missing` | trial and main: no host-monitor sample covers the block |

**Host monitors** (one row every 5 s; `docs/data-dictionary.md`). On the Mac and Windows the container runs in a Linux VM on the host, so other CPU has two parts: the host's (everything except the VM and the runtime's helpers, from `top` on the Mac and from performance counters on Windows) and the VM kernel's, which the batch reads itself. On the VM the container runs on the machine's own kernel, so the batch's reading is the whole machine: the VM's monitor records no CPU figure (-1, which the rule ignores) and checks for things that must not run (alert). On the VM the launcher and the monitor run on vCPU 0 (the monitor at the lowest priority) and the benchmark on vCPU 1.

**Pending.** An attempt stays pending until the following canary has run and a host sample covers its end.

**Drift** is evaluated against the cell's reference: the median of its 5 first attempts, frozen once all exist. A block drifts when |block median ÷ reference − 1| > max(D_m, 3 × bootstrap SE of the block median ÷ reference).

**Attempt policy:**
- **External flag:** the block is repeated, at most twice (3 attempts in total). Each repeat is inserted at a seeded random later position in the queue. The headline result is the first attempt without an external flag. If all 3 are flagged, the block is "exhausted" and keeps no headline.
- **Drift flag only (main plans):** one diagnostic re-run. The original attempt stays the headline, because re-running until the numbers agree would make the confidence intervals falsely narrow. Trial plans only record drift.
- **Smoke plans** record flags and never repeat.
- **Untimed experiments (E8, E9, E13)** are never flagged or repeated: sizes and memory do not depend on machine noise, and the first attempt is the result.
- **All attempts are kept.** The analysis reports the flagged fraction and a sensitivity analysis over all attempts.

## 6. Calibration (trial run → `calibration.json`)

The formulas are fixed in code (`quality.FormulaText`) and committed before any trial:
- **D_m** = max(3 %, 2 × d95), where d95 is the 95th percentile of |repetition median ÷ median of the cell's repetition medians − 1| over the trial's E1 cells.
- **C_m** = max(3 %, 3 × robust SD of the trial's canary deviations from the batch's first canary). **For a machine whose first canary is too random to be a reference (the VM, decision of 26 Oct)** the calibration (`kpi-batch calibrate --canary-reference median`) stores the median canary of the trial as the fixed reference `canary_ref_rate`, and C_m = max(3 %, 3 × robust SD of the deviations from it). Every later canary of that machine is compared with that one number.
- **E10 margin per parameter set** = max(2 %, 2F), where F = |mean| + the 90 % t half-width of the per-repetition log(issuer median ÷ issuer-again median).
- **Fast/slow classification:** from the probe pass.
- **Batch sizes:** from the probe medians and E0's timer resolution.
- **Add-on trials.** An experiment that adds arms the first trial did not probe gets a short add-on trial on each machine before its first main run: E2's `trial-e2` probes every E2 and E2c arm and derives **only their batch sizes**, with the same rule. Its `calibration.json` is the machine's committed section with those batch sizes added and `batch_calibrated_from.E2` set; nothing else changes. It is reviewed, merged and committed like the first calibration. E2 main runs refuse to start without it.
- **Steal and other-CPU limits** keep the approved starting values (2 % and 5 %).

Each machine's section is merged into `config/thresholds.json` and committed before that machine's first main run, and never changed afterwards.

**Main runs refuse to start** without a calibrated section that covers every cell they need. A batch refuses to resume if the image, harness commit, thresholds, plan or seed changed.

## 7. Runs and resumption

- **Running.** The machine's launcher (`scripts/run-mac.sh` on the Mac, `scripts/run-linux.sh` on the VM, `scripts/run-windows.ps1` on the Windows laptop) runs one batch in one container with its own named volume. The VM's launcher detaches into a tmux session under `systemd-inhibit`; the Windows launcher keeps the laptop awake with `SetThreadExecutionState` for the batch only.
- **Resuming.** The batch's state is saved after every block. It resumes exactly where it stopped:
  - after the launcher's `stop`, Ctrl-C, a reboot or an error;
  - in a new session, with a new E0 and a canary;
  - an interrupted block is discarded and re-run.
- **Host monitor.** Samples are delivered into the volume once a minute with `docker cp`. No process ever runs inside the measurement container besides the batch itself.
- **At the end:**
  1. the launcher copies the folder out;
  2. it verifies `SHA256SUMS`;
  3. it scans every file for the machine's identifying text;
  4. only then does it move the folder into `results/`.

## 8. Extending a finished run

A block can end with no clean attempt: all 3 attempts were flagged for an external cause, usually a short burst of macOS system activity during a long (over a minute) operation. Instead of running a second plan, the launcher's `extend <run-id> --blocks ...` runs just those blocks again **inside the same run folder**:

- **Same rules.** The blocks run under the same plan, thresholds, seed derivation and quality rules. The attempt numbers continue (a4, a5, a6), and each extended block gets a fresh budget of 3 attempts. Only blocks without a headline attempt can be extended.
- **Same code.** The launcher refuses unless everything that produces the numbers is identical to the run's harness commit (checked with git): `cmd/kpi-prim`, `cmd/kpi-env`, `cmd/kpi-canary-native`, `internal/`, `docker/`, the machine's own plan folder (`plans/mac/` on the Mac), `config/`, `go.mod` and `go.sum`. The orchestrator `cmd/kpi-batch` may differ, because it implements the extension itself; its changed files are listed in the run's manifest. Decisions for runs that are not extended are unchanged. The image and harness commit may then differ, because the launcher scripts and docs change between commits; both old and new values are recorded.
- **Recorded, not hidden.** The extension is session 2 in the manifest (`sessions`, `extensions`, `notes`), the state, the journal (`extension_queued`) and `console.txt`. A new E0 and canary are captured (`env/e0-s2.json`, `env/build-info-s2/`). Input sets are regenerated: the same message seeds, new keys.
- **Safety.** Before anything starts, the finished run is copied to `scratch/backup/<run-id>` (it is also in git). The new folder replaces the old one only after its checksums verify and the privacy scan is clean.

## 9. Which experiments run on which machine (decided 21 Oct 2026)

The first plan ran every experiment on all three machines. To finish the measurements in time, the scope per machine is:

| Experiment | Mac (M1) | Windows laptop | University VM |
|---|---|---|---|
| E0 environment and correctness | every run | every run | every run |
| E1 primitive latency | done | all 30 algorithms | all 30 algorithms |
| E2 integration cost | done (full) | without SLH-DSA (18 algorithms) | without SLH-DSA (18 algorithms) |
| E3 integration layers | done | not run | not run |
| E4 build sensitivity | B2, B3 done | B2, B3 | B4 (no OpenSSL) only |
| E5 native OS vs container | done | not run | not run |
| E6 message-size sweep | done | not run | not run |
| E7 rejection timing | done | not run | not run |
| E8 sizes | done | not run | not run |
| E9 memory | done | run | run |
| E10 issuer vs holder | done | run | run |
| E11 lifecycle | done (all 991 cells) | the 16 curated suites (501 cells) | the 16 curated suites (501 cells) |
| E12 concurrency | workers 1, 2, 4 | workers 1, 2, 4, 8 | workers 1, 2 (both vCPUs) |
| E13 ledger and IPFS model | done | not run | not run |
| E14 Fabric latency against payload size | done (`scripts/e14-run-mac.sh`) | not run | never (live QChain network) |

Reasons: sizes (E8, E13) depend on the algorithm, not the processor; E7 is a ratio; E3 and E6 are comparisons whose Mac result answers the question; SLH-DSA dominates the time of E1, E2, E3, E6 and E11 (every slow cell gets at least 60 s), and its speed on each machine still comes from E1 and E11 (curated). The results that depend on the processor stay on both machines. Statistics are unchanged (5 repetitions, the same quality rules); the reduced plans are separate plan files, so every result says what it covered.

**The plans** (25 Oct 2026) are in `plans/vm/` and `plans/windows/` (`plans/README.md`); the experiment code is the same as on the Mac.
- **E2:** `e2-noslh` runs E2 and E2c for the 18 algorithms outside SLH-DSA, after the add-on trial `trial-e2-noslh`, which probes exactly those cells.
- **E11:** an E11 item with `"suites": "curated"` (`internal/plan/e11_curated.go`) keeps every cell of the 16 curated suites at every N and k, the production suite's 256- and 4096-byte value cells and the 13 chained runs: 501 of the 991 cells (16 activations, 90 issuances, 50 retrievals, 98 presentations, 234 verifications, 13 chained). The 490 cells it drops exist only for the all-pairs sweep (N = 10, k = 5). Its add-on trial `trial-e11-curated` probes the same 501 cells.
- **E12:** the Mac's design (4 suites, 2 stages, 30 s windows, 5 repetitions) with W = 1, 2, 4 and 8 on Windows (`e12-windows`) and W = 1 and 2 on the VM (`e12-vm`). The VM's E12 was first planned as one test of at most 15 min; on 25 Oct it became the full set, with the VM's whole run set to finish within 2 days.
- **E4:** the same plan as the Mac's; the liboqs build is the launcher's `--variant`.

## 10. Analysis (after the runs, 10 Nov 2026)

The analysis code is in `analysis/` ([analysis/README.md](../analysis/README.md)); its results are in `results/combined/` (analysis tables, presentation tables and figures) and in each machine's `results/<machine>/figures/` and `tables/`, and are written up in [results-windows.md](results-windows.md) and [results-comparison.md](results-comparison.md). The rules were fixed by plan section 9 before the runs; the details below are the choices made when they were implemented.

- **What enters a result:** the headline attempt of every block (the first attempt without an external flag). Drift re-runs never do. A block without a headline attempt (all three, and later six, attempts flagged) has no headline result.
- **Cell value:** the median over the repetitions of each repetition's median, with a 95 % percentile bootstrap interval resampled over the repetitions (10,000 resamples, seed 20261110). Fast cells use `p50_ns`; tails use the median of each repetition's `p99_ns`.
- **Comparing machines:** the ratio of cell medians with a bootstrap interval (each machine resampled separately), and Cliff's delta over the repetition medians. A ratio between two arms of one block (E2) is paired per repetition. With five repetitions per machine the exact two-sided Mann-Whitney p-value cannot be below 0.008, so a Holm correction over many cells has no power; this is why ratios, intervals and Cliff's delta are reported.
- **Equivalence (E10):** TOST on the paired per-repetition log-ratio of holder to issuer, with the 90 % t-interval inside the machine's margin from `config/thresholds.json`.
- **Sensitivity analysis (pre-declared in plan section 9):** the same cells computed from all non-diagnostic attempts, and from attempts under 8 % combined other CPU. It is the only source of numbers for blocks without a headline attempt (the laptop's E12 cells at four workers); table `T14` compares the headline and all-attempts medians of every cell.
- **Lifecycle (E11):** the sum of the five stage medians at 10 attributes, 5 disclosed, 16-byte values (activation and issuance and verification with the suite, retrieval with the KEM, presentation with the signature scheme); a chained run is reported separately.
- **Figures and presentation tables (12 Nov 2026):** every logarithmic axis says "(log scale)" and is labelled in plain units (10 µs, 1 ms, 1 KB, 2×); bar charts print every value; the machines keep one colour each (Mac blue, laptop green, VM orange) that no other quantity uses; a layout check reports overlapping or cut-off text. The presentation tables round the numbers for reading; the analysis tables keep full precision.
- **Machines are compared only where the experiment ran on both** ([section 9](#9-which-experiments-run-on-which-machine-decided-21-oct-2026)); experiments run on the Mac alone are reported for the Mac.

