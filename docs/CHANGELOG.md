# Changelog

## 2026-10-25 — Submission-build macOS DMG

Added a repeatable Tauri DMG command and macOS release guide. Produced and verified a real ARM64 DMG, inspected its app and bundled engine from a read-only mount, and launch-verified the native UI plus packaged-engine history handshake. The artifact remains a local unsigned `0.0.0` build; Developer ID signing, notarization, stapling, clean-machine Gatekeeper validation and final version selection remain distribution blockers. No runtime feature, authentication, admission, sandbox, commit, push or upload.

## 2026-10-23 — Phase 4G isolated protocol experiment

Added a standard-library-only standalone challenge model and race tests, with fixed synthetic manifest, bounded canonical frames, session/expiry/replay checks and no production integration. Inspected actual local code-signing metadata and recorded native binding/transport/packaging approval gates. No native authentication or peer identity verification is claimed. Historical reports preserved.

## 2026-10-23 — Phase 4F authentication feasibility

Added Apple-sourced macOS mechanism comparison, threat/authority analysis, native ownership and challenge protocol proposals, authentication provenance schema proposal and layered test matrix. Compile-only public SDK check performed without authentication calls. Production mechanism remains unselected pending native evidence and explicit factor/trust decisions. Existing Phase 4E code and historical reports preserved; no runtime integration, migration or public command.

## 2026-10-23 — Phase 4E trust boundary

Added closed engine-owned capability-provider registration, identity/target/scope/generation validation and conservative supervisor assessment. Hardened Go IPC envelopes/run.start against unsupported authority fields and ambiguous duplicate keys. Added provider, IPC and durable approval-reference tests; documented verified authority inventory and security decisions. User requires OS-backed authentication; trusted approval handler remains blocked. No positive launch, migration, public command, UI/backend or new dependency.

## 2026-10-23 — Phase 4D durable admission repository

- Added migration 3 for admission aggregates, immutable operation/audit results and unique challenge/approval/claim/revocation references; prior migrations/history preserved.
- Added transactional revision/one-use claims, exact retry receipts, corruption validation through domain replay and fresh-session conservative recovery.
- Added concurrency, injected-failure, migration and stopped-restore tests. No execution integration, authenticated approval, private-key mechanism, UI or isolation backend.
- Recorded separate Phase 4D evidence; preserved Phase 4A–4C and outstanding Phase 1–3 GUI gates. Older binaries refuse schema 3; rollback requires a stopped complete backup and loses later writes.

## 2026-10-23 — Phase 4C internal admission domain

- Added immutable internal execution-binding values with explicit version/domain, bounded canonical encoding, SHA-256 and independent fixed vectors.
- Added external challenge/approval/evidence references and pure epoch/order/state transitions with permanently consumed claims and explicit unknown outcomes.
- Kept authentication, secret commitments, durable order/claims and backend identity as unresolved external responsibilities. No policy/supervisor, wire/schema/migration, dependency, UI or launch behavior changes.
- Added a separate Phase 4C status report and retained Phase 4B and Phase 1–3 acceptance evidence.

## 2026-10-22 — Phase 4B durable admission and launch-binding design

- Audited actual policy/supervisor, adapter version/execution, consent/host/UI and SQLite recovery boundaries. Preserved Phase 4A refusal and legacy behavior unchanged.
- Proposed one-use durable claim plus separate launch observation, explicit uncertain crash outcomes, revocation ordering, exact private launch binding and engine-owned provider/approval authority.
- Added ADR-0007, concrete relational migration proposal, failure matrix and separately approvable implementation/test milestones. No migration or runtime code changed.
- Recorded documentation validation in the separate Phase 4B status report; runtime suites were not rerun and Phase 1–3 native GUI gates remain open.

## 2026-10-22 — Phase 4A execution-policy core

- Added deterministic internal Go policy types, mandatory restriction validation, exact capability evidence matching and actionable refusal codes; conservative provider reports unknown without host probes.
- Added opt-in supervisor policy admission that always refuses before process setup. Existing legacy Codex launches remain unchanged and are not claimed policy-verified.
- Added policy and pre-spawn refusal tests; retained existing execution safeguards. No positive enforcing backend, UI claim, wire/schema/migration change or durable admission history.
- Recorded scope and verification separately in IMPLEMENTATION_STATUS_PHASE_04A.md; Phase 1–3 GUI gates remain open.

## 2026-10-22 — Separate implementation status reports

- Split Phase 1–4 evidence into individual Markdown files and retained IMPLEMENTATION_STATUS.md as the status index.
- Recorded the required per-phase/per-milestone status-file convention in AGENTS.md and updated phase-plan links. Historical results and outstanding acceptance gates are preserved; no runtime changes.

## 2026-10-22 — Phase 4 architecture and execution-policy design only

