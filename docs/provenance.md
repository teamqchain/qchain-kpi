# Provenance

This file records where every piece of code, library and test data in the harness comes from. Each source is pinned so that a result can be traced to exact inputs.

## QChain production code copied verbatim

The harness measures QChain's own functions. It does not reimplement them. They were copied from the QChain main repository at commit `9026aba` (decision of 10 Oct 2026: "copy the real functions").

**Source repositories since 9 November 2026.** `9026aba` is a commit of the former single repository `QChain-PQC-blockchain` (28 Sep 2026, "QWallet: Some build configuration"; its twin on GitHub `main` was `ad271e1`). That repository was split into separate repositories under `teamqchain`. Every file cited in this document as `<path> @ 9026aba` exists byte for byte, with the same line numbers, in the new repositories at these commits (checked on 10 Nov 2026):

| Cited path @ 9026aba | Now | Commit |
|---|---|---|
| `offchain/<file>` | `qchain-backend`, `<file>`. At that commit the Go files sat at the repository root; later commits moved them into packages, so cite this commit, not `main`. | `d2c0629` |
| `qchain-network/<path>` | `qchain-network`, `<path>` (the classical network; this commit carries the tag `classical-network-final`) | `35edaf0` |
| `UI_App/<path>` | `qchain-wallet`, `<path>` | `135bbf9` |

`9026aba` itself exists only in the local clone of the former repository on the Mac, which `scripts/check-sync.sh` reads. The commit IDs above are those of the new repositories, whose histories were rewritten by the split.

The only edits are:
- the package clause (`main` → `qchain`);
- trimmed import blocks;
- the single parameterisation listed below.

Every copied file starts with a header that names its source.

