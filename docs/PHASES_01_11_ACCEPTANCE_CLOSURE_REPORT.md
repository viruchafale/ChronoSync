# Phases 1–11 remaining acceptance closure

Acceptance date: 2026-11-05; deterministic fault-coverage continuation: 2026-11-06. Environment: macOS 27.2 arm64, local working tree with preserved uncommitted Phase 8–11 and audit changes.

## Executive result

Every safe and available local acceptance task was completed without starting Phase 12, invoking a live provider, executing historical commands, changing production identities, or producing a new application bundle/DMG. One repository-test gap was closed: deterministic recovery after governance deletion is interrupted between durable intent and physical removal.

The product is not fully accepted. Actual native GUI workflows remain `MANUAL_ACCEPTANCE_PENDING` because the current unbundled executable is not exposed to the available GUI automation. Physical power-loss/disk-full acceptance remains `MANUAL_ACCEPTANCE_PENDING`. Credential-dependent Phase 10–11 acceptance remains `EXTERNAL_BLOCKED` because no valid Developer ID Application identity, reviewed TeamIdentifier/identity selection, rendered entitlements, or authorized provisioning profile is available.

## Native GUI acceptance by phase

| Phase                     | Result                      | Actual native evidence                                                                                                           | Remaining limitation                                                                |
| ------------------------- | --------------------------- | -------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------- |
| 1 recorder                | `MANUAL_ACCEPTANCE_PENDING` | Current unbundled executable launched, remained alive, registered in front with LaunchServices, and stopped normally.            | No automation surface; recorder start/stop/event/restart UI was not clicked.        |
| 2 timeline                | `MANUAL_ACCEPTANCE_PENDING` | No native workflow executed. Synthetic engine IPC integration passed timeline paging/filter/detail and restart persistence.      | IPC/component evidence is not GUI acceptance.                                       |
| 3 snapshots               | `MANUAL_ACCEPTANCE_PENDING` | No native workflow executed. Synthetic engine IPC integration passed capture, restore, ancestry/isolation and restart.           | Dialogs and visible native state were not exercised.                                |
| 4 policy/admission        | `MANUAL_ACCEPTANCE_PENDING` | Fail-closed Go/Rust/UI tests pass; no unauthorized launch path was enabled.                                                      | Native refusal/error presentation not clicked.                                      |
| 5 replay                  | `MANUAL_ACCEPTANCE_PENDING` | Synthetic fixture and UI tests pass without provider/process/network fallback.                                                   | Native selection/dialog workflow not clicked.                                       |
| 6 analysis                | `MANUAL_ACCEPTANCE_PENDING` | Comparison/report engine and renderer tests pass.                                                                                | Native report workflow not clicked.                                                 |
| 7 governance              | `MANUAL_ACCEPTANCE_PENDING` | Engine/UI tests and the new deletion-interruption regression pass.                                                               | Native confirmations and lineage/retention presentation not clicked.                |
| 8 exchange/trust          | `MANUAL_ACCEPTANCE_PENDING` | Package/trust adversarial and UI tests pass; packages remain non-executable.                                                     | Native file/confirmation dialogs not clicked.                                       |
| 9 custody                 | `MANUAL_ACCEPTANCE_PENDING` | Earlier disposable login-Keychain roundtrip evidence remains historical; current default Rust suite keeps the real test ignored. | No current native UI ceremony; no production item touched.                          |
| 10 identity/user presence | `EXTERNAL_BLOCKED`          | Unsigned refusal and typed structural tests pass.                                                                                | Positive OS-backed authentication requires an authorized signed entitled candidate. |
| 11 authorized release     | `EXTERNAL_BLOCKED`          | Credential-neutral preflight and exact verifier tests pass.                                                                      | No identity/profile, Candidates A/B, notarization, stapling or Gatekeeper evidence. |

### GUI investigation evidence

The executable at `apps/desktop/src-tauri/target/release/chronosync-desktop` launched successfully. Read-only process/LaunchServices inspection found the running process and a frontmost LaunchServices registration whose bundle and executable paths both pointed directly to that unbundled binary. The available computer-control inventory nevertheless exposed no ChronoSync app. The process was then stopped normally. No clicks, screenshots, GUI assertions, live agent calls, or workflow acceptance are claimed.

Creating a `.app` solely to make automation discovery easier was prohibited. The exact manual workflow remains in `NATIVE_GUI_ACCEPTANCE_CHECKLIST.md`.

## Physical fault and recovery acceptance

Existing deterministic coverage includes SQLite rollback/trigger failures, abrupt child exit with an open uncommitted transaction, restart recovery, interrupted snapshot preparation with orphan/temp evidence, retained partial restore output, corruption refusal and governance damage accounting.

This pass added `TestInterruptedDeletionRemainsPendingAndRetryCompletes` and, on continuation, `TestInterruptedDeletionAfterRemovalCommitsTombstoneOnRetry`. The first uses a disposable private dataset and Unix directory permissions to force physical removal to fail after durable deletion intent. It proves:

1. deletion returns an error;
2. the governed object remains present;
3. retention is durably `deletion_pending` at revision 2;
4. audit exposes that pending state;
5. restoring access and retrying removes the object and commits `deleted` revision 3; and
6. a further retry is idempotent.

The complementary test begins from durable `deletion_pending`, prevents tombstone publication while leaving object removal available, and proves that audit reports `PENDING_DELETION` after removal. Restoring retention-directory access lets a retry commit revision 3 without recreating or re-removing the object.

Phase 2 now also has `TestDatabaseCapacityFaultRollsBackEvent`. It caps a disposable SQLite database at its current logical page count, writes bounded inline synthetic events until SQLite refuses growth, then proves exact committed count/sequence and `PRAGMA quick_check=ok`.

