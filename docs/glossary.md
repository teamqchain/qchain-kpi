# Glossary

The study's own vocabulary, in alphabetical order. Where a term has a precise definition in the code, the file is named.

| Term | Meaning |
|---|---|
| **A/A floor** | The difference the instrument measures between two identical arms (E10's `issuer` and `issuer-again`). It sets the E10 equivalence margin: twice its upper bound, at least 2 %. |
| **Add-on trial** | A short probe run made after a machine's first trial, for an experiment whose cells the first trial did not cover (E2, E6, E7, E11). It adds batch sizes or fast/slow classes to the machine's thresholds and changes nothing else. |
| **Analysis table** | A generated CSV in `results/combined/analysis_tables/` (T0 to T14): every number at full precision, one row per cell. |
| **Arm** | One measured variant inside a block, for example `reuse`, `reinit` and `prod` in E2, or `issuer`, `holder`, `issuer-again` in E10. Arms of one block are interleaved in a seeded random order. |
| **Attempt** | One execution of a block. `a1` is the first; a flagged block is repeated as `a2`, `a3`; an extension continues with `a4` to `a6` and, after a second extension, `a7` to `a9`. |
| **Batch** | One run of a plan by `kpi-batch`: a queue of blocks, canaries, checkpoints and quality verdicts, resumable after an interruption. |
| **Batch size** | Calls per timed sample for operations faster than 100 times the timer's resolution. |
| **Block** | The unit of measurement: one fresh process that measures one experiment, algorithm, operation (or stage) and repetition. Its id is `E1.ml-dsa-44.sign.r3`. |
| **Build variant** | How liboqs was compiled: `b1-prod` (the production build, with run-time CPU dispatch), `b2-portable` (no optimised code), `b3-native` (compiled for the machine's own processor), `b4-noossl` (production without OpenSSL). |
| **Canary** | A fixed SHA3-256 workload run at the start, every 10 minutes and at the end of every batch. Its rate against a reference shows whether the CPU was slower than usual. |
| **Canary reference** | The rate the canary is compared with: the batch's first canary on the Mac; on the VM and the laptop a fixed rate, the median canary of the machine's trial. |
| **Cell** | A block id without the repetition (`E1.ml-dsa-44.sign`); in the analysis also the arm. A cell's value is the median over its repetitions. |
| **Chained run** | E11's whole lifecycle executed in one process, against the sum of the separately measured stages. |
| **Cliff's delta** | An effect size between two machines: +1 when every repetition of one is slower than every repetition of the other. |
| **Combined other CPU** | Host other-CPU plus in-VM other-CPU scaled to the host's CPU count; above the machine's limit (5 %) an attempt is flagged `other`. |
| **Diagnostic re-run** | A re-run of a block flagged only for drift. It never replaces the headline; it is kept beside it. |
| **Drift** | A block whose median deviates from its cell's reference by more than the machine's drift limit with no external cause. |
| **Exhausted block** | A block whose attempts were all flagged, so it has no headline attempt. |
| **Experiment (E0 to E14)** | E0 environment, E1 primitive latency, E2 integration cost, E3 integration layers, E4 build sensitivity, E5 native macOS, E6 message size, E7 rejection timing, E8 sizes, E9 memory, E10 issuer and holder, E11 lifecycle, E12 concurrency, E13 ledger and IPFS model, E14 Fabric latency. |
| **Extension** | Running the blocks of a finished run that have no clean attempt again inside the same run folder (`run-*.sh extend`), with a fresh budget of three attempts. The measuring code must be identical to the run's. |
| **External flag** | `steal`, `other`, `thermal`, `quota`, `battery`, `canary`, `core` or `host_missing`: a cause outside the measurement. A flagged attempt is repeated. |
| **Harness commit** | The repository's last commit outside `results/` (`git log -1 -- . ':!results'`). Each image is labelled with it; a run refuses to start on a different commit. |
| **Headline attempt** | The first attempt of a block without an external flag. Only headline attempts enter the results. |
| **Lifecycle** | E11's sum of five stages: activation, issuance, retrieval, presentation, verification. |
| **Plan** | A JSON file in `plans/<machine>/` that lists the blocks of a batch: `smoke`, `trial`, `trial-e*` (add-on), or a main plan such as `e1` or `e11-curated`. |
| **Presentation table** | A generated table in `results/combined/tables/` or `results/<machine>/tables/`, as CSV (to paste) and Markdown (to read): one or more per experiment, numbers rounded for reading. |
| **Probe** | Five calls of every operation in a trial, used to classify cells as fast or slow and to estimate run times. |
| **Production pattern (`prod`)** | E2's arm that runs QChain's own production functions: keys and outputs as hex strings, the library object initialised for every call. |
| **Repetition (`r1` to `r5`)** | Five independent processes per cell, shuffled through the batch. |
| **Run** | One batch's folder in `results/<machine>/`, named `<UTC yyyymmdd-hhmm>_<machine>_<runtime>_<variant>`. |
| **Sensitivity analysis** | The same cells computed from all attempts, and from attempts under 8 % combined other CPU, to show what the flagging rules change (table `T14`). |
| **Series** | Several plans run one after another by `run-linux-series.sh` or `run-windows-series.ps1`. |
| **Steal** | Time the hypervisor gave the VM's vCPU to someone else; above 2 % an attempt is flagged. |
| **Suite** | A signature scheme and a KEM used together, for example the production suite ML-DSA-44 + ML-KEM-768. The 16 curated suites are listed in `internal/plan/e11_curated.go`. |
| **Thresholds** | Each machine's limits in `config/thresholds.json`, calibrated from its own trial and never changed afterwards. |
| **TOST** | Two one-sided tests: equivalence holds when the 90 % interval of the difference lies inside the margin (E10). |
| **Trial** | A machine's first calibration run: probes, E10 arms, short repetitions of a few operations and canaries; it fixes the machine's thresholds. |
