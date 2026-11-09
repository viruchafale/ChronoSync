# Phases 1–11 remediation ledger

Audit date: 2026-11-04; acceptance-closure update: 2026-11-06. This ledger records findings discovered during the comprehensive audit and closure pass; historical phase reports are not rewritten to imply these fixes always existed.

## Backlog and disposition

| ID      | Phase              | Priority                 | Finding/root cause/impact                                                                                                                                                                                                                                | Required fix and acceptance                                                                                                                                                                        | Disposition                                                                            |
| ------- | ------------------ | ------------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| AUD-001 | 11                 | P1                       | Release preflight counted any Developer ID identity but did not bind the explicitly selected identity and TeamIdentifier. An unrelated installed certificate plus arbitrary environment values could produce false readiness before later build failure. | Parse Developer ID inventory internally; require exact selected identity and final TeamIdentifier match; emit no names; add wrong-selection/team tests.                                            | `RESOLVED` — focused release tests/lint pass.                                          |
| AUD-002 | 10–11              | P1                       | Signed entitlements/profile were validated with XML substring searches; expected text could occur in a wrong field or with an extra access group. Profile expiry and signature timestamps were unchecked.                                                | Parse actual plists, require exact team/application/single access group and profile prefix, reject expired/malformed profiles, require timestamps on app/engine; add least-privilege/expiry tests. | `RESOLVED` — focused release tests/lint pass; real candidate remains external-blocked. |
| AUD-003 | 1–3 IPC            | P2                       | Legacy timeline/snapshot parameter decoders used permissive `json.Unmarshal` while later methods were strict. Unknown requested constraints could be silently ignored.                                                                                   | Use existing duplicate/unknown/trailing-data rejecting decoder for every implemented parameter object; reject nonempty inventory params; add legacy/snapshot regression test.                      | `RESOLVED` — uncached IPC race test passes.                                            |
| AUD-004 | 1–3, 5–9           | P2                       | Native GUI acceptance has remained pending; prior native builds/component tests are not a walkthrough. Current unbundled executable launches and registers frontmost with LaunchServices but is absent from the automation inventory.                    | Do not create a prohibited new bundle. Keep a precise disposable/manual checklist and retain manual status.                                                                                        | `MANUAL_ACCEPTANCE_PENDING`; startup confirmed, no workflow clicked.                   |
| AUD-005 | 2–3, 7, 9          | P2                       | Real disk-full, power-loss/fsync and hostile same-user race behavior is not safely reproducible; tests use transactional/synthetic fault injection.                                                                                                      | Preserve fault tests and honest labels; run only on disposable fault-injection infrastructure in a future authorized acceptance.                                                                   | `MANUAL_ACCEPTANCE_PENDING`; deterministic coverage strengthened by AUD-012/013.       |
| AUD-006 | 5                  | P2                       | Replay graph has structural absence of network/provider/process code, but OS-level denied-egress execution has not been demonstrated.                                                                                                                    | Keep source dependency tests; do not claim OS sandbox. Run under an authorized OS egress-denied harness when available.                                                                            | `IMPLEMENTED_BUT_UNVERIFIED`; no unsafe product change warranted.                      |
| AUD-007 | 1–3, 9             | P2                       | Linux/Windows compile paths exist but recorder supervision/storage/custody runtime support is intentionally incomplete.                                                                                                                                  | Cross-build engine; preserve explicit unsupported results and avoid portability claims.                                                                                                            | `EXTERNAL_BLOCKED` for runtime platform acceptance; amd64 builds pass.                 |
| AUD-008 | 10–11              | P0 if bypassed, external | No authorized Developer ID identity/team/profile exists; protected positive signing, A/B continuity, notarization and Gatekeeper cannot be accepted.                                                                                                     | Do not fabricate/bypass. Keep production on Phase 9 login custody and exact typed blockers.                                                                                                        | `EXTERNAL_BLOCKED`; zero valid identities reconfirmed.                                 |
| AUD-009 | 1–11 tooling       | P3                       | Pinned temporary Rust toolchain lacks `libLLVM.dylib` for `rust-objcopy`; release executable succeeds but stripping warns.                                                                                                                               | Use a complete reviewed Rust toolchain for an actual release; do not weaken build checks or create an artifact now.                                                                                | `EXTERNAL_BLOCKED` tooling limitation; non-fatal for no-bundle audit.                  |
| AUD-010 | 1–11 evidence      | P3                       | Historical performance evidence could become stale and opt-in tests are skipped by ordinary gates.                                                                                                                                                       | Rerun 100k timeline and snapshot workloads and record exact current results.                                                                                                                       | `RESOLVED` — 7.196 ms page query; all snapshot workloads pass.                         |
| AUD-011 | 1–11 documentation | P3                       | There was no single current audit matrix, remediation ledger or consolidated manual checklist.                                                                                                                                                           | Add matrix, ledger, checklist, status and comprehensive audit report; link from indexes.                                                                                                           | `RESOLVED` after final documentation check.                                            |
| AUD-012 | 7 recovery         | P2                       | Deletion had no focused tests for the two interruption windows after durable intent: failure before object removal and failure after removal but before tombstone.                                                                                       | Inject disposable permission failures in both windows; assert object/audit state, revision-2 pending state, revision-3 retry completion and idempotency under the race detector.                   | `RESOLVED` — both focused deletion-window tests pass.                                  |
| AUD-013 | 2 capacity         | P2                       | SQL-trigger rollback existed, but deterministic database-capacity exhaustion did not prove an event remains atomic when SQLite reports full.                                                                                                             | Cap a disposable SQLite database at its current page count, write bounded inline events until rejection, and assert exact count/sequence plus `quick_check=ok`.                                    | `RESOLVED` — focused storage race test passes; not real ENOSPC.                        |

