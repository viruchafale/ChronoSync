# macOS release identity architecture

The intended external distribution identity class is `Developer ID Application`. The concrete identity and TeamIdentifier are external release inputs. ChronoSync source contains neither and release tooling must refuse an unset/placeholder value.

The stable app tuple is the signing TeamIdentifier plus bundle identifier `dev.chronosync.desktop`. The data-protection Keychain default application access group is expected to match that tuple. ChronoSync does not need to share seeds with the Go engine: only the Rust host accesses Keychain, so no broader helper access group is justified.

The release profile requires hardened runtime and a minimum macOS version of 10.15 for the data-protection Keychain selection used by the native probe. The entitlement template declares the application/team identifiers and only the app's own keychain access group. It does not enable App Sandbox, app groups, network, automation, debug, JIT or library-validation exceptions.

Signing order for an authorized candidate is deepest nested code first, then the outer app. The bundled Go engine must have a reviewed identifier and the same team; it receives no Keychain entitlement. The final verifier checks strict signatures, non-ad-hoc status, TeamIdentifier equality, app identifier, hardened-runtime flag and signed entitlements. A config file or successful unsigned build does not satisfy these checks.

Upgrades must retain the same TeamIdentifier, bundle identifier and access group. Changing any member can strand protected items. Certificate renewal within the same authorized team is conceptually compatible but remains an actual upgrade acceptance test, not an assumption.

Phase 11 adds a non-sensitive preflight and strict Candidate A/B continuity check. The current host reports zero valid identities, so it produced no signed candidate and established no certificate, team, access-group or upgrade continuity evidence. Structural readiness is not promoted to distribution acceptance.
