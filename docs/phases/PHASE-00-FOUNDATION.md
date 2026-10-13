# Phase 00 — Foundation

## 1. Objective and user-visible outcome

Establish a reviewable repository, truthful minimal shell and engine, contracts, and a complete implementation roadmap.

## 2. Problem

An empty directory has no conventions, boundaries, reproducible checks, or evidence for starting the recorder.

## 3. Scope and non-goals

Architecture, docs, tool audit, scaffolding, CI and compatibility tests. No agent execution, persistent store, live IPC, decorative dashboard or later-phase engine.

## 4. Prerequisites and dependencies

User bootstrap specification; installed toolchain and official compatibility documentation.

## 5. Design and component boundaries

Go version CLI and JSON schema are the only executable engine boundary. React shell is static; Tauri host has no privileged commands.

## 6. Ordered implementation tasks

1. Audit empty root, Git status, tool versions and Codex help/auth status without reading secrets.
2. Inspect official Codex/Tauri documentation and record ADR with trade-offs.
3. Create root conventions, app/engine scaffolds and version contract.
4. Write security/data/system design and all eight phase plans.
5. Install dependencies, lock versions, execute portable checks, investigate failures.
6. Run native build when Rust exists; otherwise record exact blocker and leave phase incomplete.
7. Update status/changelog with command evidence and stop for review.

## 7. Expected files/modules

README.md, AGENTS.md, docs/, apps/desktop/, engine/, contracts/, scripts/, Makefile, .github/workflows/.

## 8. Interfaces and contracts

Implemented version schema v1; proposed IPC/event envelopes explicitly labeled unimplemented.

## 9. Security and failure considerations

Preserve existing work; never launch a live agent, publish, or silently install system toolchains. Dependency downloads require available network.

## 10. Unit, integration and end-to-end tests

Unit: CLI invalid arguments/output errors and incompatible version reports. Integration: execute Go and validate JSON with Ajv. E2E: native shell launch/status inspection after native build; no agent smoke test.

## 11. Acceptance criteria

All required docs and 16 plan sections exist; make check passes; native build/format/Clippy and shell smoke pass or remain explicit completion blockers; no capability misrepresentation.

## 12. Validation commands

```sh
make check
npm run desktop:build
cargo fmt --manifest-path apps/desktop/src-tauri/Cargo.toml --check
cargo clippy --manifest-path apps/desktop/src-tauri/Cargo.toml -- -D warnings
```

## 13. Risks and unresolved questions

Missing Rust; native dependency resolution/lockfile and OS runtime remain unverified. Dependency audit must be reviewed before bootstrap closes.

## 14. Exit criteria

Portable and native acceptance evidence recorded, docs current, user receives final report. Missing native gates prevent COMPLETE.

## 15. Documentation updates

Update this plan's task evidence and status, ../IMPLEMENTATION_STATUS.md, ../CHANGELOG.md, relevant ARCHITECTURE.md, SYSTEM_DESIGN.md, DATA_MODEL.md, SECURITY_MODEL.md, TESTING_STRATEGY.md and integration docs. Record new decisions in ../decisions/. Include actual commands and distinguish passed/failed/skipped/unverified outcomes.

## 16. Status

`BLOCKED`

Scaffolding and documentation written; verification evidence and remaining gates are tracked in [implementation status](../IMPLEMENTATION_STATUS.md). Do not mark complete while native checks remain blocked.

Task evidence: steps 1–5 and 7 completed, including zero-finding dependency audit and successful portable checks. Step 6 attempted and blocked: Cargo is not installed. Native build, fmt/Clippy and shell smoke remain required before COMPLETE. Phase 1 has not begun.

## Phase 1 prerequisite follow-up — 2026-10-21

Official temporary Rust 1.99.1/Cargo/rustfmt/Clippy installed under /tmp, Cargo.lock generated, native release build and Rust checks now passed during authorized Phase 1. The original missing-Cargo blocker is resolved. Native GUI automation timed out; original GUI smoke evidence remains unverified. Historical Phase 0 evidence above is retained rather than rewritten as a prior success.
