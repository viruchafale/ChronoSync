# Phases 1–11 comprehensive audit and remediation report

Audit date: 2026-11-04.

## 1. Executive summary

`PASS_WITH_LIMITATIONS`

The current repository implementation, tests, phase plans, ADRs, reports, migration history, native boundary and protected DMG were re-audited against Phases 1–11. Three repository-fixable defects were reproduced and fixed: release preflight did not bind the selected Developer ID identity/team; release verifier used unsafe substring checks and omitted profile expiry/signature timestamp enforcement; and legacy timeline/snapshot IPC silently ignored unknown fields. Focused regressions pass.

No repository-fixable P0 remains. All repository-fixable P1 findings are resolved. Remaining issues require a real GUI/manual fault environment, unsupported operating-system runtime work beyond existing claims, or authorized Apple credentials. The project is not fully accepted: native GUI workflows and signed protected/notarized release security remain pending/blocked.

## 2. Phase-by-phase audit results

| Phase                    | Result                  | Current evidence and limitation                                                                                                                                    |
| ------------------------ | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| 1 recorder               | `PASS_WITH_LIMITATIONS` | Recorder/process/restart tests pass; current native workflow walkthrough is manual-pending and live provider capture was prohibited.                               |
| 2 durable timeline       | `PASS_WITH_LIMITATIONS` | Integrity/migration/recovery tests pass; fresh 100k page query is 7.196 ms. Physical power-loss/disk-full acceptance remains pending.                              |
| 3 snapshots/branches     | `PASS_WITH_LIMITATIONS` | CAS/restore/branch/recovery and opt-in workloads pass; native GUI and physical fault acceptance remain pending.                                                    |
| 4 policy/admission       | `PASS_WITH_LIMITATIONS` | Production policy requests refuse before launch and authority fields fail closed. Positive enforcement/authentication remains intentionally blocked.               |
| 5 replay fixtures        | `PASS_WITH_LIMITATIONS` | Fixture/transport/unified tests and source isolation pass; native dialogs and OS-level egress denial remain unverified.                                            |
| 6 analysis               | `PASS`                  | Stable observable comparisons, append-only reports, explicit gaps and non-causal findings pass; no live boundary exists.                                           |
| 7 governance             | `PASS_WITH_LIMITATIONS` | Immutable datasets/lineage, reference audit and non-cascading deletion pass; physical crash window/native dialogs remain unverified.                               |
| 8 exchange/trust         | `PASS_WITH_LIMITATIONS` | Ed25519 package/trust/revocation/adversarial tests pass; native file/confirmation dialogs remain manual-pending.                                                   |
| 9 login-Keychain custody | `PASS_WITH_LIMITATIONS` | Real disposable login-Keychain round trip and migration/recovery regressions pass; Windows/Linux and physical/account migration are unsupported/unverified.        |
| 10 release/user presence | `BLOCKED`               | Structural verifier and exact unsigned `-34018` refusal pass; audit hardened identity/plist/expiry/timestamp checks. Positive signed evidence is external-blocked. |
| 11 authorized acceptance | `BLOCKED`               | Exact credential-neutral blockers pass; zero valid identities. Candidates A/B, protected positive migration, notarization and Gatekeeper remain external-blocked.  |

The criterion-level evidence is in [the audit matrix](PHASES_01_11_AUDIT_MATRIX.md).

## 3. Complete issue inventory

The authoritative inventory, priorities, root causes, changes, tests and remaining blockers are in [the remediation ledger](PHASES_01_11_REMEDIATION_LEDGER.md). Summary: 2 P1 resolved, 1 P2 resolved, 4 P2 manual/external limitations retained, 1 external signing blocker, and evidence/documentation/tooling P3 items addressed or explicitly limited.

## 4. Issues resolved

### Exact release identity readiness

Preflight now matches the exact external signing identity against installed Developer ID Application inventory and verifies the supplied TeamIdentifier matches the identity suffix. Wrong selection/team produce distinct blockers. CLI output still exposes only counts and blockers.

### Semantic release metadata verification

Actual signed entitlements and decoded provisioning profiles are parsed as plists. Validation requires exact team, application identifier and one reviewed Keychain access group; profile team/prefix and future expiration; and Developer ID/hardened-runtime/timestamped app and engine signatures. Extra access groups and expired profiles fail focused tests.

### Strict IPC parameters

Timeline and snapshot methods now use the same decoder as newer services: duplicate/case-alias/unknown/trailing values fail instead of being silently discarded. Valid protocol requests and integrations remain compatible.

## 5. Files changed and rationale

