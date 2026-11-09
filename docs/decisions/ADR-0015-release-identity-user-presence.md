# ADR-0015 — macOS release identity and user-presence boundary

- Status: Accepted for Phase 10 safe implementation
- Date: 2026-11-04

## Context

Phase 9 preserved Ed25519 compatibility by storing a 32-byte seed as a generic-password item in the file-based login Keychain. Apple documents that biometric/user-presence access control is a data-protection Keychain feature. On macOS, data-protection access groups derive from the host process's signed entitlements and those restricted entitlements require authorization by a provisioning profile. The current app and engine are linker ad-hoc signed with no TeamIdentifier or entitlements, and `security find-identity -v -p codesigning` reports zero valid identities.

## Decision

Keep Phase 9 production custody unchanged until a real signed release candidate passes positive user-presence acceptance. Add a credential-neutral Tauri release overlay, a TeamIdentifier entitlement template, and validation tooling that refuses placeholders, mismatched identifiers, missing hardened runtime, unsigned/ad-hoc code, differing host/engine teams or missing expected entitlements.

Compile and directly exercise an isolated Security.framework data-protection generic-password probe using `SecAccessControl` with `AccessibleWhenUnlockedThisDeviceOnly` plus `USER_PRESENCE`. Use a separate service namespace and disposable account. Map missing entitlement, cancellation, denial, missing item, malformed request and platform absence to closed typed outcomes. Do not register a renderer command or change normal signing behavior while positive signed-build acceptance is unavailable.

A future same-key upgrade must copy the validated Phase 9 seed into a separate protected item, retrieve it through actual user presence, derive the unchanged public key/key ID, sign an engine challenge, and durably activate protected custody before the login item becomes merely a retained rollback source. Cleanup is a later separately confirmed action; no automatic deletion is allowed. If same-key import is unavailable, replacement must create a new identity and cannot inherit trust.

## Release identity

The expected distribution identity class is `Developer ID Application`; its concrete certificate name and TeamIdentifier must come from the authorized Apple Developer account and never be invented or committed. The app identifier remains `dev.chronosync.desktop`; the bundled engine uses a distinct reviewed code identifier and the same signing team. Sign nested code before the outer app and verify both actual signatures afterward. Hardened runtime is required. Provisioning/entitlement authorization, notarization, stapling and Gatekeeper acceptance are separate evidence gates.

## Consequences

- Phase 8 packages, Ed25519 verification, public identity v1, key IDs, trust and revocation remain unchanged.
- Entitlement/release tooling can be structurally tested without credentials, but that is not proof of OS enforcement.
- Current ad-hoc builds remain development builds and must report protected custody unavailable.
- Positive signing, prompt cancellation/denial, upgrade activation and release continuity remain blocked until an actual authorized signed build is tested.
- No SQL migration, protocol change, renderer authority, new dependency, DMG rebuild or execution-authority change is introduced by the safe implementation.

## Rejected alternatives

- P-256/Secure Enclave replacement: breaks the Ed25519 identity and package contract.
- Calling LocalAuthentication before reading an ordinary login item: creates a bypassable ceremony rather than Keychain-enforced access.
- Treating an ad-hoc signature or plist as a stable release identity: does not establish TeamIdentifier continuity or entitlement authorization.
- Replacing/deleting the Phase 9 item before proof: risks permanent key loss.
- Falling back to login-Keychain signing after protected authorization failure: violates user-presence semantics.
- Committing a real TeamIdentifier, certificate, provisioning profile or credential: leaks account-specific release material and couples source to an unverified identity.

## Required future acceptance

Use an authorized disposable release candidate signed by the intended Developer ID Application identity with its provisioning profile. Verify actual host/engine code signatures and entitlements, create and retrieve a protected item, observe success/cancel/deny, sign and verify exact Phase 8 bytes, test restart/upgrade continuity, then notarize/staple and assess Gatekeeper separately. Until then, protected production activation is blocked.

## Phase 11 acceptance attempt

Phase 11 was authorized on 2026-11-04, but the host still reported zero valid code-signing identities and had no explicit TeamIdentifier/profile or notarization authorization. Credential-neutral preflight and A/B continuity validation were added and passed structurally. No candidate was signed, no protected positive path or migration was activated, and no artifact was uploaded. This confirms rather than relaxes the decision: production remains Phase 9 custody until the required external environment exists.
