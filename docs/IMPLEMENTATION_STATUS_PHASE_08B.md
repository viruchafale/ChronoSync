# Phase 8B implementation status — personal signing identities

Implemented versioned personal Ed25519 identities using Go standard-library cryptography, domain-separated SHA-256 key IDs, `crypto/rand` generation, public/private separation, local creation, listing and explicit rotation linkage. Local creation appends a trusted event; rotation creates a distinct key and does not inherit another identity's trust.

Private seeds remain engine-only plaintext `0600` files in `0700` application data and never enter packages, IPC results, logs or reports. This is explicitly not Keychain, Secure Enclave or hardware-backed protection. Organization identity and certificate policy are documented extension points only.
