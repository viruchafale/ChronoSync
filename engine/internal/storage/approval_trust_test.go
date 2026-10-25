package storage

import (
	"context"
	"strings"
	"sync"
	"testing"

	"chronosync.local/engine/internal/admission"
)

// References below are synthetic and deliberately unauthenticated. These tests
// prove storage sequencing, never OS-backed approval or production readiness.
func TestApprovalReferenceBindingExpiryAndConsumption(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*AdmissionChange)
	}{
		{"wrong-manifest", func(r *AdmissionChange) { r.Command.Approval.Binding = adigest("f") }},
		{"wrong-challenge", func(r *AdmissionChange) { r.Command.Approval.Challenge = aid() }},
		{"wrong-epoch", func(r *AdmissionChange) { r.Command.Approval.Epoch = admission.Epoch(aid()) }},
		{"expired", func(r *AdmissionChange) { r.Command.Now = 901 }},
		{"invalid-principal", func(r *AdmissionChange) { r.Command.Approval.PrincipalReference = "synthetic-private-canary" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, _ := admissionStore(t)
			r := advanceAdmission(t, s, createAdmission(t, s), admission.IssueChallenge)
			request := admissionRequest(t, s, r, admission.AcceptApproval)
			tc.change(&request)
			if _, e := s.ChangeAdmission(context.Background(), request); e == nil || strings.Contains(e.Error(), "canary") {
				t.Fatal(e)
			}
			after, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
			if e != nil || after.Revision != r.Revision || after.Snapshot != r.Snapshot {
				t.Fatal(after, e)
			}
			var count int
			if e = s.db.QueryRow(`SELECT count(*) FROM admission_operations WHERE CAST(request AS TEXT) LIKE '%synthetic-private-canary%'`).Scan(&count); e != nil || count != 0 {
				t.Fatal("private rejected value persisted", e)
			}
		})
	}
	s, _ := admissionStore(t)
	r := advanceAdmission(t, s, createAdmission(t, s), admission.IssueChallenge)
	request := admissionRequest(t, s, r, admission.AcceptApproval)
	approved, e := s.ChangeAdmission(context.Background(), request)
	if e != nil {
		t.Fatal(e)
	}
	retry, e := s.ChangeAdmission(context.Background(), request)
	if e != nil || !retry.Replayed || retry.Snapshot != approved.Snapshot {
		t.Fatal(retry, e)
	}
	conflict := request
	conflict.Command.Approval.PrincipalReference = aid()
	if _, e = s.ChangeAdmission(context.Background(), conflict); storageCode(e) != "CONFLICT" {
		t.Fatal(e)
	}
	if _, e = s.ChangeAdmission(context.Background(), admissionRequest(t, s, approved, admission.AcceptApproval)); storageCode(e) != "ADMISSION_TRANSITION" {
		t.Fatal("reused challenge", e)
	}
	revoked := advanceAdmission(t, s, approved, admission.Revoke)
	if _, e = s.ChangeAdmission(context.Background(), admissionRequest(t, s, revoked, admission.Prepare)); storageCode(e) != "ADMISSION_TRANSITION" {
		t.Fatal("revoked approval prepared", e)
	}
}
func TestApprovalConcurrentResponsesAndRestart(t *testing.T) {
	s, path := admissionStore(t)
	r := advanceAdmission(t, s, createAdmission(t, s), admission.IssueChallenge)
	requests := []AdmissionChange{admissionRequest(t, s, r, admission.AcceptApproval), admissionRequest(t, s, r, admission.AcceptApproval)}
	results := make(chan error, 2)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for _, req := range requests {
		wg.Add(1)
		go func(req AdmissionChange) {
			defer wg.Done()
			<-start
			_, e := s.ChangeAdmission(context.Background(), req)
			results <- e
		}(req)
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for e := range results {
		if e == nil {
			success++
		} else if c := storageCode(e); c != "STORE_BUSY" && c != "CONFLICT" {
			t.Fatal(e)
		}
	}
	if success != 1 {
		t.Fatal("multiple approvals", success)
	}
	stored, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil || stored.Revision != 2 || len(stored.Audit) != 3 {
		t.Fatal(stored, e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	recovered, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil || recovered.Snapshot.State != admission.Invalidated || recovered.Snapshot.Approval != stored.Snapshot.Approval {
		t.Fatal(recovered, e)
	}
}
func TestApprovalChallengeGlobalUniqueness(t *testing.T) {
	s, _ := admissionStore(t)
	first := advanceAdmission(t, s, createAdmission(t, s), admission.IssueChallenge)
	second := createAdmission(t, s)
	request := admissionRequest(t, s, second, admission.IssueChallenge)
	request.Command.Challenge.ID = first.Snapshot.Challenge.ID
	if _, e := s.ChangeAdmission(context.Background(), request); e == nil {
		t.Fatal("challenge owned twice")
	}
	stored, e := s.GetAdmission(context.Background(), second.Snapshot.Admission)
	if e != nil || stored.Revision != 0 {
		t.Fatal("partial challenge transition", stored, e)
	}
}
