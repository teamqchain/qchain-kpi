"""F0 run quality, F5 lifecycle, F6 heatmap, F7 envelope scaling, F8 integration cost, F10 memory, F11 ledger,
F12 concurrency, F13 rejection timing and message size."""
import os
import sys

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from figures_common import *  # noqa: E402,F401,F403
import e11  # noqa: E402
from e1 import family  # noqa: E402
from matplotlib.lines import Line2D  # noqa: E402
from matplotlib.patches import Patch  # noqa: E402

FAM_ORDER = {f: i for i, f in enumerate(FAMILIES)}


def stage_share(ax, rows, suites, y, label=True):
    """100 % bars split by lifecycle stage, each segment of 10 % or more labelled with its share."""
    left = np.zeros(len(suites))
    for st, c, tc in zip(STAGES, STAGECOL, STAGETEXT):
        v = np.array([float(rows[s][st + "_us"]) / float(rows[s]["lifecycle_total_us"]) * 100 for s in suites])
        ax.barh(y, v, left=left, color=c, height=0.72, label=st, linewidth=0.6, edgecolor="white")
        for yi, l0, vi in zip(y, left, v):
            if label and vi >= 10:
                ax.text(l0 + vi / 2, yi, "%.0f %%" % vi, ha="center", va="center", fontsize=5.5, color=tc)
        left += v
    ax.set_xlim(0, 100)
    ax.set_xticks([0, 20, 40, 60, 80, 100])
    ax.set_xticklabels(["0 %", "20 %", "40 %", "60 %", "80 %", "100 %"])


def f5():
    T = table("T3b_E11_lifecycle_per_suite.csv")
    suites = [r["suite"] for r in T if r["machine"] == "win-ryzen"]
    suites = sorted(suites, key=lambda s: float([r for r in T if r["machine"] == "win-ryzen" and r["suite"] == s][0]["lifecycle_total_us"]))
    fig, axs = new_fig(10.5, 5.0, 1, 2, gridspec_kw={"width_ratios": [1.15, 1]}, sharey=True)
    ax = axs[0]
    y = np.arange(len(suites))
    allv = []
    for mach, off in (("mac-m1", -0.24), ("win-ryzen", 0.0), ("vm-x86v2", 0.24)):
        rows = {r["suite"]: r for r in T if r["machine"] == mach}
        x = np.array([float(rows[s]["lifecycle_total_us"]) / 1000 for s in suites])
        lo = np.array([float(rows[s]["total_ci_lo_us"]) / 1000 for s in suites])
        hi = np.array([float(rows[s]["total_ci_hi_us"]) / 1000 for s in suites])
        allv += list(lo) + list(hi)
        ax.hlines(y + off, lo, hi, color=COL[mach], linewidth=1.2)
        ax.scatter(x, y + off, s=14, color=COL[mach], marker=MARK[mach], zorder=3, linewidths=0)
    ax.set_yticks(y)
    ax.set_yticklabels(suites)
    ax.set_ylim(len(suites) - 0.5, -0.5)
    time_axis(ax, "x", "lifecycle cost: median and 95 % CI", unit="ms")
    ax.set_xlim(min(allv) / 1.6, max(allv) * 1.6)
    ax.grid(axis="x")
    rows = {r["suite"]: r for r in T if r["machine"] == "win-ryzen"}
    stage_share(axs[1], rows, suites, y)
    axs[1].set_xlabel("share of the lifecycle cost (laptop)")
    hm, lm = machine_handles(kind="dot")
    hs = [Patch(color=c) for c in STAGECOL]
    headline(fig, "F5  Credential lifecycle per algorithm suite (E11: activation + issuance + retrieval + presentation + verification; 10 attributes, 5 disclosed)",
             hm + hs, lm + STAGES, ncol=8)
    save(fig, "F5_lifecycle_per_suite")


