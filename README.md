# ChronoSync

ChronoSync is evolving into a local-first **Multi-Agent AI Workflow Debugging and Evaluation Platform**. Today it records observable Codex executions, persists SQLite timelines, and captures, restores and branches workspace snapshots using immutable content storage.

Phase 4 delivers architecture and execution-policy **design only**: workflow correlation, evaluation evidence, and conditional platform capability comparisons. Multi-agent execution, Claude Code, evaluation runners, replay and new isolation backends are not implemented. See the [Phase 4 plan and decision register](docs/phases/PHASE-04-SANDBOX-GUARDRAILS.md). Phase 1–3 native GUI acceptance remains open.

Phase 4A adds an internal typed policy resolver and conservative pre-launch refusal guard. No enforcement backend or positive policy-controlled launch is enabled; existing Codex execution remains unchanged and is not policy-verified. See the separate [Phase 4A status](docs/IMPLEMENTATION_STATUS_PHASE_04A.md).

Phase 4D adds internal admission persistence and conservative recovery (SQLite migration 3), without connecting it to execution. Older engines refuse the upgraded database; back up the complete stopped app-data directory before upgrade. See [Phase 4D status](docs/IMPLEMENTATION_STATUS_PHASE_04D.md).

**Validation:** offline engine/UI/IPC tests, native macOS build and a real read-only Codex capture passed. Native GUI automation timed out, so end-to-end GUI verification remains incomplete. See [current status](docs/IMPLEMENTATION_STATUS.md) for exact results and remaining acceptance gates.

## Prerequisites and platforms

