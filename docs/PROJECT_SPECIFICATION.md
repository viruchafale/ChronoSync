# Project specification

## Phase 4 product direction — proposed, not runtime functionality

The product vision is a **Multi-Agent AI Workflow Debugging and Evaluation Platform**. Extend historical debugging with explicit workflow membership, evidence-backed relationships and separate evaluations; retain the existing recorder, timeline and snapshot foundation. A workflow groups a logical task, a run is one adapter execution, and an event is one captured observation. Grouping is not scheduling or permission to execute.

Proposed user journeys: group existing runs without changing their events; inspect an observed handoff with its evidence; compare two evaluations with compatible input and metric definitions; review requested versus enforceable permissions before a future launch. Unknown relationships and unavailable measurements must remain visibly unknown.

This milestone authorizes documentation only. It excludes migrations, schema changes, simultaneous agent execution, automatic handoffs, sandbox implementation, new launch permissions, Claude support, evaluation execution/UI, replay, destructive GC and cleanup. Phase 1–3 GUI gates remain required. No network research or dependency installation is part of this milestone.

Design acceptance: each relationship has provenance; old event bytes remain unchanged; policy resolution refuses an unsupported required restriction; evaluation judgments cannot modify source evidence. These are future behavioral requirements, not passing runtime tests. Decisions and implementation approvals are recorded in the [Phase 4 plan](phases/PHASE-04-SANDBOX-GUARDRAILS.md).

## Goals, users, and use cases

Serve developers investigating failed agent edits, maintainers comparing alternative prompts, and tooling engineers creating offline regressions. A developer selects an explicit workspace and execution policy, records observable execution, inspects event details and changes, and eventually forks a checkpoint without changing the original history. A maintainer exports a redacted fixture and runs assertions without provider calls for represented interactions.

Local operation must not require a cloud service. Live Codex use still contacts its provider and requires user authentication. Team/cloud synchronization, universal agent support, hidden reasoning capture, and guaranteed deterministic LLM reproduction are non-goals.

## Operation vocabulary

| Operation                  | Meaning                                                | Guarantee boundary                                                |
| -------------------------- | ------------------------------------------------------ | ----------------------------------------------------------------- |
| Historical inspection      | Read immutable recorded evidence                       | Only captured and integrity-checked data                          |
| Interaction replay         | Serve a recorded response for a matching request       | Only supported and captured transport boundaries                  |
| Live re-execution          | Launch an agent/model again                            | May differ and may cost money                                     |
| State reconstruction       | Materialize supported file snapshot                    | Excludes uncaptured state, OS and provider internals              |
| Deterministic reproduction | Reproduce all relevant results under controlled inputs | Never claim without a scoped nondeterminism manifest and evidence |

## Measurable requirements

- R1 (P1): each accepted run and event has a unique local ID; unsupported upstream types remain visible as unknown evidence, never silently dropped.
- R2 (P1): terminal state includes process exit, adapter outcome, and reason; deadline/cancel tests terminate a process tree within a documented grace period.
- R3 (P2): after injected crashes, committed event sequence remains queryable and contiguous; unfinished runs become interrupted, never successful.
- R4 (P2/P3): every external object reference verifies SHA-256 on read; corruption produces a visible error.
- R5 (P3): modifying a branch cannot change ancestor event bytes or snapshots; restoration refuses unowned/dirty targets without explicit consent.
- R6 (P4): denied path, symlink escape, unsupported isolation, and network policies fail closed under adversarial tests.
- R7 (P5): exported replay fixtures run with outbound network denied and report unmatched interactions as failures.
- R8 (P6): manifests identify controlled and uncontrolled random, scheduling, filesystem, time, and external sources; ordering tests preserve evidenced dependencies without assuming distributed clocks are necessary.
- R9 (P6): configurable AST rules report file/rule/evidence and false-positive overrides; no semantic correctness guarantee.
- R10 (P7): shared adapter contract suite passes for Codex and Claude without provider-specific types leaking into core.
- R11 (all): inspectable errors, keyboard-operable UI, no fake actions, explicit project access/command consent, no automatic repository history modification.

Performance targets to measure in P2: paginated first 100 events from a 100,000-event local synthetic run within 200 ms on a recorded reference machine, bounded ingestion queues, and explicit overflow failure. These are acceptance targets, not current benchmarks. Accessibility target: semantic labels, focus visibility and keyboard paths for all shipped controls; test each new interactive view.
