# Phase 9B implementation status — OS-backed key custody

Implemented a narrow Rust `KeyCustody` interface for create, import, public-key retrieval, sign, status and deletion. The macOS backend uses the login Keychain service `local.chronosync.signing.v1`, stable Phase 8 key IDs, pinned `security-framework` 3.7.0, `ed25519-dalek` 3.0.0, `sha2` 0.11.1 and `zeroize` 1.9.1. Windows/Linux return explicit unsupported outcomes.

Go creates bounded expiring one-use sessions for exact Phase 8 signing payloads and verifies the returned signature before packaging. The renderer receives no seed, arbitrary signing payload or path. Real Keychain create/retrieve/sign/delete passes on the tested macOS host; per-sign user-presence access control is blocked on missing application entitlement.
