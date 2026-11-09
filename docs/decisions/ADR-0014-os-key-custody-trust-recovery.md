# ADR-0014 — macOS Keychain custody and trust recovery

- Status: Accepted for Phase 9 implementation
- Date: 2026-11-03

## Context

Phase 8 Ed25519 identities are portable and strictly verified, but their private seeds are plaintext application-data files. Phase 9 must improve storage without changing the identity/key ID, signature payload or package format, and must safely cross the non-transactional Keychain/filesystem boundary.

Apple CryptoKit implements Ed25519 as `Curve25519.Signing.PrivateKey`. Apple explicitly documents that this type has no direct Keychain `SecKey` corollary and is stored as a generic-password item. Secure Enclave signing exposes P-256 rather than the required Ed25519 contract. Therefore a compatible design cannot truthfully claim opaque `SecKey`, Secure Enclave or hardware-backed Ed25519 custody.

Primary platform references: [Storing CryptoKit keys in the Keychain](https://developer.apple.com/documentation/cryptokit/storing-cryptokit-keys-in-the-keychain), [Secure Enclave P-256 signing keys](https://developer.apple.com/documentation/cryptokit/secureenclave/p256/signing/privatekey), and [Restricting Keychain item accessibility](https://developer.apple.com/documentation/security/restricting-keychain-item-accessibility).

## Decision

Implement a narrow Rust native custody broker. On the current macOS build it stores the 32-byte Ed25519 seed as a local login-Keychain generic-password item. It generates/signs using maintained Rust cryptographic dependencies and zeroizes temporary seed buffers. Windows/Linux return `unsupported`. No generic signing command is registered for renderer use.

The preferred data-protection Keychain configuration with device-only/unlocked accessibility and user-presence access control was implemented and executed first, but the unsigned Cargo test received `errSecMissingEntitlement` (`-34018`). Removing only the data-protection selector still failed; removing the entitlement-dependent access-control attributes allowed a real login-Keychain create/retrieve/sign/delete round trip. Phase 9 records the stronger configuration as blocked pending a stable signed/entitled application instead of claiming or silently emulating it.

Go remains authoritative for unchanged Phase 8 identity and package contracts. OS-backed identity creation sends only public key/metadata to Go. Export uses a random expiring one-use engine session for the exact canonical Phase 8 signing payload; Rust signs, Go verifies and finalizes. Normal Go IPC never receives a Keychain seed. Legacy migration alone transfers an already-engine-owned seed through a bounded native-only response after confirmation; Rust immediately imports and zeroizes it.

Store append-only custody/migration events under private application data. Retain a legacy seed until the Keychain public key and proof signature match and a retirement-pending transition is durable. Reconcile every intermediate state after restart. Never replace a conflicting Keychain item, reconstruct a lost key, export a seed, inherit trust on rotation or automatically repair corrupt trust.

Harden trust events with strict timestamps, reasons, state enumeration, identity existence/consistency and a bounded audit result. Continue to disclose that unkeyed integrity cannot prevent a coherent same-user rewrite or rollback.

## Dependency decision

The native host may add maintained, pinned Rust dependencies for Ed25519, SHA-256, zeroization and Security-framework access. They replace neither Go verification nor the wire algorithm. Their purpose is confined to native custody/signing; no networking or provider capability is added. Exact versions and audit results are recorded after resolution.

## Consequences

- Existing Phase 8 key IDs, public identities, signing payloads, packages and historical signatures remain compatible.
- Stored seeds receive macOS login-Keychain protection, but signing material exists briefly in trusted native memory and per-sign OS user presence is not enforced in the current build.
- Stable app identity matters. Unsigned/ad-hoc builds may prompt differently or lose ACL continuity; Developer ID/notarization remains a distribution concern.
- Device-only items intentionally do not synchronize or migrate to another device. Loss can be permanent after legacy retirement.
- Migration is recoverable and idempotent but not atomically transactional across Keychain and files.
- No SQL migration, protocol-major change, Secure Enclave claim, cross-platform custody claim or execution-authority change is introduced.

## Rejected alternatives

- Change to P-256/Secure Enclave: breaks Phase 8 key IDs/signatures and silently changes the trust contract.
- Store seeds in renderer/browser storage: violates the private-key boundary.
- Keep normal signing in Go by returning Keychain seeds over IPC: needlessly expands secret exposure.
- Shell out to `/usr/bin/security`: broadens process authority and risks secret/argument/output leakage.
- Delete legacy immediately after Keychain insertion: cannot prove signing/public-key continuity and is unsafe across crashes.
- iCloud-synchronizing Keychain items or seed export: creates an undeclared recovery/synchronization channel.
- Automatic trust reset after corruption: permits rollback and revocation bypass.

## Evidence required

Compile and test the actual macOS broker, fixed Ed25519 compatibility vectors, one-use session/refusal behavior, migration interruption/restart/idempotency, trust corruption/revocation and complete Phase 8 regression. Report real Keychain prompt/integration evidence separately from fake-broker automation and preserve the existing DMG unchanged.
