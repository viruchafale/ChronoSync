# Phase 04 — Multi-agent architecture and execution-policy design

## Phase 4D authorized repository milestone

The subsequent request authorizes focused admission persistence and migration 3, atomic claims/audit/idempotency and conservative recovery. Schema and privacy/restoration limits were documented before migration implementation. Three tables persist the bounded Phase 4C aggregate and reference uniqueness; transitions replay the unchanged domain. No positive provider, supervisor consumer, process launch, UI or public contract is added.

Deliverables and checks are in the separate [Phase 4D status](../IMPLEMENTATION_STATUS_PHASE_04D.md). Prior Phase 4A–4C reports remain historical evidence. Acceptance includes concurrency/fault/upgrade tests, unchanged domain/canonical fixtures and conservative supervisor behavior, plus portable/native checks where supported. Stop after storage/recovery. Native GUI acceptance from Phase 1–3 remains open; authentication/key/backend/positive-launch prerequisites remain unapproved.

## Phase 4C authorized domain-contract milestone

The user's Phase 4C request explicitly authorizes internal domain types, bounded canonical encoding and pure transition tests. Implemented scope: immutable binding version 1, explicit external identity/input/approval/evidence references, caller-supplied epoch, challenge and one-use modeled claim/outcome semantics. No SQLite/IPC/UI/supervisor changes or positive launch path. Historical Phase 4B report is preserved.

The new [canonical specification](../SYSTEM_DESIGN.md) records exact bytes/order/bounds and their existing-budget rationale. [Phase 4C status](../IMPLEMENTATION_STATUS_PHASE_04C.md) records actual checks. Unresolved principal authentication, private commitment algorithm/key management and backend race-free identity remain external interfaces/limitations, not production defaults. This scope does not resolve or implement them.

Acceptance: fixed vectors, malformed/duplicate/order/bounds tests, approval/evidence/epoch/refusal transitions and post-claim invariants; existing portable/native checks where available. Stop at domain types/tests. Persistence must later enforce atomic revisions, approval consumption and unique attempts; it requires its own approved schema/recovery contract. No later milestone is automatically authorized.

## Phase 4B design milestone and proposed implementation sequence

Phase 4B authorizes audit, durable admission/launch-binding design and test planning only. It does not implement any milestone below. See [ADR-0007](../decisions/ADR-0007-durable-admission-launch-binding.md) and the separate [Phase 4B status](../IMPLEMENTATION_STATUS_PHASE_04B.md). Original design and Phase 4A evidence remain historical records, not retroactively changed acceptance.

| Future milestone                                | Prerequisites and likely files                                                                                                                        | Tests and security review                                                                                                                                                                               | Acceptance and exclusions                                                                                                                                                                             |
| ----------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A. Admission/binding domain contracts           | Approve canonical encoding, identity/privacy, epoch and revocation semantics; proposed internal/admission and policy boundary docs                    | Pure golden canonical vectors, argv empty/boundary cases, unsupported versions, state transitions, duplicate/replay, secret redaction; privacy and authority review                                     | Deterministic immutable bindings and refusal states; no storage migration, process, backend or UI                                                                                                     |
| B. Durable approval/admission storage           | A plus approved concrete DDL and stopped-backup rollback plan; internal/storage admission repository and recovery                                     | Real schema-1/2 upgrades, FK/version/checksum errors, transaction failure at every boundary, competing claims, idempotency conflicts, ambiguous commit and prior-epoch recovery; audit integrity review | At most one persisted claim, preserved old event/CAS bytes, unknown outcome without retry; no OS launch or provider positivity                                                                        |
| C. Trusted provider and approval provenance     | A; B needed for persistent approvals; approved registration and trusted-host confirmation/key lifecycle; policy/provider port, host approval boundary | Forged IPC evidence, expiry/clock rollback, revoked generation, changed backend, wrong scope, unsupported controls; provider authority review                                                           | Conservative default remains, test providers cannot register in production; no positive launch or new credential grants                                                                               |
| D. Backend feasibility and implementation       | Separate target OS/version/threat model/privilege/dependency approval; supervisor platform files and new backend port                                 | Approved disposable adversarial file/network/socket/credential/resource/descendant tests, packaging compatibility and crash evidence                                                                    | Precisely scoped verified capabilities and object-binding mechanism; no universal sandbox claim, no automatic install/elevation, no production admission yet                                          |
| E. Supervisor admission consumption and binding | A–D plus reviewed durable claim protocol; recorder/adapter launch planning, supervisor, admission repository                                          | Fault injection around claim/spawn/outcome, concurrent revocation, path/handle replacement, mutated argv/env/stdin, expired evidence, backend death and attempted retry; final security review          | No launch without current approved binding and durable claim; unknown outcomes consumed; retain consent/cancel/output regressions. Positive launch requires explicit authorization for this milestone |
| F. Typed IPC and truthful UI                    | Prior authoritative engine behavior and approved UX/authentication; contracts, ipc, Rust commands/bridge, React controls                              | Version negotiation, hostile/duplicate requests, scope checks, challenge replay, refused/expired/unknown display, keyboard/native GUI tests                                                             | Engine owns decisions; no sandbox badge beyond verified scope, no silent legacy fallback. No workflows, Claude, evaluation, replay or GC                                                              |

