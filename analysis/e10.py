"""E10: are issuer-side and holder-side ML-DSA signing equivalent?  TOST on the paired
per-repetition log-ratio of medians (90 % t interval inside +-margin; margin per machine
and parameter set from config/thresholds.json), plus the A/A floor (issuer vs issuer-again)."""
import csv
import json
import os
import sys

import numpy as np
from scipy import stats

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa


def tost(logr, margin, alpha=0.05):
    n = len(logr)
    m = float(np.mean(logr))
    se = float(np.std(logr, ddof=1) / np.sqrt(n))
    t = stats.t.ppf(1 - alpha, n - 1)
    lo, hi = m - t * se, m + t * se
    return m, lo, hi, (lo > -margin and hi < margin)


def main():
    th = json.load(open(os.path.join(ROOT, "config", "thresholds.json")))["machines"]
    rows_out = []
    for md in MACHINES:
        mach = MACHINES[md]
        for run in runs(md, plan="e10"):
            c = cells(attempts(run))
            for alg in ("ml-dsa-44", "ml-dsa-65", "ml-dsa-87"):
                iss = np.array(c[("E10", alg, "sign", "issuer")]["values"])
                hol = np.array(c[("E10", alg, "sign", "holder")]["values"])
                again = np.array(c[("E10", alg, "sign", "issuer-again")]["values"])
                margin = th[mach]["e10_margin"][alg]
                m, lo, hi, ok = tost(np.log(hol / iss), margin)
                am, alo, ahi, _ = tost(np.log(again / iss), margin)
                rows_out.append([mach, alg, round(float(np.median(iss)) / 1000, 2), round(float(np.median(hol)) / 1000, 2),
                                 round(float(np.exp(m)) - 1, 4), round(float(np.exp(lo)) - 1, 4), round(float(np.exp(hi)) - 1, 4),
                                 margin, ok, round(float(np.exp(am)) - 1, 4), round(float(np.exp(alo)) - 1, 4),
                                 round(float(np.exp(ahi)) - 1, 4), len(iss)])
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T6_E10_issuer_holder_equivalence.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "alg", "issuer_p50_us", "holder_p50_us", "holder_vs_issuer", "tost90_lo", "tost90_hi", "margin",
                    "equivalent", "aa_issuer_again_vs_issuer", "aa_lo", "aa_hi", "reps"])
        w.writerows(rows_out)
    return rows_out


if __name__ == "__main__":
    for r in main():
        print(r[0], r[1], "issuer %.1f us holder %.1f us diff %+.2f%% [%+.2f%%, %+.2f%%] margin %.2f%% -> %s | A/A %+.2f%% [%+.2f%%, %+.2f%%]" % (
            r[2], r[3], 100 * r[4], 100 * r[5], 100 * r[6], 100 * r[7], "EQUIVALENT" if r[8] else "not shown", 100 * r[9], 100 * r[10], 100 * r[11]))
