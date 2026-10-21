// Package admission defines pure internal contracts. It performs no persistence,
// authentication, identity discovery, clock reads or process execution.
package admission

import (
	"bytes"
	"chronosync.local/engine/internal/policy"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"
)

const Version uint32 = 1
const Domain = "chronosync-launch-binding\x00"
const MaxArguments = 128
const MaxArgumentBytes = 32768
const MaxCanonicalBytes = 256 * 1024
const MaxPathBytes = 1024
const MaxEnvironment = 64

type Code string

const (
	Invalid            Code = "invalid_contract"
	UnsupportedVersion Code = "unsupported_version"
	IllegalTransition  Code = "illegal_transition"
	BindingMismatch    Code = "binding_mismatch"
	ApprovalMissing    Code = "approval_missing"
	ApprovalExpired    Code = "approval_expired"
	EvidenceInvalid    Code = "evidence_invalid"
	EpochMismatch      Code = "epoch_mismatch"
	Revoked            Code = "revoked"
	Expired            Code = "expired"
	AlreadyClaimed     Code = "already_claimed"
	OrderingInvalid    Code = "ordering_invalid"
)

type Error struct{ Code Code }

func (e *Error) Error() string { return "admission: " + string(e.Code) }
func fail(c Code) error        { return &Error{c} }

// ID and Epoch are opaque caller-allocated identities. Constructing an Epoch
// validates syntax only; it does not generate uniqueness or prove rollback safety.
type ID string
type Epoch string
type Digest string

func hexSize(s string, n int) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == n && hex.EncodeToString(b) == s
}
func validID(id string) bool { return hexSize(id, 16) && id != strings.Repeat("0", 32) }
func NewEpoch(id ID) (Epoch, error) {
	if !validID(string(id)) {
		return "", fail(Invalid)
	}
	return Epoch(id), nil
}

var labelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)
var envPattern = regexp.MustCompile(`^[A-Z_][A-Z0-9_]{0,63}$`)

func validDigest(d Digest) bool { return hexSize(string(d), 32) }
func label(s string) bool       { return labelPattern.MatchString(s) }
func text(s string, max int, empty bool) bool {
	return (empty || s != "") && len(s) <= max && utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}

type IdentityStatus string

const (
	IdentityUnresolved IdentityStatus = "unresolved"
	IdentityReference  IdentityStatus = "external_reference"
)

// Reference never means verified. The backend mechanism remains unresolved.
type ObjectIdentity struct {
	Path            string // Exact non-secret resolved path representation; not resolved here.
	ArtifactDigest  Digest
	ObjectReference Digest
	Status          IdentityStatus
}
type EnvironmentRule string

const (
	PublicConfiguration EnvironmentRule = "public_configuration_reference"
	PrivateReference    EnvironmentRule = "private_reference"
)

// Environment contains no value field. References name externally supplied exact
// value rules; private values/commitment generation and key management are absent.
type Environment struct {
	Name          string
	Rule          EnvironmentRule
	Reference     ID
	VersionDigest Digest
}
type InputKind string

const (
	NoInput               InputKind = "none"
	PrivateInputReference InputKind = "private_reference"
)

type Input struct {
	Kind       InputKind
	Reference  ID
	Commitment Digest
}
type EvidenceReference struct {
	Dimension  policy.Dimension
	ID         ID
	Digest     Digest
	Provider   string
	Generation uint64
	ValidFrom  int64 // Explicit positive Unix milliseconds, never clock access.
	ValidUntil int64
}
type BindingSpec struct {
	Version               uint32
	Admission             ID
	Epoch                 Epoch
	Adapter               string
	AdapterVersion        string
	AdapterConfiguration  Digest
	Executable            ObjectIdentity
	Arguments             []string // Public argv only. Secret-bearing argv is unsupported.
	Environment           []Environment
	Input                 Input
	WorkingDirectory      ObjectIdentity
	Workspace             ObjectIdentity
	WorkspaceGeneration   uint64
	WorkspaceState        Digest
	PolicyRequest         Digest
	EffectiveRestrictions Digest
	Target                policy.Target
	Evidence              []EvidenceReference // Backend evidence only: approval/audit are later.
	ParentRun             ID                  // Empty explicitly means no parent. No invented workflow identity.
}

