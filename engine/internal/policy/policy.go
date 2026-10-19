// Package policy evaluates supplied evidence, not the security of the host.
// A satisfied decision is not a launch authorization or a sandbox guarantee.
package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"regexp"
	"time"
)

const Version = 1

type Dimension string

const (
	Workspace     Dimension = "workspace"
	ExternalFiles Dimension = "external_files"
	Network       Dimension = "network"
	Processes     Dimension = "processes"
	Environment   Dimension = "environment"
	Credentials   Dimension = "credentials"
	HostServices  Dimension = "host_services"
	History       Dimension = "history"
	CPU           Dimension = "cpu_milliseconds"
	Memory        Dimension = "memory_bytes"
	Disk          Dimension = "disk_bytes"
	ProcessCount  Dimension = "process_count"
	Output        Dimension = "output_bytes"
	Timeout       Dimension = "timeout_milliseconds"
	Cancellation  Dimension = "descendant_cleanup_milliseconds"
	Approval      Dimension = "human_approval"
	Audit         Dimension = "audit"
)

// Dimensions returns a copy. Every dimension must be explicitly specified;
// omission is unresolved, not permission. This is not a default policy.
func Dimensions() []Dimension {
	return []Dimension{Workspace, ExternalFiles, Network, Processes, Environment, Credentials, HostServices, History, CPU, Memory, Disk, ProcessCount, Output, Timeout, Cancellation, Approval, Audit}
}

type Mode string

const (
	Deny      Mode = "deny"
	ReadOnly  Mode = "read_only"
	ReadWrite Mode = "read_write"
	Allowlist Mode = "allowlist"
	Bounded   Mode = "bounded"
	Required  Mode = "required"
)

// Requirement is mandatory. Names are environment variable names only, never
// values. General path/network/process allowlists await separate design approval.
type Requirement struct {
	Dimension Dimension
	Mode      Mode
	Limit     int64
	Names     []string
}

// Scope uses opaque SHA-256 identities bound by the trusted caller. The core
// does not resolve paths, authenticate executables, or collect host details.
type Scope struct {
	WorkspaceDigest  string
	ExecutableDigest string
}
type RequestedPolicy struct {
	Version      int
	ID           string
	Scope        Scope
	Requirements []Requirement
}
type Target struct {
	Platform            string
	Backend             string
	BackendVersion      string
	ConfigurationDigest string
}
type Availability string

const (
	Available   Availability = "available"
	Unavailable Availability = "unavailable"
	Unknown     Availability = "unknown"
	Unsupported Availability = "unsupported"
)

type Level string

const (
	Enforced Level = "enforced"
	Observed Level = "observed"
	Advisory Level = "advisory"
)

type Limitation string

const (
	NoTrustedProvider Limitation = "no_trusted_provider"
	ConfigurationOnly Limitation = "configuration_only"
	PartialCoverage   Limitation = "partial_coverage"
)

// Evidence is provenance, not a certificate. Only a trusted provider may supply
// positive results. A syntactically valid digest does not authenticate evidence.
type Evidence struct {
	Source      string
	Digest      string
	Verified    bool
	CheckedAt   time.Time
	ValidUntil  time.Time
	Limitations []Limitation
}
type EnforcementCapability struct {
	Version      int
	Target       Target
	Scope        Scope
	Restriction  Requirement
	Availability Availability
	Level        Level
	Evidence     Evidence
}
type EffectivePolicy struct {
	Version       int
	RequestDigest string
	Target        Target
	Scope         Scope
	Established   []Requirement
}
type Outcome string

const (
	Satisfied Outcome = "requirements_satisfied"
	Refused   Outcome = "refused"
)

type Code string

