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

The evidence for the currently generated local artifact is in the [submission-build status](IMPLEMENTATION_STATUS_SUBMISSION_BUILD.md).