- Go 1.26.x; audited Go 1.26.2.
- Node 22.14+ on the Node 22 line; npm 10.9.2 and the checked-in lockfile.
- Rust stable with Cargo, rustfmt and Clippy. [Install Rust](https://rustup.rs/), then `rustup component add rustfmt clippy`.
- [Tauri 2 system prerequisites](https://v2.tauri.app/start/prerequisites/). macOS command-line tools are installed locally. Windows requires MSVC/WebView2; Linux requires WebKitGTK 4.1 and packages listed in CI.
- Authenticated Codex CLI 0.163.0 or newer; only 0.163.0 has live verification. Newer versions must retain the required flags/events. The UI requires the absolute path to the trusted installed Codex executable.
- macOS arm64 is the tested native target. Linux process-group support is implemented but not locally verified. **Windows recording is blocked** until job-object supervision and private-store ACL handling exist. Windows native compilation in CI is not proof of recorder support.

## Install and run

From the root:

```sh
npm ci
make engine-build
./bin/chronosync-engine version
npm run desktop:dev
```

The desktop build/dev hooks compile and copy the Go executable as an application resource. Rust launches that exact binary using private stdio, with no engine network listener. `npm run dev` previews the UI in a browser but deliberately refuses recording outside Tauri.

In the native app enter an absolute project path, absolute Codex executable path, task, policy and timeout. Review the consent text before starting. Read-only is the default. **Workspace-write edits the selected project directly**; no worktree, branch or restore is created in this phase. Select a run to inspect its timeline and event details. Cancellation remains pending until confirmed by the engine.

Recordings live under the Tauri application-data directory (`dev.chronosync.desktop/recordings`, resolved by the OS). The standalone engine defaults to `os.UserConfigDir()/ChronoSync/recordings`; use `serve --data-dir ABSOLUTE_PATH` for an explicitly selected alternative. Do not place recordings in source control. POSIX storage requires a private directory (0700) and uses files with 0600 permissions.

## Checks and builds

```sh
make check                       # Go race/vet/build; TS/lint/format; UI/contracts; IPC integration
make native-check                # sidecar; Rust fmt/Clippy/tests; native release build
npm run desktop:build            # native executable, no installer
npm run desktop:bundle:macos     # local unsigned macOS .app for smoke testing
npm run test:recorder            # real engine + synthetic CLI, no model calls; build engine first
npm run test:integration         # actual Go version output checked against schema
npm test                        # contract and timeline UI behavior tests
npm run typecheck
npm run lint
npm run format:check
npm run check:docs
```

Use `npm run format` and `gofmt -w engine` to format. In the restricted development sandbox run `GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod make check`. The locally installed temporary Rust toolchain needs `PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo`; it does not alter your shell profile and may disappear on cleanup/restart. Install Rust normally for continuing development.

## Opt-in live tests

These contact your Codex provider, may consume account usage, and record only in a newly created disposable directory containing public test data. Default tests never call a model.

```sh
make engine-build
CHRONOSYNC_LIVE_CODEX=/absolute/path/to/codex npm run test:live -- --consent
CHRONOSYNC_LIVE_CODEX=/absolute/path/to/codex npm run test:live -- --consent --write-fixture
```

The first test asks Codex to read a public file under read-only policy. The second authorizes workspace-write in its disposable fixture only and verifies creation of generated.txt. Both require actual command output and successful terminal state and verify reopening history. They print the temporary recordings location; they do not copy captures into source control.

## Limits and design

Prompts are sent via stdin and not persisted. Captured outputs can still contain private text; redaction is heuristic, not complete. No automatic upload of recordings occurs; live Codex itself contacts its provider. Project configuration/rules and provider-side operations are not fully observable. Process groups are not sandboxes, and hard engine/OS crashes may leave detached processes requiring investigation. File observation skips all symlinks, sensitive/configured paths, large files and generated directories; it cannot attribute concurrent external changes or see transient changes between scans.

See [documentation index](docs/README.md), [security model](docs/SECURITY_MODEL.md), and [Phase 3 plan](docs/phases/PHASE-03-SNAPSHOTS-BRANCHING.md). There is no replay or deterministic reproduction claim. No license, signing identity, or auto-update mechanism has been selected. No commit or publication was performed.

## Durable history and load test

The first Phase 2 open imports existing Phase 1 journals into history.sqlite once per run, retaining original files. Back up the entire recordings directory with the app stopped before changing versions. Unknown database versions/checksum changes refuse opening; no destructive repair or downgrade is attempted. Do not copy just a live SQLite database or put the store on a network filesystem.

Timeline pages contain at most 100 summaries. Next/previous retain the captured high-water sequence; Refresh starts a new traversal including newly appended events. Active runs poll the current page. Search matches only the first 1024 bytes of sanitized message/reason/upstream type, with ASCII case-insensitive literal matching; it does not search command output. Detail inspection verifies payload integrity. Missing data is an error, not an empty event. Commands in historical records are never executed.

Run the opt-in synthetic 100k workload from engine (no model calls):

```sh
CHRONOSYNC_LOAD=1 go test ./internal/storage -run 'TestLoad100k|TestConcurrentProducersReaders' -v -count=1
```

Add the same cache variables in restricted environments. Workload, timings and remaining GUI/fault-test limitations are recorded in [implementation status](docs/IMPLEMENTATION_STATUS.md). External payloads/orphans and old journals are retained; automated retention/GC is not implemented.

## Capture, restore and branch

Select a stopped execution and use Workspace snapshots. Review the source-byte/privacy notice, add relative exclusions, then give separate capture/materialization consent. Capture stores current included files, not the historical state at an earlier event. The recorder must be idle. Inspect committed/failed/damaged status and omission counts. Verify checks stored hashes; Restore and Branch create a fresh private directory and display its path. Edit that directory, select it as the snapshot source and capture again to store changed content while sharing unchanged immutable objects. Branches do not automatically execute Codex.

The first manifest version supports portable ASCII filenames, regular files/directories and executable status. Unicode paths are rejected; symlinks are explicitly omitted, special files rejected. Defaults exclude Git metadata, credentials and dependency/build/cache paths. Other secrets may remain in source bytes. Limits are 8 MiB/file, 64 MiB included bytes, 10k entries and a 10-second operation deadline. No developer directory is a restore destination. Independent materializations still consume disk; branching is not zero-cost and provides no replay/sandbox guarantee.

Migration 2 upgrades metadata transactionally; an older engine refuses the upgraded schema. Back up the entire app-data directory while stopped before upgrading. Failed materializations and unreferenced/temp objects are retained, never automatically deleted. Inspect retained objects reports diagnostics only. Native GUI acceptance remains unverified; see current status for tests and limitations.

Snapshot load test, from engine:

```sh
CHRONOSYNC_SNAPSHOT_LOAD=1 go test ./internal/snapshot -run TestSnapshotPerformance -v -count=1
```
