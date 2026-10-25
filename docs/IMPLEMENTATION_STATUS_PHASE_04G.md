# Phase 4G experiment report — isolated authentication protocol

## Factor decision update — 2026-10-24

The user explicitly approved **macOS device-owner authentication**, including its supported password or Apple Watch factors, for the isolated Phase 4G prototype. The native policy is `LAPolicy.deviceOwnerAuthentication`; biometric-only is not required. This resolves the factor-policy gate recorded as pending in the original October 23 experiment below. It does not authorize a checkbox fallback, another policy, production approval, or launch.

Native bindings, XPC/coordinator interoperability and fixture signing/packaging remain separately unapproved under Phase 4G Sections 1 and 5. No native call, new dependency or artifact change was made by this decision update. The concrete proposal below remains the next approval item. Documentation checks passed; runtime tests were not rerun for this documentation-only decision.

Date: 2026-10-23. **Protocol model implemented; native authentication and peer-authenticated transport blocked at explicit approval gates.** No actual OS authentication or peer identity verification occurred. This is not completion of the full native experiment and does not support production integration. [Status index](IMPLEMENTATION_STATUS.md).

## Preflight and observed environment

Reviewed Phase 4F's conditional Rust-host/LocalAuthentication proposal and dedicated mutual XPC identity requirement, Phase 4A–4E evidence, ADR-0008, security/system/testing documents, current provider/admission/supervisor/IPC boundaries, and Rust/Tauri bridge, React consent and packaging. Existing uncommitted Phase 4E/F work was present; a file-hash baseline includes those untracked files, avoiding attribution of older code to this phase.

Observed `sw_vers`: macOS 26.6.2, build 25G83. `uname -m`: arm64. This establishes OS and architecture, not Touch ID model/enrollment, security posture or biometric availability. Go 1.26.2 darwin/arm64 is available. Apple SDK path is `/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk`; previous Phase 4F compile-only evidence is historical, not a native call performed in 4G.

Read-only `codesign -dvv` on the actual release desktop executable and bundled engine reports `Signature=adhoc`, no TeamIdentifier, no internal requirements, unbound Info.plist and no sealed resources. Their identifiers are `chronosync_desktop-398c7dae007ed25e` and `a.out`, respectively. These local linker signatures do not establish the mutually pinned developer identity proposed in Phase 4F. No signing identity/keychain inventory, credential access or signing operation was performed. This finding does not prove that no suitable identity exists elsewhere; it describes these artifacts only.

The native host currently uses a child process with inherited stdio and typed renderer commands. No private authenticated XPC route or LA callback exists. Tauri bundling remains disabled and no explicit authentication deployment floor is configured. Current Rust lockfile/cache contains objc2 0.6.5, objc2-foundation 0.3.2 and block2 0.6.2, but no objc2-local-authentication dependency. Availability as transitive dependencies is not authorization to introduce a native authentication integration.

## Implemented isolated boundary

New `experiments/macos-auth-protocol` is a separate Go standard-library-only module and test harness. It imports no ChronoSync production package, has no exported entry point, executable, filesystem/storage adapter, subprocess launcher, renderer route or network transport. It cannot change durable admissions, start Codex, or issue anything consumed by the supervisor. The main engine/Tauri build does not include it. No dependencies were installed and no production source/configuration was changed.

The coordinator generates random challenge, synthetic admission, epoch and native-session IDs. It binds a fixed display-only manifest digest, version, explicit policy label and bounded deadline. Canonicalization has independently reproduced digest vectors. Messages are canonical typed JSON capped at 4096 bytes. Full tuple equality, expiry, cancellation/disconnect and one-time completion are serialized under a mutex. Unknown/reused/mismatched/stale/late/malformed outcomes refuse. A fresh coordinator rotates epoch/session. The package stores nothing beyond memory.

The out-of-band peer-verifier interface is a **test seam only**. Nil or mismatched verifier refuses; the sole successful verifier lives in _test.go. It proves protocol behavior given a verifier result, not that any peer has been authenticated. No `verified` field or arbitrary success Boolean is accepted as independent authority. A successful test observation has no production consumer.

