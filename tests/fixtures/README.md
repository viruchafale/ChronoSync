# Fixture policy

No live captures are included in Phase 0. Add parser fixtures in Phase 1 with provenance (`synthetic`, `official-example`, or `sanitized-live`), adapter/CLI version, schema version, license/source if applicable, redaction version and capture limitations. Never copy private sessions or auth material. A synthetic fixture must never be used as evidence of a successful live integration.

Proposed P5 directory: `manifest.json`, `events.jsonl`, `interactions.jsonl`, `objects/`, `assertions.json`. Manifest declares fixture version, hashes, platforms, adapter capabilities and unmatched-interaction policy. Portable paths are relative and validated; archive imports must reject symlinks/traversal and enforce size limits. Assertions cover lifecycle, output, file hashes and ordering; sensitive missing/redacted data explicitly reduces replay scope.
