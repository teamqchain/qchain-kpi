"""Presentation tables for the paper and the reports: every experiment, CSV (to paste into Word) and Markdown (to read
on GitHub).

    results/combined/tables/          all three machines side by side, and the Mac-only experiments
    results/<machine>/tables/         one machine, with every statistic of each cell

Built from the analysis tables in results/combined/analysis_tables (T0-T14), from the E0 environment captures and,
for the latency tails, from the raw per-call files. Numbers are rounded for reading; the analysis tables keep the
full precision.
"""
import csv
import json
import os
import sys
from collections import defaultdict

import numpy as np

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa: E402,F401,F403
from e1 import family  # noqa: E402

ORDER = ["mac-m1", "win-ryzen", "vm-x86v2"]
NAME = {"mac-m1": "Mac M1", "win-ryzen": "Laptop", "vm-x86v2": "VM"}
LONG = {"mac-m1": "Mac M1 (Apple M1, ARM with NEON)", "win-ryzen": "Laptop (AMD Ryzen 5 7520U, x86 with AVX2)",
        "vm-x86v2": "VM (virtualised x86 server without AVX2)"}
FAMORDER = {"ML-DSA": 0, "Falcon": 1, "SLH-DSA": 2, "ML-KEM": 3, "HQC": 4, "classical": 5}
OPS_SIG = ["keygen", "sign", "verify"]
OPS_KEM = ["keygen", "encaps", "decaps"]
OPNAME = {"keygen": "key generation", "sign": "signing", "verify": "verification", "encaps": "encapsulation", "decaps": "decapsulation"}
INDEX = defaultdict(list)  # folder -> [(file base, title)]
BY_NAME = {NAME[m]: i for i, m in enumerate(ORDER)}  # sort key: machines in the fixed order
FIG_DESC = {"F1a": "Signature schemes: key generation", "F1b": "Signature schemes: signing", "F1c": "Signature schemes: verification",
            "F2a": "Key-encapsulation mechanisms: key generation", "F2b": "Key-encapsulation mechanisms: encapsulation",
            "F2c": "Key-encapsulation mechanisms: decapsulation", "F5a": "Credential lifecycle cost per suite",
            "F5b": "Credential lifecycle split by stage"}


def T(name):
    return read_csv(os.path.join(OUT_TABLES, name))


# ------------------------------------------------------------------------------------------------ cells and formats

class C:
    """One table cell: the value written to the CSV and the text written to the Markdown."""
    def __init__(self, value, text=None):
        self.value, self.text = value, (text if text is not None else ("" if value is None else str(value)))


def _nan(v):
    """Missing: None, an empty field (an arm that does not exist for this cell) or NaN."""
    return v is None or v == "" or (isinstance(v, float) and v != v)


def us(v):
    """Microseconds: 8.3, 60.2, 1,312."""
    if _nan(v):
        return C("", "–")
    v = float(v)
    if v < 100:
        return C(round(v, 2), "%.1f" % v)
    return C(round(v, 1), "{:,.0f}".format(v))


def ms(v):
    if _nan(v):
        return C("", "–")
    v = float(v)
    if v < 10:
        return C(round(v, 3), "%.2f" % v)
    if v < 1000:
        return C(round(v, 2), "%.1f" % v)
    return C(round(v, 1), "{:,.0f}".format(v))


def num(v, digits=2, sep=False):
    if _nan(v) or v == "":
        return C("", "–")
    v = float(v)
    return C(round(v, digits), ("{:,.%df}" % digits).format(v) if sep else ("%." + str(digits) + "f") % v)


def ratio(v, digits=2):
    if _nan(v) or v == "":
        return C("", "–")
    return C(round(float(v), digits + 1), ("%." + str(digits) + "f×") % float(v))


def pct(v, digits=1, signed=False):
    if _nan(v) or v == "":
        return C("", "–")
    return C(round(float(v), digits + 2), (("%+." if signed else "%.") + str(digits) + "f %%") % float(v))


def integer(v):
    if _nan(v) or v == "":
        return C("", "–")
    return C(int(round(float(v))), "{:,}".format(int(round(float(v)))))


def rng(values, digits=2, suffix="×"):
    """Median (min–max) of a list."""
    v = [float(x) for x in values if x != "" and not _nan(x)]
    if not v:
        return C("", "–")
    med = float(np.median(v))
    return C(round(med, digits + 1), ("%.{0}f%s (%.{0}f–%.{0}f)".format(digits)) % (med, suffix, min(v), max(v)))


def alg_key(a):
    return (FAMORDER[family(a)], a)


# ------------------------------------------------------------------------------------------------ writer

def emit(folder, base, title, note, header, rows, text_cols=1, source=None):
    """Write <base>.csv and <base>.md into folder. rows: lists of C or plain strings."""
    os.makedirs(folder, exist_ok=True)
    cells = [[x if isinstance(x, C) else C(x) for x in r] for r in rows]
    with open(os.path.join(folder, base + ".csv"), "w", newline="") as f:
        w = csv.writer(f, lineterminator="\n")
        w.writerow(header)
        for r in cells:
            w.writerow([c.value for c in r])
    md = ["# " + title, ""]
    if note:
        md += [note, ""]
    import re
    numeric = re.compile(r"^(–|[+\-]?[\d,]+(\.\d+)?( ?%|×)?( \*)?( \([\d,.]+[–-][\d,.]+\))?)$")
    align = []
    for k in range(len(header)):
        col = [r[k].text for r in cells if k < len(r)]
        right = k >= text_cols and col and all(numeric.match(t.strip()) for t in col)
        align.append("---:" if right else "---")
    md.append("| " + " | ".join(header) + " |")
    md.append("|" + "|".join(align) + "|")
    for r in cells:
        md.append("| " + " | ".join(c.text.replace("|", "\\|") for c in r) + " |")
    md.append("")
    md.append("Generated by `analysis/paper_tables.py`%s." % (" from `results/combined/analysis_tables/%s`" % source if source else ""))
    with open(os.path.join(folder, base + ".md"), "w") as f:
        f.write("\n".join(md) + "\n")
    INDEX[folder].append((base, title))


# ------------------------------------------------------------------------------------------------ quality and environment

def q1(folder, machines, suffix=""):
    rows = []
    for r in T("T0_run_quality.csv"):
        if r["machine"] not in machines:
            continue
        flags = ", ".join("%s %s" % (r["flag_" + k], k) for k in ("other", "canary", "thermal", "steal") if r["flag_" + k] not in ("", "0"))
        canary = "%s–%s" % (r["canary_min"], r["canary_max"]) if r["canary_min"] != "" else "–"
        rows.append([NAME[r["machine"]], plan_label(r["plan"], r["variant"]), r["run"], integer(r["blocks"]), integer(r["attempts"]),
                     pct(r["flagged_pct"]) if r["flagged_pct"] != "" else C("", "not applied"), flags or ("–" if r["flagged_pct"] == "" else "none"),
                     integer(r["blocks_without_headline"]) if r["blocks_without_headline"] != "" else C("", "–"),
                     integer(r["extension_sessions"]), canary, r["note"]])
    rows.sort(key=lambda x: BY_NAME[x[0]])
    emit(folder, "Q1_run_quality" + suffix, "Q1  Run quality of every main run" + (" (%s)" % LONG[machines[0]] if suffix else ""),
         "An attempt is flagged when something outside the benchmark disturbed it (other CPU use, a slow canary, heat, steal); "
         "a flagged block is repeated and only its first clean attempt, the headline, enters the results. "
         "Canary: the canary benchmark's rate relative to the reference.",
         ["Machine", "Experiment", "Run", "Blocks", "Attempts", "Flagged", "Flags by cause", "Blocks without a headline", "Extension sessions",
          "Canary range", "Note"], rows, text_cols=3, source="T0_run_quality.csv")


