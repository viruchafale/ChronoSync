# Phase 10 release identity and user-presence threat model

## Assets and boundaries

Assets are Phase 8/9 Ed25519 seeds and identities, Keychain access groups, release signing identity, entitlement authorization, protected-item state, one-use signing payloads, trust/revocation history and historical packages. Boundaries are renderer → typed native workflow, Go signing session → Rust broker, Rust → Security.framework, release tooling → codesign output, and update candidate → previously installed identity/access group.

## Threats and controls

| Threat                                     | Required control                                                                       | Residual risk                                                             |
| ------------------------------------------ | -------------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| Entitlement text mistaken for enforcement  | Verify actual signed entitlements, TeamIdentifier and native API behavior              | Apple account/profile configuration can still change                      |
| Ad-hoc build mistaken for release identity | Refuse `Signature=adhoc`, absent team or placeholder identity in release verifier      | Stolen valid Developer ID remains trusted by code checks                  |
| Renderer requests arbitrary signing        | Preserve engine-created one-use exact payload; no generic sign command                 | Trusted host/engine compromise                                            |
| Fake presence ceremony                     | Keychain item itself requires `USER_PRESENCE`; no login-item fallback                  | Same-user malware controlling the authorized process after authentication |
| Key loss during upgrade                    | Separate service, retain old item through proof/activation, no auto cleanup            | Two stores cannot commit atomically                                       |
| Identity substitution                      | Derive same Ed25519 public key/key ID and verify challenge signature                   | Extractable seed exists briefly in native memory                          |
| Cancel/deny ambiguity                      | Closed typed outcomes and no signature/session success                                 | OS prompt wording/mechanism varies                                        |
| Update loses access                        | Stable team, bundle ID, application identifier/access group and explicit upgrade test  | Certificate/team/provisioning mistakes can strand items                   |
| Helper/engine identity mismatch            | Sign nested engine first and verify same expected team; engine never accesses Keychain | Tauri/build changes require revalidation                                  |
| Credential leakage                         | Credentials/profiles remain external; tooling prints metadata only                     | CI/keychain compromise is outside repository controls                     |

## Evidence classes

Mock tests validate mappings only. Structural tests validate files and refusal logic only. Native negative tests prove actual API refusal in the tested unsigned context. Native positive tests require a prompt and successful protected retrieval. Signed-build acceptance additionally requires actual certificate/team/entitlement verification. Manual GUI acceptance covers prompt presentation/accessibility, not cryptographic semantics.

## Non-goals

User presence is not civil identity, informed consent, same-user-malware resistance, hardware-backed Ed25519, execution authorization, notarization, global trust or cross-platform enforcement. The current environment cannot establish positive user-presence or release continuity because it has no valid signing identity or authorized profile.