Full encoding, isolation and proposed transport interface: [experiment README](../experiments/macos-auth-protocol/README.md). Production conservative provider, storage/migration 3, supervisor refusal, existing agent behavior and public IPC remain unchanged.

## Explicit gates and concrete native proposal

### Factor policy — pending explicit decision

The user was asked whether this prototype may use macOS device-owner authentication (which may accept password/Apple Watch) or must be biometric-only. No choice is inferred from the earlier general OS-backed-authentication requirement. Until an explicit answer arrives, there is no native call or fallback implementation. The two protocol labels in tests are synthetic branch data, not an approved policy.

### Native binding and transport — approval required before introduction

Phase 4F proposes LAContext in the Rust native host, with a private mutually peer-authenticated channel to an engine-side coordinator. The current native artifacts and bridge cannot demonstrate that guarantee. Phase 4G Sections 1 and 5 explicitly require approval before new dependencies/helpers/signing/entitlement/packaging changes, or require stopping with interface and test plan when that boundary cannot be established.

Concrete proposal for the next isolated slice:

1. A separate, development-only native host target and disposable coordinator fixture, neither linked to the production engine/admission repository. No additional production Tauri command or React endpoint.
2. Use the already cached objc2 0.6.5, objc2-foundation 0.3.2 and block2 0.6.2 as explicit dependencies of the isolated native fixture. Review minimal Objective-C LAContext declarations/FFI for ownership, block lifetime, thread dispatch and cancellation. Alternatively, an additional generated LocalAuthentication binding crate requires its own pinned-version/source review before download; no such crate is selected or installed now.
3. Private NSXPCConnection interface with mutual peer code requirements set before activation. Exact host/coordinator code identities and version/update rules must be fixed in the fixture configuration; neither a PID/UID nor any-signed-binary check is acceptable. Do not route native result messages through the existing production stdio path. Go-side Foundation/XPC interoperability is not implemented; approval must cover the native adapter/fixture approach before introducing that build dependency.
4. Approve separate test-bundle packaging and a signing trust profile for both artifacts. A developer-identity profile needs an explicitly designated identity; none was inspected or assumed available. An exact per-build artifact-pinning experiment would be a different profile requiring review, not an automatic substitute for developer identity or same-user resistance. No ad-hoc artifact currently passes a demonstrated mutual-peer test.
5. Start with macOS 13+ ARM64 deployment for the documented NSXPCConnection API, then test on the observed OS. No privileged helper, new Keychain entitlement, signing key generation, security-setting change or account credential collection. Determine exact native packaging entitlements from the approved fixture design; do not add broad exceptions to make it run.

Security cost: this introduces unsafe native interoperability, a new process/channel lifecycle, packaging/signature verification and independent native UI. A host statement still depends on uncompromised host/OS and is not a cryptographically transferable Apple assertion. Any need for additional helper, dependency or entitlement must return for approval. The protocol model is ready for review without installing these components.

## Native behavior contract and test plan — unimplemented

Native host must fetch/display the fixed synthetic operation from the trusted coordinator, retain one fresh LAContext, use only the explicitly approved policy, set documented unlock-reuse duration to zero and invalidate after every terminal result. UI must say it authenticates for a synthetic display-only experiment and does not prove the user read or cryptographically approved the manifest. No real agent or command follows success.

Map denial, user cancellation, system interruption, unavailable/unenrolled authentication, lockout, invalid context and timeout to bounded outcomes. No automatic weaker policy, retry or second context after an ambiguous result. Cancellation/disconnect invalidates context and pending connection; a late callback is discarded. The receiving coordinator authenticates the channel first, then matches exact challenge/session/manifest/deadline. A local channel alone does not establish identity.