const (
	InvalidPolicy           Code = "invalid_policy"
	UnsupportedRequirement  Code = "unsupported_requirement"
	MissingRequirement      Code = "missing_requirement"
	ConflictingRequirement  Code = "conflicting_requirement"
	InvalidTarget           Code = "invalid_target"
	InvalidCapability       Code = "invalid_capability"
	MissingCapability       Code = "missing_capability"
	CapabilityUnavailable   Code = "capability_unavailable"
	CapabilityUnknown       Code = "capability_unknown"
	CapabilityUnsupported   Code = "capability_unsupported"
	UnverifiedEvidence      Code = "unverified_evidence"
	StaleEvidence           Code = "stale_evidence"
	WrongTarget             Code = "wrong_target"
	WrongScope              Code = "wrong_scope"
	InsufficientEnforcement Code = "insufficient_enforcement"
	RestrictionMismatch     Code = "restriction_mismatch"
)

type Reason struct {
	Dimension Dimension
	Code      Code
}

// Decision never contains raw request values or host details in refusal reasons.
// Evidence and Established are detached copies; partial establishment is not consent.
type Decision struct {
	Outcome   Outcome
	Effective EffectivePolicy
	Reasons   []Reason
	Evidence  []EnforcementCapability
}

var identifier = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.-]{0,63}$`)
var envName = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,63}$`)

func digest(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == s
}
func validScope(s Scope) bool { return digest(s.WorkspaceDigest) && digest(s.ExecutableDigest) }
func validTarget(t Target) bool {
	return (t.Platform == "darwin" || t.Platform == "linux" || t.Platform == "windows") && identifier.MatchString(t.Backend) && identifier.MatchString(t.BackendVersion) && digest(t.ConfigurationDigest)
}
func known(d Dimension) bool {
	for _, v := range Dimensions() {
		if d == v {
			return true
		}
	}
	return false
}
func validRequirement(r Requirement) bool {
	if !known(r.Dimension) {
		return false
	}
	mode := Deny
	switch r.Dimension {
	case Workspace:
		if r.Mode != ReadOnly && r.Mode != ReadWrite {
			return false
		}
		mode = r.Mode
	case Environment:
		mode = Allowlist
	case CPU, Memory, Disk, ProcessCount, Output, Timeout, Cancellation:
		mode = Bounded
	case Approval, Audit:
		mode = Required
	}
	if r.Mode != mode {
		return false
	}
	if mode == Bounded {
		if r.Limit <= 0 {
			return false
		}
	} else if r.Limit != 0 {
		return false
	}
	if mode != Allowlist {
		return len(r.Names) == 0
	}
	if len(r.Names) > 64 {
		return false
	}
	for i, n := range r.Names {
		if !envName.MatchString(n) || (i > 0 && n <= r.Names[i-1]) {
			return false
		}
	}
	return true
}
func copyRequirement(r Requirement) Requirement {
	r.Names = append([]string(nil), r.Names...)
	return r
}
func copyCapability(c EnforcementCapability) EnforcementCapability {
	c.Restriction = copyRequirement(c.Restriction)
	c.Evidence.Limitations = append([]Limitation(nil), c.Evidence.Limitations...)
	return c
}
func validCapability(c EnforcementCapability) bool {
	if c.Version != Version || !validTarget(c.Target) || !validScope(c.Scope) || !validRequirement(c.Restriction) || !identifier.MatchString(c.Evidence.Source) {
		return false
	}
	switch c.Availability {
	case Available, Unavailable, Unknown, Unsupported:
	default:
		return false
	}
	switch c.Level {
	case Enforced, Observed, Advisory:
	default:
		return false
	}
	if len(c.Evidence.Limitations) > 3 {
		return false
	}
	for i, l := range c.Evidence.Limitations {
		switch l {
		case NoTrustedProvider, ConfigurationOnly, PartialCoverage:
		default:
			return false
		}
		for _, prior := range c.Evidence.Limitations[:i] {
			if prior == l {
				return false
			}
		}
	}
	if c.Evidence.Verified && (c.Availability != Available || !digest(c.Evidence.Digest) || c.Evidence.CheckedAt.IsZero() || !c.Evidence.ValidUntil.After(c.Evidence.CheckedAt)) {
		return false
	}
	return c.Evidence.Digest == "" || digest(c.Evidence.Digest)
}

