package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"runtime"
	"time"
)

const ConservativeProviderID = "chronosync-no-provider"
const providerContractVersion = 1

// ProviderIdentity identifies reviewed code, not a certificate or user claim.
type ProviderIdentity struct {
	ContractVersion             int
	ID, Implementation, Version string
}
type ProviderRequest struct {
	Policy RequestedPolicy
	Target Target
}
type ProviderEvidence struct {
	Identity     ProviderIdentity
	Target       Target
	Scope        Scope
	Generation   uint64
	Capabilities []EnforcementCapability
}
type ProviderGeneration struct {
	Generation uint64
	Revoked    bool
}

// RegisteredProvider is an internal port. Implementing it does not register or
// confer trust. Only the closed constructor below installs production code.
// Providers must honor cancellation and must not retain/mutate returned data.
type RegisteredProvider interface {
	Assess(context.Context, ProviderRequest) (ProviderEvidence, error)
	CurrentGeneration(context.Context) (ProviderGeneration, error)
}
type providerRegistration struct {
	identity ProviderIdentity
	target   Target
	provider RegisteredProvider
}

// ProviderRegistry has no registration, deserialization or injected-clock API.
// Its zero value refuses. Test providers exist only in same-package _test files.
type ProviderRegistry struct {
	entries map[string]providerRegistration
	now     func() time.Time
}
type ProviderError struct{ Code string }

func (e *ProviderError) Error() string { return "capability provider refused: " + e.Code }
func providerFailure(code string) (Decision, error) {
	return Decision{Outcome: Refused}, &ProviderError{Code: code}
}
func ConservativeTarget() Target {
	h := sha256.Sum256([]byte("chronosync-supervisor-no-policy-backend-v1"))
	return Target{Platform: runtime.GOOS, Backend: "supervisor-no-policy-backend", BackendVersion: "1", ConfigurationDigest: hex.EncodeToString(h[:])}
}
func NewProviderRegistry() *ProviderRegistry {
	identity := ProviderIdentity{providerContractVersion, ConservativeProviderID, "chronosync-conservative", "1"}
	target := ConservativeTarget()
	return &ProviderRegistry{entries: map[string]providerRegistration{ConservativeProviderID: {identity, target, conservativeRegistered{identity, target}}}, now: time.Now}
}

// Resolve obtains evidence itself; callers supply intent, never evidence or
// verification flags. The result is an assessment, not transferable authority.
// Generation revalidation bounds this call only, not a later launch window.
func (r *ProviderRegistry) Resolve(ctx context.Context, id string, p RequestedPolicy, target Target) (Decision, error) {
	if r == nil || r.now == nil {
		return providerFailure("unregistered_provider")
	}
	entry, ok := r.entries[id]
	if !ok || entry.provider == nil {
		return providerFailure("unregistered_provider")
	}
	if entry.identity.ContractVersion != providerContractVersion || entry.identity.ID != id || !identifier.MatchString(entry.identity.ID) || !identifier.MatchString(entry.identity.Implementation) || !identifier.MatchString(entry.identity.Version) {
		return providerFailure("invalid_registration")
	}
	if target != entry.target {
		return providerFailure("wrong_target")
	}
	// Validate the request through the existing resolver before provider access.
	start := r.now()
	validation := Resolve(p, target, nil, start)
	for _, reason := range validation.Reasons {
		if reason.Code != MissingCapability {
			return providerFailure("invalid_request")
		}
	}
	p.Requirements = append([]Requirement(nil), p.Requirements...)
	for i := range p.Requirements {
		p.Requirements[i] = copyRequirement(p.Requirements[i])
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if ctx.Err() != nil {
		return providerFailure("provider_cancelled")
	}
	before, err := entry.provider.CurrentGeneration(ctx)
	if err != nil {
		return providerFailure("provider_failure")
	}
	if before.Generation == 0 || before.Revoked {
		return providerFailure("provider_revoked")
	}
	// Give the provider a detached request; it cannot change the resolver input.
	request := p
	request.Requirements = append([]Requirement(nil), p.Requirements...)
	for i := range request.Requirements {
		request.Requirements[i] = copyRequirement(request.Requirements[i])
	}
	evidence, err := entry.provider.Assess(ctx, ProviderRequest{request, target})
	if err != nil {
		return providerFailure("provider_failure")
	}
	if evidence.Identity != entry.identity {
		return providerFailure("provider_identity_mismatch")
	}
	if evidence.Target != target {
		return providerFailure("wrong_target")
	}
	if evidence.Scope != p.Scope {
		return providerFailure("wrong_scope")
	}
	if evidence.Generation != before.Generation {
		return providerFailure("provider_revoked")
	}
	if len(evidence.Capabilities) > len(Dimensions()) {
		return providerFailure("invalid_evidence")
	}
	for _, c := range evidence.Capabilities {
		if c.Evidence.Source != id {
			return providerFailure("provider_identity_mismatch")
		}
	}
	after, err := entry.provider.CurrentGeneration(ctx)
	if err != nil {
		return providerFailure("provider_failure")
	}
	if ctx.Err() != nil {
		return providerFailure("provider_cancelled")
	}
	if after != before || after.Revoked {
		return providerFailure("provider_revoked")
	}
	now := r.now()
	if now.Before(start) {
		return providerFailure("clock_regressed")
	}
	return Resolve(p, target, evidence.Capabilities, now), nil
}

type conservativeRegistered struct {
	identity ProviderIdentity
	target   Target
}

func (p conservativeRegistered) CurrentGeneration(context.Context) (ProviderGeneration, error) {
	// No enforcement exists to revoke; this generation attests only unknown state.
	return ProviderGeneration{Generation: 1}, nil
}
func (p conservativeRegistered) Assess(_ context.Context, r ProviderRequest) (ProviderEvidence, error) {
	return ProviderEvidence{p.identity, p.target, r.Policy.Scope, 1, (ConservativeProvider{}).Capabilities(r.Policy, p.target)}, nil
}
