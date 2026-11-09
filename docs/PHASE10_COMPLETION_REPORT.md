# Phase 10 completion report — release identity and user-presence enforcement

## 1. Overall Phase 10 status

`BLOCKED — safe structural and native-negative work complete; signed positive acceptance unavailable`

Phase 10A release architecture, Phase 10B feasibility probing and all locally available Phase 10F regressions are complete. Phase 10C positive user-presence signing, Phase 10D production migration and Phase 10E signed upgrade continuity remain deliberately disabled. This machine has no valid Developer ID identity, known authorized TeamIdentifier or provisioning profile, so manufacturing a positive result would be unsafe and misleading.

Evidence labels in this report are: **implemented**, **design-only**, **structural**, **native API**, **signed-build**, **manual**, and **unverified**. Structural or mock results never count as signed-build or user-presence proof.

## 2. Release identity architecture

The distribution identity class is Developer ID Application. The stable app identity is the external TeamIdentifier plus bundle ID `dev.chronosync.desktop`; its default access group must match the authorized application identifier. The Go engine uses code identifier `dev.chronosync.engine`, is signed before the outer app with the same team, and receives no Keychain entitlement. The outer app owns Keychain access. Hardened runtime is required. Credentials and the concrete team remain external.

## 3. macOS signing feasibility

Credential-neutral release configuration and verification are **implemented/structural**. `security find-identity -v -p codesigning` returned `0 valid identities found`. Current development code is ad-hoc, without a TeamIdentifier or signed release entitlements. Actual Developer ID signing, notarization, stapling and Gatekeeper acceptance are **unverified**, not failed simulations.

## 4. Keychain access-control findings

The real Security.framework probe requests a data-protection generic-password item with `AccessibleWhenUnlockedThisDeviceOnly` and `USER_PRESENCE`. In the actual unsigned/unentitled process it returned OSStatus `-34018`; the typed result is `unsupported_access_control`. This is **native API** evidence of fail-closed refusal in this context. It is not evidence of positive prompt behavior. The mechanism preserves Ed25519 seed compatibility but remains extractable in authorized native memory; it is not Secure Enclave or hardware-backed Ed25519.

## 5. User-presence enforcement status

Positive enforcement is **blocked/unverified**. The production broker remains Phase 9 login-Keychain custody and is not relabeled protected. No protected renderer command or fallback path was added. A future production path may activate only after a real signed candidate creates and retrieves the protected item and signs an engine-owned exact challenge after user presence.

## 6. Signing and authorization behavior

The closed outcome vocabulary is `signed`, `user_presence_required`, `authorization_cancelled`, `authorization_denied`, `key_missing`, `identity_mismatch`, `unsupported_access_control`, `unsupported_platform`, `malformed_request`, and `internal_error`. Status mappings and malformed refusal are **implemented/tested**. Success/cancel/deny prompt behavior is **unverified**. Existing Go one-use exact-payload signing sessions, Rust-only seed access and final Go verification remain unchanged.

## 7. Upgrade and migration behavior

The same-key state machine is **design-only**. It keeps login and protected items separate, requires explicit confirmation, protected retrieval, unchanged public-key/key-ID derivation and challenge verification before activation, and never deletes the login item automatically. Cancellation, restart, missing keys and incompatible access control cannot activate protected custody. No production migration or database state was added because the prerequisite positive native evidence is absent.

## 8. Identity continuity results

Existing Phase 8/9 public identities, key IDs, historical signatures, trust/revocation semantics and package v1 remain unchanged and their regressions pass. Signed app-upgrade continuity is **unverified**. If same-key protected storage proves impossible, replacement must create a distinct identity with explicit confirmation and no inherited trust.

## 9. Entitlement configuration

`Entitlements.release.plist.template` contains placeholders for the external team, app identifier and app-only Keychain access group. The release overlay requires the rendered plist, hardened runtime, macOS 10.15 and an external embedded provisioning profile. Rendering rejects missing/placeholder/invalid team values, uses exclusive creation and refuses overwrite. The final verifier checks the signed entitlements and decoded profile. This is **structurally validated**; OS authorization is **unverified**.

