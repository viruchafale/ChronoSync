package admission

import (
	"chronosync.local/engine/internal/policy"
	"errors"
	"reflect"
	"testing"
)

func start(t *testing.T) Admission {
	t.Helper()
	a, e := New(mustBinding(t), 100, 1000)
	if e != nil {
		t.Fatal(e)
	}
	return a
}
func command(a Admission, action Action) Command {
	s := a.Snapshot()
	return Command{Action: action, Epoch: s.Epoch, Order: s.LastOrder + 1, Now: s.LastTime + 1, Attempt: id(40), Checks: Checks{Binding: s.Binding, PolicyRequest: a.binding.spec.PolicyRequest, EffectiveRestrictions: a.binding.spec.EffectiveRestrictions, Outcome: policy.Satisfied, EvidenceGenerationDigest: a.binding.EvidenceDigest(), Approval: s.Approval.ID, ApprovalValidation: ExternalValidation, ValidFrom: 100, ValidUntil: 1000}}
}
func step(t *testing.T, a Admission, c Command) Admission {
	t.Helper()
	next, e := Apply(a, c)
	if e != nil {
		t.Fatalf("%s from %s: %v", c.Action, a.Snapshot().State, e)
	}
	return next
}
func approved(t *testing.T) Admission {
	a := start(t)
	c := command(a, IssueChallenge)
	c.Challenge = Challenge{id(30), a.Snapshot().Binding, a.Snapshot().Epoch, c.Now, 900}
	a = step(t, a, c)
	c = command(a, AcceptApproval)
	c.Approval = ApprovalReference{id(31), id(30), a.Snapshot().Binding, a.Snapshot().Epoch, id(32), dg("a"), c.Now, 800}
	return step(t, a, c)
}
func prepared(t *testing.T) Admission { a := approved(t); return step(t, a, command(a, Prepare)) }
func claimed(t *testing.T) Admission  { a := prepared(t); return step(t, a, command(a, Claim)) }
func code(t *testing.T, a Admission, c Command, want Code) {
	t.Helper()
	before := a.Snapshot()
	next, e := Apply(a, c)
	var typed *Error
	if !errors.As(e, &typed) || typed.Code != want {
		t.Fatalf("%s: want %s got %v", c.Action, want, e)
	}
	if !reflect.DeepEqual(next.Snapshot(), before) || !reflect.DeepEqual(a.Snapshot(), before) {
		t.Fatal("failed transition mutated state")
	}
}
func TestValidLifecycleAndTerminalObservations(t *testing.T) {
	a := claimed(t)
	if a.Snapshot().State != Claimed || a.Snapshot().Attempt != id(40) {
		t.Fatal(a.Snapshot())
	}
	code(t, a, command(a, Claim), AlreadyClaimed)
	code(t, a, command(a, BeginAttempt), IllegalTransition)
	a = step(t, a, command(a, FinalValidate))
	a = step(t, a, command(a, BeginAttempt))
	a = step(t, a, command(a, ObserveStarted))
	if a.Snapshot().Outcome != Started {
		t.Fatal(a.Snapshot())
	}
	for _, act := range []Action{Claim, Prepare, BeginAttempt, ObserveStarted, ObserveUnknown, ObserveNotStarted, Refuse, Expire} {
		c := command(a, act)
		c.Now = 1100
		_, e := Apply(a, c)
		if e == nil {
			t.Fatalf("terminal reused via %s", act)
		}
	}
}
func TestApprovalValidation(t *testing.T) {
	a := start(t)
	code(t, a, command(a, Claim), IllegalTransition)
	c := command(a, IssueChallenge)
	c.Challenge = Challenge{id(30), a.Snapshot().Binding, a.Snapshot().Epoch, c.Now, 900}
	a = step(t, a, c)
	tests := []struct {
		name   string
		change func(*Command)
		want   Code
	}{
		{"missing", func(c *Command) { c.Approval = ApprovalReference{} }, ApprovalMissing},
		{"binding", func(c *Command) { c.Approval.Binding = dg("b") }, BindingMismatch},
		{"epoch", func(c *Command) { c.Approval.Epoch = Epoch(id(88)) }, EpochMismatch},
		{"wrong-challenge", func(c *Command) { c.Approval.Challenge = id(88) }, ApprovalMissing},
		{"expired", func(c *Command) { c.Now = 800 }, ApprovalExpired},
		{"response-after-challenge", func(c *Command) { c.Approval.ApprovedAt = 901; c.Approval.ExpiresAt = 950; c.Now = 902 }, ApprovalExpired},
		{"missing-principal", func(c *Command) { c.Approval.PrincipalReference = "" }, ApprovalMissing},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := command(a, AcceptApproval)
			c.Approval = ApprovalReference{id(31), id(30), a.Snapshot().Binding, a.Snapshot().Epoch, id(32), dg("a"), c.Now, 800}
			tc.change(&c)
			code(t, a, c, tc.want)
		})
	}
	a = approved(t)
	c = command(a, Prepare)
	c.Checks.ApprovalValidation = Unchecked
	code(t, a, c, ApprovalMissing)
}
func TestReadinessAtEveryBoundary(t *testing.T) {
	for _, stage := range []Action{Prepare, Claim, FinalValidate, BeginAttempt} {
		t.Run(string(stage), func(t *testing.T) {
			a := approved(t)
			if stage != Prepare {
				a = step(t, a, command(a, Prepare))
			}
			if stage == FinalValidate || stage == BeginAttempt {
				a = step(t, a, command(a, Claim))
			}
			if stage == BeginAttempt {
				a = step(t, a, command(a, FinalValidate))
			}
			tests := []struct {
				name   string
				change func(*Command)
				want   Code
			}{
				{"missing-evidence", func(c *Command) { c.Checks = Checks{} }, BindingMismatch},
				{"stale", func(c *Command) { c.Checks.ValidUntil = c.Now }, EvidenceInvalid},
				{"future", func(c *Command) { c.Checks.ValidFrom = c.Now + 1 }, EvidenceInvalid},
				{"different-evidence", func(c *Command) { c.Checks.EvidenceGenerationDigest = dg("c") }, EvidenceInvalid},
				{"refused-assessment", func(c *Command) { c.Checks.Outcome = policy.Refused }, EvidenceInvalid},
				{"wrong-policy", func(c *Command) { c.Checks.PolicyRequest = dg("c") }, BindingMismatch},
				{"wrong-effective", func(c *Command) { c.Checks.EffectiveRestrictions = dg("c") }, BindingMismatch},
				{"expired-approval", func(c *Command) { c.Now = 800 }, ApprovalExpired},
				{"wrong-approval", func(c *Command) { c.Checks.Approval = id(90) }, ApprovalMissing},
			}
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) { c := command(a, stage); tc.change(&c); code(t, a, c, tc.want) })
			}
		})
	}
}
func TestRevocationOrder(t *testing.T) {
	a := prepared(t)
	r := command(a, Revoke)
	r.Revocation = RevocationReference{id(80), a.Snapshot().Binding, 1}
	revoked := step(t, a, r)
	code(t, revoked, command(revoked, Claim), IllegalTransition)
	// A claim ordered first consumes permanently; later revocation blocks attempt.
	a = step(t, a, command(a, Claim))
	r = command(a, Revoke)
	r.Revocation = RevocationReference{id(80), a.Snapshot().Binding, 1}
	a = step(t, a, r)
	code(t, a, command(a, FinalValidate), Revoked)
	code(t, a, command(a, Claim), AlreadyClaimed)
	a = step(t, a, command(a, ObserveNotStarted))
	if a.Snapshot().State != Claimed {
		t.Fatal("recycled after revoke")
	}
	b := claimed(t)
	b = step(t, b, command(b, FinalValidate))
	b = step(t, b, command(b, BeginAttempt))
	b = step(t, b, command(b, ObserveStarted))
	r = command(b, Revoke)
	r.Revocation = RevocationReference{id(81), b.Snapshot().Binding, 1}
	b = step(t, b, r)
	if !b.Snapshot().CancellationRequired || b.Snapshot().Outcome != Started {
		t.Fatal("revocation rewrote launch history")
	}
	code(t, b, command(b, BeginAttempt), IllegalTransition)
}
func TestFailedUnknownAndRecoveryNeverReusable(t *testing.T) {
	for _, act := range []Action{ObserveNotStarted, ObserveUnknown, Recover} {
		t.Run(string(act), func(t *testing.T) {
			a := claimed(t)
			c := command(a, act)
			if act == Recover {
				c.Epoch = Epoch(id(99))
			}
			a = step(t, a, c)
			if a.Snapshot().State != Claimed {
				t.Fatal("claim released")
			}
			if act == Recover && a.Snapshot().Outcome != OutcomeUnknown {
				t.Fatal("recovery invented no-start")
			}
			for _, action := range []Action{Claim, Prepare, FinalValidate, BeginAttempt, ObserveStarted, ObserveNotStarted, ObserveUnknown} {
				if _, e := Apply(a, command(a, action)); e == nil {
					t.Fatalf("reused via %s", action)
				}
			}
		})
	}
	for _, a := range []Admission{start(t), approved(t), prepared(t)} {
		c := command(a, Recover)
		c.Epoch = Epoch(id(99))
		b := step(t, a, c)
		if b.Snapshot().State != Invalidated {
			t.Fatal("old epoch reusable")
		}
		code(t, b, command(b, Claim), IllegalTransition)
	}
}
func TestVersionsEpochsOrderingAndRefusal(t *testing.T) {
	a := prepared(t)
	c := command(a, Claim)
	c.Epoch = Epoch(id(90))
	code(t, a, c, EpochMismatch)
	c = command(a, Claim)
	c.Order = a.Snapshot().LastOrder
	code(t, a, c, OrderingInvalid)
	c = command(a, Claim)
	c.Now = a.Snapshot().LastTime - 1
	code(t, a, c, OrderingInvalid)
	broken := a
	broken.snapshot.Version = 99
	code(t, broken, command(broken, Claim), UnsupportedVersion)
	broken = a
	broken.snapshot.State = "future"
	code(t, broken, command(broken, Claim), Invalid)
	c = command(a, Recover)
	code(t, a, c, EpochMismatch)
	a = step(t, a, command(a, Refuse))
	code(t, a, command(a, Refuse), IllegalTransition)
	b := prepared(t)
	c = command(b, Expire)
	c.Now = 800
	b = step(t, b, c)
	code(t, b, command(b, Claim), IllegalTransition)
}
func TestUnresolvedIdentityNeverReady(t *testing.T) {
	a := approved(t)
	s := a.binding.Spec()
	s.Executable.Status = IdentityUnresolved
	s.Executable.ObjectReference = ""
	b, e := NewBinding(s)
	if e != nil {
		t.Fatal(e)
	}
	// Construct a consistent modeled approval for the new, explicitly unresolved binding.
	a.binding = b
	a.snapshot.Binding = b.Hash()
	a.snapshot.Challenge.Binding = b.Hash()
	a.snapshot.Approval.Binding = b.Hash()
	code(t, a, command(a, Prepare), EvidenceInvalid)
}
func TestOneUseInvariantAcrossActionSequences(t *testing.T) {
	actions := []Action{Prepare, Claim, FinalValidate, BeginAttempt, ObserveNotStarted, ObserveStarted, ObserveUnknown, Refuse, Expire, Revoke, Recover}
	var walk func(Admission, int)
	walk = func(a Admission, depth int) {
		if depth == 0 {
			return
		}
		for _, act := range actions {
			c := command(a, act)
			if act == Recover {
				c.Epoch = Epoch(id(90))
			}
			if act == Revoke {
				c.Revocation = RevocationReference{id(80), a.Snapshot().Binding, 1}
			}
			if act == Expire {
				c.Now = 1001
			}
			b, e := Apply(a, c)
			if e != nil {
				continue
			}
			if b.Snapshot().State != Claimed || b.Snapshot().Attempt != a.Snapshot().Attempt {
				t.Fatal("consumed admission became reusable")
			}
			walk(b, depth-1)
		}
	}
	walk(claimed(t), 4)
}
func TestValueIsolation(t *testing.T) {
	a := prepared(t)
	before := a.Snapshot()
	b := step(t, a, command(a, Claim))
	view := b.Snapshot()
	view.State = Preparing
	if a.Snapshot() != before || b.Snapshot().State != Claimed {
		t.Fatal("state alias")
	}
}

func TestExpiredBackendReferenceAndDefiniteAttemptFailure(t *testing.T) {
	a := approved(t)
	spec := a.binding.Spec()
	spec.Evidence[0].ValidUntil = a.Snapshot().LastTime + 1
	b, err := NewBinding(spec)
	if err != nil {
		t.Fatal(err)
	}
	a.binding = b
	a.snapshot.Binding = b.Hash()
	a.snapshot.Challenge.Binding = b.Hash()
	a.snapshot.Approval.Binding = b.Hash()
	code(t, a, command(a, Prepare), EvidenceInvalid)
	a = claimed(t)
	a = step(t, a, command(a, FinalValidate))
	a = step(t, a, command(a, BeginAttempt))
	a = step(t, a, command(a, ObserveNotStarted))
	if a.Snapshot().State != Claimed || a.Snapshot().Outcome != NotStarted {
		t.Fatal("failed attempt recycled")
	}
	r := command(a, Revoke)
	r.Revocation = RevocationReference{id(88), a.Snapshot().Binding, 1}
	a = step(t, a, r)
	if a.Snapshot().CancellationRequired {
		t.Fatal("conclusively absent child presented as requiring cancellation")
	}
	code(t, a, command(a, Claim), AlreadyClaimed)
}
