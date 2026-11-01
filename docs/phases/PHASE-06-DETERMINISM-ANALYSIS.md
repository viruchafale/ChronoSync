# Phase 06 — Observable determinism analysis and evaluation

## 1. Objective and user-visible outcome

Compare evidence ChronoSync actually recorded, explain observed divergence, aggregate repeated observations, apply explicit thresholds and retain historical reports without executing history or claiming hidden model determinism.

## 2. Problem

Recorded events and snapshots expose only bounded observations. They do not expose hidden model state, complete provider traffic, distributed logical clocks or causal explanations. Missing evidence must remain explicit.

## 3. Scope and non-goals

Implemented scope includes hash-only observations, repeated comparisons, structured divergence, policies, historical reports and a desktop analysis view. Non-goals include command/provider/agent execution, network fallback, snapshot execution, authority changes, hidden-state inference and causal claims.

## 4. Prerequisites and dependencies

The implementation consumes existing event v2 history, run v1 metadata and snapshot v1 identities through repository ports. It preserves Phase 5 fixture formats and adds no third-party dependency or SQLite migration.

## 5. Design and component boundaries

`engine/internal/analysis` owns observation/comparison/experiment/policy/report format v1. Reports live under private application data at `analysis/v1/reports`. Typed IPC flows through the existing private stdio and Tauri bridge to React. See [ADR-0011](../decisions/ADR-0011-determinism-evidence-and-report-persistence.md).

## 6. Ordered implementation tasks

1. Define evidence/provenance and explicit gaps.
2. Retain repeated observations and individual comparisons.
3. Detect first, missing, additional, reordered, changed and workspace divergence.
4. Expose typed desktop comparison/history behavior.
5. Apply explicit policies and persist integrity-tagged reports.
6. Verify ten disposable scenarios and restart.

All six tasks are implemented.

## 7. Expected files/modules

Implemented files are under `engine/internal/analysis`, `engine/internal/ipc`, `contracts/analysis.ts`, `apps/desktop/src/Analysis.tsx`, the Tauri command host, Phase 6 status/report documents and ADR-0011.

## 8. Interfaces and contracts

Statuses are exact, partial, divergent, insufficient and unsupported. Policies carry versioned thresholds, maximum divergence counts, required dimensions and unsupported handling. Reports preserve observations, comparisons, aggregates, policy rules and a SHA-256 integrity field.

## 9. Security and failure considerations

Raw supported values are hashed in memory and omitted from reports, but fingerprints reveal equality and may be guessable. Unknown/truncated/corrupt evidence cannot satisfy required dimensions. Report integrity is not authenticity. No comparison result authorizes execution.

## 10. Unit, integration and end-to-end tests

Pure tests cover identical, changed-output, missing, additional, reordered, workspace-divergent, incomplete, unsupported and multiple-observation datasets. IPC tests cover real stored history, policy revision and restart. React tests cover selection, aligned divergence, history and reevaluation.

## 11. Acceptance criteria

All ten requested scenarios produce stable explainable results; original observations remain available; missing required evidence never passes; restart preserves reports; renderer inputs are revalidated; no live boundary is invoked.

## 12. Validation commands

```sh
GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off make check
CARGO_HOME=/private/tmp/chronosync-phase5-cargo RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup CARGO_NET_OFFLINE=true make native-check
```

Actual results are recorded in the [Phase 6 completion report](../PHASE6_COMPLETION_REPORT.md).

## 13. Risks and unresolved questions

Automated native window interaction, OS-wide offline enforcement, cross-platform runtime behavior, signed packaging, benchmark governance and any live/executable evaluator remain unresolved and outside Phase 6.

## 14. Exit criteria

The domain, persistence, IPC, UI, ten datasets, full portable/native checks and documentation must pass while the submitted DMG remains unchanged.

## 15. Documentation updates

Separate 6A–6F status files, the status index, architecture, system/data/security/testing documents, changelog, contracts, ADR-0011 and the consolidated report record implemented evidence and limitations.

## 16. Status

`COMPLETE`

Phase 6A–6F are complete under the authorized observable-evidence boundary. Phase 7 requires separate authorization.
