# Phase 08 — Benchmark exchange, trust and provenance

## 1. Objective and user-visible outcome

Allow developers to create personal Ed25519 signing identities, export governed benchmarks and selected supporting reports as signed non-executable packages, inspect and verify packages, make explicit local trust decisions, import complete compatible evidence, and revoke identities without confusing signature validity, trust, safety or execution authorization.

## 2. Problem

Phase 7 benchmark files are local and integrity-tagged but have no portable provenance envelope, signer identity, trust store, revocation history or safe multi-file exchange. A plain digest cannot establish which key signed content, while a valid signature cannot establish that the signer is trusted or that a benchmark is safe or representative.

## 3. Scope and non-goals

Phase 8A–8F implement a personal local identity/trust store, Ed25519 signing and verification, deterministic bounded exchange packages, provenance/report inclusion, strict native import/export, explicit trust/revocation controls, and adversarial validation. Organization policies are extension points only. There is no provider/LLM call, command or agent execution, plugin, network client, synchronization, trust server, external metric, workspace restore, admission/approval change, execution authorization or Phase 9 work.

## 4. Prerequisites and dependencies

The implementation consumes canonical Phase 6 reports and Phase 7 benchmark envelopes through narrow internal ports, private application-data storage, protocol-1 stdio, the existing Tauri dialog plugin and React/TypeScript. Ed25519, SHA-256, canonical JSON and ZIP parsing use the Go standard library. Rust uses the already locked transitive `base64` crate directly for bounded binary IPC; no cryptographic dependency is added.

## 5. Design and component boundaries

Add `internal/exchange`, independent of adapters, providers, process launch, networking, snapshots, admission and approval. Private identities/trust history live under `exchange/v1`; public package entries are validated in memory and imported only after complete verification and required user consent. The trusted native host owns paths and confirmations. Renderer contracts expose public identity metadata, verification outcomes and base64 package transport but never private key bytes or filesystem paths.

## 6. Ordered implementation tasks

1. Freeze the threat model and ADR-0013 before signing code.
2. Implement personal identity generation, public identity records, private seed storage, rotation linkage and append-only trust/revocation events.
3. Implement domain-separated Ed25519 signature verification with typed validity/trust outcomes.
4. Implement deterministic store-only ZIP exchange format with canonical manifest, exact entry hashes and strict limits/path/type validation.
5. Add report export/import and benchmark exchange orchestration with complete preflight, idempotent publication and explicit incomplete-provenance behavior.
6. Add strict IPC, native dialogs/confirmations, React management/import/export UI, adversarial tests, documentation and complete gates.

## 7. Expected files/modules

- `engine/internal/exchange/{model,identity,trust,package,service}.go` and focused tests.
- Minimal internal analysis/governance canonical-byte ports for verified export/import.
- `engine/internal/ipc/exchange.go`, service construction and capability updates.
- `contracts/exchange.ts`, Tauri commands, `Exchange.tsx` and component tests.
- ADR-0013, threat model, exchange/cryptographic specifications, six milestone status files, consolidated report and architecture/data/security/testing/index/changelog updates.

## 8. Interfaces and contracts

Identity v1 uses algorithm `ed25519`, raw 32-byte public keys encoded as base64 and a SHA-256 key ID over a domain-separated algorithm/public-key payload. Private seeds remain in private storage. Trust is append-only `untrusted`, `trusted` or permanently `revoked`; rotation creates a distinct identity and never inherits trust automatically.

Exchange v1 is a deterministic store-only ZIP containing exact allowlisted paths: canonical `manifest.json`, one `benchmark.benchmark`, zero or more `reports/<id>.json`, and canonical `signature.json`. The manifest hashes every content entry, declares included/omitted provenance and compatibility, and yields a signer-independent package content ID. The signature covers a domain-separated canonical signing payload. Verification reports cryptographic validity separately from local trust.

## 9. Security and failure considerations

Validate archive metadata before entry reads; refuse compression, encryption, traversal, absolute/backslash/non-normalized paths, duplicate names, links/special modes, unknown entries, excessive counts/sizes and mismatched hashes. Parse canonical JSON with duplicate/unknown-field refusal. Verify all content and provenance before trust lookup or mutation. Untrusted valid packages require native confirmation; invalid, revoked, unsupported or malformed packages cannot import. Cross-store publication is idempotent but not one filesystem transaction; imported reports may remain as unmanaged immutable evidence if interruption occurs before the dataset publish, and audit/status must say so.

Private Ed25519 seeds are plaintext `0600` files in `0700` application data. This is not Keychain, Secure Enclave, hardware-backed protection or same-user resistance. Revocation is local policy history, not a global revocation service. A trusted signature proves only that the corresponding key signed covered bytes.

## 10. Unit, integration and end-to-end tests

Cover trusted/untrusted valid packages, content/signature tampering, unknown algorithm/version, revoked signer, missing/mismatched provenance, duplicate/traversal/symlink/special entries, compressed and oversized entries/packages, entry-count excess, malformed/noncanonical/duplicate-key/unknown fields, interrupted import, concurrent duplicates, rotation, restart persistence, revocation persistence, clean-machine incomplete provenance and re-export verification. Source audits assert no process/provider/network/snapshot/admission/approval dependency in production exchange code.

## 11. Acceptance criteria

All requested Phase 8F scenarios produce stable typed outcomes; trust never precedes complete cryptographic verification; valid-untrusted remains distinct from invalid; revocation persists; complete packages import idempotently; incomplete packages remain inspectable but cannot silently enter governed storage; prior formats/migrations remain compatible; portable/native gates pass; the submitted DMG remains unchanged.

## 12. Validation commands

```sh
GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off GOSUMDB=off make check
CARGO_HOME=/private/tmp/chronosync-phase5-cargo RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup CARGO_NET_OFFLINE=true make native-check
```

Focused race/vet, fixed cryptographic vectors, adversarial archive, round-trip, restart, concurrency, source/dependency and DMG checksum checks are also required. No bundle command may run.

## 13. Risks and unresolved questions

Plaintext seed storage is vulnerable to same-user compromise and backup leakage. Local revocation cannot notify another installation. ZIP parser/library defects and filesystem races remain possible despite strict bounds. Native GUI automation, real power-loss/disk-full behavior and cross-platform semantics may remain unverified. Organization roots, certificates, transparency, remote revocation and synchronization are intentionally deferred.

## 14. Exit criteria

Milestones 8A–8F, separate status files, ADR/threat/specification documents, typed engine/native/renderer integration, requested adversarial evidence and full gates are complete with limitations stated. No prior artifact is rewritten, no private key crosses the engine boundary, and Phase 9 does not begin.

## 15. Documentation updates

Update this plan, ADR-0013, Phase 8 threat model, exchange and cryptographic specifications, six status files, consolidated report, status/document indexes, architecture, system/data/security/testing, wire contracts, README, AGENTS and changelog. Record actual failures, skipped checks and unverified behavior.

## 16. Status

`COMPLETE`

Authorized, planned and completed on 2026-11-01. Threat modeling and ADR-0013 preceded signing implementation. Focused adversarial/race/vet checks, `make check`, offline locked `make native-check` and unchanged submitted-DMG checksum evidence are recorded in the completion report. Native GUI automation, real power-loss behavior, OS-level egress denial and cross-platform archive semantics remain unverified.
