# E14 plan: Fabric transaction latency against payload size (Mac)

Status: **plan for approval, 22 Oct 2026. Nothing is coded.** Decisions of 21 Oct are the user's; everything marked "to verify" is checked in phase A.

## 1. Question

E13 calculated how many bytes a credential costs on the ledger. E14 measures how long a transaction takes on a real Fabric network, and whether the bigger post-quantum payloads change that:

- **Latency:** time from submitting a transaction to its commit, for payloads the size of QChain's issuance transaction under each signature scheme (E13's sizes).
- **Block behaviour:** how many transactions fit in a block, which checks E13's block model (a block closes at 10 transactions, 512 KB or 2 s).
- **Not measured:** post-quantum identities (Fabric 2.5 cannot use them; E13's projection covers that) and any cryptography in the chaincode (QChain's chaincode does none).

## 2. Decisions (user, 21 Oct)

| Topic | Decision |
|---|---|
| Where | The Mac (Colima, Docker), never the VM |
| Network | Mirror the live network: 2 organisations, 2 peers, 1 Raft orderer, CouchDB, 3 CAs, TLS; Fabric 2.5.15 and CouchDB 3.3.2 pinned |
| Payloads | 8 representative schemes plus a tiny control (section 6) |
| Modes | Sequential and burst |
| Chaincode | Synthetic, JavaScript like QChain's, stores N bytes, no cryptography |

## 3. The network (mirrors QChain's `qchain-network`, read-only copy of commit `ad271e1`)

| Component | Live network | E14 |
|---|---|---|
| Peers | `peer0.government.uae.com` (7051), `peer0.general.uae.com` (9051), Fabric 2.5.15 | same, image `hyperledger/fabric-peer:2.5.15` |
| Orderer | one `etcdraft` consenter, no system channel (`osnadmin` join) | same, `hyperledger/fabric-orderer:2.5.15` |
| State database | **one** CouchDB 3.3.2 (`couchdb0`) used by **both** peers | same (mirrored as is; see risks) |
| CAs | 3 Fabric CAs (government, general, orderer) | same, `hyperledger/fabric-ca:1.5.15` (the live CA version is not on record; the CAs are not in the timed path) |
| Channel | `mychannel`, profile `TwoOrgsChannel`, capabilities V2_0 / V2_5 | same |
| Block settings | BatchTimeout 2 s, MaxMessageCount 10, PreferredMaxBytes 512 KB, AbsoluteMaxBytes 99 MB | same |
| Endorsement | ImplicitMeta MAJORITY of two organisations, so both must endorse | same |
| Chaincode launch | peers build and run Node chaincode through the Docker socket (`CORE_VM_ENDPOINT`), runtime image `hyperledger/fabric-nodeenv:2.5` | same |
| Excluded | IPFS, the Go backend, MySQL, the web gateway | not present |

- Colima has no host mounts, so certificates and configuration live in Docker **named volumes**, filled with `docker cp` or from a tools container; nothing is installed on the Mac (the Fabric tools run from the `fabric-tools` and `fabric-ca` images).
- The Docker network keeps the name `docker_fabric_net` (compose project `docker`), as the live `CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE` expects.
- Setup needs the internet (image pulls, `npm install` inside the chaincode build); the measurement runs with Wi-Fi off and no outside traffic.

## 4. Synthetic chaincode (`fabric/chaincode`)

