# Disposable native GUI acceptance candidate status

Updated: 2026-11-08. Preparation for human acceptance of existing Phases 1–11; not Phase 12.

A separate local macOS application bundle was built successfully from the current dirty working tree into an isolated Cargo target:

`/private/tmp/chronosync-native-gui-acceptance-20261108/release/bundle/macos/ChronoSync.app`

The bundle preserves identifier `dev.chronosync.desktop` and version `0.0.0`. Its arm64 host executable SHA-256 is `49075b0a8c28ec6824bc9cd9e98d24137406d67b74af401045635f143df1380b`. The bundled engine reports version `0.8.0` and has SHA-256 `0baecfb8bf3e38af20e44c9d2f7f6c33bc7a183ac84caec7ebb47870962af8f8`.

This is not a production-signed candidate. The host has only a linker-generated ad-hoc signature, no TeamIdentifier, no sealed bundle resources and no notarization. Strict `codesign` verification did not pass. It is suitable only for local disposable visual, native confirmation, file-picker, focus and cancellation checks on this development Mac. Positive protected-Keychain/user-presence, Developer ID, notarization and Gatekeeper acceptance remain blocked.

The app launched, exposed a native accessibility surface, rendered the current workbench, connected to its bundled private-stdio engine and quit normally. No confirmation, file picker, identity, Keychain or destructive workflow was activated. The build hook recreated the repository sidecar with byte-identical output. The protected DMG retained SHA-256 `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`.

Use [DISPOSABLE_NATIVE_GUI_ACCEPTANCE_CHECKLIST.md](DISPOSABLE_NATIVE_GUI_ACCEPTANCE_CHECKLIST.md) under a dedicated disposable macOS user account. The detailed identity, build and limitation record is in [DISPOSABLE_NATIVE_GUI_ACCEPTANCE_REPORT.md](DISPOSABLE_NATIVE_GUI_ACCEPTANCE_REPORT.md). Native workflows remain manually unverified until an operator records the checklist results.