// Resolve is deterministic for its explicit inputs, including now. It does not
// read clocks, paths, environment or platform facilities and never launches.
// Exact scope/constraint matching is deliberately conservative: even a purported
// stronger but different constraint needs matching evidence, not inferred safety.
func Resolve(p RequestedPolicy, target Target, capabilities []EnforcementCapability, now time.Time) Decision {
	d := Decision{Outcome: Refused, Effective: EffectivePolicy{Version: Version, Target: target, Scope: p.Scope}}
	add := func(dim Dimension, code Code) { d.Reasons = append(d.Reasons, Reason{dim, code}) }
	if p.Version != Version || !identifier.MatchString(p.ID) || !validScope(p.Scope) || len(p.Requirements) > len(Dimensions()) {
		add("", InvalidPolicy)
		return d
	}
	if !validTarget(target) || now.IsZero() {
		add("", InvalidTarget)
		return d
	}
	requirements := make(map[Dimension]Requirement)
	for _, r := range p.Requirements {
		if !known(r.Dimension) {
			add("", UnsupportedRequirement)
			continue
		}
		if _, ok := requirements[r.Dimension]; ok {
			add(r.Dimension, ConflictingRequirement)
		}
		requirements[r.Dimension] = r
		if !validRequirement(r) {
			add(r.Dimension, InvalidPolicy)
		}
	}
	for _, dim := range Dimensions() {
		if _, ok := requirements[dim]; !ok {
			add(dim, MissingRequirement)
		}
	}
	// Denied process creation is incompatible with a positive child-process count
	// only if count means new children. Here count bounds the entire execution
	// boundary including the initial agent, so count >=1 remains meaningful.
	if len(d.Reasons) > 0 {
		return d
	}
	canonical := p
	canonical.Requirements = nil
	for _, dim := range Dimensions() {
		canonical.Requirements = append(canonical.Requirements, copyRequirement(requirements[dim]))
	}
	raw, _ := json.Marshal(canonical)
	hash := sha256.Sum256(raw)
	d.Effective.RequestDigest = hex.EncodeToString(hash[:])
	if len(capabilities) > len(Dimensions()) {
		add("", InvalidCapability)
		return d
	}
	caps := make(map[Dimension]EnforcementCapability)
	for _, c := range capabilities {
		if !validCapability(c) {
			add("", InvalidCapability)
			continue
		}
		dim := c.Restriction.Dimension
		if _, ok := caps[dim]; ok {
			add(dim, InvalidCapability)
		}
		caps[dim] = copyCapability(c)
	}
	if len(d.Reasons) > 0 {
		return d
	}
	for _, dim := range Dimensions() {
		r := requirements[dim]
		c, ok := caps[dim]
		if !ok {
			add(dim, MissingCapability)
			continue
		}
		d.Evidence = append(d.Evidence, c)
		switch {
		case c.Target != target:
			add(dim, WrongTarget)
		case c.Scope != p.Scope:
			add(dim, WrongScope)
		case c.Availability == Unavailable:
			add(dim, CapabilityUnavailable)
		case c.Availability == Unknown:
			add(dim, CapabilityUnknown)
		case c.Availability == Unsupported:
			add(dim, CapabilityUnsupported)
		case !c.Evidence.Verified:
			add(dim, UnverifiedEvidence)
		case now.Before(c.Evidence.CheckedAt) || !now.Before(c.Evidence.ValidUntil):
			add(dim, StaleEvidence)
		case c.Level != Enforced || len(c.Evidence.Limitations) > 0:
			add(dim, InsufficientEnforcement)
		case !reflect.DeepEqual(copyRequirement(r), c.Restriction):
			add(dim, RestrictionMismatch)
		default:
			d.Effective.Established = append(d.Effective.Established, copyRequirement(r))
		}
	}
	if len(d.Reasons) == 0 {
		d.Outcome = Satisfied
	}
	return d
}
