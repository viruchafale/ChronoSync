# ADR-0009: Inert fingerprint replay fixture foundation

Status: accepted for the authorized Phase 5A backend milestone, 2026-10-28.

## Context

ChronoSync event v2 stores observable, redacted Codex lifecycle records. A command completion can contain command text, aggregated output and exit code, but history does not contain controlled provider request/response transport, model state, complete tool arguments/results, network behavior or a proof that a later workspace snapshot represents every event. Re-executing a recorded command would treat untrusted historical data as instructions and violate the replay boundary.

## Decision

Phase 5A defines a canonical JSON fixture containing only ordered `command.completed` interactions. Raw command, item ID and output are replaced with SHA-256 fingerprints; exit code remains an integer expectation. The source run ID, event version/count/high-water, adapter, redaction profile and supported type are explicit. The deterministic replay ID derives from the fixture body, and an outer SHA-256 covers canonical fixture JSON.

Export requires a completed run with complete, consecutive, integrity-checked history. It tolerates only the listed lifecycle boundaries and rejects all other event kinds. Supported payloads are passed through the existing redactor again and rejected if values change, contain redaction/truncation markers, lack required fields or carry truncation metadata. This avoids inventing unavailable bytes and prevents known redacted values from becoming matching inputs.

The evaluator receives only command/item fingerprints and source sequence. A controlled implementation returns an output fingerprint and exit code. The runner reports completed/pass or completed/mismatched results, injected failure, cancellation or unsupported interaction. Only a completed matching result can return detached canonical bytes as a reusable regression fixture; the caller owns any later private file destination.

The fixture remains outside SQLite and public IPC. No migration, renderer command, supervisor import, provider adapter, network client, subprocess, snapshot restore, Phase 4 admission/policy input, dependency or DMG change is introduced.

## Consequences

This milestone validates stable export/import and regression comparison mechanics for an inert controlled boundary. It does not replay a command, reproduce a workspace, call an AI, recreate messages/tool calls, certify offline enforcement or claim deterministic LLM behavior. SHA-256 fingerprints reveal equality and can permit guessing of low-entropy values. Existing redaction is heuristic, so fixtures require review before sharing even though supported raw text is absent.

A later milestone needs separately authorized controlled transport recording, consent/export destinations and enforced offline execution before expanding supported interactions or adding UI/IPC. Live AI rerun remains a new execution, never replay fallback.
