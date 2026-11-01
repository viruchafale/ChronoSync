# Phase 5B implementation report — synthetic transport contract

## Result and scope

Phase 5B adds an internal, in-memory synthetic transport port in `engine/internal/replay/transport`. It records bounded fingerprints into a separate experimental canonical fixture and returns only an exactly matched recorded response reference plus sanitized response metadata. It has no HTTP, MCP, provider, socket, subprocess, storage, IPC, renderer, admission, approval or supervisor integration.

Phase 5A fixture format 1 remains the command-completion contract. Phase 5B neither imports nor reinterprets it, and all existing Phase 5A export/import/evaluation and refusal tests continue to pass.

## Contract and matching behavior

`SyntheticTransport.Exchange` accepts a `SyntheticRequest` containing transport name/protocol/version, a bounded operation token, a small allowlisted metadata set and an optional occurrence. Request and response fingerprint bodies carry independent explicit format version 1 tags and use canonical struct JSON plus SHA-256.

Recording entries have consecutive global sequences. For each `(transport identity, request fingerprint)` key, occurrences start at one and increase in recording order. A unique recorded request can be matched with occurrence zero. Repeated requests with occurrence zero return `ambiguous`; callers must request occurrences 1, 2, … in order. Missing keys/occurrences return `unmatched`, repeats/skips return `invalid`, unsupported structures return `unsupported`, and cancelled contexts return `cancelled` without consuming state. `Reset` clears one matcher's occurrence state.

Exactly one eligible entry returns `matched` with its generated response reference, response fingerprint and detached sanitized response value. The matcher never selects the first ambiguous entry and has no live fallback.

## Redaction, fixture and immutability

The current redactor cannot prove arbitrary URLs, headers, prompts or bodies safe, so the contract refuses all request/response bodies and accepts no URLs or general header map. Metadata names are allowlisted separately for requests and responses; authorization, cookies, tokens and credential fields are unsupported. Values are bounded token-shaped strings, pass through the existing redactor before hashing, and are refused if changed, marked, truncated, duplicated or malformed.

The fixture stores request fingerprints rather than request operation/metadata. Its response catalog stores only status and allowlisted sanitized metadata. Caller slices are copied during construction, loaded fixture values are owned, and returned response slices are detached. Mutation tests verify that changing draft or returned buffers cannot alter recorded bytes or later results.

The separate envelope declares its domain and fixture/request/response versions. Strict loading rejects unknown fields and versions, noncanonical/trailing/oversized bytes, invalid envelope/response fingerprints, sequence gaps/reordering, invalid occurrences, malformed identities, missing/unreferenced response references and response-catalog order changes. Unkeyed fingerprints reveal equality and may permit guessing of low-entropy values; they are not encryption, signatures, authenticity evidence or proof of origin.

## Network-denial evidence and limitation

Tests use only in-memory request, recording and matcher values. A recursive production-source dependency test rejects any reachable network, `os/exec`, storage, adapter, supervisor or third-party import from `internal/replay/transport`. An independent `go list -deps` audit found none of those forbidden dependencies.

The contract was initially placed in `internal/replay`, but the audit showed that Phase 5A's SQLite dependency graph pulled in unrelated standard networking and `os/exec` packages. It was moved before acceptance into the dependency-light child package. This is structural network unavailability through the port. The environment has no established OS/CI egress-denial harness, so Phase 5B does not claim OS-level network isolation for the test process or host.

## Files and compatibility

- `engine/internal/replay/transport`: port, canonical fingerprints, immutable recording, strict experimental fixture, matcher and tests.
- `docs/PHASE5B_IMPLEMENTATION_PLAN.md`: pre-source plan and recorded scope correction.
- `docs/IMPLEMENTATION_STATUS_PHASE_05B.md`: separate milestone status.
- Architecture, system/data/security/testing/integration, phase, changelog and index documents: implemented boundary and evidence.

There is no dependency addition, SQL migration, event/protocol/TypeScript/Rust contract, renderer change or DMG rebuild. Production Codex, policy, admission, approval and supervisor paths are unchanged.

## Verification

| Command or check                                                                                                                                                         | Result                                                                                                                                                                                                                                                                          |
| ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off go test -race ./internal/replay ./internal/replay/transport` and matching `go vet`                                 | Passed Phase 5A compatibility/refusal tests and focused 5B canonicalization, matching, mutation, redaction/refusal, cancellation/reset, strict import and forbidden-dependency tests.                                                                                           |
| `GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off go list -deps ./internal/replay/transport` filtered for `net`, `net/*`, `os/exec`, storage, adapter and supervisor | Passed with no matching dependency. The package also has no third-party production dependency.                                                                                                                                                                                  |
| `GOCACHE=/private/tmp/chronosync-go-build GOPROXY=off make check`                                                                                                        | Passed Go formatting, vet, all race tests and build; Prettier, ESLint, strict TypeScript, 22 Vitest tests, frontend build, documentation checks, engine-version integration and synthetic recorder/snapshot integration.                                                        |
| Offline `make native-check` with the established temporary Cargo/Rustup paths                                                                                            | Did not complete. Sidecar generation and Rust formatting passed; Clippy compilation failed because the current temporary Rust 1.99.1 toolchain lacks its `proc_macro` standard crate. Earlier attempts exposed missing Cargo/Go paths and Rustup metadata, then were corrected. |
| Submitted DMG SHA-256                                                                                                                                                    | Unchanged at `e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05`; no bundle command ran.                                                                                                                                                                         |

The native failure is a current toolchain-installation limitation, not a product pass. Phase 5B changes Go and Markdown only; the prior Phase 5A native suite passed against the same unchanged Rust bridge, but that historical evidence is not represented as a new 5B native run.

## Limitations and next milestone

- The experiment does not capture or replay actual HTTP, MCP, Codex, provider or tool traffic.
- Bodies, streaming, URLs, timing, errors and arbitrary headers are unrepresented.
- Existing redaction remains heuristic; refusal reduces exposure but cannot prove every accepted token is non-sensitive.
- The fixture is internal and experimental, with no consent, destination, private persistence, inspect-before-share flow or public compatibility promise.
- No OS sandbox enforces process-wide offline execution.

A separately authorized Phase 5C should first decide whether to harden this experimental contract or design a controlled test-tool transport boundary with explicit consent and OS/CI egress denial. It should not connect real provider traffic, IPC/UI or live fallback until privacy, persistence and enforcement are reviewed.
