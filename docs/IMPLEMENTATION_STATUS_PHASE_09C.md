# Phase 9C implementation status — safe plaintext-key migration

Implemented explicit native-confirmed discovery and migration for validated Phase 8 legacy seeds. Append-only states record pending, Keychain-created, identity-verified, retirement-pending, migrated, failed and recovery-required outcomes. The same public key/key ID and an engine challenge signature must verify before the legacy file is retired and its directory synced.

Focused tests cover mismatch, invalid seed, interruption/restart before and after Keychain creation/verification, concurrent refusal, failure and recovery-required recording/explicit resume, idempotent retirement and historical-package signature compatibility. The Keychain and filesystem are explicitly not one transaction; ambiguous outcomes remain visible.
