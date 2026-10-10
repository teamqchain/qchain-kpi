# Decisions log (harness)

The decisions about the study itself (scope, machines, run rules, thresholds) are in the research plan's decision log, `docs/plan.md` §19. This file records the decisions taken while building and testing the harness. Each one is dated and gives its reason.

| Date | Decision | Why |
|---|---|---|
| 10 Oct | Coding plan approved with the refinements of its §13, and Go 1.24.13 everywhere | production's builder image resolves to Go 1.24.13 |
| 10 Oct | liboqs pinned to the 0.16.0 release; liboqs-go `v0.16.0` | production builds an unpinned `main`; a study needs a fixed version |
| 10 Oct | DHKEM implemented by hand on OpenSSL 3.0 primitives and checked against the 8 RFC 9180 vectors | OpenSSL 3.0 (Debian bookworm) has no HPKE API |
| 10 Oct | Production functions copied verbatim (commit `9026aba`); only the signature algorithm name is parameterised | user decision: measure the real code, with provenance |
| 10 Oct | Input sets generated per (algorithm, repetition) by a separate untimed process | No crypto runs before the cold call. The repetitions include key-to-key variation. |
| 10 Oct | The probe pass uses its own small input set (P = 6) | a cold call plus 5 calls needs 6 inputs; full SLH-DSA-s pools would add minutes per algorithm |
| 10 Oct | Classical adapters perform no cryptographic operation when constructed. The RSA public key size is fixed at 422 bytes (SPKI DER, 3072-bit, e = 65537); the private key length is variable. | Constructing the RSA object used to generate a key, which would have run crypto before the cold call |
| 10 Oct | Raw samples are kept in memory during the timed phase and written afterwards | no file I/O or compression between samples |
| 10 Oct | Keygen correctness: every key's lengths are checked on the spot; a seeded subset (every 50th, at least 3) is fully exercised after the timed phase | approved subset rule. Deferring the full checks keeps their cache effects out of the timed samples. |
| 10 Oct | **Containers pinned to one CPU with no `--cpus` quota** (user decision) | In the Mac smoke run, a quota equal to the pinned CPU made Linux pause a fully busy container in 29 of 100 periods, about 1.8 ms each time. Every pause lands inside a timed sample, and the quota flag would have repeated every block. kpi-batch now refuses trial and main runs if a quota is set; the throttling check stays as a sanity check. |
| 10 Oct | Preflight for trial and main runs also refuses while Microsoft Edge is open (user decision) | Edge is the browser actually in use on the Mac; its tabs use CPU unpredictably |
| 10 Oct | Host-monitor chunks are delivered with `docker cp` every 10 s (a new chunk each minute); session facts arrive as `session.json` before every start or resume | no bind mounts (spaced paths, Colima without mounts), and no process runs inside the measurement container |
| 10 Oct | The Mac monitor skips `top`'s first two samples | the second sample still has empty per-process figures |
| 10 Oct | `pgrep -a` in the launcher's preflight | macOS `pgrep` skips its own ancestors by default, so VS Code was not seen when the launcher ran inside it |
| 10 Oct | Results are verified (`SHA256SUMS`) and privacy-scanned in `scratch/staging/` before they are moved into `results/` | a failed check must never leave files where they could be committed |
| 10 Oct | Mac preflight (trial and main) refuses while automatic macOS updates are on (`AutomaticDownload` or `AutomaticallyInstallMacOSUpdates`). It notes automatic App Store updates. | Night-time inventory: on this Mac both were on, so macOS could download an update and restart during a run and change the OS mid-study. Time Machine has no destination. Spotlight is on; the host monitor catches its activity. |
| 10 Oct | The host monitor and chunk delivery start at the very beginning of a session, before the clock-offset measurement | so the batch's first blocks are covered by warmed-up host samples |
| 10 Oct | The three Mac smoke runs (pipeline tests with tiny budgets) were deleted from `results/` after they had served their purpose (user decision). Their findings are recorded above: the CPU-quota throttling that led to the pin-only decision, the stop/resume checks, and the launcher fixes. | They are never used for results, and keeping them would clutter the record. |