## 10. Supported platforms

The safe configuration and native probe target macOS. Phase 9 already reports Windows/Linux OS-backed custody unsupported; Phase 10 adds no cross-platform user-presence claim. Development/ad-hoc builds explicitly report protected access unavailable.

## 11. Database, protocol and dependency changes

There is no SQL migration, protocol-major change, package-format change, engine capability change or Phase 10 dependency addition. Locked `security-framework` 3.7.0 is reused. Phase 8 package and Phase 9 public-identity v1 remain authoritative.

## 12. Files and modules changed

Release implementation is in `apps/desktop/src-tauri/tauri.release.conf.json`, `Entitlements.release.plist.template`, `scripts/macos-release-identity.mjs`, its focused check, and release-aware sidecar signing. The isolated probe is `apps/desktop/src-tauri/src/protected_keychain.rs`. ADR-0015, the Phase 10 plan/threat model, release guide, user-presence/upgrade contract, six milestone reports and cumulative repository documentation record the boundary. Generated entitlement and provisioning files are ignored.

## 13. Native API test results

- Rust default suite: 6 passed, 0 failed, 2 intentionally ignored real-Keychain tests.
- Focused protected-Keychain real test: passed; exact unsigned-context result `unsupported_access_control(-34018)`.
- Focused existing Phase 9 login-Keychain round trip: passed separately, confirming normal custody still works.
- Rust format and Clippy with warnings denied: passed.

The real protected test is negative native evidence. Positive prompt success/cancel/deny remains unavailable.

## 14. Signed-build acceptance results

The template, parser, release overlay, refusal behavior and expected identifiers pass structural checks. Release sidecar signing refuses missing/ad-hoc identity. No signed candidate exists, so final strict app/engine signature verification, authority/team/access-group checks, upgrade continuity, signed package export, notarization and Gatekeeper checks are **blocked/unverified**.

## 15. Full `make check` results

Passed on 2026-11-04: Go formatting, vet, race tests and build; Prettier, ESLint and strict TypeScript; 8 Vitest files/34 tests; Vite build; documentation validation; release-config validation; executable/schema integration; recorder/snapshot/restart integration. Integrations used synthetic local processes only and made no provider call.

## 16. Full `make native-check` results

Passed on 2026-11-04: sidecar build, `cargo fmt --check`, Clippy with warnings denied, Rust tests and Tauri `--no-bundle` release executable build. `rust-objcopy` emitted a non-fatal warning because the pinned temporary toolchain cannot load `libLLVM.dylib`; stripping was skipped, but compilation and the executable succeeded. No DMG was built.

## 17. Manual GUI acceptance status

**Not run/unverified.** A meaningful prompt acceptance requires an authorized signed build. No prompt screenshot or simulated dialog is treated as security evidence.

## 18. Security limitations and unverified behavior

User presence is not civil identity, informed consent or protection from malicious code controlling the authorized same-user process. The Ed25519 seed remains extractable in native memory after authorization. Access-group behavior, positive/cancel/deny prompts, protected signing, migration recovery, update continuity, notarization and Gatekeeper behavior are unverified. The existing submitted DMG is a historical ad-hoc development artifact, not upgraded by this work.

## 19. Submitted DMG checksum confirmation

The existing file `apps/desktop/src-tauri/target/release/bundle/dmg/ChronoSync_0.0.0_aarch64.dmg` was not rebuilt or replaced. It remains 9,479,647 bytes with filesystem timestamp `2026-10-25T21:17:17+0530` and SHA-256:

```text
e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05
```

## 20. Recommended Phase 11 direction

Do not begin Phase 11 automatically. The next separately authorized security milestone should provision a disposable, authorized Developer ID release environment; render the real team entitlement outside source control; build and verify nested signatures; exercise protected create/retrieve success, cancel and deny; test same-key restart-safe upgrade and old-package verification; then evaluate notarization/Gatekeeper separately. Only positive evidence should enable production migration.

## Phase 10F acceptance matrix