Discovery, provider trust and backend feasibility may inform one another, but positive production evidence is blocked until actual backend tests pass. A mock provider can test pure decision/state logic only. Adapter.Version is currently a separate process; future approval must either use trusted cached artifact metadata without execution or model that probe as a separately bound/approved attempt before the main execution. Never treat a probe as a free extra launch under a one-use admission.

Review still required: exact canonical bytes and bounds; native approval identity strength; secret commitment key storage/rotation; backend atomic binding; capability revocation hooks; migration DDL/version; response to revocation of running work. The recommended first approval is milestone A only, with those contract choices made concrete before coding. All other milestones need separately scoped approval. No next implementation is implicitly authorized by completion of Phase 4B.

## Phase 4A authorized extension

The subsequent Phase 4A request explicitly authorizes internal Go policy types/resolution, conservative capability reporting and safe pre-launch refusal tests. This supersedes the design-only constraint below for that slice, not for sandbox implementation. Deliverables: explicit mandatory dimensions, exact evidence/target/scope matching, deterministic decision reasons, conservative unknown provider, and supervisor refusal before process setup. Existing Codex launches/UI are unchanged and not policy-verified. No positive admission, schema/migration, backend, workflow inheritance or weaker-policy approval flow is introduced.

Acceptance: synthetic allowed/refused resolution and pre-spawn refusal tests plus existing execution regressions, make check/native-check, documentation and separate [Phase 4A status](../IMPLEMENTATION_STATUS_PHASE_04A.md). The original completed design report is preserved. Decision-register approval for core types and refusal is supplied by the new request; durable admission, mandatory product permission profiles, workflow ceilings, positive provider/launch binding and platform investigation remain separate approval gates. Stop after Phase 4A.

## 1. Objective and user-visible outcome

Document the Multi-Agent AI Workflow Debugging and Evaluation Platform architecture and requested/effective policy boundary. This authorized milestone is design only; no new runtime feature ships.

## 2. Problem

Existing runs, snapshots and timelines need explicit correlation and evaluation evidence. Current process supervision and upstream sandbox flags do not establish general isolation. The original sandbox implementation plan is deferred under the user's explicit narrower scope.

## 3. Scope and non-goals

Audit, conceptual model, correlation, incremental persistence proposal, policy model, platform capability matrix, evaluation boundary and decisions. Exclude migrations, event schemas, launch permissions, backends, orchestration, Claude, evaluator execution, replay, cleanup/GC and host changes. No network research or external service calls.

## 4. Prerequisites and dependencies

Use actual Phase 1–3 code, ADR-0001 through ADR-0004 and current test evidence. Existing native GUI gates remain unverified; this milestone does not accept those phases. Existing installed dependencies suffice; no new dependency.

## 5. Design and component boundaries

