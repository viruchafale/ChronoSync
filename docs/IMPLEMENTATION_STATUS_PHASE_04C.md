# Phase 4C implementation status — admission domain contracts

As of 2026-10-23. COMPLETE for internal contracts and pure tests; persistence and positive admission remain unimplemented. This milestone implements pure internal types and tests only. [Status index](IMPLEMENTATION_STATUS.md). The Phase 4A/4B reports and Phase 1–3 acceptance gaps are preserved.

## Preflight, authority and unresolved decisions

Read the status index/reports, Phase 4 plan, system/data/security/testing documents and ADR-0005/0006/0007; inspected actual policy and supervisor code and existing limits. Engine 0.3.0, protocol 1, event 2, run 1, manifest 1, internal policy 1 and migrations 1–2 remain unchanged. Git metadata is absent; git status reports not a repository. A file-hash inventory records existing content before changes.

The user explicitly authorized domain types, canonicalization rules and pure tests. The required ADR-0007 decisions were separated before coding: canonical bytes/bounds and pure sequencing are implemented in this scope; human-authentication strength, private commitment algorithm/key lifecycle, backend-bound filesystem identity and trustworthy epoch generation/restoration detection remain unresolved. The package stops at external references, never implements those production mechanisms or supplies permissive defaults. No new framework/dependency.

## Changed files and contracts

New package `engine/internal/admission`: binding.go, hash.go, state.go, binding_test.go, state_test.go and synthetic fixed vectors/provenance under testdata. Binding copies inputs and returns detached bytes/specs; no external mutation API exists. Admission is a value with private state and copied snapshots; Apply returns a new value or unchanged input with typed errors. No persistence decoder is introduced.

Binding version 1 covers admission/epoch, adapter/configuration, executable path/artifact/object identity limitation, exact public argv, environment value-rule references, private stdin commitment reference or explicit none, cwd/workspace identity/generation/state, policy/effective references, backend target, backend evidence references/expiry and optional parent run. Existing policy Target, Dimension and Outcome types are reused; policy resolution is not duplicated. Unknown/malformed fields, duplicate references, missing backend dimensions and invalid bounds reject.

Exact binary encoding/order, UTF-8/no-normalization rules and limits are in [system design](SYSTEM_DESIGN.md). The SHA-256 domain/version distinguishes binding bytes from the separately domain-separated evidence-set reference. Existing policy hash is unchanged. Fixed fixture: 3,975 bytes; SHA-256 `8023cca6d68e98cb10c801aaebdb4727cad3a88bd9f07491b5604ed76e8d6342`, independently encoded with Python big-endian operations and checked by Go. Fixtures contain synthetic public IDs/digests only.

New bounds: 128 argv entries, 32 KiB per argument, 1,024-byte paths and 256 KiB canonical total. These align with existing recorder text/event and snapshot path budgets where applicable; the argv count is a documented conservative domain bound, not a claimed OS limit. Existing policy grammar/64 environment-name bound and dimension enumeration are reused. No default expiry, production secret commitment or permission profile is selected.

Public strings cannot be proven secret-free by syntax. Inputs must come from a reviewed non-secret launch plan; secret-bearing argv is unsupported. Environment and stdin types have no raw-value field. References do not make a guessable public hash secret: production private commitments remain blocked pending an approved mechanism. No credentials or private prompts were read or stored.

## State and assurance boundaries

Modeled progression: preparing → awaiting_approval → approved → prepared → claimed, with separate pending/not_started/started/outcome_unknown observations. Challenge and approval refs bind manifest/epoch/expiry; receipt is not human authentication. Readiness and each claim/final-validation/attempt stage require matching explicit external assessment/approval references, current evidence, and non-unresolved identity. No production provider emits the external_validation marker.

Strict local order and nondecreasing supplied time prevent backwards transitions; they are not durable or authenticated ordering. Claimed admissions permanently retain attempt identity after failure/unknown/revocation. Recovery with a different epoch invalidates unclaimed states and changes pending claimed outcome to unknown; it never proves no process started. Revoke-before-claim invalidates; revoke-after-claim blocks further attempt and records cancellation-required without pretending cancellation occurred. Repeated terminal observations and unknown versions/states reject.

NewEpoch validates a caller-supplied nonzero identity only. It does not generate secure randomness, a trusted counter or detect restoration. A caller reusing an old epoch can defeat intended invalidation outside the model; future engine/storage/backend mechanisms must address that. All successful state transitions are synthetic domain results, not durable records or launch tokens. Two evaluations of the same prepared value do not implement atomic one-use consumption: future SQLite revisions/constraints must select one accepted claim. No global approval/challenge registry exists.

Existing `internal/policy` and `internal/supervisor` are unchanged. ConservativeProvider remains unknown; every non-nil supervisor policy still refuses. No imported admission package is wired into launch, recorder, storage, IPC, Rust or React. No migration, public contract, runtime capability advertisement, dependency or historical snapshot change.

## Actual verification

- Initial compilation of the new package passed before tests (reported no test files at that stage; not acceptance evidence).
- `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod go -C engine test -race ./internal/admission`: passed canonical fixed vector, mutation/order/copy/bounds/path/Unicode/identity tests, approval/evidence/state/epoch/revocation tests and bounded post-claim action traversal.
- Same offline/cache environment with `go -C engine test -race ./internal/admission ./internal/policy ./internal/supervisor`: passed (cached results reported by Go). Existing conservative refusal regression remains intact; no tests removed or weakened.
- `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod make check`: passed Go formatting/vet/race/build, Prettier, ESLint, strict TS, 22 frontend/contract tests, frontend build, docs/version-schema integration and real-engine synthetic recorder/timeline/snapshot integration. Integration run e0ca11fee62f3267ddf3a19159235b81 completed with 11 events; restore/branch/restart assertions passed. Log: `/tmp/chronosync-phase4c-check.log`.
- `CARGO_NET_OFFLINE=true GOPROXY=off GOSUMDB=off PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod make native-check`: passed sidecar, Rust fmt/locked Clippy, one real-engine bridge test and release build. Log: `/tmp/chronosync-phase4c-native.log`. This is not GUI acceptance.
- Final additional expired backend-reference and definite failed-attempt/revocation tests passed with `go -C engine test -race ./internal/admission` in the same offline/cache environment after the broad checks began. No production code changed after those checks began.
- Final Markdown formatting and `npm run check:docs` passed. Local-link scan includes new fixture provenance and ADR links. All existing non-Markdown source/config/assets and the Phase 4A/4B reports match the pre-change hash inventory; only new admission-package files and documented Markdown updates were added/changed.
- An optional read-only process inventory (`ps`) during a slow frontend-test stage was denied by the workspace sandbox. No escalation or host change was attempted; the established check completed successfully and its log supplied the result.

No live agents, external model providers, new platform probes, privileged facilities, authentication UI, untrusted application code or backend experiments were run by the new tests. Existing repository synthetic subprocess checks are run only through established test targets. Domain evidence proves neither OS isolation nor durable atomicity, authenticated approval or rollback resistance.

## Remaining gates and next approval

Phase 1–3 native GUI acceptance remains unverified/blocked. No GUI walkthrough, signing, remote CI or new cross-platform runtime guarantee is claimed.

Stop at Phase 4C. The next implementation requires explicit approval for the admission repository/migration/recovery contract, including atomic revision/claim/approval uniqueness, audit ordering, version handling and backup/restoration invalidation. Approval authentication, private input/key management and backend object binding must be settled before any real readiness/admission integration; no positive launch, UI or backend is automatically authorized.
