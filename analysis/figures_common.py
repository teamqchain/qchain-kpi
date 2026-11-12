"""Figure style, colours, axis helpers and the layout check shared by the figure scripts (matplotlib).

Rules every figure follows:
- The three machines keep one colour everywhere (Okabe-Ito, colour-blind safe) and are always listed in the same
  order: Mac blue, laptop bluish green, VM vermillion (orange). No other quantity uses these three hues: algorithm
  families, lifecycle stages and size series have colours of their own.
- Every logarithmic axis says "(log scale)" in its label, and its ticks are written in plain units
  (1 µs, 10 µs, 1 ms ...; 1 KB, 10 KB ...; 1×, 2×, 5× ...), never as powers of ten.
- Bar charts print the value at the end of every bar.
- save() checks every figure after drawing it: no two texts overlap and nothing falls outside the saved image.
"""
import os
import sys

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa: E402
from matplotlib.ticker import FixedFormatter, FixedLocator, NullFormatter  # noqa: E402
from matplotlib.transforms import Bbox  # noqa: E402

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa: E402,F401,F403

ORDER = ["mac-m1", "win-ryzen", "vm-x86v2"]  # every figure lists the machines in this order
COL = {"mac-m1": "#0072B2", "win-ryzen": "#009E73", "vm-x86v2": "#D55E00"}
MARK = {"mac-m1": "o", "win-ryzen": "^", "vm-x86v2": "s"}
SHORT = {"mac-m1": "Mac M1 (ARM, NEON)", "win-ryzen": "Laptop (Ryzen, AVX2)", "vm-x86v2": "VM (x86, no AVX2)"}
NAME = {"mac-m1": "Mac M1", "win-ryzen": "Laptop", "vm-x86v2": "VM"}
MAC_LIGHT = "#8FC1E3"  # a second series of Mac-only data (E14 bursts)
# Algorithm families: none of these is close to a machine colour (checked for normal and colour-blind vision).
FAMCOL = {"ML-DSA": "#332288", "Falcon": "#CC79A7", "SLH-DSA": "#DDAA33", "ML-KEM": "#882255", "HQC": "#AA99DD", "classical": "#222222"}
FAMILIES = ["ML-DSA", "Falcon", "SLH-DSA", "ML-KEM", "HQC", "classical"]
NEUTRAL = "#4D4D4D"
NEUTRAL_LIGHT = "#A6A6A6"
STAGES = ["activation", "issuance", "retrieval", "presentation", "verification"]
STAGECOL = ["#E7D4E8", "#C2A5CF", "#9970AB", "#762A83", "#40004B"]  # one purple ramp in lifecycle order
STAGETEXT = ["black", "black", "white", "white", "white"]

plt.rcParams.update({
    "font.size": 8, "axes.titlesize": 9, "axes.labelsize": 8, "legend.fontsize": 7, "xtick.labelsize": 7, "ytick.labelsize": 7,
    "axes.spines.top": False, "axes.spines.right": False, "axes.linewidth": 0.6, "grid.linewidth": 0.4, "grid.alpha": 0.5,
    "figure.dpi": 150, "savefig.bbox": "tight", "savefig.pad_inches": 0.06, "pdf.fonttype": 42, "lines.linewidth": 1.0,
    "figure.constrained_layout.h_pad": 0.06, "figure.constrained_layout.w_pad": 0.06,
})

PROBLEMS = []  # layout problems found by save(); run_all.py prints them


def table(name):
    return read_csv(os.path.join(OUT_TABLES, name))


def new_fig(w, h, nrows=1, ncols=1, **kw):
    """A figure whose layout engine keeps titles, legends and labels apart (constrained layout)."""
    return plt.subplots(nrows, ncols, figsize=(w, h), layout="constrained", **kw)


def headline(fig, title, handles=None, labels=None, ncol=3, where="lower"):
    """Figure title at the top; the legend outside the plotting area, under the panels (never on the title)."""
    fig.suptitle(title, fontsize=9)
    if handles:
        fig.legend(handles, labels, loc="outside %s center" % where, ncol=ncol, frameon=False)


def machine_handles(machines=ORDER, kind="bar"):
    from matplotlib.lines import Line2D
    from matplotlib.patches import Patch
    if kind == "bar":
        return [Patch(color=COL[m], linewidth=0) for m in machines], [SHORT[m] for m in machines]
    return [Line2D([0], [0], color=COL[m], marker=MARK[m], markersize=4, linestyle="" if kind == "dot" else "-") for m in machines], [SHORT[m] for m in machines]


# ------------------------------------------------------------------------------------------------ number formats

