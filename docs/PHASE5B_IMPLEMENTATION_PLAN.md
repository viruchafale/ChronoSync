# Phase 5B implementation plan — synthetic transport contract

## Authorization and scope

The user authorized the isolated Phase 5B backend milestone: a structured synthetic transport port, immutable request/response fingerprints, deterministic in-memory matching and tests whose production dependency graph has no networking or process-launch path. This milestone builds on Phase 5A without changing fixture-v1 bytes or reinterpreting `command.completed` interactions.

Phase 5B does not add HTTP, MCP, provider SDK, socket, subprocess, supervisor, approval/policy, public IPC, renderer, database, migration or submitted-DMG behavior. It does not replay a provider or claim OS-level network isolation or deterministic LLM execution.

## Repository findings

- Phase 5A fixture v1 is a canonical external JSON envelope for command-completion fingerprints. Its format constants, strict loader and evaluator are intentionally narrow and must remain byte compatible.
- The existing redactor handles common credential-shaped bounded text heuristically. It cannot prove that arbitrary URLs, headers, prompts or bodies are safe, and a redacted request may lose matching identity.
- Event v2, the recorder and SQLite store have no controlled request/response transport events. Adding transport capture there would require a broader privacy, schema and consent design.
- Production Codex adapter and supervisor paths can launch a process. They must remain unreachable from this experiment.
- The current test environment has no established OS/CI network-denial facility. Structural package isolation can prove that this implementation has no network-capable dependency, but cannot prove process-wide or OS-level egress denial.

## Contract and ownership model

Add a separate internal experimental contract in `internal/replay/transport` with:

- a `SyntheticTransport` interface that accepts only a structured request and returns a typed inert outcome;
- versioned transport identity, request-fingerprint and response-fingerprint domains;
- bounded canonical operation identifiers and an allowlist of non-credential metadata names;
- no arbitrary request or response body support;
- immutable recorded entries containing global sequence, per-request occurrence, transport identity, request fingerprint and a recorded response reference;
- an immutable response catalog containing the response fingerprint and bounded sanitized response metadata;
- typed `matched`, `unmatched`, `ambiguous`, `unsupported`, `cancelled` and `invalid` outcomes.

Constructors copy caller-owned slices before retaining them. Match results return detached copies. Canonical fixture loading also creates fresh owned values, so mutation after insertion or retrieval cannot alter stored state.

## Fingerprinting, redaction and persistence

Request and response fingerprints use stable JSON structs with explicit version tags and SHA-256. Before hashing, every accepted string passes through the existing redactor. Inputs are refused if redaction changes them, produces a redaction/truncation marker, exceeds a bound, uses an unapproved metadata name, contains duplicate/unsorted-inconsistent metadata, or contains a body. Authorization, cookie, token, credential, prompt and similar names are never accepted.

The experimental fixture persists request fingerprints rather than request operations or metadata. It persists only the minimum sanitized response status and allowlisted metadata needed to return a synthetic response, keyed by a generated response reference. It never persists bodies, credentials, cookies, tokens or private prompt content. Fingerprints are unkeyed equality identifiers: they are not encryption, signatures, authenticity evidence or tamper-proof provenance, and low-entropy inputs can be guessed.

The separate fixture has its own format/version/domain, strict canonical JSON envelope, byte/count/string bounds and digest. Import rejects unknown fields/versions, noncanonical bytes, invalid digests, duplicate or nonconsecutive global sequence, invalid occurrence order, missing/duplicate response references, inconsistent response fingerprints and oversized data. Phase 5A `Load`, `seal`, format version 1 and canonical bytes remain unchanged.

## Matching semantics

Entries are globally sequenced from 1 without gaps. Occurrences for each `(transport identity, request fingerprint)` start at 1 and increase by one in global-record order.

For an incoming request, the matcher validates and fingerprints the structured value, then selects records with the same identity and request fingerprint. With one recorded occurrence, occurrence zero may select it. With several recorded occurrences, occurrence zero is ambiguous and no response is chosen. Repeated requests must explicitly request occurrences 1, 2, … in that order; skipping, repeating or reordering an occurrence is invalid. A valid requested occurrence with no record is unmatched. Each matcher owns consumption state; `Reset` clears that state and a newly constructed matcher starts clean. Cancellation checked before validation and before state mutation returns cancelled and consumes nothing.

Exactly one eligible entry yields its detached recorded response. No result falls back to a live service, guesses, fabricates success or silently selects the first duplicate.

## Intended files

- `engine/internal/replay/transport/transport.go`: inert port, canonical fingerprints, immutable recording and deterministic matcher.
- `engine/internal/replay/transport/fixture.go`: separate experimental canonical format and strict loader.
- `engine/internal/replay/transport/transport_test.go`: stability, ordering, ambiguity, cancellation, reset, mutation, redaction, malformed fixture and forbidden-dependency tests.
- `docs/PHASE5B_IMPLEMENTATION_REPORT.md` and `docs/IMPLEMENTATION_STATUS_PHASE_05B.md`: actual evidence, limits and next milestone.
- Current architecture, data, security, testing, changelog, phase and index documents: concise 5B boundary and status updates.

No public TypeScript/Rust contract or SQL migration is planned.

## Verification plan

1. Run `gofmt`, focused replay race tests and replay vet/build.
2. Run the repository `make check` using established offline Go caches.
3. Run `make native-check` with locked/offline Cargo caches if available.
4. Confirm Phase 5A fixture tests still pass, production replay files did not change semantics, no forbidden imports or dependencies enter the transport implementation, and the submitted DMG hash is unchanged.
5. Record passed, failed, skipped and structurally unverified claims in the separate 5B report.

The initial implementation was placed directly in `internal/replay`. A dependency audit showed that this inherited Phase 5A storage/SQLite transitive packages, including unrelated standard networking and `os/exec` packages. Before acceptance it was moved into `internal/replay/transport`; the focused package now has no third-party, network, storage or launcher dependency. No Phase 5A source contract changed.

## Stop boundary

Stop after this isolated contract, tests and documentation. Real transport interception/capture, provider or MCP integration, OS-level offline enforcement, event/storage migration, IPC/UI, response bodies and Phase 4 launch binding require separate reviewed milestones. Do not begin Phase 5C, commit, push, publish or rebuild the DMG automatically.
