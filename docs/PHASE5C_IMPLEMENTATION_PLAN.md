# Phase 5C implementation plan — fixture lifecycle and persistence

## Scope and foundation

Phase 5C is authorized as part of the continuous Phase 5 effort. It adds private lifecycle storage for the existing Phase 5A regression fixture and Phase 5B synthetic transport fixture without changing either format, event v2, SQLite migration 3 or protocol major 1.

## Design

Use a dedicated `internal/fixtures` service rooted at the existing private application-data directory. Canonical fixture bytes are stored in `fixtures/v1/sha256/<sha256>.fixture`; the filename is generated from validated bytes and never accepted as a path. Metadata is derived from validated content, keeping the immutable content authoritative and avoiding a database migration or mutable catalog.

Publication writes a private exclusive temporary file, syncs it, renames without overwriting a different object, and syncs the directory. Existing hash objects are re-read and compared, making duplicate imports deterministic. Reads validate filename, type, size, file identity/permissions, content hash and the strict format loader. Corruption is reported and never repaired. Inventory is bounded; partial temporary files are never treated as fixtures. Delete accepts only a digest identity and requires explicit consent. Export returns detached canonical bytes only after explicit consent.

The service supports validation/import, run eligibility, creation from an eligible Phase 5A run, success-only persistence from an existing Phase 5A evaluation result, list, inspect, export and delete. Inspection reports supported interaction counts/types and fixed privacy warnings before export. No raw prompt or credential field is added.

## Trust and UI boundary

Backend IPC will carry fixture text/identities only and will never accept import/export paths. In Phase 5E the trusted Tauri host will own native open/save dialogs, bounded file reads, exclusive destination creation and native confirmation for export/delete. Renderer booleans alone will not stand in for trusted confirmation.

## Tests and compatibility

Tests cover stable create/import/list/inspect/export/delete, duplicates, restart, mutation copies, traversal identities, permissions, partial temp files, corruption, unsupported/oversized fixtures, failed evaluation refusal and no overwrite. Phase 5A/5B tests remain unchanged and must pass. Phase 5C receives a separate status/report before proceeding to 5D.
