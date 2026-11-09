# Developer workbench overhaul report

## Outcome and scope

Implemented the UI overhaul of existing Phase 1–11 capabilities. This is not Phase 12. No backend feature, IPC/schema change, dependency, release publication, commit or new app bundle was made for this milestone. Pre-existing uncommitted backend and release work remains intact.

The [inventory and plan](GUI_OVERHAUL_PLAN.md) maps every existing screen to its real data source and redesign. The shell uses graphite surfaces, restrained violet actions, cyan technical identifiers and semantic warning/failure colors. CSS variables define surfaces, borders, typography, spacing, radii, focus rings, transition and dialog shadow. SVG icons are local and dependency-free. Technical content uses monospace; payloads remain selectable, escaped text with original whitespace.

## Implemented

- Nine meaningful destinations consolidate existing functionality: overview, capture, timeline, snapshots/branches, replay/regression, comparison/analysis, benchmarks/retention, exchange/trust and security/settings.
- `Workbench` provides navigation, breadcrumbs, selected workspace context, sidebar density persistence, keyboard-accessible command search, named native HTML dialog, focus restoration and skip link.
- Overview filters loaded real executions by workspace, status or ID. The count and search do not imply a database-wide query. Empty/error states do not fabricate evidence.
- Timeline retains existing server query filters and pagination. A split list/inspector displays actual event metadata and payload, selected rows, exact ID/payload copy and a keyboard-operable width slider. CSS also provides pointer resizing.
- Every feature module uses the shared design system and section anchors. Analysis has side-by-side baseline/target configuration. Snapshot, fixture, report and key IDs have copy actions; status badges retain text labels.
- Restore and branch actions require a named confirmation dialog after existing consent. Cancelling does not call materialization. Existing trust, custody, deletion and native confirmation paths remain intact.
- Security/settings states implementation limits explicitly and does not assert a live signed-release assessment. The status bar reports the most recent recorder response, not a continuous connectivity probe.

No production fake-data mode, simulated backend, provider fallback or command execution was added. Feature panels remain mounted when hidden to retain drafts/results. Snapshot views are scoped to the selected execution. Navigation is in memory; sidebar density alone is persisted. Clipboard failures are visible.

## Automated validation

Actual commands run on this machine:

Final portable run on 2026-11-07: 39 frontend tests passed in nine files. Production output: 275.24 kB JavaScript (84.11 kB gzip), 11.44 kB CSS (3.48 kB gzip). `git diff --check` passed.

- `GOCACHE=/private/tmp/chronosync-go-cache make check`: passed Go formatting/vet/race tests/build, Prettier, ESLint, TypeScript, Vitest, Vite production build, documentation checks, release configuration checks, schema integration and synthetic recorder/snapshot IPC integration.
- `make native-check` with the existing isolated Rust/Cargo caches: passed sidecar build, Rust formatting, Clippy with warnings denied, Rust tests and Tauri `build --no-bundle`. Six Rust tests passed; two real Keychain ceremony tests were explicitly ignored by the existing suite.
- Frontend regression coverage includes history and payload selection, stale response protection, filtering/paging, cancellation, recording consent, native errors, snapshot capture/branch/failure behavior, replay, analysis, governance and exchange. Added tests cover shell routing/preferences, command search/keyboard/focus, exact clipboard content/failure, restore cancellation/confirmation and empty workspace navigation.

Existing assertions were preserved. Two analysis tests now query the heading's accessible name because a status badge splits its text across elements. The recording test now navigates to capture before using its controls. No failing test was deleted.

## Visual verification and blockers

**Visually inspected:** actual local Vite browser overview and capture views, including dark surfaces, sidebar, form layout, empty history and honest browser/native-IPC error. Overview heading was reduced after review to keep the engineering workspace compact.

**Not visually accepted:** populated timeline/inspector, snapshot confirmation, all populated evaluation/governance/trust results, exact target viewport sizes and native dialog behavior. Component tests exercise these behaviors but are not screenshot evidence.

During continued review, Chrome automation exposed only browser chrome/tab controls; page content disappeared from its accessibility tree and screenshot capture returned a flat blank surface even after reload/address navigation. The native automation API returned `Invalid app` for the freshly built standalone `target/release/chronosync-desktop` executable. No new bundle was created to work around this, as requested. Earlier browser observations are partial evidence only. Native GUI acceptance is unverified.

### Manual visual-review checklist

Use the existing authorized development environment, with disposable synthetic evidence. Do not use the submitted DMG as a newly rebuilt candidate.

1. At 1440×900, 1280×800 and 1024×768, visit all nine destinations; check scroll containment, long paths/hashes, no clipped controls and sidebar collapse/expand persistence.
2. Open the palette with Cmd/Ctrl+K; search, Tab through results, navigate, cancel with Escape and confirm focus restoration. Check visible focus and a screen reader's dialog name.
3. Select a recorded run and event; change filters/pages, resize the split using pointer and keyboard slider, copy IDs/payload, and verify empty/loading/error/truncation states against actual engine data.
4. Capture a disposable snapshot with explicit consent. Cancel restoration and verify no new workspace; then confirm restore/branch, inspect recorded state, source ancestry and managed paths.
5. Review fixture evaluation/mismatch, analysis missing evidence/divergence, policies and historical reports. Verify the baseline/target layout and findings do not imply causality or whole-run determinism.
6. Review benchmark compatibility, lineage, retention preview, references and confirmed deletion on disposable data. Review exchange signature/trust/revocation separately, native confirmation cancellation and custody errors. Do not treat a hash badge as authenticity.
7. Check security/settings remains explicit about unavailable/unassessed release behavior. No positive release acceptance can be inferred from this UI or the successful build.

## Files and limitations

Milestone changes: `apps/desktop/src/{Workbench.tsx,Workbench.test.tsx,App.tsx,App.test.tsx,style.css,Snapshots.tsx,Snapshots.test.tsx,Replay.tsx,Analysis.tsx,Analysis.test.tsx,Governance.tsx,Exchange.tsx}`, the default/minimum window dimensions in `apps/desktop/src-tauri/tauri.conf.json`, this report/plan/status and documentation index/changelog/architecture links. Other dirty files predate this milestone.

Responsive rules collapse navigation and reorganize panels at narrower widths; the default native window is 1280×800, minimum 760×600. Exact-size visual acceptance remains open. Payload rendering is formatted plain text, without an added syntax-highlighting dependency. Security/settings is a boundary explanation, not a new capability inspector. No known automated functional regression remains; uninspected layouts may still contain visual defects and must not be described as fully accepted.

Protected `ChronoSync_0.0.0_aarch64.dmg` SHA-256 remains `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`, matching the pre-overhaul value.
