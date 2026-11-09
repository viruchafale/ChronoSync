# Engineering instructions

ChronoSync has completed the explicitly authorized Phases 1–11 comprehensive audit, repository-fixable remediation and safe local acceptance-closure pass. Preserve the audit matrix/ledger, deterministic governed-deletion recovery and logical database-capacity regressions, Phase 4 fail-closed authority, fixture/report/dataset/lineage/package contracts, strict IPC/package verification, append-only migration/trust behavior and production provider/admission/supervisor behavior. Replay, analysis, governance and exchange must never execute historical commands, call a provider, add network/live fallback, infer trust from validity, infer safety from trust, claim whole-run/LLM determinism, or connect results to launch authorization. Production custody remains the Phase 9 login Keychain: do not describe it as Secure Enclave, hardware-backed, non-extractable, per-sign user-presence protected, same-user-malware resistant or cross-platform. Native GUI/physical fault acceptance remains manual, and Phase 11 reports zero valid signing identities; protected signing, migration, signed-upgrade continuity, notarization and Gatekeeper acceptance remain externally blocked. Documentation and honest unsupported/unverified evidence are part of done. Do not start Phase 12 without user authorization.

## Before implementation

1. Read this file, `docs/IMPLEMENTATION_STATUS.md` and its linked phase/milestone status file, the relevant phase plan, and relevant architecture/design/security documents.
2. Inspect the repository and Git state; preserve user edits. Identify conventions and reusable code.
3. Check dependencies and available tool versions; do not assume missing tools exist.
4. State phase, scope, intended files, risks, and verification plan. Do not start unrelated phases.

## During implementation

1. Make small reviewable changes. Keep Go domain code independent of UI and adapters.
2. Use typed versioned interfaces and explicit errors. Add tests alongside behavior; label stubs honestly.
3. Explain dependency purposes and trade-offs. New frameworks or material architecture changes require an ADR with evidence.
4. Never put secrets, credentials, private prompts, or sensitive payloads into fixtures, logs, or source control. Treat captured content as untrusted.
5. Update the phase plan as tasks change or finish. Document materially failed approaches and scope changes with reasons.
6. Never delete or weaken failing tests without an explicit engineering justification.
7. Do not execute agents or arbitrary host commands without an explicit execution boundary and user consent. Never bypass sandbox restrictions.
8. Never commit, push, publish, overwrite user work, or clean user worktrees without authorization.

## After implementation

1. Run formatter, linter, type checking, tests, and builds relevant to changes. Run integrations when dependencies are available.
2. Record actual important commands and results; distinguish passed, failed, skipped, and unverified checks.
3. Keep a separate Markdown implementation-status file for each phase or separately scoped implementation milestone. Use `docs/IMPLEMENTATION_STATUS_PHASE_NN.md` (with a descriptive milestone suffix when needed); keep `docs/IMPLEMENTATION_STATUS.md` as an index only. Never combine new milestone reports into earlier status files. Update the relevant status file, index, phase plan, architecture as needed, and changelog. Completion requires acceptance-test evidence, not merely code.
4. Report changes, decisions, limitations, and next task. Never claim live Codex integration, sandbox security, replay guarantees, or cross-platform behavior without evidence.

## Conventions

Use npm and its lockfile, Go standard formatting and vet, strict TypeScript, ESLint and Prettier. Use Rust only for the Tauri host. `make check` checks the portable foundation; Rust checks are separate. Keep schema changes compatible or increment the protocol major. Dependencies flow UI → host → engine → ports; adapters implement ports. No UI imports in Go. No network listener for local IPC. Future data belongs in private OS application-data directories, never implicitly in a user's source repository.
