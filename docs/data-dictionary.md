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
| `env/e0-s<N>.json` | kpi-env | E0 environment capture at the start of session N (see below). |
| `env/build-info/` | image build | Records made while the image was built: `oqs-build.txt`, `oqs-CMakeCache.txt`, `oqsconfig.h`, `oqs-cmake.txt`, `oqs-ninja.txt`, `oqs-gate.txt` (liboqs correctness gate), `acvp-sha256-check.txt`, `go-test.txt`, `go-version.txt`, `apt-source-*.txt`, `dpkg-*.tsv`. |
| `monitor/host-s<N>-<seq>.csv` (+ `.ok`) | launcher | Host monitor samples of session N, delivered every minute (see below). A chunk is read only once its `.ok` marker exists. |
| `calibration.json` | kpi-batch (trial only) | The machine's thresholds derived from this trial (see `docs/methodology.md`). |
| `estimate.json` | kpi-batch (trial only) | Projected hours and raw-data size of every main plan on this machine. |
| `STOP.done-s<N>` | kpi-batch | Present if session N was stopped with `run.sh stop`. |
| `SHA256SUMS` | kpi-batch | SHA-256 of every file above (`shasum -c` format). Written last. |
| `console.txt` | launcher | Everything kpi-batch printed, over all sessions (`docker logs`). |
| `launcher/` | launcher | Per session N: `session-sN.json` (what was delivered to the batch), `clock-offset-sN.txt`, `native-canary-sN-plain.json` and `native-canary-sN-background.json` (Mac core references), `host-monitor-sN.csv` (every monitor row), `preflight-sN.txt` (preflight problems; `forced` if `--force` was used). |

`console.txt`, `launcher/` and monitor chunks delivered after the batch ended are not in `SHA256SUMS`. `kpi-batch verify` accepts exactly these unlisted files.

## Identifiers

| Name | Example | Meaning |
|---|---|---|
| block id | `E1.ml-dsa-44.sign.r3` | experiment, algorithm, operation, repetition |
| attempt key | `E1.ml-dsa-44.sign.r3.a2` | block id + attempt number (`a1` = the first attempt) |
| cell | `E1.ml-dsa-44.sign` | block id without the repetition |
| arm | `reuse`, `issuer`, `holder`, `issuer-again` | a measured variant inside a block |

- **Operations:**
  - `keygen`, `sign`, `verify` for the 22 signature schemes;
  - `keygen`, `encaps`, `decaps` for the 8 KEMs.
  - E10 blocks use `sign`.
  - Probe blocks use `probe` as the experiment.
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
| `arm` | Arm index (see `arms`). For the very first call of that arm in the process, `cold:<index>` (E1 and probe: no other cryptographic operation ran before it in the process) or `first:<index>` (E10: several arms share the process). |
| `ns` | Duration of one timed sample. When `batch` > 1 this is the total of `batch` calls; divide by `batch` for the per-call time. |
| `bytes` | Output size of the call: signature length (sign), or public plus secret key length (keygen). |

Rows are in execution order. Arms of an E10 block interleave in a seeded random order per round. Warm-up calls are never recorded.

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