Existing React → typed Rust → private stdio Go → domain ports remains. Workflow catalog references existing runs/events/snapshots. Policy resolver is separate from backend enforcement and observations. Evaluation reads frozen evidence and writes separate judgments. See [architecture](../ARCHITECTURE.md), [system design](../SYSTEM_DESIGN.md), [data model](../DATA_MODEL.md) and [security matrix](../SECURITY_MODEL.md).

## 6. Ordered design tasks

1. Audit source, interfaces, schemas, persistence and acceptance evidence; preserve source hashes.
2. Define workflow identity, membership, provenance and ordering without rewriting events.
3. Define policy resolution and conditional platform capabilities without selecting a backend.
4. Define immutable evaluation inputs and comparable results.
5. Record decisions, update existing docs and validate offline checks.
6. Stop. Any implementation requires a separately authorized scope.

## 7. Expected files/modules

Documentation only: README, AGENTS scope, project specification, architecture, system/data/security/testing documents, status, changelog, index, this plan and ADR-0005/0006. Runtime packages, schemas, SQL, lockfiles and dependencies remain unchanged.

## 8. Interfaces and contracts

Proposed RequestedPolicy/EnforcementCapability/EffectivePolicy/PolicyObservation, workflow membership/relationships and evaluation evidence manifests. These are conceptual contracts, not exported types. Preserve engine 0.3.0, protocol 1, event 2, run 1, snapshot manifest 1 and migrations 1–2.

## 9. Security and failure considerations

No unsupported required capability may be silently weakened. Evidence provenance, credential/history separation and bounded queries matter across workflows. Restoring a branch does not authorize execution. Existing read-only flags, cwd and process groups are not isolation. Unknown cleanup/capability state stays visible.

## 10. Unit, integration and end-to-end tests

This milestone runs existing safe offline checks and document validation. Future resolver, correlation, migration, evaluation and adversarial OS tests are specified in [testing strategy](../TESTING_STRATEGY.md); they are not implemented or passed here. Native GUI walkthrough remains a separate gate.

## 11. Acceptance criteria

Audit matches actual code; every proposed concept is distinguished from implemented behavior; policy has four separate concepts; matrix covers all three OSes with explicit uncertainty; evaluation preserves evidence; required docs and decisions exist; source/contract hashes remain unchanged; checks have actual results; Phase 1–3 acceptance gaps remain.

## 12. Validation commands

Run `make check` with the documented temporary Go caches and network fetching disabled; `make native-check` with the installed temporary Rust toolchain and Cargo offline. Run local Prettier and `npm run check:docs` after edits. Extend local link inspection to new ADRs; manually inspect Mermaid edges and labels (no renderer available/claimed). Source SHA-256 inventory before/after substitutes for Git diff because this workspace has no .git. Exact results reside in [status](../IMPLEMENTATION_STATUS_PHASE_04.md).

## 13. Decision register and unresolved questions

All recommendations below are design positions, **not implementation approval**. Every row requires separate implementation authorization. Backend/credential/privilege decisions additionally require concrete platform evidence and explicit scope approval.