| #   | Scenario                                 | Evidence class                         | Result                                                                |
| --- | ---------------------------------------- | -------------------------------------- | --------------------------------------------------------------------- |
| 1   | Signing with user presence               | Signed-build/native positive           | Blocked: no authorized signed context                                 |
| 2   | Successful authorization                 | Signed-build/native positive/manual    | Blocked: no authorized signed context                                 |
| 3   | User cancellation                        | Unit mapping; native positive required | Typed mapping passed; real prompt unverified                          |
| 4   | User denial                              | Unit mapping; native positive required | Typed mapping passed; real prompt unverified                          |
| 5   | Missing key                              | Unit mapping                           | Passed                                                                |
| 6   | Keychain access failure                  | Unit mapping/native negative           | Passed mapping; actual `-34018` refusal also observed                 |
| 7   | Unsupported access-control configuration | Native API                             | Passed: exact `unsupported_access_control(-34018)`                    |
| 8   | Malformed signing request                | Unit                                   | Passed before Keychain access                                         |
| 9   | Repeated signing requests                | Phase 9 regression/race                | Passed existing one-use session regression                            |
| 10  | Signing identity mismatch                | Phase 9 regression                     | Passed existing public-key/signature proof checks                     |
| 11  | Existing login-Keychain identity upgrade | Design-only                            | Blocked; no migration enabled                                         |
| 12  | Preservation of existing public identity | Regression/design                      | Current identity unchanged; signed migration unverified               |
| 13  | Preservation of historical signatures    | Regression/design                      | Old package verification passed; signed migration unverified          |
| 14  | Interrupted migration                    | Design-only                            | State and retention rule specified; execution unverified              |
| 15  | Restart during migration                 | Design-only                            | State and retention rule specified; execution unverified              |
| 16  | Migration idempotency                    | Design-only                            | Required by contract; execution unverified                            |
| 17  | Authorization cancellation in migration  | Design-only                            | Must retain login item/no activation; execution unverified            |
| 18  | Missing legacy key                       | Design-only/typed outcome              | Must fail closed; production migration unverified                     |
| 19  | Incompatible access-control mode         | Native negative/design                 | Actual unsupported result observed; migration remains disabled        |
| 20  | Explicit identity replacement            | Existing Phase 8 semantics/design      | New identity/no trust inheritance specified; no replacement performed |
| 21  | Ad-hoc development build behavior        | Actual inspection/native negative      | Passed: no team/entitlements; protected API refused                   |
| 22  | Stable signing identity configuration    | Structural                             | Passed configuration/refusal; actual identity unavailable             |
| 23  | Entitlement validation                   | Structural                             | Template/render/config passed; signed entitlement unverified          |
| 24  | Bundle signing verification              | Signed-build                           | Blocked: no candidate                                                 |
| 25  | Engine/helper signing identity           | Structural/actual development inspect  | Verifier passed tests; current binaries ad-hoc; release unverified    |
| 26  | Upgrade identity continuity              | Signed-build                           | Blocked: no old/new authorized candidates                             |
| 27  | Old package verification after upgrade   | Regression; signed upgrade required    | Old-package verification passed; post-upgrade portion blocked         |
| 28  | Keychain access-group consistency        | Structural/signed-build                | Verifier logic passed; actual authorization blocked                   |
| 29  | Phase 8 signed package export            | Go race regression                     | Passed                                                                |
| 30  | Phase 8 package import                   | Go race regression                     | Passed                                                                |
| 31  | Trusted identity verification            | Go race regression                     | Passed                                                                |
| 32  | Untrusted identity confirmation          | Go race/UI regression                  | Passed                                                                |
| 33  | Revoked identity refusal                 | Go race regression                     | Passed                                                                |
| 34  | Tampered package refusal                 | Go race regression                     | Passed                                                                |
| 35  | Trust-store restart persistence          | Go race regression                     | Passed                                                                |
| 36  | Full project regression gates            | Portable/native                        | `make check` and `make native-check` passed                           |

No scenario called a provider or executed a historical command.
