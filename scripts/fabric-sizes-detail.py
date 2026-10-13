import json, sys
T = [json.loads(l) for l in open(sys.argv[1])]
print("--- every issueCredential: block, envelope, attribute count, arguments, record bytes, validation code")
for t in T:
    if t.get("fn") == "issueCredential":
        v = [x["value"] for r in t["rwsets"] for x in r["writes"]][0]
        print(t["block"], t["env"], "nfields", v["fields"]["FieldHashes"]["count"], "args", t["args"], "record", v["bytes"], "code", t["valid"])
print("--- one record of each kind: key bytes, record bytes, length of every field")
seen = set()
for t in T:
    for r in t.get("rwsets", []):
        for x in r["writes"]:
            v = x["value"]
            k = (t["fn"], v.get("doctype"))
            if k in seen:
                continue
            seen.add(k)
            print(k, "key", x["key"], "bytes", v["bytes"], "unknown_fields", v.get("unknown_fields"), "issuer_shape", v.get("issuer_shape"), "fields", v.get("fields"))
print("--- read/write set structure, one transaction per function")
seen = set()
for t in T:
    f = t.get("fn")
    if f and f not in seen:
        seen.add(f)
        print(f, [(r["ns"], "reads", r["reads"], "rwset", r["rwset"], "writes", [(x["key"], x["value"]["bytes"]) for x in r["writes"]]) for r in t["rwsets"]])
print("--- validation codes:", sorted({(t.get("group"), t["valid"]) for t in T}))
