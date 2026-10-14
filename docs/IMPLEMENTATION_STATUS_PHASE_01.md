# Phase 1 implementation status

Implemented; native GUI acceptance remains unverified/blocked.

Evidence and limitations below describe this phase at the time recorded. Later phases do not retroactively accept its outstanding gates.

[Status index](IMPLEMENTATION_STATUS.md)

As of 2026-10-21. **Phase 1: BLOCKED on native GUI end-to-end verification, not on recorder implementation.** Working engine, native bridge/UI and automated tests are implemented. Real Codex read-only and write-fixture recordings passed. Do not mark Phase 1 COMPLETE without the remaining GUI acceptance evidence or an explicit approved exception. Phase 2–7 are NOT_STARTED.

Phase 0's missing Rust/Cargo compile blocker was addressed using an official Rust 1.99.1 toolchain installed under /tmp, without modifying shell profiles. Native format/Clippy/test/release build now pass, and Cargo.lock exists. The native GUI smoke gate was not retroactively assumed passed: UI automation timed out.

## Implemented modules and behavior

- Go model/adapter/codex/supervisor/recorder/storage/workspace/ipc packages; version and serve CLI.
- Engine 0.1.0, protocol v1 with capability negotiation, event schema v2 and typed Go/TypeScript payloads. Legacy proposed event v1 remains for historical reference.
- Random local IDs, monotonic decimal sequence, UTC observation times, explicit lifecycle and terminal reasons. No distributed causal-order claim.
- Supported structured Codex lifecycle/message/summary/command/tool/file reports; bounded unknown/malformed handling and pre-storage redaction.
- One active execution, explicit consent, selected workspace/policy/timeout, direct argv and stdin prompt. POSIX group cancellation, deadline, output bounds and ordinary descendant cleanup.
- Private, exclusively locked, checksummed/fsynced JSONL staging history plus atomic metadata. Restart marks unfinished runs interrupted. Damaged records warn; damaged run directories do not block unrelated histories. No SQLite yet.
- Metadata-only before/after file hashing through os.Root, exclusions, symlink skipping and independently labeled changes.
- Rust packaged-resource bridge, bounded stdio request/reply and cleanup; React start/inspect/cancel UI with no fake fallback.
- Source/test fixtures contain synthetic data only. Live temporary journals remain outside the repository.

## Final automated verification — passed

The normal local commands are documented in README. This sandbox required the explicit environment prefix below.

