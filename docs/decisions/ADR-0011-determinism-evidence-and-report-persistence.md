# ADR-0011: Observable determinism evidence and report persistence

Status: accepted for Phase 6A–6F, 2026-10-31.

## Context

ChronoSync records ordered event v2 history, bounded typed payloads and optional committed workspace snapshots. Phase 5 adds inert command and synthetic transport fixture comparisons. None of those sources expose hidden model state, complete provider traffic, distributed logical clocks or a causal account of why an execution differed. Re-executing history would violate the existing replay boundary.

Phase 6 needs repeated comparison, divergence analysis, explicit policies and durable historical reports while preserving original observations and existing database/fixture formats.

## Decision

Create a pure Go `internal/analysis` domain. Observation format v1 freezes only evidence ChronoSync can read: execution identity and high water, per-run event order/type, hashes of available input/output/tool/workspace values, committed snapshot manifest identities, bounded non-secret configuration metadata, and explicit missing/unsupported evidence. Raw command, output, item, workspace path and changed path strings are hashed in memory and absent from reports.

Comparison aligns bounded event identities and reports exact, partial, divergent, insufficient or unsupported results. It retains individual alignments, metrics and findings for missing, additional, reordered and changed evidence. Every finding labels causality `unknown`; per-run sequence is observed order only. Multiple observations remain intact beneath aggregate metrics.

Evaluation policy v1 explicitly names thresholds, maximum divergence counts, required dimensions and unsupported handling. Missing required evidence cannot pass. Reevaluation creates a new report that references copied preserved observations; it never rewrites event history or an earlier report.

Store canonical integrity-tagged reports under private application data at `analysis/v1/reports/<random-id>.json`. Use private temporary files, sync, no-replace publication, count/size/total limits, strict canonical decoding and a SHA-256 integrity field. Interrupted temporary files are ignored and retained. This is integrity/tamper detection, not authentication or a signature. No SQLite migration is added.

Expose additive protocol-1 methods behind `determinism-analysis-v1`. IPC accepts execution identities and a typed policy, never a filesystem path or executable content. Tauri forwards typed values and adds no permission. The renderer displays actual backend results only.

## Consequences

Phase 6 can compare and retain observable evidence without commands, agents, providers, networking, snapshot restoration, admission or launch authorization. Results are stable for identical stored evidence and an explicit policy. They do not prove LLM determinism, process-wide offline enforcement, causal relationships or cross-platform equivalence.

Fingerprints reveal equality and may permit guesses of low-entropy values. Private reports still contain execution/event identifiers, configuration labels and hashes. Missing, truncated, unsupported or corrupt evidence lowers quality or makes a policy incomparable; it is never silently treated as a match.
