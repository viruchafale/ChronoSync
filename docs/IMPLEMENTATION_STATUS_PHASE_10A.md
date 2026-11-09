# Phase 10A implementation status — release identity and entitlement architecture

## Status

`COMPLETE — architecture and credential-neutral configuration`

The release boundary is specified in ADR-0015, the Phase 10 threat model, the release-identity guide and the 16-section plan. The external distribution class is Developer ID Application; its concrete certificate name, TeamIdentifier and provisioning profile remain external inputs and are not invented or committed.

## Implemented

- Added a release-only Tauri overlay requiring hardened runtime, macOS 10.15, a generated entitlement file and an embedded provisioning profile.
- Added an entitlement template for the stable app identifier `dev.chronosync.desktop` and the app's own access group.
- Added tooling that rejects absent/placeholder TeamIdentifiers, creates the entitlement file exclusively with mode `0600`, refuses overwrite, and checks the release overlay.
- Kept the engine outside the Keychain access group; release verification instead requires the reviewed identifier `dev.chronosync.engine` and the same signing team.
- Kept development/ad-hoc and distribution builds explicitly distinct.

## Evidence and limits

`npm run test:release-config` passed. Both the template and a disposable rendered plist passed `plutil -lint`; a second render refused overwrite. `security find-identity -v -p codesigning` reported `0 valid identities found`. Current development host/engine binaries are ad-hoc, without a stable TeamIdentifier or release entitlements. Configuration is structurally validated, not evidence of OS authorization, Developer ID signing, notarization or Gatekeeper acceptance.
