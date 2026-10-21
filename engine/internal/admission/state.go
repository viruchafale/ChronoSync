package admission

import "chronosync.local/engine/internal/policy"

type State string

const (
	Preparing        State = "preparing"
	AwaitingApproval State = "awaiting_approval"
	Approved         State = "approved"
	Prepared         State = "prepared"
	Claimed          State = "claimed"
	Refused          State = "refused"
	ExpiredState     State = "expired"
	Invalidated      State = "invalidated"
)

type LaunchOutcome string

const (
	Pending        LaunchOutcome = "pending"
	NotStarted     LaunchOutcome = "not_started"
	Started        LaunchOutcome = "started"
	OutcomeUnknown LaunchOutcome = "outcome_unknown"
)

// Challenge and ApprovalReference model references, not authenticators. There is
// no UI assertion, key material or strong-human-authentication claim here.
type Challenge struct {
	ID                  ID
	Binding             Digest
	Epoch               Epoch
	IssuedAt, ExpiresAt int64
}
type ApprovalReference struct {
	ID                    ID
	Challenge             ID
	Binding               Digest
	Epoch                 Epoch
	PrincipalReference    ID
	EvidenceDigest        Digest
	ApprovedAt, ExpiresAt int64
}
type Validation string

const (
	Unchecked          Validation = "unchecked"
	ExternalValidation Validation = "external_validation"
)

// Checks must eventually originate from a trusted coordinator. A caller can
// fabricate these values: this package is not an authentication boundary. Tests
// inject them solely to exercise state logic; no production provider exists.
type Checks struct {
	Binding                  Digest
	PolicyRequest            Digest
	EffectiveRestrictions    Digest
	Outcome                  policy.Outcome
	EvidenceGenerationDigest Digest
	Approval                 ID
	ApprovalValidation       Validation
	ValidFrom, ValidUntil    int64
}

// EvidenceDigest commits to the canonical backend evidence references, using a
// separate domain and the full binding hash. It is not provider authentication.
func (b Binding) EvidenceDigest() Digest {
	return referenceHash("chronosync-admission-evidence\x00", b.Hash())
}

type RevocationReference struct {
	ID         ID
	Binding    Digest
	Generation uint64
}
type Snapshot struct {
	Version                        uint32
	Admission                      ID
	Binding                        Digest
	Epoch                          Epoch
	State                          State
	Outcome                        LaunchOutcome
	CreatedAt, ExpiresAt, LastTime int64
	LastOrder                      uint64
	Challenge                      Challenge
	Approval                       ApprovalReference
	Attempt                        ID
	FinalValidated                 bool
	Attempted                      bool
	Revocation                     RevocationReference
	CancellationRequired           bool
}
type Admission struct {
	binding  Binding
	snapshot Snapshot
}

func (a Admission) Snapshot() Snapshot { return a.snapshot }
func New(b Binding, created, expires int64) (Admission, error) {
	if !b.valid() || created <= 0 || expires <= created {
		return Admission{}, fail(Invalid)
	}
	return Admission{b, Snapshot{Version: Version, Admission: b.spec.Admission, Binding: b.Hash(), Epoch: b.spec.Epoch, State: Preparing, Outcome: Pending, CreatedAt: created, ExpiresAt: expires, LastTime: created}}, nil
}

type Action string

const (
	IssueChallenge    Action = "issue_challenge"
	AcceptApproval    Action = "accept_approval_reference"
	Prepare           Action = "prepare"
	Claim             Action = "claim"
	FinalValidate     Action = "final_validate"
	BeginAttempt      Action = "begin_attempt"
	ObserveNotStarted Action = "observe_not_started"
	ObserveStarted    Action = "observe_started"
	ObserveUnknown    Action = "observe_unknown"
	Refuse            Action = "refuse"
	Expire            Action = "expire"
	Revoke            Action = "revoke"
	Recover           Action = "recover"
)

type Command struct {
	Action     Action
	Epoch      Epoch
	Order      uint64 // Coordinator-supplied local order, not a durable audit sequence.
	Now        int64  // Explicit time, not proof of when anything occurred.
	Challenge  Challenge
	Approval   ApprovalReference
	Checks     Checks
	Attempt    ID
	Revocation RevocationReference
}

