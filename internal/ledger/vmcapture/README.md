# Read-only capture of the live QChain ledger and IPFS store (E13 calibration)

Captured on 13 Oct 2026 from the UoS VM, read-only, with commands relayed one at a time. Nothing was stopped, sent or
changed on the VM; the raw block file never left it.

| Item | Value |
|---|---|
| Fabric peer | v2.5.15, `docker-peer0.general.uae.com-1` |
| Ledger file | `blockfile_000000`, 942,077 bytes, SHA-256 `e6c965b62bc0b4415c2fe29b72623612ea5c34f92d57cecba0c7a15685a0f271` (copied with `docker exec ... cat`, hash checked on both sides) |
| Contents | 61 blocks, 61 transactions (one per block) |
| IPFS | Kubo 0.40.1, user systemd service, flatfs datastore |

## Files

- `capture-export.json`: every block and transaction as sizes only, made on the VM by `scripts/fabric-block-sizes.py`
  (SHA-256 `5d6c8e42...` before its patch, `b29348b9...` after) through `scripts/fabric-capture-export.py`. File SHA-256
  `cd4d6d1490216b5f7e5de3e7c736886a1b6bbc757e4877d01a622e656fbbb45c`, 151,131 bytes; it was pasted compressed and the
  hash re-checked on the Mac.
- `ipfs-pins.txt`: every pinned IPFS object as file and block sizes (VM step 7).
- `ipfs-link.txt`: each issued credential's envelope size, linked through the CID on the chain (VM step 8,
  `scripts/fabric-ipfs-link.py`).

## Privacy

Only byte lengths, counts, validation codes, algorithm OIDs, and the public chaincode and MSP names (as in the main
repository) were exported. No key, signature, hash, CID, name, certificate or record content is in these files. The
issuer identity appears only as its length and its shape with every value replaced by `x`.

## Facts relied on by the model (internal/ledger)

Block-file layout (Fabric v2.5.15 `common/ledger/blkstorage/block_serialization.go`), protobuf framing of every message,
two endorsements (General and Government), ECDSA P-256 certificates and signatures, orderer metadata per block,
IPFS dag-pb/UnixFS wrapping (single block: file + 8, 11 or 14 bytes; 256 KiB chunks). `go test ./internal/ledger`
reproduces every transaction and block of the capture from these rules.
