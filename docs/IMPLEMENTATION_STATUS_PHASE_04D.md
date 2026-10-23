# Phase 4D implementation status — durable admission repository

As of 2026-10-23. Implementation and applicable repository checks complete. Persistence/recovery only: no process-launch integration or positive policy admission. [Status index](IMPLEMENTATION_STATUS.md). Phase 4A–4C reports remain unchanged.

## Preflight and decisions

Inspected actual admission domain/canonical vectors, policy/supervisor, SQLite migrations/checksums, writer gate/lease/recovery, status reports, Phase 4 plan, system/data/security/testing documents and ADR-0005/0006/0007. The repository has no Git metadata; reviewed files directly and recorded a pre-change hash inventory. Go 1.26.2 darwin/arm64, Node 22.14.0 and npm 10.9.2 are available; native checks use the existing temporary Rust installation.

The user's Phase 4D request authorizes migration/persistence only. Schema, constraints, transaction/idempotency semantics and restoration/privacy limits were written in DATA_MODEL before implementing migration 3. Existing references can be stored without choosing their authentication or commitment algorithms; they remain unverified evidence references, never production enforcement authority. Silent live DB rollback has no reliable trusted detector and remains unsupported; supported restore requires stopping/reopening, and all positive launches remain refused.

## Implemented files and data model

- `engine/internal/storage/admissions.go`: internal AdmissionRepository interface and typed create/change/read/receipt records; migration 3; bounded strict JSON serialization; domain replay/validation; atomic mutation and recovery.
- `engine/internal/storage/store.go`: registers migration 3, store-local admission coordination/epoch, and startup admission recovery. Original migration 1 and 2 SQL is unchanged.
- `engine/internal/storage/admissions_test.go`: synthetic temporary-database concurrency/fault/migration/recovery tests.
- `engine/internal/storage/snapshots_test.go`: the existing version-1 upgrade assertion now expects three migration ledger rows rather than two. Its upgrade/snapshot recovery test was preserved; this is required by the new authorized migration, not weakened coverage.
- Updated README, engineering scope, Phase 4 plan, architecture/data/system/security/testing docs, ADR-0007, changelog and status index. No old milestone reports rewritten.

Migration 3 creates admission_records (immutable binding/initial times and revisioned snapshot), admission_operations (append-only typed request/result/digest at consecutive revisions) and admission_references (unique typed challenge/approval/claim/revocation ownership tied by FK to the exact audit revision). Optional binding parent refers to an existing run. Capability references remain in immutable binding data; approval/challenge/revocation refs remain in immutable domain commands/results. No new run or approval is fabricated for old executions.

Binding/spec encoding is storage JSON version 1; domain binding version 1/canonical bytes are unchanged. Records have explicit storage version checks and bounded strict typed decoding; canonical JSON equality rejects duplicate/unknown/ambiguous keys. The repository reconstructs with NewBinding/New/Apply and compares every audit result/projection/reference; it never hydrates arbitrary domain state or substitutes SQL transition rules. Limits: 1 MiB binding/create envelope, 64 KiB transition/result envelope, 64 operations per aggregate; read transactions have 2-second limits and writes reuse the existing 5-second transaction bound/nonblocking writer gate. Recovery inventories 100 IDs at a time. No retention, repair, cleanup or delete endpoint.

## Durable semantics

Each approved transition verifies expected revision in the transaction, assigns the next audit/domain order, invokes Apply, conditionally updates state and appends audit plus uniqueness references before commit. A failed write/audit/domain operation changes nothing. SQL allows one claim/attempt per admission and one owner per typed reference; a failed/uncertain outcome never releases the claim. No DB transaction is claimed atomic with an OS process operation—none is performed here.

Same admission/operation ID and identical normalized typed request returns its original receipt with Replayed=true. Same key/different content is CONFLICT. A new key with stale revision conflicts; repeated terminal actions at a fresh revision are domain refusals. This applies to claim/revocation/observation as well as creation. A replayed receipt can be older than the current state; callers must query current state and must never treat a receipt as launch permission. A cancelled/uncertain response does not establish rollback or commit; retry/lookup the same identity to reconcile. Unsuccessful transitions are not audit entries; explicit legal Refuse/not_started/unknown observations are separate operations.

Open performs admission integrity/FK/replay validation and conservative recovery before returning. Recovery serializes with admission writers, disables them while in progress and creates a fresh random session epoch. Unclaimed nonterminal records are invalidated, claimed pending records become outcome_unknown, and terminal/conclusive/unknown histories stay unchanged. Commits are per admission; interrupted partial recovery is safe to repeat. Recovery failure leaves writes disabled and preserves all data. No process resumes, retries or PID-based kills. Prior-session mutations refuse; exact historic retries only return replayed receipts.

