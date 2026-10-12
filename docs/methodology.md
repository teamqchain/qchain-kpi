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
  - **Batch sizes** for the new arms come from an add-on trial (`plans/trial-e2.json`, see section 6).
- **E8 (sizes; nothing is timed):** for every algorithm, one block for key generation (public and secret key) and one for sign (signature) or encaps (ciphertext and shared secret), one repetition. A block produces up to 1,000 keys or 10,000 outputs (at least 100; it stops after 60 s once the minimum is reached) and records the length of every item, so the real distributions of unpadded Falcon and ECDSA (DER) signatures and of RSA PKCS#8 private keys are captured; every other length is fixed and is confirmed on every item. Every output is checked: signatures are verified, ciphertexts decapsulated, and every 50th key fully exercised. QChain stores keys, signatures, ciphertexts and shared secrets as lowercase hex strings in JSON, so an item of n bytes takes 2n hex characters (2n + 2 bytes as a JSON string); the analysis derives these from the recorded lengths. Credential-level sizes (records, envelopes, payloads) belong to E13 and E11. Because sizes do not depend on machine noise, E8 blocks get no run-quality flags, are never repeated, and their first attempt is the result.
- **Probe pass (trial):** for every algorithm and operation, a cold call plus 5 calls. It freezes the classification and feeds the runtime estimate. The **probe-E2 pass** (add-on trial) does the same for every E2 and E2c arm: a first call plus 5 calls per arm, unbatched.
- **Canary:** SHA3-256 throughput over 15 windows of 100 ms, in pure Go (independent of the liboqs build). It runs at the start of every session, every 10 minutes (5 in the trial) and at the end.

## 5. Quality control (`cmd/kpi-batch/verdict.go`)

Each attempt gets external-cause flags:

| Flag | Rule |
|---|---|
| `steal` | hypervisor steal on the pinned CPU > 2 % of its time (VM) |
| `other` | host other CPU (max over the block's host samples) + container-kernel other CPU × (its CPUs ÷ host CPUs) > 5 % of host capacity |
| `quota` | CPU-quota throttling of the container (`nr_throttled` > 0) |
| `thermal` | Mac: thermal pressure above nominal. Windows: `% Performance Limit` < 100 (phase 1b). |
| `battery`, `alert` | on battery; Low Power Mode on (Mac); a forbidden container (VM, phase 1b) |
| `canary` | the canary just before or just after the block deviates from the batch's first canary by more than C_m |
| `core` | Mac: that canary ran at efficiency-core speed. It is compared with the native references measured at the start of every session: the plain run gives the P-core reference, `taskpolicy -b` the E-core reference, and the threshold is their geometric mean. |
| `host_missing` | trial and main: no host-monitor sample covers the block |

**Pending.** An attempt stays pending until the following canary has run and a host sample covers its end.

**Drift** is evaluated against the cell's reference: the median of its 5 first attempts, frozen once all exist. A block drifts when |block median ÷ reference − 1| > max(D_m, 3 × bootstrap SE of the block median ÷ reference).

**Attempt policy:**
- **External flag:** the block is repeated, at most twice (3 attempts in total). Each repeat is inserted at a seeded random later position in the queue. The headline result is the first attempt without an external flag. If all 3 are flagged, the block is "exhausted" and keeps no headline.
- **Drift flag only (main plans):** one diagnostic re-run. The original attempt stays the headline, because re-running until the numbers agree would make the confidence intervals falsely narrow. Trial plans only record drift.
- **Smoke plans** record flags and never repeat.
- **All attempts are kept.** The analysis reports the flagged fraction and a sensitivity analysis over all attempts.

## 6. Calibration (trial run → `calibration.json`)

The formulas are fixed in code (`quality.FormulaText`) and committed before any trial:
- **D_m** = max(3 %, 2 × d95), where d95 is the 95th percentile of |repetition median ÷ median of the cell's repetition medians − 1| over the trial's E1 cells.
- **C_m** = max(3 %, 3 × robust SD of the trial's canary deviations).
- **E10 margin per parameter set** = max(2 %, 2F), where F = |mean| + the 90 % t half-width of the per-repetition log(issuer median ÷ issuer-again median).
- **Fast/slow classification:** from the probe pass.
- **Batch sizes:** from the probe medians and E0's timer resolution.
- **Add-on trials.** An experiment that adds arms the first trial did not probe gets a short add-on trial on each machine before its first main run: E2's `trial-e2` probes every E2 and E2c arm and derives **only their batch sizes**, with the same rule. Its `calibration.json` is the machine's committed section with those batch sizes added and `batch_calibrated_from.E2` set; nothing else changes. It is reviewed, merged and committed like the first calibration. E2 main runs refuse to start without it.
- **Steal and other-CPU limits** keep the approved starting values (2 % and 5 %).

Each machine's section is merged into `config/thresholds.json` and committed before that machine's first main run, and never changed afterwards.

**Main runs refuse to start** without a calibrated section that covers every cell they need. A batch refuses to resume if the image, harness commit, thresholds, plan or seed changed.

## 7. Runs and resumption

- **Running.** `scripts/run.sh` runs one batch in one container with its own named volume.
- **Resuming.** The batch's state is saved after every block. It resumes exactly where it stopped:
  - after `run.sh stop`, Ctrl-C, a reboot or an error;
  - in a new session, with a new E0 and a canary;
  - an interrupted block is discarded and re-run.
- **Host monitor.** Samples are delivered into the volume once a minute with `docker cp`. No process ever runs inside the measurement container besides the batch itself.
- **At the end:**
  1. the launcher copies the folder out;
  2. it verifies `SHA256SUMS`;
  3. it scans every file for the machine's identifying text;
  4. only then does it move the folder into `results/`.

## 8. Extending a finished run

A block can end with no clean attempt: all 3 attempts were flagged for an external cause, usually a short burst of macOS system activity during a long (over a minute) operation. Instead of running a second plan, `run.sh extend <run-id> --blocks ...` runs just those blocks again **inside the same run folder**:

- **Same rules.** The blocks run under the same plan, thresholds, seed derivation and quality rules. The attempt numbers continue (a4, a5, a6), and each extended block gets a fresh budget of 3 attempts. Only blocks without a headline attempt can be extended.
- **Same code.** The launcher refuses unless everything that produces the numbers is identical to the run's harness commit (checked with git): `cmd/kpi-prim`, `cmd/kpi-env`, `cmd/kpi-canary-native`, `internal/`, `docker/`, `plans/`, `config/`, `go.mod` and `go.sum`. The orchestrator `cmd/kpi-batch` may differ, because it implements the extension itself; its changed files are listed in the run's manifest. Decisions for runs that are not extended are unchanged. The image and harness commit may then differ, because the launcher scripts and docs change between commits; both old and new values are recorded.
- **Recorded, not hidden.** The extension is session 2 in the manifest (`sessions`, `extensions`, `notes`), the state, the journal (`extension_queued`) and `console.txt`. A new E0 and canary are captured (`env/e0-s2.json`, `env/build-info-s2/`). Input sets are regenerated: the same message seeds, new keys.
- **Safety.** Before anything starts, the finished run is copied to `scratch/backup/<run-id>` (it is also in git). The new folder replaces the old one only after its checksums verify and the privacy scan is clean.
