# Isolated macOS authentication protocol model

Development-only Phase 4G experiment. This package is a deterministic protocol model, **not a native authenticator or authenticated transport**. All identifiers/manifests are synthetic. No actual admission, agent, supervisor, database, key or credential is accessed.

Run independently of production:

```sh
GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache go -C experiments/macos-auth-protocol test -race -count=1 ./...
GOPROXY=off GOSUMDB=off GOCACHE=/tmp/chronosync-go-cache go -C experiments/macos-auth-protocol vet ./...
```

The separate module has only standard-library dependencies, no exported entry points and no executable main. Its test-only verifier exists only in protocol_test.go. Nil verifier refuses; no real transport creates or verifies peer handles. An import/export isolation test and before/after production-file hashes guard the intended boundary. Successful model output is just a local observation, never an approval token. The main repository make targets do not run this separate module automatically.

## Canonical model

Fixed synthetic manifest bytes are in protocol.go: version 1, display-only operation, synthetic workspace, no-execution argument and no private input. Manifest digest is SHA-256 of `chronosync-auth-experiment-manifest-v1` + NUL + those exact UTF-8 JSON bytes. It is not a production execution-manifest format.

Challenge v1 is ordered compact typed JSON: version, challenge_id, synthetic_admission_id, manifest_digest, epoch, native_session, issued_ms, expires_ms, policy. IDs are random nonzero 128-bit lowercase hex; the digest is fixed to the synthetic manifest. TTL is 1 ms through 60 s, with a live time.Time deadline (monotonic when the clock supplies it) plus serialized millisecond bounds. Challenge digest is SHA-256 of `chronosync-auth-experiment-challenge-v1` + NUL + canonical bytes. Tests independently reproduce vectors using Python json/hashlib; hashes prove byte identity only.

Completion is canonical compact JSON with the full challenge and one outcome enum. Limit 4096 bytes. Unknown/missing/duplicate/aliased fields, whitespace variants, malformed types, trailing bodies and unknown outcomes refuse. Future native serializers must match this experimental profile or negotiate a reviewed replacement. It is not production IPC.

Each coordinator issues one challenge. Wrong peer/tuple refuses; terminal outcomes consume once; cancellation, deadline, clock regression, disconnect and restart prevent a late success. Restart creates new epoch/native-session identities. No persisted retry or recovery exists; disposable state is lost. Concurrent completions serialize under a mutex. A nil verifier cannot accept a result.

Both account and biometric policy labels occur only as deterministic test data. On 2026-10-24 the user approved macOS device-owner authentication for the future native prototype, including supported password/Apple Watch factors. These tests do **not** implement that native policy, permit a non-OS fallback, or model a real successful OS prompt. Native binding/transport/packaging approvals remain outstanding.

## Native transport interface proposal — not implemented

The receiver must obtain an opaque connection context from a future reviewed transport. Peer identity is out-of-band; JSON cannot supply a verified flag, UID, PID, signing label or trusted handle. Before decoding a completion, the transport verifies the exact expected peer and connection generation. Protocol validation then checks session/challenge separately. Native disconnect revokes that connection and cancels its pending context.

The proposed Rust host callback owner would retain a fresh LAContext for one pending challenge, enforce the explicitly selected policy with zero reuse interval, map OS failures to static outcome codes, invalidate after completion/cancellation, and reject late callbacks. Its result travels only over a mutually verified native XPC connection. A separate disposable coordinator validates the peer/tuple; production engine code is not the receiver.

This cannot be implemented safely by reusing production stdio or adding a renderer completion callback. Current binaries are ad-hoc signed and the repository has no native XPC setup. Proposed next step requires approval of native bindings and separate test-bundle packaging/signature requirements; see the [Phase 4G report](../../docs/IMPLEMENTATION_STATUS_PHASE_04G.md). No native calls or XPC artifacts are present here.
