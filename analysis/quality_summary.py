"""Run-quality summary of every main run (table T0): how clean was each measurement?"""
import csv
import os
import re
import sys
from collections import Counter, defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa


def summarise(run):
    q = read_csv(os.path.join(run["dir"], "quality.csv"))
    q = [r for r in q if r["diagnostic"] != "true"]
    blocks = defaultdict(list)
    for r in q:
        blocks[re.sub(r"\.a\d+$", "", r["id"])].append(r)
    flagged = [r for r in q if r["external_flags"]]
    flags = Counter(f for r in flagged for f in r["external_flags"].split(";"))
    exhausted = [k for k, rs in blocks.items() if not any(x["headline"] == "true" for x in rs)]
    longest = max(len(v) for v in blocks.values())
    canary = [fnum(r["ratio_to_first"]) for r in read_csv(os.path.join(run["dir"], "canary.csv"))]
    hm = []
    p = os.path.join(run["dir"], "launcher", "host-monitor-s1.csv")
    if os.path.exists(p):
        hm = read_csv(p)
    temp = [fnum(m.group(1)) for r in hm for m in [re.search(r"temp_c=([-0-9.]+)", r.get("detail", ""))] if m]
    thermal_rows = sum(1 for r in hm if r.get("throttle") not in ("0", "", None))
    return {
        "machine": run["machine"], "plan": run["plan"], "variant": run["variant"], "run": run["id"],
        "blocks": len(blocks), "attempts": len(q), "attempts_per_block_max": longest,
        "flagged_attempts": len(flagged), "flagged_pct": round(100 * len(flagged) / len(q), 1),
        "flag_other": flags.get("other", 0), "flag_canary": flags.get("canary", 0),
        "flag_thermal": flags.get("thermal", 0), "flag_steal": flags.get("steal", 0),
        "blocks_without_headline": len(exhausted),
        "extension_sessions": max(0, len(run["manifest"].get("sessions", [])) - 1),
        "canary_min": round(min(canary), 4) if canary else "", "canary_max": round(max(canary), 4) if canary else "",
        "temp_c_max": max(temp) if temp else "", "host_throttle_rows": thermal_rows if hm else "",
    }


def main():
    rows = [summarise(r) for r in runs() if os.path.exists(os.path.join(r["dir"], "quality.csv"))]  # E14 has its own layout
    os.makedirs(OUT_TABLES, exist_ok=True)
    path = os.path.join(OUT_TABLES, "T0_run_quality.csv")
    with open(path, "w", newline="") as f:
        w = csv.DictWriter(f, fieldnames=list(rows[0].keys()), lineterminator="\n")
        w.writeheader()
        w.writerows(rows)
    return rows


if __name__ == "__main__":
    for r in main():
        print(r["machine"], r["plan"], r["variant"], "blocks", r["blocks"], "attempts", r["attempts"],
              "flagged %", r["flagged_pct"], "no-headline", r["blocks_without_headline"],
              "canary", r["canary_min"], r["canary_max"], "temp", r["temp_c_max"])
