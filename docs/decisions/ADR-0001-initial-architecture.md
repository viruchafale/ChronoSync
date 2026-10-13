# ADR-0001: Initial architecture

Historical bootstrap decisions: ADR-0002 supersedes the required Lamport event format; ADR-0004 supersedes Git-worktree/domain-prefixed CAS assumptions and defers GC; [ADR-0005](ADR-0005-workflows-evaluation.md) supersedes mandatory distributed clocks for workflow correlation. These original alternatives remain as decision history, not current implementation claims.

Date: 2026-10-21. Status: accepted for Phase 0; implementation gates below remain open.

## Evidence and context

Empty directory, no Git history or user work to migrate. macOS arm64 has Go 1.26.2, Node 22.14.0, npm 10.9.2, Codex CLI 0.163.0, Git 2.47.0 and Apple command-line tools. Rust/Cargo absent. Official Codex and Tauri documentation was inspected; see CODEX_INTEGRATION.md and ENVIRONMENT_AUDIT.md.

## Decisions and alternatives

| Area              | Selected                                                               | Alternatives and trade-offs                                                                                                                                                                                        |
| ----------------- | ---------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Desktop           | Tauri 2, React 19, strict TS, Vite 7                                   | Electron duplicates a browser runtime; native UI adds platform-specific work. Rust is host glue, Go owns domain logic. Vite 7 keeps this bootstrap on Node 22.14; major upgrades require compatibility checks.     |
| Packages          | npm with committed lockfile                                            | pnpm/bun add setup with no existing convention benefit.                                                                                                                                                            |
| Engine IPC        | host-owned Go sidecar, bounded JSONL over inherited stdio              | Unix sockets/named pipes add discovery/permissions differences; HTTP adds ports and authentication attack surface; FFI couples runtimes and crashes. Stdio requires explicit backpressure and process supervision. |
| Process ownership | one engine per app instance, one active run initially                  | No detached daemon in first milestone; simpler shutdown and ownership. Sidecar packaging is a P1 gate.                                                                                                             |
| Storage           | SQLite WAL metadata, immutable SHA-256 file objects                    | SQLite-only makes large payload retention costly; file-only lacks transactional queries. SQLite driver selection deferred to P2 after portability benchmark.                                                       |
| Workspaces        | owned Git worktrees plus explicit snapshot manifests                   | Worktrees share Git objects, not necessarily filesystem CoW. No reset/checkout of original workspace. Non-Git support deferred.                                                                                    |
| Adapter           | Codex exec JSONL first                                                 | App-server provides richer bidirectional sessions but larger lifecycle/protocol surface. SDK adds another runtime. No screen scraping or undocumented internals.                                                   |
| Isolation         | supported Codex policy for P1; explicit platform isolation gates in P4 | Path checks and worktrees are not containment. Containers optional, never assumed installed.                                                                                                                       |
| Tests             | Go tests/race/vet, Vitest/Ajv, ESLint/TS/Prettier, native CI matrix    | Small stack; no E2E framework until interactive workflows exist.                                                                                                                                                   |

## Dependency adjustments

Keep the requested phase numbering. Reserve ordering fields and capability manifests now; implement single-writer durable ingestion in P2 and minimum atomic blob operations there when payloads need them. P3 expands CAS/snapshots/GC. P1 already needs deadlines, cancellation, and bounded reads; P4 strengthens them. Lamport rules enter the event contract before concurrent capture; P6 supplies distributed ordering analysis and tests. Minimal redaction is mandatory before any P1 capture, not delayed to P5.

## Consequences and unresolved gates

Phase 0 has no running IPC bridge and no bundled Go sidecar. P1 must validate packaged sidecar lookup, protocol negotiation, OS process groups/Windows job objects, and host shutdown before agent launch UI ships. P2 must select a SQLite driver and test filesystem durability. Rust lockfile must be generated and checked in before reproducible native releases. Target OS matrix is unverified until actual CI runs. Signing, updates, license, and release identifiers remain undecided. No major requested architecture component was replaced.
