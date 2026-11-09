# Phase 9F implementation status — adversarial and end-to-end validation

Disposable Go/Rust/React tests cover custody creation/public/signing, denial/missing/deletion/malformed/unsupported/conflict behavior, one-use signing sessions, migration mismatch/corruption/interruption/restart/concurrency/idempotency, trust/identity corruption, rotation/revocation and Phase 8 package compatibility. The direct macOS login-Keychain round trip passes; the stronger entitlement-dependent attempt and exact failure are retained as evidence.

Final portable/native gates, source/dependency audits, native GUI status and unchanged submitted-DMG checksum are recorded in the [Phase 9 completion report](PHASE9_COMPLETION_REPORT.md).

Final evidence on 2026-11-03:

- `go test -race ./internal/exchange ./internal/ipc`: passed.
- focused Exchange UI test: 5/5 passed.
- default Rust tests: 4 passed, the real Keychain test intentionally ignored.
- separately authorized real login-Keychain round trip: 1/1 passed and deleted
  its disposable item. The workspace-sandbox attempt returned `-50` first.
- `npm audit --offline --json`: zero reported vulnerabilities across 253
  dependencies; Go modules verified; direct native dependency trees resolved to
  the pinned versions.
- `make check`: passed after Prettier corrected three files.
- `make native-check`: passed after `cargo fmt`; its no-bundle release build
  warned that this temporary Rust toolchain could not run `rust-objcopy` because
  `libLLVM.dylib` was absent, but Cargo/Tauri completed successfully.
- the existing DMG remains 9,479,647 bytes with SHA-256
  `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`.
