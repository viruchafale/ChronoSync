package policy

// CapabilityProvider is separate from resolution. It is an internal trusted
// dependency, never an IPC/UI supplied claim of enforcement.
type CapabilityProvider interface {
	Capabilities(RequestedPolicy, Target) []EnforcementCapability
}

// ConservativeProvider makes no host probes and cannot emit verified evidence.
// Unknown describes missing knowledge, not an unsupported OS API claim.
type ConservativeProvider struct{}

func (ConservativeProvider) Capabilities(p RequestedPolicy, t Target) []EnforcementCapability {
	if len(p.Requirements) > len(Dimensions()) {
		return nil
	}
	out := make([]EnforcementCapability, 0, len(p.Requirements))
	for _, r := range p.Requirements {
		out = append(out, EnforcementCapability{Version: Version, Target: t, Scope: p.Scope, Restriction: copyRequirement(r), Availability: Unknown, Level: Advisory, Evidence: Evidence{Source: "chronosync-no-provider", Limitations: []Limitation{NoTrustedProvider}}})
	}
	return out
}
