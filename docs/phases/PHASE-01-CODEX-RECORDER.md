# Phase 01 — Codex execution recorder

## 1. Objective and user-visible outcome

Explicitly start one approved Codex execution in an owned disposable workspace and inspect its real observable events and terminal result.

## 2. Problem

Terminal text alone is insufficient structured evidence; child processes can hang, leak secrets or outlive the UI.

## 3. Scope and non-goals

One exec JSONL workflow, normalization, IDs, bounded staging capture, consent, host IPC, limits/cancel and before/after change manifest. No durable SQLite timeline, generic network interception or replay.

## 4. Prerequisites and dependencies

Phase 1 authorized by the user. Missing Rust prerequisite addressed with temporary official toolchain; native build passes. Codex 0.163.0 authenticated and live disposable Git tests pass. GUI smoke remains blocked.

## 5. Design and component boundaries

Adapter emits vendor records into a bounded EventSink; supervisor owns the process tree; engine normalizes; Rust owns sidecar lifecycle; UI inspects honest captured records.

## 6. Ordered implementation tasks

1. Define versioned adapter ports and vendor compatibility/provenance fixtures; implement bounded incremental JSONL parser.
2. Test malformed/unknown/partial records, huge lines, separate stderr and schema drift.
3. Implement owned workspace/consent validation, env allowlist and pre-storage redaction.
4. Add supervisor with deadline, bounded bytes, context cancellation and POSIX groups/Windows jobs; gate unsupported platforms.
5. Implement run state machine, unique IDs, staging journal and terminal manifest, including zero-change/changed workspace evidence.
6. Implement negotiated stdio host bridge and target-specific Go sidecar packaging; test EOF/crash/backpressure before any launch control.
7. Add accessible start/inspect/cancel flow with exact capabilities and policy disclosure.
8. With separate user consent run the read-only live milestone in CODEX_INTEGRATION.md, then an approved write/change test.
9. Record observed event coverage and sanitized evidence; fail honestly on missing commands or auth errors.

## 7. Expected files/modules

engine/internal/{adapter,adapter/codex,model,recorder,storage,redact,supervisor,ipc,workspace}, engine/cmd/, apps/desktop/src-tauri/, apps/desktop/src/, contracts/, tests/fixtures/codex/.

## 8. Interfaces and contracts

Implemented StartSpec, adapter Sink/Result, protocol capabilities; hello/run.start/run.list/run.cancel/event.list; event schema v2 (typed inline payload and local sequence), separate OS exit/adapter outcome. See ADR-0002 for the justified departure from proposed v1.

## 9. Security and failure considerations

No shell interpolation, inherited unrestricted config or bypass flags. Read-only policy is upstream-dependent. Sink failure cancels execution; no silent truncation. Deadlines are not proof of deadlock.

## 10. Unit, integration and end-to-end tests

Unit: parser/state transitions/redaction. Integration: actual stub child executable for output flood, grandchildren, cancel, EOF, nonzero exit and sidecar packaging. E2E: explicitly authorized real Codex read-only and write captures; keyboard inspection/cancel; auth failure must not be a pass.

## 11. Acceptance criteria

Unique events preserve emitted vendor types and unknowns; no secrets in fixtures; child-tree cleanup meets documented grace; host loss stops run; real command/output and changed-file evidence are inspectable; both success and failure manifests are honest.

## 12. Validation commands

```sh
make check
make native-check
npm run desktop:bundle:macos
CHRONOSYNC_LIVE_CODEX=/absolute/path/to/codex npm run test:live -- --consent
CHRONOSYNC_LIVE_CODEX=/absolute/path/to/codex npm run test:live -- --consent --write-fixture
```

These commands now exist. make check and native-check passed using the sandbox toolchain/cache environment in IMPLEMENTATION_STATUS.md; live read-only and write-fixture commands passed with approved execution outside the outer sandbox. GUI test automation did not complete.

## 13. Risks and unresolved questions

Vendor shape drift, model access, incomplete command/file observability, Windows containment and bundled sidecar lookup. Add opt-in live test harness before claiming integration.

## 14. Exit criteria

Parser/supervisor/host tests pass and approved live capture plus cancellation evidence is documented; phase does not exit on synthetic fixtures alone.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`BLOCKED`

Implementation and automated/native compilation checks finished. Blocked on actual native GUI acceptance evidence; no approved exception exists. Phase 2 remains NOT_STARTED.

## Implementation and acceptance evidence

| Task / criterion                               | Status and evidence                                                                                                                                                       |
| ---------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Tool audit, supported integration and contract | DONE: CLI 0.163.0 help, official docs, typed adapter/event v2 and ADR-0002.                                                                                               |
| Parser and event normalization                 | DONE: documented mappings, synthetic fixtures, unknown omission, malformed warning/failure, actual live command output.                                                   |
| Lifecycle/IDs/order                            | DONE: explicit state transition tests, random IDs, monotonic local decimal sequences and typed terminal reasons.                                                          |
| Persistence/restart/partial history            | DONE for bounded P1 journal: checksums, private lock/permissions, torn/malformed handling, damaged-run isolation and real restart tests. No SQLite migration exists.      |
| Consent/environment/redaction                  | DONE within documented limits; no shell interpolation, prompt omitted, bounded heuristic redaction and tests.                                                             |
| Timeouts/cancellation                          | DONE on macOS using real synthetic processes and ordinary grandchild cleanup; Rust bridge confirms cancel and reopened state. Live-model cancellation remains unverified. |
| Workspace changes                              | DONE: create/modify/delete/exclusion/symlink tests plus live generated.txt creation in disposable workspace. No file bodies/snapshots.                                    |
| Go, frontend and native builds                 | PASSED: make check and make native-check; 16 TS/UI tests; native Rust bridge test and release build.                                                                      |
| Supported live execution                       | PASSED: read-only run 801419b60519f32eee535f9253041c6d and write run a7a6cc2ed9076936e09f694f4b86e28d. Failed sandbox initialization also retained truthful history.      |
| Desktop start/inspect/cancel behavior          | Component behavior and actual Rust/Go bridge tests PASSED; real native-window E2E BLOCKED by automation timeout.                                                          |
| Platform cleanup/security                      | macOS ordinary group verified. Linux cross-compiles but runtime unverified; Windows recorder blocked. No sandbox or detached-process guarantee.                           |
| Documentation                                  | Updated README, contracts, architecture/system/data/security/testing/Codex docs, status/changelog and this plan.                                                          |

Supported kinds: execution.state/diagnostic/warning; agent.thread/turn_started/turn_completed/turn_failed/message/reasoning_summary; command.started/updated/completed; tool.started/updated/completed; file.reported; workspace.changed; adapter.unknown/warning/error. Only emitted evidence and explicit ChronoSync observations are recorded. Tool arguments/results and unknown raw bodies are omitted; no hidden reasoning or network capture claim.

Known limitations: heuristic redaction, selected workspace edited directly in workspace-write, no hard process resource sandbox, no guaranteed cleanup on SIGKILL or detached sessions, staging scans rather than indexed SQLite, 100 newest run summaries, no full history retention UI, partial/bounded output. See status for failed attempts and fixes. No known outstanding failing automated check remains; native GUI behavior is unverified, not assumed correct.

Exact validation commands/results and toolchain environment are in ../IMPLEMENTATION_STATUS.md. Next task is the native-window acceptance sequence there. Do not mark COMPLETE or start Phase 2 based only on the passing component/bridge tests.
