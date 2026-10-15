# Phase 2 implementation status

Implemented; native GUI acceptance remains unverified/blocked.

Evidence and limitations below describe this phase at the time recorded. Later phases do not retroactively accept its outstanding gates.

[Status index](IMPLEMENTATION_STATUS.md)

As of 2026-10-22. **Phase 2: IMPLEMENTED; acceptance BLOCKED on native GUI end-to-end verification.** The user explicitly authorized Phase 2. Phase 1's automated/live recorder evidence was reviewed and its portable/native baseline rerun successfully before changes. Its earlier native GUI gap remains open. Phase 3–7 remain NOT_STARTED; none are authorized by this implementation.

## Phase 2 delivered

- SQLite metadata, events, lifecycle and migration checksum ledger; pinned pure-Go modernc.org/sqlite v1.61.0, observed SQLite 3.54.1. Go Repository/Inspector ports separate recorder and IPC from persistence.
- Private exclusive owner, WAL/FULL/foreign keys, bounded writer admission and transaction/query timeouts. Atomic sequence + lifecycle commits; metadata updates cannot manufacture success.
- Event v2 remains the wire model. Added optional structured truncation and run start/history/recovery/recorder metadata. Engine 0.2.0 advertises timeline-v1 under protocol 1.
- Non-destructive, transactional legacy journal import; original journal corruption tests preserved in storage/legacy. Source journals and orphan payload files are retained.
- SHA-256/length-verified external event files above 8 KiB; 256 KiB maximum event. Startup SQLite/relationship/payload validation and idempotent interruption recovery. Damaged history never projects completed status.
- Typed run/event query and detail methods, type/sequence filters, literal diagnostic search, scoped high-water keyset event cursors and keyset run pages. Compatibility event.list also uses bounded SQL retrieval.
- Existing React timeline now retains one page (100 summaries maximum), with filters/search, next/previous, status/times/count, lazy detail, truncation, unavailable payload, loading/empty/error handling. No historical command execution or replay.

## Verification evidence

Baseline before edits: `GOCACHE=/tmp/chronosync-go-cache make check` and the temporary-toolchain `make native-check` both exited 0; 16 existing Vitest tests and real Rust bridge test passed. Final Phase 2 commands use `GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod` in this restricted environment.

| Command                                                                                                                                                     | Actual result                                                                                                                                                                                                                                                          |
| ----------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `make check` with the cache variables above                                                                                                                 | Passed: Go fmt/vet/race/build, Prettier/ESLint/strict TS, 19 Vitest tests, frontend build, docs, version/schema process test, recorder/timeline process integration. Final rerun passed; synthetic recorder/timeline integration run 9200f185e4bb7c90e2c6a4fc4f629948. |
| `PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo` plus the cache variables and `make native-check` | Passed Rust fmt, locked Clippy, real bridge cancellation/restart/query/detail test, native macOS release build.                                                                                                                                                        |
| `CHRONOSYNC_LOAD=1 go test ./internal/storage -run 'TestLoad100k\|TestConcurrentProducersReaders' -v -count=1` (engine cwd, cache variables)                | Passed 100k history and concurrent producer/read measurements; details below.                                                                                                                                                                                          |
| `GOOS=linux GOARCH=amd64 go build -o /tmp/chronosync-engine-linux ./cmd/chronosync-engine` (engine cwd, cache variables)                                    | Passed; cross-compilation only, not Linux runtime.                                                                                                                                                                                                                     |
| `GOOS=windows GOARCH=amd64 go build -o /tmp/chronosync-engine-windows.exe ./cmd/chronosync-engine` (engine cwd, cache variables)                            | Passed cross-compilation; Windows recording remains intentionally blocked.                                                                                                                                                                                             |
| `npm run desktop:bundle:macos` (temporary toolchain/cache variables)                                                                                        | Passed; refreshed local unsigned Phase 2 ChronoSync.app, approximately 20.39 MiB. Artifact creation is not GUI verification.                                                                                                                                           |
| `go mod tidy` (engine cwd, cache variables)                                                                                                                 | Passed after approved dependency download; initial restricted DNS attempt failed.                                                                                                                                                                                      |