def f6():
    c = e11.load("mac")
    sigs = sorted({k[1].split("+")[0] for k in c if "+" in k[1]})
    kems = sorted({k[1].split("+")[1] for k in c if "+" in k[1]})
    tot = {}
    for s in sigs:
        for k in kems:
            sv = e11.stage_vals(c, s + "+" + k)
            if sv:
                tot[(s, k)] = float(sum(np.median(v) for v in sv.values()))
    base = tot[("ecdsa-p256", "dhkem-x25519")]
    sigs = sorted(sigs, key=lambda s: (np.median([tot[(s, k)] for k in kems if (s, k) in tot]), s))
    kems = sorted(kems, key=lambda k: (np.median([tot[(s, k)] for s in sigs if (s, k) in tot]), k))
    M = np.array([[tot[(s, k)] / base for k in kems] for s in sigs])
    fig, ax = new_fig(6.6, 7.0)
    from matplotlib.colors import LogNorm
    im = ax.imshow(M, norm=LogNorm(vmin=0.5, vmax=3000), cmap="viridis_r", aspect="auto")
    ax.set_xticks(range(len(kems)))
    ax.set_xticklabels(kems, rotation=60, ha="right")
    ax.set_yticks(range(len(sigs)))
    ax.set_yticklabels(sigs)
    for i in range(len(sigs)):
        for j in range(len(kems)):
            ax.text(j, i, "%.1f" % M[i, j], ha="center", va="center", fontsize=4.5, color="white" if M[i, j] > 8 else "black")
    cb = fig.colorbar(im, ax=ax, shrink=0.7)
    ticks = [(0.5, "0.5×"), (1, "1×"), (10, "10×"), (100, "100×"), (1000, "1,000×")]
    cb.set_ticks([t for t, _ in ticks])
    cb.set_ticklabels([s for _, s in ticks])
    cb.ax.minorticks_off()
    cb.set_label("lifecycle cost relative to ECDSA P-256 + X25519 (log scale)")
    ax.set_xlabel("key-encapsulation mechanism")
    ax.set_ylabel("signature scheme")
    fig.suptitle("F6  Lifecycle cost of every signature × KEM pair (E11, Mac, all 176 pairs)", fontsize=9)
    save(fig, "F6_pair_heatmap")


def f7():
    S = table("T3c_E11_scaling_production_suite.csv")
    fig, axs = new_fig(10.5, 3.8, 1, 3)
    n_ticks = [(1, "1"), (5, "5"), (10, "10"), (20, "20"), (64, "64")]
    for ax, st in zip(axs[:2], ("issuance", "verification")):
        allv = []
        for mach in ORDER:
            for v, ls in ((16, "-"), (4096, ":")):
                rows = sorted([r for r in S if (r["machine"], r["stage"], int(r["value_bytes"])) == (mach, st, v)], key=lambda r: int(r["attributes_n"]))
                vals = [float(r["p50_us"]) for r in rows]
                allv += vals
                ax.plot([int(r["attributes_n"]) for r in rows], vals, color=COL[mach], linestyle=ls, marker=MARK[mach], markersize=3)
        log_axis(ax, "x", n_ticks, "attributes in the credential")
        ax.set_xlim(0.8, 80)
        time_axis(ax, "y", "median time of the stage")
        ax.set_ylim(min(allv) / 1.6, max(allv) * 1.6)
        ax.set_title("%s (production suite)" % st)
        ax.grid(True)
    B = table("T5b_E13_ipfs_envelope_bytes.csv")
    ax = axs[2]
    allv = []
    for kem, c in (("ml-kem-768", FAMCOL["ML-KEM"]), ("hqc-1", FAMCOL["HQC"]), ("dhkem-x25519", FAMCOL["classical"])):
        for v, ls in ((16, "-"), (4096, ":")):
            rows = sorted([r for r in B if r["kem"] == kem and int(r["value_bytes"]) == v], key=lambda r: int(r["attributes_n"]))
            vals = [int(r["envelope_bytes"]) for r in rows]
            allv += vals
            ax.plot([int(r["attributes_n"]) for r in rows], vals, color=c, linestyle=ls, marker="o", markersize=3)
    log_axis(ax, "x", n_ticks, "attributes in the credential")
    ax.set_xlim(0.8, 80)
    byte_axis(ax, "y", "envelope size on IPFS")
    ax.set_ylim(min(allv) / 1.6, max(allv) * 1.6)
    ax.set_title("IPFS envelope size (E13)")
    ax.grid(True)
    h = [Line2D([0], [0], color=COL[m], marker=MARK[m], markersize=4) for m in ORDER]
    l = [SHORT[m] for m in ORDER]
    h += [Line2D([0], [0], color=FAMCOL[f], marker="o", markersize=4) for f in ("ML-KEM", "HQC", "classical")]
    l += ["ML-KEM-768 envelope", "HQC-1 envelope", "X25519 envelope"]
    h += [Line2D([0], [0], color="#555555", linestyle="-"), Line2D([0], [0], color="#555555", linestyle=":")]
    l += ["16-byte attribute values", "4,096-byte attribute values"]
    headline(fig, "F7  Envelope time and size against the number of attributes and the value size (E11 times, E13 sizes)", h, l, ncol=4)
    save(fig, "F7_envelope_scaling")


