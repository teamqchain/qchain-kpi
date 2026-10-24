#!/usr/bin/env python3
"""Checks scripts/e13-block-sizes.py against the synthetic block file made by
testdata/fabric-blockgen (real Fabric protobuf types, made-up data): every size the
generator wrote must be found, the invalid transaction flagged, and no name or key
may appear in the output.   python3 scripts/e13-test-block-sizes.py"""
import json, os, subprocess, sys, tempfile
here = os.path.dirname(os.path.abspath(__file__))
fx = os.path.join(here, "..", "testdata", "fabric")
with tempfile.TemporaryDirectory() as d:
    out = os.path.join(d, "tx.jsonl")
    rep = subprocess.run([sys.executable, "-I", os.path.join(here, "e13-block-sizes.py"), os.path.join(fx, "synthetic.blockfile"), out],
                         check=True, capture_output=True, text=True).stdout
    got = [t for t in map(json.loads, open(out)) if t.get("type") == 3]
    exp = [json.loads(l) for l in open(os.path.join(fx, "synthetic.expected.jsonl"))]
    assert len(got) == len(exp), (len(got), len(exp))
    for t, e in zip(got, exp):
        w = t["rwsets"][0]["writes"][0]
        for name, a, b in [("block", t["block"], e["Block"]), ("fn", t["fn"], e["Fn"]), ("env", t["env"], e["Env"]),
                           ("payload", t["payload"], e["Payload"]), ("pem", t["creator"]["pem"], e["Pem"]),
                           ("nend", len(t["endorsements"]), e["NEnd"]), ("results", t["results"], e["Results"]),
                           ("args", t["args"], e["Args"]), ("value", w["value"]["bytes"], e["ValueBytes"]), ("key", w["key"], e["KeyLen"])]:
            assert a == b, (e["Block"], e["Idx"], name, a, b)
    assert [t["valid"] for t in got if t["fn"] == "issueCredential"] == [0, 0, 0, 0, 11]
    text = rep + open(out).read()
    for word in ("user1", "peer0", "general.uae.com", "Aaaa", "Bbbbbb", "H-0001", "CRED-", "orderer0", "Bachelor"):
        assert word not in text, word
print("e13-block-sizes: ok (%d transactions, all sizes match, nothing sensitive printed)" % len(exp))