- Documented the expanded multi-agent debugging/evaluation vision and audited existing recorder, timeline, snapshot and host/UI boundaries.
- Proposed workflow catalog, provenance relationships, unchanged event-v2 correlation, incremental persistence and immutable evaluation evidence in ADR-0005.
- Defined requested/capability/effective/observed policy separation, explicit refusal and conditional platform investigation matrix in ADR-0006 and the security model. No backend chosen or installed.
- Replaced Phase 4's implementation scope with the authorized design milestone, ten-decision register and separate approval gates. Marked conflicting bootstrap proposals historical/superseded.
- Passed offline portable/native baseline checks and final documentation checks; preserved source/contract hashes and existing Phase 1–3 native GUI acceptance gaps. No runtime, migration, schema, dependency, live integration, replay, GC or later-phase implementation.

## 2026-10-21 — Phase 0 foundation

- Audited empty non-Git directory and installed Go/Node/Codex/Apple tools; missing Rust recorded.
- Selected Tauri 2/React/strict TypeScript, Go, private stdio sidecar design, SQLite metadata, immutable SHA-256 blobs and supervised Codex exec JSONL.
- Added version-only engine and static desktop shell with explicit unimplemented-feature status, contracts, checks and CI configuration.
- Added full documentation and eight phase plans; moved minimum redaction/supervision into P1 and atomic payload publication into P2 as prerequisite work.
- Recorded npm sandbox DNS failure, dependency audit remediation and Go cache restriction; final verification evidence resides in IMPLEMENTATION_STATUS.md.
- No live agent run, commit, push, publication or Phase 1 implementation.

- Completed portable verification: Go race/vet/build, 12 contract tests, strict TS/lint/format/frontend build, documentation and executable/schema integration checks. Upgraded Vitest/ESLint after audit; updated installation reports zero vulnerabilities.
- Reserved event wire types/schema without implementing capture; added generated native icon assets.
- Marked Phase 0 BLOCKED on missing Cargo/native compilation and smoke-test evidence. Phase 1 remains NOT_STARTED.

## 2026-10-21 — Phase 1 execution recorder

- Implemented provider-independent run/event v2 contracts, Codex exec JSONL adapter, bounded POSIX supervisor, redaction, lifecycle and local ordering.
- Added private checksummed/fsynced staging journals, restart recovery, bounded corruption warnings and isolation of damaged histories; no competing database or Phase 2 implementation.
- Added os.Root-based bounded metadata observation with exclusions and symlink skipping; distinct vendor-reported and independently observed changes.
- Connected React consent/start/status/cancel/history/details to the actual Go engine through a packaged-resource Tauri stdio bridge. Added native sidecar build and unsigned macOS bundle command.
- Added Go parser/state/supervisor/storage/workspace/IPC/recorder tests, four UI tests, cross-language process integration and a real Rust-to-Go cancellation/restart test.
- Resolved missing native compiler prerequisite with temporary official Rust toolchain; generated Cargo.lock. Portable checks, native checks/build and two live public-fixture Codex recordings passed. No Git commit was made.
- Preserved a failed live initialization attempt honestly; approved outer-sandbox retry passed with Codex policy intact. Live read-only run captured command output; live write run observed generated.txt.
- Marked Phase 1 BLOCKED on native GUI E2E after automation timed out. No claim of verified GUI, Windows recording, sandbox security, deterministic replay or later-phase capabilities. Phase 2 not started.

## 2026-10-22 — Phase 2 durable storage and timeline

- Added pinned pure-Go SQLite driver, transactional schema migration/checksum ledger, private WAL store and bounded writer admission/query deadlines.
- Retained the journal implementation/tests solely for non-destructive legacy import. Added atomic event/lifecycle persistence, external payload integrity, recovery annotations and explicit structured errors.
- Kept event v2; added optional truncation/run history metadata and timeline-v1 capability. Added typed run/event keyset queries, supported diagnostic search, filters and lazy details through Go/stdio/Rust/React.
- Changed the existing timeline to one bounded page with next/previous/filter/search and unavailable/truncated states. Added storage crash/rollback/concurrency/import tests and UI/process/native-bridge tests.
- Recorded 100k-event performance and passed portable/native checks. Native GUI acceptance remains blocked; real disk-full and power-loss cases remain unverified. No Phase 3 implementation, live model test, commit or publication.

## 2026-10-22 — Phase 3 snapshots and branches

- Added manifest v1 portability policy and validation before capture/restore, exact-byte immutable object storage and migration 2 with snapshot/branch/reference metadata. Migration 1 remains unchanged.
- Implemented consented, idle-workspace two-pass capture, bounded directory reads, explicit exclusions/link omissions and failure on changed/unreadable/unsupported content.
- Added verified restore into fresh owned directories, logical CoW branch ancestry and later snapshots, preserving original events and source workspaces. No writable hardlinks to CAS.
- Added interrupted-operation recovery and retained temp/orphan diagnostics; no destructive GC or arbitrary destination cleanup.
- Connected typed Go/Rust/TypeScript operations and minimal snapshot controls; added manifest/object/migration/capture/restore/branch/recovery/concurrency/IPC/native-bridge/UI tests and three measured workloads.
- Portable/native checks pass; native-window acceptance and real filesystem fault guarantees remain unverified. No Phase 4, live model call, commit or publication.
