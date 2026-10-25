# Engineering instructions

ChronoSync is working on the explicitly authorized Phase 4G isolated macOS authentication prototype. Independent disposable protocol tests are allowed. Native authentication policy requires explicit factor selection; new dependencies/helpers/signing/entitlements/packaging require approval before introduction. Preserve current Phase 4E/F work and production provider/admission/supervisor/agent behavior. No positive launch, production authentication integration, migrations, public IPC or key management. Documentation and honest blocked/unverified evidence are part of done. Do not start another phase without authorization.

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
