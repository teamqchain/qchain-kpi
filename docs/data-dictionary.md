# Data dictionary

Every run of the harness produces one folder:

```
results/<os>/<run-id>/
```

- `<os>` is `mac`, `windows` or `linux`.
- `<run-id>` is `<UTC yyyymmdd-hhmm>_<machine>_<runtime>_<variant>`, for example `20261010-1830_mac-m1_colima-vz_b1-prod`.

All times are in nanoseconds:
- **Durations** come from Go's monotonic clock (`time.Now`/`time.Since`).
- **Timestamps** (`*_unix_ns`, `utc_ns`) are wall-clock Unix time in UTC. They use the container's clock, except in the host monitor files, which use the host's clock.

The folder holds only allow-listed facts: no host name, user name, path outside the container, IP or MAC address, serial number or AnyDesk ID. The launcher enforces this with a privacy scan of every file before the folder enters `results/`.

## Files

| File | Written by | Content |
|---|---|---|
| `manifest.json` | kpi-batch | What the run is. Covers the plan (name, kind, SHA-256), seed and image ID, and the harness commit. It also records the Go version, the liboqs build record (variant, commit, flags, gcc, cmake) and the apt source of each image stage. Further fields: the thresholds used (values and file hash), the container preflight facts, the allow-listed host facts, the sessions, the status and the final counts. |
| `state.json` | kpi-batch | The batch's resumable state. Holds the queue, the completed attempts, this batch's fast/slow classification from its probe pass, the repeats and diagnostics already queued, and the input sets generated. It also holds the facts the quality rules need. |
| `journal.csv` | kpi-batch | Append-only event log: `utc_ns,event,key,detail`. Events: `session_start`, `e0`, `canary`, `prep`, `block_start`, `block_done`, `block_failed`, `block_interrupted`, `queued`, `checkpoint`, `cover_timeout`, `stopped`, `failed`, `batch_done`. |
| `blocks.jsonl` | kpi-batch | One JSON line per completed attempt. It is the complete `proto.Result` printed by kpi-prim (see `internal/proto`). |
| `blocks.csv` | kpi-batch | The same data as a table, one row per arm (see below). |
| `quality.csv` | kpi-batch | The quality verdict of every attempt. Rewritten at each checkpoint; final at the end (see below). |
| `canary.jsonl`, `canary.csv` | kpi-batch | One row per canary run (see below). |
| `prep.csv` | kpi-batch | One row per generated input set: `utc_ns,session,alg,rep,pool,e10,prep_ns,fingerprint` (`rep` 0 = the probe pass's small set). |
| `raw/<exp>/<alg>/<op>-r<rep>-a<attempt>.csv.gz` | kpi-prim | Every timed call of one attempt (see below). |
| `env/build-info-s<N>/` | image build | Build records of the image used by an extension session N (see methodology section 8). |
| `env/e0-s<N>.json` | kpi-env | E0 environment capture at the start of session N (see below). |
| `env/build-info/` | image build | Records made while the image was built: `oqs-build.txt`, `oqs-CMakeCache.txt`, `oqsconfig.h`, `oqs-cmake.txt`, `oqs-ninja.txt`, `oqs-gate.txt` (liboqs correctness gate), `acvp-sha256-check.txt`, `go-test.txt`, `go-version.txt`, `apt-source-*.txt`, `dpkg-*.tsv`. |
| `monitor/host-s<N>-<seq>.csv` (+ `.ok`) | launcher | Host monitor samples of session N, delivered every minute (see below). A chunk is read only once its `.ok` marker exists. |
| `calibration.json` | kpi-batch (trial only) | The machine's thresholds derived from this trial (see `docs/methodology.md`). |
| `estimate.json` | kpi-batch (trial only) | Projected hours and raw-data size of every main plan on this machine. | Plans this trial cannot estimate (a first trial cannot estimate E2, E6, E7 or E11, which need their own add-on trial) are listed in `not_estimated` with the reason.
| `SHA256SUMS` | kpi-batch | SHA-256 of every file above (`shasum -c` format). Written last. |
| `console.txt` | launcher | Everything kpi-batch printed, over all sessions (`docker logs`). |
| `launcher/` | launcher | Per session N: `session-sN.json` (what was delivered to the batch), `clock-offset-sN.txt`, `native-canary-sN-plain.json` and `native-canary-sN-background.json` (Mac core references), `host-monitor-sN.csv` (every monitor row), `preflight-sN.txt` (preflight problems; `forced` if `--force` was used). |

`console.txt`, `launcher/` and monitor chunks delivered after the batch ended are not in `SHA256SUMS`. `kpi-batch verify` accepts exactly these unlisted files.

## Identifiers

| Name | Example | Meaning |
|---|---|---|
| block id | `E1.ml-dsa-44.sign.r3` | experiment, algorithm, operation, repetition (experiments `E1`, `E10`, `E2`, `E2c`, `E6`, `E7`, `E8`, `E11`, `E13`, `probe`, `probe-E2`, `probe-E6`, `probe-E7`, `probe-E11`) |
| attempt key | `E1.ml-dsa-44.sign.r3.a2` | block id + attempt number (`a1` = the first attempt) |
| cell | `E1.ml-dsa-44.sign` | block id without the repetition |
| arm | `reuse`, `issuer`, `holder`, `issuer-again`, `reinit`, `prod`, `hex-decode-sk`, `hex-decode-pk`, `hex-decode-sig`, `hex-decode-ct`, `init`, `clean`, `hex-encode-pk`, `hex-encode-sk`, `hex-encode-sig`, `hex-encode-ct`, `hex-encode-ss` | a measured variant inside a block (E2 and E2c arms: `docs/methodology.md` section 4 and `internal/proto/e2.go`) |

- **Operations:**
  - `keygen`, `sign`, `verify` for the 22 signature schemes;
  - `keygen`, `encaps`, `decaps` for the 8 KEMs.
  - E10 blocks use `sign`.
  - E2 and E2c blocks use every operation of the algorithm.
  - E8 blocks use `keygen` and `sign` (signatures) or `keygen` and `encaps` (KEMs); their items are `pk`, `sk`, `sig`, `ct`, `ss`.
  - Probe blocks use `probe` (the reuse arm) or `probe-E2` (every E2 and E2c arm) as the experiment.
- **Algorithm ids:** as in `internal/algs` (for example `slh-dsa-shake-256s`, `falcon-padded-1024`, `dhkem-x25519`).

## `raw/...csv.gz`

These are gzip-compressed CSV files.

**Header.** Lines start with `# key=value`:
- block identity: `key`, `id`, `exp`, `alg`, `op`, `rep`, `attempt`, `profile`, `mode`, `seed`, `diagnostic`;
- arm map and calls per sample: `arms` (for example `0:issuer,1:holder,2:issuer-again`) and `batch` (calls per sample, per arm);
- `first_label`;
- inputs: `pool` (P) and `prep_id` (the first 16 hex digits of SHA-256 of the public key); E10 adds `holder_prep_id`;
- `budget`;
- environment: `harness_commit`, `go`, `liboqs`, `openssl`, `gomaxprocs`, `cpus`;
- `start_unix_ns`.

**Columns.** `arm,ns`, plus `bytes` when the output size varies (signatures of unpadded Falcon and ECDSA, and RSA private keys).

| Column | Meaning |
|---|---|
| `arm` | Arm index (see `arms`). For the very first call of that arm in the process, `cold:<index>` (E1 and probe: no other cryptographic operation ran before it in the process) or `first:<index>` (E10, E2, E2c and probe-E2: several arms share the process). |
| `ns` | Duration of one timed sample. When `batch` > 1 this is the total of `batch` calls; divide by `batch` for the per-call time. |
| `bytes` | Output size of the call: signature length (sign), or public plus secret key length (keygen). |

Rows are in execution order. Arms of an E10, E2 or E2c block interleave in a seeded random order per round. A probe-E2 block probes its arms one after the other. Warm-up calls are never recorded.

## `blocks.csv` (one row per arm of an attempt)

| Column | Meaning |
|---|---|
| `key`, `id`, `exp`, `alg`, `op`, `rep`, `attempt`, `diagnostic`, `profile`, `mode`, `seed` | identity of the attempt. `diagnostic=true` marks a drift re-run, which never enters the headline. `mode` is `fast`, `slow` or `probe`. |
| `arm`, `batch`, `first_label`, `first_ns` | The arm, its calls per sample, and the duration of its first call. |
| `n` | Number of timed samples. |
| `p50_ns` … `max_ns`, `mean_ns`, `sd_ns`, `cov` | Per-call statistics over the samples (median, mean, SD, coefficient of variation, p90, p95, p99, p99.9, minimum, maximum). Percentiles use type 7 (linear interpolation). |
| `boot_se_p50_ns` | Bootstrap standard error of the median (200 resamples, fixed seed). It is used by the drift rule. |
| `sum_ns` | Total measured time of the arm. Slow mode samples until this reaches 60 s. |
| `min_bytes`, `max_bytes` | Output size range. |
| `heap_bytes_per_op`, `allocs_per_op` | Go heap bytes and allocations per call, measured after the timed phase (100 calls fast, 2 calls slow). |
| `wall_ns`, `start_unix_ns`, `end_unix_ns` | From the first call to the last sample. |
| `gc_forced`, `gc_automatic` | Untimed forced collections. Collections the runtime started by itself (expected 0; automatic collection is off). |
| `steal_pct` | Hypervisor steal time on the pinned CPU over the block (VM). |
| `vm_other_pct` | Busy CPU time of the container's kernel minus this container's own CPU time, as % of all that kernel's CPUs. |
| `vm_busy_pct` | Busy CPU time of the whole kernel, as % of its CPUs. |
| `nr_throttled`, `throttled_usec` | CPU-quota throttling of the container during the block. |
| `pool`, `prep_id`, `keys_checked` | Input pool size, key fingerprint, and keys fully exercised (keygen). |
| `rows`, `raw_file`, `raw_sha256` | The raw file and its checksum. |

## `quality.csv` (one row per attempt)

| Column | Meaning |
|---|---|
| `status` | `ok`, `flagged`, or `pending` (waiting for host samples or the next canary). |
| `external_flags` | `;`-separated: `steal`, `other`, `quota`, `thermal`, `battery`, `alert`, `canary`, `core`, `host_missing`. |
| `drift`, `drift_dev`, `drift_limit` | `yes`/`no`/`pending`/`n/a`. `drift_dev` is \|block median ÷ reference − 1\| for the worst arm; the limit is max(D_m, 3 × bootstrap SE ÷ reference). |
| `steal_pct`, `vm_other_pct`, `host_max_other_pct`, `combined_other_pct` | Combined = host other % (max over the overlapping host samples) + VM other % × (VM CPUs ÷ host CPUs). |
| `host_samples`, `host_covered`, `host_throttled`, `host_on_battery`, `host_alert` | Host monitor verdict over the block. |
| `canary_before_ratio`, `canary_after_ratio`, `core_before`, `core_after` | The canaries just before and just after the block. Each ratio is the canary's rate ÷ the batch's first canary rate. The core class is Mac only. |
| `headline` | `true` for the attempt the results use: the first attempt without an external flag. Diagnostic attempts are never headline. |
| `action` | `repeat` (an external flag queued a new attempt), `exhausted` (3 attempts, all flagged), `diagnostic` (drift re-run queued), `record` (flag recorded only), or empty. |

## `canary.csv`

| Column | Meaning |
|---|---|
| `seq`, `reason` | Running number. `reason` is `start`, `periodic`, `resume` or `end`. |
| `start_unix_ns`, `end_unix_ns` | When it ran. |
| `median_rate_per_s`, `rates_per_s` | SHA3-256 hashes per second: the median, and every window (`;`-separated). |
| `ratio_to_first` | Rate ÷ the batch's first canary rate. |
| `core_class` | Mac: `P-like` or `E-like` against the session's native references (the threshold is their geometric mean). |
| `windows`, `window_ms`, `digest_check`, `cpus` | Canary parameters, proof of work, and the CPUs it ran on. |

## `monitor/host-*.csv` and `launcher/host-monitor-s<N>.csv`

The columns are `utc_ns,other_pct,throttle,on_ac,alert,detail`, as described in `internal/quality/host.go`.

On the Mac:
- **`other_pct`** is host CPU busy minus the benchmark VM process, the container runtime's helpers and the monitor, as % of the 8 logical CPUs.
- **`throttle`** is 1 if macOS thermal pressure is above nominal.
- **`alert`** is 1 if Low Power Mode is on.
- **`detail`** carries the raw readings: user %, sys %, excluded core %, thermal level and battery %.

A sample describes the 5 s before its timestamp.

## `env/e0-s<N>.json`

The E0 capture contains:
- harness commit, Go version, GOMAXPROCS and the CPU affinity;
- liboqs version, enabled algorithms, the runtime CPU-feature detection, and `oqs-build.txt`;
- OpenSSL and glibc versions;
- kernel release, version and machine;
- CPU fields, flags and caches from `/proc/cpuinfo` and sysfs, plus CPU vulnerability mitigations;
- clock source, `clock_getres`, the effective timer resolution (smallest non-zero step between consecutive `time.Now()` calls) and its overhead;
- cgroup limits;
- memory total and free space in the run volume;
- time zone;
- a `crypto/rand` 32-byte read benchmark;
- the 30 algorithms with their sizes;
- hashes of the build records.

## E8 (sizes)

E8 blocks are not timed, so their files differ:
- **Raw file** `raw/E8/<alg>/<op>-r1-a1.csv.gz`: header lines as usual (including `items`, `budget` and `encoding`), then `item,bytes`, one row per item in production order (for key generation `pk` then `sk` for each key).
- **`blocks.csv`:** one row per item (`arm` = `pk`, `sk`, `sig`, `ct` or `ss`); `mode` is `sizes`; the statistics columns (`p50_ns`, `mean_ns`, `min_ns`, `max_ns`, ...) hold **bytes**, not nanoseconds; `n` is the number of items; `min_bytes`/`max_bytes` are the shortest and longest item; `first_label` is empty and `first_ns` is 0.
- **`quality.csv`:** host and canary facts are recorded, but no flags apply (`external_flags` empty, `drift` = `n/a`), and the first attempt is always the headline.
- **Encoded sizes:** hex characters = 2 × bytes; JSON string = 2 × bytes + 2.

## E11 (lifecycle composites)

- **Block ids** name a cell: `E11.<algorithm field>.<operation field>.r<rep>`.
  - Algorithm field: `<signature>+<KEM>` (activation, issuance, verification, chained), the KEM alone (retrieval) or the signature alone (presentation).
  - Operation field: the stage and the settings it depends on, for example `issuance-n10-v16`, `presentation-k5-v16`, `verification-n64-k32-v16`, `activation`.
  - `internal/cells` parses both fields (`ParseCell`).
- **Arms:** `stage` (the whole stage); verification also has `chain-decode` (decoding the chain's response alone).
- **Raw header:** as usual, plus `prep_ns` (the block's untimed input preparation) and `gc` (Go's default GC, no collections forced).
- **`blocks.jsonl`:** `prep_ns` per attempt; `gc_automatic` counts the collections Go ran by itself (expected above 0 for E11, unlike the micro-benchmarks).
- **Classification:** `classification["<algorithm field>/<operation field>"]` in `config/thresholds.json`, from `trial-e11` (probe median of the whole stage at least 30 ms = slow); `class_calibrated_from.E11` names that trial.

## E13 (ledger and IPFS bytes)

E13 blocks are not timed (like E8: no run-quality flags, never repeated, the first attempt is the result).
- **Block ids:** `E13.<signature scheme>.ledger.r1` and `E13.<KEM>.ipfs.r1`.
- **Raw file** `raw/E13/<alg>/<ledger|ipfs>-r1-a1.csv.gz`: header lines as usual, plus `validation` (what `ledger.Validate` reproduced, with the capture's SHA-256), `fabric_profile` (the measured Fabric constants as JSON), `height` (500000), `signature_samples`, `scenarios`; then `scenario,tx,kem,n,v,quantity,min,p50,max`.
  - `scenario`: `measured`, `projected-ml-dsa-44`, `projected-ml-dsa-65`, `projected-ml-dsa-87` (projection: Fabric on ML-DSA), `whatif-base64`, `whatif-keyref`, `whatif-both` (hypothetical encodings). IPFS rows are `measured`.
  - `tx`: `issueCredential`, `updateExpiry`, `suspendCredential`, `restoreCredential`, `revokeCredential` (per N), `registerHolder` (n = 0), `bindHolderKeys` (one row set per KEM in `kem`, n = 0), `envelope` (IPFS).
  - `n`: attributes; `v`: attribute value bytes (16 for ledger rows).
  - `quantity` (bytes unless noted): `envelope` (the transaction as stored in the block), `message` (payload + client signature: what the orderer's block cutter counts), `args` (all chaincode arguments, the function name included), `record` (the stored state record), `frame_1` (a block holding this transaction alone, as in the block file), `per_block_max` (count: how many such transactions one block holds when they arrive faster than the batch timeout), `share_max` (this transaction's share of such a block, rounded up); IPFS: `file`, `stored` (sum of IPFS blocks), `blocks` (count), `on_disk` (flatfs with 4 KiB file-system blocks).
  - `min`, `p50`, `max`: over the sampled signature lengths (unpadded Falcon, ECDSA); equal when the length is fixed.
- **`blocks.csv`:** one row per arm: ledger blocks `issue-n<N>` (the measured `issueCredential` envelope, bytes over the samples), IPFS blocks `env-n<N>-v<V>` (file bytes); `mode` is `sizes`, statistics columns hold bytes.
- **Derived:** GB per million credentials per peer = bytes × 10⁶ / 10⁹ (`frame_1` at one transaction per block, `share_max` at full blocks).

## E6 (message-size sweep)

- **Block ids:** `E6.<scheme>.<sign|verify>-m<bytes>.r<rep>`, bytes = 64, 1024, 10240, 102400 or 1048576 (`probe-E6` in the add-on trial).
- **Arm:** `reuse`, as in E1; the first call is `cold`.
- **Classification and batching:** `classification["<scheme>/<op>-m<bytes>"]` and `batch["<scheme>/<op>-m<bytes>/reuse"]` in `config/thresholds.json`, from `trial-e6`; `class_calibrated_from.E6` and `batch_calibrated_from.E6` name that trial.
- **Prep:** made with `--e6`: per size a pool of min(P, max(8, 64 MiB / size)) messages, generated from a stored seed in the measuring process, and their signatures (`prep_ns` includes them).

## E7 (rejection timing)

- **Block ids:** `E7.<algorithm>.verify.r<rep>` (signature schemes) or `E7.<algorithm>.decaps.r<rep>` (KEMs); `probe-E7` in the add-on trial.
- **Arms:** `valid`, `badsig`, `wrongmsg` (verify); `valid`, `tampered` (decaps); first calls labelled `first`.
- **Raw file bytes column:** the sample's outcome, not a size: `0` = rejected (verify returned false, or decaps returned a different secret: implicit rejection), `1` = rejected with an error (malformed input), `2` = accepted. For a batched sample it is its last call's outcome; every call is checked. In `blocks.csv`, `min_bytes` and `max_bytes` are the lowest and highest outcome codes.
- **Classification:** the operation's (`<algorithm>/verify` or `<algorithm>/decaps`, from the first trial); batch sizes `batch["<algorithm>/<op>/<arm>"]` from `trial-e7` (`batch_calibrated_from.E7`).

