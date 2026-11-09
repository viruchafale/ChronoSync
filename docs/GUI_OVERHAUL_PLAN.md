# Developer workbench overhaul plan

This separately authorized frontend milestone redesigns existing Phase 1–11 capabilities. It is not Phase 12. Preserve Tauri/React/Go, all IPC contracts, consent gates, synthetic replay boundaries and the submitted DMG. Do not bundle or publish.

## Screen inventory and strategy

| Screen / existing component        | Existing behavior and data source                                                         | Original UI problem                            | Implemented strategy                                                                                      |
| ---------------------------------- | ----------------------------------------------------------------------------------------- | ---------------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| Overview / App                     | RecorderAPI.queryRuns, history paging and run selection                                   | History buried in one long page                | Dedicated searchable execution list, status badges, empty state and capture entry                         |
| Capture / App                      | RecorderAPI.start/cancel; explicit consent and policy fields                              | Form competes with unrelated tools             | Dedicated compact configuration page; original warnings and consent retained                              |
| Timeline / App                     | RecorderAPI.query/detail; paging, search, kind/sequence filters                           | Payload and list read as one vertical document | Split event navigation/inspector, selected state, adjustable width, metadata and exact payload copy       |
| Snapshots / Snapshots              | SnapshotAPI capture/list/validate/materialize/branches/inventory                          | Long mixed controls and raw IDs                | Section navigation, artifact badges/copy, explicit restore/branch confirmation and ancestry               |
| Replay / Replay                    | ReplayAPI library/import/export/evaluate/delete                                           | Fixture selection and results lack hierarchy   | Section navigation, consistent forms/lists/status and copyable fixture IDs; synthetic boundaries retained |
| Compare / Analysis                 | AnalysisAPI experiments/comparison/policies/report history                                | Baseline and targets stacked without grouping  | Side-by-side setup, report status and identifiers, structured existing findings                           |
| Benchmarks / Governance            | GovernanceAPI dataset, compatibility, lineage, retention/audit/delete                     | Multiple operations difficult to find          | Section navigation and consistent compact controls; preview/reference/confirmation evidence retained      |
| Exchange / Exchange                | ExchangeAPI trust/identity/custody/import/export                                          | Dense trust and package operations             | Grouped navigation, trust badges, copyable keys, original native confirmations and diagnostics            |
| Security / Workbench settings view | Static documented implementation boundaries; actual custody operations remain in Exchange | No consolidated explanation                    | Explicit unavailable/unassessed states; no fabricated runtime attestation                                 |

## Work and verification

- [x] Inspect existing tests, components, bindings, tools and Git state; preserve existing dirty work.
- [x] Establish shared CSS tokens, shell, navigation, palette, badges and copy controls without dependencies.
- [x] Apply layout and hierarchy across every existing feature component.
- [x] Preserve existing operations and add restore/branch confirmation.
- [x] Test navigation, palette, clipboard, timeline, consent, errors and feature regressions.
- [x] Run portable and no-bundle native validation; record final results in the status report.
- [x] Inspect browser overview and capture rendering.
- [x] Complete browser visual review of populated views at 1440×900 and 1280×800 and the compact 1024×768 layout; retain honest screenshot/blocker classifications in the separate refinement report. Native dialogs remain outside this pass.

Follow-up evidence: [UI visual refinement report](UI_VISUAL_REFINEMENT_REPORT.md) and [status](IMPLEMENTATION_STATUS_UI_VISUAL_REFINEMENT.md).

Final browser closure: [GUI acceptance closure report](GUI_ACCEPTANCE_CLOSURE_REPORT.md) and [status](IMPLEMENTATION_STATUS_GUI_ACCEPTANCE_CLOSURE.md). Security now passes at all three target viewports; native macOS dialogs remain manual.

Risks: hidden feature panels remain mounted to preserve form state; snapshot selection is scoped to the selected run. Browser preview cannot access native IPC. Renderer status is the last recorder response, not continuous engine health or signed-release acceptance. HTML dialogs add UI confirmation only; backend consent/authorization remains authoritative.
