# Phase 9 completion report

## 1. Overall Phase 9 status

Phase 9A–9F implementation is complete within the macOS login-Keychain custody boundary. Final acceptance evidence is recorded below.

## 2. Threat model

The dedicated threat model covers Keychain/application compromise, raw-memory exposure, arbitrary signing requests, key loss, migration interruption, trust corruption, revocation bypass and recovery ambiguity. It separates OS-backed storage, hardware custody, user presence, application identity and filesystem permissions.

## 3. Platform feasibility findings

Apple CryptoKit Ed25519 has no native Keychain `SecKey` corollary and Secure Enclave P-256 is incompatible. Login-Keychain generic-password Ed25519 works. The stronger data-protection/user-presence attempt returned `errSecMissingEntitlement` under the unsigned test binary.

## 4. OS-backed key custody architecture

Rust owns a narrow Keychain/Ed25519 broker; Go owns canonical one-use signing sessions, public identity, migration/trust state and final signature verification. React/TypeScript never receive secrets or arbitrary signing payloads.

## 5. Supported platforms

macOS login-Keychain custody is implemented and directly tested. Windows/Linux return explicit unsupported outcomes. No cross-platform or Secure Enclave custody is claimed.

## 6. Key storage guarantees

Seeds leave Phase 8 files after verified migration and reside as login-Keychain generic-password data. They materialize briefly in zeroized native memory for signing. The current build provides no per-sign OS authentication, non-extractable key or hardware guarantee.

## 7. Identity lifecycle

New OS-backed identities prove key possession before Go registration. Public identity v1/key IDs remain unchanged. Rotation creates a distinct identity, old signatures remain verifiable, revocation is terminal and trust is not restored by import.

## 8. Legacy migration state machine

The append-only state sequence is pending → Keychain created → identity verified → retirement pending → migrated, with explicit failed/recovery-required branches. Legacy retirement follows public-key equality and proof-signature verification.

## 9. Migration and restart safety results

Focused race tests passed for a complete migration, restart after identity
verification, idempotent retirement, interrupted pending state, fresh challenge
after restart, recorded failure/resume, concurrent refusal, mismatched public
key, invalid legacy seed, corrupt identity/history and unchanged historical
signature verification. No test treats Keychain and file publication as an
atomic transaction. Real power loss between every boundary remains unverified.

## 10. Recovery and rotation behavior

Missing/deleted keys are not reconstructable from public records. Safe recovery is a retained verified legacy seed or a new explicitly created rotation identity; no hidden backup/export exists.

## 11. Trust-store integrity behavior

Trust events receive strict canonical, timestamp, reason, identity, revision, transition and integrity checks. Audit fails closed on corruption and performs no automatic repair or trust restoration.

## 12. Security and privacy boundaries

No secret reaches renderer APIs, packages, reports or logs. No provider, command, agent, network, snapshot restore, admission or execution authorization is added. Same-user/trusted-process compromise and coherent trust rollback remain possible.

## 13. Database, protocol and dependency changes

SQLite remains migration 3 and protocol remains 1. Engine advances to 0.8.0 with additive `key-custody-recovery-v1`. Native dependencies are pinned to `security-framework` 3.7.0, `ed25519-dalek` 3.0.0, `sha2` 0.11.1 and `zeroize` 1.9.1.

## 14. Files and modules changed

Added Go custody/migration/audit contracts/tests, additive IPC, Rust Keychain broker/orchestration/tests, TypeScript/React custody controls/tests, ADR-0014, plan/threat/specification, six status files and this report. Updated cumulative contracts/architecture/security/testing/index/changelog guidance.

## 15. Focused and adversarial test results

`go test -race ./internal/exchange ./internal/ipc` passed. The focused React
suite passed 5/5. Rust passed 4 default tests with the real Keychain case
explicitly ignored, then the separately authorized actual macOS round trip
passed 1/1 and deleted its disposable item. The first real test inside the
workspace sandbox returned Security.framework `-50`; repeating it with actual
Keychain access passed. No provider or historical command ran.

