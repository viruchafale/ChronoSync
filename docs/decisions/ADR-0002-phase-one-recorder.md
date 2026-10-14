# ADR-0002: Phase 1 recorder and staging persistence

Status: accepted for implementation. Date: 2026-10-21.

The user authorized Phase 1 despite the unresolved Phase 0 native gate. Native verification will be attempted with a temporary Rust toolchain; Phase 0 is not retroactively declared complete.

Use Codex 0.163.0's documented exec JSONL interface. The adapter owns vendor parsing, while the recorder owns lifecycle, IDs and order. A POSIX process group provides cancellation on macOS/Linux; Windows recording is blocked until job-object cleanup is implemented and tested. This is not an OS security sandbox.

Persist bounded sanitized inline payloads in a versioned append-only JSONL staging journal with fsync per record. Run metadata is atomically replaced; the event journal is authoritative for terminal state. Partial final lines are explicitly reported; valid preceding events survive. No SQLite implementation is introduced before Phase 2. A process-lifetime advisory lock prevents concurrent writers, and restart marks nonterminal runs interrupted. Hard engine crashes may leave descendants: recovery does not claim process absence or automatically resume them.

Adopt event schema v2 because the proposed v1 required a Lamport value without actual distributed instrumentation and had no inline typed payload. v2 uses one local decimal sequence and UTC observation timestamp; it makes no causal-order/replay claim. Protocol v1 retains negotiation and adds method capabilities; engine version becomes 0.1.0. Unsupported older event schemas are explicitly refused.

Workspace observation hashes bounded regular files through Go os.Root, excludes sensitive/dependency/build paths, skips all symlinks, and records metadata-only changes separately from vendor file-change reports. It does not create worktrees, restore files, or constrain agent subprocess file access. User-selected workspaces require explicit read-only/workspace-write consent. Prompts are sent via stdin and not persisted; captured text is redacted heuristically. Raw vendor payloads are omitted for unknown or malformed records to avoid unnecessary secret capture.

The Rust host owns one stdio engine and serializes bounded request/reply RPCs. UI polls paginated events; no background network listener. Go records independently of UI polling. Only typed allowlisted host commands are exposed, with per-window capabilities. A built Go resource is resolved from the installed application, not project PATH. Codex is selected from an explicit absolute installation path.

Required tests: parser/event validation, lifecycle, real fixture subprocesses, cancellation/timeout, process-group cleanup, journal recovery, symlink exclusions, IPC and UI behavior. Live Codex verification is separate and must report actual results, including failed authentication/network/policy cases.