func knownState(s State) bool {
	switch s {
	case Preparing, AwaitingApproval, Approved, Prepared, Claimed, Refused, ExpiredState, Invalidated:
		return true
	}
	return false
}
func (a Admission) validate() error {
	s := a.snapshot
	if s.Version != Version {
		return fail(UnsupportedVersion)
	}
	if !a.binding.valid() || !knownState(s.State) || s.Binding != a.binding.Hash() || s.Admission != a.binding.spec.Admission || s.Epoch != a.binding.spec.Epoch {
		return fail(Invalid)
	}
	if s.CreatedAt <= 0 || s.ExpiresAt <= s.CreatedAt || s.LastTime < s.CreatedAt {
		return fail(Invalid)
	}
	switch s.Outcome {
	case Pending, NotStarted, Started, OutcomeUnknown:
	default:
		return fail(Invalid)
	}
	if s.State == Claimed {
		if !validID(string(s.Attempt)) {
			return fail(Invalid)
		}
	} else if s.Attempt != "" || s.Attempted || s.FinalValidated || s.Outcome != Pending {
		return fail(Invalid)
	}
	if s.Outcome == Started && !s.Attempted {
		return fail(Invalid)
	}
	return nil
}
func (a Admission) approval(now int64) error {
	s := a.snapshot
	p := s.Approval
	c := s.Challenge
	if !validID(string(p.ID)) || !validID(string(p.PrincipalReference)) || !validDigest(p.EvidenceDigest) || p.Challenge != c.ID {
		return fail(ApprovalMissing)
	}
	if p.Binding != s.Binding || c.Binding != s.Binding {
		return fail(BindingMismatch)
	}
	if p.Epoch != s.Epoch || c.Epoch != s.Epoch {
		return fail(EpochMismatch)
	}
	if now < p.ApprovedAt || p.ApprovedAt < c.IssuedAt || p.ApprovedAt >= c.ExpiresAt || p.ExpiresAt <= p.ApprovedAt || p.ExpiresAt > c.ExpiresAt || now >= p.ExpiresAt {
		return fail(ApprovalExpired)
	}
	return nil
}
func (a Admission) ready(c Command) error {
	s := a.snapshot
	x := c.Checks
	if s.Revocation.ID != "" {
		return fail(Revoked)
	}
	if c.Now >= s.ExpiresAt {
		return fail(Expired)
	}
	if e := a.approval(c.Now); e != nil {
		return e
	}
	if a.binding.IdentityUnresolved() {
		return fail(EvidenceInvalid)
	}
	if x.Binding != s.Binding || x.PolicyRequest != a.binding.spec.PolicyRequest || x.EffectiveRestrictions != a.binding.spec.EffectiveRestrictions {
		return fail(BindingMismatch)
	}
	if x.Approval != s.Approval.ID || x.ApprovalValidation != ExternalValidation {
		return fail(ApprovalMissing)
	}
	if x.Outcome != policy.Satisfied || x.EvidenceGenerationDigest != a.binding.EvidenceDigest() || x.ValidFrom <= 0 || x.ValidUntil <= x.ValidFrom || c.Now < x.ValidFrom || c.Now >= x.ValidUntil {
		return fail(EvidenceInvalid)
	}
	for _, e := range a.binding.spec.Evidence {
		if c.Now < e.ValidFrom || c.Now >= e.ValidUntil {
			return fail(EvidenceInvalid)
		}
	}
	return nil
}