// Binding is immutable through the exported API; all slices are copied.
type Binding struct {
	spec      BindingSpec
	canonical []byte
	hash      Digest
}

func clone(s BindingSpec) BindingSpec {
	s.Arguments = slices.Clone(s.Arguments)
	s.Environment = slices.Clone(s.Environment)
	s.Evidence = slices.Clone(s.Evidence)
	return s
}
func (b Binding) Spec() BindingSpec { return clone(b.spec) }
func (b Binding) Bytes() []byte     { return slices.Clone(b.canonical) }
func (b Binding) Hash() Digest      { return b.hash }
func (b Binding) valid() bool {
	return b.spec.Version == Version && validDigest(b.hash) && len(b.canonical) > 0
}
func (b Binding) IdentityUnresolved() bool {
	return !b.valid() || b.spec.Executable.Status == IdentityUnresolved || b.spec.Workspace.Status == IdentityUnresolved || b.spec.WorkingDirectory.Status == IdentityUnresolved
}

// lexicalPath validates a bounded spelling only; it never resolves a path or
// proves identity. Windows uses an explicit drive:/ representation; UNC/device
// paths await a reviewed backend profile rather than host-dependent cleaning.
func lexicalPath(value, platform string) bool {
	if !text(value, MaxPathBytes, false) || strings.Contains(value, "\\") {
		return false
	}
	for _, r := range value {
		if r < 32 || r == 127 {
			return false
		}
	}
	rest := value
	if platform == "windows" {
		if len(value) < 3 || value[0] < 'A' || value[0] > 'Z' || value[1:3] != ":/" {
			return false
		}
		rest = value[3:]
	} else {
		if !strings.HasPrefix(value, "/") {
			return false
		}
		rest = value[1:]
	}
	if rest == "" {
		return true
	}
	for _, part := range strings.Split(rest, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
		if platform == "windows" {
			if strings.ContainsAny(part, `<>:"|?*`) || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
				return false
			}
			stem := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
			if stem == "CON" || stem == "PRN" || stem == "AUX" || stem == "NUL" || (len(stem) == 4 && (strings.HasPrefix(stem, "COM") || strings.HasPrefix(stem, "LPT")) && stem[3] >= '1' && stem[3] <= '9') {
				return false
			}
		}
	}
	return true
}
func validObject(o ObjectIdentity) bool {
	if !text(o.Path, MaxPathBytes, false) || !validDigest(o.ArtifactDigest) {
		return false
	}
	switch o.Status {
	case IdentityUnresolved:
		return o.ObjectReference == ""
	case IdentityReference:
		return validDigest(o.ObjectReference)
	default:
		return false
	}
}
func backendDimension(d policy.Dimension) bool {
	for _, v := range policy.Dimensions() {
		if v == d && v != policy.Approval && v != policy.Audit {
			return true
		}
	}
	return false
}
func validEvidence(e EvidenceReference) bool {
	return backendDimension(e.Dimension) && validID(string(e.ID)) && validDigest(e.Digest) && label(e.Provider) && e.Generation > 0 && e.ValidFrom > 0 && e.ValidUntil > e.ValidFrom
}
func NewBinding(input BindingSpec) (Binding, error) {
	if input.Version != Version {
		return Binding{}, fail(UnsupportedVersion)
	}
	s := clone(input)
	for _, o := range []ObjectIdentity{s.Executable, s.Workspace, s.WorkingDirectory} {
		if !lexicalPath(o.Path, s.Target.Platform) {
			return Binding{}, fail(Invalid)
		}
	}
	if !validID(string(s.Admission)) || !validID(string(s.Epoch)) || !label(s.Adapter) || !label(s.AdapterVersion) || !validDigest(s.AdapterConfiguration) || !validObject(s.Executable) || !validObject(s.Workspace) || !validObject(s.WorkingDirectory) || s.WorkspaceGeneration == 0 || !validDigest(s.WorkspaceState) || !validDigest(s.PolicyRequest) || !validDigest(s.EffectiveRestrictions) {
		return Binding{}, fail(Invalid)
	}
	if s.ParentRun != "" && !validID(string(s.ParentRun)) {
		return Binding{}, fail(Invalid)
	}
	if len(s.Arguments) > MaxArguments || len(s.Environment) > MaxEnvironment || len(s.Evidence) != len(policy.Dimensions())-2 {
		return Binding{}, fail(Invalid)
	}
	for _, arg := range s.Arguments {
		if !text(arg, MaxArgumentBytes, true) {
			return Binding{}, fail(Invalid)
		}
	}
	switch s.Input.Kind {
	case NoInput:
		if s.Input.Reference != "" || s.Input.Commitment != "" {
			return Binding{}, fail(Invalid)
		}
	case PrivateInputReference:
		if !validID(string(s.Input.Reference)) || !validDigest(s.Input.Commitment) {
			return Binding{}, fail(Invalid)
		}
	default:
		return Binding{}, fail(Invalid)
	}
	if (s.Target.Platform != "darwin" && s.Target.Platform != "linux" && s.Target.Platform != "windows") || !label(s.Target.Backend) || !label(s.Target.BackendVersion) || !validDigest(Digest(s.Target.ConfigurationDigest)) {
		return Binding{}, fail(Invalid)
	}
	slices.SortFunc(s.Environment, func(a, b Environment) int { return strings.Compare(a.Name, b.Name) })
	for i, e := range s.Environment {
		if !envPattern.MatchString(e.Name) || !validID(string(e.Reference)) || !validDigest(e.VersionDigest) || (e.Rule != PublicConfiguration && e.Rule != PrivateReference) || (i > 0 && e.Name == s.Environment[i-1].Name) {
			return Binding{}, fail(Invalid)
		}
	}
	slices.SortFunc(s.Evidence, func(a, b EvidenceReference) int { return strings.Compare(string(a.Dimension), string(b.Dimension)) })
	for i, e := range s.Evidence {
		if !validEvidence(e) || (i > 0 && e.Dimension == s.Evidence[i-1].Dimension) {
			return Binding{}, fail(Invalid)
		}
		for _, prior := range s.Evidence[:i] {
			if prior.ID == e.ID {
				return Binding{}, fail(Invalid)
			}
		}
	}
	raw := encode(s)
	if len(raw) > MaxCanonicalBytes {
		return Binding{}, fail(Invalid)
	}
	hash := sha256.Sum256(raw)
	return Binding{s, raw, Digest(hex.EncodeToString(hash[:]))}, nil
}

