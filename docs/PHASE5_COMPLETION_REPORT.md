# Phase 5 completion report

## Delivered architecture

Phase 5 now has four separated layers:

1. Phase 5A exports, validates and compares inert command-completion fingerprints.
2. Phase 5B records and deterministically matches bounded synthetic transport fingerprints with no network/launcher dependency.
3. Phase 5C stores validated canonical bytes as immutable private content-addressed fixture files without a SQL migration.
4. Phase 5D coordinates a fresh matcher and Phase 5A comparison from a versioned inert plan; Phase 5E exposes the lifecycle through typed engine/Rust/React boundaries; Phase 5F verifies the complete disposable workflow and restart.

`fixtures/v1/sha256/<sha256>.fixture` is authoritative. Inspection metadata is derived from strict content loaders. IPC never receives a fixture path. Tauri owns native file dialogs and trusted export/delete confirmation. See [ADR-0010](decisions/ADR-0010-fixture-lifecycle-and-native-consent.md).

## Implemented capabilities

- Determine whether a completed execution is eligible and create a stored Phase 5A fixture.
- Import either supported fixture format, validate canonical bytes/integrity, deduplicate by content hash, list, inspect details/warnings, export with native review/confirmation and delete with native confirmation.
- Refuse traversal identities, unsafe permissions/files, unsupported versions/fields, noncanonical/tampered/oversized bytes, incomplete history and failed evaluations.
- Match repeated synthetic requests with explicit occurrences and typed unmatched/ambiguous/invalid/unsupported/cancelled outcomes.
- Compare controlled output fingerprints and exit codes with stable field-level mismatch results, bounded timeout and fresh matcher state per evaluation.
- Show eligible execution controls, fixture inventory/details, import/export/delete, editable versioned evaluation plan and pass/fail details in the desktop UI.
- Preserve fixtures across engine/store restart; expose corrupted regular entries as damaged records, refuse their inspection/export/evaluation without repair, and permit only explicit confirmed deletion.

## Formats, dependencies and compatibility

Phase 5A fixture v1, Phase 5B transport fixture v1, event v2, snapshot manifest v1 and SQLite migration 3 are unchanged. Engine 0.4.0 adds `replay-fixtures-v1` to protocol 1. New TypeScript types describe fixture/evaluation IPC. Tauri’s official dialog plugin is the only new product dependency; it is used by the host and no renderer filesystem capability is enabled.

The submitted DMG was not rebuilt or modified. Existing Phase 1–4 recorder, timeline, snapshot, policy, admission and conservative launch behavior remains intact.

## Security and privacy limits

Fixtures contain unkeyed hashes that reveal equality and permit guessing low-entropy values. Redaction remains heuristic. Phase 5B refuses arbitrary bodies, URLs and credential metadata; accepted token-shaped values still require review. Digests are integrity identifiers, not signatures, encryption, authenticity or origin proof.

Evaluation never invokes a provider, HTTP/MCP transport, historical command, supervisor, admission or approval. The isolated synthetic transport dependency graph has no network or process-launch route. The wider desktop process is not in an OS egress sandbox, so Phase 5 does not prove process-wide offline enforcement or deterministic LLM execution.

Native dialogs grant user-selected file access. Import treats bytes as untrusted and bounds/validates them in both host and engine. Export uses exclusive creation and will not overwrite an existing file. Automated native dialog interaction, Windows/Linux runtime behavior, signed/notarized packaging and clean-machine fixture exchange remain unverified.

## Verification

| Command or check                                                                                         | Result                                                                                                                                                                                                     |
| -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Focused `go test -race` and `go vet` for fixtures, replay/transport/unified and IPC                      | Passed lifecycle, compatibility/refusal, matching, unified result and disposable restart/corruption workflow tests.                                                                                        |
| `GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off make check`                                        | Passed Go format/vet/all race tests/build, Prettier, ESLint, strict TypeScript, 5 Vitest files/24 tests, frontend production build, docs, version integration and synthetic recorder/snapshot integration. |
| Project-scoped offline `make native-check` with Rust 1.99.1, locked Cargo and no global toolchain change | Passed sidecar build, Rust formatting, Clippy with warnings denied, real-engine bridge test, frontend build and optimized native executable. No bundle command ran.                                        |
| `go list -deps ./internal/replay/transport` forbidden dependency audit and recursive source test         | Passed: no network, `os/exec`, storage, adapter, supervisor or third-party production dependency in the isolated synthetic transport package.                                                              |
| Phase 5F `TestSyntheticRegressionWorkflowPersistsAcrossRestart`                                          | Passed in-memory recording, 5A creation, 5B import, inspection, pass/mismatch/cancellation, export/delete/re-import, restart availability and corruption refusal.                                          |
| Submitted DMG SHA-256                                                                                    | Unchanged at `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`; the submitted artifact was not rebuilt.                                                                                   |

The prior missing-`proc_macro` blocker was an incomplete temporary toolchain. A complete project-scoped Rust 1.99.1 toolchain under `/private/tmp` resolved it without changing global settings. Native source/build/bridge verification now passes. Actual macOS window and open/save/confirmation interaction remains unautomated, so component/build evidence is not represented as a native GUI walkthrough.

## Phase 6 recommendation

Phase 6 should analyze determinism evidence from these controlled results without changing replay into live rerun. Start with metric/provenance contracts and repeated synthetic datasets. Seed control, model determinism claims, OS-level offline enforcement and live provider comparison require separate capability evidence and authorization. Do not infer reproducibility from a passing fingerprint fixture.
