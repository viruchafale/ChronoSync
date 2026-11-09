# Phase 9D implementation status — identity recovery and rotation

Identity records expose custody backend/status and migration state. Explicit Keychain status distinguishes available, denied, missing, corrupt, unsupported and recovery-required. Lost keys are never reconstructed from public records; there is no seed export or hidden backup. Rotation creates a new Phase 8 identity with an optional covered `supersedes` link, while the old key and historical signatures remain independently verifiable.

Revocation remains the safe response to suspected compromise. New local identity creation has its own confirmed trust event; imported/rotated identities do not inherit another key's trust. Login-Keychain/account/device/application-identity recovery limitations are documented.