## Detailed remediation evidence

### AUD-001 — exact release identity binding

- Reproduction: focused pure test showed readiness could be true with an inventory containing `Developer ID Application: Example (A1B2C3D4E5)` while `APPLE_SIGNING_IDENTITY` named a different identity.
- Root cause: readiness used only `developerIdApplicationCount > 0` plus nonempty environment values.
- Change: `parseIdentityInventory` retains names/team identifiers only in memory; `acceptanceReadiness` requires exact name and team equality. CLI output retains counts/blockers only.
- Regression: valid exact selection passes; unknown selection and wrong team each return a distinct blocker.

### AUD-002 — semantic signed metadata verification

- Reproduction: verifier accepted XML by substring and could not distinguish field placement, extra access groups or expired profiles.
- Root cause: credential-free Phase 10 implementation stopped at textual structural checks.
- Change: actual `codesign`/`security cms` plist output is converted through macOS `plutil`; pure validators require exact fields and a single access group. Both code signatures require hardened runtime, Developer ID authority, expected identifiers/team and a timestamp. Profile team/prefix/entitlements and future expiration are required.
- Regression: exact entitlement/profile succeeds; extra access group and expired profile fail.
- Limitation: no real signed candidate exists, so this is structural verifier remediation, not signed acceptance.

### AUD-003 — strict IPC parameters

- Reproduction: unknown fields were accepted by `run.query`, `run.cancel`, `event.list` and snapshot operations because Go's default decoder discarded them.
- Root cause: Phase 4E hardened the envelope and `run.start`; newer services adopted the helper, while historical methods remained permissive.
- Change: all timeline/snapshot parameter objects now use `strictRequest`; `object.inventory` requires an empty object.
- Regression: table-driven server/direct-dispatch checks reject unknown fields; valid existing integrations continue to pass.

### AUD-012 — interrupted governed deletion

- Reproduction gap: the service state machine already appended `deletion_pending` before removal, but tests did not isolate a removal failure after that append.
- Change: one disposable test removes dataset-directory write permission and proves the object remains pending; a complementary test removes retention-directory write permission after pending intent and proves a removed object remains audit-visible until tombstone retry.
- Regression: focused governance race test and full Go race suite pass.
- Limitation: deterministic permission injection is not physical power-loss, ENOSPC or fsync certification.

### AUD-013 — logical database-capacity exhaustion

- Reproduction gap: trigger-based SQL failure proved rollback but did not exercise SQLite's full-capacity result.
- Change: a disposable test limits `max_page_count`, appends bounded 7,000-byte synthetic inline messages until SQLite refuses growth, and verifies exact committed count/sequence and `PRAGMA quick_check`.
- Regression: focused storage race test passes.
- Limitation: SQLite logical page exhaustion is not filesystem ENOSPC, device failure or power loss.

## Priority closure

- P0 repository-fixable issues: none remained after preserving the external signing block; no bypass was attempted.
- P1 repository-fixable issues: AUD-001 and AUD-002 resolved.
- P2 repository-fixable issues: AUD-003, AUD-012 and AUD-013 resolved; AUD-004–007 require manual/external environments and retain explicit status.
- P3 repository-fixable issues: evidence/docs refreshed; the incomplete temporary release-stripping tool remains an external tooling limitation.

No test was disabled, weakened or deleted. No production data, identity, trust record, Keychain item or submitted artifact was modified.
