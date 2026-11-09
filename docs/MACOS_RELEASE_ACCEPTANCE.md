# Authorized macOS release-security acceptance

## Evidence ladder

1. **Structural:** template/config/parser/preflight tests; no OS enforcement claim.
2. **Native negative:** actual unsigned API refusal; no positive prompt claim.
3. **Signed build:** strict app/engine signature, team, identifier, entitlement and profile verification.
4. **Native positive:** protected item creation/retrieval and exact challenge signing inside that signed build.
5. **Manual GUI:** operator observes success, cancellation and denial prompts without recording secrets.
6. **Upgrade:** Candidate B with the same identity retrieves Candidate A's disposable protected item and preserves key/package/trust state.
7. **External service:** notarization accepted, ticket stapled and Gatekeeper assessed separately.

No lower rung proves a higher one.

## Credential-neutral preflight

Run:

```sh
npm run release:preflight
```

The JSON contains only OS/SDK/deployment metadata, identity counts and typed blockers. It intentionally does not choose an identity, infer a TeamIdentifier or print certificate subject details. Readiness requires all of:

- an operator-authorized Developer ID Application identity with private-key access;
- explicit `APPLE_SIGNING_IDENTITY` and actual 10-character `APPLE_TEAM_ID` values;
- externally supplied rendered entitlements and matching provisioning profile;
- macOS and SDK tooling compatible with the 10.15 deployment floor.

Readiness is necessary, not sufficient. The actual candidate must still pass strict verification.

## Isolated candidate sequence

Use a freshly created acceptance target outside the submitted artifact tree. Render entitlements exclusively, sign the engine first, let Tauri sign the outer app, and run `verify`. Build Candidate B into another target and run `verify-continuity` against A/B. Never replace the installed app, existing DMG, login identity, public identity or trust store.

The current Phase 10 release guide contains the concrete build command. The continuity command is:

```sh
node scripts/macos-release-identity.mjs verify-continuity \
  --team-id "$APPLE_TEAM_ID" \
  --candidate-a /private/disposable/A/ChronoSync.app \
  --candidate-b /private/disposable/B/ChronoSync.app
```

## Protected and migration acceptance

Positive tests must execute from the strictly verified candidate and use a disposable service/account and identity. Record OSStatus and typed outcomes, never seeds/passwords/prompts. Test success, cancellation, denial, missing/malformed/mismatch, one-use sessions and no fallback. Migration additionally retains the login item, proves unchanged public key/key ID and an engine challenge, tests restart/idempotency and performs no cleanup.

## Notarization and Gatekeeper

Notarization is an explicit external upload and requires separately available authorization. Submit only the disposable candidate/artifact after signed/native acceptance. Record Apple's result, staple and validate the ticket, then assess Gatekeeper. Do not notarize or replace the submitted DMG. A successful signature alone is not notarization or Gatekeeper acceptance.

## Current status

Preflight is blocked because this host has zero valid signing identities and no external team/profile inputs. No candidate, prompt, migration, upload, stapling or Gatekeeper test was performed.