def f8():
    T = table("T8_E3_integration_layers.csv")
    E2 = table("T3_E2_integration_overhead.csv")
    fig, axs = new_fig(10.5, 3.9, 1, 2)
    ax = axs[0]
    for i, fam in enumerate(FAMILIES):
        v = [float(r["l1_over_l0"]) for r in T if r["family"] == fam]
        ax.scatter(np.full(len(v), i) + np.linspace(-0.18, 0.18, len(v)), v, s=7, color=COL["mac-m1"], linewidths=0, alpha=0.8, zorder=2)
        med = float(np.median(v))
        ax.hlines(med, i - 0.3, i + 0.3, color="black", zorder=3)
        ax.annotate(fmt_ratio(med, 3), (i + 0.3, med), xytext=(2, 0), textcoords="offset points", fontsize=5.5, va="center")
    ax.axhline(1, color="#999999", linewidth=0.6)
    ax.set_xticks(range(len(FAMILIES)))
    ax.set_xticklabels(FAMILIES, rotation=30, ha="right")
    ax.set_xlim(-0.6, len(FAMILIES) - 0.05)
    ax.set_ylabel("Go binding ÷ C library call (E3)")
    ax.set_title("E3 (Mac): the Go binding adds almost nothing")
    ax.grid(axis="y")
    ax = axs[1]
    w = 0.27
    texts = []
    for j, mach in enumerate(ORDER):
        pos, med = [], []
        for i, fam in enumerate(FAMILIES):
            v = [float(r["prod_over_reuse"]) for r in E2 if r["machine"] == mach and r["family"] == fam]
            pos.append(i + (j - 1) * w)
            med.append(float(np.median(v)) if v else float("nan"))
        ax.bar(pos, med, width=w, color=COL[mach], linewidth=0)
        texts += label_bars(ax, pos, med, [fmt_ratio(m) if m == m else "" for m in med], orient="v", fontsize=5.2, rotation=90)
        for p, m in zip(pos, med):
            if m != m:
                ax.text(p, 0.05, "not run", rotation=90, ha="center", va="bottom", fontsize=4.8, color="#777777")
    ax.axhline(1, color="#999999", linewidth=0.6)
    ax.set_xticks(range(len(FAMILIES)))
    ax.set_xticklabels(FAMILIES, rotation=30, ha="right")
    ax.set_ylabel("production pattern ÷ reused key (E2, family median)")
    ax.set_title("E2: the deployed pattern (hex strings, key import per call)")
    ax.set_ylim(0, 1.45)
    ax.grid(axis="y")
    fit_labels(fig, ax, texts, orient="v")
    hh, ll = machine_handles(kind="bar")
    headline(fig, "F8  What the integration adds on top of the C library (E3 Mac, every operation as a dot; E2 three machines)", hh, ll)
    save(fig, "F8_integration_cost")


