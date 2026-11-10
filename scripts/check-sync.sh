#!/bin/bash
# Proves that every production copy in internal/qchain still matches the QChain
# main repository at commit 9026aba, apart from the edits listed in
# docs/provenance.md. Read-only on both repositories.
#
# "The main repository" is the former single repository QChain-PQC-blockchain, split
# into the qchain-* repositories on 9 November 2026. Commit 9026aba exists only in
# its local clone, which is kept for this check; the same files are in the new
# repositories (see "Source repositories" in docs/provenance.md).
#
#   scripts/check-sync.sh [MAIN_REPO]
#   default: ../../QChain-PQC-blockchain, i.e. /Applications/Programs for Fun/QChain-PQC-blockchain
#   when this repository is at /Applications/Programs for Fun/teamqchain/qchain-kpi
#
# For each copy, the documented edits are undone in memory and every copied
# line range of the original must then appear in the copy, unchanged and in
# one piece. Exit 0 = all copies match; 1 = a difference (printed).
set -euo pipefail
repo="$(cd "$(dirname "$0")/.." && pwd -P)"
main="${1:-$repo/../../QChain-PQC-blockchain}"
[ -d "$main/.git" ] || { echo "check-sync: $main is not a git repository" >&2; exit 2; }
git -C "$main" cat-file -e "9026aba^{commit}" 2>/dev/null || { echo "check-sync: commit 9026aba not found in $main" >&2; exit 2; }

/usr/bin/env python3 - "$repo" "$main" <<'PY'
import re, subprocess, sys
repo, main = sys.argv[1], sys.argv[2]
COMMIT = "9026aba"

def original(path):
    return subprocess.run(["git", "-C", main, "show", f"{COMMIT}:{path}"], check=True,
                          capture_output=True, text=True).stdout.split("\n")

# copy file -> (source path, [(first, last) line ranges; None = whole file without its package clause], undo edits)
UNDO_CRYPTO = [(r'^var sigName = "ML-DSA-44" // const in production.*$', 'const sigName = "ML-DSA-44"'),
               (r'^func pqcSignLiboqs\(', 'func pqcSign('), (r'^func pqcVerifyLiboqs\(', 'func pqcVerify(')]
UNDO_KEM = [(r'^func kemGenerateKeypairLiboqs\(', 'func kemGenerateKeypair('),
            (r'^func kemEncapLiboqs\(', 'func kemEncap('), (r'^func kemDecapLiboqs\(', 'func kemDecap(')]
UNDO_MOBILE = [(r'^var \( // decision B.*$', 'const (')]
COPIES = {
    "commitment_prod.go":     ("offchain/commitment.go", None, []),
    "crypto_prod.go":         ("offchain/crypto.go", [(29, 80)], UNDO_CRYPTO),
    "config_prod.go":         ("offchain/config.go", [(22, 22), (102, 110)], []),
    "chain_prod.go":          ("offchain/chain.go", [(42, 76), (277, 301)], []),
    "chain_snapshot_prod.go": ("offchain/chain.go", [(93, 106)], []),
    "kem_prod.go":            ("offchain/kem.go", None, UNDO_KEM),
    "envelope_prod.go":       ("offchain/envelope.go", None, []),
    "verification_prod.go":   ("offchain/verification.go", [(33, 221)], []),
    "httputil_prod.go":       ("offchain/httputil.go", [(37, 40)], []),
    "mobile_prod.go":         ("offchain/mobile.go", [(33, 37)], UNDO_MOBILE),
    "wallet_prod.go":         ("offchain/cmd/keygen/main.go", [(36, 133)], "wallet"),
}

def undo_wallet(lines):
    # Remove the marked HARNESS EDIT wrapper and restore `ss, err :=`.
    out, i = [], 0
    while i < len(lines):
        l = lines[i]
        if "HARNESS EDIT (decision A" in l:
            # skip: two comment lines, `var ss []byte`, the classical branch (4 lines), `} else {`
            j = i
            while "} else {" not in lines[j]:
                j += 1
            i = j + 1
            continue
        if l.strip() == "} // end of HARNESS EDIT":
            i += 1
            continue
        out.append(l.replace("\tss, err = kem.DecapSecret(kemCt)", "\tss, err := kem.DecapSecret(kemCt)"))
        i += 1
    return out

def contains(hay, block):
    n = len(block)
    return any(hay[i:i + n] == block for i in range(len(hay) - n + 1))

bad = 0
for copy, (src, ranges, undo) in COPIES.items():
    have = open(f"{repo}/internal/qchain/{copy}").read().split("\n")
    if undo == "wallet":
        have = undo_wallet(have)
    else:
        for pat, rep in undo:
            have = [re.sub(pat, rep, l) for l in have]
    orig = original(src)
    blocks = []
    if ranges is None:
        assert orig[0] == "package main", src
        blocks.append(orig[1:])
        # strip trailing empty lines of the original (files end with a newline)
        while blocks[0] and blocks[0][-1] == "":
            blocks[0].pop()
    else:
        for a, b in ranges:
            blocks.append(orig[a - 1:b])
    for k, blk in enumerate(blocks):
        if not contains(have, blk):
            bad += 1
            where = "whole file" if ranges is None else f"lines {ranges[k][0]}-{ranges[k][1]}"
            print(f"DIFFERS: internal/qchain/{copy} vs {src} ({where})")
            # first differing line, for the report
            for i, l in enumerate(blk):
                if l not in have:
                    print(f"  first line not found: {l!r}")
                    break
    if not bad:
        pass
print(f"check-sync: {len(COPIES)} copies checked against {COMMIT}: " + ("all match" if bad == 0 else f"{bad} block(s) differ"))
sys.exit(1 if bad else 0)
PY