def q2(folder):
    rows = [[NAME[r["machine"]], plan_label(r["plan"]), integer(r["cells"]), integer(r["cells_with_extra_attempts"]),
             pct(r["median_abs_diff_pct"], 2), pct(r["p95_abs_diff_pct"], 2), pct(r["max_abs_diff_pct"], 2),
             integer(r["cells_over_1pct"]), integer(r["cells_over_3pct"])] for r in T("T14_sensitivity_headline_vs_all_attempts.csv")]
    rows.sort(key=lambda x: BY_NAME[x[0]])
    emit(folder, "Q2_sensitivity", "Q2  Sensitivity analysis: headline attempts against all attempts",
         "For every cell, the median over the headline attempts is compared with the median over all attempts (flagged ones included). "
         "Differences are absolute, in % of the headline median.",
         ["Machine", "Experiment", "Cells", "Cells with extra attempts", "Median difference", "95th percentile", "Largest", "Cells over 1 %", "Cells over 3 %"],
         rows, text_cols=2, source="T14_sensitivity_headline_vs_all_attempts.csv")


HOST = {"mac-m1": "MacBook Air (M1, 2020), 8 GB; macOS; Docker in Colima (Apple Virtualization framework)",
        "win-ryzen": "HP laptop, AMD Ryzen 5 7520U (4 cores, 8 threads, 15 W), 8 GB; Windows 11; Docker Desktop on WSL2",
        "vm-x86v2": "Proxmox VE (KVM) virtual machine, 2 vCPUs of type x86-64-v2-AES on an Intel Xeon Silver 4510 host, 30 GiB; Ubuntu 24.04; Docker"}


def env(machine):
    run = runs(MACHINE_DIR[machine], plan="e1", variant="b1-prod")[0]
    return json.load(open(os.path.join(run["dir"], "env", "e0-s1.json"))), run


def e00(folder, machines, suffix=""):
    E = {m: env(m) for m in machines}

    def feat(j, k):
        arm = j["kernel"]["machine"] == "aarch64"
        if arm != k.startswith("ARM_"):
            return "n/a"  # an x86 feature on the ARM machine, or the reverse
        return "yes" if j["liboqs"]["cpu_extensions_runtime"].get(k) else "no"

    def cpu(j):
        f = j["cpu"]["fields"]
        if "model name" in f:
            return f["model name"]
        return "Apple M1 core (ARMv%s, implementer %s)" % (f.get("CPU architecture", "?"), f.get("CPU implementer", "?"))

    spec = [
        ("Host", lambda j, m: HOST[m]),
        ("Processor seen by the container", lambda j, m: cpu(j)),
        ("Logical CPUs visible / used", lambda j, m: "%d / 1 (pinned, GOMAXPROCS=1)" % j["cpu"]["logical_cpus"]),
        ("AVX2 / BMI2 (x86)", lambda j, m: "%s / %s" % (feat(j, "AVX2"), feat(j, "BMI2"))),
        ("AES-NI / PCLMULQDQ (x86)", lambda j, m: "%s / %s" % (feat(j, "AES"), feat(j, "PCLMULQDQ"))),
        ("NEON / SHA-3 instructions (ARM)", lambda j, m: "%s / %s" % (feat(j, "ARM_NEON"), feat(j, "ARM_SHA3"))),
        ("Memory: Linux system / container limit", lambda j, m: "%.1f GiB / %.0f GiB, no swap" % (j["mem_total_kb"] / 1048576, int(j["cgroup"]["memory.max"]) / 2 ** 30)),
        ("Kernel", lambda j, m: j["kernel"]["release"]),
        ("Container libraries", lambda j, m: "glibc %s; %s" % (j["glibc"], j["openssl"])),
        ("Go", lambda j, m: j["harness"]["go"]),
        ("liboqs", lambda j, m: "%s (commit %s), build %s (%s)" % (j["liboqs"]["version"], j["liboqs"]["build"]["liboqs_commit"][:8],
                                                                j["liboqs"]["build"]["variant"], j["liboqs"]["build"]["variant_flags"])),
        ("Compiler of liboqs", lambda j, m: "gcc %s, %s" % (j["liboqs"]["build"]["gcc"], j["liboqs"]["build"]["cmake"])),
        ("Clock source", lambda j, m: j["clock"]["clocksource"]),
        ("Timer: effective resolution / time.Now overhead", lambda j, m: "%d ns / %d ns" % (j["clock"]["effective_resolution_ns"], j["clock"]["time_now_overhead_median_ns"])),
        ("Random bytes (32 B from crypto/rand), median", lambda j, m: "%d ns" % j["crypto_rand_read_32_bytes"]["median_ns"]),
        ("cgroup", lambda j, m: "v%s, cpu.max %s, cpuset %s" % (j["cgroup"]["version"], j["cgroup"]["cpu.max"], j["cgroup"]["cpuset.cpus.effective"])),
        ("Harness commit (E1 run)", lambda j, m: j["harness"]["commit"][:8]),
        ("E1 run", lambda j, m: E[m][1]["id"]),
    ]
    rows = [[name] + [fn(E[m][0], m) for m in machines] for name, fn in spec]
    emit(folder, "E00_environment" + suffix, "E00  Environment of the measurements (E0)" + (" (%s)" % LONG[machines[0]] if suffix else ""),
         "Captured by E0 at the start of every session of every run; shown here for the first session of each machine's E1 run. "
         "The host line comes from the machine descriptions (results/<machine>/MACHINE.md).",
         ["Item"] + [NAME[m] for m in machines], rows, text_cols=1 + len(machines))


# ------------------------------------------------------------------------------------------------ E1

