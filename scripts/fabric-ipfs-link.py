#!/usr/bin/env python3
"""E13: link every issued credential on the ledger to its stored IPFS envelope, sizes only.

    python3 fabric-ipfs-link.py BLOCKFILE [--no-ipfs]

For each valid issueCredential transaction it takes the CID argument, asks the local
IPFS daemon for the object's size (ipfs cat piped to a byte counter, ipfs block stat,
ipfs refs), and prints: block number, attribute count (from the field-hash argument),
file bytes, root block bytes, child blocks. It never prints a CID or any content.
Needs fabric-block-sizes.py in the same folder. Read-only."""
import importlib.util, json, os, re, subprocess, sys

here = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("fbs", os.path.join(here, "fabric-block-sizes.py"))
fbs = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fbs)
CID = re.compile(rb"^(Qm[1-9A-HJ-NP-Za-km-z]{44}|b[a-z2-7]{20,100})$")


def ipfs_sizes(cid):
    try:
        cat = subprocess.run(["ipfs", "cat", cid], capture_output=True, timeout=60)
        blk = subprocess.run(["ipfs", "block", "stat", cid], capture_output=True, text=True, timeout=60)
        refs = subprocess.run(["ipfs", "refs", "-r", cid], capture_output=True, text=True, timeout=60)
        size = [l.split()[1] for l in blk.stdout.splitlines() if l.startswith("Size")]
        return len(cat.stdout), int(size[0]), len(refs.stdout.split())
    except Exception as e:
        return None


def main():
    raw = open(sys.argv[1], "rb").read()
    use_ipfs = "--no-ipfs" not in sys.argv
    print("block, attributes, file bytes, root block bytes, child blocks   (CIDs are never printed)")
    i = n = 0
    while i < len(raw):
        ln, j = fbs.rd_varint(raw, i)
        if ln == 0 or j + ln > len(raw):
            break
        blob = raw[j:j + ln]
        number, k = fbs.rd_varint(blob, 0)
        _, k = fbs.raw_bytes(blob, k)
        _, k = fbs.raw_bytes(blob, k)
        ntx, k = fbs.rd_varint(blob, k)
        envs = []
        for _ in range(ntx):
            e, k = fbs.raw_bytes(blob, k)
            envs.append(e)
        nmeta, k = fbs.rd_varint(blob, k)
        md = []
        for _ in range(nmeta):
            m, k = fbs.raw_bytes(blob, k)
            md.append(m)
        flt = md[2] if len(md) > 2 else b""
        for idx, env in enumerate(envs):
            try:
                p = fbs.msg(fbs.g(fbs.msg(env), 1))
                if fbs.gi(fbs.msg(fbs.g(fbs.msg(fbs.g(p, 1)), 1)), 1) != 3:
                    continue
                tx = fbs.msg(fbs.g(p, 2))
                cap = fbs.msg(fbs.g(fbs.msg(fbs.gall(tx, 1)[0]), 2))
                spec_ = fbs.msg(fbs.g(fbs.msg(fbs.g(fbs.msg(fbs.g(cap, 1)), 1)), 1))
                args = fbs.gall(fbs.msg(fbs.g(spec_, 3)), 1)
                if args[0] != b"issueCredential" or (idx < len(flt) and flt[idx] != 0):
                    continue
                cid = [a for a in args[1:] if CID.match(a)]
                nf = len(json.loads(args[7].decode()))  # field-hash JSON
                res = ipfs_sizes(cid[0].decode()) if (use_ipfs and cid) else None
                print(number, nf, *(res if res else ("-", "-", "-")))
                n += 1
            except Exception as e:
                print(number, "unreadable:", type(e).__name__)
        i = j + ln
    print("credentials listed:", n)


if __name__ == "__main__":
    main()
