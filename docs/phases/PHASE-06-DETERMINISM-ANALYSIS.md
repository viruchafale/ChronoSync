# Phase 06 — Determinism and advanced analysis

## 1. Objective and user-visible outcome

Explain remaining nondeterminism and inspect configurable code-change/workflow findings with evidence.

## 2. Problem

Fixed seeds, wall clocks and network logs cannot independently reproduce concurrent opaque model executions.

## 3. Scope and non-goals

Supported PRNG seed controls, Lamport concurrency analysis, Tree-sitter checks, stuck/runaway heuristics and reproducibility manifests. No hidden model RNG access or perfect semantic analysis.

## 4. Prerequisites and dependencies

P5 scoped replay; event clock fields from P1/P2; P4 budgets; approved grammar/runtime licensing and compatibility review.

## 5. Design and component boundaries

Instrumented workers implement seed/clock interfaces; analysis reads immutable evidence and produces versioned findings without mutating history. Uncontrolled sources remain listed.

## 6. Ordered implementation tasks

1. Inventory random/time/environment/scheduling/provider sources and mark controllability per adapter/tool.
2. Add explicit seed injection for owned test tools and supported runtimes; record generator/version/seed without weakening cryptographic IDs.
3. Implement concurrent producer Lamport propagation and observed causal edges, preserving unknown causality.
4. Integrate pinned Tree-sitter grammars with incremental diff analysis and parser-error reporting.
5. Add configurable file-count/growth/repetition/scope rules and user override/false-positive explanations.
6. Add repeated-tool/stall detectors with bounded windows and distinct heuristic versus confirmed termination reasons.
7. Expose reproducibility manifest and comparative findings; record changed environment and unsupported controls.
8. Benchmark parser/analysis on large diffs and run controlled/uncontrolled repeated-execution experiments.

## 7. Expected files/modules

engine/internal/{determinism,clock,analysis,supervisor}, contracts/, tests/fixtures/analysis/, apps/desktop/src/analysis/.

## 8. Interfaces and contracts

ReproducibilityManifest, SeedControl capabilities, decimal Lamport counters, Finding(rule,version,path,evidence,severity), stall reason taxonomy.

## 9. Security and failure considerations

Never seed security randomness or claim model seed control absent supported API. Parse untrusted files with size/time limits. Findings do not prove malware or semantics.

## 10. Unit, integration and end-to-end tests

Unit: Lamport max-plus-one, overflow, PRNG scope and rule thresholds. Integration: concurrent schedules with causal messages, unsupported generators, malformed grammars/files and false-positive fixtures. E2E: compare repeated scoped fixture runs and inspect honest uncontrolled-source report.

## 11. Acceptance criteria

All observed causal edges increase Lamport values; unrelated timestamps create no edges; supported seeded tools reproduce scoped outputs; uncontrolled sources remain visible; rules expose evidence and configuration; silence never labeled confirmed deadlock.

## 12. Validation commands

```sh
make check
npm run desktop:build
```

These are existing baseline commands. Add phase-specific test suites to these targets and an explicit opt-in live or platform test command where needed before claiming this phase passes. Baseline success alone does not satisfy the phase acceptance criteria.

## 13. Risks and unresolved questions

Tree-sitter language/runtime selection, grammar licensing, platform RNG differences, model nondeterminism and overaggressive stall heuristics.

## 14. Exit criteria

Controlled-source experiments, concurrent ordering tests, rule false-positive suite and reproducibility disclosure are reviewed and pass.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`NOT_STARTED`

No implementation or acceptance tests for this phase have been completed. Await authorization after preceding phase review.
