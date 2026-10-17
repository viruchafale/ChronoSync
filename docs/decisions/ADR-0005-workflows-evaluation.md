# ADR-0005: Workflow correlation and immutable evaluation evidence

Status: proposed architecture recorded for the authorized Phase 4 design milestone, 2026-10-22. Runtime implementation and migrations await approval. This supersedes ADR-0001's mandatory Lamport-before-concurrency proposal; ADR-0002 event v2 and ADR-0003/0004 storage contracts remain authoritative.

## Evidence and decision

The current recorder has one active run, event-v2 IDs and per-run sequences; SQLite already stores run/event references and snapshot ancestry. No workflow entity, scheduler or evaluation runner exists. `adapter.Adapter` normalizes provider observations but `model.Event.Validate` currently restricts adapter to Codex. No new framework is needed to catalog multiple recorded runs.

Recommend a first-class workflow catalog with separate versioned membership and provenance relationships. Keep runs and events as-is, using side associations rather than rewriting event envelopes. Parent, handoff and dependency are distinct relationships; unresolved evidence remains unresolved. Tool invocation starts as an event projection. Do not create a table for every conceptual noun. Reject the alternative of replacing event storage or inferring a workflow from shared timestamps/workspace paths: both lose evidence or invent causality. A metadata-only correlation layer is smaller initially but lacks explicit identity, revision and ownership of annotations needed for stable inspection.

Per-run sequence remains authoritative. Cross-run display sorting is noncausal; explicit evidence supports edges. No Lamport clock is necessary for the proposed catalog. A future distributed collector must justify its own ordering requirements and protocol changes. Workflow metadata alone needs no event-v3 migration. Future Claude support does require a separately reviewed adapter identity/schema compatibility change because current contracts are Codex-only; do not claim generic wire compatibility from the Go interface alone.

Evaluation is a separate consumer of frozen evidence manifests and producer of versioned judgments. Existing mutable run summaries and workspace paths are insufficient immutable inputs. Preserve event digests/high waters, snapshot manifest hashes, source provenance, evaluator/configuration/dataset/test-suite identities and incomplete-input reasons. No estimated or absent metric becomes an observed fact. Existing CAS can be reused after a future artifact-format review without modifying existing object bytes.

## Consequences and approval gate

Extra joins, membership revisions and evidence manifests add complexity but preserve the historical record. No orchestration, concurrency or launch authority follows from grouping or a handoff. Future migration tests must prove byte preservation, FK/cycle checks, stable pagination, interrupted transaction behavior and old-binary refusal. Evaluate performance before adding a global event index. Phase 1–3 GUI gates remain open.

The [decision register](../phases/PHASE-04-SANDBOX-GUARDRAILS.md) records alternatives, security/compatibility implications, missing evidence and approvals. Proposed transaction/recovery details are in [data model](../DATA_MODEL.md); evaluation semantics are in [system design](../SYSTEM_DESIGN.md).
