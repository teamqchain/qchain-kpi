"""E2: what the deployed integration pattern adds on top of the library call (RQ3).
Arms of one block are interleaved, so the ratio prod/reuse is paired per repetition."""
import csv
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa
from e1 import family  # noqa

COMPONENTS = ["hex-decode-sk", "hex-decode-pk", "hex-decode-sig", "hex-decode-ct", "init", "clean",
              "hex-encode-pk", "hex-encode-sk", "hex-encode-sig", "hex-encode-ct", "hex-encode-ss"]


def paired(c, k_num, k_den):
    a, b = np.array(c[k_num]["values"]), np.array(c[k_den]["values"])
    n = min(len(a), len(b))
    return a[:n] / b[:n]


def load(md, plan):
    out = {}
    for run in runs(md, plan=plan, variant="b1-prod"):
        c = cells(attempts(run))
        out.update({k: v for k, v in c.items() if k[0] in ("E2", "E2c")})
    return out


def main():
    plans = {"mac": "e2", "linux": "e2-noslh", "windows": "e2-noslh"}
    rows = []
    for md, plan in plans.items():
        mach = MACHINES[md]
        c = load(md, plan)
        ops = sorted({(k[1], k[2]) for k in c})
        for alg, op in ops:
            reuse = c.get(("E2", alg, op, "reuse")) or c.get(("E2c", alg, op, "reuse"))
            key = lambda arm: next((k for k in c if k[1] == alg and k[2] == op and k[3] == arm), None)
            if not reuse or key("prod") is None:
                continue
            kr, kp, ki = key("reuse"), key("prod"), key("reinit")
            pr = paired(c, kp, kr)
            med, lo, hi = boot_ci(pr)
            ri = paired(c, ki, kr) if ki else None
            comp = {a: float(np.median(c[key(a)]["values"])) for a in COMPONENTS if key(a)}
            rows.append([mach, family(alg), alg, op, round(float(np.median(c[kr]["values"])) / 1000, 3),
                         round(float(np.median(c[ki]["values"])) / 1000, 3) if ki else "",
                         round(float(np.median(c[kp]["values"])) / 1000, 3), round(med, 4), round(lo, 4), round(hi, 4),
                         round(float(np.median(ri)), 4) if ri is not None else "",
                         round(sum(comp.values()) / 1000, 3), round(comp.get("init", 0) / 1000, 3),
                         round(sum(v for a, v in comp.items() if a.startswith("hex-")) / 1000, 3)])
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T3_E2_integration_overhead.csv")
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "family", "alg", "op", "reuse_us", "reinit_us", "prod_us", "prod_over_reuse", "ci_lo", "ci_hi",
                    "reinit_over_reuse", "components_sum_us", "init_us", "hex_us"])
        w.writerows(rows)
    return rows


if __name__ == "__main__":
    rows = main()
    by = defaultdict(list)
    for r in rows:
        by[(r[0], r[1])].append(r[7])
    for (m, fam), v in sorted(by.items()):
        print("%-9s %-9s median prod/reuse %.3f  (min %.3f, max %.3f, n=%d)" % (m, fam, np.median(v), min(v), max(v), len(v)))
    print("pqc keygen/sign/verify examples (laptop):")
    for r in rows:
        if r[0] == "win-ryzen" and r[2] in ("ml-dsa-44", "ml-kem-768", "ecdsa-p256", "falcon-512"):
            print("  ", r[2], r[3], "reuse %.1f us prod %.1f us ratio %.3f [%.3f, %.3f]; hex %.2f init %.2f" % (r[4], r[6], r[7], r[8], r[9], r[13], r[12]))
