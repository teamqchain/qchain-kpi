"""E11: the credential lifecycle per algorithm suite (RQ2), and its scaling (T3b, T3c)."""
import csv
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa

PLANS = {"mac": "e11", "linux": "e11-curated", "windows": "e11-curated"}
PROD = "ml-dsa-44+ml-kem-768"
BASE = "ecdsa-p256+dhkem-p256"


def load(md, mode="headline", limit=None):
    out = {}
    for run in runs(md, plan=PLANS[md], variant="b1-prod"):
        c = cells(attempts(run), "p50_ns", mode, limit)
        out.update({k: v for k, v in c.items() if k[0] == "E11" and k[3] == "stage"})
    return out


def stage_vals(c, suite, n=10, k=5, v=16):
    sig, kem = suite.split("+")
    keys = {
        "activation": (suite, "activation"),
        "issuance": (suite, "issuance-n%d-v%d" % (n, v)),
        "retrieval": (kem, "retrieval-n%d-v%d" % (n, v)),
        "presentation": (sig, "presentation-k%d-v%d" % (k, v)),
        "verification": (suite, "verification-n%d-k%d-v%d" % (n, k, v)),
    }
    out = {}
    for s, (a, o) in keys.items():
        kk = ("E11", a, o, "stage")
        if kk not in c:
            return None
        out[s] = np.array(c[kk]["values"])
    return out


def main():
    os.makedirs(OUT_TABLES, exist_ok=True)
    allc = {MACHINES[m]: load(m) for m in MACHINES}
    # T3b: lifecycle per suite
    rows = []
    for mach, c in allc.items():
        suites = sorted({k[1] for k in c if "+" in k[1]})
        base = stage_vals(c, BASE)
        base_tot = sum(v for v in base.values()) if base else None
        for s in suites:
            sv = stage_vals(c, s)
            if sv is None:
                continue
            total = sum(sv.values())
            med, lo, hi = boot_ci(total)
            row = [mach, s] + [round(float(np.median(sv[x])) / 1000, 2) for x in sv] + [round(med / 1000, 2), round(lo / 1000, 2), round(hi / 1000, 2)]
            row.append(round(float(np.median(total / base_tot)), 2) if base_tot is not None else "")
            rows.append(row)
    path = os.path.join(OUT_TABLES, "T3b_E11_lifecycle_per_suite.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "suite", "activation_us", "issuance_us", "retrieval_us", "presentation_us", "verification_us",
                    "lifecycle_total_us", "total_ci_lo_us", "total_ci_hi_us", "total_vs_ecdsa_p256_dhkem_p256"])
        w.writerows(rows)
    # T3c: scaling of the production suite
    rows2 = []
    for mach, c in allc.items():
        for stage, fmt, a_of in (("issuance", "issuance-n%d-v%d", lambda s: s), ("verification", "verification-n%d-k%d-v%d", lambda s: s)):
            for v in (16, 256, 4096):
                for n in (1, 5, 10, 20, 64):
                    if stage == "issuance":
                        kk = ("E11", PROD, fmt % (n, v), "stage")
                    else:
                        kk = ("E11", PROD, fmt % (n, max(1, n // 2) if n != 5 else 3, v), "stage")
                        if kk not in c:
                            kk = ("E11", PROD, fmt % (n, n, v), "stage")
                    if kk in c:
                        med, lo, hi = boot_ci(c[kk]["values"])
                        rows2.append([mach, stage, n, v, round(med / 1000, 2), round(lo / 1000, 2), round(hi / 1000, 2)])
    path2 = os.path.join(OUT_TABLES, "T3c_E11_scaling_production_suite.csv")
    with open(path2, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "stage", "attributes_n", "value_bytes", "p50_us", "ci_lo_us", "ci_hi_us"])
        w.writerows(rows2)
    return rows, rows2, allc


if __name__ == "__main__":
    rows, rows2, allc = main()
    print("machine suite act iss ret pres ver TOTAL ratio-to-classical")
    for r in rows:
        if r[1] in (PROD, BASE, "ed25519+dhkem-x25519", "slh-dsa-sha2-128f+ml-kem-512", "falcon-512+ml-kem-512", "ml-dsa-65+ml-kem-768", "ml-dsa-87+ml-kem-1024"):
            print(r[0][:3], "%-32s %6.1f %6.1f %6.1f %6.1f %6.1f | TOTAL %7.1f [%7.1f, %7.1f] x%s" % (r[1], *r[2:8], r[8], r[9], r[10]))
    c = allc["win-ryzen"]
    ch = [(k[2], round(np.median(v["values"]) / 1000, 1)) for k, v in c.items() if k[1] == PROD and k[2] == "chained-n10-k5-v16"]
    sv = stage_vals(c, PROD)
    print("laptop chained n10 k5:", ch, "sum of stages:", round(float(sum(np.median(v) for v in sv.values())) / 1000, 1))
