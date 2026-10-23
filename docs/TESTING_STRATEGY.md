# Testing strategy

## Phase 4D storage tests

Temporary SQLite tests exercise actual Phase 4C bindings/transitions, with no agent launches. They cover create/reopen, exact and conflicting retries, stale revisions, eight concurrent claim callers, revocation/claim and recovery/claim ordering, atomic audit/state/reference writes, before-commit/audit failures and simulated lost post-commit results, domain failure, writer and actual SQLite contention, cancelled/expired contexts, unknown versions, corrupt digests/projections, missing audit/reference rows, FK/claim/audit uniqueness, fresh epochs and stopped-copy restoration.

Migration fixtures are built from actual migration-1/2 SQL (plus an empty ledger case), retain event bytes and valid snapshot/branch metadata, and verify unchanged old checksums/no fabricated admissions. A late ledger trigger failure tests DDL+ledger rollback; removing that synthetic fault allows a clean retry. Checksum mismatch and future schema refusal are tested. This is transactional fault injection, not certification of power-loss/fsync/disk-full behavior. One pre-existing upgrade assertion now expects three ledger rows instead of two because migration 3 is explicitly authorized; its original upgrade/recovery coverage remains.

The domain canonical fixtures and policy/supervisor tests remain unchanged. Runtime suites use existing synthetic subprocess fixtures only; storage tests do not run agents or consume admissions for process launch. Actual commands/results are recorded in the separate [Phase 4D report](IMPLEMENTATION_STATUS_PHASE_04D.md).

## Phase 4C implemented pure-domain tests

Admission tests use only synthetic public inputs, explicit epochs/times and external assessment references. No process/backend/SQLite operations are called by the package or its tests. Fixed canonical bytes and SHA-256 were independently encoded with Python struct operations and are checked in under admission/testdata; the fixture README documents synthetic provenance.

Coverage includes canonical field mutation sensitivity, sorted collection independence, exact argv ordering, empty/nil handling, Unicode byte distinctions, duplicate/unknown/malformed/bounds rejection, identity limitations, absence of raw env/stdin value fields, detached binding/state copies, valid challenge/approval/preparation, expired/missing/mismatched approval, stale/missing/mismatched evidence, repeated readiness checks, duplicate claim/refusal, revoke-before/after-claim, failed/unknown outcomes, epoch invalidation and unknown versions/states. A bounded exhaustive traversal of post-claim actions asserts that no successful transition releases the attempt or returns to a reusable state.

The fixed vector proves serialization stability, not authenticity. Synthetic external_validation is not a trusted human or backend provider. State tests establish pure transition semantics, not durable consumption, concurrency serialization, OS isolation or rollback protection. Actual checks and limits are recorded separately in [Phase 4C status](IMPLEMENTATION_STATUS_PHASE_04C.md).

## Phase 4B admission test plan — not runtime tests performed

Phase 4B changes documentation only; run formatting and local-link/structured-document checks and verify source hashes. Do not count Phase 4A tests as rerun. The following tests are acceptance requirements for later approved milestones:

- Domain: one-time IDs, canonical golden vectors across representations, ordered/empty argv, environment sorting/value commitments, request/effective digest mismatch, unknown versions, state transition rejection and bounded input. Property tests should prove no consumed state returns to prepared.
- Trust: renderer-injected verified evidence/effective policy/actor/expiry is refused; challenges cannot replay across request/session/engine epoch; provider registration and actual backend context must match. Lost keys, stale clocks and restored database snapshots invalidate prior readiness.
- Storage: atomic preparation/approval/claim/outcome audit, unique claim under concurrent callers, idempotency same-body retrieval versus changed-body conflict, FK and digest/version checks, audit projection disagreement, real migration-1/2 upgrade and unchanged event/CAS bytes. Inject failures before/after commit and ambiguous acknowledgement without relying on real power loss claims.
- Launch harness: inject a fake backend boundary, crash after each durable/OS step, prove no second attempt after claim, and distinguish not_started from started/outcome_unknown. No absent PID inference. A fake backend proves protocol state transitions only.
- Real backend, separately approved: replacement/symlink/reparse races, interpreter/library/config dependencies, changed cwd/workspace generation, input/secret-version drift, external writers, capability revocation between preparation and launch, descendants, cleanup and storage failure after spawn. Record exact OS/backend context and independently verify the claimed restriction. No privileged probe is authorized now.
- Existing recorder regression: consent, 1–3600 second timeout, cancellation, line/total output bounds, EOF and storage-failure behavior; bind both version-probe and execution launches before enabling a new mode. GUI tests must distinguish assessment, prepared, consumed, started, refused and unknown; retries cannot conceal an earlier unknown outcome. Phase 1–3 native-window acceptance remains separate.

See the [milestone sequence](phases/PHASE-04-SANDBOX-GUARDRAILS.md) for prerequisites and exclusions. No runtime test or build is required to substantiate this documentation-only milestone.

## Phase 4A implemented tests

