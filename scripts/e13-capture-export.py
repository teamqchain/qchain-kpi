#!/usr/bin/env python3
"""E13: export the sizes-only capture for the Mac, compressed for pasting.

    python3 e13-capture-export.py BLOCKFILE OUT.json

Uses e13-block-sizes.py (same folder) to read every block and transaction,
writes OUT.json (numbers, lengths, algorithm OIDs and public chaincode/MSP names
only; view it with `less`), then prints its SHA-256 and the file gzip-compressed
and base64-encoded, in 76-character lines between BEGIN and END markers."""
import base64, gzip, hashlib, importlib.util, json, os, sys

here = os.path.dirname(os.path.abspath(__file__))
spec = importlib.util.spec_from_file_location("fbs", os.path.join(here, "e13-block-sizes.py"))
fbs = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fbs)

raw = open(sys.argv[1], "rb").read()
blocks, txs = [], []
i = 0
while i < len(raw):
    ln, j = fbs.rd_varint(raw, i)
    if ln == 0 or j + ln > len(raw):
        break
    b, t = fbs.parse_block(raw[j:j + ln], j + ln - i)
    for x in t:
        x["block"] = b["number"]
    blocks.append(b)
    txs.extend(t)
    i = j + ln
doc = {"file_bytes": len(raw), "parsed_bytes": i, "blocks": blocks, "txs": txs}
data = json.dumps(doc, separators=(",", ":"), sort_keys=True).encode()
open(sys.argv[2], "wb").write(data)
print("blocks", len(blocks), "transactions", len(txs), "json bytes", len(data))
print("sha256", hashlib.sha256(data).hexdigest())
b64 = base64.b64encode(gzip.compress(data, 9, mtime=0)).decode()
print("BEGIN")
for k in range(0, len(b64), 76):
    print(b64[k:k + 76])
print("END")
