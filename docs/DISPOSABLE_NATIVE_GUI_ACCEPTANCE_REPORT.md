# Disposable native GUI acceptance candidate report

## Outcome and boundary

A human-operable `.app` candidate was prepared successfully for the remaining native GUI checks for Phases 1–11. This is a local disposable development candidate, not a release candidate and not Phase 12. Building and launch smoke do not verify native dialogs, Keychain behavior or user presence.

## Candidate identity

| Field                        | Recorded value                                                                               |
| ---------------------------- | -------------------------------------------------------------------------------------------- |
| App path                     | `/private/tmp/chronosync-native-gui-acceptance-20261108/release/bundle/macos/ChronoSync.app` |
| Product                      | `ChronoSync`                                                                                 |
| Bundle identifier            | `dev.chronosync.desktop`                                                                     |
| App version / bundle version | `0.0.0` / `0.0.0`                                                                            |
| Architecture                 | arm64                                                                                        |
| Host executable              | `Contents/MacOS/chronosync-desktop`                                                          |
| Host SHA-256                 | `49075b0a8c28ec6824bc9cd9e98d24137406d67b74af401045635f143df1380b`                           |
| Bundled engine               | `Contents/Resources/binaries/chronosync-engine`                                              |
| Engine SHA-256               | `0baecfb8bf3e38af20e44c9d2f7f6c33bc7a183ac84caec7ebb47870962af8f8`                           |
| Engine report                | version `0.8.0`, protocol 1, current Phase 1–11 capabilities                                 |
| Size reported by Tauri       | 24.10 MiB                                                                                    |

## Source state

The candidate was built from Git HEAD `9364eaac746a5f52382eeba881db729bd4f15964` plus the preserved dirty working tree containing the authorized Phase 8–11, audit, GUI overhaul, visual refinement and acceptance-closure work. Immediately after the build the worktree contained 44 modified tracked paths and 76 untracked paths. No cleanup, reset, commit or source-state substitution occurred.

## Build

The existing `desktop:bundle:macos` script was used with the repository's already available offline Rust/Cargo caches and a dedicated target directory:

```sh
PATH=/private/tmp/chronosync-phase5-cargo/bin:/usr/local/bin:/usr/local/go/bin:/usr/bin:/bin:/usr/sbin:/sbin \
RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup \
RUSTUP_TOOLCHAIN=stable \
CARGO_HOME=/private/tmp/chronosync-acceptance-cargo \
CARGO_NET_OFFLINE=true \
CARGO_TARGET_DIR=/private/tmp/chronosync-native-gui-acceptance-20261108 \
GOCACHE=/private/tmp/chronosync-go-cache \
npm run desktop:bundle:macos
```

The command completed successfully, rebuilt the frontend and host, and produced one app bundle. It did not request a DMG bundle. The hook rebuilt `apps/desktop/src-tauri/binaries/chronosync-engine`; its before/after SHA-256 remained `0baecfb8bf3e38af20e44c9d2f7f6c33bc7a183ac84caec7ebb47870962af8f8`.

## Signature and release classification

`codesign -dv --verbose=4` reports `Signature=adhoc`, linker-signed flags, no TeamIdentifier and identifier `chronosync_desktop-d9e6f269c1016a66` for the host signature. Strict deep bundle verification failed with `code has no resources but signature indicates they must be present`; `spctl` did not produce an acceptance result. No Developer ID identity exists on this host, and none was invented or selected.

Consequently this candidate:

- is not production-signed, notarized, stapled or Gatekeeper-accepted;
- cannot establish positive protected-Keychain or per-sign user-presence behavior;
- must not be published or used for Phase 11 signed-release acceptance;
- is usable for local native window, confirmation, file-picker, focus, cancellation and explicit login-Keychain development checks.

## Launch smoke

The app was launched directly through the native automation surface. It appeared as `ChronoSync`, exposed the expected accessibility tree, rendered the Overview destination, changed status from `Connecting to engine…` to `Engine responded`, and displayed the private-stdio/local-evidence disclosure. It then quit normally with Command-Q. No fixture, export, import, identity, trust, Keychain, provider or destructive action was invoked.

This proves only that the disposable bundle can be opened for a human acceptance session.

## Data isolation requirement

The stable bundle identifier intentionally maps to the normal ChronoSync application-data namespace and the Phase 9 implementation uses the normal login Keychain service `local.chronosync.signing.v1`. The candidate must therefore be tested in a dedicated disposable macOS standard-user account. Do not rename the bundle identifier or alter custody behavior merely to isolate tests. Do not run identity or migration ceremonies against a developer's normal account.

## Protected artifact

Before and after the app-only build, the protected artifact remained:

`apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg`

SHA-256:

`e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`

The build target was outside the repository release target. The DMG was not rebuilt, mounted, overwritten or modified.

## Remaining acceptance

The tests in [DISPOSABLE_NATIVE_GUI_ACCEPTANCE_CHECKLIST.md](DISPOSABLE_NATIVE_GUI_ACCEPTANCE_CHECKLIST.md) remain human tasks. In particular, native dialog appearance and cancellation are unverified, login-Keychain operations require explicit operator authorization, and positive protected user presence remains externally blocked for this ad-hoc candidate.
