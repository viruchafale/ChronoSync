# Phase 6 completion report

## Delivered architecture

Phase 6 adds a separate analysis path over existing evidence:

1. Observation v1 derives bounded hash-only evidence and explicit gaps from integrity-checked event history and snapshot metadata.
2. Comparison aligns observable event identities and emits stable metrics, first divergence, findings and limitations without causal inference.
3. Experiments retain 2–20 individual observations/comparisons and derive transparent aggregates.
4. Policy v1 evaluates explicit thresholds and required dimensions; missing/unsupported evidence cannot silently pass.
5. Integrity-tagged append-only private reports preserve observations for historical inspection and new-policy reevaluation.
6. Typed engine/Rust/React boundaries expose comparison, aligned timelines, evidence quality and historical results.

See [ADR-0011](decisions/ADR-0011-determinism-evidence-and-report-persistence.md).

## Comparison and divergence contracts

Comparison statuses are `exact_match`, `partial_match`, `divergent`, `insufficient_evidence` and `unsupported_comparison`. Metrics retain event, order and available-dimension agreement plus explicit counts for aligned, missing, additional, reordered and changed evidence. Workspace agreement is optional rather than invented when no comparable workspace evidence exists.

Findings distinguish missing/additional events, changed fingerprints, reordered interactions, workspace/snapshot differences, configuration differences and unsupported evidence. `first_divergence` is the earliest observed sequence position among findings. It is not a causal root. Every finding carries `causality: unknown`, and the report explains that per-run sequence is not cross-run causal order.

## Policies and historical reports

Policy v1 specifies exact and partial event-agreement minima, a dimension-agreement minimum, maximum missing/additional/reordered/changed counts, required dimensions, and whether unsupported evidence is a failure or incomparable. Each rule records its actual value, required relation and explanation. A report passes only when every rule passes for every retained comparison.

Reports live at `analysis/v1/reports/<id>.json` under private application data. They are canonical, bounded, exclusively created, synced and integrity-tagged. Reevaluation creates a new report from preserved observations. SHA-256 detects modification but is not authentication, origin proof, encryption or a signature.

## Security and privacy limits

Analysis never invokes an agent/provider, uses networking, launches a command, restores a snapshot, changes approval/admission, or authorizes execution. Raw command, output, item ID, workspace path and changed paths are hashed in memory and absent from reports. Fingerprints still reveal equality and can permit guessing low-entropy values; identifiers and metadata remain private data.

ChronoSync does not record hidden model state, controlled provider request/response traffic for event-v2, complete causal clocks or proof that every event used a specific snapshot. Those gaps are explicit. Results do not prove whole-run or LLM determinism, process-wide offline enforcement, causation or cross-platform equivalence.

## Acceptance datasets

Automated tests cover all requested cases: identical observations; one changed output; missing and additional tool interactions; reordered interactions; workspace divergence; incomplete history; unsupported evidence; multiple observations with retained individual comparisons; and restart persistence. The IPC workflow uses disposable stored event histories and no live provider or command.

## Verification

| Command or check                                                             | Result                                                                                                                                                                                                                   |
| ---------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Focused `go test -race`/`go vet` for analysis, IPC and engine command        | Passed evidence extraction, all ten datasets, thresholds, persistence/integrity, strict IPC, revised-policy and restart tests.                                                                                           |
| `GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off make check`            | Passed Go format/vet/all race tests/build, Prettier, ESLint, strict TypeScript, 6 Vitest files/26 tests, frontend production build, documentation, version integration and synthetic recorder/snapshot integration.      |
| Project-scoped offline `make native-check` with locked Cargo and Rust 1.99.1 | Passed sidecar build, Rust formatting, Clippy with warnings denied, real-engine capability/bridge test, frontend build and optimized native executable. No bundle command ran.                                           |
| Recursive Phase 6 source-boundary audit                                      | Passed: no provider adapter, HTTP, process launcher, supervisor, admission or policy import in `engine/internal/analysis`; storage/snapshot interfaces and private report files are the intentional evidence boundaries. |
| Submitted DMG SHA-256                                                        | Unchanged at `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`; it was not rebuilt.                                                                                                                     |

The first portable verification attempt stopped on formatting in the newly added Markdown files. After formatting, the next attempt exposed the documentation checker’s required 16-section/status shape for the Phase 6 phase plan. The implemented plan was converted to that established structure, and the final full gate passed. No test or checker was weakened.

## Remaining limitations and Phase 7 direction

Automated native window interaction, clean-machine behavior, Windows/Linux runtime behavior, signed/notarized packaging and OS-level egress enforcement remain unverified. Phase 7 should be separately authorized and should begin with an evidence-retention and benchmark-governance design. Any executable evaluator, live provider comparison, automatic rerun or external metric service is a new trust/execution boundary and must not be inferred from Phase 6.