def f10():
    M = table("T4_E9_memory.csv")
    fig, axs = new_fig(10.5, 5.2, 1, 2)
    algs = sorted({r["alg"] for r in M})
    for ax, op in zip(axs, ("sign", "decaps")):
        names, vals = [], []
        for a in algs:
            r = [x for x in M if x["alg"] == a and x["op"] == op and x["route"] == "c" and x["quantity"] == "stack" and x["machine"] == "win-ryzen"]
            if r:
                names.append(a)
                vals.append(float(r[0]["median"]) / 1024)
        o = sorted(range(len(vals)), key=lambda i: (vals[i], names[i]))
        y = np.arange(len(o))
        v = [vals[i] for i in o]
        ax.barh(y, v, color=[FAMCOL[family(names[i])] for i in o], height=0.7, linewidth=0)
        ax.set_yticks(y)
        ax.set_yticklabels([names[i] for i in o])
        ax.set_ylim(len(o) - 0.5, -0.5)
        log_axis(ax, "x", [(1, "1 KiB"), (10, "10 KiB"), (100, "100 KiB"), (1000, "1,000 KiB")], "peak stack use of one %s call" % op)
        ax.set_xlim(1, max(v) * 1.5)
        texts = label_bars(ax, y, v, ["%.1f KiB" % x for x in v])
        fit_labels(fig, ax, texts)
        ax.set_title("%s stack (laptop; Mac and VM within 3 %%)" % {"sign": "signing", "decaps": "decapsulation"}[op])
        ax.grid(axis="x")
    fams = [f for f in FAMILIES]
    headline(fig, "F10  Memory: stack high-water mark per call (E9, stack painting)", [Patch(color=FAMCOL[f]) for f in fams], fams, ncol=6)
    save(fig, "F10_memory_stack")


def f11():
    L = [r for r in table("T5_E13_ledger_issue_credential.csv") if r["scenario"] == "measured" and r["attributes_n"] == "10"]

    def val(a, q):
        return float([r for r in L if r["sig_scheme"] == a and r["quantity"] == q][0]["median"])

    sigs = sorted({r["sig_scheme"] for r in L}, key=lambda a: (val(a, "envelope"), a))  # name breaks ties: set order varies per process
    fig, axs = new_fig(10.5, 5.2, 1, 3, sharey=True)
    y = np.arange(len(sigs))
    kb = [val(a, "envelope") / 1000 for a in sigs]
    per = [val(a, "per_block_max") for a in sigs]
    gb = [val(a, "envelope") * 1e6 / 1e9 for a in sigs]
    for ax, v, txt in ((axs[0], kb, ["%.1f KB" % x for x in kb]), (axs[1], per, ["%d" % x for x in per]), (axs[2], gb, ["%.1f GB" % x for x in gb])):
        ax.barh(y, v, height=0.62, color=COL["mac-m1"], linewidth=0)
        ax.grid(axis="x")
        ax._txt = txt
    log_axis(axs[0], "x", [(1, "1 KB"), (10, "10 KB"), (100, "100 KB"), (1000, "1 MB")], "kilobytes per issueCredential transaction (n = 10)")
    axs[0].set_xlim(1, max(kb) * 1.5)
    axs[1].set_xlabel("credentials per full block")
    axs[1].set_xlim(0, 11)
    log_axis(axs[2], "x", [(1, "1 GB"), (10, "10 GB"), (100, "100 GB"), (1000, "1 TB")], "gigabytes per million credentials (one peer)")
    axs[2].set_xlim(1, max(gb) * 1.5)
    for ax, v in zip(axs, (kb, per, gb)):
        fit_labels(fig, ax, label_bars(ax, y, v, ax._txt))
    axs[0].set_yticks(y)
    axs[0].set_yticklabels(sigs)
    axs[0].set_ylim(len(sigs) - 0.5, -0.5)
    fig.suptitle("F11  Ledger footprint of a credential by signature scheme (E13 model, calibrated on a captured block)", fontsize=9)
    save(fig, "F11_ledger_footprint")


