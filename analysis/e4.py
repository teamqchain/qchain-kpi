"""E4: build sensitivity (T7).  Per cell: E4 median / E1 median of the same machine (b1-prod).
Mac and laptop: portable (b2) and native (b3); VM: no OpenSSL (b4).  Classical cells are controls."""
import csv
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa
from e1 import family, machine_cells, ratio_ci, cliffs_delta  # noqa

PLAN = [("mac", ["b2-portable", "b3-native"]), ("windows", ["b2-portable", "b3-native"]), ("linux", ["b4-noossl"])]


def main():
    rows = []
    for md, variants in PLAN:
        mach = MACHINES[md]
        base = machine_cells(md, "e1", "b1-prod")
        for v in variants:
            e4 = machine_cells(md, "e4", v)
            for k, x in sorted(e4.items(), key=lambda kv: (family(kv[0][1]), kv[0][1], kv[0][2])):
                kb = ("E1",) + k[1:]
                if kb not in base:
                    continue
                r, lo, hi = ratio_ci(x["p50"], base[kb]["p50"])
                rows.append([mach, v, family(k[1]), k[1], k[2], round(float(np.median(base[kb]["p50"])) / 1000, 3), round(float(np.median(x["p50"])) / 1000, 3),
                             round(r, 3), round(lo, 3), round(hi, 3), round(cliffs_delta(x["p50"], base[kb]["p50"]), 2)])
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T7_E4_build_sensitivity.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "variant", "family", "alg", "op", "b1_p50_us", "variant_p50_us", "ratio_variant_over_b1", "ci_lo", "ci_hi", "cliffs_delta"])
        w.writerows(rows)
    return rows


if __name__ == "__main__":
    rows = main()
    by = defaultdict(list)
    for r in rows:
        by[(r[0], r[1], r[2])].append(r[7])
    for (m, v, fam), vals in sorted(by.items()):
        print("%-4s %-13s %-9s median variant/b1 %.3f  (min %.3f, max %.3f, n=%d)" % (m[:3], v, fam, np.median(vals), min(vals), max(vals), len(vals)))
