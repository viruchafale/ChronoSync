# macOS DMG release

ChronoSync uses Tauri's macOS bundler and packages the Go engine as an application resource. The current configuration produces an ARM64 build with bundle identifier `dev.chronosync.desktop` and application version `0.0.0`.

## Build

Install the checked-in dependencies and the Rust components listed in the root README, then run from the repository root:

```sh
make check
make native-check
npm run desktop:bundle:dmg
```

The DMG is written to:

```text
apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg
```

The Tauri build hook rebuilds `apps/desktop/src-tauri/binaries/chronosync-engine` before compiling the frontend and desktop host. The engine is bundled at `ChronoSync.app/Contents/Resources/binaries/chronosync-engine` and must remain executable.

The repository configuration does not contain a Developer ID signing identity, notarization credentials, or an update channel. The command above creates a local unsigned DMG. It is suitable for local verification but is not ready for public distribution through normal macOS Gatekeeper flows.

## Phase 10 release-candidate profile

The checked-in `tauri.release.conf.json` is a credential-neutral overlay. It
requires hardened runtime, macOS 10.15, a generated entitlement file and an
externally supplied provisioning profile. It intentionally contains no
certificate name or TeamIdentifier. The profile is not used by normal
development/native checks.

On an authorized release Mac, inspect the available identities and obtain the
intended TeamIdentifier/profile from the Apple Developer account:

```sh
security find-identity -v -p codesigning
node scripts/macos-release-identity.mjs render \
  --team-id "$APPLE_TEAM_ID" \
  --output apps/desktop/src-tauri/Entitlements.release.plist
```

Place the matching externally managed profile at
`apps/desktop/src-tauri/embedded.provisionprofile`. Both generated files are
ignored by Git. Build an app candidate into a disposable target, not the
submitted artifact location:

```sh
export APPLE_SIGNING_IDENTITY='Developer ID Application: reviewed external identity'
export CHRONOSYNC_RELEASE_SIGN_SIDECAR=1
export CARGO_TARGET_DIR=/tmp/chronosync-release-target
cd apps/desktop
tauri build --bundles app --config src-tauri/tauri.release.conf.json
```

The sidecar hook refuses an absent/ad-hoc identity and signs the engine first
with identifier `dev.chronosync.engine`. Tauri then signs the outer app. Verify
the completed candidate using the expected team:

```sh
node scripts/macos-release-identity.mjs verify \
  --team-id "$APPLE_TEAM_ID" \
  --app /tmp/chronosync-release-target/release/bundle/macos/ChronoSync.app
```

The verifier requires strict valid signatures, Developer ID Application
authority, hardened runtime, matching TeamIdentifiers, the stable app and
engine identifiers, expected signed entitlements and a matching embedded
profile. It does not notarize, staple or establish Keychain prompt behavior.
Those remain separate acceptance gates. Never commit the certificate, profile,
rendered entitlements or notarization credentials.

## Verify

Set `artifact` to the generated path, then verify the image before mounting it:

```sh
artifact=apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg
shasum -a 256 "$artifact"
hdiutil verify "$artifact"
```

Mount the image read-only and inspect the bundle:

```sh
mkdir -p /tmp/chronosync-dmg-mount
hdiutil attach -readonly -nobrowse -mountpoint /tmp/chronosync-dmg-mount "$artifact"
plutil -p /tmp/chronosync-dmg-mount/ChronoSync.app/Contents/Info.plist
file /tmp/chronosync-dmg-mount/ChronoSync.app/Contents/MacOS/chronosync-desktop
file /tmp/chronosync-dmg-mount/ChronoSync.app/Contents/Resources/binaries/chronosync-engine
/tmp/chronosync-dmg-mount/ChronoSync.app/Contents/Resources/binaries/chronosync-engine version
hdiutil detach /tmp/chronosync-dmg-mount
```

For a release intended for other Macs, add an approved Developer ID Application signing and notarization process, verify the sealed nested engine and app with `codesign`, assess the result with `spctl`, staple the notarization ticket, and repeat the DMG mount and launch checks. Keep credentials outside the repository and CI logs.

## Phase 11 acceptance preflight and continuity

Before any signed construction, run `npm run release:preflight`. It emits only platform/deployment metadata, identity counts and typed blockers. It does not select an identity or infer a TeamIdentifier. A ready result still requires operator confirmation that the existing identity is authorized.

After strict verification of two separately built disposable candidates, validate stable app/engine identifiers and TeamIdentifier with:

```sh
node scripts/macos-release-identity.mjs verify-continuity \
  --team-id "$APPLE_TEAM_ID" \
  --candidate-a /private/disposable/A/ChronoSync.app \
  --candidate-b /private/disposable/B/ChronoSync.app
```

On the current host the preflight reports zero valid identities, so no candidate construction, protected positive test, notarization or Gatekeeper claim is authorized. See [the acceptance guide](MACOS_RELEASE_ACCEPTANCE.md).

The evidence for the currently generated local artifact is in the [submission-build status](IMPLEMENTATION_STATUS_SUBMISSION_BUILD.md).
