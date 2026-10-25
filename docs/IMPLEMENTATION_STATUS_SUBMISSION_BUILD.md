# Submission build status — macOS DMG

**Date:** 2026-10-25

**Host:** macOS 26.6.2 ARM64

**Scope:** Build and verify the existing desktop application as a real macOS DMG. No Phase 4E–4G runtime behavior, authentication, admission, sandbox, schema, public API, signing credential, or release upload changed.

## Result

A real, checksum-valid, mountable ARM64 DMG was produced from the existing Tauri application. The mounted app launched and rendered its native window. “Refresh executions” completed in the empty-history state, exercising packaged-engine discovery, engine startup, protocol negotiation and a `run.query` bridge request. No Codex execution or external agent was started.

This is a **local unsigned build**, not a public-distribution release. The app and bundled engine have linker-generated ad-hoc Mach-O signatures; the application has no sealed resources or Developer ID team identity. Deep `codesign` verification fails and Gatekeeper does not accept the bundle. Signing, notarization and stapling remain required before distribution.

## Artifact

| Property            | Observed value                                                                  |
| ------------------- | ------------------------------------------------------------------------------- |
| Path                | `apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg` |
| Size                | 9,479,647 bytes (Tauri reported 9.04 MiB)                                       |
| SHA-256             | `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`              |
| Application version | `0.0.0` (`CFBundleShortVersionString` and `CFBundleVersion`)                    |
| Bundle identifier   | `dev.chronosync.desktop`                                                        |
| App architecture    | Mach-O 64-bit executable ARM64                                                  |
| Engine architecture | Mach-O 64-bit executable ARM64                                                  |
| Engine location     | `ChronoSync.app/Contents/Resources/binaries/chronosync-engine`                  |

The checksum identifies this exact local artifact and will change on a rebuild.

## Build and test evidence

| Command or check                                                                                            | Result                                                                                                                                                                                                         |
| ----------------------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOPROXY=off GOSUMDB=off ... make check`                                                                    | Passed: Go formatting, vet, race tests and build; Prettier, ESLint, TypeScript, 4 Vitest files/22 tests, frontend build, documentation links, version integration and synthetic recorder/snapshot integration. |
| First constrained `make native-check` attempt                                                               | Failed before compilation because the temporary `PATH` omitted the installed Go binary. This was an invocation error, not a product failure.                                                                   |
| `CARGO_NET_OFFLINE=true GOPROXY=off GOSUMDB=off PATH=/tmp/chronosync-cargo/bin:$PATH ... make native-check` | Passed: sidecar build, Rust format, locked Clippy with warnings denied, one real-engine bridge test, and optimized native build.                                                                               |
| First sandboxed `npm run desktop:bundle:dmg` attempt                                                        | Built the app, then `hdiutil` failed with `Device not configured` because disk-image device access was unavailable in the filesystem sandbox.                                                                  |
| Same offline DMG command with native `hdiutil` access                                                       | Passed; Tauri produced one DMG. No dependency download was needed.                                                                                                                                             |
| `hdiutil verify`                                                                                            | Passed; image checksum reported valid.                                                                                                                                                                         |
| Read-only mount and bundle inspection                                                                       | Passed; image contains `ChronoSync.app`, an Applications link, metadata, icons and an executable packaged engine.                                                                                              |
| Bundled `chronosync-engine version`                                                                         | Passed: protocol 1, engine `0.3.0`, and `version`, recorder, event, timeline and snapshot capabilities returned.                                                                                               |
| `codesign --verify --deep --strict`                                                                         | Failed: resources are not sealed by a distribution signature.                                                                                                                                                  |
| `spctl --assess --type execute`                                                                             | Failed in the code-signing subsystem; the local build is not Gatekeeper-ready.                                                                                                                                 |

## Launch verification

The app was launched directly from the read-only mounted DMG with native UI automation. The `ChronoSync` window rendered the Phase 3 recorder form and empty execution-history panel. Invoking “Refresh executions” returned “No recorded executions loaded” without a bridge error. The UI action causes the Rust host to locate `Contents/Resources/binaries/chronosync-engine`, start `serve` with the private application-data path, negotiate protocol 1, and issue the history query. The app was then quit; neither desktop nor engine process remained, and the image detached successfully.

No execution form was submitted, no Codex process was started, and no live provider call was made. Existing synthetic recorder and Rust bridge tests cover execution/cancellation mechanics. This check does not establish end-to-end manual recording acceptance, OS-backed authentication, peer code identity, sandbox enforcement, replay, or positive policy admission.

## Repository changes

- Added `desktop:bundle:dmg` as the repeatable Tauri DMG command.
- Added the macOS release guide and this separate submission-build evidence report.
- Linked the release documents from the README and documentation/status indexes.
- Added a changelog entry.

All previously present Phase 4E–4G working-tree changes were preserved. The generated application, sidecar and DMG remain ignored build outputs. No commit, push, upload or publication was performed.

## Remaining distribution blockers

1. Select the real release version; `0.0.0` is the current repository version.
2. Configure an approved Apple Developer ID Application identity and sign the app, nested engine and sealed resources.
3. Notarize the signed artifact, staple the ticket, and verify it with `codesign`, `spctl` and an offline Gatekeeper test on a clean supported Mac.
4. Define and test the supported macOS deployment matrix. The generated metadata currently reports `LSMinimumSystemVersion` 10.13, while newer security design work proposes a higher floor for future native authentication experiments.
5. Perform the outstanding end-to-end manual recorder acceptance pass on a clean installation before claiming production readiness.
