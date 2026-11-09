# Phase 11 completion report — authorized macOS release-security acceptance

## 1. Overall Phase 11 status

`BLOCKED — credential-neutral and native-negative acceptance complete; authorized signing environment absent`

Phase 11A preflight and every safe Phase 11G regression are complete. Phase 11B–11F signed construction, positive protected signing, migration, A/B upgrade, notarization and Gatekeeper tracks cannot begin because this host has zero valid code-signing identities and no explicit authorized team/profile/notarization inputs. Production remains Phase 9 login-Keychain custody.

## 2. Authorized signing identity availability

Unavailable. `security find-identity -v -p codesigning` returned `0 valid identities found`. The preflight reports zero Developer ID Application identities and does not select or print certificate subject details.

## 3. Certificate and TeamIdentifier verification

Blocked/unverified. There is no certificate chain or private-key authorization to verify, and no TeamIdentifier was supplied. The tooling validates the required 10-character syntax but never invents a value.

## 4. Signed candidate construction results

Blocked. No Candidate A or B was built. Existing release configuration, nested-engine signing order and strict bundle verifier remain structurally tested. No existing app, artifact or installation was replaced.

## 5. Entitlement verification

Structural only. Template placeholder count, app identifier, access-group shape, exclusive rendering, release overlay and expected profile location pass tests. There is no rendered real entitlement/profile and no signed-entitlement OS authorization evidence.

## 6. Positive user-presence result

Blocked/unverified. A positive result requires the authorized signed host. The unsigned host continues to return `errSecMissingEntitlement` (`-34018`), which is native-negative evidence only.

## 7. Authorization success, cancellation and denial results

Typed mappings are tested structurally. Actual success/cancel/deny prompts were not displayed or exercised because no authorized candidate exists. Manual GUI acceptance is unverified.

## 8. Protected Keychain signing results

No protected identity was created or retrieved and no protected challenge was signed. Malformed input and unsupported access fail closed; no unauthenticated fallback was added. Existing Go one-use sessions and Rust login-Keychain signing remain unchanged.

## 9. Phase 9 identity migration results

No data-protection migration was executed. The design retains the login item until protected retrieval proves the same public key/key ID and an engine challenge signature. No cleanup, rotation, duplicate identity or trust inheritance occurred. Existing Phase 9 login migration/recovery regressions remain the available evidence.

## 10. Historical signature compatibility

No format or cryptographic contract changed. Phase 8 exchange package v1, Ed25519 verification, public identity v1, key IDs, trust and revocation semantics remain unchanged. Their race/regression suites are part of `make check`.

## 11. Upgrade identity continuity

Structural continuity validation accepts matching nonempty app/engine identifiers and team and rejects a changed team after strict candidate verification. Actual Candidate A → B Keychain/public-key/trust/revocation/migration continuity is blocked.

## 12. Notarization result

Blocked. No signed candidate or authorized notarization credential exists. No upload or submission occurred and no notarization claim is made.

## 13. Gatekeeper result

Blocked/unverified. No notarized/stapled disposable candidate exists. The historical submitted DMG's prior unsigned status is unchanged and is not Phase 11 acceptance evidence.

## 14. Full regression results

Credential-neutral release tests and preflight pass. The focused real unsigned protected-Keychain test passed by observing the exact expected refusal; the focused Phase 9 disposable login-Keychain create/retrieve/sign/delete test also passed. `go mod verify` passed; the locked native dependency remains `security-framework` 3.7.0; offline npm audit reported zero vulnerabilities across 253 dependencies. A source audit found the protected API only in the test-only Rust module registered under `#[cfg(test)]`. No provider calls or historical commands were made.

## 15. `make check` result

Passed on 2026-11-04 after one corrected test-only lint failure. The first run found `structuredClone` absent from the configured ESLint globals; the focused check was changed to explicit object copies without weakening lint. The rerun passed Go formatting, vet, all race tests and build; Prettier, ESLint and strict TypeScript; 8 Vitest files/34 tests; Vite build; twelve-plan documentation validation; release preflight/config tests; executable/schema integration; and synthetic recorder/snapshot/restart integration.

## 16. `make native-check` result

Passed on 2026-11-04: sidecar build, Rust format, Clippy with warnings denied, 6 default tests passed/2 intentionally ignored, and Tauri `--no-bundle` release-executable build. The pinned temporary `rust-objcopy` could not load `libLLVM.dylib`, so debug stripping emitted a non-fatal warning; compilation and the executable succeeded. The two ignored real Keychain tests were run separately with authorization and each passed.

## 17. Files and modules changed

Phase 11 extends `scripts/macos-release-identity.mjs` and its focused check with non-sensitive preflight and candidate-continuity validation; adds an npm preflight command; adds the 16-section Phase 11 plan, threat model, acceptance guide, seven milestone files and this report; and updates release/custody/architecture/security/testing/status/changelog/README/contract documentation. No runtime custody module, renderer command, Go protocol or database code changed for Phase 11.

## 18. Security limitations

Preflight readiness is not operator authorization, signed-build proof or OS enforcement. User presence is not civil identity or same-user-malware resistance. Ed25519 seed bytes would still enter authorized native memory. Certificate/profile policies, prompt behavior, two-store migration, update continuity, notarization and clean-machine Gatekeeper behavior remain unverified.

## 19. Manual GUI acceptance status

Not run. No authorized signed application existed from which a meaningful protected-Keychain prompt could be assessed.

## 20. Submitted DMG checksum confirmation

The existing submitted file was not built, signed, notarized, uploaded or replaced. It remains 9,479,647 bytes with timestamp `2026-10-25T21:17:17+0530` and SHA-256:

```text
e4ecbc78744bacf05605588e5f268016a89e487c778be0df4a4199e49c6c8d05
```

## 21. Remaining blockers

1. Operator-authorized Developer ID Application certificate and accessible private key.
2. Actual TeamIdentifier plus matching authorized provisioning profile/access group.
3. Explicit operator approval/use of that identity for disposable Candidates A/B.
4. Interactive positive/success/cancel/deny acceptance in the signed host.
5. Successful disposable same-key migration and A/B continuity evidence.
6. Separately authorized notarization credentials/upload and clean-machine Gatekeeper assessment.

## 22. Recommended Phase 12 direction

Do not start Phase 12 automatically. First resume this blocked Phase 11 acceptance on an authorized release Mac. Only after signed/native-positive/migration/upgrade/notarization/Gatekeeper evidence exists should a separately authorized phase consider production activation, cleanup policy or release automation.

## Evidence classification

| Track                             | Evidence                               | Result                         |
| --------------------------------- | -------------------------------------- | ------------------------------ |
| Environment                       | Actual public tool output              | Blocked: zero valid identities |
| Release configuration             | Structural tests                       | Passed                         |
| Candidate continuity logic        | Structural synthetic metadata          | Passed; real A/B blocked       |
| Protected API unsigned context    | Native negative                        | Passed: exact `-34018` refusal |
| Phase 9 disposable login Keychain | Native positive for login custody only | Passed                         |
| Protected signing/prompt          | Signed/native positive/manual          | Blocked                        |
| Same-key protected migration      | Signed/native positive                 | Blocked                        |
| Notarization/Gatekeeper           | External service/manual                | Blocked                        |
