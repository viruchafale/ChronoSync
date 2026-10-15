# Phase 02 — Event storage and timeline

## 1. Objective and user-visible outcome

Reopen a recorded run after restart and filter/inspect a durable ordered timeline.

## 2. Problem

P1 staging journals cannot provide transactional queries, reliable recovery or schema evolution.

## 3. Scope and non-goals

SQLite migrations, ingestion/retrieval, integrity, minimal durable payload objects and crash recovery. No branch restoration or complete CAS GC UI.

## 4. Prerequisites and dependencies

P1 event contracts and captures; SQLite driver portability experiment; writable private app-data store.

## 5. Design and component boundaries

Single-writer repository port owns WAL transactions; minimal object writer durably publishes payloads before references; UI uses paginated query DTOs.

## 6. Ordered implementation tasks

1. Benchmark/select SQLite driver in an ADR; define migrations and refuse newer schemas.
2. Implement app-data writer lock, schema version table, WAL/foreign keys and backup strategy.
3. Implement bounded staging importer with idempotent event IDs and atomic payload-object publication.
4. Commit event sequence and run state together, then notify UI; test storage failure cancellation.
5. Add indexed pagination/filter/detail APIs and accessible timeline/error views.
6. Recover interrupted runs and reconcile temp/orphan objects without automatic model resume.
7. Run crash/migration/corruption and 100k-event benchmarks; record reference hardware.

## 7. Expected files/modules

Actual: engine/internal/storage/{store,query,store_test}.go and legacy/, model/recorder/ipc/redact/contract, engine/go.mod/go.sum, contracts/, existing desktop App/api and Rust host/bridge, process integration script. Migration 1 is embedded alongside the store instead of a separate migrations directory.

## 8. Interfaces and contracts

Run/Event tables, page cursor containing ingest sequence, schema_migrations checksums, payload references and structured integrity errors.

## 9. Security and failure considerations

Prevent SQL injection with parameters; bounded pages; refuse unknown future schemas; disk-full must cancel capture. Crash recovery must not invent success.

## 10. Unit, integration and end-to-end tests

Unit: cursor/invariant/migration validation. Integration: kill at blob/DB commit boundaries, duplicate imports, WAL reopen, disk-full and corrupt objects. E2E: record → restart → filter/detail; missing payload visible.

## 11. Acceptance criteria

Committed events survive crash with no duplicate IDs or sequence gaps; unsupported schema rejected; every payload verifies hash; 100-event page under 200 ms for 100k events on recorded reference hardware.

## 12. Validation commands

```sh
make check
npm run desktop:build
```

These are existing baseline commands. Add phase-specific test suites to these targets and an explicit opt-in live or platform test command where needed before claiming this phase passes. Baseline success alone does not satisfy the phase acceptance criteria.

## 13. Risks and unresolved questions

Driver portability, fsync semantics across filesystems, retained private data size and WAL contention.

## 14. Exit criteria

Automated migration/recovery/query/benchmark targets have evidence. Native GUI retrieval acceptance remains blocked; real disk exhaustion and power-loss fault cases remain explicitly unverified.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`BLOCKED` — implementation and automated acceptance delivered; native GUI E2E remains unverified.

Explicitly authorized on 2026-10-22. See [actual results](../IMPLEMENTATION_STATUS_PHASE_02.md) and [ADR-0003](../decisions/ADR-0003-durable-timeline.md).

Task evidence:

1. Done: pinned driver, portability build and measured 100k workload; no claim of comparative driver benchmarking.
2. Done: process lease, migration 1/checksum validation, WAL/FULL/foreign keys, documented stopped-directory backup procedure. Automated backup/future-schema viewer deferred; unknown schemas refuse opening.
3. Done: bounded legacy reader (32 MiB/10k events per legacy run), transaction per imported run, source preservation and idempotent imports; private durable payload publication.
4. Done: atomic lifecycle/sequence, storage errors propagate through the existing recorder sink; rollback, aborted transaction and cancellation tests pass.
5. Implemented and component/IPC/native-bridge tested: typed queries, search/type/range filtering, 100-row pages and lazy details. Actual native-window walkthrough remains BLOCKED.
6. Done for safe recovery: interruption annotations, integrity/reference checks and retained crash evidence. Temp/orphan objects are deliberately retained; reconciliation/GC deferred rather than deleting possible evidence.
7. Passed synthetic abrupt-exit, migration/checksum/SQL-failure/missing-object tests and 100k measurements. Actual power loss/disk-full/fsync matrix remains unverified.

Scope clarification: a complete CAS, automatic repair, snapshot support and live database backup are not Phase 2 deliverables. No Phase 3 work started.