| Decision / options                                           | Recommendation and trade-off                                                                                 | Security and compatibility                                                                               | Evidence needed / approval gate                                                                                                     |
| ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| 1. Workflow entity vs inferred correlation                   | Small first-class catalog plus membership; more metadata than tags, stable identity/revisions                | No execution authority; additive metadata, old events unchanged                                          | Membership/query prototypes and scale tests; approve catalog and migration before code                                              |
| 2. Existing-event linkage: rewrite vs side associations      | Associate existing runs; events inherit context by lookup; historical runs initially unassigned              | Never fabricate workflow history; no event-v2 rewrite                                                    | Hash-preservation and transactional upgrade/recovery tests; approve future SQL                                                      |
| 3. Parent/handoff/dependency: one edge vs typed provenance   | Separate typed relationships; unresolved refs stay pending; more validation                                  | No inferred link presented as observed; parent cycles rejected, handoff observations may repeat          | Missing/duplicate/conflicting references and cycle tests; approve semantics before API                                              |
| 4. Policy per run vs workflow vs both                        | Workflow defaults/ceiling plus per-run immutable resolution; approval bookkeeping cost                       | No implicit widening or branch execution; legacy runs show policy not recorded                           | Precedence, identity drift and approval binding tests; approve mandatory restrictions and storage                                   |
| 5. Capability failure: refuse vs best effort                 | Refuse required gaps; allowed weakening creates newly approved request                                       | No silent downgrade or sandbox boolean; existing launches unchanged in this task                         | Synthetic complete/partial/stale/unknown matrices; approve resolver and any weaker mode                                             |
| 6. Backend order: native vs containers                       | First macOS supported mechanism feasibility and Linux comparison; Windows prerequisites; containers optional | No backend selected, no automatic elevation/install; current platform blocks remain                      | Authoritative API/version research and authorized disposable probes; approve target, threat model and privileges before experiments |
| 7. Event-v3 vs metadata extension                            | Workflow catalog needs no event change; future new event semantics/provider support reviewed separately      | Keep event-v2/protocol1; negotiated future commands and TS/Go/schema readers must agree                  | Old-client compatibility and strict enum tests; approve any later version/capability changes                                        |
| 8. Evaluation refs: paths/live summaries vs frozen manifests | Pin event hashes/high waters, snapshots, definition/config/dataset versions; extra artifact retention        | Judgments cannot rewrite evidence; unknown cost/tokens remain missing                                    | Integrity, missing evidence, incompatible metric and reproducibility tests; approve artifact schema and retention before runner     |
| 9. Claude: core branches vs separate adapter                 | Separate adapter using Version/Record/Sink normalization and conformance suite                               | Current model/schema is Codex-only; authentication/flags/permission mapping cannot be assumed compatible | Authorized CLI/API research, synthetic fixtures, provider-specific consent and version negotiation; separate Phase 7 approval       |
| 10. Implementation readiness: proceed now vs staged approval | Stop at design; next pure resolver only, followed by separately scoped catalog/backend work                  | No implicit migrations, untrusted launches or Phase 5 work; GUI gates remain open                        | Approve exact types, deny defaults, tests and exclusions before next implementation; no approval inferred from this design          |

Open questions: supported macOS API/distribution route; Linux feature/delegation matrix; Windows CLI/ACL/handle compatibility; mandatory product restrictions; capability evidence freshness; durable approval identity and revocation; workflow size caps and revision retention; evaluator artifact format and source-revision instrumentation. None is resolved by a cross-build or ordinary process test.

## 14. Exit criteria

Design acceptance only: coherent audit/design/decision register and passing applicable documentation checks. Sandbox implementation remains NOT_STARTED. Next recommended implementation task, only after approval: pure Go versioned policy types and a fail-closed resolver with synthetic capability/precedence/staleness tests; no process launching, backend probes, schema changes or UI security claims. This is distinct from completing outstanding native GUI acceptance.

## 15. Documentation updates

Updated existing documents rather than adding duplicate SECURITY/TESTING files. ADR-0005 records workflow/evaluation decisions; ADR-0006 records policy boundary and deferred backend selection. Historical conflicting roadmap text is explicitly labeled superseded; phase evidence and outstanding GUI/fault gates are retained.

## 16. Status

Phase 4D durable admission repository, migration 3, atomic claims/audit and conservative recovery: `COMPLETE`; focused race tests, `make check` and `make native-check` passed. Evidence, restoration limits and remaining trust gates are recorded in the separate [Phase 4D report](../IMPLEMENTATION_STATUS_PHASE_04D.md). No execution integration or positive admission is enabled.

Phase 4C internal domain contracts and pure tests: `COMPLETE`; portable/native checks passed as recorded separately. Authentication/key mechanisms and positive admission remain unimplemented and unapproved.

Phase 4A core and conservative refusal boundary: `COMPLETE`, with portable/native checks and targeted tests recorded in its separate status file. Positive admission and sandbox backends remain `NOT_STARTED`.

`COMPLETE` — authorized architecture/design milestone only, subject to the recorded validation evidence. Runtime sandbox implementation: `NOT_STARTED`. No Phase 5 work authorized. Phase 1–3 native GUI acceptance remains UNVERIFIED/BLOCKED.
