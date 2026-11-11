"""E14 (T13) and the sensitivity analysis (T14): would the flagged attempts have changed the results?"""
import csv
import glob
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa
from e1 import family  # noqa


def e14():
    """The MAIN E14 run (plan e14, kind main), aggregated per payload point and mode over its repetitions."""
    import json
    d = None
    for cand in sorted(glob.glob(os.path.join(RESULTS, "mac", "*fabric2.5.15"))):
        m = json.load(open(os.path.join(cand, "manifest.json")))
        if m["plan"]["kind"] == "main" and m.get("status") == "done":
            d = cand
    rows_in = read_csv(os.path.join(d, "blocks.csv"))
    groups = defaultdict(list)
    for r in rows_in:
        groups[(r["point"], r["mode"])].append(r)
    rows = []
    for (point, mode), rs in sorted(groups.items()):
        med = lambda c: float(np.median([fnum(r[c]) for r in rs]))
        rows.append([point, mode, len(rs), sum(int(r["n"]) for r in rs), int(med("payload_bytes")), int(med("envelope_mean")),
                     round(med("total_p50_ns") / 1e6, 1), round(med("total_p99_ns") / 1e6, 1), round(med("endorse_p50_ns") / 1e6, 2),
                     round(med("commit_p50_ns") / 1e6, 1), sum(int(r["mvcc_conflicts"]) for r in rs), os.path.basename(d)])
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T13_E14_fabric_latency.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["point", "mode", "blocks", "transactions", "payload_bytes", "envelope_bytes", "total_p50_ms", "total_p99_ms",
                    "endorse_p50_ms", "commit_p50_ms", "mvcc_conflicts", "run"])
        w.writerows(rows)
    return rows


PLAN_SETS = [("mac", "e1"), ("mac", "e2"), ("mac", "e11"), ("mac", "e6"), ("mac", "e3"),
             ("linux", "e1"), ("linux", "e2-noslh"), ("linux", "e11-curated"),
             ("windows", "e1"), ("windows", "e2-noslh"), ("windows", "e11-curated")]


def sens():
    out = []
    for md, plan in PLAN_SETS:
        for run in runs(md, plan=plan, variant="b1-prod"):
            rows = attempts(run)
            h = cells(rows, "p50_ns", "headline")
            a = cells(rows, "p50_ns", "all")
            diffs, nflag = [], 0
            for k, v in h.items():
                if k not in a:
                    continue
                d = float(np.median(a[k]["values"])) / float(np.median(v["values"])) - 1
                diffs.append(abs(d))
                if a[k]["attempts"] > v["reps"]:
                    nflag += 1
            diffs = np.array(diffs)
            out.append([MACHINES[md], plan, len(diffs), nflag, round(100 * float(np.median(diffs)), 3), round(100 * float(np.percentile(diffs, 95)), 2),
                        round(100 * float(diffs.max()), 2), int((diffs > 0.01).sum()), int((diffs > 0.03).sum())])
    path = os.path.join(OUT_TABLES, "T14_sensitivity_headline_vs_all_attempts.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "plan", "cells", "cells_with_extra_attempts", "median_abs_diff_pct", "p95_abs_diff_pct", "max_abs_diff_pct", "cells_over_1pct", "cells_over_3pct"])
        w.writerows(out)
    return out


if __name__ == "__main__":
    for r in e14():
        print(r)
    print("machine plan cells extra-attempt-cells median%% p95%% max%% >1%% >3%%")
    for r in sens():
        print(*r)