def e1_tables(folder):
    T2 = T("T2_E1_primitive_latency.csv")
    K = {(r["machine"], r["alg"], r["op"]): r for r in T2}
    for base, ops, what in (("E01a_signature_latency", OPS_SIG, "signature schemes"), ("E01b_kem_latency", OPS_KEM, "key-encapsulation mechanisms")):
        algs = sorted({r["alg"] for r in T2 if r["op"] == ops[1]}, key=alg_key)
        rows = [[family(a), a] + [us(float(K[(m, a, op)]["p50_ns"]) / 1000) for op in ops for m in ORDER] for a in algs]
        emit(folder, base, "%s  %s: median latency per call (µs)" % (base.split("_")[0], what[0].upper() + what[1:]),
             "Median over five repetitions of each repetition's median (headline attempts), production build, one CPU. "
             "95 % intervals, p99, CoV and first-call costs are in the per-machine tables.",
             ["Family", "Algorithm"] + ["%s, %s" % (OPNAME[op][0].upper() + OPNAME[op][1:], NAME[m]) for op in ops for m in ORDER],
             rows, text_cols=2, source="T2_E1_primitive_latency.csv")
    algs = sorted({r["alg"] for r in T2}, key=alg_key)
    rows, rows_fc = [], []
    for a in algs:
        for op in [o for o in ("keygen", "sign", "verify", "encaps", "decaps") if (ORDER[0], a, o) in K]:
            rows.append([family(a), a, OPNAME[op]] + [us(float(K[(m, a, op)]["p99_ns"]) / 1000) for m in ORDER] +
                        [ratio(float(K[(m, a, op)]["p99_ns"]) / float(K[(m, a, op)]["p50_ns"]), 1) for m in ORDER])
            rows_fc.append([family(a), a, OPNAME[op]] + [us(float(K[(m, a, op)]["first_call_ns"]) / 1000) for m in ORDER] +
                           [us((float(K[(m, a, op)]["first_call_ns"]) - float(K[(m, a, op)]["p50_ns"])) / 1000) for m in ORDER])
    emit(folder, "E01c_tail_latency", "E01c  99th-percentile latency per call (µs), and p99 against the median",
         "p99 of the individual calls of the headline attempts (median over the repetitions).",
         ["Family", "Algorithm", "Operation"] + ["p99, %s" % NAME[m] for m in ORDER] + ["p99 ÷ median, %s" % NAME[m] for m in ORDER],
         rows, text_cols=3, source="T2_E1_primitive_latency.csv")
    emit(folder, "E01d_first_call", "E01d  The first call in a fresh process (µs)",
         "Every block is a new process; its first call is timed apart from the warm calls. Extra = first call − warm median.",
         ["Family", "Algorithm", "Operation"] + ["First call, %s" % NAME[m] for m in ORDER] + ["Extra, %s" % NAME[m] for m in ORDER],
         rows_fc, text_cols=3, source="T2_E1_primitive_latency.csv")
    R = T("T2b_E1_machine_ratios.csv")
    pairs = [("win-ryzen", "mac-m1"), ("vm-x86v2", "mac-m1"), ("vm-x86v2", "win-ryzen")]
    fams = sorted({r["family"] for r in R}, key=lambda f: FAMORDER[f])
    rows = []
    for fam in fams:
        row = [fam]
        for n, d in pairs:
            sel = [r for r in R if (r["numerator"], r["denominator"], r["family"]) == (n, d, fam)]
            row.append(rng([r["ratio"] for r in sel]))
            row.append("%d of %d" % (sum(1 for r in sel if abs(float(r["cliffs_delta"])) == 1), len(sel)))
        rows.append(row)
    emit(folder, "E01e_machine_ratios", "E01e  How much slower one machine is than another, by family (E1)",
         "Ratio of median latencies, family median (minimum–maximum over the family's operations). "
         "'Fully separated': operations where every repetition of one machine is slower than every repetition of the other (Cliff's delta of 1).",
         ["Family"] + [x for n, d in pairs for x in ("%s ÷ %s" % (NAME[n], NAME[d]), "Fully separated")], rows, text_cols=1,
         source="T2b_E1_machine_ratios.csv")
    import figures1
    rows = []
    for alg, op in figures1.TAIL_CASES:
        for m in ORDER:
            xs = figures1.raw_samples(MACHINE_DIR[m], "e1", "b1-prod", alg, op) / 1000
            q = np.percentile(xs, [50, 90, 99, 99.9])
            rows.append([alg, OPNAME[op], NAME[m], integer(len(xs))] + [us(x) for x in q] + [us(xs.max())])
    emit(folder, "E01f_latency_tails", "E01f  Latency tails of one repetition (µs), the data of figure F3",
         "Warm calls of repetition 1 (headline attempt; the first, cold call excluded).",
         ["Algorithm", "Operation", "Machine", "Calls", "p50", "p90", "p99", "p99.9", "Maximum"], rows, text_cols=3)
    sizes = {(r["alg"], r["item"]): float(r["median_bytes"]) for r in T("T1_E8_sizes.csv")}
    sigs = sorted({r["alg"] for r in T2 if r["op"] == "sign"}, key=alg_key)
    rows = [[family(a), a, integer(sizes[(a, "pk")] + sizes[(a, "sig")])] +
            [us((float(K[(m, a, "sign")]["p50_ns"]) + float(K[(m, a, "verify")]["p50_ns"])) / 1000) for m in ORDER] for a in sigs]
    emit(folder, "E01g_size_vs_speed", "E01g  Size against speed for the signature schemes, the data of figure F4",
         "Public key + signature (E8, median for variable-length signatures) against sign + verify (E1 medians).",
         ["Family", "Algorithm", "Public key + signature (bytes)"] + ["Sign + verify, %s (µs)" % NAME[m] for m in ORDER], rows, text_cols=2)


def e1_machine(folder, m):
    T2 = [r for r in T("T2_E1_primitive_latency.csv") if r["machine"] == m]
    for base, ops, what in (("E01a_signature_latency", OPS_SIG, "Signature schemes"), ("E01b_kem_latency", OPS_KEM, "Key-encapsulation mechanisms")):
        sel = sorted([r for r in T2 if r["op"] in ops and any(x["alg"] == r["alg"] and x["op"] == ops[1] for x in T2)],
                     key=lambda r: (alg_key(r["alg"]), ops.index(r["op"])))
        rows = [[family(r["alg"]), r["alg"], OPNAME[r["op"]], us(float(r["p50_ns"]) / 1000),
                 C("%s–%s" % (us(float(r["p50_ci_lo_ns"]) / 1000).value, us(float(r["p50_ci_hi_ns"]) / 1000).value),
                   "%s–%s" % (us(float(r["p50_ci_lo_ns"]) / 1000).text, us(float(r["p50_ci_hi_ns"]) / 1000).text)),
                 us(float(r["p99_ns"]) / 1000), us(float(r["mean_ns"]) / 1000), num(r["cov"], 2), us(float(r["first_call_ns"]) / 1000),
                 integer(r["reps"]), integer(r["attempts"])] for r in sel]
        emit(folder, "%s_%s" % (base, m), "%s  %s on the %s: latency per call (µs)" % (base.split("_")[0], what, LONG[m]),
             "Median over five repetitions (headline attempts) with its 95 % bootstrap interval over the repetitions; p99, mean and "
             "coefficient of variation of the individual calls; the first call of a fresh process.",
             ["Family", "Algorithm", "Operation", "Median", "95 % CI", "p99", "Mean", "CoV", "First call", "Repetitions", "Attempts"],
             rows, text_cols=3, source="T2_E1_primitive_latency.csv")


# ------------------------------------------------------------------------------------------------ E2-E7