Policy tests cover full synthetic satisfaction; mandatory unknown/unavailable/unsupported; malformed, missing and conflicting requirements; wrong platform/backend/configuration/scope; unverified/stale/partial evidence; refused widening; deterministic order, digests and independent result copies. Supervisor tests pass a marker-writing test child to Run and verify policy refusal before any marker/output. Existing consent/output/timeout/cancellation tests are retained. Tests use no host isolation assumptions, privileged facilities or live models.

See [Phase 4A status](IMPLEMENTATION_STATUS_PHASE_04A.md) for exact scope, contract semantics, verification and remaining gates. Earlier Phase 4 sections below describe the original design milestone; proposed backends and positive admission remain unimplemented.

## Phase 4 design validation and future test requirements

This milestone changes documentation only. Existing offline suites verify the working foundation, not the proposed workflow/policy/evaluation services. No new backend, live agent or platform escape experiment runs. Check Markdown formatting/local links, version references, diagram status labels and non-Markdown source hashes. Mermaid is manually inspected, not renderer-validated. Actual commands/results are in [status](IMPLEMENTATION_STATUS.md).

Before future implementation acceptance require:

- Policy unit tests: every dimension, deny defaults, required versus observed/unknown, workflow ceiling/run precedence, stale capability identity, changed workspace/executable, refused silent downgrade, newly approved weaker request, expired/revoked approval and sanitized audit failure. Mock capabilities prove resolver decisions only.
- Workflow tests: old runs unassigned; original event hashes unchanged; valid membership revisions, duplicate upstream IDs across runs, partial tool lifecycle, conflicting/late observations, terminal append refusal, unresolved targets, parent cycles, handoff provenance and inferred labels; stable paged traversal under concurrent appends/membership changes and bounded fan-out.
- Persistence tests: genuine migration-1/2 upgrade, unchanged checksums/event/object bytes, atomic run membership and relationship revisions, foreign-key/run-event consistency, rollback/crash/full-disk failures, unknown-version refusal and stopped-backup restoration. No migration suite is added now.
- Evaluation tests: frozen input digests, history damage/missing snapshots, incomplete/active-run high waters, evaluator/config/dataset/test-suite identity, null versus zero, units and metric incompatibility, provider versus estimated costs, monotonic versus wall-clock duration and judgment/evidence separation. A model evaluator needs separate consent and test fixtures.
- Separately approved OS integration tests: deny external paths including symlinks/mounts/reparse points, internet/DNS/IPv6 and local sockets/pipes, credentials/inherited handles and CAS history access, process escape and descendants, resource exhaustion, backend drift, cancel/EOF/crash and unknown survivors. Record exact OS/kernel/backend/configuration and required privileges. Test inside an approved disposable boundary; never assume a container is safe merely because it starts.
- Future GUI tests: dimension-level effective-policy disclosure, refusal, changed approval, unknown enforcement and inaccessible controls; keyboard/focus/error paths. Existing Phase 1–3 native capture/inspect/cancel/timeline/snapshot/restore/branch/restart/failure/keyboard walkthrough remains unverified. Component tests and native builds do not satisfy it.

## Implemented Phase 3 checks (2026-10-22)

Manifest tests were written and passed before capture/restore. They reject traversal, absolute/ambiguous/Unicode names, case collisions and missing parents, reserved names, unsupported versions/kinds/modes, bad hashes/lengths, excessive entries/aggregate bytes, excluded metadata and trailing JSON. Objects tests cover exact bytes, concurrent identical puts/dedup, missing/corrupt objects and temp evidence. Capture tests cover representative contents, empty files/directories, executable status, explicit/default exclusions, broken/escaping/cyclic links, special FIFO rejection, unreadable/large files and changes between passes. Directory traversal is batched; no large-directory allocation is required.

Storage tests upgrade an actual migration-1 database without rewriting its checksum, retain all prior Phase 2 tests, and exercise interrupted capture/branch rows. An abrupt subprocess exit leaves a preparing snapshot, published unreferenced object and temp file; reopen deterministically marks failed while inventory retains evidence. Restore tests compare bytes/modes, reject existing/malicious/symlink destinations and retain partial directories on injected read failure. Branch tests cover concurrent admission, ancestry, changed-content isolation, deduplication and unchanged original events. Cancellation of validation does not falsely mark damage. These tests do not certify real power-loss/fsync/disk-full handling.

The actual engine process integration now captures, restores, branches, edits/recaptures a branch, creates a child branch and reopens metadata without executing historical commands. Rust's real-engine bridge test adds snapshot capture/branch/verify. Three new component tests cover source consent/exclusions, preparing → ready and damaged/failure states. The native-window GUI gate remains unverified.

Opt-in workload (engine cwd): `CHRONOSYNC_SNAPSHOT_LOAD=1 go test ./internal/snapshot -run TestSnapshotPerformance -v -count=1`. It uses fresh private temporary APFS stores and three synthetic distributions (10 unique 1 KiB files, 1k repeated 1 KiB files, 16 unique 128 KiB files). Timings include validation/publication/materialization synchronization. Manifest and object byte counts plus actual measured results are in [status](IMPLEMENTATION_STATUS.md). Default CI skips load tests and never calls a live model.

## Implemented checks

