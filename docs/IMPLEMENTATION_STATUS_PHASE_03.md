# Phase 3 implementation status

Implemented; native GUI acceptance remains unverified/blocked.

Evidence and limitations below describe this phase at the time recorded. Later phases do not retroactively accept its outstanding gates.

[Status index](IMPLEMENTATION_STATUS.md)

## Phase 3 implementation

- New Go snapshot domain/ports, manifest validation, capture, restore and service modules; immutable object store; SQLite snapshot/branch repository and ordered migration 2. Migration 1's SQL/checksum is unchanged.
- Manifest v1/portable-ascii-v1: explicit relative paths and directory parents, sorted entries, collision/path/mode/reference limits, effective exclusions and omission reasons. ASCII-only names are a deliberate restriction; Unicode is rejected rather than normalized. Symlinks are explicitly omitted, special files rejected.
- Exact-byte SHA-256 CAS with verified deduplication, no-replace publication, private read-only objects, sync and integrity checks. Event payload v2 storage remains untouched. No destructive GC; retained orphan/temp counts are exposed.
- Explicit source-byte consent; captures only after the recorder is idle. Two bounded content/metadata scans detect observed changes. Source bytes are stored, not redacted into inaccurate restorations. This is not an atomic filesystem snapshot.
- preparing → committed/failed/damaged snapshots and preparing → ready/failed materializations. Objects precede transactional metadata/reference publication. Startup marks interrupted operations failed, verifies committed snapshots and preserves evidence.
- Restore/branch writes independent files into generated private managed-workspaces directories, never a user-selected overwrite destination. Verification and sync precede ready. Failed directories remain for inspection. Branch ancestry/source-run links and later branch snapshots share immutable objects without duplicating event history.
- Typed private-stdio and Rust commands plus small existing-UI snapshot controls: capture consent/exclusions, source selection, verify, restore, branch, status/ancestry and retained-object diagnostics. Pages remain bounded. Engine 0.3.0 adds snapshots-v1; event v2/protocol 1 remain unchanged.

## Verification and results

Pre-change baselines: portable make check and native make native-check exited 0, including 19 frontend tests and the real engine bridge test. All Phase 3 tests use synthetic projects and no live LLM calls.

Use `GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod` in this restricted environment. Native commands additionally use `PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo`.

| Actual command                                                                                                                   | Result                                                                                                                                                       |
| -------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `go -C engine test ./internal/snapshot` before capture/restore implementation                                                    | Passed manifest round-trip and invalid version/path/collision/hash/size/mode/reference-policy cases.                                                         |
| `make check` with cache variables                                                                                                | Passed Go fmt/vet/race/build; Prettier/ESLint/strict TS; 22 Vitest tests; frontend build; docs/schema integration; actual recorder/snapshot IPC integration. |
| `make native-check` with toolchain/cache variables                                                                               | Passed Rust fmt, locked Clippy/test, real Go bridge capture/branch/verify plus recorder cancellation/restart, native release build.                          |
| `GOOS=linux GOARCH=amd64 go build -o /tmp/chronosync-engine-linux ./cmd/chronosync-engine` (engine cwd, cache variables)         | Passed cross-build, not runtime verification.                                                                                                                |
| `GOOS=windows GOARCH=amd64 go build -o /tmp/chronosync-engine-windows.exe ./cmd/chronosync-engine` (engine cwd, cache variables) | Passed cross-build; Windows recording/storage/snapshots remain blocked.                                                                                      |
| `npm run desktop:bundle:macos` with toolchain/cache variables                                                                    | Passed; refreshed unsigned Phase 3 ChronoSync.app, 20.67 MiB. Not GUI verification.                                                                          |
| `go -C engine test -race ./internal/snapshot` with cache variables                                                               | Passed final added consent, active-source and parent-relationship boundary tests.                                                                            |
| `CHRONOSYNC_SNAPSHOT_LOAD=1 go test ./internal/snapshot -run TestSnapshotPerformance -v -count=1` (engine cwd, cache variables)  | Passed three measured workloads below.                                                                                                                       |

Additional tests cover actual immutable content/deduplication and concurrent identical puts; missing/corrupt objects; source changes between scans; empty files/directories/executable status; configured/default exclusions; broken/escaping/cyclic symlink omissions; FIFO rejection without opening; unreadable/oversized content; metadata migration from a true Phase 2 database; cancellation without falsely marking damage; interrupted preparing states; abrupt subprocess exit after object publication with an uncommitted snapshot; orphan/temp diagnostics; restoration of exact bytes; existing-destination/traversal/symlink rejection; failed partial materialization retention; concurrent branch admission; changed-content isolation and unchanged original run events.