def e2_tables(folder):
    E2 = T("T3_E2_integration_overhead.csv")
    fams = sorted({r["family"] for r in E2}, key=lambda f: FAMORDER[f])
    rows = []
    for fam in fams:
        row = [fam]
        for m in ORDER:
            sel = [r for r in E2 if r["machine"] == m and r["family"] == fam]
            row += [rng([r["prod_over_reuse"] for r in sel]), rng([r["reinit_over_reuse"] for r in sel])] if sel else [C("", "not run")] * 2
        rows.append(row)
    emit(folder, "E02a_integration_by_family", "E02a  What QChain's deployed pattern adds, by family (E2)",
         "Production pattern (keys as hex strings, decoded per call; a new library object per call) and re-initialisation alone, "
         "against a reused key, family median (minimum–maximum). SLH-DSA ran on the Mac only.",
         ["Family"] + [x for m in ORDER for x in ("Production ÷ reuse, %s" % NAME[m], "Re-init ÷ reuse, %s" % NAME[m])], rows,
         text_cols=1, source="T3_E2_integration_overhead.csv")
    K = {(r["machine"], r["alg"], r["op"]): r for r in E2}
    keys = sorted({(r["alg"], r["op"]) for r in E2}, key=lambda k: (alg_key(k[0]), k[1]))
    rows = []
    for a, op in keys:
        row = [family(a), a, OPNAME[op]]
        for m in ORDER:
            r = K.get((m, a, op))
            row += [us(r["reuse_us"]), us(r["prod_us"]), ratio(r["prod_over_reuse"])] if r else [C("", "–")] * 3
        rows.append(row)
    emit(folder, "E02b_integration_per_operation", "E02b  Reused key against the production pattern, every operation (µs)",
         "Medians over five repetitions. '–': not run on that machine (SLH-DSA on the laptop and the VM).",
         ["Family", "Algorithm", "Operation"] + [x for m in ORDER for x in ("Reuse, %s" % NAME[m], "Production, %s" % NAME[m], "Ratio, %s" % NAME[m])],
         rows, text_cols=3, source="T3_E2_integration_overhead.csv")


def e2_machine(folder, m):
    E2 = sorted([r for r in T("T3_E2_integration_overhead.csv") if r["machine"] == m], key=lambda r: (alg_key(r["alg"]), r["op"]))
    rows = [[r["family"], r["alg"], OPNAME[r["op"]], us(r["reuse_us"]), us(r["reinit_us"]), us(r["prod_us"]), ratio(r["prod_over_reuse"], 3),
             C("%s–%s" % (r["ci_lo"], r["ci_hi"]), "%.3f–%.3f" % (float(r["ci_lo"]), float(r["ci_hi"]))), ratio(r["reinit_over_reuse"], 3),
             us(r["init_us"]), us(r["hex_us"])] for r in E2]
    emit(folder, "E02_integration_%s" % m, "E02  QChain's deployed pattern against a reused key on the %s (µs)" % LONG[m],
         "Reuse: one key, one library object. Re-init: a new library object per call. Production: QChain's code (hex keys decoded per call). "
         "Components: library-object initialisation and hex conversion, timed alone.",
         ["Family", "Algorithm", "Operation", "Reuse", "Re-init", "Production", "Production ÷ reuse", "95 % CI", "Re-init ÷ reuse",
          "Initialisation", "Hex conversion"], rows, text_cols=3, source="T3_E2_integration_overhead.csv")


def e3_table(folder):
    rows = [[r["family"], r["alg"], OPNAME[r["op"]], us(r["l0_c_us"]), us(r["l0g_go_to_c_us"]), us(r["l1_binding_us"]),
             us(r["l2_go_stdlib_us"]) if r["l2_go_stdlib_us"] else C("", "–"), num(r["boundary_ns"], 0), num(r["binding_ns"], 0),
             ratio(r["l1_over_l0"], 3), ratio(r["l2_over_l1"], 2) if r["l2_over_l1"] else C("", "–")]
            for r in sorted(T("T8_E3_integration_layers.csv"), key=lambda r: (alg_key(r["alg"]), r["op"]))]
    emit(folder, "E03_integration_layers", "E03  Integration layers on the Mac (E3): C library, Go-to-C call, Go binding, Go's own library",
         "Medians (µs) over five repetitions. Boundary: the empty Go-to-C call; binding: the Go binding's own work (ns). "
         "Go's own library exists for the classical algorithms only.",
         ["Family", "Algorithm", "Operation", "C library", "Go → C", "Go binding", "Go standard library", "Boundary (ns)", "Binding (ns)",
          "Binding ÷ C", "Go library ÷ binding"], rows, text_cols=3, source="T8_E3_integration_layers.csv")


E4_PANELS = [("mac-m1", "b2-portable"), ("mac-m1", "b3-native"), ("win-ryzen", "b2-portable"), ("win-ryzen", "b3-native"), ("vm-x86v2", "b4-noossl")]
VSHORT = {"b2-portable": "portable (b2)", "b3-native": "native (b3)", "b4-noossl": "no OpenSSL (b4)"}


def e4_tables(folder):
    B = T("T7_E4_build_sensitivity.csv")
    fams = sorted({r["family"] for r in B}, key=lambda f: FAMORDER[f])
    rows = [[fam] + [rng([r["ratio_variant_over_b1"] for r in B if (r["machine"], r["variant"], r["family"]) == (m, v, fam)]) for m, v in E4_PANELS]
            for fam in fams]
    emit(folder, "E04a_build_by_family", "E04a  liboqs build variants against the production build, by family (E4)",
         "Variant ÷ production build (b1), family median (minimum–maximum). b2: generic target, no hand-written vector code; "
         "b3: compiled for the machine's CPU; b4: b1 without OpenSSL.",
         ["Family"] + ["%s, %s" % (NAME[m], VSHORT[v]) for m, v in E4_PANELS], rows, text_cols=1, source="T7_E4_build_sensitivity.csv")
    K = {(r["machine"], r["variant"], r["alg"], r["op"]): r for r in B}
    keys = sorted({(r["alg"], r["op"]) for r in B}, key=lambda k: (alg_key(k[0]), k[1]))
    rows = [[family(a), a, OPNAME[op]] + [ratio(K[(m, v, a, op)]["ratio_variant_over_b1"]) if (m, v, a, op) in K else C("", "–") for m, v in E4_PANELS]
            for a, op in keys]
    emit(folder, "E04b_build_per_operation", "E04b  liboqs build variants against the production build, every operation (E4)",
         "Variant ÷ production build (b1), ratio of medians.", ["Family", "Algorithm", "Operation"] + ["%s, %s" % (NAME[m], VSHORT[v]) for m, v in E4_PANELS],
         rows, text_cols=3, source="T7_E4_build_sensitivity.csv")


