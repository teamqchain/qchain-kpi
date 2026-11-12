"""Bar-chart views of the results, in the style of the project poster: horizontal bars, a log axis in plain units
(1 µs, 10 µs, 1 ms ...) and the value at the end of every bar.

Combined (results/combined/figures):
  F1a-F1c  signature schemes, key generation / signing / verification, three machines
  F2a-F2c  key-encapsulation mechanisms, key generation / encapsulation / decapsulation, three machines
  F5a      lifecycle cost of the 16 curated suites, three machines
  F14      key, signature and ciphertext sizes (E8)
  F15      issuer-side against holder-side signing, equivalence test (E10)
  F16      native macOS against the container (E5)
  F17      Fabric latency against payload size, and MVCC conflicts (E14)
Per machine (results/<machine>/figures): F1a-F1c, F2a-F2c and F5a for that machine alone, and F5b, its lifecycle
split by stage.
"""
import os
import sys

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from figures_common import *  # noqa: E402,F401,F403
from e1 import family  # noqa: E402
from figures1 import KEMS, KEYS, SIGS, order  # noqa: E402
from figures2 import stage_share  # noqa: E402
from matplotlib.patches import Patch, Rectangle  # noqa: E402

OPNAME = {"keygen": "key generation", "sign": "signing", "verify": "verification", "encaps": "encapsulation", "decaps": "decapsulation"}
PRODUCTION = {"ml-dsa-44", "ml-kem-768", "ml-dsa-44+ml-kem-768"}  # what QChain deploys: bold in every chart


def bold_production(ax):
    for t in ax.get_yticklabels():
        if t.get_text() in PRODUCTION:
            t.set_fontweight("bold")


def family_lines(ax, cats, fam_of):
    """Thin lines between algorithm families."""
    for i in range(1, len(cats)):
        if fam_of(cats[i]) != fam_of(cats[i - 1]):
            ax.axhline(i - 0.5, color="#BBBBBB", linewidth=0.5, zorder=0)


def bars(fig, ax, cats, series, unit="us", fmt=fmt_us, label="median latency per call", fam_of=None, fontsize=5.5, log=True):
    """Horizontal bars: one row per category, one bar per series (machine) in each row, values printed at the bar ends.
    series: [(name, colour, [value per category])]."""
    n = len(series)
    h = 0.8 / n
    y = np.arange(len(cats))
    texts, allv = [], []
    for k, (_, colour, vals) in enumerate(series):
        pos = y - 0.4 + h * (k + 0.5)
        ax.barh(pos, vals, height=h * 0.92, color=colour, linewidth=0)
        texts += label_bars(ax, pos, vals, [fmt(v) for v in vals], fontsize=fontsize)
        allv += [v for v in vals if v == v]
    ax.set_yticks(y)
    ax.set_yticklabels(cats)
    ax.set_ylim(len(cats) - 0.5, -0.5)
    if log:
        if unit in ("us", "ms"):
            time_axis(ax, "x", label, unit=unit)
        else:
            byte_axis(ax, "x", label)
        ax.set_xlim(min(allv) / 3, max(allv) * 2)
    else:
        ax.set_xlabel(label)
        ax.set_xlim(0, max(allv) * 1.1)
    ax.grid(axis="x")
    if fam_of:
        family_lines(ax, cats, fam_of)
    bold_production(ax)
    fit_labels(fig, ax, texts)


def e1_bars(algs, op, machines, name, title, outdir=None):
    cats = order(algs)
    per_bar = 0.13 if len(machines) == 3 else 0.24
    fig, ax = new_fig(7.4, 1.2 + per_bar * len(cats) * len(machines) + 0.12 * len(cats))
    series = [(m, COL[m], [float(KEYS[(m, a, op)]["p50_ns"]) / 1000 for a in cats]) for m in machines]
    bars(fig, ax, cats, series, fam_of=family, fontsize=5.2 if len(machines) == 3 else 5.8)
    hh, ll = machine_handles(machines)
    headline(fig, title, hh, ll)
    save(fig, name, outdir)


def lifecycle_rows(machine):
    T = table("T3b_E11_lifecycle_per_suite.csv")
    return {r["suite"]: r for r in T if r["machine"] == machine}


