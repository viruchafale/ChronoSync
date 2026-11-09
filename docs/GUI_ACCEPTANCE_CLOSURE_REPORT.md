# Final GUI acceptance closure report

## Outcome and scope

The repository-fixable GUI acceptance work for existing Phases 1–11 is complete. Phase 12 was not started. The isolated visual harness still renders real React components with bounded inert synthetic data, rejects native/write operations, has no engine connection and shows its synthetic-evidence banner.

Browser visual acceptance passes at the requested Security viewports. Native macOS confirmation dialogs remain a genuine manual acceptance blocker because the current no-bundle executable is not exposed as an accessible application surface to the available automation. No bundle, DMG or release artifact was rebuilt to work around that boundary.

## Automated acceptance

The final run covers Prettier, ESLint, strict TypeScript, Vitest, the frontend production build, the visual-harness TypeScript configuration, `make check`, no-bundle `make native-check` and `git diff --check`. Results are recorded below after the final run.

## Visually verified acceptance

- Security at 1440×900: retained target capture inspected; content, metadata, hierarchy, spacing, semantic wording and panel bounds pass.
- Security at 1280×800: retained target capture inspected; the full panel remains readable without clipping or horizontal overflow.
- Security at 1024×768: live exact Chrome device preview inspected and a genuine 2048×1536 Retina capture retained as `1024x768/security.png`; compact navigation, content wrapping, panel sizing and bottom status/banner placement pass.
- Every previously retained PNG was inspected: 34 target captures and two 1440×542 diagnostic captures. With the new Security capture, the repository holds 35 target captures and two diagnostics.
- Current Overview, Capture, Timeline, Snapshots, Replay, Analysis, Governance, Exchange and Security layouts show no unresolved repository-level visual defect in the retained target evidence.
- Historical files containing `before`, plus the earlier 1280 Governance and Snapshot states, remain valid defect evidence. They visibly preserve superseded density, column-spacing or action-label issues and are not used as proof of the current state.
- The two 1440×542 captures remain diagnostic-only because their height is not a target viewport.

The Security destination has no action button or renderer dialog. Its status language consistently distinguishes unavailable, synthetic and unassessed states. Existing HTML dialog bounds and action presentation remain covered by the prior browser interaction pass and automated component tests; this closure does not relabel them as native dialogs.

## Native-window investigation

Chrome reliability was restored without a dependency or repository change. The dedicated window titled `ChronoSync — synthetic visual test` was rebound using `com.google.Chrome`; its DevTools accessibility tree reported Width 1024 and Height 768. The previous failure occurred because native app-name binding selects the currently active Chrome window, while DevTools focus/window changes caused later actions to target a different window or no window.

The current no-bundle executable at `apps/desktop/src-tauri/target/release/chronosync-desktop` was then launched directly. It remained running but was absent from the native automation inventory. Both supported binding attempts failed:

- full executable path: `Invalid app`
- configured identifier `dev.chronosync.desktop`: `Invalid app`

This is an automation-surface limitation, not evidence that the Tauri window or its dialogs pass or fail. The executable is a bare Mach-O rather than a discoverable `.app` bundle for this automation inventory. Building a fresh bundle solely to gain a binding would violate the task's protected-artifact constraint. Actual native confirmation, file-picker and Keychain ceremony appearance therefore remains manually unverified.

## Manual acceptance still required

An operator must use the existing current candidate or a separately authorized disposable candidate, follow [the native GUI checklist](NATIVE_GUI_ACCEPTANCE_CHECKLIST.md), and record sanitized screenshots. At minimum, inspect Security at the three target content sizes, open a non-destructive export confirmation and cancel it, open an import file picker and cancel it, verify focus/keyboard/cancel behavior, and separately perform only the already-authorized disposable Keychain ceremonies. The protected submitted DMG must not be substituted for the current source UI or modified.

## Changes made

- Added the missing genuine `docs/evidence/ui-refinement/1024x768/security.png` capture.
- Updated the visual evidence README and prior refinement report to close the 1024 Security limitation while retaining honest native-dialog limits.
- Added a separate closure status and this consolidated report.
- Updated the status index, native manual checklist, GUI plan and changelog.
- No frontend or backend source defect required a change in this closure pass.

## Validation

All requested checks passed on 2026-11-08:

- `npm run format:check`: passed.
- `npm run lint`: passed.
- `npm run typecheck`: passed in strict mode.
- `npm test -- --run`: 9 files and 40 tests passed.
- `npm run build`: passed; 41 modules transformed and production assets emitted.
- `npx tsc --noEmit -p tests/visual/tsconfig.json`: passed.
- `GOCACHE=/private/tmp/chronosync-go-cache make check`: passed, including Go formatting, vet, race tests/build, documentation, release configuration, schema integration and inert recorder/snapshot integration. No live agent was invoked.
- Offline-cache `make native-check`: passed. Rust formatting and Clippy passed; 6 tests passed and the 2 existing interactive real-Keychain tests remained ignored; the no-bundle release executable built successfully.
- `git diff --check`: passed with no output.
- Evidence inventory: 35 target PNGs and 2 diagnostic PNGs. The new Security file is 2048×1536 pixels, representing the 1024×768 Retina viewport.

## Protected artifact

`apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg` was read-only hashed after the inspection. Expected and observed SHA-256:

`e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`

The DMG was not rebuilt, mounted, edited or published.

## Remaining blocker

`MANUAL_ACCEPTANCE_PENDING`: actual native macOS confirmations, file pickers and Keychain/user-presence dialogs require a human-operable bundled candidate that represents the current source. No browser screenshot or component test is claimed as native-dialog verification.