def fmt_us(v):
    """A time given in microseconds, in plain units: 8.3 µs, 60 µs, 1.8 ms, 21.1 ms, 431 ms, 1.50 s."""
    if v != v:
        return ""
    if v < 10:
        return "%.1f µs" % v
    if v < 1000:
        return "%.0f µs" % v
    ms = v / 1000
    if ms < 100:
        return "%.1f ms" % ms
    if ms < 1000:
        return "%.0f ms" % ms
    return "%.2f s" % (ms / 1000)


def fmt_bytes(v):
    return "{:,} B".format(int(round(v)))


def fmt_ratio(v, digits=2):
    return ("%." + str(digits) + "f×") % v


def tint(colour, f=0.55):
    """A lighter shade of a colour (f of the way to white): the second bar of a family-coloured pair."""
    from matplotlib.colors import to_rgb
    r, g, b = to_rgb(colour)
    return (r + (1 - r) * f, g + (1 - g) * f, b + (1 - b) * f)


# ------------------------------------------------------------------------------------------------ log axes in plain units

TIME_TICKS = [(0.1, "0.1 µs"), (1, "1 µs"), (10, "10 µs"), (100, "100 µs"), (1e3, "1 ms"), (1e4, "10 ms"), (1e5, "100 ms"), (1e6, "1 s"), (1e7, "10 s")]
BYTE_TICKS = [(1, "1 B"), (10, "10 B"), (100, "100 B"), (1e3, "1 KB"), (1e4, "10 KB"), (1e5, "100 KB"), (1e6, "1 MB"), (1e7, "10 MB")]
RATIO_TICKS = [(0.1, "0.1×"), (0.2, "0.2×"), (0.5, "0.5×"), (1, "1×"), (2, "2×"), (5, "5×"), (10, "10×"), (20, "20×"), (50, "50×"),
               (100, "100×"), (1000, "1,000×")]
SHARE_TICKS = [(1, "100 %"), (0.1, "10 %"), (0.01, "1 %"), (1e-3, "0.1 %"), (1e-4, "0.01 %"), (1e-5, "0.001 %"), (1e-6, "0.0001 %")]


def log_axis(ax, which, ticks, label=None, base=10):
    """Logarithmic axis with fixed ticks in plain units; the label gains "(log scale)"."""
    (ax.set_xscale if which == "x" else ax.set_yscale)("log", base=base)
    axis = ax.xaxis if which == "x" else ax.yaxis
    axis.set_major_locator(FixedLocator([t for t, _ in ticks]))
    axis.set_major_formatter(FixedFormatter([s for _, s in ticks]))
    axis.set_minor_formatter(NullFormatter())
    if label is not None:
        (ax.set_xlabel if which == "x" else ax.set_ylabel)(label + " (log scale)")


def time_axis(ax, which="x", label="latency per call", unit="us"):
    """Time axis in plain units. unit: the unit of the plotted numbers ("us" or "ms")."""
    f = 1 if unit == "us" else 1e-3
    log_axis(ax, which, [(t * f, s) for t, s in TIME_TICKS], label)


def byte_axis(ax, which="x", label="bytes", unit="B"):
    f = {"B": 1, "KB": 1e-3, "GB": 1e-9}[unit]
    if unit == "GB":
        ticks = [(0.1, "0.1 GB"), (1, "1 GB"), (10, "10 GB"), (100, "100 GB"), (1000, "1 TB")]
    else:
        ticks = [(t * f, s) for t, s in BYTE_TICKS]
    log_axis(ax, which, ticks, label)


def ratio_axis(ax, which="y", label="ratio", lim=None):
    log_axis(ax, which, RATIO_TICKS, label)
    if lim:
        (ax.set_xlim if which == "x" else ax.set_ylim)(*lim)


# ------------------------------------------------------------------------------------------------ value labels on bars

def label_bars(ax, positions, values, texts, orient="h", fontsize=5.5, color="#222222", pad=1.1, rotation=0):
    """Write texts[i] just past the end of bar i. Returns the text artists (fit_labels makes room for them)."""
    out = []
    log = (ax.get_xscale() if orient == "h" else ax.get_yscale()) == "log"
    for p, v, s in zip(positions, values, texts):
        if v != v or not s:
            continue
        if orient == "h":
            x = v * pad if log else v
            out.append(ax.annotate(s, (x, p), xytext=(2, 0), textcoords="offset points", ha="left", va="center", fontsize=fontsize, color=color))
        else:
            y = v * pad if log else v
            out.append(ax.annotate(s, (p, y), xytext=(0, 2), textcoords="offset points", ha="center", va="bottom", fontsize=fontsize, color=color,
                                   rotation=rotation))
    return out


