"""Experiments run on the Mac only (docs/methodology.md section 9): E8 sizes (T1), E13 ledger and IPFS bytes (T5),
E3 integration layers (T8), E5 native macOS (T9), E6 message-size sweep (T10), E7 rejection timing (T11)."""
import csv
import glob
import gzip
import os
import re
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa
from e1 import family, machine_cells, ratio_ci  # noqa


def write(name, header, rows):
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, name)
    with open(path, "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(header)
        w.writerows(rows)
    return path


def e8():
    run = runs("mac", plan="e8")[0]
    rows = []
    for r in attempts(run):
        if not r["headline"]:
            continue
        rows.append([family(r["alg"]), r["alg"], r["op"], r["arm"], int(float(r["min_bytes"])), int(float(r["p50_ns"])), int(float(r["max_bytes"])), int(float(r["n"]))])
    rows.sort()
    write("T1_E8_sizes.csv", ["family", "alg", "op", "item", "min_bytes", "median_bytes", "max_bytes", "items"], rows)
    return rows


def e13():
    run = runs("mac", plan="e13")[0]
    rows, every = [], []
    for f in sorted(glob.glob(os.path.join(run["dir"], "raw", "E13", "*", "ledger-r1-a1.csv.gz"))):
        alg = f.split("/")[-2]
        lines = gzip.open(f, "rt").read().splitlines()
        i = next(k for k, l in enumerate(lines) if l.startswith("scenario,tx"))
        for l in lines[i + 1:]:
            s, tx, kem, n, v, q, lo, p50, hi = l.split(",")
            if s.startswith("projected"):
                continue  # not part of the study: the analysis reads the measured and what-if scenarios only
            if tx == "issueCredential" and v == "16" and q in ("envelope", "frame_1", "per_block_max", "share_max"):
                rows.append([alg, s, int(n), q, int(lo), int(float(p50)), int(hi)])
            if s == "measured" and v == "16" and q in ("envelope", "record", "args", "message", "frame_1"):
                every.append([alg, tx, kem, int(n), q, int(lo), int(float(p50)), int(hi)])
    write("T5_E13_ledger_issue_credential.csv", ["sig_scheme", "scenario", "attributes_n", "quantity", "min", "median", "max"], rows)
    write("T5c_E13_ledger_every_transaction.csv", ["sig_scheme", "tx", "kem", "attributes_n", "quantity", "min", "median", "max"], every)
    ipfs = []
    for r in attempts(run):
        if r["op"] == "ipfs" and r["headline"] and r["arm"].startswith("env-"):
            m = re.match(r"env-n(\d+)-v(\d+)", r["arm"])
            ipfs.append([r["alg"], int(m.group(1)), int(m.group(2)), int(float(r["p50_ns"]))])
    ipfs.sort()
    write("T5b_E13_ipfs_envelope_bytes.csv", ["kem", "attributes_n", "value_bytes", "envelope_bytes"], ipfs)
    return rows, ipfs


def e3():
    run = runs("mac", plan="e3")[0]
    c = cells(attempts(run))
    keys = sorted({(k[1], k[2]) for k in c if k[3] == "l1"})
    rows = []
    for alg, op in keys:
        v = {a: c.get(("E3", alg, op, a)) for a in ("l0", "l0g", "l1", "l2")}
        if not (v["l0"] and v["l0g"] and v["l1"]):
            continue
        m = {a: float(np.median(x["values"])) for a, x in v.items() if x}
        rows.append([family(alg), alg, op, round(m["l0"] / 1000, 3), round(m["l0g"] / 1000, 3), round(m["l1"] / 1000, 3),
                     round(m["l2"] / 1000, 3) if "l2" in m else "", round(m["l0g"] - m["l0"], 1), round(m["l1"] - m["l0g"], 1),
                     round(m["l1"] / m["l0"], 4), round(m["l2"] / m["l1"], 3) if "l2" in m else ""])
    write("T8_E3_integration_layers.csv", ["family", "alg", "op", "l0_c_us", "l0g_go_to_c_us", "l1_binding_us", "l2_go_stdlib_us",
                                           "boundary_ns", "binding_ns", "l1_over_l0", "l2_over_l1"], rows)
    null = {k[3]: float(np.median(v["values"])) for k, v in c.items() if k[1] == "none"}
    return rows, null


def e5():
    base = machine_cells("mac", "e1", "b1-prod")
    nat = machine_cells("mac", "e5", "b1-prod")
    rows = []
    for k, x in sorted(nat.items(), key=lambda kv: (family(kv[0][1]), kv[0][1], kv[0][2])):
        kb = ("E1",) + k[1:]
        if kb in base:
            r, lo, hi = ratio_ci(x["p50"], base[kb]["p50"])
            rows.append([family(k[1]), k[1], k[2], round(float(np.median(base[kb]["p50"])) / 1000, 3), round(float(np.median(x["p50"])) / 1000, 3), round(r, 3), round(lo, 3), round(hi, 3)])
    write("T9_E5_native_vs_container.csv", ["family", "alg", "op", "container_p50_us", "native_p50_us", "native_over_container", "ci_lo", "ci_hi"], rows)
    return rows


def e6():
    run = runs("mac", plan="e6")[0]
    c = cells(attempts(run))
    rows = []
    for (exp, alg, op, arm), v in sorted(c.items(), key=lambda kv: (family(kv[0][1]), kv[0][1], kv[0][2])):
        m = re.match(r"(sign|verify)-m(\d+)", op)
        if not m:
            continue
        med, lo, hi = boot_ci(v["values"])
        size = int(m.group(2))
        rows.append([family(alg), alg, m.group(1), size, round(med / 1000, 3), round(lo / 1000, 3), round(hi / 1000, 3), round(size / (med / 1e9) / 1e6, 2)])
    write("T10_E6_message_size.csv", ["family", "alg", "op", "message_bytes", "p50_us", "ci_lo_us", "ci_hi_us", "throughput_MB_s"], rows)
    return rows


def e7():
    run = runs("mac", plan="e7")[0]
    c = cells(attempts(run))
    rows = []
    algs = sorted({(k[1], k[2]) for k in c})
    for alg, op in algs:
        a = {k[3]: float(np.median(v["values"])) for k, v in c.items() if k[1] == alg and k[2] == op}
        base = a.get("valid")
        if not base:
            continue
        rows.append([family(alg), alg, op, round(base / 1000, 3)] + [round(a[x] / base, 4) if x in a else "" for x in ("badsig", "wrongmsg", "tampered")])
    write("T11_E7_rejection_timing.csv", ["family", "alg", "op", "valid_p50_us", "badsig_over_valid", "wrongmsg_over_valid", "tampered_over_valid"], rows)
    return rows


if __name__ == "__main__":
    r8 = e8()
    print("E8 rows", len(r8))
    for r in r8:
        if r[1] in ("ml-dsa-44", "ml-kem-768", "falcon-512", "slh-dsa-sha2-128f", "ecdsa-p256", "hqc-1") and r[2] in ("keygen", "sign", "encaps"):
            print("  ", r[1], r[3], r[4], r[5], r[6])
    r13, ip = e13()
    print("E13 rows", len(r13))
    for r in r13:
        if r[0] in ("ml-dsa-44", "falcon-512", "slh-dsa-sha2-128f", "ecdsa-p256") and r[2] == 10 and r[1] == "measured" and r[3] in ("envelope", "per_block_max"):
            print("  ", r)
    r3, null = e3()
    print("E3 rows", len(r3), "null", {k: round(v, 1) for k, v in null.items()})
    for r in r3:
        if r[1] in ("ml-dsa-44", "ml-kem-768", "ecdsa-p256") :
            print("  ", r)
    r5 = e5()
    print("E5 rows", len(r5), "median native/container by family:")
    by = defaultdict(list)
    for r in r5: by[r[0]].append(r[5])
    for k, v in by.items(): print("   ", k, round(float(np.median(v)), 3), round(min(v), 3), round(max(v), 3))
    r6 = e6()
    print("E6 rows", len(r6))
    for r in r6:
        if r[1] == "ml-dsa-44" or (r[1] == "ed25519" and r[2] == "sign"):
            print("  ", r)
    r7 = e7()
    print("E7 rows", len(r7))
    for r in r7:
        if r[1] in ("ml-dsa-44", "falcon-512", "ml-kem-768", "ecdsa-p256", "hqc-1"):
            print("  ", r)
