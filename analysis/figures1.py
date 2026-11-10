"""F1, F2 (primitive latency), F3 (distribution tails), F4 (size against speed), F9 (platform comparison)."""
import gzip
import glob
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from figures_common import *  # noqa

T2 = table("T2_E1_primitive_latency.csv")
SIGS = [r["alg"] for r in T2 if r["machine"] == "mac-m1" and r["op"] == "sign"]
KEMS = [r["alg"] for r in T2 if r["machine"] == "mac-m1" and r["op"] == "encaps"]
KEYS = {(r["machine"], r["alg"], r["op"]): r for r in T2}


def order(algs, op=None):
    """One order for every panel of a figure: family, then the Mac's median of the figure's reference operation."""
    fam = {"ML-DSA": 0, "Falcon": 1, "SLH-DSA": 2, "ML-KEM": 0, "HQC": 1, "classical": 9}
    from e1 import family
    ref = "sign" if "sign" in {r["op"] for r in T2 if r["alg"] == algs[0]} else "encaps"
    return sorted(algs, key=lambda a: (fam[family(a)], float(KEYS[("mac-m1", a, ref)]["p50_ns"])))


def dot_panel(ax, algs, op, title):
    algs = order(algs, op)
    y = np.arange(len(algs))
    for mach in COL:
        xs, lo, hi, p99 = [], [], [], []
        for a in algs:
            r = KEYS[(mach, a, op)]
            xs.append(float(r["p50_ns"]) / 1000)
            p99.append(float(r["p99_ns"]) / 1000)
        off = {"mac-m1": -0.22, "win-ryzen": 0.0, "vm-x86v2": 0.22}[mach]
        ax.hlines(y + off, xs, p99, color=COL[mach], alpha=0.35, linewidth=1.0)
        ax.scatter(xs, y + off, s=14, color=COL[mach], marker=MARK[mach], zorder=3, label=SHORT[mach], linewidths=0)
    ax.set_yticks(y)
    ax.set_yticklabels(algs)
    ax.set_xscale("log")
    ax.invert_yaxis()
    ax.set_title(title)
    ax.set_xlabel("latency per call, microseconds (median; line to p99)")
    ax.grid(axis="x")


def f1_f2():
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 6.2), sharey=True)
    for ax, op in zip(axs, ("keygen", "sign", "verify")):
        dot_panel(ax, SIGS, op, op)
    h, l = axs[0].get_legend_handles_labels()
    fig.legend(h, l, loc="upper right", ncol=3, frameon=False, bbox_to_anchor=(0.99, 0.985))
    fig.suptitle("F1  Signature schemes: latency of every operation on the three machines (E1, 5 repetitions)", x=0.02, ha="left", y=1.0, fontsize=9)
    fig.tight_layout()
    save(fig, "F1_signature_latency")
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 3.4), sharey=True)
    for ax, op in zip(axs, ("keygen", "encaps", "decaps")):
        dot_panel(ax, KEMS, op, op)
    h, l = axs[0].get_legend_handles_labels()
    fig.legend(h, l, loc="upper right", ncol=3, frameon=False, bbox_to_anchor=(0.99, 1.04))
    fig.suptitle("F2  Key-encapsulation mechanisms: latency of every operation (E1)", y=1.02, fontsize=9)
    fig.tight_layout()
    save(fig, "F2_kem_latency")


def raw_samples(machine_dir, plan, variant, alg, op, rep=1, limit=20000):
    """Warm per-call latencies (ns) of the headline attempt of one block; the first (cold) call is dropped."""
    run = runs(machine_dir, plan=plan, variant=variant)[0]
    bid = "E1.%s.%s.r%d" % (alg, op, rep)
    att = [r for r in read_csv(os.path.join(run["dir"], "quality.csv")) if r["id"] == bid and r["headline"] == "true"]
    f = glob.glob(os.path.join(run["dir"], "raw", "E1", alg, "%s-r%d-a%s.csv.gz" % (op, rep, att[0]["attempt"])))[0]
    xs = []
    with gzip.open(f, "rt") as fh:
        col = None
        for line in fh:
            if line.startswith("#"):
                continue
            parts = line.strip().split(",")
            if parts[0] == "arm":
                col = parts.index("ns")
                continue
            if ":" in parts[0]:
                continue
            xs.append(float(parts[col]))
    return np.array(xs[:limit])


