"""E1: primitive latency (T2) and the cross-machine comparison (T2b)."""
import csv
import os
import sys

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa

SIGS_PQ = ["ml-dsa-44", "ml-dsa-65", "ml-dsa-87", "falcon-512", "falcon-1024", "falcon-padded-512", "falcon-padded-1024"]
CLASSICAL = ["ecdsa-p256", "ed25519", "rsa3072-pss", "dhkem-p256", "dhkem-x25519"]


def family(alg):
    if alg.startswith("ml-dsa"): return "ML-DSA"
    if alg.startswith("falcon"): return "Falcon"
    if alg.startswith("slh-dsa"): return "SLH-DSA"
    if alg.startswith("ml-kem"): return "ML-KEM"
    if alg.startswith("hqc"): return "HQC"
    return "classical"


def cliffs_delta(a, b):
    a, b = np.asarray(a), np.asarray(b)
    gt = sum((x > y) for x in a for y in b)
    lt = sum((x < y) for x in a for y in b)
    return (gt - lt) / (len(a) * len(b))


def machine_cells(machine_dir, plan="e1", variant="b1-prod", mode="headline", limit=None):
    out = {}
    for run in runs(machine_dir, plan=plan, variant=variant):
        rows = attempts(run)
        stats = {c: cells(rows, c, mode, limit) for c in ("p50_ns", "p99_ns", "mean_ns", "cov", "first_ns", "n")}
        for k, v in stats["p50_ns"].items():
            out[k] = {"p50": v["values"], "reps": v["reps"], "attempts": v["attempts"]}
            for c in ("p99_ns", "mean_ns", "cov", "first_ns", "n"):
                out[k][c] = stats[c].get(k, {"values": []})["values"]
    return out


def write_T2(allc):
    path = os.path.join(OUT_TABLES, "T2_E1_primitive_latency.csv")
    os.makedirs(OUT_TABLES, exist_ok=True)
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["machine", "family", "alg", "op", "p50_ns", "p50_ci_lo_ns", "p50_ci_hi_ns", "p99_ns", "mean_ns",
                    "cov", "first_call_ns", "samples_per_block", "reps", "attempts"])
        for mach, cs in allc.items():
            for (exp, alg, op, arm), v in sorted(cs.items(), key=lambda kv: (family(kv[0][1]), kv[0][1], kv[0][2])):
                med, lo, hi = boot_ci(v["p50"])
                w.writerow([mach, family(alg), alg, op, round(med, 1), round(lo, 1), round(hi, 1),
                            round(float(np.median(v["p99_ns"])), 1), round(float(np.median(v["mean_ns"])), 1),
                            round(float(np.median(v["cov"])), 4), round(float(np.median(v["first_ns"])), 1),
                            int(np.median(v["n"])), v["reps"], v["attempts"]])
    return path


def ratio_ci(a, b, n=10000, seed=SEED):
    """Ratio of medians a/b with a bootstrap interval (each side resampled over its repetitions)."""
    a, b = np.asarray(a, float), np.asarray(b, float)
    rng = np.random.default_rng(seed)
    ia = rng.integers(0, len(a), size=(n, len(a)))
    ib = rng.integers(0, len(b), size=(n, len(b)))
    r = np.median(a[ia], axis=1) / np.median(b[ib], axis=1)
    return float(np.median(a) / np.median(b)), float(np.percentile(r, 2.5)), float(np.percentile(r, 97.5))


def write_T2b(allc):
    path = os.path.join(OUT_TABLES, "T2b_E1_machine_ratios.csv")
    pairs = [("win-ryzen", "mac-m1"), ("vm-x86v2", "mac-m1"), ("vm-x86v2", "win-ryzen")]
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(["numerator", "denominator", "family", "alg", "op", "ratio", "ci_lo", "ci_hi", "cliffs_delta", "disjoint"])
        for num, den in pairs:
            for k, v in sorted(allc[num].items(), key=lambda kv: (family(kv[0][1]), kv[0][1], kv[0][2])):
                if k not in allc[den]:
                    continue
                r, lo, hi = ratio_ci(v["p50"], allc[den][k]["p50"])
                d = cliffs_delta(v["p50"], allc[den][k]["p50"])
                w.writerow([num, den, family(k[1]), k[1], k[2], round(r, 3), round(lo, 3), round(hi, 3), round(d, 2), abs(d) == 1.0])
    return path


def main():
    allc = {MACHINES[m]: machine_cells(m) for m in MACHINES}
    return allc, write_T2(allc), write_T2b(allc)


if __name__ == "__main__":
    allc, p1, p2 = main()
    print(p1, p2)
    import collections
    rows = list(csv.DictReader(open(p2)))
    for pair in (("win-ryzen", "mac-m1"), ("vm-x86v2", "mac-m1"), ("vm-x86v2", "win-ryzen")):
        sel = [r for r in rows if (r["numerator"], r["denominator"]) == pair]
        by = collections.defaultdict(list)
        for r in sel: by[(r["family"], r["op"])].append(float(r["ratio"]))
        print(pair, "cells", len(sel), "all disjoint:", sum(r["disjoint"] == "True" for r in sel))
        for fam in ("ML-DSA", "Falcon", "SLH-DSA", "ML-KEM", "HQC", "classical"):
            vals = [x for (f, o), v in by.items() if f == fam for x in v]
            if vals: print("   %-9s median ratio %.2f (min %.2f max %.2f, n=%d)" % (fam, np.median(vals), min(vals), max(vals), len(vals)))
