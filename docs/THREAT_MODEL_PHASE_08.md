# Phase 8 benchmark exchange threat model

## Assets

- Personal Ed25519 private seeds and public identity metadata.
- Explicit trusted/revoked identity history.
- Canonical Phase 6 reports and Phase 7 benchmark datasets.
- Package integrity, provenance declarations and privacy choices.
- Private application-data paths and the closed execution/admission boundary.

## Boundaries and assumptions

Package bytes, filenames, metadata, public keys and signatures are hostile until fully verified. React is untrusted for paths, confirmations and private keys. The native host owns file selection and consequential confirmation; the Go engine owns cryptography, parsing, trust state and import policy. The installed host/engine and OS are trusted only within existing project assumptions. Local filesystem permissions do not protect against malware running as the same user.

## Attacker capabilities

An attacker may modify, truncate, reorder, duplicate or replay package bytes; forge metadata; supply arbitrary ZIP structures; use stolen valid keys; present revoked/rotated identities; race imports; interrupt writes; and induce users to trust an unexpected key. Same-user malware may read plaintext keys or coherently rewrite local trust data. No network attacker exists in the implementation because Phase 8 has no network client or server.

## Verification order

1. Enforce package byte limit before ZIP parsing.
2. Inspect every central-directory entry and reject unsupported paths, modes, methods, duplicates and limits before content use.
3. Read bounded bytes in memory; never extract.
4. Strictly decode canonical manifest/signature records and verify exact entry hashes/lengths.
5. Validate benchmark/report formats and complete semantic references.
6. Verify key ID, algorithm and Ed25519 signature.
7. Resolve local revocation/trust state.
8. Show privacy, provenance, compatibility and trust outcome.
9. Obtain native confirmation when required.
10. Publish idempotently through internal report/governance ports.

Trust lookup cannot make malformed bytes acceptable. Confirmation cannot override invalid signatures, revocation, unsupported formats or incomplete provenance.

## Residual risks and non-goals

- Plaintext `0600` seed files are not Keychain, Secure Enclave or hardware-backed keys.
- Local revocation has no global distribution, trusted timestamp or rollback resistance.
- Labels are self-asserted metadata and not verified human/organization identity.
- Trusted authorship does not establish safety, quality, representativeness or authorization.
- Clean-machine imports intentionally lack original run history; only explicitly included canonical reports satisfy benchmark references.
- Real power-loss, disk-full, same-user races, cross-platform archive behavior and native GUI ceremonies require additional evidence.
- Organization PKI, transparency logs, remote trust services, synchronization, executable plugins and execution authorization are outside Phase 8.
