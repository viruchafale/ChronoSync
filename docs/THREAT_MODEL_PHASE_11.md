# Phase 11 authorized release-acceptance threat model

## Assets and boundaries

Assets are the Developer ID private-key authorization, certificate/team identity, provisioning profile and access group, notarization authorization, disposable candidates, Phase 8/9 Ed25519 identities and signatures, protected/login Keychain items, trust/revocation history and the submitted DMG. Boundaries are operator release Keychain → build tools, external profile → signed app, nested engine → outer signature, signed host → data-protection Keychain, candidate A → candidate B, and notarization upload → Apple service.

## Threats and controls

| Threat                                     | Required control                                                                                                        | Residual risk                                         |
| ------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------- |
| Unauthorized certificate use               | Require an existing operator-authorized identity and actual private-key signing; never select another person's identity | Local account/Keychain compromise                     |
| Identity metadata treated as authorization | Build a disposable candidate and strictly verify its chain, team, identifiers and nested signatures                     | Certificate revocation/status changes after testing   |
| Credential leakage                         | Keep certificate keys, TeamIdentifier/profile and notarization secrets external; preflight emits counts/blockers only   | Shell/environment/CI compromise outside repository    |
| Acceptance overwrites production           | Separate target/install/service/account paths; refuse submitted artifact and existing identity stores                   | Operator supplies an unsafe path outside tooling      |
| Entitlement text treated as enforcement    | Require signed entitlements/profile plus positive protected create/retrieve/sign behavior                               | OS behavior/version changes require reacceptance      |
| Fake user-presence ceremony                | Protect the Keychain item itself with `USER_PRESENCE`; no login-item fallback                                           | Same-user malicious control after authentication      |
| Migration loses or rotates identity        | Retain login item, prove same public key/key ID and challenge signature before activation                               | Two Keychain stores cannot commit atomically          |
| Candidate B strands protected item         | Require same team, bundle ID and access group, then retrieve the same item after upgrade                                | Provisioning/certificate policy can change externally |
| Signing conflated with notarization        | Separate submission, result, stapling and Gatekeeper evidence                                                           | Gatekeeper policy and clean-machine state vary        |
| Structural/mock evidence overstated        | Label structural, native-negative, native-positive, signed, manual and external-service evidence separately             | Reviewers may ignore evidence labels                  |

## Current environment result

On 2026-11-04, macOS 27.2 arm64 with SDK 26.5 reports zero valid code-signing identities. No authorized TeamIdentifier, rendered entitlements, provisioning profile or notarization credential is present. The credential-neutral preflight therefore blocks candidate construction and all dependent positive tracks. No production identity, Keychain item, trust record, installed application or submitted DMG is touched.

## Non-goals

Phase 11 does not establish civil identity, informed consent, protection from same-user malware, hardware-backed Ed25519, organization PKI, remote trust, execution authorization or cross-platform user-presence. A future signed result applies only to the tested OS/build/team/profile combination.
