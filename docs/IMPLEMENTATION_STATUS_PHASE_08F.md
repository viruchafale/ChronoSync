# Phase 8F implementation status — adversarial validation

Disposable race/integration/component tests cover trusted/untrusted valid packages, content/signature tampering, unsupported algorithm/version, revoked identity, omitted/mismatched provenance, duplicate/traversal/symlink/compressed/oversized/excess-entry archives, malformed/duplicate-key/unknown-field manifests, interrupted and concurrent imports, rotation, trust/revocation restart, clean-machine import and independent re-signing.

Real storage integration creates Phase 6/7 evidence on one installation and imports it into a separate empty store without running commands or providers. Full gates, dependency/source audits, limitations and unchanged submitted-DMG checksum are recorded in the [Phase 8 completion report](PHASE8_COMPLETION_REPORT.md).

Acceptance evidence on 2026-11-01: focused exchange race tests passed all five groups (including the 25 requested scenarios and RFC 8032 vector), `make check` passed with 8 Vitest files/32 tests, and offline locked `make native-check` passed Clippy, 1/1 Rust test and a no-bundle release build. Direct prohibited-import audit passed. Native GUI automation and OS-level egress isolation remain unverified.