def f12():
    C = table("T12_E12_concurrency.csv")
    fig, axs = new_fig(10.5, 3.8, 1, 2)
    for ax, st in zip(axs, ("issuance-n10-v16", "verification-n10-k5-v16")):
        for mach in ORDER:
            rows = sorted([r for r in C if r["machine"] == mach and r["suite"] == "ml-dsa-44+ml-kem-768" and r["stage"] == st], key=lambda r: int(r["workers"]))
            ax.plot([int(r["workers"]) for r in rows], [float(r["speedup_vs_1"]) for r in rows], color=COL[mach], marker=MARK[mach], markersize=3)
        ax.plot([1, 8], [1, 8], color="#999999", linewidth=0.6, linestyle="--")
        log_axis(ax, "x", [(1, "1"), (2, "2"), (4, "4"), (8, "8")], "workers", base=2)
        log_axis(ax, "y", [(1, "1×"), (2, "2×"), (4, "4×"), (8, "8×")], "throughput speed-up over one worker", base=2)
        ax.set_xlim(0.85, 9.4)
        ax.set_ylim(0.85, 9.4)
        ax.set_title("%s (ML-DSA-44 + ML-KEM-768)" % st.split("-")[0])
        ax.grid(True)
    h, l = machine_handles(kind="line")
    h.append(Line2D([0], [0], color="#999999", linewidth=0.6, linestyle="--"))
    l.append("ideal (speed-up = workers)")
    headline(fig, "F12  Concurrency scaling (E12; laptop 4 cores / 8 threads, VM 2 vCPUs, Mac up to 4 workers)", h, l, ncol=4)
    save(fig, "F12_concurrency_scaling")


TESTS = [("badsig_over_valid", "o", "corrupted signature"), ("wrongmsg_over_valid", "s", "signature over a different message"),
         ("tampered_over_valid", "^", "tampered ciphertext or public key")]


