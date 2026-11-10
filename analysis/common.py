"""Shared loading code for the KPI analysis (plain numpy + csv; no pandas needed).

A run folder is results/<os>/<run-id>/.  blocks.csv has one row per arm of every
attempt, quality.csv one row per attempt; the headline attempt of a block is
the first one without an external flag (quality.csv, headline == "true").
Statistics follow docs/plan.md section 9: medians over the repetitions with
bootstrap confidence intervals resampled over the repetitions.
"""
import csv
import glob
import json
import os
import re
from collections import defaultdict

import numpy as np

ROOT = os.path.normpath(os.path.join(os.path.dirname(__file__), ".."))
RESULTS = os.path.join(ROOT, "results")
OUT_TABLES = os.path.join(RESULTS, "combined", "tables")
OUT_FIGS = os.path.join(RESULTS, "combined", "figures")

MACHINES = {"mac": "mac-m1", "linux": "vm-x86v2", "windows": "win-ryzen"}
LABEL = {"mac-m1": "Mac (M1)", "vm-x86v2": "VM (x86, no AVX2)", "win-ryzen": "Laptop (Ryzen, AVX2)"}
SEED = 20261110


def read_csv(path):
    with open(path, newline="") as f:
        return list(csv.DictReader(f))


def fnum(x, default=float("nan")):
    try:
        return float(x)
    except (TypeError, ValueError):
        return default


def runs(machine_dir=None, plan=None, variant=None, kind="main"):
    """Run folders of one machine (a results/ sub-folder name), main runs only."""
    out = []
    dirs = [machine_dir] if machine_dir else list(MACHINES)
    for md in dirs:
        for d in sorted(glob.glob(os.path.join(RESULTS, md, "2026*"))):
            mp = os.path.join(d, "manifest.json")
            if not os.path.exists(mp):
                continue
            m = json.load(open(mp))
            if m.get("status") != "done" or m["plan"]["kind"] != kind:
                continue
            v = m.get("build", {}).get("variant") or d.rsplit("_", 1)[-1]
            if plan and m["plan"]["name"] != plan:
                continue
            if variant and v != variant:
                continue
            out.append({"dir": d, "machine": MACHINES[md], "os": md, "plan": m["plan"]["name"],
                        "variant": v, "id": os.path.basename(d), "manifest": m})
    return out


def attempts(run):
    """All attempts of a run joined with their quality verdict.

    Returns a list of dicts: the blocks.csv row of every arm plus the quality
    columns (headline, external_flags, combined_other_pct, ...) of its attempt.
    """
    q = {r["key"]: r for r in read_csv(os.path.join(run["dir"], "quality.csv"))}
    rows = []
    for r in read_csv(os.path.join(run["dir"], "blocks.csv")):
        qq = q.get(r["key"])
        if qq is None:
            continue
        r = dict(r)
        r["headline"] = qq["headline"] == "true"
        r["external_flags"] = qq["external_flags"]
        r["combined_other_pct"] = fnum(qq["combined_other_pct"])
        r["status"] = qq["status"]
        r["run"] = run["id"]
        r["machine"] = run["machine"]
        r["variant"] = run["variant"]
        rows.append(r)
    return rows


def boot_ci(values, stat=np.median, n=10000, seed=SEED):
    """95 % percentile bootstrap interval of stat over the repetitions."""
    v = np.asarray([x for x in values if not np.isnan(x)], dtype=float)
    if len(v) == 0:
        return (float("nan"),) * 3
    if len(v) == 1:
        return (float(v[0]),) * 3
    rng = np.random.default_rng(seed)
    idx = rng.integers(0, len(v), size=(n, len(v)))
    bs = stat(v[idx], axis=1)
    return float(stat(v)), float(np.percentile(bs, 2.5)), float(np.percentile(bs, 97.5))


def cells(rows, col="p50_ns", mode="headline", limit=None):
    """Aggregate attempts into cells keyed by (exp, alg, op, arm).

    mode "headline": the headline attempt of every repetition.
    mode "all":      every non-diagnostic attempt (sensitivity analysis).
    mode "under":    non-diagnostic attempts with combined other CPU under `limit` %.
    Returns {key: {"values": [per-repetition values], "reps": n, "attempts": n}}.
    """
    per = defaultdict(lambda: defaultdict(list))
    for r in rows:
        if r["diagnostic"] == "true":
            continue
        if mode == "headline" and not r["headline"]:
            continue
        if mode == "under" and not (r["combined_other_pct"] < limit or r["headline"]):
            continue
        per[(r["exp"], r["alg"], r["op"], r["arm"])][int(r["rep"])].append(fnum(r[col]))
    out = {}
    for k, reps in per.items():
        vals = [float(np.median(v)) for _, v in sorted(reps.items())]
        out[k] = {"values": vals, "reps": len(vals), "attempts": sum(len(v) for v in reps.values())}
    return out