// Apply returns a new value, never mutates its input, and has no side effects.
// Prepared/Claimed name modeled states, NOT committed records or launch rights.
// Future storage must atomically compare revision/order, consume approval and
// enforce unique attempt identity; calling Apply concurrently cannot do that.
func Apply(a Admission, c Command) (Admission, error) {
	if err := a.validate(); err != nil {
		return a, err
	}
	s := a.snapshot
	if c.Order <= s.LastOrder || c.Now < s.LastTime || c.Now <= 0 {
		return a, fail(OrderingInvalid)
	}
	if c.Action != Recover && c.Epoch != s.Epoch {
		return a, fail(EpochMismatch)
	}
	if s.State == Refused || s.State == ExpiredState || s.State == Invalidated {
		return a, fail(IllegalTransition)
	}
	illegal := func() (Admission, error) { return a, fail(IllegalTransition) }
	switch c.Action {
	case IssueChallenge:
		if s.State != Preparing {
			return illegal()
		}
		q := c.Challenge
		if !validID(string(q.ID)) || q.Binding != s.Binding || q.Epoch != s.Epoch || q.IssuedAt != c.Now || q.ExpiresAt <= q.IssuedAt || q.ExpiresAt > s.ExpiresAt {
			return a, fail(Invalid)
		}
		s.Challenge = q
		s.State = AwaitingApproval
	case AcceptApproval:
		if s.State != AwaitingApproval {
			return illegal()
		}
		s.Approval = c.Approval
		next := a
		next.snapshot = s
		if err := next.approval(c.Now); err != nil {
			return a, err
		}
		// Reference receipt is not trusted authentication or readiness.
		s.State = Approved
	case Prepare, Claim, FinalValidate, BeginAttempt:
		expected := map[Action]State{Prepare: Approved, Claim: Prepared, FinalValidate: Claimed, BeginAttempt: Claimed}[c.Action]
		if s.State != expected {
			if c.Action == Claim && s.State == Claimed {
				return a, fail(AlreadyClaimed)
			}
			return illegal()
		}
		if s.State == Claimed && (s.Outcome != Pending || s.Attempted) {
			return illegal()
		}
		if err := a.ready(c); err != nil {
			return a, err
		}
		switch c.Action {
		case Prepare:
			s.State = Prepared
		case Claim:
			if !validID(string(c.Attempt)) {
				return a, fail(Invalid)
			}
			s.Attempt = c.Attempt
			s.State = Claimed
		case FinalValidate:
			if s.FinalValidated {
				return illegal()
			}
			s.FinalValidated = true
		case BeginAttempt:
			if !s.FinalValidated {
				return illegal()
			}
			s.Attempted = true
		}
	case ObserveNotStarted, ObserveStarted, ObserveUnknown:
		if s.State != Claimed || s.Outcome != Pending {
			return illegal()
		}
		switch c.Action {
		case ObserveNotStarted:
			s.Outcome = NotStarted
		case ObserveStarted:
			if !s.Attempted {
				return illegal()
			}
			s.Outcome = Started
		case ObserveUnknown:
			s.Outcome = OutcomeUnknown
		}
	case Refuse:
		if s.State == Claimed {
			return illegal()
		}
		s.State = Refused
	case Expire:
		if s.State == Claimed {
			return illegal()
		}
		expired := c.Now >= s.ExpiresAt || (s.Challenge.ID != "" && c.Now >= s.Challenge.ExpiresAt) || (s.Approval.ID != "" && c.Now >= s.Approval.ExpiresAt)
		if !expired {
			return illegal()
		}
		s.State = ExpiredState
	case Revoke:
		r := c.Revocation
		if !validID(string(r.ID)) || r.Binding != s.Binding || r.Generation == 0 {
			return a, fail(Invalid)
		}
		if s.Revocation.ID != "" {
			return illegal()
		}
		s.Revocation = r
		if s.State == Claimed {
			s.CancellationRequired = (s.Attempted && s.Outcome != NotStarted) || s.Outcome == OutcomeUnknown
		} else {
			s.State = Invalidated
		}
	case Recover:
		if !validID(string(c.Epoch)) || c.Epoch == s.Epoch {
			return a, fail(EpochMismatch)
		}
		if s.State == Claimed {
			if s.Outcome != Pending {
				return illegal()
			}
			s.Outcome = OutcomeUnknown
			s.CancellationRequired = true
		} else {
			s.State = Invalidated
		}
	default:
		return illegal()
	}
	s.LastOrder = c.Order
	s.LastTime = c.Now
	a.snapshot = s
	return a, nil
}