def f13():
    """E7 as one row per family (range of its tests and their median, zoomed on 1×; an outlier is written out), and E6."""
    S = table("T11_E7_rejection_timing.csv")
    M = table("T10_E6_message_size.csv")
    rat = [(float(r[c]), r, name) for r in S for c, _, name in TESTS if r[c]]
    usual = [x for x in rat if x[0] >= 0.9]
    odd = [x for x in rat if x[0] < 0.9]
    fig = plt.figure(figsize=(10.5, 3.9), layout="constrained")
    gs = fig.add_gridspec(1, 3, width_ratios=[1.15, 1, 1])
    ax = fig.add_subplot(gs[0, 0])
    y = np.arange(len(FAMILIES))
    for yi, fam in zip(y, FAMILIES):
        v = [x for x, r, _ in usual if family(r["alg"]) == fam]
        lo, hi, med = min(v), max(v), float(np.median(v))
        ax.hlines(yi, lo, hi, color=COL["mac-m1"], linewidth=2.2, alpha=0.45, zorder=2)
        ax.scatter(med, yi, s=26, color=COL["mac-m1"], zorder=3, linewidths=0)
        ax.annotate("%d tests: %.3f× (%.3f–%.3f)" % (len(v), med, lo, hi), (hi, yi), xytext=(5, 0), textcoords="offset points", fontsize=5.8, va="center")
    ax.axvline(1, color="#999999", linewidth=0.7, zorder=1)
    ax.set_yticks(y)
    ax.set_yticklabels(FAMILIES)
    ax.set_ylim(len(FAMILIES) - 0.4, -0.6)
    ax.set_xlim(0.96, 1.04)
    ax.set_xticks([0.96, 0.98, 1.0, 1.02, 1.04])
    ax.set_xticklabels(["%.2f×" % t for t in (0.96, 0.98, 1.0, 1.02, 1.04)])
    ax.set_xlabel("time to reject ÷ time to accept (line: range of the tests; dot: median)")
    if odd:
        note = "\n".join("Not shown: %s, %s, %.2f×%s" % (r["alg"], "tampered public point" if r["alg"].startswith("dhkem") else name, v,
                          "\n(the malformed point is refused before any computation, by design)" if r["alg"].startswith("dhkem") else "")
                          for v, r, name in odd)
        ax.text(0.01, -0.33, note, transform=ax.transAxes, fontsize=5.8, va="top")
        ax._labels_may_leave = True
    ax.set_title("E7 (Mac): rejecting an invalid input takes as long as accepting a valid one\n(%d of %d tests within %.1f %%)"
                 % (len(usual), len(rat), 100 * max(abs(x[0] - 1) for x in usual)), fontsize=8)
    ax.grid(axis="x")
    sizes = [(64, "64 B"), (1024, "1 KiB"), (10240, "10 KiB"), (102400, "100 KiB"), (1048576, "1 MiB")]
    algs = (("ml-dsa-44", "ML-DSA"), ("falcon-512", "Falcon"), ("slh-dsa-sha2-128f", "SLH-DSA"), ("ed25519", "classical"))
    for k, op in enumerate(("sign", "verify")):
        ax = fig.add_subplot(gs[0, k + 1])
        allv = []
        for alg, fam in algs:
            rows = sorted([r for r in M if r["alg"] == alg and r["op"] == op], key=lambda r: int(r["message_bytes"]))
            vals = [float(r["p50_us"]) for r in rows]
            allv += vals
            ax.plot([int(r["message_bytes"]) for r in rows], vals, color=FAMCOL[fam], marker="o", markersize=3)
        log_axis(ax, "x", sizes, "message size")
        ax.set_xlim(40, 1.7e6)
        time_axis(ax, "y", "%s latency" % {"sign": "signing", "verify": "verification"}[op])
        ax.set_ylim(min(allv) / 1.6, max(allv) * 1.6)
        ax.set_title("E6 (Mac): %s against message size" % {"sign": "signing", "verify": "verification"}[op])
        ax.grid(True)
    h = [Line2D([0], [0], color=FAMCOL[fam], marker="o", markersize=3) for _, fam in algs]
    headline(fig, "F13  Rejection timing (E7) and message-size scaling (E6), Mac", h, ["E6: " + alg for alg, _ in algs], ncol=4)
    save(fig, "F13_negative_path_and_message_size")


def f0():
    Q = table("T0_run_quality.csv")
    rows = [r for m in ORDER for r in Q if r["machine"] == m]
    fig, ax = new_fig(7.6, 5.4)
    y = np.arange(len(rows))
    vals = [fnum(r["flagged_pct"]) for r in rows]
    ax.barh(y, [0 if v != v else v for v in vals], color=[COL[r["machine"]] for r in rows], height=0.7, linewidth=0)
    txt = []
    for r, v in zip(rows, vals):
        if v != v:
            txt.append("not flagged: recorded once")
        elif r["note"]:
            txt.append("0 % (not timed)")
        else:
            txt.append("%.1f %%" % v)
    texts = [ax.annotate(s, (0 if v != v else v, yi), xytext=(2, 0), textcoords="offset points", fontsize=5.5, va="center") for s, v, yi in zip(txt, vals, y)]
    ax.set_yticks(y)
    ax.set_yticklabels(["%s: %s" % (NAME[r["machine"]], plan_label(r["plan"], r["variant"])) for r in rows], fontsize=5.8)
    ax.set_ylim(len(rows) - 0.5, -0.5)
    ax.set_xlim(0, max(v for v in vals if v == v) * 1.15)
    ax.set_xlabel("attempts flagged for an external cause (% of all attempts of the run)")
    ax.grid(axis="x")
    fit_labels(fig, ax, texts)
    hh, ll = machine_handles(kind="bar")
    headline(fig, "F0  Run quality: share of attempts flagged and repeated, per main run", hh, ll)
    save(fig, "F0_run_quality")


if __name__ == "__main__":
    for fn in (f0, f5, f6, f7, f8, f10, f11, f12, f13):
        fn()
    print("\n".join(PROBLEMS) or "no layout problems")
