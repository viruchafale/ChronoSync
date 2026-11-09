# Testing strategy

## Phases 1–11 comprehensive re-audit

The 2026-11-04 audit reruns the entire Go race suite uncached, both full gates, focused real disposable Keychain tests, opt-in 100k timeline and snapshot workloads, Linux/Windows engine cross-builds, release preflight/configuration tests, module/dependency checks and immutable-DMG verification. The acceptance-closure pass reruns both full gates and adds governed-deletion tests for interruption before removal and after removal/before tombstone, plus SQLite logical-capacity rollback and integrity coverage. Focused regressions also cover exact release identity/team selection, semantic entitlement/profile validation and strict legacy IPC parameters. Native GUI launch smoke is recorded separately from workflow acceptance; the latter remains manual. See the [matrix](PHASES_01_11_AUDIT_MATRIX.md), [ledger](PHASES_01_11_REMEDIATION_LEDGER.md), [closure report](PHASES_01_11_ACCEPTANCE_CLOSURE_REPORT.md) and [manual checklist](NATIVE_GUI_ACCEPTANCE_CHECKLIST.md).

## Phase 10 release identity and user-presence acceptance

Node structural tests validate TeamIdentifier refusal/rendering, release
overlay invariants, code-signature parsing and fail-closed sidecar signing. Rust
tests validate the closed authorization outcome mapping and malformed-request
refusal. A separately invoked actual Security.framework test asserts the exact
missing-entitlement outcome in the unsigned context.

These are not positive user-presence evidence. Success/cancel/deny prompt
behavior, same-key protected migration, signed update continuity, notarization,
Gatekeeper and manual GUI acceptance require an authorized Developer ID build
and remain blocked. Existing Phase 8/9 exchange, migration, trust, UI and native
bridge suites provide regression evidence without provider or historical-command execution.

## Phase 11 authorized release acceptance

Phase 11 adds release-preflight and candidate-continuity tests. Synthetic identity-list/signature metadata exercises parsers and mismatch refusal only. The actual preflight is environment evidence but not signed-build evidence. Positive protected signing requires a strictly verified candidate and actual prompt success/cancel/deny; migration requires a disposable same-key workflow; notarization and Gatekeeper require separate external/manual results. With zero valid signing identities, these tracks are reported blocked. Existing portable and native gates remain the available regression authority.

## Phase 9 custody and recovery acceptance

Go race tests exercise unchanged identity/package compatibility, exact one-use
signing sessions, legacy migration transitions, restart/idempotency,
concurrency, invalid seeds, key mismatch, corruption, revocation and trust
audit refusal using disposable stores. Rust tests exercise the narrow broker
with denial, missing, malformed, conflicting and unsupported outcomes. React
component tests cover explicit migration, custody checks, trust audit and
visible native-command errors.

The real macOS test creates a disposable login-Keychain item, retrieves it,
signs/verifies and deletes it. It is ignored by the default suite and run
explicitly because it touches the user's Keychain. Attempted data-protection/
user-presence variants failed with `-34018`; those failures are platform
evidence, not passes. Native confirmation automation, real power-loss at every
cross-store boundary, account/device migration and cross-platform custody
remain unverified. No test calls a provider or executes a historical command.

## Phase 6 determinism-analysis acceptance

Pure Go race tests cover identical, changed-output, missing, additional, reordered, workspace-divergent, incomplete, unsupported and multiple-observation datasets; hash-only extraction; explicit no-causality findings; threshold/incomparable behavior; report reevaluation, integrity refusal and restart. IPC integration creates disposable real event histories, persists a report and revised-policy report, then reopens both after restart. React tests exercise real selection/submission, aligned divergence/evidence quality, historical loading and reevaluation. A source audit forbids provider, HTTP, process-launch, supervisor and production-adapter imports; the application service intentionally consumes existing storage/snapshot evidence ports and private report files. Native checks compile and test only the bridge/build boundary; native window interaction remains separate acceptance evidence.

## Completed Phase 5 verification

Lifecycle tests cover content-addressed publication, duplicate import, success-only handoff, restart, detached export, consent, delete, partial files, traversal, permissions, unsupported data and corruption. Unified tests cover pass/mismatch, repeated occurrences, ambiguous/unmatched/unsupported/cancelled paths, fresh state, malformed plans/fixtures and canonical result bytes. IPC end-to-end tests use an in-memory adapter and disposable workspace/store to exercise record → create/import → inspect → pass/mismatch/cancel → export/delete/re-import → restart → corruption refusal. React tests cover actual fixture controls and mismatch display; Rust tests negotiate the real engine and retain recorder/snapshot coverage. Native dialog interaction and OS-level network denial remain manual/unverified gates.

## Phase 5B synthetic transport contract

Focused race tests cover canonical bytes/fingerprints, metadata-order normalization, explicit repeated-request occurrences, ambiguous and unmatched requests, invalid order, cancellation, reset, caller/return-buffer mutation, body/credential/prompt-shaped refusal, malformed and tampered fixtures, bounds and missing references. A recursive production-source dependency test rejects networking, storage, adapters, supervisors, `os/exec` and third-party imports reachable from `internal/replay/transport`. An independent `go list -deps` audit confirms those forbidden packages are absent. This is structural offline evidence, not OS-level network isolation.

