# Phase 10E implementation status — signed-build identity continuity

## Status

`BLOCKED — structural verifier complete; distribution acceptance unavailable`

## Implemented

- Release sidecar signing refuses release mode without a non-ad-hoc external `APPLE_SIGNING_IDENTITY`, signs nested engine code first with hardened runtime and identifier `dev.chronosync.engine`, and leaves normal development behavior unchanged.
- The candidate verifier checks strict app/engine signatures, Developer ID Application authority, expected TeamIdentifier, hardened-runtime flags, app and engine identifiers, signed app entitlements, embedded provisioning profile and matching authorized access group.
- Release documentation records build/sign/verify order and stable identity requirements without storing credentials.

## Evidence and limits

Parser, template, refusal and configuration tests pass. The actual machine reports zero valid code-signing identities; current artifacts are ad-hoc. Therefore bundle signing, nested-code identity, access-group authorization, upgrade continuity, package signing from a distribution build, notarization, stapling and Gatekeeper acceptance are unverified. Old Phase 8 package verification remains covered by the passing regression suite.