def e4_machine(folder, m):
    B = sorted([r for r in T("T7_E4_build_sensitivity.csv") if r["machine"] == m], key=lambda r: (r["variant"], alg_key(r["alg"]), r["op"]))
    rows = [[VSHORT[r["variant"]], r["family"], r["alg"], OPNAME[r["op"]], us(r["b1_p50_us"]), us(r["variant_p50_us"]), ratio(r["ratio_variant_over_b1"], 3),
             C("%s–%s" % (r["ci_lo"], r["ci_hi"]), "%.3f–%.3f" % (float(r["ci_lo"]), float(r["ci_hi"]))), num(r["cliffs_delta"], 2)] for r in B]
    emit(folder, "E04_build_%s" % m, "E04  liboqs build variants on the %s (µs)" % LONG[m],
         "Production build (b1) against the variant(s) this machine ran; medians over five repetitions, 95 % interval of the ratio, Cliff's delta.",
         ["Variant", "Family", "Algorithm", "Operation", "Production (b1)", "Variant", "Variant ÷ b1", "95 % CI", "Cliff's delta"],
         rows, text_cols=4, source="T7_E4_build_sensitivity.csv")


def e5_tables(folder):
    N = T("T9_E5_native_vs_container.csv")
    fams = sorted({r["family"] for r in N}, key=lambda f: FAMORDER[f])
    rows = [[fam, rng([r["native_over_container"] for r in N if r["family"] == fam]), integer(sum(1 for r in N if r["family"] == fam))] for fam in fams]
    emit(folder, "E05a_native_by_family", "E05a  Native macOS against the Linux container on the Mac, by family (E5)",
         "Native ÷ container, family median (minimum–maximum); under 1× the native build is faster. "
         "Native: Apple clang; container: gcc 12.", ["Family", "Native ÷ container", "Operations"], rows, text_cols=1,
         source="T9_E5_native_vs_container.csv")
    rows = [[r["family"], r["alg"], OPNAME[r["op"]], us(r["container_p50_us"]), us(r["native_p50_us"]), ratio(r["native_over_container"], 3),
             C("%s–%s" % (r["ci_lo"], r["ci_hi"]), "%.3f–%.3f" % (float(r["ci_lo"]), float(r["ci_hi"])))]
            for r in sorted(N, key=lambda r: (alg_key(r["alg"]), r["op"]))]
    emit(folder, "E05b_native_per_operation", "E05b  Native macOS against the container, every operation (µs)",
         "Medians over five repetitions; 95 % interval of the ratio.",
         ["Family", "Algorithm", "Operation", "Container", "Native", "Native ÷ container", "95 % CI"], rows, text_cols=3,
         source="T9_E5_native_vs_container.csv")


def e6_table(folder):
    M = T("T10_E6_message_size.csv")
    sizes = sorted({int(r["message_bytes"]) for r in M})
    sname = {64: "64 B", 1024: "1 KiB", 10240: "10 KiB", 102400: "100 KiB", 1048576: "1 MiB"}
    K = {(r["alg"], r["op"], int(r["message_bytes"])): r for r in M}
    keys = sorted({(r["alg"], r["op"]) for r in M}, key=lambda k: (alg_key(k[0]), k[1]))
    rows = [[family(a), a, OPNAME[op]] + [us(K[(a, op, s)]["p50_us"]) for s in sizes] + [num(K[(a, op, sizes[-1])]["throughput_MB_s"], 0)] for a, op in keys]
    emit(folder, "E06_message_size", "E06  Signing and verification time against message size on the Mac (µs)",
         "Medians over five repetitions; throughput at 1 MiB in MB/s. QChain signs a 64-character hash, so only the first column applies to it.",
         ["Family", "Algorithm", "Operation"] + [sname.get(s, "%d B" % s) for s in sizes] + ["MB/s at 1 MiB"], rows, text_cols=3,
         source="T10_E6_message_size.csv")


def e7_table(folder):
    rows = [[r["family"], r["alg"], OPNAME[r["op"]], us(r["valid_p50_us"]), ratio(r["badsig_over_valid"], 3), ratio(r["wrongmsg_over_valid"], 3),
             ratio(r["tampered_over_valid"], 3)] for r in sorted(T("T11_E7_rejection_timing.csv"), key=lambda r: alg_key(r["alg"]))]
    emit(folder, "E07_rejection_timing", "E07  Time to reject an invalid input against the time to accept a valid one, Mac (E7)",
         "Ratio of medians. Signature schemes: a corrupted signature and a signature over a different message; KEMs: a tampered ciphertext "
         "(DHKEM: a tampered public point, refused before any computation). '–': the test does not apply.",
         ["Family", "Algorithm", "Operation", "Valid input (µs)", "Corrupted signature", "Different message", "Tampered ciphertext"],
         rows, text_cols=3, source="T11_E7_rejection_timing.csv")


def e8_table(folder):
    S = T("T1_E8_sizes.csv")
    K = {(r["alg"], r["item"]): r for r in S}

    def sz(a, item):
        r = K.get((a, item))
        if not r:
            return C("", "–")
        if r["min_bytes"] == r["max_bytes"]:
            return integer(r["median_bytes"])
        return C(int(r["median_bytes"]), "{:,} ({:,}–{:,})".format(int(r["median_bytes"]), int(r["min_bytes"]), int(r["max_bytes"])))
    algs = sorted({r["alg"] for r in S}, key=alg_key)
    rows = [[family(a), a, sz(a, "pk"), sz(a, "sk"), sz(a, "sig") if (a, "sig") in K else sz(a, "ct"), sz(a, "ss")] for a in algs]
    emit(folder, "E08_sizes", "E08  Sizes of keys, signatures, ciphertexts and shared secrets (bytes)",
         "Measured on 1,000 objects each. Variable-length items: median (minimum–maximum). Classical keys in their usual encodings "
         "(uncompressed P-256 points, raw Ed25519 and X25519 keys, DER RSA).",
         ["Family", "Algorithm", "Public key", "Secret key", "Signature or ciphertext", "Shared secret"], rows, text_cols=2, source="T1_E8_sizes.csv")


# ------------------------------------------------------------------------------------------------ E9-E14

def e9_rows(machines):
    M = T("T4_E9_memory.csv")
    K = {(r["machine"], r["alg"], r["op"], r["route"], r["quantity"]): r for r in M}
    keys = sorted({(r["alg"], r["op"]) for r in M}, key=lambda k: (alg_key(k[0]), k[1]))

    def g(m, a, op, route, q, f=1.0, digits=1):
        r = K.get((m, a, op, route, q))
        return num(float(r["median"]) / f, digits, sep=True) if r else C("", "–")
    rows = []
    for a, op in keys:
        rows.append([family(a), a, OPNAME[op]] + [g(m, a, op, "c", "stack", 1024) for m in machines] +
                    [g(machines[0], a, op, "c", "heap-peak", 1, 0), g(machines[0], a, op, "c", "heap-bytes", 1, 0),
                     g(machines[0], a, op, "prod", "go-bytes", 1, 0), g(machines[0], a, op, "prod", "go-allocs", 1, 0)] +
                    [g(m, a, op, "c", "rss-peak", 2 ** 20, 1) for m in machines])
    return rows


def e9_tables(folder, machines, suffix=""):
    one = len(machines) == 1
    emit(folder, "E09_memory" + suffix, "E09  Memory per call (E9)" + (" on the %s" % LONG[machines[0]] if one else ""),
         "Stack high-water mark (KiB) and C heap of one call through the C library; Go allocation of one call through QChain's production "
         "code; peak resident memory of the process (MiB). Heap and Go figures are identical on the three machines%s." %
         ("" if one else " (shown once, Mac)"),
         ["Family", "Algorithm", "Operation"] + ["Stack (KiB)%s" % ("" if one else ", " + NAME[m]) for m in machines] +
         ["C heap peak (B)", "C heap allocated (B)", "Go bytes (production)", "Go allocations (production)"] +
         ["Resident peak (MiB)%s" % ("" if one else ", " + NAME[m]) for m in machines], e9_rows(machines), text_cols=3, source="T4_E9_memory.csv")