| Harness file | Source (main repo @ 9026aba) | Edits |
|---|---|---|
| `internal/qchain/commitment_prod.go` | `offchain/commitment.go`, entire file | package clause only |
| `internal/qchain/crypto_prod.go` | `offchain/crypto.go` lines 29–80 (`sigName`, `pqcSign`, `pqcVerify`, `sha3Hex`) | `sigName` is a `var` instead of a `const`, so the harness can select the signature algorithm (`SetSignatureAlgorithm`, `SetSuite`). Production hard-codes ML-DSA-44. **Decision A (11 Oct):** `pqcSign`/`pqcVerify` are renamed `pqcSignLiboqs`/`pqcVerifyLiboqs` (bodies unchanged); `dispatch.go` defines the original names (see below). |
| `internal/qchain/config_prod.go` | `offchain/config.go` line 22 (`import _ "time/tzdata"`) and lines 102–110 (`mustLoadLocation`) | trimmed imports |
| `internal/qchain/chain_prod.go` | `offchain/chain.go` lines 42–76 (`ChainCredential`, `ChainHolder`) and 277–301 (`dubaiDate`, `isExpired`, `effectiveStatus`) | trimmed imports |
| `internal/qchain/chain_snapshot_prod.go` | `offchain/chain.go` lines 93–106 (`ChainSnapshot`, `holderOf`) | package clause only |
| `internal/qchain/envelope_prod.go` | `offchain/envelope.go`, entire file | package clause only |
| `internal/qchain/verification_prod.go` | `offchain/verification.go` lines 33–221 (failure reasons, `DisclosedPayload`, `stringMap`, `parseDisclosedPayload`, `VerifyInput`, `VerifyOutcome`, `verifyPresentation`, `presentationRequest`, `validatePresentationRequest`) | trimmed imports |
| `internal/qchain/httputil_prod.go` | `offchain/httputil.go` lines 37–40 (`decodeBody`) | trimmed imports |
| `internal/qchain/mobile_prod.go` | `offchain/mobile.go` lines 33–37 (expected hex lengths of the wallet's public keys) | **Decision B (11 Oct):** `const` became `var`; `SetSuite` sets the lengths of the measured suite (production hard-codes ML-KEM-768 and ML-DSA-44). |
| `internal/qchain/wallet_prod.go` | keygen tool `offchain/cmd/keygen/main.go` lines 36–133 (`decryptEnvelope`, the Go mirror of QWallet's `CryptoService.decryptEnvelope`) | trimmed imports; **decision A:** one marked edit routes the KEM decapsulation step to the OpenSSL twin for DHKEM; for liboqs KEMs the original lines run, of which only `ss, err :=` became `ss, err =`. |
| `internal/qchain/kem_prod.go` | `offchain/kem.go`, entire file (`kemName`, `kemGenerateKeypair`, `kemEncap`, `kemDecap`, `deriveKey`, the AES-GCM helpers) | package clause only. `kemName` is already a `var` in production; the harness selects the KEM with `SetKEMAlgorithm`/`SetSuite`. **Decision A:** `kemGenerateKeypair`/`kemEncap`/`kemDecap` are renamed `...Liboqs` (bodies unchanged). |

`scripts/check-sync.sh` checks every copy against the main repository at `9026aba`: it undoes the documented edits in memory and requires every copied line range to appear unchanged and in one piece (11 copies, all matching on 12 Oct).

**Classical suites (decision A, 11 Oct): `internal/qchain/dispatch.go`.** The copied code calls four liboqs helpers. Their bodies are kept under `...Liboqs` names; `dispatch.go` defines the original names. For a liboqs algorithm they call the copied body directly; for a classical one they run the same pattern on OpenSSL (hex-decode, a fresh key object, the operation, free the object, hex-encode). Where production passes only a secret key but OpenSSL needs the public half (ECDSA import, RFC 9180 DHKEM decapsulation), the twin looks it up in a table filled when the key pair was generated. E2's `prod` arm (`prod.go`) calls the `...Liboqs` bodies directly, so its measured path does not go through the dispatcher.

**Harness mirrors (not copies, written for the harness):**
- **`internal/qchain/mirrors.go` (E11):** production steps written inline in HTTP handlers, restated in the same order with the same calls, minus the excluded I/O (each left-out line is named in the code):
  - `ActivationCheck`: `handleRegisterHolderKeys` (`mobile.go` 86–123): body decoding and the key checks, with the suite's key lengths;
  - `Issue`: `handleIssueCredential` (`credentials.go` 140–209): attributes, salts and field hashes, envelope, Dubai `issuedAt`, commitment, signature, field-hash JSON; the IPFS CID is passed in;
  - `EnvelopeCheck`: `looksLikeEnvelope` as in `handleGetEnvelope` (`mobile.go` 178);
  - `WalletKeygen`, `WalletSign`: the wallet's key generation and presentation signature (`crypto_service.dart` `generateKemKeyPair`, `generateSigningKeyPair`, `buildSignedDisclosedPayload`, `signPayload`), whose steps equal production's `kemGenerateKeypair`, the keygen tool's `dsa` case and `pqcSign`;
  - `PresentationCheck`: `validatePresentationRequest` as in `handleGeneratePresentation` (`mobile.go` 483);
  - `ChainResponse` / `DecodeChainResponse`: the chaincode's `getCredentials` response built byte for byte (sorted keys, `JSON.stringify({credentials, holders})`; `QChaincode.js` 211–231 and 263) and the backend's decoding of it (`chain.go` 200–204, 216–225);
  - `Verify`: `verifyPresentation` as called by `handleResolveSession` (`mobile.go` 769–778).
- **`internal/qchain/prod.go`:** exported entry points to the copied functions for E2's prod arm, plus `ProdSigKeygen`, a mirror of production's signature key generation. Production has no function for it: the steps are inline in the keygen tool (`offchain/cmd/keygen/main.go` lines 162–174, the `dsa` case, and the same steps at lines 222–238 @ 9026aba). The mirror keeps their order (`Init` with no key, `GenerateKeyPair`, `ExportSecretKey`, hex-encode both keys, deferred `Clean`), takes the algorithm from `sigName`, and returns errors instead of calling `log.Fatalf`.
- **`internal/qchain/payload.go`:** a Go mirror of the wallet's presentation payload (Dart, `UI_App/lib/services/crypto_service.dart`, `buildSignedDisclosedPayload`, lines 619–673 @ 9026aba). It builds the payload as sorted, compact JSON with no HTML escaping. Timestamps follow the format of Dart's `DateTime.toIso8601String()` for local time. Synthetic attribute values are ASCII alphanumerics only, so Go and Dart encode them identically.
- **`internal/qchain/messages.go`:** generators of realistic inputs, built with the copied functions:
  - attributes `field01…`, 16-character values, 16-byte salts;
  - a well-formed CIDv0 standing in for the IPFS CID;
  - Dubai-time `issuedAt`;
  - `expiryDate` present.

**Golden checks (`internal/qchain/qchain_test.go`):**
- **Commitment:** the copied code reproduces production's golden canonical commitment from `offchain/commitment_test.go` lines 21–52 @ 9026aba, byte for byte.
- **CID pattern:** production's pattern (`offchain/crypto.go:92`) is restated in the test. The CIDs generated by the harness match it.


## E14: QChain's Fabric network (QChain commit `71a1db6`, 25 Sep 2026)

Copied **byte for byte** (they run unchanged inside the E14 containers; `scripts/e14-network-mac.sh` mounts them at the live paths):

| QChain file | Harness copy | SHA-256 (first 16 hex) |
|---|---|---|
| `qchain-network/config/configtx.yaml` | `fabric/network/config/configtx.yaml` | `8b9def5d4b8cbfce` |
| `qchain-network/config/core.yaml` | `fabric/network/config/core.yaml` | `7471c8a6983caf13` |
| `qchain-network/fabric-ca/government/fabric-ca-server-config.yaml` | `fabric/network/fabric-ca/government/fabric-ca-server-config.yaml` | `89b058d7ceb9b483` |
| `qchain-network/fabric-ca/general/fabric-ca-server-config.yaml` | `fabric/network/fabric-ca/general/fabric-ca-server-config.yaml` | `4de6f1e3641aebc8` |
| `qchain-network/fabric-ca/orderer/fabric-ca-server-config.yaml` | `fabric/network/fabric-ca/orderer/fabric-ca-server-config.yaml` | `c5f6837beed3bb58` |
| `qchain-network/scripts/registerEnroll.sh` | `fabric/network/scripts/registerEnroll.sh` | `68b74264a8216d77` |
| `qchain-network/scripts/env-gov.sh` | `fabric/network/scripts/env-gov.sh` | `e4d006dc17f2e12a` |
| `qchain-network/scripts/env-gen.sh` | `fabric/network/scripts/env-gen.sh` | `98d9b6072b766efc` |

Derived (not copies; each says what it follows):

| Harness file | Follows | Differences |
|---|---|---|
| `scripts/e14-network-mac.sh` | `qchain-network/docker/docker-compose.yaml` and `docker-compose-ca.yaml`, `scripts/start-fabric-network.sh` | plain `docker run` (no compose on the Mac); images pinned (`:latest` live; the live peer is v2.5.15, from the 13 Oct capture); no published ports; files in the volume `kpi-fabric-material`, mounted with volume sub-paths at the live paths; same names, environment variables and shared CouchDB |
| `fabric/network/scripts/channel.sh` | `docs/setup.md` "Create and Join Channel (Using OSNAdmin)" | container names instead of `localhost`; waits until the orderer reports the channel active |
| `fabric/network/scripts/deploy-cc.sh` | `docs/setup.md` "Deploy Chaincode" | the synthetic chaincode `kpipayload` 1.0 instead of `qchaincode` 2.0; package id from `calculatepackageid`; seeds the holder record as `issuer1` |
| `fabric/chaincode/KpiPayload.js` | `qchain-network/chaincode/QChaincode.js` `issueCredential`, `_checkAccess`, `_exists`, `_put` | the payload instead of the credential fields; same libraries (versions pinned to what QChain's ranges resolve to: fabric-contract-api and fabric-shim 2.5.8, json-stringify-deterministic 1.0.18, sort-keys-recursive 2.1.10) |
| `fabric/client/gateway.go` | `offchain/fabric.go` `getContract`, `chain.go` `withContract` and `chainSubmit` | `SubmitTransaction`'s steps taken one by one so that each is timed; identity and TLS files read once; same library versions (`offchain/go.mod`) |
| `fabric/client/canary.go` | the harness's `internal/canary/canary.go` | package and identifier names only |

## Live-ledger capture and the facts E13 relies on

- **Capture** (`internal/ledger/vmcapture/`, details and hashes in its README): the General peer's block file (Fabric v2.5.15, 61 blocks) and the IPFS store (Kubo 0.40.1), read-only on 13 Oct 2026; exported as sizes only. `go test ./internal/ledger` and every E13 block reproduce it to the byte.
- **Fabric** (v2.5.15 source, read at tag `v2.5.15`): block-file layout `common/ledger/blkstorage/block_serialization.go` (`serializeBlock`, `addHeaderBytes`, `addDataBytesAndConstructTxIndexInfo`, `addMetadataBytes`) and the length prefix in `blockfile_mgr.go`; block cutting in `orderer/common/blockcutter` (payload + signature counted against PreferredMaxBytes; at most MaxMessageCount per block).
- **QChain network** (`qchain-network/config/configtx.yaml` @ 9026aba): etcdraft, BatchTimeout 2 s, MaxMessageCount 10, PreferredMaxBytes 512 KB; two peer organisations (GovernmentMSP, GeneralMSP).
- **Chaincode** (`qchain-network/chaincode/QChaincode.js` @ 9026aba): the arguments of every call, the keys it reads (`_exists` / `_getHolder` / `_getCredential`) and writes (`_put`: `sortKeysRecursive`, `JSON.stringify`), the fields the state changes add (`SuspendedAt`, `SuspendedReason`, `RestoredAt`, `RevokedAt`, `ExpiryUpdatedAt`; `_txISO` is second precision, UTC), and the responses (`{"success":true,"credentialID":...}`, `{"success":true,"holderID":...}`, the holder record). Modelled in `internal/ledger/calls.go`, checked against all 57 captured QChain transactions.
- **Backend IPFS upload** (`offchain/crypto.go` `uploadJSONToIPFS` @ 9026aba): go-ipfs-api `Shell.Add` with defaults, once per issuance (the envelope only).

## Libraries

| Component | Version / pin | Where pinned |
|---|---|---|
| liboqs | tag `0.16.0`, commit `5a1a854b0dc9f2141bdc771c555ee60c37950183` (asserted after clone) | `docker/Dockerfile`, `docker/oqs-build.sh` |
| liboqs-go | `v0.16.0` | `go.mod`, `go.sum` (hash matches sum.golang.org) |
| golang.org/x/crypto | `v0.28.0` (same as production) | `go.mod`, `go.sum` |
| Go toolchain | 1.24.13, image `golang:1.24.13-bookworm@sha256:1a6d4452c65dea36aac2e2d606b01b4a029ec90cc1ae53890540ce6173ea77ac` (production's `golang:1.24-bookworm` resolves to 1.24.13) | `docker/Dockerfile`; every binary refuses to run under another version |
| Runtime base | `debian:bookworm-slim@sha256:7c7b2c966bc9ee8cedfeef67e0e279108992c77681fa595db4a9d65c06ccc587` | `docker/Dockerfile` |
| Debian packages (gcc 12, cmake, ninja, OpenSSL 3.0, pytest …) | snapshot.debian.org `20261009T000000Z`, falling back to live mirrors only if the snapshot service is unreachable | `docker/apt-setup.sh`. The source used and every package version are recorded in `env/build-info/apt-source-*.txt` and `dpkg-*.tsv`; the manifest flags a live-mirror build. |

**Build configuration:**
- The liboqs CMake flags of each build variant are in `docker/oqs-build.sh`.
- Each image records the resolved configuration: `oqs-build.txt`, `oqs-CMakeCache.txt` and `oqsconfig.h`. All three are copied into every run's `env/build-info/`.
- Production builds liboqs from an unpinned `main`. The harness pins the 0.16.0 release instead and records this as a deliberate difference.

## Test data

| Data | Source | Integrity |
|---|---|---|
| RFC 9180 DHKEM vectors (`testdata/rfc9180/dhkem-vectors.json`): 8 vectors for mode 0 and KDF HKDF-SHA256, 4 for kem_id 16 (P-256) and 4 for kem_id 32 (X25519) | `https://raw.githubusercontent.com/cfrg/draft-irtf-cfrg-hpke/5f503c564da00b0687b3de75f1dfbdfc4079ad31/test-vectors.json` | source SHA-256 `61fc662f01996cd06d713dacf5e133167bd309a1f329442d53f1e21a47b3ede6` |
| NIST ACVP vectors used by liboqs's own tests (ML-KEM, ML-DSA, SLH-DSA; 8 files) | `https://raw.githubusercontent.com/usnistgov/ACVP-Server/refs/tags/v1.1.0.41/gen-val/json-files/` | SHA-256 per file in `docker/acvp-sha256.txt`. The build verifies each file, then serves the verified copy to the tests (`docker/acvp_conftest.py`). |

## Correctness gates (every image build, on every machine)

The image build fails if any of these fails:

1. **liboqs.**
   - All 25 post-quantum algorithms must be compiled in (`oqsconfig.h`).
   - liboqs's own pytest suites must pass:
     - `test_cmdline.py`: round trips;
     - `test_kat.py`: known-answer tests;
     - `test_acvp_vectors.py`: NIST ACVP.
   - Output: `oqs-gate.txt`.
2. **Harness.** `go vet` and `go test` of the whole harness. Output: `go-test.txt`. This covers:
   - the 8 RFC 9180 vectors;
   - round trips of all 30 adapters, including that keys survive liboqs-go's `Clean`;
   - the QChain golden checks;
   - the measuring process end to end (input sets, E1, E10, the probe, and the detection of a wrong result);
   - the orchestrator's flag, repeat, headline and resume rules;
   - the plan expansion, the checksum files and the session allow-list.
3. **At run time.**
   - Every process refuses a Go toolchain other than 1.24.13 or a liboqs other than 0.16.0 with all 25 algorithms.
   - Every measured result is checked for correctness outside the timed region.
