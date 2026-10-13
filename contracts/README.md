# Contract status

Implemented engine protocol v1 advertises version, stdio-recorder-v1 event-v2 and timeline-v1. `version.schema.json` describes the compatibility report. The active recorder schema is `recorded-event.schema.json` with TypeScript in `recorder.ts` and Go in `internal/model`. Old event.schema.json/event.ts/EventEnvelope are historical proposed v1 and remain tested for compatibility reference; they are never emitted by the recorder.

Actual methods: hello, run.start, run.list, run.cancel, event.list. Each JSONL request has protocol_version=1, nonempty string id, method and params. A response echoes id/version and supplies result or structured error. Handshake is required. Invalid requests fail without starting an agent. Requests are handled serially; UI polls event pages, so no event notifications or pending-64 queue is implemented. run.list returns the newest 100 summaries. event.list accepts {id,after:decimal-string} and returns {run,events,warnings,next}; pages cap at 100 events/512 KiB event content. IDs/sequence cursors are validated in Go. Frame limit is 1 MiB. Cancellation acknowledgement is separate from terminal status. Errors currently use PROTOCOL_ERROR or REQUEST_FAILED; richer codes below remain proposed.

Run start requires workspace, prompt, codex_path, sandbox, timeout_seconds, consent and optional excludes. Child execution/path selection is explicit user consent, not a generic host shell API. See DATA_MODEL and ADR-0002 for payload bounds and recovery.

---

## Archived Phase 0 design (not an implementation claim)

# Contracts

`version.schema.json`, `version.ts`, and the Go `contract.Version` describe the implemented `chronosync-engine version` output. Ajv checks the real executable output in `npm run test:integration`. The CLI rejects recording commands. These types are hand-maintained while the surface is tiny; add generated bindings before expanding IPC.

## Proposed IPC v1 (not implemented)

UTF-8 JSON objects, one per LF line, maximum 1 MiB encoded frame, no batches. Oversized or malformed frames terminate the connection with a diagnostic. Large data travels by object ID and paginated reads, never arbitrary filesystem paths. Requests: `{protocol_version:1,id:string,method:string,params:object}`. Responses echo id/version and contain exactly one of result or `{code,message,retryable}` error. Notifications use `{protocol_version:1,method:"event",params:Event}` without id. IDs are opaque unique strings scoped to connection. A pending-request cap of 64 and bounded notification queue are initial limits to test.

Handshake `hello` exchanges protocol major, engine build, and capabilities before accepting methods. Major mismatch rejects connection; additive optional fields require capability negotiation. Implemented methods: `run.start`, `run.cancel`, `run.list`, `run.get`, `run.query`, `event.list`, `event.query`, `event.get`. The new query/detail methods require timeline-v1. Unknown method → UNSUPPORTED; bad params → INVALID_ARGUMENT; denied policy → PERMISSION_DENIED; corrupt blob → INTEGRITY_ERROR; store failure → STORAGE_ERROR; overload → RESOURCE_EXHAUSTED. No error embeds secrets or arbitrary raw upstream text.

Host validates messages before delivering to UI. Engine validates again at the trust boundary. Cancellation is idempotent, run-scoped, and acknowledged separately from confirmed process exit. Connection EOF triggers engine cancellation and bounded cleanup; it never starts a replacement run automatically. Dead engine invalidates pending requests, and the host exposes interrupted state. No UI caller can choose engine path, arbitrary shell argv, or arbitrary storage location.

## Proposed event envelope

`schema_version:1`, `event_id`, `run_id`, `producer_id`, `producer_seq` (decimal string), `ingest_seq` (decimal string), `lamport` (decimal string), `observed_at` (UTC RFC3339), `kind`, `causes` (event ID array), `payload_ref` (nullable SHA-256 object ID), `redaction_version`, `adapter_type`, `adapter_version`. Decimal strings avoid JS integer precision loss. `ingest_seq` is local durable presentation order, not causal proof. Unknown vendor types map to `adapter.unknown` with sanitized payload. See DATA_MODEL.md for invariants and planned schema-generation gates.

`event.schema.json`, `event.ts`, and Go `EventEnvelope` reserve the proposed v1 shape. They do not implement recording or semantic validation. Before ingestion, P1 must validate real timestamps, signed 64-bit counter bounds, IDs, causal references and payload integrity in addition to this structural schema. Contract samples are synthetic and never represent a live capture.

## Timeline v1

Typed queries/results are in recorder.ts and Go storage/query.go. Event wire schema stays v2 with optional truncation metadata; run-v1 gains optional start/history/recovery/recorder fields. Event pages contain summaries rather than fake empty payloads; event.get returns the verified full event. Limits are 1..100, event cursor strings are opaque and scoped to ID/filters/high water. Type and inclusive sequence bounds are optional; diagnostic search is a literal ASCII-case-insensitive substring in the first 1024 indexed bytes. Counts are total recorded events through high water. See [data model](../docs/DATA_MODEL.md) for full contracts and actual structured error codes (the initial proposed names above are historical).

## Snapshots v1

Engine 0.3.0 advertises snapshots-v1 alongside existing recorder/timeline capabilities under protocol 1. Event v2 is unchanged. Types live in snapshot.ts and Go internal/snapshot. Methods: snapshot.capture/list/get/validate/restore, branch.create/list/get and object.inventory. Capture accepts run_id, optional event_id/branch_id, excludes and consent; workspace is derived from the registered source. Materialization accepts snapshot_id, optional parent_id and consent, never a destination path. List requests use run_id/after/limit; empty next ends the page. Native commands are explicitly typed and allowlisted. Validation verifies actual manifest/file objects; get retrieves metadata only. Inventory reports retained object/temp/unreferenced counts, never deletes. Snapshot errors use explicit operation codes without raw source contents. Both native-window UI and filesystem power-loss guarantees remain unverified.