def e10_table(folder, machines, suffix=""):
    rows = []
    for r in sorted(T("T6_E10_issuer_holder_equivalence.csv"), key=lambda r: (int(r["alg"].split("-")[-1]), ORDER.index(r["machine"]))):
        if r["machine"] not in machines:
            continue
        rows.append([NAME[r["machine"]], r["alg"].upper(), us(r["issuer_p50_us"]), us(r["holder_p50_us"]), pct(float(r["holder_vs_issuer"]) * 100, 2, True),
                     C("%s–%s" % (r["tost90_lo"], r["tost90_hi"]), "%+.2f to %+.2f %%" % (float(r["tost90_lo"]) * 100, float(r["tost90_hi"]) * 100)),
                     C(r["margin"], "±%.2f %%" % (float(r["margin"]) * 100)), "yes" if r["equivalent"] == "True" else "no",
                     pct(float(r["aa_issuer_again_vs_issuer"]) * 100, 2, True)])
    emit(folder, "E10_issuer_holder" + suffix, "E10  Issuer-side against holder-side ML-DSA signing: equivalence test" +
         (" (%s)" % LONG[machines[0]] if suffix else ""),
         "Two one-sided tests on the paired per-repetition log-ratio: equivalent when the 90 % interval lies inside the machine's margin. "
         "A/A: the issuer measured again against itself (the instrument's own noise).",
         ["Machine", "ML-DSA set", "Issuer (µs)", "Holder (µs)", "Holder vs issuer", "90 % CI", "Margin", "Equivalent", "A/A difference"],
         rows, text_cols=2, source="T6_E10_issuer_holder_equivalence.csv")


def curated():
    lap = {r["suite"]: r for r in T("T3b_E11_lifecycle_per_suite.csv") if r["machine"] == "win-ryzen"}
    return sorted(lap, key=lambda s: (float(lap[s]["lifecycle_total_us"]), s))


STAGES = ["activation", "issuance", "retrieval", "presentation", "verification"]


def e11_tables(folder):
    L = {(r["machine"], r["suite"]): r for r in T("T3b_E11_lifecycle_per_suite.csv")}
    suites = curated()
    rows = [[s] + [ms(float(L[(m, s)]["lifecycle_total_us"]) / 1000) for m in ORDER] + [ratio(L[(m, s)]["total_vs_ecdsa_p256_dhkem_p256"]) for m in ORDER]
            for s in suites]
    emit(folder, "E11a_lifecycle_total", "E11a  Credential lifecycle cost of the 16 curated suites (ms)",
         "Sum of the five stages' medians (activation, issuance, retrieval, presentation, verification), 10 attributes of 16 bytes, 5 disclosed; "
         "and the same against the classical suite ECDSA P-256 + DHKEM(P-256).",
         ["Suite"] + ["Total, %s" % NAME[m] for m in ORDER] + ["vs ECDSA + ECDH P-256, %s" % NAME[m] for m in ORDER], rows, text_cols=1,
         source="T3b_E11_lifecycle_per_suite.csv")
    rows = [[s, NAME[m]] + [ms(float(L[(m, s)][st + "_us"]) / 1000) for st in STAGES] + [ms(float(L[(m, s)]["lifecycle_total_us"]) / 1000)]
            for s in suites for m in ORDER]
    emit(folder, "E11b_lifecycle_stages", "E11b  Credential lifecycle by stage, 16 curated suites, three machines (ms)",
         "Median of each stage; 10 attributes of 16 bytes, 5 disclosed.", ["Suite", "Machine"] + [st[0].upper() + st[1:] for st in STAGES] + ["Total"],
         rows, text_cols=2, source="T3b_E11_lifecycle_per_suite.csv")
    S = T("T3c_E11_scaling_production_suite.csv")
    K = {(r["machine"], r["stage"], int(r["attributes_n"]), int(r["value_bytes"])): r for r in S}
    keys = sorted({(r["stage"], int(r["value_bytes"]), int(r["attributes_n"])) for r in S})
    rows = [[st, integer(n), integer(v)] + [us(K[(m, st, n, v)]["p50_us"]) for m in ORDER] for st, v, n in keys]
    emit(folder, "E11c_scaling", "E11c  Production suite (ML-DSA-44 + ML-KEM-768): issuance and verification against attributes and value size (µs)",
         "Median over five repetitions; verification discloses half of the attributes.", ["Stage", "Attributes", "Value bytes"] + [NAME[m] for m in ORDER],
         rows, text_cols=1, source="T3c_E11_scaling_production_suite.csv")
    mac = {r["suite"]: r for r in T("T3b_E11_lifecycle_per_suite.csv") if r["machine"] == "mac-m1"}
    sigs = sorted({s.split("+")[0] for s in mac}, key=alg_key)
    kems = sorted({s.split("+")[1] for s in mac}, key=alg_key)
    rows = [[sg] + [ms(float(mac[sg + "+" + k]["lifecycle_total_us"]) / 1000) if sg + "+" + k in mac else C("", "–") for k in kems] for sg in sigs]
    emit(folder, "E11d_all_pairs", "E11d  Lifecycle cost of every signature × KEM pair on the Mac (ms), the data of figure F6",
         "Sum of the five stages' medians; 10 attributes, 5 disclosed.", ["Signature scheme"] + kems, rows, text_cols=1,
         source="T3b_E11_lifecycle_per_suite.csv")


def e11_machine(folder, m):
    L = {r["suite"]: r for r in T("T3b_E11_lifecycle_per_suite.csv") if r["machine"] == m}
    rows = [[s] + [ms(float(L[s][st + "_us"]) / 1000) for st in STAGES] + [ms(float(L[s]["lifecycle_total_us"]) / 1000),
            C("%s–%s" % (L[s]["total_ci_lo_us"], L[s]["total_ci_hi_us"]), "%s–%s" % (ms(float(L[s]["total_ci_lo_us"]) / 1000).text, ms(float(L[s]["total_ci_hi_us"]) / 1000).text)),
            ratio(L[s]["total_vs_ecdsa_p256_dhkem_p256"])] for s in curated()]
    emit(folder, "E11a_lifecycle_%s" % m, "E11a  Credential lifecycle of the 16 curated suites on the %s (ms)" % LONG[m],
         "Median of each stage, their sum with its 95 % interval, and the sum against the classical suite ECDSA P-256 + DHKEM(P-256); "
         "10 attributes of 16 bytes, 5 disclosed.", ["Suite"] + [st[0].upper() + st[1:] for st in STAGES] + ["Total", "95 % CI", "vs classical"],
         rows, text_cols=1, source="T3b_E11_lifecycle_per_suite.csv")
    S = sorted([r for r in T("T3c_E11_scaling_production_suite.csv") if r["machine"] == m], key=lambda r: (r["stage"], int(r["value_bytes"]), int(r["attributes_n"])))
    rows = [[r["stage"], integer(r["attributes_n"]), integer(r["value_bytes"]), us(r["p50_us"]),
             C("%s–%s" % (r["ci_lo_us"], r["ci_hi_us"]), "%s–%s" % (us(r["ci_lo_us"]).text, us(r["ci_hi_us"]).text))] for r in S]
    emit(folder, "E11c_scaling_%s" % m, "E11c  Production suite scaling on the %s (µs)" % LONG[m], "Median over five repetitions with its 95 % interval.",
         ["Stage", "Attributes", "Value bytes", "Median", "95 % CI"], rows, text_cols=1, source="T3c_E11_scaling_production_suite.csv")