Supported stopped-backup restoration invalidates old readiness when reopened with a new epoch. Epoch identity is not a monotonic rollback detector, and coherent in-place rollback under a live process is not reliably detected. This limitation must be resolved before any positive launch integration; persistence adds no OS isolation, authentic approval, credential safety or anti-tamper certification.

## Privacy and compatibility

Only domain-approved public descriptors and opaque references/commitments are stored. No environment values, private stdin/prompts, credentials or keys are collected; no private commitment algorithm chosen. Public argv/path fields still require reviewed non-secret input; there is no full secret detector. Unused command fields are rejected rather than becoming an ignored-data channel. Errors do not expose SQL or raw request payloads.

Engine 0.3.0, protocol 1, event 2, run 1, manifest 1 and policy/admission domain 1 remain unchanged; SQLite is now migration 3. Existing event/snapshot/branch records are preserved byte-for-byte in migration tests. An older engine refuses schema 3. No down migration: restore a complete stopped backup with its compatible binary, losing writes after that backup. Keep private permissions and all SQLite sidecars/payload/object files; never restore into an active store.

Admission domain, canonical fixtures, policy, supervisor, Codex, recorder, IPC and UI sources are unchanged. Repository methods are not wired to any process-launch path or public API. The conservative provider and non-nil policy refusal remain intact.

## Actual verification and development outcomes

- Initial existing-storage tests passed after schema registration; new tests initially failed compilation because migration-test imports were staged before their tests. Added the planned migration tests and corrected fixture construction; no runtime failure or test deletion was hidden.
- `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod go -C engine test -race ./internal/storage -run TestAdmission -count=1`: passed repository lifecycle/idempotency, eight concurrent claims, stale revision, claim/revoke and claim/recovery races, audit/pre-commit/post-commit fault injection, rollback, cancelled/expired contexts, actual SQLite writer contention, corruption/missing relationships, schema upgrades/late migration rollback/checksum/future-version refusal, stopped backup restoration and prior-epoch refusal.
- Same offline/cache environment with `go -C engine test -race ./internal/storage ./internal/admission ./internal/policy ./internal/supervisor`: passed (domain/policy/supervisor reported cached). Follow-up storage tests include malformed unused fields, a real SQL writer lock and broken FK startup refusal.
- `make check` with the offline Go/cache environment above: passed gofmt, Go vet, full race-enabled Go tests, engine build, Prettier, ESLint, TypeScript, 22 tests in four Vitest files, frontend build, document checks, schema integration, and synthetic recorder/snapshot/branch/restart integrations. Recorder evidence run: `d20a8754d93257014225ae0b7aea8917`, 11 events, completed.
- `CARGO_NET_OFFLINE=true GOPROXY=off GOSUMDB=off PATH=/tmp/chronosync-cargo/bin:$PATH RUSTUP_HOME=/tmp/chronosync-rustup CARGO_HOME=/tmp/chronosync-cargo GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/tmp/chronosync-go-mod make native-check`: passed sidecar build, Rust formatting, Clippy with warnings denied, one real-engine bridge negotiation/cancellation test, and release desktop build. No GUI acceptance inferred.
- Pre-change SHA-256 comparison confirms Phase 4A–4C reports, admission domain/canonical fixtures, policy, supervisor, recorder, adapters, IPC, UI sources and dependency lockfiles unchanged. The packaged engine sidecar was rebuilt by native checks.

Tests cover injected failures rather than genuine power loss, disk exhaustion or every fsync fault. The post-commit fault models a lost response; no actual process launch or admission consumption by an agent occurs. No provider calls, privileged probes, host changes or dependency installs. Existing portable/native integration targets retain their synthetic subprocess boundaries.

## Remaining gates and next approval

Phase 1–3 native GUI acceptance remains unverified/blocked; no GUI walkthrough is claimed. Prior platform/fault-test limitations remain. No sandbox, authenticated approval, trusted capability provider, secret commitment/key management, live rollback detector or positive launch exists.

Stop after Phase 4D. Next implementation requires a scoped security review/approval for trusted approval/provider provenance and launch-binding prerequisites. Do not connect repository receipts to the supervisor until those mechanisms, fresh revalidation/revocation and backend-specific identity/enforcement are independently approved and verified. No later work is automatically authorized.
