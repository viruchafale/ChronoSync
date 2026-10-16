# Phase 03 — Snapshots, CAS, and branching

## 1. Objective and user-visible outcome

Restore a supported checkpoint into an isolated owned workspace and branch without changing original history.

## 2. Problem

Git commits alone omit dirty/untracked state; copies waste storage and can mutate shared history accidentally.

## 3. Scope and non-goals

Immutable CAS, snapshot manifests, deduplication, restore, ancestry, logical CoW, conservative reference diagnostics and recovery. Destructive GC is deferred. No claim of whole-machine snapshot or deterministic rerun.

## 4. Prerequisites and dependencies

P2 durable store and object primitive; snapshot consistency/exclusion policy defined in ADR-0004. Git worktrees evaluated and not required for owned plain-directory materialization.

## 5. Design and component boundaries

Snapshot service references immutable file objects; branch service owns leases and ancestry; workspace service materializes isolated directories without modifying source HEAD/index.

## 6. Ordered implementation tasks

1. Finalize portable manifest/case/symlink/mode rules and explicit exclusions.
2. Quiesce supported writers, detect unstable files and publish only complete snapshots.
3. Implement object verification/dedup and manifest reachability.
4. Create branch with immutable parent/source-snapshot references and a distinct branch identity; do not invent a new execution.
5. Materialize into a new owned directory; verify hashes before atomic ready publication.
6. Implement ownership-aware restore/cleanup that refuses dirty unowned targets.
7. Add non-destructive reference/temp/orphan diagnostics; defer destructive GC until its safety can be established.
8. Expose ancestry, restore scope and incomplete-state errors in UI.

## 7. Expected files/modules

engine/internal/{objects,workspace,snapshot,branch}, engine/migrations/, contracts/, apps/desktop/src/branches/.

## 8. Interfaces and contracts

Snapshot manifest v1, object-format v1, Branch(parent_run,parent_checkpoint,snapshot_ref), ownership registry/leases.

## 9. Security and failure considerations

No writable hardlinks to CAS, no escaping symlinks, no cleanup of unrelated worktrees. Concurrent external writes invalidate snapshot consistency; never conceal exclusions.

## 10. Unit, integration and end-to-end tests

Unit: manifest canonicalization/ancestry cycle rejection. Integration: dirty/untracked capture, symlinks/case collisions, interrupted materialization, concurrent admission and retained-object inventory. E2E: capture → branch → edit → verify original unchanged → restore supported bytes/modes.

## 11. Acceptance criteria

Ancestor records/hashes unchanged after branch edits; duplicate blobs occupy one object; restored supported tree hashes match; interrupted branch is never ready; no cleanup deletes referenced objects.

## 12. Validation commands

```sh
make check
npm run desktop:build
```

These are existing baseline commands. Add phase-specific test suites to these targets and an explicit opt-in live or platform test command where needed before claiming this phase passes. Baseline success alone does not satisfy the phase acceptance criteria.

## 13. Risks and unresolved questions

Filesystem metadata portability, external writers, Git lock races, retention and disk pressure; filesystem reflinks optional.

## 14. Exit criteria

Local macOS automated branch isolation, restore equivalence, interruption recovery and retention checks pass. Native-window acceptance and other-platform runtime remain unverified.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`BLOCKED` — implementation and automated tests delivered; native GUI acceptance remains unverified.

Explicitly authorized by the Phase 3 request. Phase 2 native GUI remains unverified. Baseline portable/native checks passed. Manifest policy and publication protocol are specified in ADR-0004; manifest validation is implemented before capture/restore.

Task evidence: manifest rules/tests were added first; objects and no-replace/dedup tests followed; migration 2, capture/recovery, restore and branching then shipped through existing IPC/native/UI. Core integration, concurrency/failure tests and three performance workloads have measured evidence in [status](../IMPLEMENTATION_STATUS_PHASE_03.md). The first strict-TS fixture error was fixed without reducing coverage. Actual native-window and cross-platform runtime/power-loss checks are unverified. Original history is preserved; restore never overwrites a developer directory. Full GC, Git worktrees and automatic live execution on branches are not implemented. The user explicitly authorized these Phase 3 capabilities despite the prior GUI gate. No Phase 4 work started.

| Task                        | Actual status                                                                                         |
| --------------------------- | ----------------------------------------------------------------------------------------------------- |
| 1. Manifest rules           | Done; ADR-0004 and manifest tests preceded capture/restore.                                           |
| 2. Consistent capture       | Done for two-pass observed consistency; filesystem-atomic capture is not claimed.                     |
| 3. Objects/reachability     | Done; integrity/dedup/no-replace plus SQL reference rows.                                             |
| 4. Branch identity/ancestry | Done; new branch ID, source/parent records, no invented execution.                                    |
| 5. Materialize/verify       | Done; actual independent copies, sync/verification before ready.                                      |
| 6. Ownership/failure safety | Done by exclusive generated destinations and retained failed directories; destructive cleanup absent. |
| 7. GC/recovery              | Recovery/diagnostics done; destructive GC explicitly deferred.                                        |
| 8. UI/API                   | Implemented, component/IPC/native bridge tested; native-window acceptance unverified.                 |
