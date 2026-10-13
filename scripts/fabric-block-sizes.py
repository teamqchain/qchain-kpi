#!/usr/bin/env python3
"""E13 helper: sizes-only reader for a Hyperledger Fabric v2.x block file.

    python3 fabric-block-sizes.py BLOCKFILE [PER_TX_JSONL]

Reads a peer's blockfile_000000 (a sequence of: varint length, Block protobuf)
and prints byte sizes only: block and envelope components, the submitter's and
endorsers' certificates (DER layout, algorithm OIDs, signature lengths), the
chaincode arguments, the read/write set, and the length of every field of a
written record. It never prints a key, a value, a hash, a name or a
certificate: only counts, lengths, algorithm OIDs, the public chaincode and
MSP names, and the "shape" of the Issuer string with every value replaced by x.

Pure Python standard library (3.8+). Read-only.
"""
import base64
import collections
import json
import re
import statistics
import sys

KNOWN_CC = {"qchaincode", "_lifecycle", "lscc", "cscc", "qscc"}
KNOWN_FN = {
    "init", "registerHolder", "bindHolderKeys", "issueCredential", "updateExpiry", "revokeCredential",
    "suspendCredential", "restoreCredential", "getCredential", "getHolder", "getCredentials", "getHolders",
    "InstallChaincode", "ApproveChaincodeDefinitionForMyOrg", "CommitChaincodeDefinition",
    "CheckCommitReadiness", "QueryApprovedChaincodeDefinition", "QueryChaincodeDefinition",
}
KNOWN_MSP = {"GeneralMSP", "GovernmentMSP", "OrdererMSP"}
KNOWN_DOCTYPE = {"credential", "holder"}
KNOWN_FIELDS = {
    "DocType", "CommitmentVersion", "ID", "Holder", "Issuer", "IssuerOrgID", "Status", "CredentialType",
    "IssuedAt", "ExpiryDate", "CID", "FieldHashes", "CredentialHash", "Signature", "PublicKey",
    "FirstName", "LastName", "KemPublicKey", "DsaPublicKey", "Reason", "SuspendedAt", "RevokedAt",
    "UpdatedAt", "KemAlgorithm", "DsaAlgorithm",
}
OID_NAMES = {
    "1.2.840.10045.2.1": "ecPublicKey", "1.2.840.10045.3.1.7": "prime256v1", "1.3.132.0.34": "secp384r1",
    "1.2.840.10045.4.3.2": "ecdsa-with-SHA256", "1.2.840.10045.4.3.3": "ecdsa-with-SHA384",
    "1.2.840.113549.1.1.1": "rsaEncryption", "1.2.840.113549.1.1.11": "sha256WithRSA", "1.3.101.112": "Ed25519",
    "2.16.840.1.101.3.4.3.17": "ML-DSA-44", "2.16.840.1.101.3.4.3.18": "ML-DSA-65",
    "2.16.840.1.101.3.4.3.19": "ML-DSA-87",
}


def rd_varint(b, i):
    r = 0
    s = 0
    while True:
        c = b[i]
        i += 1
        r |= (c & 0x7F) << s
        if not c & 0x80:
            return r, i
        s += 7


def msg(b):
    """A protobuf message as {field number: [values]} (bytes or int)."""
    d = {}
    i = 0
    n = len(b)
    while i < n:
        k, i = rd_varint(b, i)
        f, w = k >> 3, k & 7
        if w == 0:
            v, i = rd_varint(b, i)
        elif w == 2:
            ln, i = rd_varint(b, i)
            if i + ln > n:
                raise ValueError("truncated field")
            v = b[i:i + ln]
            i += ln
        elif w == 1:
            v = b[i:i + 8]
            i += 8
        elif w == 5:
            v = b[i:i + 4]
            i += 4
        else:
            raise ValueError("wire type %d" % w)
        d.setdefault(f, []).append(v)
    return d


def g(m, f, default=b""):
    return m.get(f, [default])[0]


def gi(m, f):
    v = g(m, f, 0)
    return v if isinstance(v, int) else 0


def gall(m, f):
    return m.get(f, [])


# ---- minimal DER reader (lengths and OIDs only) ----------------------------------------

