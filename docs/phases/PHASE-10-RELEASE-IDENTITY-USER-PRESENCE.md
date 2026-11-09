# Phase 10 — Release identity and user-presence enforcement

## 1. Objective and user-visible outcome

Establish a reviewable macOS release-signing and entitlement architecture, test the real data-protection Keychain/user-presence API boundary, and prepare a fail-closed upgrade contract without changing Ed25519 identities, historical signatures, package v1, trust, or execution authority. Because this host has no valid Apple signing identity or provisioning profile, Phase 10 must label positive user-presence and signed-upgrade behavior blocked rather than simulate it.

## 2. Problem

Phase 9 stores extractable Ed25519 seeds in the login Keychain. The stronger data-protection Keychain requires an app-like signed user-context process whose access groups derive from authorized entitlements. Current binaries are linker ad-hoc signed, have no TeamIdentifier or entitlements, and the host reports zero valid code-signing identities. Moving an existing seed before a protected item can be retrieved and used would risk permanent identity loss.

## 3. Scope and non-goals

Phase 10A–10F cover Apple/Tauri release-identity research, credential-neutral entitlement configuration, structural release verification tooling, real native access-control feasibility evidence, a typed signing/upgrade contract, compatibility regression and honest blocker reporting. There is no certificate creation, invented TeamIdentifier, notarization, DMG rebuild, provider/agent/command execution, Phase 4 change, organization PKI, remote service, cross-platform claim or Phase 11 work.

## 4. Prerequisites and dependencies

Preserve Phase 8 identity/package/signature v1 and Phase 9 one-use Go signing sessions. Reuse locked `security-framework` 3.7.0 and existing Rust cryptography; add no native dependency unless required by evidence. A positive distribution acceptance requires an actual Developer ID Application identity, stable TeamIdentifier, authorized provisioning profile/access group, signed nested engine and app, and user-authorized native testing.

## 5. Design and component boundaries

Rust remains the only Keychain boundary. Go remains authoritative for exact signing payloads and signature verification. React receives no key or arbitrary payload. A separate protected-Keychain service/account namespace prevents an incomplete upgrade from replacing Phase 9 custody. Release tooling renders entitlements only from an explicit TeamIdentifier and verifies the finished app/engine signatures, identifiers and entitlements; it never stores credentials.

## 6. Ordered implementation tasks

1. Freeze platform evidence, threat model, ADR-0015 and this plan.
2. Add a Tauri release overlay plus entitlement template with no fabricated identity.
3. Add deterministic render/structural/signed-bundle validation tooling and tests.
4. Add an isolated real Security.framework data-protection/user-presence probe with typed outcomes and no production fallback.
5. Specify the restart-safe same-key upgrade state machine; enable no destructive migration without positive signed-build proof.
6. Run the 36-scenario structural/mock/native/regression matrix, full gates and unchanged-DMG verification.
7. Complete six separate status files, cumulative docs and consolidated report; stop before Phase 11.

## 7. Expected files/modules

- `apps/desktop/src-tauri/tauri.release.conf.json` and entitlement template.
- `scripts/macos-release-identity.mjs` plus focused tests.
- A narrowly scoped protected-Keychain feasibility module/test in the Rust host.
- ADR-0015, Phase 10 threat/release/user-presence documents, six status files and completion report.
- Cumulative README, architecture, security, testing, status index and changelog updates.

## 8. Interfaces and contracts

Protected signing outcomes are `signed`, `user_presence_required`, `authorization_cancelled`, `authorization_denied`, `key_missing`, `identity_mismatch`, `unsupported_access_control`, `unsupported_platform`, `malformed_request`, and `internal_error`. The intended upgrade states are `login_key_detected`, `upgrade_pending`, `protected_item_created`, `protected_identity_verified`, `activation_pending`, `protected_active`, `upgrade_cancelled`, `upgrade_failed`, and `recovery_required`. No wire method or durable transition is enabled until positive signed-build acceptance exists.

## 9. Security and failure considerations

Entitlement text is not evidence that the OS honored it. Ad-hoc signing is not Developer ID identity. User presence authenticates device-owner interaction, not civil identity or comprehension. The Ed25519 seed remains extractable in native memory. Upgrade must retain the login-Keychain item through protected retrieval, same-key derivation and proof signing; failure/cancellation never deletes either item or silently signs through the old item under a protected label.

## 10. Unit, integration and end-to-end tests

Cover the requested 36 scenarios with explicit evidence classes: pure mapping/config tests, fake state-machine tests where implemented, real native negative/positive API tests, structural bundle checks, signed-build checks and manual GUI checks. Mocks and an entitlement plist never count as user-presence proof. Existing exchange/race/restart/tampering/revocation tests remain authoritative regressions.

## 11. Acceptance criteria

Release configuration refuses missing/placeholder TeamIdentifier and validates a completed signed app/engine consistently. The actual unsigned environment produces a typed unsupported-access-control result and never falls back. Phase 8/9 identities, signatures and trust remain unchanged. Positive user-presence, upgrade activation, Developer ID continuity, notarization and Gatekeeper acceptance are marked blocked unless actual evidence becomes available.

## 12. Validation commands

```sh
GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/Users/viru/go/pkg/mod make check
CARGO_HOME=/private/tmp/chronosync-phase5-cargo RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup CARGO_NET_OFFLINE=true make native-check
npm run test:release-config
```

Run focused Rust protected-Keychain tests separately with real Keychain access. Run signed-bundle verification only against a newly built, authorized release candidate; do not rebuild the submitted DMG.

## 13. Risks and unresolved questions

No Developer ID certificate, TeamIdentifier, provisioning profile or notarization credential is available. Actual access-group authorization, prompt success/cancellation/denial, signed upgrade continuity, nested-code signing and Gatekeeper behavior therefore cannot be accepted in this environment. Tauri nested-resource signing behavior must be verified on the actual candidate, not inferred from configuration.

## 14. Exit criteria

Phase 10A–10F have separate status files and evidence classifications; safe tooling/probes pass; unsupported positive paths remain disabled; full regressions pass; DMG bytes are unchanged; no credential, private key, TeamIdentifier guess or later-phase implementation enters source.

## 15. Documentation updates

Update ADR-0015, threat/release/user-presence/upgrade specifications, six milestone files, completion report, release guide, README, architecture/system/security/testing/contracts/status index/changelog and AGENTS. Record every failed, skipped, manual and unverified check.

## 16. Status

`BLOCKED` — safe structural/native-negative portion complete; signed positive acceptance requires external Apple credentials.

Authorized and executed on 2026-11-04. Phase 10A, 10B and all locally available Phase 10F evidence are complete. Phase 10C positive signing, Phase 10D production migration and Phase 10E distribution continuity remain disabled because this host has no Developer ID identity, TeamIdentifier or authorized provisioning profile. No Phase 11 work is authorized.
