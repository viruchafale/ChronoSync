# User-presence Keychain and upgrade contract

## Protected item

The proposed item is a non-synchronizing data-protection Keychain generic password in a separate Phase 10 service namespace. Its `SecAccessControl` combines `AccessibleWhenUnlockedThisDeviceOnly` with `USER_PRESENCE`. It stores the same 32-byte Ed25519 seed, so it is neither a `SecKey` nor Secure Enclave/hardware-backed custody and still materializes in zeroized native memory after authorization.

## Signing outcomes

Protected signing is closed over: `signed`, `user_presence_required`, `authorization_cancelled`, `authorization_denied`, `key_missing`, `identity_mismatch`, `unsupported_access_control`, `unsupported_platform`, `malformed_request`, and `internal_error`. Only `signed` may complete the existing Go one-use session. Cancellation, denial, missing entitlement and backend errors never retry through the Phase 9 login item.

## Intended same-key upgrade

```text
login_key_detected -> upgrade_pending -> protected_item_created
  -> protected_identity_verified -> activation_pending -> protected_active

any nonterminal -> upgrade_cancelled | upgrade_failed | recovery_required
```

Explicit native confirmation precedes `upgrade_pending`. The login seed is validated and copied to the distinct protected item. A real protected retrieval must prompt, derive the exact existing public key/key ID and sign an engine challenge. Only then may durable metadata select protected signing. The login item remains retained until a later separately confirmed cleanup milestone; Phase 10 never deletes it automatically.

If protected storage is unsupported or authorization is cancelled/denied, no activation occurs and Phase 9 custody remains accurately labeled. If same-key import becomes impossible, replacement is a new Ed25519 identity with explicit confirmation, an optional `supersedes` link and no inherited trust.

## Current status

The state machine and outcome contract are design-only while the host lacks an authorized stable signing identity and provisioning profile. The real native probe is allowed to demonstrate typed refusal; it is not wired to renderer/engine production signing until a positive signed-build test succeeds.

Phase 11 rechecked the environment and found zero valid signing identities. It did not create a protected item, prompt, migration state or fallback path. Positive native acceptance still requires a strictly verified disposable candidate; login-item cleanup remains outside acceptance and requires separate destructive approval.
