"""E12: concurrency scaling (throughput and tail latency against the number of workers).
Where blocks have no headline attempt (laptop, 4 workers) the table also gives the
pre-declared sensitivity analysis: all attempts, and attempts under 8 % combined other CPU."""
import csv
import os
import re
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa

PLANS = {"mac": "e12", "linux": "e12-vm", "windows": "e12-windows"}


def stat(rows, arm, col, mode, limit=None):
    sel = [r for r in rows if r["arm"] == arm]
    c = cells(sel, col, mode, limit)
    out = defaultdict(dict)
    for (exp, alg, op, a), v in c.items():
        m = re.match(r"(.*)-w(\d+)$", op)
        out[(alg, m.group(1), int(m.group(2)))] = v
    return out


def main():
    rows_out = []
    for md, plan in PLANS.items():
        mach = MACHINES[md]
        run = runs(md, plan=plan)[0]
        rows = attempts(run)
        tp = {m: stat(rows, "throughput", "mean_ns", m, 8.0 if m == "under" else None) for m in ("headline", "all", "under")}
        p50 = {m: stat(rows, "latency", "p50_ns", m, 8.0 if m == "under" else None) for m in ("headline", "all", "under")}
        p99 = {m: stat(rows, "latency", "p99_ns", m, 8.0 if m == "under" else None) for m in ("headline", "all", "under")}
        keys = sorted(set().union(*[set(t) for t in tp.values()]))
        for k in keys:
            alg, stage, w = k
            basis = "headline" if k in tp["headline"] and tp["headline"][k]["reps"] >= 1 else "all"
            t = tp[basis][k]["values"]
            base1 = tp["headline"].get((alg, stage, 1)) or tp["all"].get((alg, stage, 1))
            med, lo, hi = boot_ci(t)
            t1 = float(np.median(base1["values"]))
            under = tp["under"].get(k)
            rows_out.append([mach, alg, stage, w, basis, tp[basis][k]["reps"], round(med, 1), round(lo, 1), round(hi, 1),
                             round(med / t1, 3), round(med / t1 / w, 3),
                             round(float(np.median(p50[basis][k]["values"])) / 1000, 1), round(float(np.median(p99[basis][k]["values"])) / 1000, 1),
                             round(float(np.median(tp["all"][k]["values"])), 1) if k in tp["all"] else "",
                             round(float(np.median(under["values"])), 1) if under else ""])
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T12_E12_concurrency.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "suite", "stage", "workers", "basis", "reps", "throughput_per_s", "ci_lo", "ci_hi", "speedup_vs_1",
                    "efficiency", "latency_p50_us", "latency_p99_us", "throughput_all_attempts", "throughput_under8pct"])
        w.writerows(rows_out)
    return rows_out


if __name__ == "__main__":
    rows = main()
    for r in rows:
        if r[1] in ("ml-dsa-44+ml-kem-768", "ecdsa-p256+dhkem-p256", "slh-dsa-sha2-128f+ml-kem-512") and r[2].startswith(("issuance", "verification")):
            print("%-4s %-30s %-24s W=%d %-12s reps %d  %9.1f/s  x%.2f eff %.2f  p50 %.0f us p99 %.0f us | all %s under8 %s" % (r[0][:3], r[1], r[2], r[3], r[4], r[5], r[6], r[9], r[10], r[11], r[12], r[13], r[14]))
