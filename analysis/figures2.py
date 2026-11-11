"""F5 lifecycle, F6 heatmap, F7 envelope scaling, F8 integration cost, F10 memory, F11 ledger, F12 concurrency, F13 negative path and message size."""
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from figures_common import *  # noqa
import e11  # noqa
from e1 import family  # noqa

STAGES = ["activation", "issuance", "retrieval", "presentation", "verification"]
STAGECOL = ["#999999", "#0072B2", "#009E73", "#E69F00", "#D55E00"]


def f5():
    T = table("T3b_E11_lifecycle_per_suite.csv")
    suites = [r["suite"] for r in T if r["machine"] == "win-ryzen"]
    suites = sorted(suites, key=lambda s: float([r for r in T if r["machine"] == "win-ryzen" and r["suite"] == s][0]["lifecycle_total_us"]))
    fig, axs = plt.subplots(1, 2, figsize=(10.5, 4.6), gridspec_kw={"width_ratios": [1.15, 1]}, sharey=True)
    ax = axs[0]
    y = np.arange(len(suites))
    for mach, off in (("mac-m1", -0.22), ("win-ryzen", 0.0), ("vm-x86v2", 0.22)):
        rows = {r["suite"]: r for r in T if r["machine"] == mach}
        x = np.array([float(rows[s]["lifecycle_total_us"]) / 1000 for s in suites])
        lo = np.array([float(rows[s]["total_ci_lo_us"]) / 1000 for s in suites])
        hi = np.array([float(rows[s]["total_ci_hi_us"]) / 1000 for s in suites])
        ax.hlines(y + off, lo, hi, color=COL[mach], linewidth=1.2)
        ax.scatter(x, y + off, s=14, color=COL[mach], marker=MARK[mach], label=SHORT[mach], zorder=3, linewidths=0)
    ax.set_xscale("log")
    ax.set_yticks(y)
    ax.set_yticklabels(suites)
    ax.invert_yaxis()
    ax.set_xlabel("lifecycle cost, milliseconds (median, 95 % CI)")
    ax.grid(axis="x")
    ax.legend(frameon=False, loc="upper right", fontsize=6)
    ax = axs[1]
    rows = {r["suite"]: r for r in T if r["machine"] == "win-ryzen"}
    left = np.zeros(len(suites))
    for st, c in zip(STAGES, STAGECOL):
        v = np.array([float(rows[s][st + "_us"]) / float(rows[s]["lifecycle_total_us"]) * 100 for s in suites])
        ax.barh(y, v, left=left, color=c, height=0.7, label=st, linewidth=0)
        left += v
    ax.set_xlim(0, 100)
    ax.set_xlabel("share of the lifecycle cost, % (laptop)")
    h, l = ax.get_legend_handles_labels()
    fig.legend(h, l, loc="lower center", ncol=5, frameon=False, bbox_to_anchor=(0.7, -0.03))
    fig.suptitle("F5  Credential lifecycle per algorithm suite (E11: activation + issuance + retrieval + presentation + verification; 10 attributes, 5 disclosed)", y=1.0, fontsize=9)
    fig.tight_layout()
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
    sigs = sorted(sigs, key=lambda s: np.median([tot[(s, k)] for k in kems if (s, k) in tot]))
    kems = sorted(kems, key=lambda k: np.median([tot[(s, k)] for s in sigs if (s, k) in tot]))
    M = np.array([[tot[(s, k)] / base for k in kems] for s in sigs])
    fig, ax = plt.subplots(figsize=(6.2, 6.6))
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
    cb.set_label("lifecycle cost relative to ECDSA P-256 + X25519 (Mac, all 176 pairs)")
    ax.set_title("F6  Lifecycle cost of every signature x KEM pair (E11, Mac)", fontsize=9)
    save(fig, "F6_pair_heatmap")