- Same dependencies as `QChaincode.js` (`fabric-contract-api`, `fabric-shim` ^2.5.8), name `kpipayload`, version 1.0, sequence 1.
- `putPayload(id, payloadHex)`: refuses a repeated id and a non-hex payload (like QChain's validation), then writes the JSON record `{DocType, ID, Payload}` (one read and one write, as in `registerHolder`/`issueCredential`). No cryptography, no access-control calls.
- `getPayload(id)`: read-back for the correctness checks.
- Tested with `node --test` inside the `fabric-nodeenv` image (Node is not installed on the Mac).

## 5. Client (`fabric/client`, its own Go module)

- **As the production backend:** `github.com/hyperledger/fabric-gateway` **v1.7.0** and `google.golang.org/grpc` v1.68.0 (the versions in `offchain/go.mod`), the issuer identity of the General organisation, `SubmitTransaction` semantics (endorse on both organisations, submit, wait for commit) and a **new connection per transaction** (QChain's `withContract` dials, connects and closes for every call).
- **Timed phases per transaction** (monotonic clock, all in nanoseconds): connect (dial, TLS, gateway), endorse, submit (orderer acknowledgement), commit wait, and the total from the start of the call to the commit status.
- **Recorded per transaction:** the block number it landed in, its index in the block, the validation code, and its envelope size.
- **Blocks:** after each block of transactions the client reads the blocks back from the peer (`qscc`) to record transactions per block, block bytes, and the cut reason (count reached, size reached, or timeout).
- **Own module:** the harness's `go.mod`/`go.sum` and measuring sources stay untouched (earlier runs can still be extended). The client is built and unit-tested only inside Docker (a `fabric/Dockerfile` stage), so the Mac's Go module cache is not touched.

## 6. Payloads (from E13: the envelope bytes of QChain's `issueCredential`, N = 5 attributes of 16 bytes)

| Point | Envelope size | Blocks E13 predicts per block |
|---|---|---|
| tiny control | about 0.5 KB | 10 |
| ECDSA P-256 | 7.1 KB | 10 |
| Falcon-512 | 12.8 KB | 10 |
| ML-DSA-44 | 21.5 KB | 10 |
| ML-DSA-65 | 27.6 KB | 10 |
| ML-DSA-87 | 35.5 KB | 10 |
| SLH-DSA-128s | 38.2 KB | 10 |
| SLH-DSA-128f | 75.1 KB | 6 |
| SLH-DSA-256f | 206.3 KB | 2 |

- **Calibration:** the payload length is set so the committed **envelope** matches the target within 1 %. The client submits a probe transaction, reads its envelope size from the block, and solves for the payload length (the envelope grows by about two times the payload, because the payload is in the arguments and in the stored record, as in QChain). The calibration is recorded.
- Each payload is random hex (so the bytes are incompressible and the record looks like keys and signatures).

## 7. Protocol

- **Blocks of work** (ID `E14.<point>.<mode>.r<rep>`): 9 points, 2 modes, 5 repetitions, in a seeded shuffled order, with tiny-control blocks at the start, middle and end of the run to show drift.
- **Warm-up:** 20 small transactions at the start of the run (the first call launches the chaincode container) and 5 before every block (not recorded).
- **Sequential:** 100 transactions one after the other, each waiting for its commit. Alone in the pipeline, a transaction waits for the 2 s batch timeout, so about 2.1 s each. 4,500 transactions, about 2.6 h.
- **Burst:** 40 transactions submitted at once (distinct keys, so no read/write conflicts), each with its own connection as in production. Gives per-transaction latency under load and the real fill of the blocks. 1,800 transactions, a few minutes in total.
- **Total run time:** about 3 h plus the network start. A run can be interrupted; completed blocks are kept and skipped on restart.
- Every transaction must commit as valid, and a seeded sample is read back and compared with what was sent.

## 8. Outputs (run folder `results/mac/<UTC>_mac-m1_colima-vz_fabric2.5.15/`)

- `manifest.json`, `journal.csv`, `blocks.csv` (one row per block of work with p50, p90, p99, mean, min, max, count), `quality.csv`, `canary.csv`, `monitor/`, `env/` (E0-style capture plus the network's image digests, chaincode hash and the calibration), `SHA256SUMS`, `console.txt`, `launcher/`.
- `raw/E14/<point>/<mode>-r<rep>.csv.gz`: one row per transaction (sequence, start time, connect, endorse, submit, commit, total, block number, index in block, validation code, envelope bytes) and `block,...` rows (block number, transactions, bytes, cut reason).
- Same privacy scan, checksum verification and INDEX row as every other run.

## 9. Run quality

- **Recorded, not repeated** (as E8, E9 and E13): latency here is dominated by the 2 s batch timer and the commit path, so the 5 % other-CPU rule would repeat blocks for no benefit. Recorded instead: host monitor samples, thermal, power, a pure-Go canary at the start, middle and end, and the tiny-control blocks. The analysis uses them to exclude or flag blocks, and says so.
- **Same machine rules as every Mac run:** on mains power, Low Power Mode off, thermal pressure nominal, apps closed, background services paused, Wi-Fi off during the run, no other containers (preflight refuses). The Fabric containers are not pinned to one CPU (they share Colima's 4 CPUs, as production shares its server).

## 10. Mac changes (recorded in the revert checklist before they happen)

- Docker images in Colima: the 6 pinned images, the 2 images the peers build for the chaincode, and the client image. About 2 GB.
- Docker volumes (`kpi-fabric-*`) for the certificates, the orderer, the peers and CouchDB, plus the usual per-run `kpi-out-*`. The containers keep their live names (for example `peer0.general.uae.com`) inside the compose project `docker` and carry the label `org.qchain.kpi=1`, so only ours are ever removed.
- Colima settings stay at 4 CPUs and 4 GiB. If the network needs more memory, I stop and ask before changing anything.
- Nothing is installed on the Mac itself; no Homebrew formula, no Go modules.

## 11. Phases and checks

| Phase | Deliverable | Check |
|---|---|---|
| A | The network on the Mac: certificates, channel, both peers joined, chaincode deployed, a few test transactions, blocks inspected | Topology equals the live one; shared CouchDB works; the block settings take effect; Colima memory is enough |
| B | Synthetic chaincode and Go client (unit tests in the Docker build), payload calibration, phases and block read-back | Calibration reproduces E13's envelope sizes within 1 %; every transaction valid |
| C | The launcher (preflight, monitor, results folder, checks), `smoke-e14` plan (3 points, few transactions) | Smoke run end to end; results verify; privacy scan clean |
| D | The main run (about 3 h), analysis hooks, documentation (methodology, data dictionary, decisions log, README) | Checked, then committed like every other run |

## 12. Risks and open items

1. **Both peers share one CouchDB in the live network.** It works there, but a fresh Mac copy might show revision conflicts or slower commits. Phase A tests it. If it misbehaves I stop and ask whether to use one CouchDB per peer (the usual layout).
2. **Colima memory (4 GiB).** About 10 containers: 3 CAs, orderer, 2 peers, CouchDB, 2 chaincode containers and the client. I expect about 1.5 GiB, which phase A measures.
3. **Internet during setup.** Image pulls and the chaincode's `npm install`; Wi-Fi must be on for phase A and off for the run.
4. **Ledger growth during a run** (thousands of transactions) can slow commits slightly; the shuffled order, the control blocks and the sequence index in the raw data show it.
5. **The first chaincode call is slow** (container start); covered by the warm-up.
6. **Envelope calibration** needs the transaction bytes from the block; if the match is worse than 1 % at some point I report it instead of forcing it.