def der_items(b, start, end):
    out = []
    i = start
    while i < end:
        hdr = i
        tag = b[i]
        i += 1
        ln = b[i]
        i += 1
        if ln & 0x80:
            nb = ln & 0x7F
            ln = int.from_bytes(b[i:i + nb], "big")
            i += nb
        out.append((tag, hdr, i, i + ln))
        i += ln
    return out


def full(item):
    return item[3] - item[1]


def oid_str(b, item):
    raw = b[item[2]:item[3]]
    parts = [raw[0] // 40, raw[0] % 40]
    v = 0
    for c in raw[1:]:
        v = (v << 7) | (c & 0x7F)
        if not c & 0x80:
            parts.append(v)
            v = 0
    s = ".".join(str(p) for p in parts)
    return OID_NAMES.get(s, s)


def cert_info(pem):
    info = {"pem": len(pem)}
    try:
        text = pem.decode("ascii")
        body = "".join(x for x in text.splitlines() if x and not x.startswith("-----"))
        der = base64.b64decode(body)
        top = der_items(der, 0, len(der))[0]
        cert = der_items(der, top[2], top[3])
        tbs, sigalg, sigbits = cert[0], cert[1], cert[2]
        ch = der_items(der, tbs[2], tbs[3])
        base = 1 if ch[0][0] == 0xA0 else 0
        spki = ch[base + 5]
        spki_items = der_items(der, spki[2], spki[3])
        alg = der_items(der, spki_items[0][2], spki_items[0][3])
        info.update({
            "der": len(der), "tbs": full(tbs), "sigalg": full(sigalg),
            "sigalg_oid": oid_str(der, der_items(der, sigalg[2], sigalg[3])[0]),
            "sig_value": sigbits[3] - sigbits[2] - 1,
            "issuer_dn": full(ch[base + 2]), "subject_dn": full(ch[base + 4]), "validity": full(ch[base + 3]),
            "spki": full(spki), "spki_oid": oid_str(der, alg[0]),
            "spki_param": oid_str(der, alg[1]) if len(alg) > 1 and alg[1][0] == 0x06 else None,
            "spki_key_bits": spki_items[1][3] - spki_items[1][2] - 1,
            "extensions": sum(full(x) for x in ch[base + 6:] if x[0] == 0xA3),
        })
    except Exception as e:  # keep going; the length is still useful
        info["parse"] = "failed:" + type(e).__name__
    return info


def identity(b):
    m = msg(b)
    mspid = g(m, 1).decode("utf-8", "replace")
    out = cert_info(g(m, 2))
    out["msp"] = mspid if mspid in KNOWN_MSP else "?"
    out["msp_len"] = len(mspid)
    out["ser"] = len(b)
    return out


def mask_dn(s):
    return re.sub(r"(?<==)[^,/:+]+", lambda m: "x" * len(m.group()), s)


def value_info(val):
    try:
        obj = json.loads(val.decode("utf-8"))
    except Exception:
        return {"bytes": len(val), "json": False}
    if not isinstance(obj, dict):
        return {"bytes": len(val), "json": True, "kind": type(obj).__name__}
    out = {"bytes": len(val), "json": True, "fields": {}, "unknown_fields": 0}
    dt = obj.get("DocType")
    out["doctype"] = dt if dt in KNOWN_DOCTYPE else "?"
    for k, v in obj.items():
        if k not in KNOWN_FIELDS:
            out["unknown_fields"] += 1
            continue
        if isinstance(v, str):
            out["fields"][k] = len(v.encode("utf-8"))
        elif isinstance(v, dict):
            out["fields"][k] = {"count": len(v), "value_lens": sorted({len(str(x)) for x in v.values()})}
        else:
            out["fields"][k] = len(json.dumps(v))
        if k == "Issuer" and isinstance(v, str):
            out["issuer_shape"] = mask_dn(v)
    return out


# ---- transactions and blocks ------------------------------------------------------------

def parse_tx(env, code):
    e = msg(env)
    payload, sig = g(e, 1), g(e, 2)
    p = msg(payload)
    hdr = msg(g(p, 1))
    data = g(p, 2)
    ch = msg(g(hdr, 1))
    sh = msg(g(hdr, 2))
    t = {
        "type": gi(ch, 1), "valid": code, "env": len(env), "payload": len(payload), "client_sig": len(sig),
        "channel_header": len(g(hdr, 1)), "sig_header": len(g(hdr, 2)), "txid": len(g(ch, 5)),
        "channel_id": len(g(ch, 4)), "chaincode_ext": len(g(ch, 7)), "tls_cert_hash": len(g(ch, 8)),
        "creator": identity(g(sh, 1)), "nonce": len(g(sh, 2)), "data": len(data),
    }
    if t["type"] != 3:
        t["group"] = "type%d" % t["type"]
        return t
    tx = msg(data)
    actions = gall(tx, 1)
    t["actions"] = len(actions)
    ta = msg(actions[0])
    t["action_header"] = len(g(ta, 1))
    cap_raw = g(ta, 2)
    cap = msg(cap_raw)
    t["action_payload"] = len(cap_raw)
    prop = g(cap, 1)
    t["proposal_payload"] = len(prop)
    pp = msg(prop)
    spec = msg(g(msg(g(pp, 1)), 1))
    cc = g(msg(g(spec, 2)), 2).decode("utf-8", "replace")
    args = gall(msg(g(spec, 3)), 1)
    fn = args[0].decode("utf-8", "replace") if args else ""
    t["cc"] = cc if cc in KNOWN_CC else "?"
    t["fn"] = fn if fn in KNOWN_FN else "?"
    t["args"] = [len(a) for a in args[1:]]
    t["args_total"] = sum(t["args"])
    t["input_spec"] = len(g(pp, 1))
    action = msg(g(cap, 2))
    prp_raw = g(action, 1)
    t["prp"] = len(prp_raw)
    ends = []
    for eb in gall(action, 2):
        em = msg(eb)
        idn = identity(g(em, 1))
        idn["sig"] = len(g(em, 2))
        idn["endorsement"] = len(eb)
        ends.append(idn)
    t["endorsements"] = ends
    prp = msg(prp_raw)
    t["proposal_hash"] = len(g(prp, 1))
    ca = msg(g(prp, 2))
    results = g(ca, 1)
    resp = msg(g(ca, 3))
    t.update({"extension": len(g(prp, 2)), "results": len(results), "events": len(g(ca, 2)),
              "resp_status": gi(resp, 1), "resp_message": len(g(resp, 2)), "resp_payload": len(g(resp, 3)),
              "ca_chaincode_id": len(g(ca, 4))})
    rw = []
    for nsb in gall(msg(results), 2):
        nsm = msg(nsb)
        name = g(nsm, 1).decode("utf-8", "replace")
        kv = msg(g(nsm, 2))
        writes = []
        for wb in gall(kv, 3):
            w = msg(wb)
            writes.append({"key": len(g(w, 1)), "delete": gi(w, 2), "value": value_info(g(w, 3))})
        rw.append({"ns": name if name in KNOWN_CC else "?", "rwset": len(g(nsm, 2)), "reads": len(gall(kv, 1)),
                   "writes": writes})
    t["rwsets"] = rw
    t["group"] = "%s/%s" % (t["cc"], t["fn"])
    return t


def parse_block(blob, frame):
    bm = msg(blob)
    header = msg(g(bm, 1))
    data = msg(g(bm, 2))
    meta = msg(g(bm, 3))
    md = gall(meta, 1)
    envs = gall(data, 1)
    flt = md[2] if len(md) > 2 else b""
    b = {
        "number": gi(header, 1), "frame": frame, "block": len(blob), "header": len(g(bm, 1)),
        "data": len(g(bm, 2)), "metadata": len(g(bm, 3)), "ntx": len(envs), "md_lens": [len(x) for x in md],
        "previous_hash": len(g(header, 2)), "data_hash": len(g(header, 3)),
    }
    sigs = []
    if md:
        sm = msg(md[0])
        for sb in gall(sm, 2):
            s = msg(sb)
            sh = msg(g(s, 1))
            idn = identity(g(sh, 1))
            idn.update({"sig": len(g(s, 2)), "sig_header": len(g(s, 1)), "id_header": len(g(s, 3)),
                        "nonce": len(g(sh, 2))})
            sigs.append(idn)
        b["md0_value"] = len(g(sm, 1))
    b["orderer_signatures"] = sigs
    txs = []
    for i, env in enumerate(envs):
        code = flt[i] if i < len(flt) else -1
        try:
            txs.append(parse_tx(env, code))
        except Exception as ex:
            txs.append({"group": "unparsed:" + type(ex).__name__, "env": len(env), "valid": code})
    return b, txs


def stat(vals):
    if not vals:
        return "-"
    return "%d / %d / %d" % (min(vals), int(statistics.median(vals)), max(vals))


def main():
    if len(sys.argv) < 2:
        print(__doc__)
        return 2
    raw = open(sys.argv[1], "rb").read()
    out_path = sys.argv[2] if len(sys.argv) > 2 else None
    blocks, txs = [], []
    i = 0
    while i < len(raw):
        try:
            ln, j = rd_varint(raw, i)
        except IndexError:
            break
        if ln == 0 or j + ln > len(raw):
            break
        b, t = parse_block(raw[j:j + ln], j + ln - i)
        blocks.append(b)
        for x in t:
            x["block"] = b["number"]
        txs.extend(t)
        i = j + ln
    print("file bytes: %d | blocks: %d | bytes after the last whole block: %d" % (len(raw), len(blocks), len(raw) - i))
    if blocks:
        print("block numbers: %d..%d | transactions: %d" % (blocks[0]["number"], blocks[-1]["number"], len(txs)))
        print("block frame bytes  min/median/max:", stat([b["frame"] for b in blocks]))
        print("transactions per block min/median/max:", stat([b["ntx"] for b in blocks]))
        print("tx-count histogram:", dict(sorted(collections.Counter(b["ntx"] for b in blocks).items())))
        print("orderer signatures per block:", dict(collections.Counter(len(b["orderer_signatures"]) for b in blocks)))
        print("block header/data/metadata bytes (first block):", blocks[0]["header"], blocks[0]["data"], blocks[0]["metadata"])
        ex = [b for b in blocks if b["orderer_signatures"]]
        if ex:
            print("one block's ordering metadata:", json.dumps({k: ex[-1][k] for k in (
                "number", "frame", "block", "header", "data", "metadata", "ntx", "md_lens", "previous_hash",
                "data_hash", "md0_value", "orderer_signatures")}, separators=(",", ":")))
    groups = collections.OrderedDict()
    for t in txs:
        groups.setdefault(t.get("group", "?"), []).append(t)
    for name, ts in groups.items():
        valid = [t for t in ts if t["valid"] == 0]
        print("\n== %s | n=%d valid=%d  (min / median / max bytes)" % (name, len(ts), len(valid)))
        rows = [("envelope", lambda t: t["env"]), ("payload", lambda t: t["payload"]),
                ("client signature", lambda t: t["client_sig"]),
                ("submitter certificate (PEM)", lambda t: t["creator"].get("pem", 0)),
                ("submitter certificate (DER)", lambda t: t["creator"].get("der", 0))]
        if ts[0].get("type") == 3:
            rows += [("arguments, total", lambda t: t["args_total"]),
                     ("proposal payload", lambda t: t["proposal_payload"]),
                     ("endorsements (count)", lambda t: len(t["endorsements"])),
                     ("endorsements, total", lambda t: sum(x["endorsement"] for x in t["endorsements"])),
                     ("proposal response payload", lambda t: t["prp"]),
                     ("read/write set", lambda t: t["results"]),
                     ("response payload", lambda t: t["resp_payload"]),
                     ("written value, largest", lambda t: max([w["value"]["bytes"] for r in t["rwsets"] for w in r["writes"]] or [0]))]
        for label, fn in rows:
            try:
                print("  %-30s %s" % (label, stat([fn(t) for t in valid or ts])))
            except Exception as ex:
                print("  %-30s n/a (%s)" % (label, type(ex).__name__))
        e = (valid or ts)[0]
        print("  one transaction in full:", json.dumps(e, separators=(",", ":"), sort_keys=True))
    if out_path:
        with open(out_path, "w") as f:
            for t in txs:
                f.write(json.dumps(t, separators=(",", ":"), sort_keys=True) + "\n")
        print("\nper-transaction sizes written to", out_path)
    return 0


if __name__ == "__main__":
    sys.exit(main())
