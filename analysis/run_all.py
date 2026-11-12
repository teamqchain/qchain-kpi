"""Regenerate every table and figure of the study.

    python3 analysis/run_all.py

Writes:
  results/combined/analysis_tables/   T0-T14: every number, one row per cell (CSV)
  results/combined/tables/            the presentation tables, every experiment (CSV + Markdown)
  results/combined/figures/           F0-F17 (PDF + PNG)
  results/<machine>/figures/, tables/ one machine's bar charts and tables, and results/<machine>/README.md

Needs only numpy, scipy and matplotlib (the system Python 3.9 has them). Reads results/<machine>/<run>/ and
config/thresholds.json. Deterministic: bootstrap seed 20261110, no dates in the files; two runs give identical files.
Every figure is checked for overlapping or cut-off text; the problems, if any, are listed at the end.
"""
import os
import sys
import time

sys.path.insert(0, os.path.dirname(__file__))

import e1, e2, e4, e9, e10, e11, e12, e14_sens, mac_only, quality_summary  # noqa: E402


def main():
    t = time.time()
    steps = [("T0 run quality", quality_summary.main), ("T2, T2b E1 latency and machine ratios", e1.main), ("T3 E2 integration overhead", e2.main),
             ("T3b, T3c E11 lifecycle", e11.main), ("T4 E9 memory", e9.main), ("T6 E10 equivalence", e10.main), ("T7 E4 build sensitivity", e4.main),
             ("T12 E12 concurrency", e12.main),
             ("T1, T5, T5b, T5c, T8-T11 Mac-only experiments", lambda: (mac_only.e8(), mac_only.e13(), mac_only.e3(), mac_only.e5(), mac_only.e6(), mac_only.e7())),
             ("T13 E14", e14_sens.e14), ("T14 sensitivity", e14_sens.sens)]
    for name, fn in steps:
        fn()
        print("ok  ", name)
    import figures1, figures2, figures3, paper_tables
    from figures_common import PROBLEMS
    figures1.f1_f2(); figures1.f3(); figures1.f4(); figures1.f9()
    for fn in (figures2.f0, figures2.f5, figures2.f6, figures2.f7, figures2.f8, figures2.f10, figures2.f11, figures2.f12, figures2.f13):
        fn()
    figures3.combined()
    print("ok   combined figures")
    figures3.per_machine()
    print("ok   per-machine figures")
    counts = paper_tables.main()
    print("ok   presentation tables: %s" % ", ".join("%d in %s" % (v, os.path.relpath(k, paper_tables.ROOT)) for k, v in counts.items()))
    print("ok   all done in %.0f s; layout problems: %d" % (time.time() - t, len(PROBLEMS)))
    for p in PROBLEMS:
        print("     ", p)


if __name__ == "__main__":
    main()