| Layer                         | Required evidence                                                                                                                       | Current status                                                                                 |
| ----------------------------- | --------------------------------------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| Protocol unit model           | Stable digest; expiry, mismatches, replay, concurrency, cancellation/timeout, restart/disconnect, malformed/oversized frames            | Implemented and tested with deterministic synthetic verifier                                   |
| Isolation                     | No production imports/launcher/store/API; no exported entry point; no production source changes                                         | Import/export/module checks plus baseline content comparison                                   |
| Native API compilation        | Actual approved Rust/native target and callbacks compile                                                                                | Not performed in 4G; Phase 4F selector/header check is not this evidence                       |
| Peer identity                 | Expected artifact accepted; unsigned/wrong-identity peer rejected; wrong session and connection replay rejected after actual peer check | Not implemented or tested; signing/packaging/native interoperability approval gate             |
| Interactive OS authentication | Approved factor, fresh prompt, observed success/denial/cancel; no native result leaks to React                                          | Not exercised; policy and fixture approval plus explicit user-present test scheduling required |
| Native lifecycle              | Context invalidation, late callback, timeout, host/coordinator crash/restart                                                            | Model tests only; real callbacks/process lifecycle unverified                                  |
| UI/accessibility              | Honest synthetic summary, focus, keyboard, VoiceOver, cancellation and error presentation                                               | Not implemented/tested                                                                         |
| Same-user/admin compromise    | Defined residual assumptions; adversarial evidence for any claimed mitigation                                                           | No resistance claimed; model success is not security evidence                                  |

Never induce biometric lockout for testing or modify security settings. Use naturally unavailable states or injected test outcomes for deterministic tests. Test doubles must not be present in native production registration paths. Manual tests, once separately authorized, must record OS/hardware class, selected policy, actual prompt and outcome without collecting credentials or biometric material.

## Actual validation

- Read-only OS/architecture, SDK path, Cargo lock/cache and `codesign -dvv` inspections completed as above. No interactive authentication, signature verification exchange or native prompt occurred.
- Independent Python json/hashlib vector calculation produced manifest digest `ab77301ae655c1e6beca9bd002eb198db2d27676c637b9bd3db620e5a2d30f9d` and fixed challenge digest `8d9faf8dca8ff1628be8b7964a691f247615d390c11ea4985b49431e364494b9`.
- Initial isolated `go test -race -count=1 ./...` and `go vet ./...` passed with network fetching disabled and temporary cache. A follow-up assertion explicitly tests malformed decoding independently of tuple mismatch; final results below cover that addition.
- Final isolated module commands with `GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache`: `go -C experiments/macos-auth-protocol test -race -count=1 ./...`, `go -C experiments/macos-auth-protocol vet ./...`, and `go -C experiments/macos-auth-protocol build ./...` all passed. Gofmt reported no unformatted files. Build compiles a library, not an authentication executable.
- Production refusal regression with the same offline/cache environment plus `GOMODCACHE=/tmp/chronosync-go-mod`: `go -C engine test -race ./internal/policy ./internal/supervisor -run 'TestRegistryProductionClosed|TestPolicyRefusedBeforeLaunch' -count=1` passed both packages uncached. No agent launched.
- `npm run check:docs`, `npm run format:check`, `git diff --check` and a local file-link scan of 43 Markdown documents passed.
- Pre/post SHA-256 inventory: every existing runtime source, dependency/configuration/asset and historical phase report unchanged, including prior untracked Phase 4E/F files. Nine existing Markdown documents updated; new isolated module/README and separate Phase 4G report added.
- Full `make check`/`make native-check` were not rerun: production code/configuration is unchanged and the separate experiment is covered by its own race/vet/build checks. No new native integration exists to build or exercise. Prior broad-check evidence remains attributed to its original phase.

## Stop and production readiness

Native work stops at the requested dependency/signing/packaging and factor-policy boundaries. OS-backed authentication remains mandatory; the missing guarantee is not replaced by stdio, filesystem permissions, a renderer Boolean or the synthetic verifier. No production design can be approved based on these model tests alone.

To proceed, explicitly choose the authentication factor policy and approve/revise the isolated binding, native transport/coordinator and signing profile above. Then schedule the user-present native test after fixture identity checks pass. Production integration, migrations, public IPC, provider registration and positive launch require separate authorization and security review. Historical Phase 1–3 GUI gates and Phase 4A–4F reports remain unchanged. No commit or push performed.
