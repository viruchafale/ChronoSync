# Phase 8 cryptographic and trust contract

## Algorithm and identity

Phase 8 supports exactly Ed25519 using Go's standard `crypto/ed25519` implementation. Keys are generated from `crypto/rand`. Public keys are 32 bytes and signatures are 64 bytes, encoded with strict padded standard base64. Unknown algorithms and lengths are refused.

The stable key identifier is:

```text
SHA-256("chronosync-signing-key-v1\0ed25519\0" || raw_public_key)
```

Identity labels, creation timestamps and `supersedes` links are self-asserted signed metadata, not verified names or organizations. Rotation creates a new key ID. Trust never transfers automatically to that new key.

## Private-key storage

The engine stores a versioned base64 Ed25519 seed under `exchange/v1/identities/private/<key-id>.json` in private application data. Directories require POSIX `0700` and files `0600`; private seed bytes never enter packages, renderer/native API results, logs or reports. This is plaintext software storage. It is not macOS Keychain, Secure Enclave, hardware-backed protection, password encryption or resistance to malicious software running as the same OS user.

## Trust and revocation

Public identities live separately under `exchange/v1/identities/public`. Trust decisions are integrity-tagged append-only events under `exchange/v1/trust`, with ordered revisions and states `untrusted`, `trusted` and permanently `revoked`. A trusted identity may be explicitly rejected back to untrusted; either untrusted or trusted may become revoked; revoked cannot transition.

Locally created identities receive an explicit trusted event as part of the native-confirmed creation operation. Externally observed identities remain untrusted until an explicit native-confirmed decision. Importing a valid untrusted package requires its own native confirmation and does not automatically trust the signer.

Local trust/revocation is not synchronized, globally timestamped or rollback-resistant. Another installation can reach a different decision. Same-user software can rewrite trust files and recompute unkeyed event integrity.

## Security meanings

- Key possession: the signer could produce a valid signature for the private key corresponding to the embedded public key.
- Signature validity: covered identity/manifest bytes match the Ed25519 signature.
- Local trust: this installation has an explicit current trusted event for that key.
- Revocation: this installation permanently refuses that key after an explicit revocation event.

None of these meanings establishes that a package is safe, that the signer is a particular legal person or organization, that benchmark evidence is representative, that an AI execution is deterministic, or that production execution is authorized.