def f7():
    S = table("T3c_E11_scaling_production_suite.csv")
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 3.2))
    for ax, st in zip(axs[:2], ("issuance", "verification")):
        for mach in COL:
            for v, ls in ((16, "-"), (4096, ":")):
                rows = [r for r in S if (r["machine"], r["stage"], int(r["value_bytes"])) == (mach, st, v)]
                rows.sort(key=lambda r: int(r["attributes_n"]))
                ax.plot([int(r["attributes_n"]) for r in rows], [float(r["p50_us"]) for r in rows], color=COL[mach], linestyle=ls, marker=MARK[mach], markersize=3,
                        label="%s, %d-byte values" % (SHORT[mach].split(" (")[0], v))
        ax.set_xscale("log")
        ax.set_yscale("log")
        ax.set_xlabel("attributes in the credential")
        ax.set_ylabel("microseconds")
        ax.set_title(st + " (production suite)")
        ax.grid(True)
    axs[0].legend(frameon=False, fontsize=5.5)
    B = table("T5b_E13_ipfs_envelope_bytes.csv")
    ax = axs[2]
    for kem, c in (("ml-kem-768", FAMCOL["ML-KEM"]), ("hqc-1", FAMCOL["HQC"]), ("dhkem-x25519", FAMCOL["classical"])):
        for v, ls in ((16, "-"), (4096, ":")):
            rows = sorted([r for r in B if r["kem"] == kem and int(r["value_bytes"]) == v], key=lambda r: int(r["attributes_n"]))
            ax.plot([int(r["attributes_n"]) for r in rows], [int(r["envelope_bytes"]) / 1000 for r in rows], color=c, linestyle=ls, marker="o", markersize=3, label="%s, %d-byte" % (kem, v))
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.set_xlabel("attributes in the credential")
    ax.set_ylabel("envelope size on IPFS, kilobytes")
    ax.set_title("IPFS envelope size (E13)")
    ax.legend(frameon=False, fontsize=5.5)
    ax.grid(True)
    fig.suptitle("F7  Envelope time and size against the number of attributes and the value size", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F7_envelope_scaling")


def f8():
    T = table("T8_E3_integration_layers.csv")
    E2 = table("T3_E2_integration_overhead.csv")
    fams = ["ML-DSA", "Falcon", "ML-KEM", "HQC", "SLH-DSA", "classical"]
    fig, axs = plt.subplots(1, 2, figsize=(10.5, 3.4))
    ax = axs[0]
    for i, fam in enumerate(fams):
        v = [float(r["l1_over_l0"]) for r in T if r["family"] == fam]
        ax.scatter(np.full(len(v), i) + np.linspace(-0.18, 0.18, len(v)), v, s=7, color=FAMCOL[fam], linewidths=0)
        ax.hlines(np.median(v), i - 0.3, i + 0.3, color="black")
    ax.axhline(1, color="#999999", linewidth=0.6)
    ax.set_xticks(range(len(fams)))
    ax.set_xticklabels(fams, rotation=30)
    ax.set_ylabel("Go binding (reuse arm) / C library call")
    ax.set_title("E3 (Mac): the Go binding adds almost nothing")
    ax.grid(axis="y")
    ax = axs[1]
    w = 0.27
    for j, mach in enumerate(("mac-m1", "win-ryzen", "vm-x86v2")):
        med = []
        for fam in fams:
            v = [float(r["prod_over_reuse"]) for r in E2 if r["machine"] == mach and r["family"] == fam]
            med.append(np.median(v) if v else np.nan)
        ax.bar(np.arange(len(fams)) + (j - 1) * w, med, width=w, color=COL[mach], label=SHORT[mach], linewidth=0)
    ax.axhline(1, color="#999999", linewidth=0.6)
    ax.set_xticks(range(len(fams)))
    ax.set_xticklabels(fams, rotation=30)
    ax.set_ylabel("production pattern / reused key")
    ax.set_title("E2: the deployed pattern (hex strings, key import per call)")
    ax.legend(frameon=False, fontsize=6)
    ax.grid(axis="y")
    fig.suptitle("F8  What the integration adds on top of the C library (E3 and E2)", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F8_integration_cost")


def f10():
    M = table("T4_E9_memory.csv")
    fig, axs = plt.subplots(1, 2, figsize=(10.5, 4.6))
    sel = [("sign", "stack"), ("decaps", "stack")]
    algs = sorted({r["alg"] for r in M})
    for ax, (op, q) in zip(axs, sel):
        names, vals = [], []
        for a in algs:
            r = [x for x in M if x["alg"] == a and x["op"] == op and x["route"] == "c" and x["quantity"] == q and x["machine"] == "win-ryzen"]
            if r:
                names.append(a)
                vals.append(float(r[0]["median"]))
        o = np.argsort(vals)
        ax.barh(np.arange(len(o)), np.array(vals)[o] / 1024, color=[FAMCOL[family(names[i])] for i in o], height=0.7, linewidth=0)
        ax.set_yticks(np.arange(len(o)))
        ax.set_yticklabels([names[i] for i in o])
        ax.set_xscale("log")
        ax.set_xlabel("peak stack use of one %s call, KiB" % op)
        ax.set_title("%s stack (laptop; Mac and VM within 3 %%)" % op)
        ax.grid(axis="x")
    fig.suptitle("F10  Memory: stack high-water mark per call (E9, stack painting)", y=1.0, fontsize=9)
    fig.tight_layout()
    save(fig, "F10_memory_stack")


def f11():
    L = table("T5_E13_ledger_issue_credential.csv")
    scen = ("measured", "projected-ml-dsa-44")
    sigs = sorted({r["sig_scheme"] for r in L}, key=lambda a: (float([r for r in L if r["sig_scheme"] == a and r["scenario"] == "measured" and r["attributes_n"] == "10" and r["quantity"] == "envelope"][0]["median"]), a))  # name breaks ties: set order varies per process
    fig, axs = plt.subplots(1, 3, figsize=(10.5, 4.8), sharey=True)
    for ax, (q, title, f) in zip(axs, (("envelope", "bytes per issueCredential transaction (n = 10)", 1), ("per_block_max", "issueCredential transactions that fit in one block", 1))):
        pass
    ax = axs[0]
    for sc, c, off in (("measured", "#0072B2", -0.18), ("projected-ml-dsa-44", "#D55E00", 0.18)):
        v = [float([r for r in L if r["sig_scheme"] == a and r["scenario"] == sc and r["attributes_n"] == "10" and r["quantity"] == "envelope"][0]["median"]) / 1000 for a in sigs]
        ax.barh(np.arange(len(sigs)) + off, v, height=0.34, color=c, label={"measured": "as captured (Fabric ECDSA identities)", "projected-ml-dsa-44": "projected: Fabric identities on ML-DSA-44"}[sc], linewidth=0)
    ax.set_yticks(np.arange(len(sigs)))
    ax.set_yticklabels(sigs)
    ax.invert_yaxis()
    ax.set_xscale("log")
    ax.set_xlabel("kilobytes per issueCredential transaction (n = 10)")
    ax.grid(axis="x")
    ax = axs[1]
    for sc, c, off in (("measured", "#0072B2", -0.18), ("projected-ml-dsa-44", "#D55E00", 0.18)):
        v = [float([r for r in L if r["sig_scheme"] == a and r["scenario"] == sc and r["attributes_n"] == "10" and r["quantity"] == "per_block_max"][0]["median"]) for a in sigs]
        ax.barh(np.arange(len(sigs)) + off, v, height=0.34, color=c, linewidth=0)
    ax.set_xlabel("credentials per full block")
    ax.grid(axis="x")
    ax = axs[2]
    for sc, c, off in (("measured", "#0072B2", -0.18), ("projected-ml-dsa-44", "#D55E00", 0.18)):
        v = []
        for a in sigs:
            e = float([r for r in L if r["sig_scheme"] == a and r["scenario"] == sc and r["attributes_n"] == "10" and r["quantity"] == "envelope"][0]["median"])
            v.append(e * 1e6 / 1e9)
        ax.barh(np.arange(len(sigs)) + off, v, height=0.34, color=c, linewidth=0)
    ax.set_xscale("log")
    ax.set_xlabel("gigabytes per million credentials (one peer)")
    ax.grid(axis="x")
    h, l = axs[0].get_legend_handles_labels()
    fig.legend(h, l, loc="lower center", ncol=2, frameon=False, bbox_to_anchor=(0.5, -0.03))
    fig.suptitle("F11  Ledger footprint of a credential by signature scheme (E13 model, calibrated on a captured block)", y=1.0, fontsize=9)
    fig.tight_layout()
    save(fig, "F11_ledger_footprint")


def f12():
    C = table("T12_E12_concurrency.csv")
    fig, axs = plt.subplots(1, 2, figsize=(10.5, 3.4))
    for ax, st in zip(axs, ("issuance-n10-v16", "verification-n10-k5-v16")):
        for mach in COL:
            rows = sorted([r for r in C if r["machine"] == mach and r["suite"] == "ml-dsa-44+ml-kem-768" and r["stage"] == st], key=lambda r: int(r["workers"]))
            ax.plot([int(r["workers"]) for r in rows], [float(r["speedup_vs_1"]) for r in rows], color=COL[mach], marker=MARK[mach], markersize=3, label=SHORT[mach])
        ax.plot([1, 8], [1, 8], color="#999999", linewidth=0.6, linestyle="--")
        ax.set_xlabel("workers")
        ax.set_ylabel("throughput speed-up over one worker")
        ax.set_title(st + " (ML-DSA-44 + ML-KEM-768)")
        ax.set_xscale("log", base=2)
        ax.set_yscale("log", base=2)
        ax.set_xticks([1, 2, 4, 8])
        ax.set_xticklabels([1, 2, 4, 8])
        ax.grid(True)
    axs[0].legend(frameon=False, fontsize=6)
    fig.suptitle("F12  Concurrency scaling (E12; dashed = ideal; laptop 4 cores / 8 threads, VM 2 vCPUs, Mac 4 workers of 8 cores)", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F12_concurrency_scaling")


def f13():
    S = table("T11_E7_rejection_timing.csv")
    M = table("T10_E6_message_size.csv")
    fig, axs = plt.subplots(1, 2, figsize=(10.5, 3.4))
    ax = axs[0]
    vals = []
    for r in S:
        for col in ("badsig_over_valid", "wrongmsg_over_valid", "tampered_over_valid"):
            if r[col]:
                vals.append((family(r["alg"]), float(r[col])))
    for i, fam in enumerate(["ML-DSA", "Falcon", "SLH-DSA", "ML-KEM", "HQC", "classical"]):
        v = [x for f, x in vals if f == fam]
        if v:
            ax.scatter(np.full(len(v), i) + np.linspace(-0.2, 0.2, len(v)), v, s=8, color=FAMCOL[fam], linewidths=0)
    ax.axhline(1, color="#999999", linewidth=0.6)
    ax.set_xticks(range(6))
    ax.set_xticklabels(["ML-DSA", "Falcon", "SLH-DSA", "ML-KEM", "HQC", "classical"], rotation=30)
    ax.set_ylabel("time to reject / time to accept")
    ax.set_title("E7 (Mac): rejected input takes as long as a valid one")
    ax.grid(axis="y")
    ax = axs[1]
    for alg, c in (("ml-dsa-44", FAMCOL["ML-DSA"]), ("falcon-512", FAMCOL["Falcon"]), ("slh-dsa-sha2-128f", FAMCOL["SLH-DSA"]), ("ed25519", FAMCOL["classical"])):
        rows = sorted([r for r in M if r["alg"] == alg and r["op"] == "sign"], key=lambda r: int(r["message_bytes"]))
        ax.plot([int(r["message_bytes"]) for r in rows], [float(r["p50_us"]) for r in rows], color=c, marker="o", markersize=3, label=alg + " sign")
    ax.set_xscale("log")
    ax.set_yscale("log")
    ax.set_xlabel("message size, bytes")
    ax.set_ylabel("sign latency, microseconds")
    ax.set_title("E6 (Mac): signing cost against message size")
    ax.legend(frameon=False, fontsize=6)
    ax.grid(True)
    fig.suptitle("F13  Negative-path timing (E7) and message-size scaling (E6)", y=1.03, fontsize=9)
    fig.tight_layout()
    save(fig, "F13_negative_path_and_message_size")


def f0():
    Q = table("T0_run_quality.csv")
    fig, ax = plt.subplots(figsize=(7.2, 4.2))
    y = 0
    labels = []
    for mach in ("mac-m1", "win-ryzen", "vm-x86v2"):
        for r in [r for r in Q if r["machine"] == mach]:
            ax.barh(y, float(r["flagged_pct"]), color=COL[mach], height=0.7, linewidth=0)
            labels.append("%s %s%s" % (mach.split("-")[0], r["plan"], "" if r["variant"] == "b1-prod" else " " + r["variant"]))
            y += 1
    ax.set_yticks(range(len(labels)))
    ax.set_yticklabels(labels, fontsize=5.5)
    ax.invert_yaxis()
    ax.set_xlabel("attempts flagged for an external cause, % (all attempts of the run)")
    ax.set_title("F0  Run quality: share of attempts flagged, per run", fontsize=9)
    ax.grid(axis="x")
    save(fig, "F0_run_quality")


if __name__ == "__main__":
    for fn in (f0, f5, f6, f7, f8, f10, f11, f12, f13):
        fn()
    print(sorted(x for x in os.listdir(OUT_FIGS) if x.endswith(".png")))
