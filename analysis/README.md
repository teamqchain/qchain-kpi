<p align="center">
  <img src="../.github/assets/qchain_logo.jpeg" width="120" alt="QChain logo" />
</p>

<h1 align="center">Analysis</h1>

<p align="center">
  <b>The code that turns every run folder into the tables and figures of the study.</b><br/>
  Written on 10 November 2026, after the last measurement; figures and tables extended on 12 November · <a href="../README.md">back to the main README</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Python-3.9%2B-3776AB" alt="Python" />
  <img src="https://img.shields.io/badge/needs-numpy%20%C2%B7%20scipy%20%C2%B7%20matplotlib-4C72B0" alt="Dependencies" />
  <img src="https://img.shields.io/badge/output-20%20analysis%20%C2%B7%2066%20presentation%20tables%20%C2%B7%2050%20figures-2EA44F" alt="Output" />
  <img src="https://img.shields.io/badge/deterministic-seed%2020261110-555555" alt="Deterministic" />
</p>

---

## Run it

```bash
python3 analysis/run_all.py      # about one minute; rewrites every generated table and figure, then lists any layout problem
```

| | |
|---|---|
| **Needs** | only numpy, scipy and matplotlib (the Mac's system Python 3.9 has them; no pandas, no virtual environment) |
| **Reads** | `results/<machine>/<run-id>/` for `mac`, `linux` and `windows`, and `config/thresholds.json` |
| **Writes** | `results/combined/analysis_tables/` (T0–T14, CSV), `results/combined/tables/` (presentation tables, CSV + Markdown), `results/combined/figures/` (F0–F17, PDF + PNG), and for each machine `results/<machine>/figures/`, `results/<machine>/tables/` and `results/<machine>/README.md`; nothing else |
| **Deterministic** | the bootstrap seed is 20261110; CSV files use `\n` line endings, the PDFs carry no creation date and every sort has a tie-breaker, so two runs give byte-identical files |

```
 results/mac/…   results/linux/…   results/windows/…            config/thresholds.json
        │  blocks.csv · quality.csv · manifest.json · env/ · raw/      │  E10 margins, limits
        └─────────────────────────────┬────────────────────────────────┘
                                      ▼
                  common.py   headline attempts → cells → bootstrap
                                      │
   ┌─────────┬─────────┬──────────┬───┴─────┬───────────┬─────────────┐
 e1.py     e2.py     e11.py    e9/e10.py  e4/e12.py  mac_only.py  e14_sens.py
   └─────────┴─────────┴──────────┴───┬─────┴───────────┴─────────────┘
                                      ▼
                     results/combined/analysis_tables  (T0–T14)
                                      │
         ┌────────────────────────────┼─────────────────────────────┐
  figures1/2.py · figures3.py    paper_tables.py              (per machine)
         ▼                            ▼                              ▼
 results/combined/figures   results/combined/tables   results/<machine>/figures, tables
```

---

## How a number is made

1. **Headline attempt.** `quality.csv` marks, for every block, the first attempt without an external flag as `headline`. Only those attempts enter the results ([methodology, attempt policy](../docs/methodology.md)). Drift re-runs (`diagnostic`) never do.
2. **Cell.** A cell is an experiment, algorithm, operation and arm (for example `E1.ml-dsa-44.sign`). Its value is the **median over the repetitions** of each repetition's median (`p50_ns` of `blocks.csv`); five repetitions per cell.
3. **Interval.** 95 % percentile bootstrap, 10,000 resamples **over the repetitions** (process-level, as in plan section 9). Ratios between machines resample each side separately; ratios between arms of one block (E2) are paired per repetition.
4. **Effect size.** Cliff's delta over the repetition medians. With five repetitions per machine the exact Mann-Whitney test cannot give a two-sided p below 0.008, so a Holm correction over many cells has no power; ratios, intervals and Cliff's delta are reported instead.
5. **Equivalence (E10).** TOST on the paired per-repetition log-ratio of holder to issuer: the 90 % t-interval must lie inside the machine's margin from `config/thresholds.json`.
6. **Sensitivity analysis.** For blocks without a headline attempt, and as a general check, the same cell computed from all attempts (`mode="all"`) and from attempts under 8 % combined other CPU (`mode="under"`). `T14` compares the headline and all-attempts medians of every cell.

---

## Files

| File | Produces |
|---|---|
| `common.py` | run discovery, joining `blocks.csv` and `quality.csv`, cells, bootstrap; the output folders; the machine ids `mac-m1`, `vm-x86v2`, `win-ryzen` and the experiment names |
| `quality_summary.py` | `T0` the quality of every run (E14 included: recorded once, not flagged) |
| `e1.py` | `T2` primitive latency, `T2b` ratios between machines |
| `e2.py` | `T3` integration overhead (production, reinit and reuse arms, components) |
| `e11.py` | `T3b` lifecycle per suite, `T3c` scaling of the production suite |
| `e9.py` | `T4` memory |
| `e10.py` | `T6` issuer and holder equivalence |
| `e4.py` | `T7` build sensitivity |
| `e12.py` | `T12` concurrency, with the sensitivity columns for blocks without a headline |
| `mac_only.py` | `T1` sizes (E8), `T5`, `T5b`, `T5c` ledger and IPFS bytes (E13), `T8` integration layers (E3), `T9` native macOS (E5), `T10` message size (E6), `T11` rejection timing (E7) |
| `e14_sens.py` | `T13` Fabric latency (E14, from the main run `20261021-2135`, selected through its manifest), `T14` sensitivity |
| `figures_common.py` | the figure rules: machine colours and order, family and stage colours, log axes in plain units, value labels, and the layout check |
| `figures1.py`, `figures2.py` | `F0` to `F13` and `F9b` |
| `figures3.py` | the bar charts: `F1a`–`F1c`, `F2a`–`F2c`, `F5a`, `F14`–`F17`, and each machine's own charts in `results/<machine>/figures/` |
| `paper_tables.py` | the presentation tables (CSV + Markdown) in `results/combined/tables/` and `results/<machine>/tables/`, their index and each machine's `README.md` |
| `run_all.py` | runs everything in order and prints the layout problems, if any |

The index of every table and figure, and of where each experiment appears, is [results/combined/README.md](../results/combined/README.md). The results are written up in [results-mac.md](../docs/results-mac.md), [results-vm.md](../docs/results-vm.md), [results-windows.md](../docs/results-windows.md) and [results-comparison.md](../docs/results-comparison.md).

---

## Figure rules

- **Machines** keep one colour everywhere and are always listed in the same order: Mac M1 blue, laptop green, VM orange (Okabe-Ito, colour-blind safe). Algorithm families and lifecycle stages use colours of their own, checked to differ from the machine colours for normal and colour-blind vision, so a colour never means two things; F14 draws each algorithm in its family colour (darker shade: public key, lighter shade: signature or ciphertext).
- **Logarithmic axes** say "(log scale)" in their label, and their ticks are plain units: 1 µs, 10 µs, 1 ms, 1 s; 1 KB, 10 KB; 1×, 2×, 5×.
- **Bar charts** print the value at the end of every bar; the axis is widened until every label fits.
- **Layout check.** After drawing a figure, `save()` measures every text (title, legend, axis labels, tick labels, value labels) and reports any two that overlap, any value label that leaves its panel and anything outside the saved image. `run_all.py` lists them at the end; the current figures have none. Every figure was also opened and looked at.

---

## Checks that were made

- The median recomputed from the raw per-call samples equals the median in `blocks.csv` to the nanosecond (three cells on three machines).
- E1, E2 and E10 agree with each other on the same operation (ML-DSA-44 signing on the laptop: 79.3, 80.0 and 79.5 µs).
- The heap figures of memory are identical on all machines; the ML-DSA and ML-KEM sizes match FIPS 204 and FIPS 203.
- Every figure was opened and read after it was drawn. This found and fixed panels that shared axis labels over different sort orders (F1, F2, F10), F3 reading the byte column instead of the latencies, the F4 frontier and labels, and on 12 November a legend over F2's title, a cut-off axis label in F9, and log axes without a "(log scale)" label.
- Every number in the four results documents was checked against the tables, and the claims that did not hold were corrected (10 and 11 November).
- **11 November:** `T13` had been computed from the E14 smoke run; `e14_sens.py` now selects the main run through its manifest (6,360 transactions) and aggregates per payload point and mode.
- **12 November:** `T0` gained the E14 run, which had been left out; F9b gained the native builds (b3), which were in `T7` but in no figure.

---

## Not here

- LaTeX versions of the presentation tables (to be added when the journal requires them).
