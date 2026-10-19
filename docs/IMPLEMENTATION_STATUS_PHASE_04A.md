# Phase 4A implementation status — execution-policy core

Status: COMPLETE for the authorized core and conservative refusal boundary; no enforcement backend or positive admission. This is a separate milestone from the [Phase 4 design report](IMPLEMENTATION_STATUS_PHASE_04.md). No Phase 1–3 acceptance gate is closed by this work.

## Authorization and preflight

The user explicitly authorized a typed deterministic policy core, conservative capability reporting, safe supervisor integration and focused tests. This supersedes the earlier design-only restriction for this slice. No backend, positive launch permission, migration, UI security label, workflow or evaluator is authorized. Existing required/observed distinctions and fail-closed design in ADR-0006 are retained; no optional preference or weaker-policy approval flow is invented.

Audited actual supervisor, Codex adapter, recorder consent/timeout, IPC request parsing, Rust typed commands/serialized bridge, React consent/policy controls and existing tests. Current formats remain engine 0.3.0, protocol 1, event 2, run 1, manifest 1 and SQLite migrations 1–2. The adapter passes Codex read-only/workspace-write flags; those flags and cwd do not prove independent enforcement. Existing POSIX cancellation uses process groups with a 500 ms grace, output is bounded at 8 MiB with 128 KiB lines, and recorder timeout is 1–3600 seconds. Windows execution remains blocked.

The workspace still has no Git metadata; inspected files directly. Go 1.26.2 darwin/arm64, Node 22.14.0 and npm 10.9.2 are available; the existing temporary Rust toolchain is used for native checks. No dependencies installed or host settings changed.

## Implemented core

`engine/internal/policy` introduces internal Go version-1 RequestedPolicy, EnforcementCapability, EffectivePolicy, Decision/Reason and evidence types. This is not a new wire contract or persistent schema. Resolve consumes supplied capabilities, target and evaluation time without reading the host, launching processes or reading a clock.

Every supported dimension must be explicit and mandatory. Missing, unknown, duplicate/conflicting or malformed requirements refuse; there is no default policy and no allow-all mode. Supported syntax is deliberately narrow:

- Workspace: explicit read-only or read-write, scoped by an opaque workspace identity digest.
- External files, network, new child processes, credentials, host services and historical app data: deny only. Process denial refers to new descendants, not the initial agent; process-count limits include that initial execution. General path, destination and executable allowlists remain unimplemented.
- Environment: sorted unique uppercase variable names, at most 64, with no values; an explicit empty list means no inherited variables.
- CPU milliseconds, memory/disk/output bytes, total process count, wall timeout and descendant cleanup milliseconds: positive signed-int64 limits, each requiring exact evidence. Units are encoded in dimension names; this core does not convert them into OS controls.
- Human approval and audit: required. Positive synthetic evidence can exercise resolution, but production has no approval/audit provider and cannot admit a launch.

Evidence includes version, platform, backend/version/configuration identity, exact requested scope/restriction, availability, enforcement level, source identifier, evidence digest, verification flag, validity interval and bounded typed limitations. No arbitrary evidence text, environment values, paths or credentials are collected. Provider input is a trusted in-process boundary, not authenticated merely by a digest/boolean and never accepted through IPC/UI. Evidence digests provide identity, not proof of truth.

Availability separates available, unavailable, unknown and unsupported; verification is independent. Only available, verified, current, fully enforced, limitation-free evidence matching the exact target, workspace/executable identities and restriction can establish a dimension. Observation/advice/configuration do not satisfy required enforcement. Even a differently bounded purportedly stronger restriction is conservatively refused rather than applying unreviewed subsumption rules. Canonical request hashing binds ID, scope and ordered restrictions; results copy input slices to avoid accidental mutation. Returned partial establishment never changes the refused outcome.

Decision outcomes are requirements_satisfied or refused. Requirements_satisfied is a deterministic assessment of supplied evidence, **not a launch token**. No additional-approval or downgrade outcome is implemented because its durable identity/revocation and permitted weakening rules remain unapproved. Refusal codes have fixed actionable messages without echoing input values. ConservativeProvider reports unknown/advisory/no_trusted_provider for all supplied dimensions, on every target; it performs no discovery/probes.