def f3():
    cases = [("ml-dsa-44", "sign"), ("falcon-512", "sign"), ("ml-kem-768", "decaps")]
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 3.2))
    for ax, (alg, op) in zip(axs, cases):
        for md in MACHINES:
            xs = raw_samples(md, "e1", "b1-prod", alg, op) / 1000
            xs = np.sort(xs)
            ax.plot(xs, 1 - np.arange(1, len(xs) + 1) / len(xs) + 1e-6, color=COL[MACHINES[md]], label=SHORT[MACHINES[md]], linewidth=1.0)
        ax.set_xscale("log")
        ax.set_yscale("log")
        ax.set_title("%s %s" % (alg, op))
        ax.set_xlabel("latency, microseconds")
        ax.set_ylabel("share of calls slower than x")
        from matplotlib.ticker import ScalarFormatter, NullFormatter
        ax.xaxis.set_major_formatter(ScalarFormatter())
        ax.xaxis.set_minor_formatter(NullFormatter())
        ax.grid(True)
    axs[0].legend(frameon=False, loc="lower left")
    fig.suptitle("F3  Latency tails: complementary ECDF of the individual calls of one repetition (E1)", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F3_latency_tails")


def f4():
    sizes = {(r["alg"], r["item"]): float(r["median_bytes"]) for r in table("T1_E8_sizes.csv")}
    from e1 import family
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 3.9), sharey=True)
    for ax, mach in zip(axs, ("mac-m1", "win-ryzen", "vm-x86v2")):
        pts = []
        for a in SIGS:
            sig, pk = sizes.get((a, "sig")), sizes.get((a, "pk"))
            if sig is None:
                continue
            t = float(KEYS[(mach, a, "sign")]["p50_ns"]) + float(KEYS[(mach, a, "verify")]["p50_ns"])
            pts.append((pk + sig, t / 1000, a, family(a)))
        for x, y, a, fam in pts:
            ax.scatter(x, y, s=16, color=FAMCOL[fam], zorder=3, linewidths=0)
            short = a.replace("slh-dsa-", "").replace("falcon-padded-", "falcon-p")
            if a.startswith(("ml-dsa", "falcon-512", "falcon-1024", "ecdsa", "ed25519", "rsa")) or a in ("slh-dsa-sha2-128f", "slh-dsa-sha2-128s", "slh-dsa-sha2-256f"):
                ax.annotate(short, (x, y), fontsize=5.5, xytext=(3, 2), textcoords="offset points")
        for subset, style, name in ((pts, (0, (4, 2)), "all schemes"), ([p for p in pts if p[3] not in ("classical",)], (0, (1, 1.5)), "post-quantum only")):
            best, line = 1e18, []
            for x, y, a, fam in sorted(subset):
                if y < best:
                    best = y
                    line.append((x, y))
            ax.step(*zip(*line), where="post", color="#777777", linewidth=0.8, linestyle=style, label="Pareto frontier, " + name)
        ax.set_xscale("log")
        ax.set_yscale("log")
        ax.set_xlabel("public key + signature, bytes")
        ax.set_title(SHORT[mach])
        ax.grid(True)
    axs[0].set_ylabel("sign + verify, microseconds")
    h, l = axs[0].get_legend_handles_labels()
    from matplotlib.lines import Line2D
    for fam in ("ML-DSA", "Falcon", "SLH-DSA", "classical"):
        h.append(Line2D([0], [0], marker="o", linestyle="", color=FAMCOL[fam], markersize=4)); l.append(fam)
    fig.legend(h, l, loc="lower center", ncol=6, frameon=False, bbox_to_anchor=(0.5, -0.06))
    fig.suptitle("F4  Size against speed for the signature schemes (sizes from E8, times from E1)", y=1.02, fontsize=9)
    fig.tight_layout()
    save(fig, "F4_size_vs_speed")


def f9():
    R = table("T2b_E1_machine_ratios.csv")
    pairs = [("win-ryzen", "mac-m1"), ("vm-x86v2", "mac-m1"), ("vm-x86v2", "win-ryzen")]
    fams = ["ML-DSA", "Falcon", "SLH-DSA", "ML-KEM", "HQC", "classical"]
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 3.2), sharey=True)
    for ax, (n, d) in zip(axs, pairs):
        for i, fam in enumerate(fams):
            v = [float(r["ratio"]) for r in R if (r["numerator"], r["denominator"], r["family"]) == (n, d, fam)]
            ax.scatter(np.full(len(v), i) + np.linspace(-0.18, 0.18, len(v)), v, s=7, color=FAMCOL[fam], linewidths=0, alpha=0.8)
            ax.hlines(np.median(v), i - 0.3, i + 0.3, color="black", linewidth=1.0)
        ax.axhline(1, color="#999999", linewidth=0.6)
        ax.set_yscale("log")
        ax.set_xticks(range(len(fams)))
        ax.set_xticklabels(fams, rotation=30)
        ax.set_title("%s / %s" % (SHORT[n].split(" (")[0], SHORT[d].split(" (")[0]))
        ax.grid(axis="y")
    axs[0].set_ylabel("ratio of median latencies (E1 cells; bar = family median)")
    fig.suptitle("F9  Platform comparison: how much slower is one machine than another, by algorithm family", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F9_platform_comparison")
    B = table("T7_E4_build_sensitivity.csv")
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 3.0), sharey=True)
    for ax, (mach, var) in zip(axs, (("mac-m1", "b2-portable"), ("win-ryzen", "b2-portable"), ("vm-x86v2", "b4-noossl"))):
        for i, fam in enumerate(fams):
            v = [float(r["ratio_variant_over_b1"]) for r in B if (r["machine"], r["variant"], r["family"]) == (mach, var, fam)]
            ax.scatter(np.full(len(v), i) + np.linspace(-0.18, 0.18, len(v)), v, s=7, color=FAMCOL[fam], linewidths=0, alpha=0.8)
            ax.hlines(np.median(v), i - 0.3, i + 0.3, color="black", linewidth=1.0)
        ax.axhline(1, color="#999999", linewidth=0.6)
        ax.set_yscale("log")
        ax.set_xticks(range(len(fams)))
        ax.set_xticklabels(fams, rotation=30)
        ax.set_title("%s, %s" % (SHORT[mach].split(" (")[0], var))
        ax.grid(axis="y")
    axs[0].set_ylabel("variant / production build (b1)")
    fig.suptitle("F9b  Build sensitivity (E4): portable code against the production build, and the VM without OpenSSL", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F9b_build_sensitivity")


if __name__ == "__main__":
    f1_f2()
    f3()
    f4()
    f9()
    print(sorted(os.listdir(OUT_FIGS)))
