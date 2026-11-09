# UI visual refinement and acceptance report

## Outcome and boundary

The developer workbench received a focused visual audit and refinement pass on 2026-11-07–08. The review used the real React screens with bounded, inert synthetic evidence injected through existing typed frontend interfaces. The harness rejects native and write operations, has no engine connection, and displays a persistent synthetic-evidence banner. Production defaults and IPC behavior are unchanged.

This is browser-rendered evidence, not native GUI or release-security acceptance. It does not establish content integrity, authentication, provider behavior, replay determinism or launch authorization. No app bundle or DMG was created or changed.

## Visual evidence

All nine destinations were rendered and visually inspected at 1440×900 and 1280×800. At 1024×768, Overview, Capture, Timeline, Snapshots, Replay, Analysis and Governance were screenshot-verified; Exchange was visually inspected in the live compact preview. A 2026-11-08 closure pass restored the dedicated Chrome binding and added a genuine Security capture at 1024×768. Security is therefore screenshot-verified at all three target viewports.

The retained evidence contains 35 valid target-viewport PNG page captures plus two earlier diagnostic captures:

- 1440×900: all nine destinations, selected timeline state, filtered command palette and before/after analysis findings.
- 1280×800: all nine destinations, including populated evaluation, governance and trust views.
- 1024×768: eight destinations, selected timeline, analysis report and timeline filter before/after evidence; Exchange remains live-preview-only at this viewport.
- Two 1440×542 diagnostic captures are retained but excluded from target-viewport acceptance.

Chrome generated two 1024 captures with incorrect zoom/crop despite displaying the requested dimensions. They were inspected, rejected and not copied into repository evidence. Screenshot files and their classification are documented in [the evidence README](evidence/ui-refinement/README.md).

## Defects found and fixed

- Excessive page/top-bar whitespace reduced; overview empty and populated states now use a compact engineering-workbench hierarchy.
- Timeline rows now separate event title, timestamp, status and sequence; warnings receive immediate semantic emphasis.
- The inspector uses a compact two-column metadata grid with safe wrapping for long event IDs and payloads.
- Timeline filters no longer expand to three rows at 1024×768; the two-column fallback begins only below 760 pixels.
- Comparison and governance tables receive consistent spacing, fixed aligned-timeline columns and safe long-value wrapping.
- Divergence selection now has a clear accent edge and surface state; failed, damaged and revoked badges use failure colors.
- Snapshot list actions use concise visible labels while preserving the full snapshot ID in accessible names and tooltips.
- Dialogs are viewport-bounded and scroll internally.
- Destination changes reset document scroll, preventing a scrolled analysis result from opening Governance midway down the page.
- Recorder errors are shown only on recorder destinations, avoiding unrelated error banners on evaluation and governance screens.

## Interaction verification

Visually inspected in native Chrome automation: all sidebar destinations, selected navigation states, compact navigation, command palette filtering, Escape dismissal and focus restoration, timeline event selection, long payload/ID wrapping, clipboard success feedback, keyboard split-width adjustment, section-anchor navigation and document-scroll reset. Pointer resize is implemented by the existing CSS resize handle but was not reliably measurable through the scaled native Chrome preview.

Automated tests cover route navigation, sidebar persistence, command-palette keyboard behavior/focus restoration, exact clipboard copy and failure feedback, destination scroll reset, recorder filters and paging, timeline selection, snapshot confirmation cancellation/confirmation and existing feature workflows. Snapshot write/error paths were not activated in the visual harness because it intentionally rejects writes.

Native macOS dialogs, Keychain ceremonies and the Tauri window are not part of this browser-only pass. Loading behavior is automated-tested; it was not retained as screenshot evidence because the bounded in-memory reads resolve immediately.

## Files changed for this pass

- `apps/desktop/src/App.tsx`
- `apps/desktop/src/Workbench.tsx`
- `apps/desktop/src/Workbench.test.tsx`
- `apps/desktop/src/Snapshots.tsx`
- `apps/desktop/src/style.css`
- `tests/visual/*` (isolated visual harness only)
- this status/report, plan/index/changelog updates and `docs/evidence/ui-refinement/*`

The worktree contains substantial earlier authorized Phase 8–11, audit and GUI-overhaul changes. They are preserved and are not re-attributed to this refinement pass.

## Validation

The final verification run passed Prettier, ESLint, strict TypeScript, Vitest, the production frontend build, `make check` and `make native-check`. The native command used the repository's isolated offline Rust/Cargo caches and `--no-bundle`; six Rust tests passed and two existing real-Keychain ceremony tests remained ignored. The frontend suite contains 40 passing tests in nine files after adding the destination-scroll regression.

The protected `ChronoSync_0.0.0_aarch64.dmg` SHA-256 remains `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`.

## Remaining limitations

Native-dialog acceptance remains unverified. Chrome native automation was restored by rebinding the dedicated window through `com.google.Chrome`; the earlier loss was caused by active-window selection after DevTools changed focus. The two invalid earlier images remain excluded. The bare no-bundle native executable still does not expose an application surface to the automation inventory, so no claim is made for native GUI acceptance, a signed release, physical user-presence prompts or the submitted DMG's visual contents. See the [final closure report](GUI_ACCEPTANCE_CLOSURE_REPORT.md).