// Encoder: fixed ASCII domain, u32 version; strings are u32 byte length + UTF-8;
// unsigned integers/counts are big endian. No normalization or maps.
func encode(s BindingSpec) []byte {
	var b bytes.Buffer
	b.WriteString(Domain)
	u32 := func(n uint32) { _ = binary.Write(&b, binary.BigEndian, n) }
	u64 := func(n uint64) { _ = binary.Write(&b, binary.BigEndian, n) }
	str := func(v string) { u32(uint32(len(v))); b.WriteString(v) }
	obj := func(o ObjectIdentity) {
		str(o.Path)
		str(string(o.ArtifactDigest))
		str(string(o.ObjectReference))
		str(string(o.Status))
	}
	u32(s.Version)
	str(string(s.Admission))
	str(string(s.Epoch))
	str(s.Adapter)
	str(s.AdapterVersion)
	str(string(s.AdapterConfiguration))
	obj(s.Executable)
	u32(uint32(len(s.Arguments)))
	for _, arg := range s.Arguments {
		str(arg)
	}
	u32(uint32(len(s.Environment)))
	for _, e := range s.Environment {
		str(e.Name)
		str(string(e.Rule))
		str(string(e.Reference))
		str(string(e.VersionDigest))
	}
	str(string(s.Input.Kind))
	str(string(s.Input.Reference))
	str(string(s.Input.Commitment))
	obj(s.WorkingDirectory)
	obj(s.Workspace)
	u64(s.WorkspaceGeneration)
	str(string(s.WorkspaceState))
	str(string(s.PolicyRequest))
	str(string(s.EffectiveRestrictions))
	str(s.Target.Platform)
	str(s.Target.Backend)
	str(s.Target.BackendVersion)
	str(s.Target.ConfigurationDigest)
	u32(uint32(len(s.Evidence)))
	for _, e := range s.Evidence {
		str(string(e.Dimension))
		str(string(e.ID))
		str(string(e.Digest))
		str(e.Provider)
		u64(e.Generation)
		u64(uint64(e.ValidFrom))
		u64(uint64(e.ValidUntil))
	}
	str(string(s.ParentRun))
	return b.Bytes()
}
