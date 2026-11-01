# ADR-0012 — Evidence retention and benchmark governance

- Status: Accepted
- Date: 2026-10-31

## Context

Phase 6 stores immutable analysis reports, but policy changes, benchmark evolution and evidence deletion require explicit provenance and lifecycle semantics. A mutable catalog or silent replacement would make historical outcomes ambiguous. Ordinary SHA-256 can detect changed canonical bytes and reveal equality, but it cannot prove authorship or authenticity.

## Decision

ChronoSync stores three bounded, versioned governance artifact families under private application data:

- canonical benchmark envelopes at `governance/v1/datasets/sha256/<content-id>.benchmark`;
- immutable evaluation lineage at `governance/v1/lineage/<lineage-id>.json`;
- append-only retention events at `governance/v1/retention/<event-id>.json`.

Benchmark v1 resolves stable case IDs to an existing report, comparison index, observation IDs, comparison digest and expected status. Stable benchmark family identity plus a positive version is distinct from the content digest. A family/version pair cannot name different bytes. Version comparison is explicit: identical content is exact; a later same-family version that retains every existing case definition and required dimension may be a compatible extension; removed/changed cases or required dimensions are incompatible; unknown formats are unsupported.

Re-evaluation invokes only the existing local Phase 6 report re-evaluator over preserved observations. It creates new reports and an immutable lineage binding the dataset content/version, exact policy, source report IDs, result report IDs and optional same-family parent lineage. It never rewrites a source report or reruns an execution.

Retention is an append-only state machine: unmanaged, active, archived, expired, damaged, deletion-pending and deleted. Expiration is metadata, not automatic deletion. Deletion requires native confirmation, a delete-enabled explicit policy and a complete current reference graph. Dependents are never cascaded. An intent precedes physical deletion and a tombstone follows it, so interrupted cross-file deletion remains visible and recoverable by audit.

No SQLite migration is introduced. The bounded immutable directories are their own source of truth, and audit derives accounting, lifecycle, references, damage, missing targets, orphaned metadata and interrupted temporary writes. Files use exclusive temporary publication, file sync, no-replace link and directory sync.

## Consequences

- Historical analysis and evaluation outcomes remain independently inspectable.
- Dataset evolution cannot silently redefine old cases.
- Reference-safe deletion requires leaf-to-root user action and may refuse when the reference graph is damaged.
- The deletion intent/file/tombstone sequence cannot be one filesystem transaction; `deletion_pending` records that uncertainty.
- Archive does not move bytes to cold storage, and expiration does not schedule work.
- Same-user software can still read, alter or replace local data; integrity is not authentication, encryption or trusted provenance.
- No provider, network, command, agent, supervisor, admission or execution-authorization dependency enters governance.

## Rejected alternatives

- Updating reports in place: loses the original policy and outcome.
- Treating family/version as content identity: permits equivocation or accidental replacement.
- Automatic expiry or cascading garbage collection: creates destructive behavior without a complete auditable policy.
- A new SQLite catalog: duplicates immutable-file state and adds a migration without a demonstrated need.
- Signing with a locally generated untrusted key: would add complexity without establishing trusted authorship.

## Verification

Unit, race and IPC integration tests cover canonical create/import, duplicate convergence, version conflict, unsupported/corrupt/missing references, compatibility outcomes, immutable re-evaluation lineage, restart, accounting, damage/temporary files, reference refusal, explicit leaf-to-root deletion and concurrent imports. Native GUI automation, real power-loss semantics and cryptographic provenance remain unverified.