def curated_suites():
    lap = lifecycle_rows("win-ryzen")
    return sorted(lap, key=lambda s: (float(lap[s]["lifecycle_total_us"]), s))


def f5a(machines, name, title, outdir=None):
    suites = curated_suites()
    per_bar = 0.13 if len(machines) == 3 else 0.24
    fig, ax = new_fig(7.4, 1.2 + per_bar * len(suites) * len(machines) + 0.12 * len(suites))
    series = []
    for m in machines:
        rows = lifecycle_rows(m)
        series.append((m, COL[m], [float(rows[s]["lifecycle_total_us"]) for s in suites]))
    bars(fig, ax, suites, series, label="lifecycle cost (sum of the five stages, median)", fontsize=5.2 if len(machines) == 3 else 5.8)
    hh, ll = machine_handles(machines)
    headline(fig, title, hh, ll)
    save(fig, name, outdir)


def f5b(machine, outdir):
    suites = curated_suites()
    rows = lifecycle_rows(machine)
    fig, ax = new_fig(7.4, 1.4 + 0.26 * len(suites))
    y = np.arange(len(suites))
    stage_share(ax, rows, suites, y)
    ax.set_yticks(y)
    ax.set_yticklabels(suites)
    ax.set_ylim(len(suites) - 0.5, -0.5)
    ax.set_xlabel("share of the lifecycle cost")
    bold_production(ax)
    headline(fig, "F5b  Where the lifecycle time goes, %s (E11; 10 attributes, 5 disclosed)" % SHORT[machine],
             [Patch(color=c) for c in STAGECOL], STAGES, ncol=5)
    save(fig, "F5b_lifecycle_stage_share_%s" % machine, outdir)


def f14():
    S = table("T1_E8_sizes.csv")
    size = {(r["alg"], r["item"]): float(r["median_bytes"]) for r in S}
    var = {(r["alg"], r["item"]) for r in S if r["min_bytes"] != r["max_bytes"]}
    sigs, kems = order(SIGS), order(KEMS)
    fig = plt.figure(figsize=(7.4, 11.6), layout="constrained")
    gs = fig.add_gridspec(2, 1, height_ratios=[len(sigs), len(kems) + 1.5])
    for k, (cats, second, title) in enumerate(((sigs, "sig", "signature schemes"), (kems, "ct", "key-encapsulation mechanisms"))):
        ax = fig.add_subplot(gs[k])
        series = [("public key", NEUTRAL, [size[(a, "pk")] for a in cats]),
                  ("signature" if second == "sig" else "ciphertext", NEUTRAL_LIGHT, [size[(a, second)] for a in cats])]
        bars(fig, ax, cats, series, unit="B", fmt=fmt_bytes, label="bytes", fam_of=family, fontsize=5.5)
        ax.set_title(title)
    hh = [Patch(color=NEUTRAL), Patch(color=NEUTRAL_LIGHT)]
    headline(fig, "F14  Sizes of public keys, signatures and ciphertexts (E8; Falcon and ECDSA signatures vary: median of 1,000)",
             hh, ["public key", "signature or ciphertext"], ncol=2)
    save(fig, "F14_key_and_signature_sizes")