- `scripts/macos-release-identity.mjs` and focused check: exact identity binding and semantic signed-metadata verification.
- `engine/internal/ipc/server.go`, `snapshots.go`, `server_test.go`: strict historical parameters plus regression coverage.
- Audit matrix, remediation ledger, manual GUI checklist, this report and separate implementation status: durable evidence/accountability.
- Cumulative README/status/testing/security/release/changelog/AGENTS documentation: link and accurately scope the remediation.

All pre-existing Phase 8–11 uncommitted work was preserved.

## 6. Tests added or modified

- Release tests: exact identity selection, team mismatch, parsed signature timestamp, single access group, exact profile team/prefix and profile expiration.
- IPC test: unknown fields rejected for timeline/cancel/event and all snapshot parameter shapes.
- No test was removed, skipped, weakened or converted to a stub.

## 7. Validation commands and actual outcomes

| Command/check                                            | Outcome                                                                                                                      |
| -------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `go test -race -count=1 ./...`                           | PASS, every Go package                                                                                                       |
| Focused IPC race test                                    | PASS                                                                                                                         |
| `CHRONOSYNC_LOAD=1 ... TestLoad100k`                     | PASS; query 7.196 ms, 100k events                                                                                            |
| `CHRONOSYNC_SNAPSHOT_LOAD=1 ... TestSnapshotPerformance` | PASS; all 3 workloads                                                                                                        |
| Linux/Windows amd64 Go cross-builds                      | PASS                                                                                                                         |
| Release config/preflight tests                           | PASS; zero identities and five exact prerequisites                                                                           |
| Real unsigned protected-Keychain test                    | PASS; expected `-34018`                                                                                                      |
| Real disposable login-Keychain round trip                | PASS                                                                                                                         |
| npm offline audit                                        | PASS; zero reported vulnerabilities across 253 dependencies                                                                  |
| `go mod verify`                                          | PASS                                                                                                                         |
| Full `make check`                                        | PASS: format/vet/race/build, ESLint/TypeScript/34 Vitest tests/Vite, docs, release, schema and recorder/snapshot integration |
| Full `make native-check`                                 | PASS: sidecar, Rust format/Clippy, 6 passed + 2 default-ignored tests, no-bundle executable; known non-fatal strip warning   |

`govulncheck`, `cargo-audit` and `cargo-deny` are not installed, so Go/Rust vulnerability-database scanning is `NOT_REPRODUCIBLE` in this environment; module verification, npm audit and locked dependency/source review were performed.

## 8. Native GUI acceptance results

`MANUAL_ACCEPTANCE_PENDING`

The current unbundled native executable launched and remained alive without terminal diagnostics. The automation inventory could not bind an accessibility surface for it. Creating a new app/DMG artifact solely for the audit is prohibited, and the submitted DMG contains the historical submission build rather than the current Phase 8–11 UI. No click workflow was fabricated. The exact remaining steps are in [the native GUI checklist](NATIVE_GUI_ACCEPTANCE_CHECKLIST.md).

## 9. Security review results

- Policy-bearing execution still refuses before process setup; no positive production admission/backend exists.
- Replay, analysis, governance and exchange production graphs contain no provider/network/process execution path.
- Packages remain bounded, strictly verified, non-extracted and non-executable; trust follows signature/provenance verification.
- Login-Keychain custody retains one-use Go challenges, Rust-only seed access and no renderer arbitrary-sign API.
- Unsigned protected custody fails closed; no fallback is relabeled user-presence protected.
- Release readiness/verifier false-positive paths found by this audit were closed.
- No user data, identity, trust record, key, Keychain item or artifact was deleted/rotated.

## 10. External blockers

- Authorized Developer ID Application certificate/private key and explicit permission.
- Matching real TeamIdentifier, profile and access group.
- Positive signed user-presence success/cancel/deny behavior.
- Same-key protected migration and Candidate A/B continuity.
- Notarization authorization/upload, stapling and clean-machine Gatekeeper assessment.
- Windows/Linux runtime implementations where the project explicitly reports unsupported.
- Physical power-loss/disk-full and hostile same-user fault environments.

## 11. Remaining manual acceptance checklist

Native recorder/history/snapshot workflows, native file/confirmation dialogs, disposable identity/trust/Keychain UI and signed prompt/upgrade flows remain listed in [the checklist](NATIVE_GUI_ACCEPTANCE_CHECKLIST.md). Launch smoke, component tests and compilation must not be substituted for those results.

## 12. Final project readiness assessment

`PASS_WITH_LIMITATIONS`

Phases 1–11 are acceptance-ready for continued developer-preview use within their documented local/synthetic boundaries. The audit found no remaining repository-fixable P0/P1 issue after remediation, and current automated evidence is broad. The project is not a production-secure signed macOS release: GUI/manual fault acceptance and authorized Developer ID/user-presence/migration/notarization/Gatekeeper evidence remain outstanding. No Phase 12 work was started.
