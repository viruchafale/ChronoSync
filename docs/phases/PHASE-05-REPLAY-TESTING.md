# Phase 05 — Interaction replay and regression testing

## 1. Objective and user-visible outcome

Export redacted versioned fixtures and replay supported captured interactions without live model/network calls.

## 2. Problem

Historical logs do not automatically contain sufficient information to replay external services or reproduce outcomes.

## 3. Scope and non-goals

Explicit transport capture, optional scoped proxy evaluation, canonical matching, fixture import/export, assertions and outcome comparison. No transparent universal Codex TLS interception or whole-run determinism claim.

## 4. Prerequisites and dependencies

P2/P3 durable objects; P4 enforced offline policy; explicitly supported observable transport boundary.

## 5. Design and component boundaries

Capture/replay transport port wraps supported tools/adapters. Fixture runner validates manifests and objects, denies live egress and emits structured assertion differences.

## 6. Ordered implementation tasks

Phase 5A completed the safe fixture foundation before transport capture: source inventory, canonical bounded fingerprint fixtures, strict import, isolated comparison and regression handoff. The detailed pre-source plan and evidence are in [PHASE5_IMPLEMENTATION_PLAN.md](../PHASE5_IMPLEMENTATION_PLAN.md) and [PHASE5A_IMPLEMENTATION_REPORT.md](../PHASE5A_IMPLEMENTATION_REPORT.md).

Phase 5B completes the next isolated experiment: a dependency-light synthetic transport port, separate strict fixture, pre-fingerprint refusal/redaction contract, immutable entries, explicit occurrence matching and structural network-denial tests. See the [Phase 5B plan](../PHASE5B_IMPLEMENTATION_PLAN.md) and [implementation report](../PHASE5B_IMPLEMENTATION_REPORT.md). This is not real transport capture or provider replay.

1. Inventory controllable transports and document provider/MCP/proxy bypass cases. Initial inventory complete; no controlled provider transport exists.
2. Implement consent scopes and pre-storage request/response redaction; mark nonreplayable records. Phase 5B implements redaction/refusal for synthetic allowlisted metadata only; consent and production capture remain open.
3. Define canonical matching with method/target/body/sequence/scope, repeated request handling and ambiguity errors. Phase 5B verifies identity/operation fingerprints and occurrences without bodies or real targets.
4. Implement strict offline replay with no automatic live fallback. Phase 5B proves structural isolation for the synthetic package; OS-level denial and production runner remain open.
5. Define portable fixture schema/provenance/redaction/compatibility and bounded safe importer. Phase 5B adds a separate experimental internal format; promotion criteria remain open.
6. Implement export pinning/hash validation and inspect-before-share workflow.
7. Add assertion runner and comparison reports for events, files and interactions.
8. Add CI command for offline fixtures and tests with network actively denied.

## 7. Expected files/modules

engine/internal/{interaction,replay,fixture}, contracts/fixture.schema.json, tests/fixtures/replay/, apps/desktop/src/replay/, scripts/.

## 8. Interfaces and contracts

Fixture manifest v1, request/response objects, ordered InteractionKey, ReplayResult and assertion failure locations; schema migration policy.

## 9. Security and failure considerations

Do not install TLS trust roots implicitly. Proxy bypass/pinning/direct sockets remain unsupported. Imports reject archive traversal/bombs; redacted inputs may invalidate matching.

## 10. Unit, integration and end-to-end tests

Unit: matcher/canonicalizer/redactor. Integration: record controlled test transport then replay offline, repeated/ambiguous/unmatched interactions, corrupted/old manifests. E2E: export → clean machine import → offline assertions and useful mismatch report.

## 11. Acceptance criteria

Fixture asserts represented responses with zero provider calls under denied egress; missing interactions fail visibly; exported secret sentinel absent; hash/schema incompatibility rejected; deterministic fixture assertions repeat.

## 12. Validation commands

```sh
make check
npm run desktop:build
```

These are existing baseline commands. Add phase-specific test suites to these targets and an explicit opt-in live or platform test command where needed before claiming this phase passes. Baseline success alone does not satisfy the phase acceptance criteria.

## 13. Risks and unresolved questions

Provider traffic inaccessible, redaction versus replay fidelity, streaming response fidelity and portability. Replay scope must be explicit.

## 14. Exit criteria

Offline runner, redaction/import security tests and scoped replay evidence pass; no universal reproducibility claim.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`COMPLETE` for Phase 5A–5F supported controlled fixture workflow. See the [consolidated completion report](../PHASE5_COMPLETION_REPORT.md).

Phase 5A implements command-completion fingerprint fixtures; 5B adds the isolated synthetic matcher; 5C adds immutable private lifecycle storage; 5D adds unified evaluation; 5E adds typed desktop controls and native consent; 5F verifies the disposable workflow and restart. Phase 5 still does not execute historical commands, call providers, restore workspaces or claim whole-run/LLM determinism. Phase 6 requires separate authorization.