## Supervisor, UI and persistence boundary

`supervisor.Spec.Policy` is an internal opt-in request. `supervisor.Run` checks it before platform setup, pipes, exec.Command or process start, and always returns a typed PolicyError using the conservative provider. There is intentionally no positive policy-launch path and no caller-injected positive provider at the supervisor boundary. Unknown evidence refuses before launch. This establishes refusal behavior, not containment.

A nil policy retains existing legacy execution semantics. Codex and recorder wire requests do not populate this field; current launches remain subject to existing consent/upstream flags/timeouts/output supervision, not the new policy guarantee. There is no UI toggle or policy badge, and no IPC capability is advertised. The internal guard cannot be claimed to enforce policies on legacy launches. Binding actual executable/workspace identity at spawn, trusted backend evidence, approval freshness and durable admission are prerequisites to positive integration.

Existing run metadata has only the upstream sandbox label and consent is not a durable bound admission. Existing event-v2 diagnostic text cannot faithfully represent structured request/effective/evidence records, and refusing before run creation should not manufacture a run/event. Consequently this slice keeps decisions transient: no persistence overload, migration, event field or format change. Durable admission needs a focused proposal covering versioned policy records, run linkage, canonical identities, revocation, atomic pre-spawn audit and recovery; approval is required before implementing it. Historical records are unchanged.

## Verification evidence

- `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod go -C engine test -race ./internal/policy ./internal/supervisor ./internal/recorder`: passed. Includes synthetic satisfied/refused evidence, unknown/unavailable/unsupported, malformed/conflicting configurations, wrong scope/platform/backend/configuration, unverified/expired/future evidence, missing provenance, observed-only/partial coverage, missing dimensions, forbidden widening and budget/env mismatch, deterministic order/hash and detached result copies.
- Supervisor tests pass a synthetic marker-writing child executable to the real Run entry point with valid and malformed policies: both refuse with no child marker/output. This is actual pre-spawn refusal, not a mocked process success or OS isolation experiment.
- Existing supervisor and recorder regression tests remain intact and pass, including consent, output bounds, timeout and cancellation. No tests weakened or removed.
- `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod make check`: passed Go format/vet/race tests/build, Prettier, ESLint, strict TypeScript, 22 frontend/contract tests, Vite build, documentation checks, version/schema integration and synthetic recorder/timeline/snapshot integration. Log: `/tmp/chronosync-phase4a-check.log`.
- `CARGO_NET_OFFLINE=true GOPROXY=off GOSUMDB=off PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod make native-check`: passed sidecar build, Rust fmt/locked Clippy, one real-engine bridge test and native release build. Log: `/tmp/chronosync-phase4a-native.log`. Native GUI not tested.
- `go -C engine test ./internal/policy -cover` with offline/cache variables: passed, 79.5% statement coverage at that point. This is not security certification. Subsequently added malformed capability metadata and environment validation tests; targeted `go -C engine test -race ./internal/policy` passed after those additions.
- Final documentation formatting/checks passed after the report update. No Mermaid changes. Source hash comparison with the prior recorded inventory showed only the existing supervisor process source changed, plus its generated packaged engine binary from the required build. New source files are the policy package and supervisor policy guard/tests. Existing wire contracts, metadata formats, migration sources and dependency lockfiles are unchanged.

## Limits and next gate

No security backend, full sandbox, credential isolation, resource enforcement, positive admission, durable policy history, workflow ceiling inheritance, weakened-policy approval flow or UI integration ships. No live provider calls, privileged probes or network research occurred. Phase 1–3 native GUI walkthroughs and earlier platform/fault-test gaps remain unverified; a native build does not close them.

Stop at this milestone. There is no automatically approved next implementation. The next review should settle durable admission/approval identity and the precise launch-binding contract, then separately authorize either that implementation or a scoped platform feasibility investigation. Do not wire synthetic positive evidence into production or treat the conservative provider as a backend.
