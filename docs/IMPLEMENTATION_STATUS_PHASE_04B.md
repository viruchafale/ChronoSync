# Phase 4B implementation status — durable admission and launch-binding design

Status: COMPLETE for design/audit/test planning only. Durable admission, migrations, positive launch behavior, UI changes and isolation backends remain NOT_STARTED. [Status index](IMPLEMENTATION_STATUS.md). The [Phase 4A report](IMPLEMENTATION_STATUS_PHASE_04A.md) is retained unchanged.

## Audit against code

Reviewed internal/policy types/Resolve/ConservativeProvider, supervisor.Spec/Run/refusal path, Codex Environment/Version/Record, recorder Start/execute, IPC start handling, Rust typed StartSpec/bridge, React execution/consent controls, storage migrations/lease/recovery, Phase 4 plan and ADR-0005/0006. The workspace has no Git metadata; git status reported not a repository. Source was inspected directly and a pre/post SHA-256 inventory substitutes for Git diff.

Findings:

- Resolve is deterministic for explicit inputs, trusts its in-process provider input and hashes ordered Go policy JSON. Satisfied means assessed evidence, not authenticated authority. Scope digests are supplied identities, not actual open-file binding. No revocation, approval identity, argv/env/stdin binding or durable admission exists.
- ConservativeProvider reports unknown/advisory; any non-nil supervisor policy refuses before process setup. Nil retains legacy semantics. Neither Codex version probing nor execution supplies a policy. Both are launches; a future admission design must account for both.
- Recorder validates consent, read-only/workspace-write selection, absolute paths and 1–3600 second timeout and creates run metadata before adapter version/execution. Supervisor bounds output to 8 MiB, lines to 128 KiB and ordinary process cleanup with a 500 ms grace. Those controls do not prove isolation.
- Rust forwards a bounded typed start request over serialized private IPC; React consent resets on input changes and does not supply production capability evidence. The boolean consent is not durable identity-bound approval.
- SQLite migrations 1–2, transaction/writer gate, private store lease and interrupted recovery provide a foundation for new records but no SQL/OS atomic operation. Recovery does not launch/kill a process based on persisted metadata. Windows storage/recording remain blocked.
- Engine 0.3.0, IPC 1, event 2, run 1, manifest 1 and internal policy 1 remain unchanged. No compatibility issue requires a format change in this design task.

## Design delivered

[System design](SYSTEM_DESIGN.md) defines prepare binding → approval → durable prepared admission → one-use durable claim → final validation → at-most-one OS attempt → separate launch observation. Claimed admissions are never reused; ambiguous outcomes remain unknown. No exactly-once guarantee is claimed. Approval/audit evidence avoids circular self-hashing: approval binds a precomputed execution manifest, while storage readiness and actual committed records are distinct facts.

The launch-binding matrix covers resolved executable/artifact and native identity, exact argv, environment names/value rules, private stdin commitments, cwd/workspace generation, policy/effective restrictions, evidence/backend context, adapter configuration and optional future ancestry. Portable hashes/path checks cannot close filesystem races; stronger binding depends on a separately reviewed backend. Secret values and guessable public hashes are excluded; key/credential lifecycle decisions remain open.

[Security model](SECURITY_MODEL.md) assigns authority to the trusted engine and registered providers. Frontend IPC carries intent and challenge responses, never trusted verified flags/effective decisions/actor timestamps. Approval's actual human-authentication strength is unresolved pending a trusted confirmation design. Digests do not authenticate providers. The conservative production provider stays unchanged.

[Data model](DATA_MODEL.md) proposes versioned immutable bindings/evidence/approvals, admissions, unique launch attempts, append-only audit/revocations and relational integrity. Transaction boundaries, one-time claims/idempotency, prior-engine epoch invalidation, no automatic recovery launch, audit ordering, retention, unknown-version refusal and stopped-backup rollback limitations are explicit. This is a reviewable migration proposal, not applied SQL. No historical approvals are fabricated.

[ADR-0007](decisions/ADR-0007-durable-admission-launch-binding.md) records alternatives/trade-offs. The [Phase 4 plan](phases/PHASE-04-SANDBOX-GUARDRAILS.md) splits future work into domain contracts, storage/recovery, trusted provider/approval provenance, backend feasibility/implementation, supervisor consumption/binding, and typed IPC/UI. Each has prerequisites, tests, security review, acceptance and exclusions. [Testing strategy](TESTING_STRATEGY.md) covers future deterministic vectors, fault windows, concurrent claims/revocation, privacy, hostile IPC and backend race tests without claiming they ran.

## Failure semantics

Missing/expired/revoked approval, mismatched identities/hashes and stale/unavailable evidence refuse launch. Changed bindings require renewed approval. Concurrent duplicate claims never create a second attempt. Storage failure before known claim commit authorizes no OS call; an ambiguous commit must be reconciled, not blindly retried. Any post-claim failure consumes permanently. Definite no-child spawn failures record not_started; ambiguous errors/crashes record outcome_unknown. Outcome-persistence failure after start stops new admissions and requests owned cancellation without claiming it succeeded. Later revocation cannot retroactively prove a process never launched. Absent or reused PIDs are not safe recovery evidence.

## Validation actually performed

- Node 22.14.0 and npm 10.9.2 observed; existing local Prettier used, no installs.
- Local Prettier write/check on changed Markdown: passed.
- `npm run check:docs`: passed structured phase plans, required docs and local links.
- Additional local Markdown file-target scan including ADRs: passed. External URLs were not fetched; no browser or privileged research performed.
- Pre/post SHA-256 inventory: all 106 non-Markdown source/config/asset files unchanged, including schemas, migration sources, lockfiles and packaged binary. Build/dependency/cache directories excluded; no new runtime source files.
- No new diagrams; existing diagrams untouched. Links checked for local target existence, not external or rendered-fragment validity.

Runtime tests, make check, make native-check, builds, native GUI, provider calls, platform probes and untrusted execution were **not run in Phase 4B**, as appropriate for documentation-only changes. Earlier passing runtime evidence belongs to Phase 4A and is not presented as rerun here.

## Unresolved decisions and exact next approval

Approve only the proposed first domain-contract milestone before implementation: internal admission/binding/state types and pure canonicalization/transition tests, with no migration, positive launch, UI or backend. That approval must settle the canonical bytes/bounds, immutable identity and one-use/epoch semantics, local approval principal/challenge model, private-input commitment approach and revocation ordering described in ADR-0007. No next implementation is authorized automatically.

Separate later approvals remain necessary for the concrete storage migration, protected key/credential lifecycle, trusted confirmation UI, authenticated provider/backend selection, platform-specific object binding and positive launch integration. At-most-one attempt trades availability for safety; database restoration, uncertain outcomes and key loss can invalidate work. No guarantee of actual isolation or complete cleanup exists without backend evidence.

Phase 1–3 native GUI acceptance and previous fault/cross-platform limitations remain open. No workflow orchestration, evaluation, Claude, replay, GC or sandbox feature is marked shipped. Stop after this design milestone.