## Phase 5A replay foundation

`go test -race ./internal/replay` uses real temporary SQLite history and controlled in-memory evaluators. It covers stable canonical export, matching/mismatched results, success-only regression bytes, altered bytes/digests, unknown fields/versions/types, missing/reordered/duplicated/inconsistent interactions, unsupported/incomplete/unavailable/redacted/truncated sources, size/malformed input, cancellation, injected evaluator failure, repeated stable evaluation and absence of raw recorded values. A source-import inventory test rejects process, supervisor, adapter and network imports. These tests do not prove OS-enforced offline execution, provider replay or native GUI acceptance.

## Phase 4G independent protocol tests

Run `go -C experiments/macos-auth-protocol test -race -count=1 ./...` and `go -C experiments/macos-auth-protocol vet ./...` with offline dependencies and the documented temporary cache. This separate module is not automatically part of make check. Tests cover independently calculated canonical vectors, bounded challenges, malformed/forged/canonical-only frames, synthetic wrong-peer/nil-verifier refusal, session/manifest mismatch, expiry, cancellation, disconnect/restart, late callback, concurrent one-time completion and absence of production imports/exported entry points.

Successful verifier behavior is synthetic test data, not macOS identity evidence. Native APIs, packaged peer verification, interactive factor selection and native accessibility remain unimplemented/unverified. Actual environment, code-signing observations and approval blockers are in the [Phase 4G report](IMPLEMENTATION_STATUS_PHASE_04G.md). No native lockout or security-setting experiments.

## Phase 4F feasibility evidence and native acceptance plan

Research, read-only SDK/source inspection and a compile-only API surface check are distinct from successful authentication. The [Phase 4F matrix](IMPLEMENTATION_STATUS_PHASE_04F.md#8-test-matrix-and-evidence-classification) separates deterministic tuple/revision/replay tests, signed native-channel integration, interactive authentication/accessibility and unestablished attacker guarantees.

A future isolated experiment must cover fresh success, cancellation/denial, unavailable/locked-out states without destructive account experiments, expiry/reuse/mismatch, concurrent responses/revocation, restart/lost acknowledgement, forged peer/result, stopped backup/stale epoch and native accessibility. No production fake authenticator or positive launch is permitted. Current-phase checks validate unchanged conservative refusal and documents only; no native authentication prompt or OS security experiment is claimed. Prior Phase 4E broad build results are not rerun evidence.

## Phase 4E trust-boundary checks

New policy registry tests use same-package _test.go fakes unavailable to production construction: unknown provider, identity/implementation/contract/version mismatch, target/platform/scope/source mismatch, revocation generation changes, provider errors, unverified/expired/observed/configuration-only/unknown evidence, malformed/oversized requests, cancellation, clock regression and detached inputs/results. Production registry tests assert conservative refusal and no verified enforcement. Existing supervisor refusal tests are preserved.

New IPC tests reject forged verification, effective policy, provider, approval/time, admission/token/checks and unsupported policy fields before any adapter call or persisted run. They also test nonexistent authority commands, malformed/oversized bodies, duplicate/case-alias keys and sanitized diagnostics. No agent is launched by approval/authority tests.

New SQLite tests exercise unauthenticated reference invariants: changed manifest/challenge/epoch, expiry, replay/conflict, one-time response, challenge uniqueness, concurrent responses, revocation and restart. Existing race/recovery/corruption/restoration suites are rerun. Synthetic private canaries in rejected requests must not reach durable operation bodies or diagnostics. No OS authentication test is claimed: a backend/ceremony is not implemented. Actual results are recorded in [Phase 4E status](IMPLEMENTATION_STATUS_PHASE_04E.md).

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

## Phase 7 governance acceptance

Governance unit/race tests cover canonical creation/import/export, duplicate convergence, family/version conflict, unsupported/corrupt/missing references, added/missing/changed compatibility, policy reevaluation, immutable lineage, accounting, damage/temporary files, explicit lifecycle, referenced deletion refusal, leaf-to-root deletion, restart and concurrent import. IPC tests use disposable real Phase 6 reports and reject unknown authority fields. React tests cover accounting/lineage display, compatibility detail and visible reference refusal; locked offline Rust checks compile native dialogs and commands.

The test suite never runs a historical command, provider or network client. Native dialog automation, real power loss/disk full, hostile same-user races, OS-level egress denial and cross-platform runtime behavior remain separate unverified evidence.

## Phase 8 signed-exchange acceptance

Focused race tests include RFC 8032 Ed25519 vector 1, local identity/rotation, trusted/untrusted/revoked outcomes, signature/content tampering, unsupported algorithms/versions, restart persistence, omitted/mismatched provenance, deterministic re-signing content identity, interrupted/concurrent imports and strict package bounds. Adversarial archive tests reject duplicate/traversal/symlink/compressed/oversized/excess-entry packages plus malformed, duplicate-key and unknown-field manifests.

IPC integration builds real Phase 6 reports and a Phase 7 dataset in one disposable store, exports/signs, inspects as untrusted in a separate clean store, requires explicit consent, imports canonical reports/dataset, persists trust across restart and independently re-signs the same content. React tests cover explicit inclusions, validity/trust separation, privacy display and trust/revocation actions. Native dialogs remain compile-tested rather than GUI-automated.
