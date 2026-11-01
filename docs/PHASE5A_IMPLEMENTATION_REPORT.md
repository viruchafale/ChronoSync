# Phase 5A implementation report — replay and regression foundation

## Result and scope

Phase 5A adds a backend-only replay fixture contract, storage-backed exporter, strict importer, isolated evaluator and success-only regression-fixture handoff. It uses completed persisted event-v2 history and does not add a database migration, public IPC, renderer UI, network client, subprocess runner, provider adapter or Phase 4 admission/policy integration.

Recorded history, recorded-interaction replay, regression evaluation and live AI rerun remain distinct:

- **Recorded history** is the original integrity-checked event timeline.
- **Recorded-interaction replay** in 5A reprocesses inert fingerprints from supported command-completion observations through a controlled evaluator.
- **Regression evaluation** compares the controlled evaluator's output fingerprint and exit code with recorded expectations.
- **Live AI rerun** is a new provider execution and is absent from replay.

## Actual fixture contract

Fixture format version 1 contains a deterministic replay ID, source execution ID, event-v2 version, source count/high-water, Codex adapter, redaction profile, supported type declaration and ordered interactions. The only supported interaction is `command.completed`.

Each interaction contains source event ID/sequence and:

- input: SHA-256 of recorded redacted command text and item ID;
- expectation: SHA-256 of recorded redacted output and the integer exit code.

Raw command, item ID and output text are absent. Canonical struct JSON avoids maps; the replay ID derives from the fixture with its ID cleared, and an envelope SHA-256 covers the final fixture body. Import rejects noncanonical bytes, unknown fields, trailing data, changed digests, unknown versions/types, malformed IDs/digests/ranges, duplicate/reordered interactions, inconsistent nested sequence and resource-limit violations. The digest detects accidental alteration; it is not a signature or authenticity proof against an editor who recomputes the envelope.

Exporter eligibility is intentionally narrow: completed run, complete history, consecutive event sequence, integrity-checked details, at least one supported command completion and only approved lifecycle boundary kinds. Unsupported events, partial/damaged/active history, missing details, incomplete command fields, redaction changes/markers and truncation refuse export with typed errors. Repeated export of unchanged eligible history is byte-stable.

## Evaluation and regression behavior

The evaluator interface receives only inert input fingerprints and source sequence. It returns output fingerprint and exit code. Evaluation validates the fixture first, checks cancellation around every interaction, reports field-level expected/actual mismatches, sanitizes unsupported reasons, and distinguishes completed, failed, cancelled and unsupported status. Completed evaluations may pass or contain regression mismatches.

`RegressionFixture` returns a detached copy of canonical bytes only when the result belongs to that exact fixture, completed, passed, evaluated every interaction and has no mismatch/unsupported entries. It does not choose or write a path; a future consented export boundary must own private persistence and sharing review.

## Safety evidence

- Package production imports exclude `os/exec`, networking, supervisor and provider adapters; a source-boundary test enforces this inventory.
- Evaluator input types have no raw command/output field, and tests reject any raw synthetic command/item/output appearing in fixture bytes.
- A secret-shaped command is rejected when re-redaction changes it. Marked-redacted and truncated values refuse export.
- Unsupported source events and unavailable event details fail rather than being skipped or invented.
- Tests use the real SQLite store with synthetic events and controlled in-memory evaluators. They do not start Codex, another provider, or a historical command.
- Results are not consumed by policy, admission, approval, supervisor, recorder or snapshot code.

These facts establish the package boundary exercised by tests. They do not establish OS-level offline enforcement against a malicious evaluator or process, universal secret detection, command behavior, provider replay or deterministic LLM execution.

## Files and compatibility

- `engine/internal/replay`: contract, exporter, strict importer, evaluator and tests.
- `docs/PHASE5_IMPLEMENTATION_PLAN.md`: mandatory pre-source reconnaissance and plan.
- `docs/decisions/ADR-0009-replay-fixture-foundation.md`: architecture decision.
- Phase/status/architecture/data/security/testing/changelog documents: implemented boundary and evidence.

SQLite remains migration 3. Event schema remains v2. Engine protocol remains v1 with the same capabilities. No TypeScript/Rust contract or dependency changed, and the existing DMG was not rebuilt or altered.

## Verification

| Command                                                                       | Result                                                                                                                                                                                                      |
| ----------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `go test -race ./internal/replay` with offline proxy and local module cache   | Passed focused storage/export/import/evaluation/safety tests.                                                                                                                                               |
| `go vet ./internal/replay` with offline proxy and local module cache          | Passed.                                                                                                                                                                                                     |
| `make check` with offline proxy and established Go build/module caches        | Passed all Go format/vet/race/build checks (including replay), Prettier, ESLint, strict TypeScript, 22 Vitest tests, frontend build, docs, version integration and synthetic recorder/snapshot integration. |
| `make native-check` with offline Cargo and established Rust/Cargo environment | Passed sidecar build, Rust format, locked Clippy with warnings denied, real-engine bridge negotiation/cancellation test, frontend build and optimized native executable. No bundle/DMG command ran.         |

The first focused attempt pointed `GOMODCACHE` at a temporary cache whose download metadata had been reset, so locked SQLite lookup stopped before compilation. Reusing the already-populated normal module cache resolved the invocation without changing dependencies.

## Limitations and remaining gates

- Phase 5A compares fingerprints for controlled command-completion observations; it does not run commands or validate their effects.
- Histories containing messages, reasoning, tools, file reports, warnings/errors, unknown adapter events or workspace changes are currently unsupported as a whole.
- Fingerprints can leak equality and permit guessing of low-entropy values. Redaction remains heuristic.
- No fixture file picker/storage policy, inspect-before-share UI, IPC contract, native GUI acceptance, clean-machine exchange or CI network-denial harness exists.
- No controlled provider/MCP/HTTP transport is recorded, so provider and tool replay remain impossible.
- No snapshot is bound into the fixture, and no workspace is restored for evaluation.
- Offline behavior is an architectural package boundary, not an OS sandbox guarantee.

## Recommended next milestone

Phase 5B should design and test one explicit synthetic transport port with consented pre-storage redaction, immutable request/response fingerprints, repeated-request matching, unmatched/ambiguous refusal and a test harness with network actively denied. Only after that contract is stable should private IPC and a minimal inspect/export/evaluate UI be considered. Live AI rerun must remain a separate action with fresh consent.
