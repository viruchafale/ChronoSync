# Phase 07 — Evidence retention and benchmark governance

## 1. Objective and user-visible outcome

Govern stored analysis evidence and versioned benchmark datasets locally: inspect retention state and storage, validate/import/export benchmark definitions, compare benchmark versions, create explicit policy/dataset evaluation lineage, audit integrity/references, and perform only confirmed reference-safe deletion.

## 2. Problem

Phase 5 fixtures and Phase 6 reports are immutable private evidence, but there is no lifecycle ledger, benchmark dataset contract, compatibility model, lineage across revised policy/dataset evaluations, storage audit, or reference-aware deletion workflow. Treating changed datasets as equivalent or deleting referenced evidence would make historical evaluation untrustworthy.

## 3. Scope and non-goals

Phase 7A–7F implement evidence retention metadata, canonical benchmark dataset v1, explicit version comparison, immutable evaluation lineage, integrity/reference audits, accounting and confirmed deletion. There is no provider/LLM call, agent rerun, historical command, network/external metric service, executable evaluator, background deletion/expiry, approval integration, deterministic-execution claim, trusted-authorship claim or Phase 8 work.

## 4. Prerequisites and dependencies

The implementation consumes existing Phase 6 analysis report APIs and types, private application-data storage, protocol-1 stdio, the official native dialog plugin already used for fixtures, and React/TypeScript. Phase 5 fixture bytes, Phase 6 report bytes, event/snapshot schemas, SQLite migration 3 and Phase 4 behavior remain unchanged. No new dependency is planned.

## 5. Design and component boundaries

Add a standard-library-only `internal/governance` package. Canonical benchmark files live under `governance/v1/datasets/sha256`; immutable lineage records under `governance/v1/lineage`; append-only retention events under `governance/v1/retention`. A service owns validation, bounded no-replace publication, reference resolution, audit/accounting and deletion orchestration over an `analysis.ReportRepository` port. Renderer APIs carry identities/canonical text only, never storage paths.

An ADR will freeze dataset, compatibility, lineage and deletion semantics. SQLite is not changed because all governed artifacts are immutable canonical files and the bounded catalog can be derived safely.

## 6. Ordered implementation tasks

1. Define lifecycle states, retention policies/events, references and accounting.
2. Define canonical benchmark dataset/envelope v1 with resolved Phase 6 report/comparison provenance.
3. Implement version-family uniqueness and explicit exact/compatible-extension/incompatible/unsupported comparison.
4. Implement dataset/policy evaluation lineage that creates new Phase 6 reports and immutable lineage without rewriting sources.
5. Implement strict audit, orphan/damage reporting, native confirmations and non-cascading reference-safe deletion.
6. Add typed engine/Rust/React integration, all requested disposable/restart/concurrency/interrupted-write cases, documentation and final gates.

## 7. Expected files/modules

- `engine/internal/governance/{model,dataset,compatibility,retention,lineage,service}.go` and focused tests.
- Minimal Phase 6 report repository additions needed for governed deletion/lineage, with compatibility tests.
- `engine/internal/ipc/governance.go`, engine construction/capability updates and integration tests.
- `contracts/governance.ts`, Tauri commands, `Governance.tsx` and component tests.
- ADR-0012, separate 7A–7F status files, consolidated report and architecture/data/security/testing/changelog/index updates.

## 8. Interfaces and contracts

Dataset v1 has stable family ID, positive version, name, creation metadata, resolved cases with stable case IDs and comparison/report/observation fingerprints, supported/required dimensions, compatibility declaration and provenance limitations. An outer SHA-256 covers canonical bytes but proves neither authorship nor authenticity.

Compatibility returns exact, compatible extension, incompatible or unsupported plus missing/added/changed cases and reasons. Retention events are immutable revisions with active, archived, expired, damaged, deletion-pending or deleted states. Lineage binds dataset content/version, source reports, newly created evaluation reports, policy and optional parent lineage.

## 9. Security and failure considerations

All imported bytes are untrusted, strictly canonical and bounded. IDs, versions, case uniqueness, dimensions, references, timestamps and integrity digests are verified. Hashes reveal equality. Deletion requires native confirmation, an exact current reference audit and no dependents; there is no cascade. A deletion intent is appended before physical removal and a tombstone afterward, making the unavoidable cross-file crash window visible. Interrupted temporary files are retained/ignored and reported by accounting. No artifact is repaired silently.

## 10. Unit, integration and end-to-end tests

Cover valid create/import/export, duplicate bytes and identity/version conflict, unsupported versions/dimensions, canonical corruption, missing report/observation/comparison references, added/missing/changed cases, compatible/incompatible versions, policy reevaluation, lineage/history retention, accounting, orphan/damage audit, referenced deletion refusal, explicit leaf-to-root deletion, restart, concurrent writer refusal/convergence and interrupted temporary publication.

## 11. Acceptance criteria

Every requested 7F dataset passes with stable explainable outcomes; existing Phase 5/6 bytes/tests remain compatible; policy/dataset revisions create new immutable records; audit exposes missing/damaged/unsupported references; deletion never cascades or bypasses references; full portable/native gates pass; submitted DMG remains byte-identical.

## 12. Validation commands

```sh
GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off GOSUMDB=off make check
CARGO_HOME=/private/tmp/chronosync-phase5-cargo RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup CARGO_NET_OFFLINE=true make native-check
```

Focused race/vet, dependency/source-boundary, corruption/restart/concurrency and DMG checksum checks are also required. No bundle command may run.

## 13. Risks and unresolved questions

Cross-file deletion cannot be one atomic filesystem transaction; deletion-pending plus audit/tombstone preserves uncertainty. Native GUI interaction, filesystem power-loss behavior, Windows/Linux runtime semantics, clean-machine exchange and signed packaging remain unverified. Archive is metadata only; no cold-storage transport is introduced. Retention expiry is informational and never automatic.

## 14. Exit criteria

Milestones 7A–7F, six separate status files, ADR/contracts/UI, requested failure/recovery evidence and full gates are complete with limitations stated. No historical artifact is rewritten and no Phase 8 capability begins.

## 15. Documentation updates

Update this plan, status index, architecture, system/data/security/testing documents, contract guide, changelog, README, AGENTS, ADR-0012, separate 7A–7F statuses and a consolidated 16-point Phase 7 acceptance report. Record actual failed/skipped/unverified checks.

## 16. Status

`COMPLETE`

Authorized on 2026-10-31 and completed on 2026-11-01. Phase 7A–7F implementation, separate milestone evidence, ADR-0012, portable/native gates and unchanged submitted-DMG verification are complete. Native GUI automation, real power-loss behavior and cross-platform runtime semantics remain explicitly unverified. Phase 8 has not started.
