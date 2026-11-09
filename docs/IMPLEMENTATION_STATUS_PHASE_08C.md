# Phase 8C implementation status — signing, verification and revocation

Implemented a domain-separated Ed25519 signature contract covering canonical identity and manifest bytes, strict key/algorithm/signature lengths, stable key IDs, append-only local trust history and permanent revocation. Outcomes keep `valid_trusted`, `valid_untrusted`, `invalid_signature`, `revoked_identity`, `unsupported_algorithm`, `unsupported_format`, `malformed_package` and `incomplete_provenance` distinct.

RFC 8032 test vector 1, signature tampering, unknown algorithm/version, rotation, restart and revocation persistence tests pass. Cryptographic validity never implies local trust; local trust never implies safety, quality or authorization.