`make check` runs Go format/vet/race tests/build, Prettier, ESLint, strict TypeScript, Vitest, production frontend build, documentation validation, version-process/schema integration and recorder-process integration. Default checks use synthetic fixtures only and never contact a model.

`make native-check` builds the sidecar, checks Rust formatting/Clippy, runs the Rust bridge test and builds the native executable. The bridge test runs the real Go process with a synthetic CLI, negotiates the protocol, validates errors, cancels a running child, and reopens the persisted cancelled run. `npm run desktop:bundle:macos` produces an unsigned local .app. The GUI automation attempt timed out, so native GUI start/inspect/cancel is not claimed verified.

## Coverage and provenance

- model: v2 serialization/validation, schema-version rejection, unique IDs and valid/invalid transitions.
- codex: checked-in synthetic JSONL mapping, command output, files, unknown omission, malformed items and error/failed turns.
- supervisor: actual subprocess success/nonzero exit, partial output, oversized lines, timeout/user cancellation, ordinary grandchild cleanup on macOS.
- recorder: injected storage failure stops an actual synthetic subprocess and preserves partial history; success/failure, malformed and missing terminal, partial history, consent/single-active run, deadline/cancel and file modification; real adapter process boundary via a synthetic executable.
- storage/legacy: original journal write/read, torn/malformed records, checksums, lease/traversal and damaged-directory isolation tests retained.
- storage: SQLite migration/checksum/future-version refusal, constraints/atomic rollback, stable cursor filtering/search, missing payloads/private permissions, idempotent legacy import/recovery, concurrent reads/producers and abrupt subprocess exit with an uncommitted transaction.
- workspace: created/modified/deleted files, default/configured exclusions, directory/file symlinks pointing outside root and cancellation.
- IPC: handshake/version rejection, errors, invalid paths, frame size and EOF cleanup. Rust separately exercises actual request/reply and cancellation.
- frontend: seven Testing Library/jsdom tests cover consent/start parameters, persisted ordered history/details, cancellation waiting for confirmation, and errors without false success, plus filter/search/paging, missing/truncated/interrupted history and loading/empty states. These are component behavior tests, not native E2E.
- cross-language integration: actual Go events validate against event-v2 JSON Schema, including sequence and persisted terminal state after restart.

Synthetic fixtures live in `engine/internal/adapter/codex/testdata`; their README records provenance. Runtime integration fixtures are generated under temporary directories. Never commit unredacted real captures, auth or private source. The original 12 Phase 0 contract tests remain intact.

## Live validation

`CHRONOSYNC_LIVE_CODEX=/absolute/path npm run test:live -- --consent` creates a disposable public-file Git repository, launches the real Codex adapter read-only with a 120-second limit, requires command output and successful terminal evidence, and verifies reopening. `--write-fixture` permits a generated.txt change in that disposable directory and asserts observed metadata. Authentication/policy/network failures are actual failures, never converted to passes. Default CI does not run these tests.

The local read-only test passed after the outer sandbox denied Codex initialization and an approved retry was used. Final run IDs, native command results and write-test outcome are in IMPLEMENTATION_STATUS.md. Provider-side interactions, hidden context and complete sandbox security are not tested by this smoke test.

## Remaining and future gates

Native GUI smoke, live cancellation/error UI paths, Linux runtime and Windows native CI have not been verified locally. Windows recording is unavailable. Remote workflow files are not evidence of a CI run. Signing, installers, full OS escape testing, power-loss/disk-full fault matrix remain explicit unverified gates. SQLite migrations are now tested.

P2 now includes synthetic crash-injection storage/migrations and opt-in 100k-event measurements; P3 verifies immutable ancestors and restore equivalence; a separately authorized P4 implementation must test actual containment per platform; P5 runs portable fixtures with egress denied; P6 validates evidence-backed ordering and heuristic false positives; P7 runs provider conformance. Keep passed, failed, skipped and unverified evidence separate. Do not weaken a test to make implementation appear complete.

## Phase 2 performance method

From engine: `CHRONOSYNC_LOAD=1 go test ./internal/storage -run 'TestLoad100k|TestConcurrentProducersReaders' -v -count=1`. Set the documented GOCACHE/GOMODCACHE under restricted environments. Default tests skip the 100k load workload and run the smaller correctness/concurrency suite. The load test creates a private temporary SQLite store, persists 100k short synthetic inline events (100 individual commits then batches of 100), measures a 100-row query including count, reports TotalAlloc delta, checkpoints for DB size at 10k/100k, and times full recovery. Separate contention measurement uses four producers with explicit conflict/busy retries plus concurrent page reads. It does not measure RSS, external-payload throughput, power loss or live LLM latency. Recorded environment and actual results are in [implementation status](IMPLEMENTATION_STATUS.md).

The original journal tests were relocated unchanged to the legacy package because corrupting a JSONL file is an importer compatibility test after SQLite becomes authoritative. They were not deleted or weakened. The process integration now validates cursor traversal against original event identities, filtered retrieval and detail equality after restart. Native bridge tests call new query/detail methods as well as cancellation. GUI E2E remains separate and unverified.
