"""F1, F2 (primitive latency), F3 (distribution tails), F4 (size against speed), F9 and F9b (platform and build comparison)."""
import glob
import gzip
import os
import sys

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from figures_common import *  # noqa: E402,F401,F403
from e1 import family  # noqa: E402

T2 = table("T2_E1_primitive_latency.csv")
SIGS = [r["alg"] for r in T2 if r["machine"] == "mac-m1" and r["op"] == "sign"]
KEMS = [r["alg"] for r in T2 if r["machine"] == "mac-m1" and r["op"] == "encaps"]
KEYS = {(r["machine"], r["alg"], r["op"]): r for r in T2}
FAMORDER = {"ML-DSA": 0, "Falcon": 1, "SLH-DSA": 2, "ML-KEM": 0, "HQC": 1, "classical": 9}
OFFSET = {"mac-m1": -0.24, "win-ryzen": 0.0, "vm-x86v2": 0.24}


def order(algs):
    """One order for every panel and every E1 figure: family, then the Mac's median of the reference operation."""
    ref = "sign" if algs[0] in SIGS else "encaps"
    return sorted(algs, key=lambda a: (FAMORDER[family(a)], float(KEYS[("mac-m1", a, ref)]["p50_ns"]), a))


def decades(values, pad=1.6):
    """Log-axis limits with a margin around the data, so no point sits on the frame."""
    v = [x for x in values if x == x and x > 0]
    return min(v) / pad, max(v) * pad


def dot_panel(ax, algs, op, title):
    algs = order(algs)
    y = np.arange(len(algs))
    allv = []
    for mach in ORDER:
        xs = np.array([float(KEYS[(mach, a, op)]["p50_ns"]) / 1000 for a in algs])
        p99 = np.array([float(KEYS[(mach, a, op)]["p99_ns"]) / 1000 for a in algs])
        allv += list(xs) + list(p99)
        ax.hlines(y + OFFSET[mach], xs, p99, color=COL[mach], alpha=0.35, linewidth=1.0)
        ax.scatter(xs, y + OFFSET[mach], s=14, color=COL[mach], marker=MARK[mach], zorder=3, linewidths=0)
    ax.set_yticks(y)
    ax.set_yticklabels(algs)
    ax.set_ylim(len(algs) - 0.5, -0.5)
    time_axis(ax, "x", "latency per call: median, line to p99")
    ax.set_xlim(*decades(allv))
    ax.set_title(title)
    ax.grid(axis="x")


def f1_f2():
    for name, algs, ops, h, title in (
            ("F1_signature_latency", SIGS, ("keygen", "sign", "verify"), 6.6,
             "F1  Signature schemes: latency of every operation on the three machines (E1, median of 5 repetitions)"),
            ("F2_kem_latency", KEMS, ("keygen", "encaps", "decaps"), 3.6,
             "F2  Key-encapsulation mechanisms: latency of every operation on the three machines (E1, median of 5 repetitions)")):
        fig, axs = new_fig(10.5, h, 1, 3, sharey=True)
        for ax, op in zip(axs, ops):
            dot_panel(ax, algs, op, {"keygen": "key generation", "sign": "signing", "verify": "verification",
                                     "encaps": "encapsulation", "decaps": "decapsulation"}[op])
        hh, ll = machine_handles(kind="dot")
        headline(fig, title, hh, ll)
        save(fig, name)


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


TAIL_CASES = [("ml-dsa-44", "sign"), ("falcon-512", "sign"), ("ml-kem-768", "decaps")]