def f15():
    T = table("T6_E10_issuer_holder_equivalence.csv")
    rows = sorted(T, key=lambda r: (int(r["alg"].split("-")[-1]), ORDER.index(r["machine"])))
    fig, ax = new_fig(7.4, 4.0)
    y = np.arange(len(rows))
    for yi, r in zip(y, rows):
        m = float(r["margin"]) * 100
        ax.add_patch(Rectangle((-m, yi - 0.36), 2 * m, 0.72, color="#E6E6E6", linewidth=0, zorder=0))
        d, lo, hi = (float(r[k]) * 100 for k in ("holder_vs_issuer", "tost90_lo", "tost90_hi"))
        ax.hlines(yi, lo, hi, color=COL[r["machine"]], linewidth=1.6, zorder=2)
        ax.scatter([d], [yi], s=22, color=COL[r["machine"]], marker=MARK[r["machine"]], zorder=3, linewidths=0)
        ax.annotate("%+.2f %%  (90 %% CI %+.2f to %+.2f; margin ±%.1f %%)  %s" % (d, lo, hi, m, "equivalent" if r["equivalent"] == "True" else "NOT equivalent"),
                    (1.02, yi), xycoords=("axes fraction", "data"), fontsize=5.8, va="center", ha="left")
    ax._labels_may_leave = True
    ax.axvline(0, color="#777777", linewidth=0.7)
    ax.set_yticks(y)
    ax.set_yticklabels(["%s, %s" % (r["alg"].upper(), NAME[r["machine"]]) for r in rows])
    ax.set_ylim(len(rows) - 0.5, -0.5)
    ax.set_xlim(-8, 8)
    ax.set_xticks([-8, -6, -4, -2, 0, 2, 4, 6, 8])
    ax.set_xticklabels(["%+d %%" % t if t else "0 %" for t in [-8, -6, -4, -2, 0, 2, 4, 6, 8]])
    ax.set_xlabel("holder-side signing against issuer-side signing (grey band: the machine's equivalence margin)")
    ax.grid(axis="x")
    for i in range(3, len(rows), 3):
        ax.axhline(i - 0.5, color="#BBBBBB", linewidth=0.5)
    hh, ll = machine_handles(kind="dot")
    headline(fig, "F15  Issuer-side and holder-side ML-DSA signing are equivalent on every machine (E10, TOST, 90 % CI)", hh, ll)
    save(fig, "F15_issuer_holder_equivalence")


def f16():
    T = table("T9_E5_native_vs_container.csv")
    fig, ax = new_fig(7.4, 3.4)
    y = np.arange(len(FAMILIES))
    texts = []
    for yi, fam in zip(y, FAMILIES):
        v = [float(r["native_over_container"]) for r in T if r["family"] == fam]
        med = float(np.median(v))
        ax.barh(yi, med - 1, left=1, height=0.6, color=COL["mac-m1"], linewidth=0, zorder=2)
        ax.scatter(v, np.full(len(v), yi) + np.linspace(-0.22, 0.22, len(v)), s=6, color="#222222", linewidths=0, alpha=0.6, zorder=3)
        texts.append(ax.annotate(fmt_ratio(med), (med, yi), xytext=(3 if med >= 1 else -3, 0), textcoords="offset points", fontsize=6, va="center",
                                 ha="left" if med >= 1 else "right", zorder=5,
                                 bbox=dict(boxstyle="square,pad=0.15", facecolor="white", edgecolor="none", alpha=0.9)))
    ax.axvline(1, color="#777777", linewidth=0.7)
    ax.set_yticks(y)
    ax.set_yticklabels(FAMILIES)
    ax.set_ylim(len(FAMILIES) - 0.5, -0.5)
    allv = [float(r["native_over_container"]) for r in T]
    ax.set_xlim(min(allv) - 0.06, max(allv) + 0.08)
    ax.set_xlabel("native macOS ÷ Docker container (bar: family median; dot: one operation; under 1× = native faster)")
    ax.grid(axis="x")
    fig.suptitle("F16  Native macOS against the Linux container on the same Mac (E5): Docker in Colima costs nothing measurable", fontsize=9)
    save(fig, "F16_native_vs_container")


POINT = {"tiny": "smallest transaction", "ecdsa-p256": "ECDSA P-256", "falcon-512": "Falcon-512", "ml-dsa-44": "ML-DSA-44", "ml-dsa-65": "ML-DSA-65",
         "ml-dsa-87": "ML-DSA-87", "slh-dsa-sha2-128s": "SLH-DSA-SHA2-128s", "slh-dsa-sha2-128f": "SLH-DSA-SHA2-128f", "slh-dsa-sha2-256f": "SLH-DSA-SHA2-256f"}


