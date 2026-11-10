# Analysis

Code that turns the run folders in `results/` into the tables and figures of the study. Written on 10 November 2026, after the last measurement.

```bash
python3 analysis/run_all.py      # about 20 seconds; rewrites results/combined/tables and results/combined/figures
```

It needs only **numpy, scipy and matplotlib** (the system Python 3.9 on the Mac has them; no pandas, no virtual environment) and reads `results/<machine>/<run-id>/` and `config/thresholds.json`. It writes nothing else. It is deterministic: the bootstrap seed is 20261110.

## How a number is made

1. **Headline attempt.** `quality.csv` marks, for every block, the first attempt without an external flag as `headline`. Only those attempts enter the results ([methodology, attempt policy](../docs/methodology.md)). Drift re-runs (`diagnostic`) never do.
2. **Cell.** A cell is an experiment, algorithm, operation and arm (for example `E1.ml-dsa-44.sign`). Its value is the **median over the repetitions** of each repetition's median (`p50_ns` of `blocks.csv`); five repetitions per cell.
3. **Interval.** 95 % percentile bootstrap, 10,000 resamples **over the repetitions** (process-level, as in plan section 9). Ratios between machines resample each side separately; ratios between arms of one block (E2) are paired per repetition.
4. **Effect size.** Cliff's delta over the repetition medians. With five repetitions per machine the exact Mann-Whitney test cannot give a two-sided p below 0.008, so a Holm correction over many cells has no power; ratios, intervals and Cliff's delta are reported instead.
5. **Equivalence (E10).** TOST on the paired per-repetition log-ratio of holder to issuer, 90 % t-interval inside the machine's margin from `config/thresholds.json`.
6. **Sensitivity analysis.** For blocks without a headline attempt and as a general check, the same cell computed from all attempts (`mode="all"`), and from attempts under 8 % combined other CPU (`mode="under"`). `T14` compares headline and all-attempts medians for every cell.

## Files

| File | Produces |
|---|---|
| `common.py` | run discovery, joining `blocks.csv` and `quality.csv`, cells, bootstrap |
| `quality_summary.py` | `T0` run quality of every run |
| `e1.py` | `T2` primitive latency, `T2b` ratios between machines |
| `e2.py` | `T3` integration overhead (production, reinit and reuse arms, components) |
| `e11.py` | `T3b` lifecycle per suite, `T3c` scaling of the production suite |
| `e9.py` | `T4` memory |
| `e10.py` | `T6` issuer and holder equivalence |
| `e4.py` | `T7` build sensitivity |
| `e12.py` | `T12` concurrency, with the sensitivity columns for blocks without a headline |
| `mac_only.py` | `T1` sizes (E8), `T5`, `T5b` ledger and IPFS bytes (E13), `T8` integration layers (E3), `T9` native macOS (E5), `T10` message size (E6), `T11` rejection timing (E7) |
| `e14_sens.py` | `T13` Fabric latency (E14), `T14` sensitivity |
| `figures_common.py`, `figures1.py`, `figures2.py` | `F0` to `F13` (PDF for the paper, PNG for reports); Okabe-Ito colours (colour-blind safe), one colour per machine |
| `run_all.py` | runs everything |

The results are written up in [docs/results-windows.md](../docs/results-windows.md) and [docs/results-comparison.md](../docs/results-comparison.md).

## Checks that were made

- The median recomputed from the raw per-call samples equals the median in `blocks.csv` to the nanosecond (three cells on three machines).
- E1, E2 and E10 agree with each other on the same operation (ML-DSA-44 signing on the laptop: 79.3, 80.0 and 79.5 µs).
- Memory heap figures are identical on all machines; ML-DSA and ML-KEM sizes match FIPS 204 and 203.
- Every figure was opened and read after it was drawn; this found and fixed two labelling defects (shared axis labels over panels sorted differently).

## Not here

- The interactive dashboard of plan section 10.
- Raw-sample analyses beyond F3 (the distribution tails): the per-call files in `results/*/*/raw/` can be read with the same `common.py`.
