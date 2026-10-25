// Package protocol is a disposable protocol model, not an authenticator.
// It has no exported entry point, native API, transport, persistence or launcher.
package protocol

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sync"
	"time"
)

const maxMessage = 4096
const syntheticManifest = `{"version":1,"operation":"display-only","workspace":"synthetic","arguments":["no-execution"],"private_input":"none"}`
const accountPolicy = "macos-device-owner"
const biometricPolicy = "macos-biometric-only"

type challenge struct {
	Version       int    `json:"version"`
	ID            string `json:"challenge_id"`
	Admission     string `json:"synthetic_admission_id"`
	Manifest      string `json:"manifest_digest"`
	Epoch         string `json:"epoch"`
	NativeSession string `json:"native_session"`
	Issued        int64  `json:"issued_ms"`
	Expires       int64  `json:"expires_ms"`
	Policy        string `json:"policy"`
}
type completion struct {
	Challenge challenge `json:"challenge"`
	Outcome   string    `json:"outcome"`
}

// peerHandle is out-of-band channel context, NEVER a field decoded from a message.
// No real transport issues these handles. Test-only verifiers model this seam.
type peerHandle [16]byte
type peerVerifier interface{ verify(peerHandle) error }
type observation struct{ outcome string }
type coordinator struct {
	mu                             sync.Mutex
	now                            func() time.Time
	verifier                       peerVerifier
	epoch, nativeSession           string
	pending                        challenge
	deadline, last                 time.Time
	issued, terminal, disconnected bool
}

func fail(code string) error { return errors.New("prototype: " + code) }
func randomID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fail("randomness_unavailable")
	}
	return hex.EncodeToString(b[:]), nil
}
func manifestDigest() string {
	h := sha256.Sum256([]byte("chronosync-auth-experiment-manifest-v1\x00" + syntheticManifest))
	return hex.EncodeToString(h[:])
}
func validHex(s string, n int) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == n && hex.EncodeToString(b) == s && !bytes.Equal(b, make([]byte, n))
}
func canonical(c challenge) ([]byte, error) {
	if c.Version != 1 || !validHex(c.ID, 16) || !validHex(c.Admission, 16) || !validHex(c.Epoch, 16) || !validHex(c.NativeSession, 16) || c.Manifest != manifestDigest() || c.Issued <= 0 || c.Expires <= c.Issued || c.Expires-c.Issued > 60000 || (c.Policy != accountPolicy && c.Policy != biometricPolicy) {
		return nil, fail("invalid_challenge")
	}
	b, e := json.Marshal(c)
	if e != nil {
		return nil, fail("invalid_challenge")
	}
	return b, nil
}
func challengeDigest(c challenge) (string, error) {
	b, e := canonical(c)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(append([]byte("chronosync-auth-experiment-challenge-v1\x00"), b...))
	return hex.EncodeToString(h[:]), nil
}

// Only tests supply a verifier. Nil always refuses. This is not registration.
func newCoordinator(now func() time.Time, v peerVerifier) (*coordinator, error) {
	epoch, e := randomID()
	if e != nil {
		return nil, e
	}
	session, e := randomID()
	if e != nil {
		return nil, e
	}
	if now == nil {
		return nil, fail("clock_required")
	}
	return &coordinator{now: now, verifier: v, epoch: epoch, nativeSession: session}, nil
}
func (c *coordinator) begin(policy string, ttl time.Duration) (challenge, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.issued || c.disconnected {
		return challenge{}, fail("session_closed")
	}
	if ttl < time.Millisecond || ttl > time.Minute {
		return challenge{}, fail("invalid_deadline")
	}
	now := c.now()
	id, e := randomID()
	if e != nil {
		return challenge{}, e
	}
	admission, e := randomID()
	if e != nil {
		return challenge{}, e
	}
	pending := challenge{1, id, admission, manifestDigest(), c.epoch, c.nativeSession, now.UnixMilli(), now.Add(ttl).UnixMilli(), policy}
	if _, e = canonical(pending); e != nil {
		return challenge{}, e
	}
	c.pending = pending
	c.deadline = now.Add(ttl)
	c.last = now
	c.issued = true
	return pending, nil
}
func decode(raw []byte) (completion, error) {
	var m completion
	if len(raw) == 0 || len(raw) > maxMessage {
		return m, fail("invalid_message")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil {
		return completion{}, fail("invalid_message")
	}
	// Exact typed bytes also reject duplicate keys, missing fields, aliases,
	// trailing bodies and ambiguous encodings. This draft protocol is canonical-only.
	encoded, e := json.Marshal(m)
	if e != nil || !bytes.Equal(raw, encoded) {
		return completion{}, fail("invalid_message")
	}
	if _, e = canonical(m.Challenge); e != nil {
		return completion{}, e
	}
	switch m.Outcome {
	case "success", "denied", "user_cancel", "system_cancel", "unavailable", "lockout", "invalid_context", "timeout":
	default:
		return completion{}, fail("invalid_outcome")
	}
	return m, nil
}
func (c *coordinator) complete(peer peerHandle, raw []byte) (observation, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.issued || c.terminal || c.disconnected {
		return observation{}, fail("not_pending")
	}
	now := c.now()
	if now.Before(c.last) || !now.Before(c.deadline) {
		c.terminal = true
		return observation{}, fail("expired_or_clock_regressed")
	}
	c.last = now
	if c.verifier == nil || c.verifier.verify(peer) != nil {
		return observation{}, fail("unverified_peer")
	}
	m, e := decode(raw)
	if e != nil {
		return observation{}, e
	}
	if m.Challenge != c.pending {
		return observation{}, fail("challenge_mismatch")
	}
	c.terminal = true
	// Observation is test/model output only, never a production approval.
	return observation{m.Outcome}, nil
}
func (c *coordinator) cancel() { c.mu.Lock(); defer c.mu.Unlock(); c.terminal = true }
func (c *coordinator) disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disconnected = true
	c.terminal = true
}