def e12_tables(folder, machines, suffix=""):
    C12 = T("T12_E12_concurrency.csv")
    K = {(r["machine"], r["suite"], r["stage"], int(r["workers"])): r for r in C12}
    keys = sorted({(r["suite"], r["stage"], int(r["workers"])) for r in C12 if r["machine"] in machines})
    rows = []
    for su, st, w in keys:
        row = [su, st.split("-")[0], integer(w)]
        for m in machines:
            r = K.get((m, su, st, w))
            if r:
                mark = "" if r["basis"] == "headline" else " *"
                row += [C(round(float(r["throughput_per_s"]), 1), "{:,.0f}{}".format(float(r["throughput_per_s"]), mark)), ratio(r["speedup_vs_1"]),
                        num(r["efficiency"], 2), us(r["latency_p99_us"])]
            else:
                row += [C("", "–")] * 4
        rows.append(row)
    emit(folder, "E12_concurrency" + suffix, "E12  Concurrency: throughput against the number of workers" + (" (%s)" % LONG[machines[0]] if suffix else ""),
         "Calls per second of one stage (10 attributes; verification discloses 5), median over five repetitions; speed-up and efficiency against "
         "one worker; p99 latency of one call (µs). * : no attempt passed the quality rules (laptop, four workers): all attempts used "
         "(sensitivity analysis). '–': not run on that machine.",
         ["Suite", "Stage", "Workers"] + [x for m in machines for x in ("Calls/s%s" % ("" if suffix else ", " + NAME[m]), "Speed-up", "Efficiency", "p99 (µs)")],
         rows, text_cols=2, source="T12_E12_concurrency.csv")


def e13_tables(folder):
    L = T("T5_E13_ledger_issue_credential.csv")
    K = {(r["sig_scheme"], r["scenario"], int(r["attributes_n"]), r["quantity"]): r for r in L}
    sigs = sorted({r["sig_scheme"] for r in L}, key=lambda a: (float(K[(a, "measured", 10, "envelope")]["median"]), a))
    ns = sorted({int(r["attributes_n"]) for r in L})
    rows = [[family(a), a] + [integer(K[(a, "measured", n, "envelope")]["median"]) for n in ns] +
            [integer(K[(a, "measured", 10, "per_block_max")]["median"]), integer(K[(a, "measured", 10, "share_max")]["median"]),
             num(float(K[(a, "measured", 10, "envelope")]["median"]) * 1e6 / 1e9, 1)] for a in sigs]
    emit(folder, "E13a_ledger_issue_credential", "E13a  Ledger bytes of one issueCredential transaction, by the issuer's signature scheme (E13)",
         "The model calibrated on a block captured from QChain's network (Fabric identities ECDSA P-256; two endorsements); attribute values of 16 bytes "
         "(the values themselves are on IPFS, the ledger holds their hashes). Variable-length signatures: median of 1,000. "
         "Per block: transactions that fit when they arrive faster than the 2 s batch timeout; share: one transaction's share of such a block.",
         ["Family", "Signature scheme"] + ["n = %d" % n for n in ns] + ["Per block (n = 10)", "Share of a block (n = 10)", "GB per million (n = 10)"],
         rows, text_cols=2, source="T5_E13_ledger_issue_credential.csv")
    E = T("T5c_E13_ledger_every_transaction.csv")
    KE = {(r["sig_scheme"], r["tx"], r["kem"], int(r["attributes_n"]), r["quantity"]): r for r in E}
    txs = [("registerHolder", "", 0), ("bindHolderKeys", "ml-kem-768", 0), ("issueCredential", "", 10), ("updateExpiry", "", 10),
           ("suspendCredential", "", 10), ("restoreCredential", "", 10), ("revokeCredential", "", 10)]
    rows = [[family(a), a] + [integer(KE[(a, tx, k, n, "envelope")]["median"]) if (a, tx, k, n, "envelope") in KE else C("", "–") for tx, k, n in txs] for a in sigs]
    emit(folder, "E13b_ledger_every_transaction", "E13b  Ledger bytes of every QChain transaction (E13)",
         "Envelope bytes as stored in the block; credential transactions with 10 attributes; the holder's keys bound with ML-KEM-768 "
         "(other KEMs in E13e). The state changes rewrite the whole credential record.",
         ["Family", "Signature scheme", "registerHolder", "bindHolderKeys", "issueCredential", "updateExpiry", "suspendCredential",
          "restoreCredential", "revokeCredential"], rows, text_cols=2, source="T5c_E13_ledger_every_transaction.csv")
    kems = sorted({r["kem"] for r in E if r["tx"] == "bindHolderKeys"}, key=alg_key)
    rows = [[family(k), k, integer(KE[("ml-dsa-44", "bindHolderKeys", k, 0, "envelope")]["median"]),
             integer(KE[("ml-dsa-44", "bindHolderKeys", k, 0, "record")]["median"])] for k in kems]
    emit(folder, "E13e_holder_key_binding", "E13e  Binding the holder's keys: ledger bytes by KEM (holder signature key ML-DSA-44)",
         "bindHolderKeys stores the holder's KEM public key and signature public key on the ledger.",
         ["Family", "KEM", "Transaction (envelope)", "Stored record"], rows, text_cols=2, source="T5c_E13_ledger_every_transaction.csv")
    B = T("T5b_E13_ipfs_envelope_bytes.csv")
    KB = {(r["kem"], int(r["attributes_n"]), int(r["value_bytes"])): int(r["envelope_bytes"]) for r in B}
    kems = sorted({r["kem"] for r in B}, key=alg_key)
    nsb = sorted({int(r["attributes_n"]) for r in B})
    vs = sorted({int(r["value_bytes"]) for r in B})
    rows = [[family(k), k, integer(v)] + [integer(KB[(k, n, v)]) for n in nsb] for k in kems for v in vs]
    emit(folder, "E13c_ipfs_envelope", "E13c  Size of the encrypted credential envelope on IPFS (bytes)",
         "File size of the envelope sealed by QChain's code for each KEM, by number of attributes and attribute value size.",
         ["Family", "KEM", "Value bytes"] + ["n = %d" % n for n in nsb], rows, text_cols=2, source="T5b_E13_ipfs_envelope_bytes.csv")
    rows = []
    for a in sigs:
        base = float(K[(a, "measured", 10, "envelope")]["median"])
        row = [family(a), a, integer(base)]
        for sc in ("whatif-base64", "whatif-keyref", "whatif-both"):
            v = float(K[(a, sc, 10, "envelope")]["median"])
            row += [integer(v), pct((v - base) / base * 100, 1, True)]
        rows.append(row)
    emit(folder, "E13d_encoding_whatif", "E13d  How much of a credential transaction is encoding: three hypothetical encodings (E13, n = 10)",
         "As stored by QChain (hex) against: signature and keys in base64; the issuer's public key replaced by a 64-hex fingerprint; both.",
         ["Family", "Signature scheme", "As stored (hex)", "Base64", "Change", "Key fingerprint", "Change", "Both", "Change"], rows, text_cols=2,
         source="T5_E13_ledger_issue_credential.csv")


