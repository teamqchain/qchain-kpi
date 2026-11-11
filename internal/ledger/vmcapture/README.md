<p align="center">
  <img src="../../../.github/assets/qchain_logo.jpeg" width="120" alt="QChain logo" />
</p>

<h1 align="center">E13 ledger capture</h1>

<p align="center">
  <b>A read-only capture of the live QChain ledger and IPFS store, used to calibrate the E13 size model.</b><br/>
  Captured on 13 October 2026 · <a href="../../../README.md">back to the main README</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/capture-read--only-2EA44F" alt="Read-only" />
  <img src="https://img.shields.io/badge/exported-sizes%20only-0A66C2" alt="Sizes only" />
  <img src="https://img.shields.io/badge/model-reproduces%20every%20byte-6E40C9" alt="Byte-exact" />
</p>

---

## The capture

Captured from the university VM with commands relayed one at a time. Nothing was stopped, sent or changed on the VM, and the raw block file never left it. The capture folder on the VM was deleted when the VM was reverted (10 November 2026); the exported sizes below are the only copy.

| Item | Value |
|---|---|
| Network | QChain's **classical** Fabric network of October 2026: ECDSA P-256 Fabric identities, credentials signed with ML-DSA-44 |
| Fabric peer | v2.5.15, `docker-peer0.general.uae.com-1` |
| Ledger file | `blockfile_000000`, 942,077 bytes, SHA-256 `e6c965b62bc0b4415c2fe29b72623612ea5c34f92d57cecba0c7a15685a0f271` (copied with `docker exec ... cat`, hash checked on both sides) |
| Contents | 61 blocks, 61 transactions (one per block) |
| IPFS | Kubo 0.40.1, user systemd service, flatfs datastore |

---

## Files

| File | What it holds |
|---|---|
| `capture-export.json` | every block and transaction as sizes only, made on the VM by `scripts/fabric-block-sizes.py` (now `scripts/e13-block-sizes.py`, byte-identical; SHA-256 `5d6c8e42...` before its patch, `b29348b9...` after) through `scripts/fabric-capture-export.py` (now `scripts/e13-capture-export.py`). File SHA-256 `cd4d6d1490216b5f7e5de3e7c736886a1b6bbc757e4877d01a622e656fbbb45c`, 151,131 bytes; it was pasted compressed and the hash re-checked on the Mac. |
| `ipfs-pins.txt` | every pinned IPFS object as file and block sizes (VM step 7) |
| `ipfs-link.txt` | each issued credential's envelope size, linked through the CID on the chain (VM step 8, `scripts/fabric-ipfs-link.py`, now `scripts/e13-ipfs-link.py`) |

---

## Privacy

Only byte lengths, counts, validation codes, algorithm OIDs, and the public chaincode and MSP names (as in QChain's repositories) were exported. No key, signature, hash, CID, name, certificate or record content is in these files. The issuer identity appears only as its length and its shape, with every value replaced by `x`.

---

## Facts the model relies on (`internal/ledger`)

Block-file layout (Fabric v2.5.15 `common/ledger/blkstorage/block_serialization.go`), protobuf framing of every message, two endorsements (General and Government), ECDSA P-256 certificates and signatures, orderer metadata per block, and IPFS dag-pb/UnixFS wrapping (single block: file + 8, 11 or 14 bytes; 256 KiB chunks). `go test ./internal/ledger` reproduces every transaction and block of the capture from these rules.

---

## As captured and projected

E13 reports two ledger scenarios (table `T5` in [results/combined](../../../results/combined/README.md)):

| Scenario | Fabric identities | `issueCredential`, 10 attributes, ML-DSA-44 |
|---|---|---|
| **As captured** | ECDSA P-256 (the network of 13 October) | 22,299 bytes (the model, which reproduces every captured transaction to the byte) |
| **Projected** | ML-DSA-44 certificates and signatures (the network deployed on 7 November 2026) | 48,825 bytes, a model |

> **Open check.** The projection has not yet been compared with a real block of the deployed ML-DSA-44 network. The plan is a
> read-only fetch of the newest `issueCredential` blocks on the VM, recording their byte sizes and attribute counts, and a
> comparison with the model at the same attribute count. Until then the projected figure is reported as a model.