Storage tests cover repeatable migration/open and checksum/future-version rejection, private modes/lease, legacy source preservation/import idempotency, constraints, literal search and invalid cursors, stable high-water pagination during appends, lifecycle atomicity, batch rollback, injected SQL failure, cancellation, missing external payloads, abrupt subprocess exit with a committed and uncommitted transaction, and concurrent producers/readers. Existing recorder shutdown/cancellation tests run against SQLite. An added injected-storage-error test confirms an actual synthetic CLI and its sleeping child are stopped promptly, preserves sequence 3 and marks storage_failure interruption. No live model calls were added or executed for Phase 2.

## Measured workload

Darwin 25.6.0 arm64, Go 1.26.2, SQLite 3.54.1; temporary local store. CPU model query was denied by the sandbox, so the exact CPU/RAM are unverified. The measured workload uses 100,000 synthetic agent.message events with short fixed diagnostic text, random IDs, UTC timestamps, inline payloads and no private content. First 100 writes are individual FULL transactions; remaining events use batches of 100. This is not a large-output/blob benchmark. The second measurement overlapped native/cross builds, so it includes machine contention.

- Total insertion: **7,646 ms**; individual-write p50 **132 µs**, p95 **878 µs** (first 100 samples).
- First 100-summary page, including actual count of 100k events: **106,136 µs** (106.1 ms); **61,776 bytes** allocated according to Go TotalAlloc delta. This is allocations, not peak RSS. Below the plan's 200 ms target on this measured run.
- Checkpointed DB: **6,774,784 bytes at 10k**, **67,653,632 bytes at 100k**. WAL/SHM and transient files are not included in these checkpointed sizes.
- Full integrity scan plus interrupted recovery: **2,117 ms**.
- Four producers ×25 writes with explicit conflict/busy retry and 50 concurrent page reads: **67.2 ms**, **94 retries**, 100 unique consecutive persisted events.
- An earlier baseline load before added checks measured 5,602 ms insertion, 40,466 µs page, 60,496 allocated bytes and 2,046 ms recovery. It is retained as development evidence, not substituted for the later result.

## Failures resolved and limitations

Dependency fetch/tidy initially failed restricted DNS, then succeeded with approved network execution. One new UI deferred-promise test initially inferred Promise<unknown>; explicit TimelinePage typing fixed strict TS. A Rust fmt attempt omitted the temporary RUSTUP_HOME and was denied; rerun with documented environment passed. Initial targeted Go commands from the repository root failed to locate the module; corrected to engine. No tests were removed or weakened: the original four journal tests moved with their format-specific implementation to legacy.

Native GUI walkthrough remains unverified after Phase 1 automation timed out; it was not relabeled as passed. Native Linux/Windows runtime, power loss, real disk exhaustion and a full fsync failure matrix are unverified. Synthetic SQL failure/abrupt exit are narrower evidence. There is no encryption, complete redaction, snapshot, branch, replay, full CAS/GC, automated backup, or future-schema viewer. Retained orphan payloads can consume disk. Full recovery scans every event and can delay startup on much larger histories; the host handshake timeout is 15 seconds. The database must be local, not a network filesystem. Windows recording remains blocked.

This directory has no Git metadata; `git status` still reports not a repository. Source and tests were reviewed directly. Final review added a regression check that a pre-existing unsafe payload path is preserved and rejected instead of overwritten; targeted storage race tests and vet passed afterward, and the app bundle was rebuilt. No commit, push or publication occurred. The [Phase 1 status](IMPLEMENTATION_STATUS_PHASE_01.md) describes the previous version, not Phase 2 storage.

## Remaining acceptance and next task

Complete the native-window acceptance pass using a disposable synthetic project: start/cancel, close/reopen, filter/search/page, open command details, inspect interrupted/missing-payload states, and check keyboard navigation. Automated coverage is not evidence that this walkthrough occurred. Record outcomes before marking Phase 2 COMPLETE. Finish remaining fault tests as explicitly scoped follow-up work; do not claim power-loss durability from a process-exit test.

After Phase 2 acceptance and explicit Phase 3 authorization, the exact first task is to finalize the snapshot manifest's portable path/case/symlink/mode rules and exclusion policy, review them against the minimal event-object format, and add manifest-validation tests before implementing capture or restore.

---
