# Phase 07 — Claude Code adapter

## 1. Objective and user-visible outcome

Record a supported Claude Code workflow through the shared engine while preserving provider-specific evidence.

## 2. Problem

A second provider must not force core event/storage/branching logic to depend on Codex-specific types.

## 3. Scope and non-goals

Inspect current official Claude interfaces, implement adapter and capability mapping, compatibility fixtures and opt-in live validation. No assumed parity or undocumented internal APIs.

## 4. Prerequisites and dependencies

P1–P6 shared contracts stable; current official Claude docs and installed tooling/auth inspected at implementation time; user consent.

## 5. Design and component boundaries

Claude adapter implements the same port; provider-specific fields stay in sanitized payloads and capability manifest. Shared engine/store/supervisor/replay remain provider-neutral.

## 6. Ordered implementation tasks

1. Audit then-current Claude CLI/SDK/server interfaces and auth; record selection ADR with dated official sources.
2. Define capability gaps and event mapping without fabricating Codex equivalents.
3. Add labeled parser fixtures and shared adapter conformance tests.
4. Implement supported launch/capture/cancellation behind existing supervisor.
5. Validate storage/snapshots/branching and supported replay boundaries with capability gating.
6. Add accessible adapter selection and provider-specific limitations.
7. With consent capture a disposable live run and cancellation/error cases; document unavailable auth honestly.
8. Review core for provider leakage and update docs/compatibility matrix.

## 7. Expected files/modules

engine/internal/adapter/claude/, tests/fixtures/claude/, contracts/capabilities, apps/desktop/src/adapters/, docs/CLAUDE_INTEGRATION.md (new).

## 8. Interfaces and contracts

Existing Adapter/RunSpec/EventSink/RunResult; versioned Claude capability map; no new core provider-specific union unless justified.

## 9. Security and failure considerations

Apply same redaction/consent/isolation policies; unsupported cancellation or permissions block capability rather than degrade silently.

## 10. Unit, integration and end-to-end tests

Unit: parser/unknown types. Integration: both adapters pass conformance, supervisor and persistence suites. E2E: approved live Claude capture plus cancellation and inspection; offline fixture replay only where supported.

## 11. Acceptance criteria

Codex regression suite stays green; Claude observable events stored without core rewrites; unsupported features disabled with reasons; successful real run evidence and cleanup verified.

## 12. Validation commands

```sh
make check
npm run desktop:build
```

These are existing baseline commands. Add phase-specific test suites to these targets and an explicit opt-in live or platform test command where needed before claiming this phase passes. Baseline success alone does not satisfy the phase acceptance criteria.

## 13. Risks and unresolved questions

Future API stability, authentication availability, licensing and event completeness are intentionally unresolved until this phase.

## 14. Exit criteria

Both adapters pass shared contracts and supported live milestones; capability differences and evidence recorded.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`NOT_STARTED`

No implementation or acceptance tests for this phase have been completed. Await authorization after preceding phase review.
