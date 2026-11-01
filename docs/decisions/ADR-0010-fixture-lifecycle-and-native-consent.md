# ADR-0010: Content-addressed fixture lifecycle and native consent

Status: accepted for Phase 5C–5F, 2026-10-31.

## Context

Phase 5 needs immutable fixture persistence, import/export and deletion without caller paths or renderer-only consent. Existing SQLite tables describe executions, snapshots and admissions; fixture bytes already have strict canonical loaders and do not need relational mutation. Browser file APIs cannot provide a consistently verified native desktop destination or trusted destructive confirmation.

## Decision

Store validated fixture bytes outside SQLite under the private application-data root at `fixtures/v1/sha256/<content-sha256>.fixture`. Derive inspection metadata from the validated bytes. Publish with private exclusive temporary files, hard-link no-replace semantics and directory sync. Verify filename/content hash, bounds, permissions and the strict format loader on every read. Never repair corruption.

Expose protocol-1 additive methods behind `replay-fixtures-v1`. IPC carries canonical fixture text and generated identities only, never filesystem paths. The Tauri host uses the official `tauri-plugin-dialog` dependency for native open/save selection and export/delete confirmation. It performs bounded regular-file reads and exclusive destination creation. The renderer can request an operation but cannot supply a path or forge the host confirmation boolean.

Unified evaluation accepts only structured synthetic requests and output fingerprints/exit codes. A new matcher is created for each evaluation, and there is no live fallback.

## Consequences

No SQL migration is required and existing fixture formats remain byte compatible. Duplicate imports converge on one hash object. Listing validates content and can fail on corruption instead of hiding it. Exported files are not overwritten; the user must choose a new destination. The official dialog plugin adds Rust/JavaScript dependencies and platform dialog code, but no general renderer filesystem plugin or permission is enabled.

Native component/build tests verify the boundary compiles and the engine bridge negotiates. Automated interaction with macOS open/save/confirmation panels remains unverified; a signed/notarized distribution remains outside Phase 5.
