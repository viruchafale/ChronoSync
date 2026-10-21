package admission

import (
	"bytes"
	"chronosync.local/engine/internal/policy"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func id(n int) ID        { return ID(fmt.Sprintf("%032x", n)) }
func dg(c string) Digest { return Digest(strings.Repeat(c, 64)) }
func bindingFixture() BindingSpec {
	s := BindingSpec{Version: Version, Admission: id(1), Epoch: Epoch(id(2)), Adapter: "codex", AdapterVersion: "synthetic-1", AdapterConfiguration: dg("a"), Executable: ObjectIdentity{"/fixture/agent", dg("b"), dg("c"), IdentityReference}, Arguments: []string{"exec", "", "--json"}, Environment: []Environment{{"LANG", PublicConfiguration, id(3), dg("d")}, {"TOKEN", PrivateReference, id(4), dg("e")}}, Input: Input{PrivateInputReference, id(5), dg("f")}, WorkingDirectory: ObjectIdentity{"/fixture/work", dg("a"), dg("b"), IdentityReference}, Workspace: ObjectIdentity{"/fixture/work", dg("a"), dg("b"), IdentityReference}, WorkspaceGeneration: 1, WorkspaceState: dg("c"), PolicyRequest: dg("d"), EffectiveRestrictions: dg("e"), Target: policy.Target{Platform: "linux", Backend: "synthetic", BackendVersion: "1", ConfigurationDigest: string(dg("f"))}}
	for i, d := range policy.Dimensions() {
		if d == policy.Approval || d == policy.Audit {
			continue
		}
		s.Evidence = append(s.Evidence, EvidenceReference{d, id(i + 10), dg("a"), "synthetic-provider", 1, 100, 10000})
	}
	return s
}
func mustBinding(t *testing.T) Binding {
	t.Helper()
	b, e := NewBinding(bindingFixture())
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func TestFixedCanonicalVector(t *testing.T) {
	b := mustBinding(t)
	vector, e := os.ReadFile("testdata/binding-v1.hex")
	if e != nil {
		t.Fatal(e)
	}
	expected, e := hex.DecodeString(strings.TrimSpace(string(vector)))
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(expected, b.Bytes()) {
		t.Fatal("canonical bytes differ from independent fixed vector")
	}
	hash, e := os.ReadFile("testdata/binding-v1.sha256")
	if e != nil {
		t.Fatal(e)
	}
	if string(b.Hash()) != strings.TrimSpace(string(hash)) {
		t.Fatal("binding hash changed")
	}
	if bytes.Equal([]byte(b.Hash()), []byte(b.EvidenceDigest())) {
		t.Fatal("domain separation")
	}
	other := append([]byte(nil), expected...)
	other[0] = 'X'
	sum := sha256.Sum256(other)
	if hex.EncodeToString(sum[:]) == string(b.Hash()) {
		t.Fatal("domain omitted")
	}
	other = append([]byte(nil), expected...)
	other[len(Domain)+3] = 2
	sum = sha256.Sum256(other)
	if hex.EncodeToString(sum[:]) == string(b.Hash()) {
		t.Fatal("version omitted")
	}
	s := bindingFixture()
	s.Version = 2
	if _, e := NewBinding(s); e == nil {
		t.Fatal("unsupported version accepted")
	}
}
func TestBindingOrderAndCopies(t *testing.T) {
	s := bindingFixture()
	b, e := NewBinding(s)
	if e != nil {
		t.Fatal(e)
	}
	slices.Reverse(s.Evidence)
	slices.Reverse(s.Environment)
	same, e := NewBinding(s)
	if e != nil || !bytes.Equal(b.Bytes(), same.Bytes()) {
		t.Fatal("unordered collections changed identity", e)
	}
	slices.Reverse(s.Arguments)
	different, e := NewBinding(s)
	if e != nil || different.Hash() == b.Hash() {
		t.Fatal("argument order lost")
	}
	before := b.Bytes()
	s.Arguments[0] = "mutated"
	view := b.Spec()
	view.Arguments[0] = "mutated"
	view.Environment[0].Name = "MUTATED"
	view.Evidence[0].Provider = "mutated"
	raw := b.Bytes()
	raw[0] = 0
	if !bytes.Equal(before, b.Bytes()) || b.Spec().Arguments[0] != "exec" {
		t.Fatal("alias escaped")
	}
	empty := bindingFixture()
	empty.Arguments = nil
	left, _ := NewBinding(empty)
	empty.Arguments = []string{}
	right, _ := NewBinding(empty)
	if left.Hash() != right.Hash() {
		t.Fatal("nil/empty representation drift")
	}
}
func TestSecurityRelevantChangesChangeHash(t *testing.T) {
	base := mustBinding(t)
	changes := map[string]func(*BindingSpec){
		"admission": func(s *BindingSpec) { s.Admission = id(70) }, "epoch": func(s *BindingSpec) { s.Epoch = Epoch(id(71)) },
		"adapter": func(s *BindingSpec) { s.Adapter = "other" }, "adapter-version": func(s *BindingSpec) { s.AdapterVersion = "2" }, "adapter-config": func(s *BindingSpec) { s.AdapterConfiguration = dg("b") },
		"executable-path": func(s *BindingSpec) { s.Executable.Path += "2" }, "executable-digest": func(s *BindingSpec) { s.Executable.ArtifactDigest = dg("c") }, "object-reference": func(s *BindingSpec) { s.Executable.ObjectReference = dg("d") },
		"arguments": func(s *BindingSpec) { s.Arguments = []string{"ex", "ec", "--json"} }, "env-rule": func(s *BindingSpec) { s.Environment[0].Rule = PrivateReference }, "env-value-version": func(s *BindingSpec) { s.Environment[0].VersionDigest = dg("a") },
		"stdin": func(s *BindingSpec) { s.Input.Commitment = dg("a") }, "cwd": func(s *BindingSpec) { s.WorkingDirectory.Path += "2" }, "workspace": func(s *BindingSpec) { s.Workspace.Path += "2" }, "workspace-generation": func(s *BindingSpec) { s.WorkspaceGeneration++ }, "workspace-state": func(s *BindingSpec) { s.WorkspaceState = dg("d") },
		"policy": func(s *BindingSpec) { s.PolicyRequest = dg("a") }, "effective": func(s *BindingSpec) { s.EffectiveRestrictions = dg("a") }, "backend": func(s *BindingSpec) { s.Target.Backend = "other" }, "backend-version": func(s *BindingSpec) { s.Target.BackendVersion = "2" }, "backend-config": func(s *BindingSpec) { s.Target.ConfigurationDigest = string(dg("a")) }, "platform": func(s *BindingSpec) { s.Target.Platform = "darwin" },
		"evidence-generation": func(s *BindingSpec) { s.Evidence[0].Generation++ }, "evidence-expiry": func(s *BindingSpec) { s.Evidence[0].ValidUntil++ }, "evidence-provider": func(s *BindingSpec) { s.Evidence[0].Provider = "other" }, "evidence-digest": func(s *BindingSpec) { s.Evidence[0].Digest = dg("b") }, "parent": func(s *BindingSpec) { s.ParentRun = id(72) },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			s := bindingFixture()
			change(&s)
			b, e := NewBinding(s)
			if e != nil || b.Hash() == base.Hash() {
				t.Fatal("change unbound", e)
			}
		})
	}
}
func TestMalformedAndBounds(t *testing.T) {
	changes := map[string]func(*BindingSpec){
		"unknown-version": func(s *BindingSpec) { s.Version = 0 }, "zero-id": func(s *BindingSpec) { s.Admission = id(0) }, "missing-epoch": func(s *BindingSpec) { s.Epoch = "" }, "digest-case": func(s *BindingSpec) { s.PolicyRequest = dg("A") },
		"arg-nul": func(s *BindingSpec) { s.Arguments = []string{"a\x00b"} }, "arg-invalid-utf8": func(s *BindingSpec) { s.Arguments = []string{string([]byte{255})} }, "arg-length": func(s *BindingSpec) { s.Arguments = []string{strings.Repeat("x", MaxArgumentBytes+1)} }, "arg-count": func(s *BindingSpec) { s.Arguments = make([]string, MaxArguments+1) }, "total-size": func(s *BindingSpec) {
			s.Arguments = make([]string, 9)
			for i := range s.Arguments {
				s.Arguments[i] = strings.Repeat("x", MaxArgumentBytes)
			}
		},
		"path-bound": func(s *BindingSpec) { s.Executable.Path = strings.Repeat("x", MaxPathBytes+1) }, "unknown-identity": func(s *BindingSpec) { s.Workspace.Status = "verified" }, "generation": func(s *BindingSpec) { s.WorkspaceGeneration = 0 },
		"duplicate-env": func(s *BindingSpec) { s.Environment = append(s.Environment, s.Environment[0]) }, "env-value": func(s *BindingSpec) { s.Environment[0].Name = "SECRET=value" }, "env-rule": func(s *BindingSpec) { s.Environment[0].Rule = "literal-secret" }, "env-count": func(s *BindingSpec) { s.Environment = make([]Environment, 65) },
		"input-default": func(s *BindingSpec) { s.Input = Input{} }, "none-with-commitment": func(s *BindingSpec) { s.Input.Kind = NoInput }, "input-missing-reference": func(s *BindingSpec) { s.Input.Reference = "" },
		"evidence-missing": func(s *BindingSpec) { s.Evidence = s.Evidence[1:] }, "evidence-duplicate": func(s *BindingSpec) { s.Evidence[1] = s.Evidence[0] }, "evidence-reused-id": func(s *BindingSpec) { s.Evidence[1].ID = s.Evidence[0].ID }, "unknown-dimension": func(s *BindingSpec) { s.Evidence[0].Dimension = "future" }, "invalid-interval": func(s *BindingSpec) { s.Evidence[0].ValidUntil = s.Evidence[0].ValidFrom }, "unknown-platform": func(s *BindingSpec) { s.Target.Platform = "other" },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			s := bindingFixture()
			change(&s)
			if _, e := NewBinding(s); e == nil {
				t.Fatal("accepted malformed binding")
			}
		})
	}
	s := bindingFixture()
	s.Arguments = []string{strings.Repeat("x", MaxArgumentBytes)}
	if _, e := NewBinding(s); e != nil {
		t.Fatal("exact argument bound", e)
	}
}
func TestNoPrivateValueFieldOrImplicitIdentity(t *testing.T) {
	s := bindingFixture()
	s.Executable.Status = IdentityUnresolved
	s.Executable.ObjectReference = ""
	b, e := NewBinding(s)
	if e != nil || !b.IdentityUnresolved() {
		t.Fatal(e)
	}
	for _, typ := range []reflect.Type{reflect.TypeOf(Environment{}), reflect.TypeOf(Input{})} {
		for i := 0; i < typ.NumField(); i++ {
			if typ.Field(i).Name == "Value" || typ.Field(i).Name == "Secret" {
				t.Fatal("private value slot")
			}
		}
	}
	if bytes.Contains(b.Bytes(), []byte("synthetic-secret-value")) {
		t.Fatal("secret leaked")
	}
	if _, e := NewEpoch(id(0)); e == nil {
		t.Fatal("zero epoch")
	}
	if _, e := NewEpoch(id(55)); e != nil {
		t.Fatal(e)
	}
}

func TestPathSpellingAndUnicodeAreNotNormalized(t *testing.T) {
	for _, path := range []string{"relative", "/a/../b", "/a/./b", "/a//b", "/a/", "/a\\b", "/a\nb"} {
		s := bindingFixture()
		s.Executable.Path = path
		if _, e := NewBinding(s); e == nil {
			t.Fatalf("ambiguous path accepted %q", path)
		}
	}
	for _, path := range []string{"C:/fixture/agent", "C:/"} {
		s := bindingFixture()
		s.Target.Platform = "windows"
		s.Executable.Path = path
		s.Workspace.Path = "C:/work"
		s.WorkingDirectory.Path = "C:/work"
		if _, e := NewBinding(s); e != nil {
			t.Fatal(path, e)
		}
	}
	s := bindingFixture()
	s.Arguments = []string{"é"}
	a, e := NewBinding(s)
	if e != nil {
		t.Fatal(e)
	}
	s.Arguments = []string{"e\u0301"}
	b, e := NewBinding(s)
	if e != nil || a.Hash() == b.Hash() {
		t.Fatal("implicit Unicode normalization")
	}
}
