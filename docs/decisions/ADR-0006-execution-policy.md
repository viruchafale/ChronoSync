# ADR-0006: Requested policy, effective restrictions and capability evidence

## Phase 4A implementation scope — explicitly authorized 2026-10-22

The user subsequently authorized the policy core and conservative supervisor integration. The historical design-only status below applies to the original milestone; it no longer blocks this narrow implementation. `internal/policy` implements versioned mandatory dimensions and exact evidence matching, explicit time/provenance and fail-closed outcomes without a backend. Internal `supervisor.Spec.Policy` requests are always refused before process setup; nil preserves legacy execution without claiming policy verification. No new launch authority, wire format, schema, dependencies or positive provider is added.

Exact-match evidence avoids inventing safe partial-order comparisons between restrictions. All dimensions are required explicitly, and unsupported allowlists/allow-all modes refuse rather than broaden access. Narrow restrictions and units are documented in [Phase 4A status](../IMPLEMENTATION_STATUS_PHASE_04A.md). Workflow inheritance, durable approval/revocation, admission history and permitted weaker policies remain design gates; no extra approval outcome is fabricated. Synthetic evidence tests are not platform certification. Existing event diagnostics and sandbox metadata are insufficient durable admission records, so persistence and UI integration remain deferred pending focused approval.

Status: proposed design, 2026-10-22; no backend selected or implemented. User authorization covers design only.

## Evidence and decision

Current StartSpec passes read-only/workspace-write to Codex and has explicit consent/timeout/exclusions. The POSIX supervisor bounds output and ordinary process lifetimes; the workspace/snapshot code constrains engine-mediated operations. Neither establishes a general child filesystem, network, credential or resource boundary. Windows recording and private storage remain blocked. No adversarial platform certification exists.

Separate RequestedPolicy, EnforcementCapability, EffectivePolicy and PolicyObservation. Resolve each dimension and refuse unmet required enforcement before spawn. A workflow sets defaults and a ceiling; every run needs its own bound policy/approval. Any permitted weakening is a new request with visible differences and explicit approval, never a silent downgrade. Persist sanitized admission evidence and invalidate approval when the capability/workspace/executable identity changes. No detected violation is not proof of compliance.

Reject a single sandbox boolean, automatic best-effort fallback and backend selection by platform name alone. They conceal meaningful differences and drift. Prefer a pure Go resolver with explicit unsupported/unknown outcomes first, then separately approved backend probes. This increases initial UX and test work but makes limitations reviewable. No new framework/dependency is justified now.

## Open selection and evidence requirements

Investigate supported macOS mechanisms/distribution compatibility and Linux namespace/seccomp/resource controls first, with Windows lifecycle/ACL prerequisites and containers as optional comparisons. These are research recommendations, not guarantees of availability. Do not choose a macOS helper/API, install a container runtime, alter privileges or execute an escape experiment without separate authorization. The [capability matrix](../SECURITY_MODEL.md) specifies conditional coverage and evidence gaps.

Approval is required before policy contracts/runtime changes, any backend or platform experiment, privileges/dependencies, user-facing security labels, durable audit storage, and credential/network grants. Adversarial tests must address files, network/local sockets, descendants, resources, credentials/history and crash uncertainty on every advertised target. Synthetic resolver tests prove decision logic only. Existing consent and process supervision remain unchanged.
