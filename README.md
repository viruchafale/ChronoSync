# ChronoSync

**Local-first execution history and workspace snapshots for Codex.**

ChronoSync is a native desktop application for recording observable Codex CLI runs, inspecting their event timelines, and capturing restorable workspace snapshots. It combines a React interface, a Rust/Tauri desktop host, and a Go engine over private stdio. Run history and snapshot content stay in private local application storage.

> **Developer preview:** the macOS ARM64 application builds and launches from a real DMG, but the artifact is currently unsigned and not notarized. Multi-agent execution, production sandbox enforcement, OS-backed approval, replay, and Windows recording are not available.

## What works today

- **Recorded Codex executions:** start read-only or workspace-write runs with explicit consent, timeout controls, cancellation, and observable command/event capture.
- **Durable timelines:** inspect paginated execution history and event details after application or engine restarts.
- **Workspace snapshots:** capture immutable content-addressed snapshots, verify stored objects, restore into a fresh private directory, and branch from an existing snapshot.
- **Local desktop bridge:** Tauri launches the bundled Go engine over inherited stdio. The engine does not open a network listener.
- **Conservative policy foundation:** typed policy and admission components refuse unsupported authority. No positive policy-controlled launch path is enabled.
- **macOS packaging:** build a mountable ARM64 DMG containing the desktop host and engine sidecar.

## Product status

| Area                                      | Status                                                                   |
| ----------------------------------------- | ------------------------------------------------------------------------ |
| Codex recorder                            | Implemented; real read-only Codex capture verified                       |
| Durable SQLite history                    | Implemented with integrity and migration checks                          |
| Snapshot capture, restore, and branch     | Implemented with bounded content-addressable storage                     |
| macOS ARM64 desktop build                 | Implemented and launch-verified from a DMG                               |
| Signed/notarized distribution             | Pending Developer ID signing, notarization, and clean-machine validation |
| Multi-agent orchestration and evaluation  | Architecture only                                                        |
| OS-backed approval and enforced isolation | Research/prototype boundary only; no production integration              |
| Windows recording                         | Blocked on job-object supervision and private-store ACL validation       |

See the [implementation status index](docs/IMPLEMENTATION_STATUS.md) for milestone evidence and the [submission-build report](docs/IMPLEMENTATION_STATUS_SUBMISSION_BUILD.md) for the current DMG verification.

## Architecture

```text
React UI  →  Rust/Tauri host  →  Go engine  →  Codex CLI
                       private stdio
```

- **UI:** React and TypeScript provide consent, recording, history, timeline, and snapshot workflows.
- **Host:** Rust/Tauri resolves the packaged engine, owns its lifecycle, and exposes typed commands to the UI.
- **Engine:** Go owns recording, event normalization, SQLite persistence, snapshot storage, policy contracts, and process supervision.
- **Adapters:** provider-specific parsing stays behind engine interfaces. Codex is the only implemented provider.

Dependencies flow from UI to host to engine ports and adapters. Captured content is treated as untrusted, and historical commands are never executed while browsing a timeline.

## Requirements

- macOS ARM64 for the locally verified native build.
- Go 1.26.x; verified with Go 1.26.2.
- Node.js 22.14+ on the Node 22 line and npm 10.9.2.
- Rust stable with Cargo, rustfmt, and Clippy.
- [Tauri 2 system prerequisites](https://v2.tauri.app/start/prerequisites/).
- An authenticated Codex CLI. Version 0.163.0 is the currently verified live integration.

## Run from source

```sh
npm ci
make engine-build
./bin/chronosync-engine version
npm run desktop:dev
```

The desktop build hook compiles the Go engine into the application resources. Browser-only Vite preview is available with `npm run dev`, but execution features deliberately require the native Tauri host.

In the app, provide:

1. An absolute project directory.
2. The absolute path to the trusted Codex executable.
3. A task, sandbox label, and timeout.
4. Explicit consent for that executable, task, workspace, policy, and local recording.

Read-only is the default. Workspace-write allows Codex to edit the selected project directly; ChronoSync does not create a worktree or automatic restore point before a run.

## Build the macOS DMG

```sh
make check
make native-check
npm run desktop:bundle:dmg
```

Output:

```text
apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg
```

The current `0.0.0` artifact is a local unsigned build. Follow the [macOS release guide](docs/RELEASE_MACOS.md) for build and verification commands. A public release still requires a final version, Developer ID signing, notarization, stapling, and Gatekeeper validation on a clean supported Mac.

## Data and security

Recordings live under the operating system application-data directory. The Tauri application uses `dev.chronosync.desktop/recordings`; the standalone engine defaults to `os.UserConfigDir()/ChronoSync/recordings`. POSIX storage requires a private `0700` directory and uses `0600` files.

Prompts are sent to Codex through stdin and are not intentionally persisted by ChronoSync. Observable output can still contain private data, and redaction is heuristic. No recording is automatically uploaded by ChronoSync, although the Codex CLI contacts its configured provider during a live run.

Process groups provide cancellation and cleanup controls; they are not an OS security sandbox. Current policy and admission components do not authorize a positive isolated launch. Snapshot capture excludes configured sensitive, generated, dependency, cache, and Git paths, but other secrets can remain in included source files. Review the [security model](docs/SECURITY_MODEL.md) before using real repositories.

## Snapshots and history

Timeline queries are bounded and use stable pagination. Stored event payloads are integrity-checked before detail display. The current snapshot format supports portable ASCII paths, regular files, directories, and executable status. Symlinks are omitted, special files are rejected, and capture is bounded by file, byte, entry, and time limits.

Restore and Branch always materialize into a fresh private directory. Branching shares immutable stored objects but each materialized workspace consumes normal filesystem space. Snapshotting does not provide deterministic replay or a security boundary.

Back up the complete application-data directory while ChronoSync is stopped before changing versions. Older engines refuse databases with newer schema versions; there is no destructive downgrade or automatic repair path.

## Development checks

```sh
make check                       # Go race/vet/build; TS/lint/format; UI/contracts; integrations
make native-check                # sidecar; Rust fmt/Clippy/tests; native release build
npm run desktop:build            # native executable without an installer
npm run desktop:bundle:macos     # local unsigned macOS .app
npm run desktop:bundle:dmg       # local unsigned ARM64 macOS DMG
npm run test:recorder            # real engine with a synthetic CLI; no model call
npm run test:integration         # engine version output against the frontend schema
npm test                         # contract and UI behavior tests
npm run typecheck
npm run lint
npm run format:check
npm run check:docs
```

Default checks never invoke a model. The opt-in live tests use a disposable directory containing public test data and may consume provider usage:

```sh
make engine-build
CHRONOSYNC_LIVE_CODEX=/absolute/path/to/codex npm run test:live -- --consent
CHRONOSYNC_LIVE_CODEX=/absolute/path/to/codex npm run test:live -- --consent --write-fixture
```

## Documentation

- [Implementation status](docs/IMPLEMENTATION_STATUS.md)
- [Architecture](docs/ARCHITECTURE.md)
- [System design](docs/SYSTEM_DESIGN.md)
- [Security model](docs/SECURITY_MODEL.md)
- [Testing strategy](docs/TESTING_STRATEGY.md)
- [Codex integration](docs/CODEX_INTEGRATION.md)
- [macOS release guide](docs/RELEASE_MACOS.md)
- [Documentation index](docs/README.md)

ChronoSync currently has no selected license, update mechanism, signed release channel, or production support commitment.
