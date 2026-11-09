# Key custody, migration and recovery contract

## Custody interface

The trusted native host exposes no generic renderer signing API. Internally its bounded custody interface supports create, public-key retrieval, sign, existence/access status and confirmed deletion. Every result maps to `available`, `denied`, `missing`, `corrupt`, `unsupported` or `recovery_required`; raw secrets are never result values.

On macOS, service `local.chronosync.signing.v1` and the stable Phase 8 key ID identify a login-Keychain generic-password item. The 32-byte value is a CryptoKit-compatible Ed25519 seed. Because Ed25519 has no Keychain `SecKey` representation, signing reads it into zeroized native memory. This is operational OS-backed storage but neither hardware-backed nor non-extractable custody.

The attempted data-protection Keychain variant used device-only/unlocked accessibility plus user-presence access control. The actual unsigned test binary received `errSecMissingEntitlement` (`-34018`) during item creation. ChronoSync therefore does not claim that stronger mode or per-sign OS authentication. Enabling it requires a separately verified stable application signing identity, entitlement/provisioning configuration and release acceptance; the implementation does not silently fall back while labeling the item device-only.

Windows and Linux explicitly return unsupported. A legacy file remains usable only until an explicit successful migration retires it.

## Native signing sessions

Go constructs the exact Phase 8 identity/manifest payload and creates a random, bounded, expiring, one-use session associated with the key ID. Rust requests that session only inside the confirmed export workflow, signs it using the matching Keychain item, and returns a strict 64-byte signature. Go consumes the session once, verifies the signature against the unchanged public identity, builds the unchanged package v1 and reverifies the final package. Arbitrary payloads, private keys and filesystem paths are never renderer parameters.

## Migration state machine

```text
legacy_detected
  -> migration_pending
  -> keychain_created
  -> identity_verified
  -> legacy_retirement_pending
  -> migrated

any nonterminal state -> migration_failed | recovery_required
recovery_required -> keychain_created (only after a new explicit confirmed proof)
```

Each transition is an append-only canonical identity-scoped event with revision, previous state, timestamp, reason and integrity digest. `legacy_detected` may be derived until authorization; `migration_pending` is durable before the one-time legacy handoff. The native broker imports the validated seed under the same key ID, returns the public key and a signature over an engine challenge, and refuses a conflicting existing item. Go checks public/key ID equality and the proof signature before `identity_verified`.

The legacy file is removed only after `legacy_retirement_pending` is durable. Removal and directory sync precede `migrated`. If the file or state is uncertain, the identity becomes `recovery_required`; the system never claims migration merely because a Keychain item exists.

## Restart and idempotency

- Before Keychain creation: the legacy file remains authoritative and retry is safe.
- After Keychain creation: the broker must prove the existing item matches; retry must not replace it.
- From recovery required: a new explicit confirmation may resume only while the validated legacy seed still exists; a fresh Keychain proof is mandatory.
- After identity verification: retry advances toward retirement without creating another item.
- Before/after legacy retirement: inventory reconciles file existence, Keychain proof/status and durable state. Ambiguity is surfaced, not guessed.
- Concurrent attempts for one identity return busy/conflict. A completed migration returns the existing completed record.

## Loss, rotation and recovery

A missing/deleted Keychain item cannot be reconstructed from the public record. If a verified legacy seed remains, an explicit confirmed resume may recover the same identity. Otherwise the user may explicitly revoke the lost key and create a new rotation identity; trust does not transfer. There is no seed export, hidden backup or automatic deletion. Keychain deletion is a distinct confirmed destructive action and is refused while it would silently remove the only usable key.

Trust corruption produces an audit failure. Recovery cannot rewrite history, un-revoke an identity or infer trust from a package. Safe actions are inspection, explicit rejection/revocation where the validated prefix permits it, or external restoration of a known-good application-data backup; unverified automatic repair is absent.

## Phase 11 protected-upgrade acceptance status

Phase 11 did not change or execute this production migration. The authorized-environment preflight found zero valid signing identities, so no disposable protected item could be retrieved and proven before activation. Existing login-Keychain migration/restart/idempotency tests pass, but protected same-key migration, cancellation/denial and A/B continuity remain blocked. The login item was not deleted and no identity, key ID, signature, trust event or migration record changed.
