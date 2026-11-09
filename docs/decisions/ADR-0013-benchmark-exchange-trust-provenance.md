# ADR-0013 — Benchmark exchange, trust and provenance

- Status: Accepted for Phase 8 personal trust
- Date: 2026-11-01

## Context

Phase 7 provides strict local benchmark integrity and reference governance, but an ordinary SHA-256 digest cannot identify a signer. Exchanging a dataset and its supporting reports introduces archive parsing, privacy, key custody, trust, revocation and partial-import risks. Cryptographic validity, local trust, content safety, benchmark representativeness and execution authorization are separate questions.

## Threat model

Assets are private signing seeds, trusted/revoked identity history, canonical benchmark/report evidence, package contents, local governed storage, privacy-sensitive identifiers and the closed Phase 4 execution boundary. Trust boundaries are renderer → native host, native host → engine stdio, untrusted package bytes → exchange parser, exchange service → identity/trust storage, and verified package → analysis/governance import ports.

Attackers include a malicious package author; a remote sender controlling all package bytes; a renderer compromise able to request allowlisted commands; concurrent local callers; and same-user malware able to read or rewrite ordinary application files. The trusted computing base is the installed host/engine and OS within documented limits. Administrator/kernel/engine compromise, secure time, globally synchronized revocation and hardware key custody are non-goals.

Threats and required mitigations:

| Threat                                                      | Mitigation                                                                                                                                    | Residual risk                                                                   |
| ----------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------- |
| Modified content or forged/malformed signature              | Ed25519 over domain-separated canonical manifest/content hashes; strict key/algorithm/signature lengths; typed invalid outcome                | Stolen key can produce valid signatures                                         |
| Valid but unknown signer                                    | Keep `valid_untrusted` distinct; native confirmation before import; no automatic trust                                                        | User may choose to trust the wrong identity                                     |
| Revoked or rotated signer                                   | Append-only local revocation; revoked verification outcome; rotation creates a new non-inheriting identity                                    | No global propagation or trusted revocation time                                |
| Private-key theft                                           | Seed never leaves engine/private `0600` storage or appears in logs/packages/IPC                                                               | Same-user malware/backups can steal plaintext seeds; no OS-backed key store     |
| Compromised trust store                                     | Integrity-tagged append-only events, strict transitions and private paths                                                                     | Same-user attacker can coherently rewrite files and recompute unkeyed integrity |
| Malicious metadata/duplicate JSON keys/unknown authority    | Canonical JSON, duplicate-key/unknown-field refusal, bounded strings and closed enums                                                         | Parser implementation defects                                                   |
| Traversal, absolute paths, links or special archive entries | Exact normalized allowlist, store-only ZIP, Unix-mode checks, no extraction, in-memory bounded reads                                          | Go ZIP parser defects; host file-selection TOCTOU                               |
| Oversize, excessive entries or decompression bomb           | 8 MiB package, 100 entries, per-role limits, aggregate limits; reject every compression method                                                | Memory remains proportional to allowed maximum                                  |
| Missing/mismatched provenance                               | Exact manifest hashes and report/dataset semantic validation before mutation; typed `incomplete_provenance`                                   | Remote source histories are intentionally unavailable                           |
| Old/conflicting benchmark replay                            | Preserve family/version/content conflict rules; compare against local versions and report warnings                                            | A user may intentionally import an older non-conflicting family                 |
| Partial/interrupted import                                  | Preflight entire package; idempotent no-replace report publication followed by dataset publication; visible unmanaged reports on interruption | Cross-directory commit is not atomic                                            |
| Trust before complete verification                          | Verification and provenance preflight precede trust lookup, confirmation and all writes                                                       | Compromised engine/host can bypass ordering                                     |

## Decision

Use Go standard-library Ed25519. It provides a mature fixed-size signature scheme, deterministic signing, compact keys/signatures and no new cryptographic dependency. Identity v1 is the tuple `(format=1, algorithm=ed25519, public_key, key_id, label, created_at, supersedes?)`; `key_id` is SHA-256 over `chronosync-signing-key-v1\0ed25519\0` plus raw public key. Private storage contains only a versioned seed record and is never exportable through application APIs.

Trust is local append-only policy with `untrusted`, `trusted` and permanent `revoked` states. Possessing a private key, producing a valid signature, being locally trusted and being safe are deliberately independent. New rotation keys do not inherit trust. Organization identities may later use the same verifier/trust-policy ports but no organization store, certificate hierarchy, server or synchronization is implemented.

Use deterministic ZIP only as a container. Every entry must use method `Store`, fixed safe metadata and an exact allowlisted normalized path. The package is never extracted. Canonical manifest v1 covers benchmark and optional report entries by path, SHA-256, length and role, plus privacy/provenance declarations. Canonical signature envelope v1 embeds the public identity and signs `chronosync-exchange-signature-v1\0` plus the canonical manifest bytes. The package content ID is the SHA-256 of a separate domain plus manifest bytes, so it is independent of signer identity.

Export includes the benchmark and only explicitly selected required reports. Raw commands/output, workspace paths, credentials, environment variables and private seeds are structurally absent from these canonical formats. Omitted report references are declared. A package with omissions is inspectable and signature-verifiable but returns `incomplete_provenance` and cannot import into Phase 7 governed storage.

Import preflights archive structure, canonical records, all hashes, signature and semantic provenance before local trust evaluation or mutation. Invalid, revoked, malformed or unsupported packages refuse. A valid untrusted package requires trusted-native confirmation. Valid trusted packages still receive compatibility and privacy warnings; trust never implies safety or correctness.

## Consequences

- No SQLite migration or prior format change is needed.
- Standard-library cryptography and ZIP avoid a new Go dependency; Rust directly uses the already locked `base64` package only for bounded binary stdio transfer.
- Multiple installations exchange public identities through signed packages; local trust/revocation decisions do not synchronize.
- Ordinary local file permissions are disclosed as software key storage, not secure hardware custody.
- Full preflight prevents malformed writes, but a crash between report and dataset publication can leave valid unmanaged reports for audit/reuse.
- Signatures provide covered-content/key possession evidence only—not civil identity, safe code, trustworthy benchmarks, causality, deterministic AI behavior or launch authorization.

## Rejected alternatives

- HMAC/shared secrets: cannot support independent public verification and complicate sharing.
- RSA/ECDSA: larger/more complex or nondeterministic contracts without a demonstrated benefit here.
- X.509/organization PKI: adds certificate policy and issuer infrastructure outside personal trust scope.
- Keychain/Secure Enclave claim without native lifecycle implementation: would overstate current protection and harm portability.
- Compressed or tar packages: compression expands the bomb surface; tar/link semantics add unnecessary ambiguity.
- Trust-on-first-use: silently converts observation into trust.
- Importing detached datasets with invented local provenance: weakens Phase 7 reference integrity.
