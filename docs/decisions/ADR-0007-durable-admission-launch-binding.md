# ADR-0007: Durable one-use admission and launch binding

## Phase 4E trust prerequisite refinement — 2026-10-23

[ADR-0008](ADR-0008-provider-and-approval-trust.md) implements closed conservative provider registration and IPC rejection without changing canonical/domain/storage contracts. The user explicitly requires OS-backed authentication; no trusted approval issuer or validation producer can be implemented until its platform mechanism and binding are reviewed. Durable references remain unauthenticated. No migration or positive supervisor consumption is authorized.

## Phase 4D persistence resolution — 2026-10-23

The user explicitly authorizes the repository/migration/atomic claim/recovery milestone while excluding positive launch. Schema was documented in DATA_MODEL before implementation: migration 3 uses an admission aggregate, append-only operation/result history and unique typed-reference rows. Immutable binding and operation envelopes contain the domain's existing references, avoiding speculative provider/workflow tables. Projection hydration replays domain NewBinding/New/Apply; no replacement state machine.

Use transaction-assigned revision as durable local audit order, conditional updates and unique claim/reference constraints. Same-key/same-body calls return historical replay-marked receipts; conflicting content refuses. Every store open/recovery establishes a fresh random session epoch, invalidates old unclaimed readiness, and preserves uncertain claims. This resolves repository semantics only, not identity authenticity, key management, OS enforcement or rollback resistance. Live in-place restoration is unsupported; a stopped restored backup must reopen. The positive execution boundary remains closed. See [Phase 4D evidence](../IMPLEMENTATION_STATUS_PHASE_04D.md); earlier phase reports remain intact.

## Phase 4C scope resolution — 2026-10-23

The subsequent Phase 4C request explicitly authorizes pure internal binding canonicalization and admission state types/tests. Its implemented version-1 byte profile and limits are specified in [system design](../SYSTEM_DESIGN.md); the new contract does not redefine Phase 4A's policy digest or any persisted/public format. Permanent modeled consumption and unknown recovery outcomes follow this ADR.

Approval principal/authentication, private-input commitment/key management, fresh epoch allocation/restoration detection and backend object binding remain unresolved external responsibilities. Phase 4C models bounded references and explicit unresolved identity; it does not select a production mechanism. Synthetic external validation exercises pure state transitions and is not authentic evidence. These approval gates remain in force before real admission or persistence integration. No Phase 4B historical result is rewritten.

Status: proposed for review in Phase 4B, 2026-10-22. Design only; no migration, launch behavior or backend is authorized by this ADR.

## Audit evidence

Phase 4A Resolve accepts supplied in-process evidence, uses explicit time and exact restrictions, and hashes a canonical ordered Go policy representation. It does not authenticate providers, bind argv/environment/stdin, obtain filesystem identities, implement revocation or persist admission. Its mandatory approval/audit dimensions require a real future provider. ConservativeProvider reports unknown; supervisor.Run refuses every non-nil Policy before process setup. Both Codex version probing and execution currently use nil Policy. The version probe is itself an executable launch and cannot be exempted from a future positive-admission design merely because its argv is --version.

Recorder creates run metadata before adapter.Version and adapter.Record, then persists observable lifecycle/events. Existing consent is a boolean, not durable bound approval. Rust forwards typed start inputs; React resets consent on edits and already describes process limits as distinct from isolation. SQLite has a process lease, bounded writer, ordered migrations 1–2 and recovery that marks interrupted runs without proving process cleanup. It cannot atomically launch a process with a database transaction.

## Recommendations and alternatives

| Question / alternatives                                                | Recommendation / rationale                                                                                   | Trade-off, evidence and approval                                                                       |
| ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------ |
| Treat Satisfied as a launch token vs separate admission                | Separate durable authorization, one-use consumption and process-start observation                            | More state, but avoids forged evidence and false start claims; approve lifecycle before implementation |
| Retry failed/uncertain launches vs consume permanently                 | One unique attempt, claimed durably before OS call; never reuse after claim                                  | At-most-one attempted spawn, no exactly-once guarantee; injected crash-window tests required           |
| Single SQL/OS transaction vs recoverable protocol                      | Separate claim and outcome transactions, explicit unknown state                                              | Availability sacrificed after ambiguous outcomes; no unsafe reconciliation by PID                      |
| Event diagnostic strings vs typed admission storage                    | New relational side records with immutable versioned bodies/audit                                            | Keeps event/run bytes unchanged; requires reviewed migration and backup/rollback tests                 |
| Frontend-supplied evidence vs engine-owned providers                   | Only registered trusted backend providers; frontend carries intent/consent challenge response                | Trusted confirmation strength and backend provenance remain gates; no plugin authority assumed         |
| Reuse policy digest vs full launch binding                             | Preserve policy digest; add a versioned binding covering actual execution plan                               | Canonical vectors and privacy review required; no silent change to internal policy v1                  |
| Persist secret hashes vs ephemeral commitments                         | No raw secrets or public hashes of guessable values; ephemeral keyed commitments or approved broker versions | Key loss/restart invalidates admission; key lifecycle and credential grants need approval              |
| Path/hash recheck vs backend-bound object identity                     | Require backend-supported binding for required race-free guarantees                                          | Generic os/exec path lookup is insufficient; supported platform mechanism remains undecided            |
| Persisted prepared admission reuse after restart vs epoch invalidation | Invalidate prior-epoch readiness; retain evidence                                                            | Requires new approval; prevents backup/restart replay without claiming process cleanup                 |

The [system design](../SYSTEM_DESIGN.md) specifies lifecycle, binding matrix and failure semantics. The [data model](../DATA_MODEL.md) contains concrete proposed tables, transactions and rollback limits. The [security model](../SECURITY_MODEL.md) defines IPC and provider authority. No backend is selected.

## Approval gates

Approve the independently reviewable plan in [Phase 4](../phases/PHASE-04-SANDBOX-GUARDRAILS.md) one milestone at a time. First candidate: internal admission/binding/state types and pure transition/canonicalization tests only, with no migration or launch path. Before that implementation, explicitly approve the canonical binding format, one-use/epoch/revocation semantics, approval identity model and treatment of private inputs. Storage migration, trusted provider, launch-binding implementation, each OS backend and UI claims require separate review/authorization. Synthetic evidence is never production authorization.

Phase 4A remains unchanged and fail-closed. Phase 1–3 native GUI gates remain open. Multi-agent orchestration, evaluation, replay and sandboxing do not ship in Phase 4B.