def fit_labels(fig, ax, texts, orient="h", tries=12):
    """Widen the value axis until every bar label fits inside the axes."""
    if not texts:
        return
    for _ in range(tries):
        fig.canvas.draw()
        r = fig.canvas.get_renderer()
        box = ax.get_window_extent(r)
        over = [t for t in texts if (t.get_window_extent(r).x1 > box.x1 - 1 if orient == "h" else t.get_window_extent(r).y1 > box.y1 - 1)]
        if not over:
            return
        lo, hi = ax.get_xlim() if orient == "h" else ax.get_ylim()
        log = (ax.get_xscale() if orient == "h" else ax.get_yscale()) == "log"
        new = hi * 1.5 if log else hi + (hi - lo) * 0.12
        (ax.set_xlim if orient == "h" else ax.set_ylim)(lo, new)


# ------------------------------------------------------------------------------------------------ save and check

def _ext(artist, r):
    try:
        bb = artist.get_window_extent(r)
    except Exception:  # noqa: BLE001
        return None
    if bb is None or bb.width <= 0 or bb.height <= 0:
        return None
    return bb


def check_layout(fig, name):
    """Overlapping texts and texts outside the saved image, after the final draw."""
    fig.canvas.draw()
    r = fig.canvas.get_renderer()
    items = []
    st = fig._suptitle
    if st is not None and st.get_text():
        items.append(("figure title", None, _ext(st, r), st.get_text()[:30]))
    for lg in fig.legends:
        items.append(("figure legend", None, _ext(lg, r), "legend"))
    for ax in fig.axes:
        if not ax.get_visible():
            continue
        for t, kind in ((ax.title, "title"), (ax._left_title, "title"), (ax._right_title, "title"), (ax.xaxis.label, "x label"), (ax.yaxis.label, "y label")):
            if t.get_visible() and t.get_text():
                items.append((kind, ax, _ext(t, r), t.get_text()[:30]))
        lg = ax.get_legend()
        if lg is not None:
            items.append(("legend", ax, _ext(lg, r), "legend"))
        for axis in (ax.xaxis, ax.yaxis):
            lo, hi = sorted(axis.get_view_interval())
            span = (hi - lo) * 1e-9
            for tick, loc in zip(axis.get_major_ticks(), axis.get_majorticklocs()):
                if not (lo - span <= loc <= hi + span):
                    continue  # matplotlib does not draw ticks outside the visible range
                for t in (tick.label1, tick.label2):
                    if t.get_visible() and t.get_text():
                        items.append(("tick", ax, _ext(t, r), t.get_text()[:20]))
        for t in ax.texts:
            if t.get_visible() and t.get_text():
                items.append(("text", ax, _ext(t, r), t.get_text()[:20]))
    items = [x for x in items if x[2] is not None]
    problems = []
    tb = fig.get_tightbbox(r)
    pad = plt.rcParams["savefig.pad_inches"]
    saved = Bbox.from_extents((tb.x0 - pad) * fig.dpi, (tb.y0 - pad) * fig.dpi, (tb.x1 + pad) * fig.dpi, (tb.y1 + pad) * fig.dpi)
    for kind, ax, bb, s in items:
        if bb.x0 < saved.x0 - 1 or bb.x1 > saved.x1 + 1 or bb.y0 < saved.y0 - 1 or bb.y1 > saved.y1 + 1:
            problems.append("cut off: %s '%s'" % (kind, s))
        if kind == "text" and ax is not None and getattr(ax, "_labels_may_leave", False) is False:
            box = ax.get_window_extent(r)
            if bb.x1 > box.x1 + 1 or bb.x0 < box.x0 - 1 or bb.y1 > box.y1 + 1 or bb.y0 < box.y0 - 1:
                problems.append("outside its panel: value label '%s'" % s)
    for i in range(len(items)):
        for j in range(i + 1, len(items)):
            a, b = items[i], items[j]
            w = min(a[2].x1, b[2].x1) - max(a[2].x0, b[2].x0)
            h = min(a[2].y1, b[2].y1) - max(a[2].y0, b[2].y0)
            if w > 1 and h > 1:
                problems.append("overlap: %s '%s' / %s '%s'" % (a[0], a[3], b[0], b[3]))
    for p in problems:
        PROBLEMS.append("%s: %s" % (name, p))
    return problems


def save(fig, name, outdir=None):
    outdir = outdir or OUT_FIGS
    os.makedirs(outdir, exist_ok=True)
    fig.savefig(os.path.join(outdir, name + ".pdf"), metadata={"CreationDate": None})  # reproducible files
    fig.savefig(os.path.join(outdir, name + ".png"), dpi=170)
    check_layout(fig, name)
    preview = os.environ.get("KPI_FIG_PREVIEW")  # optional low-resolution copies for a quick visual check
    if preview:
        os.makedirs(preview, exist_ok=True)
        fig.savefig(os.path.join(preview, name + ".png"), dpi=int(os.environ.get("KPI_FIG_PREVIEW_DPI", "80")))
    plt.close(fig)
    return name