def f3():
    fig, axs = new_fig(10.5, 3.5, 1, 3)
    for ax, (alg, op) in zip(axs, TAIL_CASES):
        allx = []
        for mach in ORDER:
            xs = np.sort(raw_samples(MACHINE_DIR[mach], "e1", "b1-prod", alg, op) / 1000)
            allx += [xs[0], xs[-1]]
            ax.plot(xs, 1 - np.arange(1, len(xs) + 1) / len(xs) + 1e-6, color=COL[mach], linewidth=1.0)
        time_axis(ax, "x", "latency of one call")
        ax.set_xlim(*decades(allx))
        log_axis(ax, "y", SHARE_TICKS, "share of calls slower than x")
        ax.set_ylim(1e-5, 1.5)
        ax.set_title("%s %s" % (alg, {"sign": "signing", "decaps": "decapsulation"}[op]))
        ax.grid(True)
    hh, ll = machine_handles(kind="line")
    headline(fig, "F3  Latency tails: complementary distribution of the individual calls of one repetition (E1)", hh, ll)
    save(fig, "F3_latency_tails")


def f4():
    sizes = {(r["alg"], r["item"]): float(r["median_bytes"]) for r in table("T1_E8_sizes.csv")}
    from matplotlib.lines import Line2D
    fig, axs = new_fig(10.5, 4.2, 1, 3, sharey=True)
    ally = []
    for ax, mach in zip(axs, ORDER):
        pts = []
        for a in SIGS:
            sig, pk = sizes.get((a, "sig")), sizes.get((a, "pk"))
            if sig is None:
                continue
            t = float(KEYS[(mach, a, "sign")]["p50_ns"]) + float(KEYS[(mach, a, "verify")]["p50_ns"])
            pts.append((pk + sig, t / 1000, a, family(a)))
            ally.append(t / 1000)
        for x, y, a, fam in pts:
            ax.scatter(x, y, s=16, color=FAMCOL[fam], zorder=3, linewidths=0)
            short = a.replace("slh-dsa-", "").replace("falcon-padded-", "falcon-p")
            if a.startswith(("ml-dsa", "falcon-512", "falcon-1024", "ecdsa", "ed25519", "rsa")) or a in ("slh-dsa-sha2-128f", "slh-dsa-sha2-128s", "slh-dsa-sha2-256f"):
                # (dx, dy, alignment) in points: the close neighbours get fixed places so their names never touch
                place = {"ml-dsa-44": (-3, -7, "right"), "ml-dsa-65": (4, -7, "left"), "ml-dsa-87": (4, 1, "left"),
                         "ecdsa-p256": (3, -7, "left"), "ed25519": (3, 4, "left")}
                dx, dy, ha = place.get(a, (-3, 2, "right") if x > 2e4 else (3, 2, "left"))
                ax.annotate(short, (x, y), fontsize=5.5, xytext=(dx, dy), textcoords="offset points", ha=ha)
        for subset, style, name in ((pts, (0, (4, 2)), "all schemes"), ([p for p in pts if p[3] != "classical"], (0, (1, 1.5)), "post-quantum only")):
            best, line = 1e18, []
            for x, y, a, fam in sorted(subset):
                if y < best:
                    best = y
                    line.append((x, y))
            ax.step(*zip(*line), where="post", color="#777777", linewidth=0.8, linestyle=style, label="Pareto frontier, " + name)
        byte_axis(ax, "x", "public key + signature")
        ax.set_xlim(30, 1e5)
        time_axis(ax, "y", "sign + verify" if mach == ORDER[0] else None)
        ax.set_title(SHORT[mach])
        ax.grid(True)
    axs[0].set_ylim(*decades(ally))
    h, l = axs[0].get_legend_handles_labels()
    for fam in ("ML-DSA", "Falcon", "SLH-DSA", "classical"):
        h.append(Line2D([0], [0], marker="o", linestyle="", color=FAMCOL[fam], markersize=4))
        l.append(fam)
    fig.legend(h, l, loc="outside lower center", ncol=6, frameon=False)
    fig.suptitle("F4  Size against speed for the signature schemes (sizes from E8, times from E1)", fontsize=9)
    save(fig, "F4_size_vs_speed")