| Actual command                                                                                                                                                | Result                                                                                                                                                          |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOCACHE=/tmp/chronosync-go-cache make check`                                                                                                                 | Exit 0 after final storage/supervisor fixes.                                                                                                                    |
| Go steps inside make check: `go vet ./...`, `go test -race ./...`, `go build -o ../bin/chronosync-engine ./cmd/chronosync-engine` (engine cwd)                | Passed all packages; lifecycle/parser/storage/scanner/process/IPC tests exercised. No live LLM in Go tests.                                                     |
| `npm run format:check`, `npm run lint`, `npm run typecheck`                                                                                                   | Prettier, ESLint and strict TypeScript passed.                                                                                                                  |
| `npm test`                                                                                                                                                    | Three test files, 16 tests passed: 12 original contracts and four UI behavior tests.                                                                            |
| `npm run build`                                                                                                                                               | Vite production frontend build passed.                                                                                                                          |
| `npm run check:docs`                                                                                                                                          | Required docs, eight 16-section plans and local links passed.                                                                                                   |
| `npm run test:integration`                                                                                                                                    | Actual Go version report validated against the shared JSON schema.                                                                                              |
| `npm run test:recorder`                                                                                                                                       | Real engine + synthetic CLI via stdio; schema/order/command/change/unknown handling and restart persistence passed. Final run bafc09963db835e57fc40f85b36b2d25. |
| `PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo GOCACHE=/tmp/chronosync-go-cache make native-check` | Exit 0: sidecar, Rust fmt, locked Clippy, Rust bridge test and native release build passed.                                                                     |
| Rust test inside native-check                                                                                                                                 | One real-engine bridge test passed: handshake, invalid request, active cancellation, confirmed terminal state and history after restart.                        |
| `GOCACHE=/tmp/chronosync-go-cache GOOS=linux GOARCH=amd64 go build -o /tmp/chronosync-engine-linux ./cmd/chronosync-engine` (engine cwd)                      | Passed cross-compilation, not Linux execution.                                                                                                                  |
| `GOCACHE=/tmp/chronosync-go-cache GOOS=windows GOARCH=amd64 go build -o /tmp/chronosync-engine-windows.exe ./cmd/chronosync-engine` (engine cwd)              | Passed cross-compilation; recording remains intentionally blocked on Windows.                                                                                   |

Native executable: apps/desktop/src-tauri/target/release/chronosync-desktop. A local unsigned .app is generated under target/release/bundle/macos/ChronoSync.app using the documented desktop:bundle:macos command. Its Go executable resource has executable permissions. Artifact creation is not GUI verification.

## Live evidence — passed after authorized sandbox retry

- `CHRONOSYNC_LIVE_CODEX=/opt/homebrew/bin/codex npm run test:live -- --consent`: first attempt under the outer sandbox failed honestly with process_failure and preserved diagnostic `failed to initialize in-process app-server client: Operation not permitted`. Failed run: 809cef4fd2f86f18b3a5e76435715129.
- The same command with approved execution outside that outer sandbox passed with Codex's own read-only policy unchanged. Run **801419b60519f32eee535f9253041c6d**: 12 events, actual command output, completed turn + process, zero observed changes, and successful restart/retrieval. The recorded command output was checked for the expected public fixture text.
- `CHRONOSYNC_LIVE_CODEX=/opt/homebrew/bin/codex npm run test:live -- --consent --write-fixture`, with approved outer execution, passed. Run **a7a6cc2ed9076936e09f694f4b86e28d**: 15 events, actual command output, one independently observed generated.txt creation, completed state and successful restart/retrieval. Only a newly created disposable public-fixture Git repository was used.
- These runs prove the observed workflow on installed Codex 0.163.0/macOS, not all models/events, hidden context, provider transports or other platforms. Live cancellation was not separately tested; cancellation is verified using real synthetic subprocesses and the Rust bridge.

## Failed attempts resolved

- npm/crates.io sandbox DNS failed; approved fetches succeeded. Added @tauri-apps/api for invoke, Testing Library/jsdom for behavior tests, serde/serde_json for the host. npm install reported zero vulnerabilities across 194 packages.
- Initial command ran go test from repository root instead of engine; corrected working directory.
- Private-store tests initially used a pre-created test directory with permissions broader than 0700. Tests now create a private store child; the security check was retained.
- UI test query matched both the executable textbox and consent label. It now targets the textbox by role/name without weakening assertions.
- New integration script initially lacked explicit timer imports for ESLint. Added imports; lint passed.
- Storage review found that one unreadable run could block all history; added visible interrupted summaries and a regression test. Warning count and malformed-frame count are bounded, and valid later events survive malformed lines.

## Blocked, skipped and unverified

- **Native GUI end-to-end smoke:** opening the built .app through native automation timed out. No verified screen, real-window start/inspect/cancel sequence or keyboard walkthrough. Do not present React tests as native GUI E2E.
- Native Windows/Linux runtime, remote GitHub CI, signing/notarization and installer distribution were not run. Windows recording has no unsafe fallback.
- Hard engine/OS crash, detached-process cleanup and full disk-full/power-loss fault matrix are unverified. Recovery explicitly says cleanup unverified; it never resumes a model automatically.
- Full secret detection, OS containment, SQLite migrations, CAS/snapshots, replay, determinism and advanced guardrails are not implemented.
- Directory still has no Git metadata (`git status` reports not a repository), so no Git diff was available. Reviewed the source/tests and file inventory directly; no commits, pushes or publications occurred. Secret-pattern review found only synthetic test strings and redaction regexes.

## Next concrete task

Run a supervised native GUI acceptance pass on macOS: launch the .app, confirm packaged engine startup, use a disposable project, start recording through the form, inspect actual ordered command details and file changes, cancel an active run, close/reopen and inspect persisted state, then check keyboard navigation. Record evidence or fix failures before marking Phase 1 COMPLETE. If the GUI automation environment remains unavailable, obtain an explicit acceptance exception or perform a manual review. Do not begin Phase 2 automatically.