Result: `PASS_WITH_LIMITATIONS`. This is deterministic permission fault injection, not real power loss, fsync failure, ENOSPC, controller loss, or hostile same-user interference.

### Remaining manual fault steps

Use only a disposable VM/volume and disposable ChronoSync data/workspace:

1. Record OS/filesystem, free space, app/engine version, test-data IDs and a pre-test backup/checksum inventory.
2. Constrain a separate volume to trigger ENOSPC during SQLite commit, payload publication, snapshot object publication/materialization and retention-event publication; restart after each point and run database integrity plus object/retention inventory.
3. Terminate only the disposable VM/process at the documented before-write, after-write/before-fsync and after-intent/before-removal points; never power-cut the user's host.
4. Copy and corrupt one disposable database/payload/CAS object at a time; verify fail-closed isolation and preservation of unrelated evidence.
5. For governed deletion, test interruptions before intent, after intent, after object removal and before the deleted tombstone; verify retry/audit semantics.
6. Record PASS/FAIL/BLOCKED per injection point and retain sanitized logs. Do not translate these results into guarantees for untested filesystems or hardware.

## Defect and remediation

`AUD-012` now covers both governed-deletion interruption windows. `AUD-013` closes the deterministic logical database-capacity gap. No production semantics, wire format, migration or dependency changed.

The unavailable GUI surface, lack of physical-fault infrastructure and absent signing credentials are environmental/manual blockers, not defects fixed by weakening security or creating prohibited artifacts.

## Validation commands and actual outcomes

| Command                                                                            | Outcome                                                                                                                                                                                                          |
| ---------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go -C engine test -race -count=1 ./internal/governance` with disposable `GOCACHE` | PASS, `ok`, 2.110 s.                                                                                                                                                                                             |
| Continuation: focused governance race suite                                        | PASS, `ok`, 2.257 s; both deletion interruption windows.                                                                                                                                                         |
| Continuation: focused storage race suite                                           | PASS, `ok`, 5.626 s; logical capacity rejection remained atomic and integrity-clean.                                                                                                                             |
| `go -C engine test -race -count=1 ./...` with disposable `GOCACHE`                 | PASS, fresh execution across every Go package; storage completed in 5.371 s.                                                                                                                                     |
| `GOCACHE=/private/tmp/chronosync-go-cache make check`                              | PASS: format, vet, full Go race suite/build, Prettier, ESLint, strict TypeScript, 8 Vitest files/34 tests, frontend build, docs, release config, schema/process integration and synthetic recorder/snapshot IPC. |
| First `make native-check`                                                          | FAIL before Rust checks: `cargo` absent from default `PATH`; invocation/tool discovery issue.                                                                                                                    |
| Pinned native check using the old disposable Cargo cache                           | FAIL: incomplete cache and denied network; retry exposed missing crate metadata. No repository failure.                                                                                                          |
| Pinned `make native-check` with fresh disposable Cargo cache                       | PASS: Rust format, locked Clippy with warnings denied, 6 tests passed/2 intentionally ignored, and optimized no-bundle executable build.                                                                         |
| `npm run release:preflight`                                                        | Completed; `readyForSignedCandidate:false`, 0 valid identities, 0 Developer ID Application identities and five typed blockers.                                                                                   |
| `security find-identity -v -p codesigning`                                         | Completed; 0 valid identities.                                                                                                                                                                                   |
| DMG `shasum` and `stat`                                                            | PASS: protected checksum, size and timestamp unchanged.                                                                                                                                                          |

The successful native command downloaded only locked dependencies into `/private/tmp/chronosync-acceptance-cargo`; it did not alter dependency manifests or create a bundle.

## Release-security result

Credential-neutral configuration and exact identity/team/entitlement/profile validation pass. Current preflight blockers are:

- `authorized_developer_id_application_identity_missing`
- `explicit_team_identifier_missing`
- `explicit_signing_identity_missing`
- `rendered_entitlements_missing`
- `authorized_provisioning_profile_missing`

Therefore protected user-presence success/cancel/deny, same-key migration, Candidate A/B continuity, notarization, stapling and clean Gatekeeper assessment are `EXTERNAL_BLOCKED`. Unsigned refusal remains the correct behavior. No certificate, profile, Keychain item, trust record or release artifact was created or changed.

## Exact remaining acceptance steps

1. On an authorized disposable macOS GUI environment, expose either the existing executable through a supported automation surface or explicitly authorize a separate non-release `.app`; execute every item in `NATIVE_GUI_ACCEPTANCE_CHECKLIST.md` and reconcile engine/database state after each workflow.
2. Run the disposable fault matrix above on a constrained VM/volume and append exact integrity/recovery evidence.
3. Supply an authorized Developer ID Application identity, exact TeamIdentifier, reviewed signing identity, rendered least-privilege entitlements and authorized unexpired profile.
4. Build Candidates A/B only under separate artifact authorization; verify signatures/entitlements/timestamps, actual user-presence success/cancel/deny, same-key migration and restart/upgrade continuity.
5. Notarize, staple and assess Gatekeeper only with explicit publication/service authorization and record Apple results separately.

## Final readiness assessment

Automated local developer-preview readiness is `PASS_WITH_LIMITATIONS`: repository gates pass and the newly covered deletion interruption recovers correctly. Native GUI readiness and physical-fault acceptance remain `MANUAL_ACCEPTANCE_PENDING`; protected signed release readiness remains `EXTERNAL_BLOCKED`. ChronoSync is not fully accepted for a production-secure macOS release. No Phase 12 work, commit, push, publication, app bundle or DMG creation occurred.