|   # | Requested scenario                    | Evidence/result                                                                                              |
| --: | ------------------------------------- | ------------------------------------------------------------------------------------------------------------ |
|   1 | OS-backed identity creation           | Real Keychain round trip plus native registration path; passed                                               |
|   2 | Public-key retrieval                  | Real/fake broker retrieval; passed                                                                           |
|   3 | Successful signing                    | Real/fake broker signing; passed                                                                             |
|   4 | Signature verification                | Go registration/export proof verification; passed                                                            |
|   5 | Key access denial                     | Fake broker typed denial; passed                                                                             |
|   6 | Missing key                           | Fake broker missing state; passed                                                                            |
|   7 | Deleted key                           | Fake and real create/delete checks; passed                                                                   |
|   8 | Malformed signing request             | Empty/oversized or malformed signature refusal; passed                                                       |
|   9 | Unsupported platform                  | Explicit unsupported broker contract; passed                                                                 |
|  10 | Identity rotation                     | Existing Phase 8 rotation/trust test plus custody registration; passed                                       |
|  11 | Valid legacy migration                | Complete migration test; passed                                                                              |
|  12 | Public/private mismatch               | Migration proof mismatch refusal; passed                                                                     |
|  13 | Invalid seed                          | Invalid legacy seed refusal; passed                                                                          |
|  14 | Interrupt before key creation         | Pending-state restart test; passed                                                                           |
|  15 | Interrupt after key creation          | Recovery state transition test; passed                                                                       |
|  16 | Interrupt after identity verification | Restart/resume test; passed                                                                                  |
|  17 | Interrupt before legacy retirement    | Retirement-pending/idempotency test; passed                                                                  |
|  18 | Restart during migration              | Fresh challenge and persisted-state test; passed                                                             |
|  19 | Concurrent migrations                 | Busy/conflict race test; passed                                                                              |
|  20 | Migration idempotency                 | Completed/retirement retry test; passed                                                                      |
|  21 | Preserve old signatures               | Pre-migration signature compatibility test; passed                                                           |
|  22 | Trust restart persistence             | Existing Phase 8 restart test rerun; passed                                                                  |
|  23 | Revocation persistence                | Existing revoked restart outcome rerun; passed                                                               |
|  24 | Key loss                              | Missing typed state/no reconstruction test; passed                                                           |
|  25 | Identity corruption                   | Corrupt identity migration refusal; passed                                                                   |
|  26 | Trust corruption                      | Canonical/integrity audit refusal; passed                                                                    |
|  27 | Re-import revoked identity            | Existing revoked import refusal rerun; passed                                                                |
|  28 | Trust recovery refusal                | Corrupt history audit performs no repair; passed                                                             |
|  29 | Rotation without inherited trust      | Rotation/trust separation test; passed                                                                       |
|  30 | Explicit recovery confirmation        | Native migration confirmation is wired and component-tested; actual dialog automation unverified             |
|  31 | Phase 8 export                        | One-use native signing path plus exchange regression; passed                                                 |
|  32 | Phase 8 verification                  | RFC vector/tampering/compatibility tests; passed                                                             |
|  33 | Phase 8 import                        | Clean-store IPC workflow; passed                                                                             |
|  34 | Trusted inspection                    | Trusted outcome regression; passed                                                                           |
|  35 | Untrusted confirmation                | Existing native-confirmed path compiled and component regression passed; actual dialog automation unverified |
|  36 | Invalid-signature refusal             | Tampering test; passed                                                                                       |
|  37 | Revoked-identity refusal              | Revoked exchange test; passed                                                                                |
|  38 | Native UI error handling              | Component-visible Keychain denial test; passed; native window automation unverified                          |
|  39 | Engine restart/recovery               | Migration/trust/clean-store restart tests; passed                                                            |
|  40 | Full regression                       | `make check` and `make native-check`; passed                                                                 |

Source audits found only the expected metadata terms in TypeScript and no
exchange dependency on network, process launch, provider, policy, admission or
supervisor packages. `go mod verify` passed. `npm audit --offline --json`
reported zero vulnerabilities across 253 dependencies. Cargo's locked tree
resolved direct native dependencies to `security-framework` 3.7.0 and
`ed25519-dalek` 3.0.0. This is not a third-party Rust vulnerability-database
scan because no such audited tool/database was installed.

## 16. Full `make check` results

Passed on 2026-11-03. It ran Go formatting, vet, all race tests and engine build;
Prettier, ESLint, strict TypeScript, all 34 Vitest tests, frontend production
build, documentation validation, executable/schema integration and synthetic
recorder/snapshot/IPC integration. The first attempt correctly stopped on three
Prettier findings; those files were formatted and the full rerun passed.

## 17. Full `make native-check` results

Passed on 2026-11-03: sidecar build, Rust format, Clippy with warnings denied,
four default Rust tests and Tauri `--no-bundle` release build. The first attempt
correctly stopped on Rust formatting and the complete rerun passed. The final
release build warned that `rust-objcopy` in the temporary toolchain could not
load `libLLVM.dylib`; stripping was skipped, but Cargo/Tauri returned success
and produced the executable. No bundle command ran.

## 18. Native GUI acceptance status

Native confirmation/dialog automation remains unverified. The direct Keychain API integration test is separate and passed.

## 19. Remaining limitations and unverified behavior

Per-sign user presence requires stable signing/entitlements and is blocked in the current build. Account/device migration, password reset, Keychain repair, stable Developer ID continuity, real crash/power-loss windows and cross-platform behavior remain unverified.

## 20. Submitted DMG checksum confirmation

The existing DMG remains 9,479,647 bytes with modification time
`2026-10-25T21:17:17+0530` and SHA-256
`e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`,
matching the Phase 8/submission evidence. It was not rebuilt or replaced.

## 21. Recommended Phase 10 direction

Phase 10 requires separate authorization. A safe direction is release-identity/entitlement engineering for data-protection Keychain user-presence enforcement, beginning with signed disposable acceptance and upgrade semantics from login-Keychain custody. Do not infer organization PKI, synchronization, remote recovery or execution authorization.
