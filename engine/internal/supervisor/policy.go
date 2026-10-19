package supervisor

import (
	"chronosync.local/engine/internal/policy"
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"time"
)

// PolicyError exposes a structured assessment without putting request contents
// in error messages. No caller-supplied evidence can enable a launch.
type PolicyError struct{ Decision policy.Decision }

func (*PolicyError) Error() string {
	return "execution refused: policy enforcement backend and durable admission are unavailable"
}

func refusePolicy(p policy.RequestedPolicy) *PolicyError {
	hash := sha256.Sum256([]byte("chronosync-supervisor-no-policy-backend-v1"))
	target := policy.Target{Platform: runtime.GOOS, Backend: "supervisor-no-policy-backend", BackendVersion: "1", ConfigurationDigest: hex.EncodeToString(hash[:])}
	caps := (policy.ConservativeProvider{}).Capabilities(p, target)
	// This boundary intentionally has no positive admission path. Scope binding,
	// trusted enforcement, approval and durable audit must be reviewed before one
	// is introduced; the existing cwd/consent/Codex labels cannot supply them.
	return &PolicyError{Decision: policy.Resolve(p, target, caps, time.Now())}
}