def ratio_dots(ax, groups, label=None):
    """One column per family: every cell as a grey dot, the family median as a black bar with its value."""
    for i, v in enumerate(groups):
        if not v:
            continue
        ax.scatter(np.full(len(v), i) + np.linspace(-0.18, 0.18, len(v)), v, s=7, color=NEUTRAL, linewidths=0, alpha=0.75, zorder=2)
        med = float(np.median(v))
        ax.hlines(med, i - 0.3, i + 0.3, color="black", linewidth=1.1, zorder=3)
        ax.annotate(fmt_ratio(med), (i + 0.3, med), xytext=(2, 0), textcoords="offset points", fontsize=5.5, va="center", ha="left")
    ax.axhline(1, color="#999999", linewidth=0.6, zorder=1)
    ax.set_xticks(range(len(FAMILIES)))
    ax.set_xticklabels(FAMILIES, rotation=40, ha="right")
    ax.set_xlim(-0.6, len(FAMILIES) - 0.1)
    ratio_axis(ax, "y", label, lim=(0.5, 40))
    ax.grid(axis="y")


def f9():
    R = table("T2b_E1_machine_ratios.csv")
    pairs = [("win-ryzen", "mac-m1"), ("vm-x86v2", "mac-m1"), ("vm-x86v2", "win-ryzen")]
    fig, axs = new_fig(10.5, 3.7, 1, 3, sharey=True)
    for k, (ax, (n, d)) in enumerate(zip(axs, pairs)):
        groups = [[float(r["ratio"]) for r in R if (r["numerator"], r["denominator"], r["family"]) == (n, d, fam)] for fam in FAMILIES]
        ratio_dots(ax, groups, "ratio of median latencies" if k == 0 else None)
        ax.set_title("%s ÷ %s" % (NAME[n], NAME[d]))
    fig.suptitle("F9  Platform comparison: how much slower one machine is than another, by algorithm family (E1; dot = one operation, bar = family median)", fontsize=9)
    save(fig, "F9_platform_comparison")

    B = table("T7_E4_build_sensitivity.csv")
    grid = [[("mac-m1", "b2-portable"), ("win-ryzen", "b2-portable"), ("vm-x86v2", "b4-noossl")],
            [("mac-m1", "b3-native"), ("win-ryzen", "b3-native"), None]]
    vname = {"b2-portable": "portable build (b2)", "b3-native": "native build (b3)", "b4-noossl": "build without OpenSSL (b4)"}
    fig, axs = new_fig(10.5, 6.6, 2, 3, sharey=True)
    for i, row in enumerate(grid):
        for j, cell in enumerate(row):
            ax = axs[i][j]
            if cell is None:
                ax.axis("off")
                ax.text(0.02, 0.5, "b1 = production build: liboqs distribution build with\nruntime CPU dispatch (the reference, ratio 1×).\n\n"
                        "b2 = portable build: generic target, no hand-written\nvector code.\n\n"
                        "b3 = native build: compiled for the machine's own CPU,\nno runtime dispatch.\n\n"
                        "b4 = the production build without OpenSSL: liboqs's own\nSHA-2/SHA-3 and the system random source (VM only).",
                        transform=ax.transAxes, fontsize=7, va="center", ha="left")
                ax._labels_may_leave = True
                continue
            mach, var = cell
            groups = [[float(r["ratio_variant_over_b1"]) for r in B if (r["machine"], r["variant"], r["family"]) == (mach, var, fam)] for fam in FAMILIES]
            ratio_dots(ax, groups, "variant ÷ production build (b1)" if j == 0 else None)
            ax.set_title("%s, %s" % (NAME[mach], vname[var]))
    fig.suptitle("F9b  Build sensitivity (E4): every liboqs build variant against the production build (dot = one operation, bar = family median)", fontsize=9)
    save(fig, "F9b_build_sensitivity")


if __name__ == "__main__":
    f1_f2()
    f3()
    f4()
    f9()
    print("\n".join(PROBLEMS) or "no layout problems")
