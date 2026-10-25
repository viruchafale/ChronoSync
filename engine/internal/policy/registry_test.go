package policy

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// This implementation and registration exist only in the test binary.
type testRegisteredProvider struct {
	evidence    ProviderEvidence
	err         error
	generations []ProviderGeneration
	calls       int
	mutate      func(ProviderRequest)
}

func (p *testRegisteredProvider) Assess(_ context.Context, r ProviderRequest) (ProviderEvidence, error) {
	if p.mutate != nil {
		p.mutate(r)
	}
	return p.evidence, p.err
}
func (p *testRegisteredProvider) CurrentGeneration(context.Context) (ProviderGeneration, error) {
	i := p.calls
	p.calls++
	if i >= len(p.generations) {
		i = len(p.generations) - 1
	}
	return p.generations[i], p.err
}
func registryFixture() (*ProviderRegistry, *testRegisteredProvider, RequestedPolicy, Target) {
	p, target, caps, now := fixture()
	id := ProviderIdentity{1, "synthetic-fixture", "test-only", "1"}
	provider := &testRegisteredProvider{evidence: ProviderEvidence{id, target, p.Scope, 1, caps}, generations: []ProviderGeneration{{Generation: 1}}}
	r := &ProviderRegistry{entries: map[string]providerRegistration{id.ID: {id, target, provider}}, now: func() time.Time { return now }}
	return r, provider, p, target
}
func TestRegistryProductionClosed(t *testing.T) {
	p, _, _, _ := fixture()
	for _, r := range []*ProviderRegistry{nil, {}, NewProviderRegistry()} {
		d, e := r.Resolve(context.Background(), "synthetic-fixture", p, ConservativeTarget())
		if e == nil || d.Outcome != Refused {
			t.Fatal("unregistered provider accepted")
		}
	}
	r := NewProviderRegistry()
	if len(r.entries) != 1 {
		t.Fatal("unexpected production registration")
	}
	d, e := r.Resolve(context.Background(), ConservativeProviderID, p, ConservativeTarget())
	if e != nil || d.Outcome != Refused || !has(d, CapabilityUnknown) {
		t.Fatal(d, e)
	}
	for _, c := range d.Evidence {
		if c.Evidence.Verified || c.Level == Enforced {
			t.Fatal("production enforcement claim")
		}
	}
}
func TestRegistryProviderBoundary(t *testing.T) {
	tests := []struct {
		name, code string
		change     func(*ProviderRegistry, *testRegisteredProvider, *RequestedPolicy, *Target)
	}{
		{"identity", "provider_identity_mismatch", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Identity.ID = "forged"
		}},
		{"implementation", "provider_identity_mismatch", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Identity.Implementation = "forged"
		}},
		{"version", "provider_identity_mismatch", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Identity.Version = "2"
		}},
		{"contract", "provider_identity_mismatch", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Identity.ContractVersion = 2
		}},
		{"platform", "wrong_target", func(_ *ProviderRegistry, _ *testRegisteredProvider, _ *RequestedPolicy, t *Target) {
			t.Platform = "darwin"
		}},
		{"backend", "wrong_target", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Target.Backend = "other"
		}},
		{"scope", "wrong_scope", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Scope.WorkspaceDigest = strings.Repeat("e", 64)
		}},
		{"source", "provider_identity_mismatch", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Capabilities[0].Evidence.Source = "forged"
		}},
		{"error", "provider_failure", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.err = errors.New("synthetic-private-canary")
		}},
		{"revoked", "provider_revoked", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.generations[0].Revoked = true
		}},
		{"generation", "provider_revoked", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Generation = 2
		}},
		{"concurrent-revocation", "provider_revoked", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.generations = append(f.generations, ProviderGeneration{Generation: 2, Revoked: true})
		}},
		{"oversize-evidence", "invalid_evidence", func(_ *ProviderRegistry, f *testRegisteredProvider, _ *RequestedPolicy, _ *Target) {
			f.evidence.Capabilities = append(f.evidence.Capabilities, f.evidence.Capabilities[0])
		}},
		{"oversize-request", "invalid_request", func(_ *ProviderRegistry, _ *testRegisteredProvider, p *RequestedPolicy, _ *Target) {
			p.Requirements = append(p.Requirements, p.Requirements[0])
		}},
		{"malformed-request", "invalid_request", func(_ *ProviderRegistry, _ *testRegisteredProvider, p *RequestedPolicy, _ *Target) { p.Version = 99 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, f, p, target := registryFixture()
			tt.change(r, f, &p, &target)
			d, e := r.Resolve(context.Background(), "synthetic-fixture", p, target)
			var pe *ProviderError
			if !errors.As(e, &pe) || pe.Code != tt.code || d.Outcome != Refused || strings.Contains(e.Error(), "canary") || len(d.Evidence) != 0 {
				t.Fatal(d, e)
			}
		})
	}
}
func TestRegistryResolverEvidenceAndClock(t *testing.T) {
	for _, tt := range []struct {
		name   string
		code   Code
		change func(*EnforcementCapability)
	}{
		{"unverified", UnverifiedEvidence, func(c *EnforcementCapability) { c.Evidence.Verified = false }},
		{"expired", StaleEvidence, func(c *EnforcementCapability) { c.Evidence.ValidUntil = c.Evidence.CheckedAt.Add(time.Second) }},
		{"observed", InsufficientEnforcement, func(c *EnforcementCapability) { c.Level = Observed }},
		{"configuration", InsufficientEnforcement, func(c *EnforcementCapability) { c.Evidence.Limitations = []Limitation{ConfigurationOnly} }},
		{"unknown", CapabilityUnknown, func(c *EnforcementCapability) { c.Availability = Unknown; c.Evidence.Verified = false }},
		{"unavailable", CapabilityUnavailable, func(c *EnforcementCapability) { c.Availability = Unavailable; c.Evidence.Verified = false }},
		{"wrong-capability-scope", WrongScope, func(c *EnforcementCapability) { c.Scope.ExecutableDigest = strings.Repeat("f", 64) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r, f, p, target := registryFixture()
			tt.change(&f.evidence.Capabilities[0])
			d, e := r.Resolve(context.Background(), "synthetic-fixture", p, target)
			if e != nil || d.Outcome != Refused || !has(d, tt.code) {
				t.Fatal(d, e)
			}
		})
	}
	r, _, p, target := registryFixture()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := r.Resolve(ctx, "synthetic-fixture", p, target); e == nil {
		t.Fatal("cancel accepted")
	}
	n := 0
	now := r.now()
	r.now = func() time.Time { n++; return now.Add(-time.Duration(n) * time.Second) }
	if _, e := r.Resolve(context.Background(), "synthetic-fixture", p, target); e == nil {
		t.Fatal("regressed clock accepted")
	}
}
func TestRegistryDetachedInputsAndResults(t *testing.T) {
	r, f, p, target := registryFixture()
	f.mutate = func(q ProviderRequest) { q.Policy.Requirements[0].Mode = ReadWrite }
	d, e := r.Resolve(context.Background(), "synthetic-fixture", p, target)
	if e != nil || d.Outcome != Satisfied || p.Requirements[0].Mode != ReadOnly {
		t.Fatal(d, e)
	}
	f.evidence.Capabilities[0].Evidence.Source = "mutated"
	if d.Evidence[0].Evidence.Source != "synthetic-fixture" {
		t.Fatal("aliased result")
	}
	// Satisfied is test-only logic evidence. Production factory cannot install f.
}