def f17():
    T = table("T13_E14_fabric_latency.csv")
    pts = sorted({r["point"] for r in T}, key=lambda p: float([r for r in T if r["point"] == p][0]["envelope_bytes"]))
    get = {(r["point"], r["mode"]): r for r in T}
    cats = ["%s (%.1f KB)" % (POINT.get(p, p), float(get[(p, "sequential")]["envelope_bytes"]) / 1000) for p in pts]
    fig, axs = new_fig(10.5, 4.6, 1, 2, gridspec_kw={"width_ratios": [1.5, 1]}, sharey=True)
    series = [("one at a time", COL["mac-m1"], [float(get[(p, "sequential")]["total_p50_ms"]) * 1000 for p in pts]),
              ("bursts of 40", MAC_LIGHT, [float(get[(p, "burst")]["total_p50_ms"]) * 1000 for p in pts])]
    bars(fig, axs[0], cats, series, label="time until committed (median)", fontsize=5.8, log=False)
    axs[0].set_xticks([0, 500e3, 1000e3, 1500e3, 2000e3])
    axs[0].set_xticklabels(["0 s", "0.5 s", "1.0 s", "1.5 s", "2.0 s"])
    fit_labels(fig, axs[0], list(axs[0].texts))
    axs[0].set_title("latency of one transaction")
    ax = axs[1]
    y = np.arange(len(pts))
    conf = [float(get[(p, "burst")]["mvcc_conflicts"]) for p in pts]
    total = [float(get[(p, "burst")]["transactions"]) for p in pts]
    ax.barh(y, conf, height=0.5, color=MAC_LIGHT, linewidth=0)
    texts = label_bars(ax, y, conf, ["%d of %d" % (c, t) for c, t in zip(conf, total)], fontsize=5.8)
    ax.set_xlim(0, max(total) * 0.6)
    ax.set_xlabel("burst transactions flagged MVCC_READ_CONFLICT")
    ax.set_title("conflicts from the shared CouchDB (bursts only)")
    ax.grid(axis="x")
    fit_labels(fig, ax, texts)
    headline(fig, "F17  Fabric latency against payload size on QChain's network (E14, Mac; payloads sized like each scheme's credential)",
             [Patch(color=COL["mac-m1"]), Patch(color=MAC_LIGHT)], ["sent one at a time (each waits for its commit)", "sent in bursts of 40"], ncol=2)
    save(fig, "F17_fabric_latency")


def combined():
    for op, tag in (("keygen", "a"), ("sign", "b"), ("verify", "c")):
        e1_bars(SIGS, op, ORDER, "F1%s_signature_%s" % (tag, op),
                "F1%s  Signature schemes, %s: median latency per call on the three machines (E1)" % (tag, OPNAME[op]))
    for op, tag in (("keygen", "a"), ("encaps", "b"), ("decaps", "c")):
        e1_bars(KEMS, op, ORDER, "F2%s_kem_%s" % (tag, op),
                "F2%s  Key-encapsulation mechanisms, %s: median latency per call on the three machines (E1)" % (tag, OPNAME[op]))
    f5a(ORDER, "F5a_lifecycle_cost", "F5a  Credential lifecycle cost per suite on the three machines (E11; 10 attributes, 5 disclosed)")
    f14()
    f15()
    f16()
    f17()


def per_machine():
    for m in ORDER:
        out = machine_out(m, "figures")
        for op, tag in (("keygen", "a"), ("sign", "b"), ("verify", "c")):
            e1_bars(SIGS, op, [m], "F1%s_signature_%s_%s" % (tag, op, m),
                    "F1%s  Signature schemes, %s, %s: median latency per call (E1)" % (tag, OPNAME[op], SHORT[m]), out)
        for op, tag in (("keygen", "a"), ("encaps", "b"), ("decaps", "c")):
            e1_bars(KEMS, op, [m], "F2%s_kem_%s_%s" % (tag, op, m),
                    "F2%s  Key-encapsulation mechanisms, %s, %s: median latency per call (E1)" % (tag, OPNAME[op], SHORT[m]), out)
        f5a([m], "F5a_lifecycle_cost_%s" % m, "F5a  Credential lifecycle cost per suite, %s (E11; 10 attributes, 5 disclosed)" % SHORT[m], out)
        f5b(m, out)


if __name__ == "__main__":
    combined()
    per_machine()
    print("\n".join(PROBLEMS) or "no layout problems")
