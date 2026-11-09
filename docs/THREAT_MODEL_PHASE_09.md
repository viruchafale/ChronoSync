# Phase 9 OS-backed key custody threat model

## Assets and boundaries

Assets are Ed25519 seeds, public identity continuity, signing intent/payloads, Keychain item references, migration/trust histories, revocation decisions and existing signed packages. Boundaries are renderer → trusted native commands, native host → private engine stdio, engine signing session → native broker, native broker → macOS Security framework, and legacy filesystem seed → one-time migration handoff.

The renderer is untrusted for seeds, paths, confirmations and arbitrary signing payloads. Go is trusted for canonical Phase 8 payload construction and verification. Rust is trusted for native confirmation, Keychain queries and Ed25519 operations. macOS Keychain protects stored bytes subject to its access control, application identity and account/device state. A stable signed app identity is not available in the current unsigned/ad-hoc development artifact.

## Platform findings and guarantees

- Apple CryptoKit supports `Curve25519.Signing.PrivateKey` (Ed25519), but Apple documents that it has no direct Keychain `SecKey` corollary and must be stored as a generic-password item.
- Secure Enclave signing supports other algorithms such as P-256, not the Phase 8 Ed25519 identity contract. Phase 9 does not change algorithms and makes no Secure Enclave claim.
- The implemented item uses the local macOS login Keychain. A real create/retrieve/sign/delete test passes. It is OS-backed storage, but the current application does not enforce per-sign OS user presence.
- The stronger data-protection Keychain item with device-only accessibility and user-presence access control returned `errSecMissingEntitlement` (`-34018`) under the unsigned Cargo test. It requires a stable signed/entitled distribution and remains design-only.
- Ed25519 signing necessarily materializes the seed in trusted native memory. This is not an opaque/non-extractable hardware key.
- Keychain availability does not guarantee backup/device migration, recovery, human identity, benchmark safety or protection from malware controlling the same application/user session.

## Threats and mitigations

| Threat                                         | Mitigation                                                                                                                                            | Residual risk                                                                                                            |
| ---------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Renderer requests arbitrary signature          | No generic renderer sign command; engine issues bounded, one-use Phase 8 payload sessions; native export orchestration only                           | Compromised trusted host/engine can sign                                                                                 |
| Seed disclosure through IPC/log/report/package | Normal Keychain signing returns signature/public status only; legacy seed handoff is migration-only, bounded and never logged; zeroize native buffers | Process memory/core dumps and compromised trusted code                                                                   |
| Keychain theft/copy                            | Store in the local login Keychain, never renderer/filesystem/package; keep stronger entitlement-backed mode blocked rather than silently claiming it  | Same-user/keychain/application compromise; seed is an extractable generic-password value; login Keychain backup behavior |
| Secure Enclave overclaim                       | Preserve Ed25519 and explicitly reject hardware-backed label                                                                                          | None beyond platform evolution                                                                                           |
| Unsigned/ad-hoc application identity changes   | Document development prompt/ACL instability; stable distribution requires stable signing identity                                                     | User may see repeated prompts or lose access after app identity changes                                                  |
| Key loss/deletion/access denial                | Typed missing/denied/recovery-required states; never reconstruct from public data; rotation is a new key                                              | Signing ability can be permanently lost                                                                                  |
| Migration interruption                         | Durable append-only state before/after Keychain operations; retain legacy until equality and proof signing; idempotent resume                         | Keychain/filesystem cannot commit atomically; orphan item possible                                                       |
| Public/private mismatch or corrupt seed        | Validate canonical legacy record, derive public/key ID and proof signature before retirement                                                          | Coherently compromised local files remain possible                                                                       |
| Concurrent migration/signing                   | Per-service gate, identity/session one-use state and broker duplicate refusal                                                                         | Multi-process Keychain races outside one running host                                                                    |
| Trust-store truncation/rewrite                 | Strict fields/timestamps/reasons/transitions, append-only audit and explicit corruption result                                                        | Unkeyed local integrity cannot resist coherent same-user rewriting/rollback                                              |
| Revoked identity re-import/rotation            | Revoked key ID remains terminal; import never restores trust; rotation never inherits trust                                                           | Other installations have independent local trust                                                                         |
| Recovery abuse                                 | Native confirmation and closed recovery actions; no seed export or automatic trust restore                                                            | User can still make an unsafe explicit choice                                                                            |

## Recovery meanings

- Key recovery means regaining access to the same private key. Without the original legacy seed or existing Keychain item, it is impossible.
- Identity recovery means creating/rotating to a new cryptographic identity and linking `supersedes`; old signatures remain verifiable.
- Trust recovery means inspecting local history and making a new allowed explicit decision. Corrupt or revoked history is never silently reset.
- Revocation permanently refuses a key ID on this installation; it does not delete, globally notify or cryptographically invalidate historical signatures.

## Non-goals and unverified behavior

No hardware custody, per-sign OS authentication in the current build, global revocation, trusted time, organization PKI, remote recovery/sync, OS-level egress isolation, provider/command execution or production authorization is added. Automated tests cannot prove behavior under every macOS account migration, password reset, Keychain repair, application re-signing, system lock or real power loss. Native confirmation-dialog interaction is reported separately from the successful direct Keychain integration test.
