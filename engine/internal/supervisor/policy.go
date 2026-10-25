package supervisor

import (
	"chronosync.local/engine/internal/policy"
	"context"
)

// PolicyError exposes a structured assessment without putting request contents
// in error messages. No caller-supplied evidence can enable a launch.
type PolicyError struct{ Decision policy.Decision }

func (*PolicyError) Error() string {
	return "execution refused: policy enforcement backend and durable admission are unavailable"
}

func refusePolicy(p policy.RequestedPolicy) *PolicyError {
	decision, err := policy.NewProviderRegistry().Resolve(context.Background(), policy.ConservativeProviderID, p, policy.ConservativeTarget())
	if err != nil {
		// Registry refusal never falls back to raw caller-supplied evidence.
		decision = policy.Decision{Outcome: policy.Refused}
	}
	// This boundary intentionally has no positive admission path. Scope binding,
	// trusted enforcement, approval and durable audit must be reviewed before one
	// is introduced; the existing cwd/consent/Codex labels cannot supply them.
	return &PolicyError{Decision: decision}
}
