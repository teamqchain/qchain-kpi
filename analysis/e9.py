"""E9: memory (T4).  Quantities per cell: stack and C-heap peak (route c), Go heap and
allocations (routes reuse and prod), process peak RSS."""
import csv
import os
import re
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa
from e1 import family  # noqa

QUANT = ["stack", "heap-peak", "heap-bytes", "heap-allocs", "heap-unfreed", "go-bytes", "go-allocs", "rss-growth", "rss-peak"]


def load(md):
    out = {}
    for run in runs(md, plan="e9", variant="b1-prod"):
        for r in attempts(run):
            if not r["headline"]:
                continue
            m = re.match(r"(.*)-(c|reuse|prod)$", r["op"])
            if not m:
                continue
            out.setdefault((r["alg"], m.group(1), m.group(2), r["arm"]), []).append(fnum(r["p50_ns"]))
    return out


def main():
    data = {MACHINES[m]: load(m) for m in MACHINES}
    rows = []
    for mach, d in data.items():
        for (alg, op, route, q), v in sorted(d.items()):
            if q in QUANT:
                rows.append([mach, family(alg), alg, op, route, q, round(float(np.median(v)), 1), round(float(np.min(v)), 1), round(float(np.max(v)), 1), len(v)])
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T4_E9_memory.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "family", "alg", "op", "route", "quantity", "median", "min", "max", "reps"])
        w.writerows(rows)
    return rows, data


if __name__ == "__main__":
    rows, data = main()
    print("rows", len(rows))
    for alg, op in (("ml-dsa-44", "sign"), ("ml-kem-768", "decaps"), ("falcon-512", "sign"), ("slh-dsa-sha2-128f", "sign"), ("hqc-1", "decaps"), ("ecdsa-p256", "sign")):
        for q in ("stack", "heap-peak", "go-bytes"):
            vals = []
            for mach in data:
                v = data[mach].get((alg, op, "c" if q != "go-bytes" else "prod", q))
                vals.append("%s %s" % (mach[:3], "%.0f" % np.median(v) if v else "-"))
            print("  %-18s %-7s %-10s %s" % (alg, op, q, " | ".join(vals)))