Final process integration run c830c3aa3e68b334ff0939c3a1fc4d78 passed capture, verified restore bytes, parent/child branches, changed branch capture, original history preservation and reopening through actual IPC. Native bridge test separately passed snapshot capture/branch/validate. Component tests cover consent/exclusions, pending → ready and damaged/failure handling. These are not native-window GUI acceptance.

## Performance evidence

Darwin arm64, Go 1.26.2, local temporary APFS storage (`df -T apfs /private/tmp` identified /System/Volumes/Data). Exact CPU model/RAM remain unverified because the earlier hardware query was sandbox-denied. Files are synthetic; each case starts with an empty store. Capture includes two scans, object verification and SQL commit. Restore and branch include separate file writes/fsync, full verification and metadata commit. Validation checks the manifest and every content reference.

| Workload                          | Logical file bytes | CAS objects / bytes (including manifest) | Manifest bytes | Capture   | Restore     | Branch      | Verify   |
| --------------------------------- | ------------------ | ---------------------------------------- | -------------- | --------- | ----------- | ----------- | -------- |
| 10 × 1 KiB, 10 unique contents    | 10,240             | 11 / 11,757                              | 1,517          | 85.73 ms  | 41.99 ms    | 37.83 ms    | 1.14 ms  |
| 1,000 × 1 KiB, 10 unique contents | 1,024,000          | 11 / 153,327                             | 143,087        | 188.58 ms | 3,836.01 ms | 3,941.93 ms | 29.60 ms |
| 16 × 128 KiB, 16 unique contents  | 2,097,152          | 17 / 2,099,559                           | 2,407          | 139.46 ms | 62.55 ms    | 62.50 ms    | 1.82 ms  |

The duplicate-heavy workload stores 10,240 unique file bytes for 1,024,000 logical bytes (100:1 content deduplication); manifest overhead reduces total CAS savings to about 6.68:1. Two materializations still write independent copies. Unique-content cases gain no content deduplication. These are local measurements, not cross-platform guarantees. The earlier two-case development measurement is superseded by this three-case run after bounded traversal was added.

## Decisions, failed attempts and limits

See [ADR-0004](decisions/ADR-0004-snapshots.md). No new dependency was introduced. Plain owned directories were chosen over Git worktrees to preserve dirty/untracked contents without executing Git hooks/configuration or modifying source HEAD/index. Branch IDs are distinct from future execution IDs; creating a branch does not fabricate an agent execution. Full GC was explicitly deferred in favor of non-destructive reference/orphan diagnostics, as permitted by the request.

An initial strict-TS check failed because a damaged-state test fixture widened a string literal; it was corrected to the typed literal without weakening assertions. Review replaced WalkDir with 100-entry directory batches to avoid allocating an entire oversized directory. Paths are additionally bounded to 64 components. No tests were removed.

Limits: 8 MiB/file, 64 MiB included bytes, 10k visited entries, 2 MiB manifest, 100 user exclusions and 10-second operation deadline. A workspace within size limits may still exceed the time limit and fail honestly. Startup verification is linear; host hello allows 120 seconds, snapshot requests 30 seconds and ordinary requests 15 seconds. Native Windows recording/storage/snapshots are blocked. Cross-platform execution, actual power loss/disk-full/fsync failure and hostile same-user filesystem races are unverified. No Unicode filename support, metadata fidelity beyond executable status, symlink restoration, encryption, automatic cleanup, replay or full OS sandbox is claimed.

Excluded filenames reduce accidental credential capture but do not detect every secret in ordinary files. Capture requires explicit consent, stores exact private bytes and never uploads them. Do not capture a project containing secrets outside the effective exclusions. An event association records context, not proof that later captured bytes existed at that event. Original developer workspaces are never restore destinations.

Git status still reports this directory is not a Git repository. Source, tests and generated fixture locations were reviewed directly; no commit, push or publication occurred.

## Remaining acceptance and next task

Native GUI walkthrough remains unverified from Phase 2 and for the added Phase 3 controls. Run a supervised synthetic-project pass through capture/exclusions, verify, restore, branch/edit/recapture, restart, failure states and keyboard navigation before declaring full desktop acceptance. Power-loss and real filesystem fault guarantees remain explicitly unverified. Automated core and component results do not substitute for those checks.

The user subsequently authorized the Phase 4 design milestone only; its policy and capability deliverables are recorded in [Phase 4 status](IMPLEMENTATION_STATUS_PHASE_04.md). Next implementation, only after separate authorization: pure Go versioned policy types and fail-closed resolution with synthetic capability/precedence/staleness tests, excluding process launch, backend probes, migrations and UI security claims. Native GUI acceptance remains independently outstanding. No Phase 4 runtime implementation was started.

---
