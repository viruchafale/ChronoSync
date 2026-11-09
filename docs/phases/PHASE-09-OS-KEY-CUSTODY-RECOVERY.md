# Phase 09 — OS-backed key custody and trust recovery

## 1. Objective and user-visible outcome

Protect personal Ed25519 signing seeds in the macOS login Keychain, migrate eligible Phase 8 file seeds without changing public identities or old signatures, expose exact loss/denial/corruption states, and harden local trust audit/recovery without adding execution authority. Per-sign OS user-presence enforcement is an explicit blocked capability for the current unsigned/ad-hoc build, not an implementation claim.

## 2. Problem

Phase 8 stores a canonical seed in a private `0600` file. That prevents accidental renderer/package disclosure but not same-user theft or ordinary backup exposure. Moving a key spans two non-transactional stores, while Ed25519 has no native `SecKey` or Secure Enclave representation. A failed migration must never silently destroy the only usable key or make custody claims that the platform does not provide.

## 3. Scope and non-goals

Phase 9A–9F cover macOS Keychain feasibility, a narrow native custody interface, backward-compatible personal identity creation/signing, explicit restart-safe legacy migration, key-loss/rotation/revocation behavior, trust audit/recovery refusal, typed desktop integration and adversarial validation. Windows/Linux return explicit unsupported outcomes. There is no Secure Enclave claim, private-key export, hidden backup, organization PKI, remote recovery/sync, provider/agent/command execution, workspace restore, admission change or Phase 10 work.

## 4. Prerequisites and dependencies

The implementation preserves Phase 8 identity/package/signature v1 and consumes its Go exchange service through additive protocol-1 methods. The Tauri host owns Keychain calls and native confirmation. A maintained Rust Ed25519 implementation, SHA-256 implementation, zeroization helper and macOS Security-framework binding are justified because CryptoKit Ed25519 keys have no direct Keychain `SecKey` corollary and the Go engine must not receive Keychain seed material during normal signing.

## 5. Design and component boundaries

Go owns canonical identities, manifests, signing-session state, migration/audit state machines, trust policy and signature verification. Rust owns a narrow `KeyCustody` interface: create, public key, sign, existence/status and confirmed deletion. On the tested macOS host the broker stores a 32-byte seed as a login-Keychain generic-password item. The seed exists briefly in zeroized native memory for generation/import/signing because Ed25519 is not an opaque `SecKey`; it never enters renderer/TypeScript, packages, reports or normal engine signing IPC. Data-protection Keychain plus per-item user presence is retained as a signed/entitled distribution design only: the actual unsigned test returned `errSecMissingEntitlement`.

## 6. Ordered implementation tasks

1. Freeze platform evidence, threat model, ADR-0014 and migration/recovery contract.
2. Add typed Go custody/migration/trust-audit contracts and a native-broker signing-session boundary.
3. Implement/test the Rust custody trait, in-memory adversarial broker and macOS Keychain broker.
4. Add native-confirmed OS-backed creation, signing, inventory and typed error handling.
5. Add explicit legacy discovery/migration with durable state transitions, proof signing and post-proof retirement.
6. Add loss/recovery/rotation controls, strict trust-event validation/audit and 40 requested scenarios.
7. Update React/contracts/docs and run complete portable/native/platform checks without bundling.

## 7. Expected files/modules

- `engine/internal/exchange/{custody,migration,audit}.go` and focused tests.
- Additive exchange IPC and engine capability/version wiring.
- `apps/desktop/src-tauri/src/key_custody.rs`, native orchestration and tests.
- Additive TypeScript custody/migration contracts and React controls/tests.
- ADR-0014, Phase 9 threat/custody/migration specifications, six milestone status files and consolidated completion report.

## 8. Interfaces and contracts

Custody backends are `legacy_file`, `macos_keychain` and `unsupported`. Access states are explicit: `available`, `denied`, `missing`, `corrupt`, `unsupported`, `recovery_required`. Native signing accepts only an engine-issued bounded one-use session containing the exact Phase 8 signing payload; completion supplies a 64-byte signature which Go verifies before package publication.

Migration states are `legacy_detected`, `migration_pending`, `keychain_created`, `identity_verified`, `legacy_retirement_pending`, `migrated`, `migration_failed`, and `recovery_required`. Durable transitions are append-only and identity-scoped. The filesystem and Keychain are never described as one transaction.

## 9. Security and failure considerations

Login-Keychain generic-password storage is OS-backed storage, not an opaque cryptographic key or hardware backing. The native confirmation is reviewed local intent, not OS authentication. The entitlement-backed data-protection/user-presence variant did not run under the unsigned test binary and is not silently downgraded or claimed. Same-user malware controlling the application process can request signing. Migration retains the legacy seed until Keychain public-key equality and proof signature succeed; retirement is no-replace metadata plus seed-file removal and directory sync, with recovery-required states for uncertain outcomes.

## 10. Unit, integration and end-to-end tests

Cover the 40 requested custody, migration, recovery, trust and Phase 8 compatibility scenarios using disposable fake brokers/stores plus real macOS compile/integration evidence where authorization permits. Add source audits proving no renderer seed/path exposure and no provider/process/network/admission dependency. Manual Keychain prompts and app-signature continuity remain separately reported.

## 11. Acceptance criteria

Login-Keychain creation/retrieval/signing/deletion are operational on the tested macOS host; the data-protection/user-presence path is reported blocked with exact entitlement evidence. Legacy migration is consented, idempotent, restart-safe and preserves key ID/signature compatibility; no usable sole key is silently deleted; revocation is terminal; trust corruption is detected and cannot be automatically reconstructed; Phase 8 exchange remains byte-compatible; full gates pass; the submitted DMG remains unchanged.

## 12. Validation commands

```sh
GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off GOSUMDB=off make check
CARGO_HOME=/private/tmp/chronosync-phase5-cargo RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup CARGO_NET_OFFLINE=true make native-check
```

Focused Go/Rust race, migration/restart, corruption, cryptographic vector/compatibility, source/dependency and Keychain integration checks are also required. No bundle command may run.

## 13. Risks and unresolved questions

Generic-password retrieval materializes the Ed25519 seed in native process memory. Per-sign user-presence access control requires a stable entitled application path that is unavailable to the current test binary. Keychain deletion, account/device migration, application identity or access changes can permanently lose signing ability. Real crash/power-loss at every cross-store boundary and cross-platform behavior may remain unverified.

## 14. Exit criteria

Milestones 9A–9F, separate status files, actual supported-platform evidence, typed engine/native/renderer integration, requested adversarial coverage and complete gates are recorded honestly. Existing signatures verify, no private key crosses renderer boundaries, no previous artifact is rewritten, and Phase 10 does not begin.

## 15. Documentation updates

Update ADR-0014, threat model, key-custody and migration/recovery specifications, six status files, consolidated report, architecture/system/data/security/testing/contracts/README/index/changelog and AGENTS. Record all failed, skipped, manual and unverified checks.

## 16. Status

`COMPLETE`

Authorized, planned and implemented on 2026-11-03. Platform and migration
architecture were documented before custody or migration production changes.
`make check` and `make native-check` pass; the separately invoked disposable
login-Keychain integration test passes with real Keychain access. Per-sign user
presence, hardware custody, GUI-dialog automation, cross-platform custody and
real power-loss/account-migration behavior remain unsupported or unverified.
