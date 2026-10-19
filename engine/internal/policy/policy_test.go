package policy

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func fixture() (RequestedPolicy, Target, []EnforcementCapability, time.Time) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	p := RequestedPolicy{Version: Version, ID: "synthetic-request", Scope: Scope{strings.Repeat("a", 64), strings.Repeat("b", 64)}}
	target := Target{"linux", "synthetic-test-only", "1", strings.Repeat("c", 64)}
	var caps []EnforcementCapability
	for _, dim := range Dimensions() {
		r := Requirement{Dimension: dim, Mode: Deny}
		switch dim {
		case Workspace:
			r.Mode = ReadOnly
		case Environment:
			r.Mode = Allowlist
			r.Names = []string{"LANG"}
		case CPU, Memory, Disk, ProcessCount, Output, Timeout, Cancellation:
			r.Mode = Bounded
			r.Limit = 100
		case Approval, Audit:
			r.Mode = Required
		}
		p.Requirements = append(p.Requirements, r)
		caps = append(caps, EnforcementCapability{Version: Version, Target: target, Scope: p.Scope, Restriction: copyRequirement(r), Availability: Available, Level: Enforced, Evidence: Evidence{Source: "synthetic-fixture", Digest: strings.Repeat("d", 64), Verified: true, CheckedAt: now.Add(-time.Minute), ValidUntil: now.Add(time.Minute)}})
	}
	return p, target, caps, now
}
func has(d Decision, code Code) bool {
	for _, r := range d.Reasons {
		if r.Code == code {
			return true
		}
	}
	return false
}
func TestSatisfiedAndDeterministic(t *testing.T) {
	p, target, caps, now := fixture()
	d := Resolve(p, target, caps, now)
	if d.Outcome != Satisfied || len(d.Reasons) != 0 || len(d.Effective.Established) != len(Dimensions()) || !digest(d.Effective.RequestDigest) {
		t.Fatalf("%+v", d)
	}
	if !reflect.DeepEqual(d, Resolve(p, target, caps, now)) {
		t.Fatal("nondeterministic")
	}
	// Caller order is irrelevant for valid policies/evidence.
	for i, j := 0, len(p.Requirements)-1; i < j; i, j = i+1, j-1 {
		p.Requirements[i], p.Requirements[j] = p.Requirements[j], p.Requirements[i]
		caps[i], caps[j] = caps[j], caps[i]
	}
	if !reflect.DeepEqual(d, Resolve(p, target, caps, now)) {
		t.Fatal("order-dependent")
	}
}
func TestRefusals(t *testing.T) {
	cases := []struct {
		name   string
		code   Code
		change func(*RequestedPolicy, *Target, *[]EnforcementCapability, *time.Time)
	}{
		{"unavailable", CapabilityUnavailable, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Availability = Unavailable
			(*c)[0].Evidence.Verified = false
		}},
		{"unknown", CapabilityUnknown, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Availability = Unknown
			(*c)[0].Evidence.Verified = false
		}},
		{"unsupported", CapabilityUnsupported, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Availability = Unsupported
			(*c)[0].Evidence.Verified = false
		}},
		{"unverified", UnverifiedEvidence, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Evidence.Verified = false
		}},
		{"wrong-platform", WrongTarget, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Target.Platform = "darwin"
		}},
		{"wrong-backend", WrongTarget, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Target.Backend = "different"
		}},
		{"changed-configuration", WrongTarget, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Target.ConfigurationDigest = strings.Repeat("e", 64)
		}},
		{"wrong-workspace", WrongScope, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Scope.WorkspaceDigest = strings.Repeat("e", 64)
		}},
		{"wrong-executable", WrongScope, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Scope.ExecutableDigest = strings.Repeat("e", 64)
		}},
		{"expired", StaleEvidence, func(_ *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, n *time.Time) { *n = n.Add(time.Minute) }},
		{"future", StaleEvidence, func(_ *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, n *time.Time) { *n = n.Add(-time.Hour) }},
		{"observed-not-enforced", InsufficientEnforcement, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Level = Observed
		}},
		{"limited-evidence", InsufficientEnforcement, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Evidence.Limitations = []Limitation{PartialCoverage}
		}},
		{"silent-write-widening", RestrictionMismatch, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Restriction.Mode = ReadWrite
		}},
		{"weaken-budget", RestrictionMismatch, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[8].Restriction.Limit++
		}},
		{"weaken-env", RestrictionMismatch, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[4].Restriction.Names = append((*c)[4].Restriction.Names, "SECRET")
		}},
		{"missing-capability", MissingCapability, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) { *c = (*c)[1:] }},
		{"invalid-version", InvalidPolicy, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) { p.Version++ }},
		{"missing-requirement", MissingRequirement, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements = p.Requirements[1:]
		}},
		{"unknown-requirement", UnsupportedRequirement, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements[0].Dimension = "secret-unknown"
		}},
		{"conflicting-duplicate", ConflictingRequirement, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements[1] = p.Requirements[0]
			p.Requirements[1].Mode = ReadWrite
		}},
		{"invalid-limit", InvalidPolicy, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements[8].Limit = -1
		}},
		{"irrelevant-limit", InvalidPolicy, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements[0].Limit = 1
		}},
		{"raw-env-value", InvalidPolicy, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements[4].Names = []string{"KEY=secret"}
		}},
		{"unsupported-allow-all", InvalidPolicy, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) {
			p.Requirements[2].Mode = "allow_all"
		}},
		{"missing-scope", InvalidPolicy, func(p *RequestedPolicy, _ *Target, _ *[]EnforcementCapability, _ *time.Time) { p.Scope = Scope{} }},
		{"invalid-target", InvalidTarget, func(_ *RequestedPolicy, g *Target, _ *[]EnforcementCapability, _ *time.Time) { g.Platform = "made-up" }},
		{"duplicate-capability", InvalidCapability, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) { (*c)[1] = (*c)[0] }},
		{"forged-positive-unknown", InvalidCapability, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Availability = Unknown
		}},
		{"missing-provenance", InvalidCapability, func(_ *RequestedPolicy, _ *Target, c *[]EnforcementCapability, _ *time.Time) {
			(*c)[0].Evidence.Source = ""
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, g, c, n := fixture()
			tc.change(&p, &g, &c, &n)
			d := Resolve(p, g, c, n)
			if d.Outcome != Refused || !has(d, tc.code) {
				t.Fatalf("want %s: %+v", tc.code, d)
			}
			if reflect.DeepEqual(d.Effective.Established, p.Requirements) {
				t.Fatal("refused policy presented as fully established")
			}
		})
	}
}
func TestNoPermissiveDefaultAndConservativeProvider(t *testing.T) {
	if d := Resolve(RequestedPolicy{}, Target{}, nil, time.Time{}); d.Outcome != Refused {
		t.Fatal(d)
	}
	p, g, _, n := fixture()
	for _, platform := range []string{"darwin", "linux", "windows"} {
		g.Platform = platform
		c := (ConservativeProvider{}).Capabilities(p, g)
		d := Resolve(p, g, c, n)
		if d.Outcome != Refused || len(d.Reasons) != len(Dimensions()) || len(d.Effective.Established) != 0 {
			t.Fatal(d)
		}
		for _, v := range c {
			if v.Evidence.Verified || v.Availability != Unknown {
				t.Fatal(v)
			}
		}
	}
}
func TestResultsDoNotAliasInputs(t *testing.T) {
	p, g, c, n := fixture()
	d := Resolve(p, g, c, n)
	before, _ := json.Marshal(d)
	p.Requirements[4].Names[0] = "CHANGED"
	c[4].Restriction.Names[0] = "OTHER"
	after, _ := json.Marshal(d)
	if string(before) != string(after) {
		t.Fatal("mutable evidence")
	}
}
func TestRequestDigestBindsScopeAndRestrictions(t *testing.T) {
	p, g, c, n := fixture()
	a := Resolve(p, g, c, n).Effective.RequestDigest
	p.Requirements[8].Limit++
	b := Resolve(p, g, c, n).Effective.RequestDigest
	p.Scope.ExecutableDigest = strings.Repeat("f", 64)
	d := Resolve(p, g, c, n).Effective.RequestDigest
	if a == b || b == d {
		t.Fatal("digest failed to bind request")
	}
}