def e14_table(folder):
    order_ = {"sequential": 0, "burst": 1, "drift": 2}
    R = sorted(T("T13_E14_fabric_latency.csv"), key=lambda r: (order_[r["mode"]], float(r["envelope_bytes"])))
    rows = [[r["point"], {"sequential": "one at a time", "burst": "bursts of 40", "drift": "drift control"}[r["mode"]], integer(r["payload_bytes"]),
             integer(r["envelope_bytes"]), integer(r["transactions"]), ms(r["total_p50_ms"]), ms(r["total_p99_ms"]), ms(r["endorse_p50_ms"]),
             ms(r["commit_p50_ms"]), integer(r["mvcc_conflicts"])] for r in R]
    emit(folder, "E14_fabric_latency", "E14  Fabric transaction latency against payload size, Mac (ms)",
         "QChain's Fabric network (two organisations, one Raft orderer, ECDSA identities, one CouchDB shared by both peers) rebuilt in Docker; "
         "payloads sized like each scheme's credential transaction (E13, 5 attributes). Alone, a transaction waits for the orderer's 2 s batch timeout. "
         "MVCC: transactions flagged MVCC_READ_CONFLICT (the shared CouchDB; their records were written).",
         ["Payload like", "Mode", "Payload (B)", "Envelope (B)", "Transactions", "Total, median", "Total, p99", "Endorsement, median", "Commit, median", "MVCC conflicts"],
         rows, text_cols=2, source="T13_E14_fabric_latency.csv")


# ------------------------------------------------------------------------------------------------ indexes

LOGO = {"combined": "../../../.github/assets/qchain_logo.jpeg", "machine": "../../.github/assets/qchain_logo.jpeg"}


def write_indexes():
    folder = OUT_PAPER
    lines = ['<p align="center">', '  <img src="%s" width="120" alt="QChain logo" />' % LOGO["combined"], "</p>", "",
             '<h1 align="center">Paper tables</h1>', "", '<p align="center">',
             "  <b>Every experiment as a table to read or paste: the three machines side by side, and the Mac-only experiments.</b><br/>",
             '  <a href="../README.md">back to the combined results</a> · <a href="../../../README.md">main README</a>', "</p>", "", "---", "",
             "> ⚠️ **Generated, never edited by hand.** `python3 analysis/paper_tables.py` (run by `analysis/run_all.py`) writes every file here.",
             "> Each table exists twice: `.csv` (numbers only, to paste into Word or a spreadsheet) and `.md` (formatted, readable on GitHub).",
             "> Full precision is in [`../analysis_tables/`](../analysis_tables/); one machine's tables with every statistic are in `results/<machine>/tables/`.",
             "", "| Table | Content |", "|---|---|"]
    for base, title in sorted(INDEX[folder]):
        lines.append("| [`%s`](%s.md) · [csv](%s.csv) | %s |" % (base, base, base, title.split("  ", 1)[1]))
    lines.append("")
    with open(os.path.join(folder, "README.md"), "w") as f:
        f.write("\n".join(lines))
    for m in ORDER:
        mdir = os.path.dirname(machine_out(m, "tables"))
        figs = sorted({os.path.splitext(x)[0] for x in os.listdir(machine_out(m, "figures"))}) if os.path.isdir(machine_out(m, "figures")) else []
        lines = ['<p align="center">', '  <img src="%s" width="120" alt="QChain logo" />' % LOGO["machine"], "</p>", "",
                 '<h1 align="center">%s</h1>' % LONG[m], "", '<p align="center">',
                 "  <b>The runs of this machine, and its own figures and tables.</b><br/>",
                 '  <a href="../combined/README.md">combined results</a> · <a href="../../README.md">main README</a>', "</p>", "", "---", "",
                 "| | |", "|---|---|",
                 "| [MACHINE.md](MACHINE.md) | the machine, its set-up and how it is described in the paper |",
                 "| [INDEX.md](INDEX.md) | every run folder of this machine (one per run, every sample, gzipped, with checksums) |",
                 "| [`figures/`](figures/) | this machine's bar charts (PDF and PNG), listed below |",
                 "| [`tables/`](tables/) | this machine's tables (CSV and Markdown), listed below |", "",
                 "> ⚠️ `figures/` and `tables/` are generated by `python3 analysis/run_all.py`; never edit them by hand. The figures and tables that compare",
                 "> the three machines are in [`../combined/`](../combined/README.md).", "", "## Figures", "", "| Figure | Files |", "|---|---|"]
        for fb in figs:
            lines.append("| %s  %s | [PNG](figures/%s.png) · [PDF](figures/%s.pdf) |" % (fb.split("_")[0], FIG_DESC.get(fb.split("_")[0], ""), fb, fb))
        lines += ["", "## Tables", "", "| Table | Content |", "|---|---|"]
        for base, title in sorted(INDEX[machine_out(m, "tables")]):
            lines.append("| [`%s`](tables/%s.md) · [csv](tables/%s.csv) | %s |" % (base, base, base, title.split("  ", 1)[1]))
        lines.append("")
        with open(os.path.join(mdir, "README.md"), "w") as f:
            f.write("\n".join(lines))


def main():
    INDEX.clear()
    out = OUT_PAPER
    q1(out, ORDER)
    q2(out)
    e00(out, ORDER)
    e1_tables(out)
    e2_tables(out)
    e3_table(out)
    e4_tables(out)
    e5_tables(out)
    e6_table(out)
    e7_table(out)
    e8_table(out)
    e9_tables(out, ORDER)
    e10_table(out, ORDER)
    e11_tables(out)
    e12_tables(out, ORDER)
    e13_tables(out)
    e14_table(out)
    for m in ORDER:
        d = machine_out(m, "tables")
        sfx = "_" + m
        q1(d, [m], sfx)
        e00(d, [m], sfx)
        e1_machine(d, m)
        e2_machine(d, m)
        if m in {r["machine"] for r in T("T7_E4_build_sensitivity.csv")}:
            e4_machine(d, m)
        e9_tables(d, [m], sfx)
        e10_table(d, [m], sfx)
        e11_machine(d, m)
        e12_tables(d, [m], sfx)
    write_indexes()
    return {k: len(v) for k, v in INDEX.items()}


if __name__ == "__main__":
    for k, v in main().items():
        print(v, "tables in", os.path.relpath(k, ROOT))
