"""Figure style and table readers shared by the figure scripts (matplotlib, Okabe-Ito colours: colour-blind safe)."""
import csv
import os
import sys

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt  # noqa

sys.path.insert(0, os.path.dirname(__file__))
from common import *  # noqa

COL = {"mac-m1": "#0072B2", "vm-x86v2": "#D55E00", "win-ryzen": "#009E73"}
MARK = {"mac-m1": "o", "vm-x86v2": "s", "win-ryzen": "^"}
SHORT = {"mac-m1": "Mac M1 (ARM, NEON)", "vm-x86v2": "VM (x86, no AVX2)", "win-ryzen": "Laptop (Ryzen, AVX2)"}
FAMCOL = {"ML-DSA": "#0072B2", "Falcon": "#CC79A7", "SLH-DSA": "#E69F00", "ML-KEM": "#009E73", "HQC": "#D55E00", "classical": "#555555"}

plt.rcParams.update({
    "font.size": 8, "axes.titlesize": 9, "axes.labelsize": 8, "legend.fontsize": 7, "xtick.labelsize": 7, "ytick.labelsize": 7,
    "axes.spines.top": False, "axes.spines.right": False, "axes.linewidth": 0.6, "grid.linewidth": 0.4, "grid.alpha": 0.5,
    "figure.dpi": 150, "savefig.bbox": "tight", "pdf.fonttype": 42, "lines.linewidth": 1.0,
})


def table(name):
    return read_csv(os.path.join(OUT_TABLES, name))


def save(fig, name):
    os.makedirs(OUT_FIGS, exist_ok=True)
    fig.savefig(os.path.join(OUT_FIGS, name + ".pdf"), metadata={"CreationDate": None})  # reproducible files
    fig.savefig(os.path.join(OUT_FIGS, name + ".png"), dpi=170)
    plt.close(fig)
    return name
