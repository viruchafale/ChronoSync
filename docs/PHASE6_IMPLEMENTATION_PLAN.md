# Phase 6 implementation plan — determinism evidence and evaluation

## Authorization and boundary

The user authorized continuous implementation of Phase 6A–6F. Phase 6 compares bounded observations that ChronoSync already recorded or that tests construct synthetically. It never re-executes a command, invokes a provider or agent, opens a network transport, restores a workspace for execution, changes admission/approval, or claims access to hidden model state. A comparison is evidence about observable records, not proof of deterministic LLM execution or causation.

Existing uncommitted Phase 5 work, fixture formats, event v2, snapshot manifest v1, SQLite migration 3, provider/admission/supervisor behavior and the submitted DMG remain preserved. Phase 7 is outside this authorization.

## Repository findings

- Integrity-checked event history exposes per-run sequence, typed payloads, terminal/history state and a stable high-water mark. Per-run sequence is the only causal ordering ChronoSync can assert.
- Event payloads can provide command/tool identifiers, command output and workspace changes, but fields may be absent, redacted or truncated. Timestamps are display evidence only.
- Committed snapshots expose manifest identities through the existing snapshot repository. A snapshot is an observed workspace capture, not proof that every event occurred against those bytes.
- Phase 5 fixtures and unified evaluation are deliberately inert and remain separate. Phase 6 consumes neither executable text nor a live fallback.
- The desktop already has run selection, timelines and typed stdio/Tauri bridges. Phase 6 can add typed analysis methods without filesystem paths or a new renderer capability.

## Architecture and versioned contracts

Add a pure `internal/analysis` domain with format version 1:

- `Observation` freezes source execution identity, high water, event ordering, available input/output/tool/workspace fingerprints, committed snapshot manifest identities, bounded non-secret configuration metadata, missing evidence and unsupported evidence.
- `Comparison` aligns observable events without inventing cross-run causality and reports exact, partial, divergent, insufficient-evidence or unsupported status; metrics retain counts and ratios alongside each aligned/missing/additional/reordered/changed item.
- `Experiment` keeps every observation and baseline-to-observation comparison, then derives bounded aggregate metrics without discarding individual evidence.
- `Policy` version 1 explicitly defines exact/partial thresholds, required dimensions, unsupported handling and regression limits. `Evaluation` records every rule outcome; missing required evidence cannot pass.
- `Report` stores observations, comparisons, aggregate, policy and evaluation together so a later policy can be applied to preserved evidence without rewriting execution history.

Add an application service that builds observations from the existing history/snapshot repository and stores strict append-only private JSON reports under `analysis/v1/reports`. Files use generated identities, exclusive private creation, size/count bounds, sync and strict validation on every read. This avoids changing existing SQLite/fixture schemas while still providing restart persistence. An ADR will document the new report boundary.

Engine protocol 1 receives additive `determinism-analysis-v1` methods for compare, report list and report get. The Tauri host only forwards typed values. The React analysis view selects two or more recorded executions, submits an explicit policy, shows aligned evidence, first divergence, metrics, evidence quality and historical reports. Renderer input remains untrusted and is strictly validated by the engine.

## Milestones and intended files

1. **6A evidence/provenance:** `internal/analysis/model.go`, `observe.go` and tests; typed evidence and honest gaps.
2. **6B repeated experiments:** `compare.go`, `experiment.go` and tests; bounded multiple observations and stable aggregate metrics.
3. **6C divergence:** structured alignment, first divergence, missing/additional/reordered/changed/workspace/unsupported findings with uncertainty labels.
4. **6D desktop:** `contracts/analysis.ts`, engine IPC, Rust commands, `Analysis.tsx` and component tests; real backend methods only.
5. **6E thresholds/history:** `policy.go`, private report service and restart tests; explicit explainable decisions and revisable policies.
6. **6F validation:** disposable datasets for all ten requested cases, full integration/restart coverage, documentation and consolidated report.

Each milestone receives its own `docs/IMPLEMENTATION_STATUS_PHASE_06X.md`. The index remains an index. Architecture, data model, security, testing, changelog, contract documentation and `AGENTS.md` will be updated only to reflect verified behavior.

## Limits and risks

- Fingerprints reveal equality and may permit guessing low-entropy values. Reports contain identifiers and metadata and remain private application data.
- Missing/redacted/truncated/corrupt evidence is recorded as unavailable, never converted to zero, equality or a passing score.
- Event alignment is an analysis heuristic. Results say “observed divergence” and never assign cause. Reordering is reported only for aligned event identities; unavailable logical clocks remain an explicit limitation.
- Comparisons, observations, events, dimensions, reports and serialized bytes are bounded. Unsupported versions and unknown fields are refused.
- No new third-party dependency is planned. A material boundary change or any live integration would require a separate ADR/review and authorization.

## Verification plan

Run focused Go race tests and vet after each backend milestone; TypeScript/component tests after desktop integration; a disposable IPC workflow covering identical, changed, missing, additional, reordered, workspace, incomplete, unsupported, repeated and restart cases; `make check`; the isolated analysis dependency audit; locked project-scoped Rust formatting/Clippy/tests/native build; `git diff --check`; and the submitted DMG SHA-256 check. Native GUI interaction, cross-platform runtime behavior and OS-wide offline enforcement will remain unverified unless directly demonstrated.

## Implementation outcome

Phase 6A–6F implementation and focused tests are complete. Observation/comparison/experiment/policy/report format v1, private report persistence, typed IPC/Rust/React integration and all ten disposable datasets are implemented. Final full-suite/native results and unchanged-artifact evidence are recorded in the [Phase 6 completion report](PHASE6_COMPLETION_REPORT.md). No Phase 7 work began.