func TestInvalidCapabilityMetadata(t *testing.T) {
	cases := map[string]func(*EnforcementCapability){
		"version":      func(c *EnforcementCapability) { c.Version = 99 },
		"availability": func(c *EnforcementCapability) { c.Availability = "claimed" },
		"level":        func(c *EnforcementCapability) { c.Level = "sandboxed" },
		"digest":       func(c *EnforcementCapability) { c.Evidence.Digest = "not-a-digest" },
		"validity":     func(c *EnforcementCapability) { c.Evidence.ValidUntil = c.Evidence.CheckedAt },
		"limitation":   func(c *EnforcementCapability) { c.Evidence.Limitations = []Limitation{"unrecognized"} },
		"duplicate-limitations": func(c *EnforcementCapability) {
			c.Evidence.Limitations = []Limitation{PartialCoverage, PartialCoverage}
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			p, g, c, n := fixture()
			change(&c[0])
			d := Resolve(p, g, c, n)
			if d.Outcome != Refused || !has(d, InvalidCapability) {
				t.Fatal(d)
			}
		})
	}
}
func TestEnvironmentValidationAndNoEcho(t *testing.T) {
	for _, names := range [][]string{{"KEY=private-value"}, {"LANG", "LANG"}, {"Z", "A"}, {"lowercase"}, make([]string, 65)} {
		p, g, c, n := fixture()
		p.Requirements[4].Names = names
		d := Resolve(p, g, c, n)
		if d.Outcome != Refused || !has(d, InvalidPolicy) {
			t.Fatal(d)
		}
		for _, r := range d.Reasons {
			if r.Message() == "" || strings.Contains(r.Message(), "private-value") {
				t.Fatal("unsafe reason")
			}
		}
	}
}
