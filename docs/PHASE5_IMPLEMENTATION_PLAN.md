# Phase 5A implementation plan — replay and regression foundation

## Authorization and scope

The user authorized the first safe Phase 5 milestone: export a bounded regression fixture from completed stored history, validate it as untrusted data, and evaluate supported recorded interactions without a provider, network call, subprocess, historical command execution, Phase 4 admission, or positive policy decision.

This milestone is backend-only. It does not add renderer controls, public IPC, a database migration, a new dependency, transport interception, live AI reruns, snapshot restoration, command execution, scoring, or a deterministic whole-run claim. Existing event v2, protocol 1, SQLite migration 3, snapshot formats, conservative policy refusal, and the submitted DMG remain unchanged.

## Repository findings

- Event v2 persists typed, redacted Codex observations with strict IDs, UTC timestamps, per-run decimal sequence, bounded payloads, and SHA-256 integrity. Large event JSON lives in the existing private payload object directory.
- `storage.Inspector` exposes stable high-water event pagination and integrity-checked detail reads. Completed runs can still be ineligible when history is partial/damaged or payload details are unavailable.
- Recorder redaction covers common credential patterns and truncates text, but is heuristic. Redacted or truncated values cannot be invented for replay.
- The adapter records command lifecycle observations, including command text, aggregated output, and exit code. It does not record a controlled provider request/response transport, model state, network interaction, or complete tool arguments/results.
- Snapshot capture preserves workspace bytes but does not prove those bytes correspond to every event or reproduce provider behavior.
- The production supervisor and Codex adapter are process-launch boundaries. Phase 5A must remain independent of both.

## Minimal contract

Add a pure Go `internal/replay` package with fixture format version 1:

- deterministic replay ID and source execution ID;
- source run/event versions, source high-water sequence, adapter, and redaction profile;
- ordered `command.completed` interactions only;
- command and item identifiers represented as SHA-256 fingerprints, never executable text;
- expected output represented as a SHA-256 fingerprint plus recorded exit code;
- an envelope SHA-256 over canonical fixture JSON;
- strict size, count, sequence, ID, version, digest, and unknown-field validation;
- typed export/validation/evaluation errors;
- completed, failed, cancelled, and unsupported evaluation results with mismatch locations.

Exporter eligibility is deliberately narrow. A run must be completed with complete history, exact consecutive event sequence, and only the explicitly allowed structural lifecycle kinds plus `command.completed`. Any unknown/unsupported kind, missing detail, gap, redaction marker, truncation, malformed command completion, or changed-on-re-redaction payload refuses export. This produces an honest supported subset instead of silently dropping evidence.

The evaluator receives only inert fingerprints and sequence metadata. It cannot receive or execute a historical command through this contract. A controlled evaluator returns output fingerprint and exit code; the runner compares them with the recorded expectation. This tests a caller-supplied controlled fixture system, not Codex or an LLM.

## Intended files

- `engine/internal/replay/model.go`: versioned fixture/result types and typed errors.
- `engine/internal/replay/export.go`: storage-backed eligible-history export and canonical envelope creation.
- `engine/internal/replay/validate.go`: bounded strict importer and integrity validation.
- `engine/internal/replay/evaluate.go`: cancellation-aware inert evaluator and mismatch reporting.
- `engine/internal/replay/replay_test.go`: storage integration, stable bytes, malformed/tampered/unsupported/incomplete/redacted/oversized/cancelled/failure/mismatch/repeatability and forbidden-boundary tests.
- Existing phase/status/architecture/security/testing/changelog indexes: accurate Phase 5A evidence and limitations.
- `docs/PHASE5A_IMPLEMENTATION_REPORT.md`: separate milestone report.

No contract schema is added in TypeScript because there is no public IPC/UI contract in this milestone. No SQL migration is needed because fixtures are returned as canonical bytes and tests keep them in controlled temporary storage; existing run/event persistence remains authoritative.

## Risks and controls

- **Secret leakage:** export applies the existing redactor again and refuses any changed, marked-redacted, or truncated supported payload. Digests can still reveal equality and heuristic redaction can miss secrets; fixtures require review before sharing.
- **Command execution:** raw command/output text is absent from the fixture and evaluator input. The replay package must not import supervisor, adapters, `os/exec`, HTTP, or networking.
- **False determinism:** the result describes fingerprint comparison for recorded command completions only. It does not reproduce commands, workspaces, provider calls, messages, tools, or model decisions.
- **Incomplete history:** damaged/partial histories, sequence gaps, missing payloads, unsupported events, and terminal inconsistencies refuse export.
- **Resource abuse:** fixture bytes, interactions, text before hashing, JSON shape, and evaluation are bounded; context cancellation is checked throughout.

## Verification plan

1. Run focused replay package tests with the race detector and Go vet.
2. Run `make check` for Go formatting/vet/race/build, frontend formatting/lint/type checks/tests/build, document links, and existing integrations.
3. Run `make native-check` with the established Rust/Cargo cache environment to prove bridge/build compatibility.
4. Confirm no schema migration, IPC method, dependency, frontend behavior, supervisor behavior, policy/admission path, or DMG artifact changed.
5. Record exact commands, passes, failures, skips, and unverified GUI/security claims in the separate Phase 5A report.

## Stop boundary and next milestone

Stop after the backend fixture/export/validation/evaluation foundation and documentation. A later separately authorized Phase 5 milestone can design explicit controlled transport recording and a private IPC/UI workflow after reviewing fixture storage, consent, export destinations, and offline enforcement. Live AI rerun remains a distinct non-replay action.
