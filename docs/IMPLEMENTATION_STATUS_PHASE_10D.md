# Phase 10D implementation status — login-Keychain upgrade

## Status

`DESIGN-ONLY — destructive migration intentionally disabled`

## Defined contract

The restart-safe same-key design uses distinct login and protected service namespaces and the states `login_key_detected`, `upgrade_pending`, `protected_item_created`, `protected_identity_verified`, `activation_pending`, `protected_active`, `upgrade_cancelled`, `upgrade_failed`, and `recovery_required`.

Activation requires explicit native confirmation, protected retrieval through real user presence, derivation of the existing public key/key ID, and a verified engine challenge signature. Cancellation, denial, restart, missing items or incompatible access control cannot activate protected custody. The Phase 9 item is retained; Phase 10 performs no automatic cleanup.

If same-key Ed25519 import is unavailable, replacement must be explicit, create a new identity, preserve historical verification, optionally link `supersedes`, and inherit no trust.

## Evidence and limits

No migration state, SQL change or production command was added because positive protected retrieval is unavailable. Existing Phase 8/9 identity, trust and historical signature regressions pass. Restart/idempotency/cancellation migration scenarios remain design-only and must be exercised against an authorized disposable signed build before implementation can activate.
