# Phase 11 — Authorized macOS release-security acceptance

## 1. Objective and user-visible outcome

Attempt the real Developer ID, data-protection Keychain, same-key migration, update-continuity and notarization acceptance path without changing existing identities, trust, packages or the submitted DMG. When credentials are absent, complete every credential-neutral gate and record the exact external prerequisite instead of simulating positive evidence.

## 2. Problem

Phase 10 proved that the unsigned host fails closed with `errSecMissingEntitlement` and prepared release configuration, but it could not prove an authorized signature, access group, user-presence prompt, protected signing, migration or update continuity. On 2026-11-04 this host still reports zero valid code-signing identities, so every signed positive track is externally blocked.

## 3. Scope and non-goals

Phase 11A–11G cover environment preflight, isolated signed-candidate validation, real protected-Keychain acceptance, disposable same-key migration, two-candidate continuity, notarization/Gatekeeper evidence and full regressions. There is no certificate enrollment/purchase, invented TeamIdentifier, credential capture, actual-app replacement, submitted-DMG rebuild, destructive legacy cleanup, provider/agent execution, Phase 4 authority change, organization PKI, remote trust or Phase 12 work.

## 4. Prerequisites and dependencies

Preserve Phase 8 exchange package v1, Ed25519 verification and trust; Phase 9 login-Keychain custody, migration and recovery; and Phase 10 release/protected-Keychain boundaries. Positive work requires an operator-authorized Developer ID Application identity with private-key access, its actual 10-character TeamIdentifier, a matching authorized profile/access group, and notarization credentials only for the separate upload track. Reuse locked dependencies; add none without demonstrated need.

## 5. Design and component boundaries

Credentials remain in the operator's macOS Keychain/external release environment. Source stores only templates and validators. Candidates A/B belong in separate disposable target/install paths. Rust remains the only Keychain boundary, Go remains authoritative for exact one-use payloads and final verification, and React receives no seed or arbitrary signing input. Phase 9 login items and current trust/public identity stores are never acceptance-test targets.

## 6. Ordered implementation tasks

1. Freeze environment, Git, certificate/profile/toolchain and submitted-DMG evidence.
2. Add a non-sensitive release-acceptance preflight and structural tests.
3. If authorized identity/profile exist, build and strictly verify isolated candidate A; otherwise record the prerequisite and skip construction.
4. Only inside that signed candidate, run protected create/retrieve/sign success, cancel and deny acceptance without capturing secrets.
5. Only after positive protected proof, run disposable same-key migration/restart/idempotency acceptance while retaining the login item.
6. Build candidate B with the same identity and test continuity without replacing the installed app.
7. Only with separately available notarization authorization, submit the disposable candidate, staple and assess Gatekeeper.
8. Run every safe regression, complete seven status files and the consolidated report, then stop.

## 7. Expected files/modules

- Existing release identity script and focused check for credential-neutral preflight/continuity validation.
- Phase 11 threat/acceptance documentation and, if credentials become available, isolated native acceptance harness code.
- `IMPLEMENTATION_STATUS_PHASE_11A.md` through `11G.md` and `PHASE11_COMPLETION_REPORT.md`.
- Cumulative release, custody, testing, status, changelog, architecture and AGENTS documentation.

## 8. Interfaces and contracts

Preflight reports only platform, SDK/deployment target, valid signing-identity count, required external-file presence and a typed readiness/blocker; it never prints certificates' private material or credentials. Candidate validation consumes explicit app paths and expected TeamIdentifier. Protected signing retains Phase 10 outcomes and the existing Go one-use exact-payload contract. No new public IPC or durable migration transition is enabled without positive signed acceptance.

## 9. Security and failure considerations

An identity listing is not proof that its private key use is authorized; an actual signed disposable candidate and strict verification are required. A profile or entitlement plist is not OS-enforcement evidence. Acceptance must not touch production identities, login items, trust records or installed applications. Cancellation/denial/error never falls back. Upload/notarization is a separate externally authorized mutation and remains blocked when credentials are absent.

## 10. Unit, integration and end-to-end tests

Test preflight parsing, zero/multiple identity refusal, TeamIdentifier/profile/config validation, placeholder removal, expected signed metadata and candidate A/B continuity logic. Run the real unsigned negative protected test plus Phase 8–10 regressions. Positive native, signed-build and GUI tests run only in an authorized candidate and are labeled separately. No mock or structural check counts as user-presence, migration, notarization or Gatekeeper proof.

## 11. Acceptance criteria

Credential-neutral preflight fails closed with the exact missing prerequisite and exposes no credential. If an authorized environment is present, both candidates verify with one stable team/app/access group, protected signing requires actual user presence, same-key migration preserves identity/history, and notarization/Gatekeeper results are independently recorded. If absent, production remains Phase 9 custody and all dependent gates are `BLOCKED`, not `COMPLETE`.

## 12. Validation commands

```sh
security find-identity -v -p codesigning
npm run test:release-config
node scripts/macos-release-identity.mjs preflight
GOCACHE=/tmp/chronosync-go-cache GOMODCACHE=/Users/viru/go/pkg/mod make check
CARGO_HOME=/private/tmp/chronosync-phase5-cargo RUSTUP_HOME=/private/tmp/chronosync-phase5-rustup CARGO_NET_OFFLINE=true make native-check
```

Run signed-bundle, protected-positive, migration/continuity, notarization and Gatekeeper commands only when their real external prerequisites exist. Never bundle the submitted DMG.

## 13. Risks and unresolved questions

There is currently no valid Developer ID identity, known authorized TeamIdentifier/profile or notarization credential. Private-key access might require operator interaction even after installation. Profile requirements for the selected data-protection access group must be proven on the candidate. Two-store migration cannot be atomic. Prompt wording/denial semantics and clean-machine Gatekeeper behavior require manual real-system evidence.

## 14. Exit criteria

Every 11A–11G milestone has a separate status file; all safe checks and regressions pass; signed/manual/upload tracks are either genuinely evidenced or explicitly blocked; production protected signing stays disabled without proof; the submitted DMG checksum is unchanged; no credentials or real TeamIdentifier enter source; no Phase 12 work begins.

## 15. Documentation updates

Update the Phase 11 plan, threat model, release/custody/migration/testing guides, seven status files, consolidated 22-section report, README/status index/changelog/architecture/contracts/AGENTS. Record commands and distinguish structural, negative-native, positive-native, signed, manual, notarization, Gatekeeper, blocked and unverified evidence.

## 16. Status

`BLOCKED`

Authorized on 2026-11-04. Credential-neutral preflight, continuity validation, native-negative tests and full regressions pass. Environment preflight found zero valid signing identities, so Candidate A/B construction, protected positive signing, migration, notarization, Gatekeeper and manual acceptance remain externally blocked. No Phase 12 work is authorized.
