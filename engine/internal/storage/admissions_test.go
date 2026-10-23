package storage

import (
	"chronosync.local/engine/internal/admission"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/policy"
	"chronosync.local/engine/internal/snapshot"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func aid() admission.ID                 { return admission.ID(model.ID()) }
func adigest(c string) admission.Digest { return admission.Digest(strings.Repeat(c, 64)) }
func admissionSeed(s *Store) AdmissionSeed {
	obj := admission.ObjectIdentity{Path: "/synthetic/work", ArtifactDigest: adigest("a"), ObjectReference: adigest("b"), Status: admission.IdentityReference}
	spec := admission.BindingSpec{Version: 1, Admission: aid(), Epoch: s.AdmissionEpoch(), Adapter: "codex", AdapterVersion: "synthetic", AdapterConfiguration: adigest("c"), Executable: obj, Arguments: []string{"--synthetic"}, Input: admission.Input{Kind: admission.NoInput}, WorkingDirectory: obj, Workspace: obj, WorkspaceGeneration: 1, WorkspaceState: adigest("d"), PolicyRequest: adigest("e"), EffectiveRestrictions: adigest("f"), Target: policy.Target{Platform: "linux", Backend: "synthetic", BackendVersion: "1", ConfigurationDigest: string(adigest("a"))}}
	for _, d := range policy.Dimensions() {
		if d == policy.Approval || d == policy.Audit {
			continue
		}
		spec.Evidence = append(spec.Evidence, admission.EvidenceReference{Dimension: d, ID: aid(), Digest: adigest("a"), Provider: "synthetic", Generation: 1, ValidFrom: 1, ValidUntil: 10000})
	}
	return AdmissionSeed{spec, 100, 1000}
}
func admissionStore(t *testing.T) (*Store, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store")
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	return s, path
}
func createAdmission(t *testing.T, s *Store) AdmissionReceipt {
	t.Helper()
	r, e := s.CreateAdmission(context.Background(), AdmissionCreate{aid(), admissionSeed(s)})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func admissionRequest(t *testing.T, s *Store, r AdmissionReceipt, action admission.Action) AdmissionChange {
	t.Helper()
	record, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil {
		t.Fatal(e)
	}
	b, e := admission.NewBinding(record.Binding)
	if e != nil {
		t.Fatal(e)
	}
	c := admission.Command{Action: action, Epoch: r.Snapshot.Epoch, Now: r.Snapshot.LastTime + 1, Attempt: aid(), Checks: admission.Checks{Binding: b.Hash(), PolicyRequest: record.Binding.PolicyRequest, EffectiveRestrictions: record.Binding.EffectiveRestrictions, Outcome: policy.Satisfied, EvidenceGenerationDigest: b.EvidenceDigest(), Approval: r.Snapshot.Approval.ID, ApprovalValidation: admission.ExternalValidation, ValidFrom: 1, ValidUntil: 10000}}
	switch action {
	case admission.IssueChallenge:
		c.Challenge = admission.Challenge{ID: aid(), Binding: b.Hash(), Epoch: r.Snapshot.Epoch, IssuedAt: c.Now, ExpiresAt: 900}
	case admission.AcceptApproval:
		c.Approval = admission.ApprovalReference{ID: aid(), Challenge: r.Snapshot.Challenge.ID, Binding: b.Hash(), Epoch: r.Snapshot.Epoch, PrincipalReference: aid(), EvidenceDigest: adigest("b"), ApprovedAt: c.Now, ExpiresAt: 800}
	case admission.Revoke:
		c.Revocation = admission.RevocationReference{ID: aid(), Binding: b.Hash(), Generation: 1}
	}
	if action != admission.Claim {
		c.Attempt = ""
	}
	if action != admission.Claim && action != admission.Prepare && action != admission.FinalValidate && action != admission.BeginAttempt {
		c.Checks = admission.Checks{}
	}
	return AdmissionChange{r.Snapshot.Admission, aid(), r.Revision, c}
}
func advanceAdmission(t *testing.T, s *Store, r AdmissionReceipt, action admission.Action) AdmissionReceipt {
	t.Helper()
	got, e := s.ChangeAdmission(context.Background(), admissionRequest(t, s, r, action))
	if e != nil {
		t.Fatal(action, e)
	}
	return got
}
func preparedAdmission(t *testing.T, s *Store) AdmissionReceipt {
	t.Helper()
	r := createAdmission(t, s)
	for _, act := range []admission.Action{admission.IssueChallenge, admission.AcceptApproval, admission.Prepare} {
		r = advanceAdmission(t, s, r, act)
	}
	return r
}
func storageCode(e error) string {
	var typed *Error
	if errors.As(e, &typed) {
		return typed.Code
	}
	return ""
}
func TestAdmissionPersistenceAndRetry(t *testing.T) {
	s, path := admissionStore(t)
	ctx := context.Background()
	req := AdmissionCreate{aid(), admissionSeed(s)}
	first, e := s.CreateAdmission(ctx, req)
	if e != nil {
		t.Fatal(e)
	}
	again, e := s.CreateAdmission(ctx, req)
	if e != nil || !again.Replayed || again.Snapshot != first.Snapshot {
		t.Fatal(again, e)
	}
	bad := req
	bad.Seed.CreatedAt++
	if _, e = s.CreateAdmission(ctx, bad); storageCode(e) != "CONFLICT" {
		t.Fatal(e)
	}
	r := preparedAdmission(t, s)
	claim := admissionRequest(t, s, r, admission.Claim)
	got, e := s.ChangeAdmission(ctx, claim)
	if e != nil {
		t.Fatal(e)
	}
	retry, e := s.ChangeAdmission(ctx, claim)
	if e != nil || !retry.Replayed || retry.Snapshot != got.Snapshot {
		t.Fatal(retry, e)
	}
	conflict := claim
	conflict.Command.Now++
	if _, e = s.ChangeAdmission(ctx, conflict); storageCode(e) != "CONFLICT" {
		t.Fatal(e)
	}
	stale := claim
	stale.Operation = aid()
	if _, e = s.ChangeAdmission(ctx, stale); storageCode(e) != "CONFLICT" {
		t.Fatal(e)
	}
	r = advanceAdmission(t, s, got, admission.FinalValidate)
	r = advanceAdmission(t, s, r, admission.BeginAttempt)
	observe := admissionRequest(t, s, r, admission.ObserveNotStarted)
	r, e = s.ChangeAdmission(ctx, observe)
	if e != nil {
		t.Fatal(e)
	}
	again, e = s.ChangeAdmission(ctx, observe)
	if e != nil || !again.Replayed {
		t.Fatal(e)
	}
	if _, e = s.ChangeAdmission(ctx, admissionRequest(t, s, r, admission.Claim)); storageCode(e) != "ADMISSION_TRANSITION" {
		t.Fatal("reused failed claim", e)
	}
	oldEpoch := s.AdmissionEpoch()
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if s.AdmissionEpoch() == oldEpoch {
		t.Fatal("reused epoch")
	}
	stored, e := s.GetAdmission(ctx, r.Snapshot.Admission)
	if e != nil || stored.Snapshot.Outcome != admission.NotStarted || stored.Revision != r.Revision {
		t.Fatal(stored, e)
	}
	// Exact original receipt remains retrievable after reopen, never current readiness.
	retry, e = s.ChangeAdmission(ctx, claim)
	if e != nil || !retry.Replayed || retry.Revision != got.Revision {
		t.Fatal(retry, e)
	}
}
func TestAdmissionConcurrentClaimsAndOrdering(t *testing.T) {
	s, _ := admissionStore(t)
	r := preparedAdmission(t, s)
	requests := make([]AdmissionChange, 8)
	for i := range requests {
		requests[i] = admissionRequest(t, s, r, admission.Claim)
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make(chan error, 8)
	for _, request := range requests {
		wg.Add(1)
		go func(req AdmissionChange) {
			defer wg.Done()
			<-start
			_, e := s.ChangeAdmission(context.Background(), req)
			if e == nil {
				successes.Add(1)
			} else {
				errs <- e
			}
		}(request)
	}
	close(start)
	wg.Wait()
	close(errs)
	if successes.Load() != 1 {
		t.Fatal("claim count", successes.Load())
	}
	for e := range errs {
		if c := storageCode(e); c != "STORE_BUSY" && c != "CONFLICT" {
			t.Fatal(e)
		}
	}
	record, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil {
		t.Fatal(e)
	}
	if record.Revision != 4 || record.Snapshot.State != admission.Claimed {
		t.Fatal(record)
	}
	for i, a := range record.Audit {
		if a.Revision != int64(i) || a.Snapshot.LastOrder != uint64(i) {
			t.Fatal("nonconsecutive durable order")
		}
	}
	var n int
	s.db.QueryRow(`SELECT count(*) FROM admission_references WHERE kind='claim' AND admission_id=?`, r.Snapshot.Admission).Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
}
func TestAdmissionRecoveryAndClaimRace(t *testing.T) {
	s, path := admissionStore(t)
	ctx := context.Background()
	prepared := preparedAdmission(t, s)
	claimed := advanceAdmission(t, s, preparedAdmission(t, s), admission.Claim)
	request := admissionRequest(t, s, prepared, admission.Claim)
	var wg sync.WaitGroup
	wg.Add(2)
	var claimErr, recoveryErr error
	go func() { defer wg.Done(); _, claimErr = s.ChangeAdmission(ctx, request) }()
	go func() { defer wg.Done(); recoveryErr = s.RecoverAdmissions(ctx) }()
	wg.Wait()
	if recoveryErr != nil {
		t.Fatal(recoveryErr)
	}
	if claimErr != nil && storageCode(claimErr) != "ADMISSION_EPOCH_MISMATCH" && storageCode(claimErr) != "CONFLICT" {
		t.Fatal(claimErr)
	}
	for _, r := range []AdmissionReceipt{prepared, claimed} {
		got, e := s.GetAdmission(ctx, r.Snapshot.Admission)
		if e != nil {
			t.Fatal(e)
		}
		if got.Snapshot.State == admission.Claimed {
			if got.Snapshot.Outcome != admission.OutcomeUnknown {
				t.Fatal("invented outcome")
			}
		} else if got.Snapshot.State != admission.Invalidated {
			t.Fatal(got)
		}
	}
	before, e := s.GetAdmission(ctx, claimed.Snapshot.Admission)
	if e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	after, e := s.GetAdmission(ctx, claimed.Snapshot.Admission)
	if e != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("repeated recovery mutated unknown history", e)
	}
}
func TestAdmissionRevokeClaimRace(t *testing.T) {
	s, _ := admissionStore(t)
	r := preparedAdmission(t, s)
	claim := admissionRequest(t, s, r, admission.Claim)
	revoke := admissionRequest(t, s, r, admission.Revoke)
	var wg sync.WaitGroup
	var count atomic.Int32
	errs := make(chan error, 2)
	for _, q := range []AdmissionChange{claim, revoke} {
		wg.Add(1)
		go func(q AdmissionChange) {
			defer wg.Done()
			_, e := s.ChangeAdmission(context.Background(), q)
			if e == nil {
				count.Add(1)
			} else {
				errs <- e
			}
		}(q)
	}
	wg.Wait()
	close(errs)
	if count.Load() != 1 {
		t.Fatal(count.Load())
	}
	for e := range errs {
		if storageCode(e) != "CONFLICT" && storageCode(e) != "STORE_BUSY" {
			t.Fatal(e)
		}
	}
	record, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil {
		t.Fatal(e)
	}
	if record.Snapshot.State == admission.Claimed {
		receipt := AdmissionReceipt{record.Revision, record.Snapshot, false}
		q := admissionRequest(t, s, receipt, admission.Revoke)
		rev, e := s.ChangeAdmission(context.Background(), q)
		if e != nil {
			t.Fatal(e)
		}
		again, e := s.ChangeAdmission(context.Background(), q)
		if e != nil || !again.Replayed {
			t.Fatal(e)
		}
		if _, e = s.ChangeAdmission(context.Background(), admissionRequest(t, s, rev, admission.FinalValidate)); e == nil {
			t.Fatal("revocation bypass")
		}
	} else if record.Snapshot.State != admission.Invalidated {
		t.Fatal(record)
	}
}
func TestAdmissionFaultsAndReconciliation(t *testing.T) {
	for _, stage := range []string{"audit_write", "before_commit", "after_commit"} {
		t.Run(stage, func(t *testing.T) {
			s, _ := admissionStore(t)
			r := preparedAdmission(t, s)
			q := admissionRequest(t, s, r, admission.Claim)
			var fired atomic.Bool
			s.admissionFault = func(at string) error {
				if at == stage && !fired.Swap(true) {
					return errors.New("synthetic fault")
				}
				return nil
			}
			_, e := s.ChangeAdmission(context.Background(), q)
			if e == nil {
				t.Fatal("fault not observed")
			}
			s.admissionFault = nil
			got, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
			if e != nil {
				t.Fatal(e)
			}
			if stage == "after_commit" {
				if got.Revision != r.Revision+1 {
					t.Fatal("lost committed claim")
				}
				retry, e := s.ChangeAdmission(context.Background(), q)
				if e != nil || !retry.Replayed {
					t.Fatal(retry, e)
				}
			} else {
				if got.Revision != r.Revision {
					t.Fatal("partial transaction")
				}
				if _, e = s.ChangeAdmission(context.Background(), q); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}
func TestAdmissionBusyCancellationTimeoutAndDomainFailure(t *testing.T) {
	s, _ := admissionStore(t)
	r := preparedAdmission(t, s)
	q := admissionRequest(t, s, r, admission.Claim)
	s.writer <- struct{}{}
	_, e := s.ChangeAdmission(context.Background(), q)
	<-s.writer
	if storageCode(e) != "STORE_BUSY" {
		t.Fatal(e)
	}
	for _, timeout := range []bool{false, true} {
		ctx, cancel := context.WithCancel(context.Background())
		if timeout {
			cancel()
			ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		} else {
			cancel()
		}
		_, e = s.ChangeAdmission(ctx, q)
		cancel()
		if e == nil {
			t.Fatal("cancelled write committed")
		}
	}
	bad := q
	bad.Command.Checks.Outcome = policy.Refused
	if _, e = s.ChangeAdmission(context.Background(), bad); storageCode(e) != "ADMISSION_TRANSITION" {
		t.Fatal(e)
	}
	bad = q
	bad.Command.Order = 99
	if _, e = s.ChangeAdmission(context.Background(), bad); storageCode(e) != "INVALID_ARGUMENT" {
		t.Fatal(e)
	}
	record, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil || record.Revision != r.Revision {
		t.Fatal(record, e)
	}
}
func TestAdmissionCorruptionPreserved(t *testing.T) {
	cases := map[string]func(*Store, AdmissionReceipt) error{
		"missing-audit": func(s *Store, r AdmissionReceipt) error {
			_, e := s.db.Exec(`DELETE FROM admission_operations WHERE admission_id=? AND revision=0`, r.Snapshot.Admission)
			return e
		},
		"missing-reference": func(s *Store, r AdmissionReceipt) error {
			_, e := s.db.Exec(`DELETE FROM admission_references WHERE admission_id=? AND kind='claim'`, r.Snapshot.Admission)
			return e
		},
		"unknown-version": func(s *Store, r AdmissionReceipt) error {
			_, e := s.db.Exec(`UPDATE admission_records SET version=99 WHERE id=?`, r.Snapshot.Admission)
			return e
		},
		"projection": func(s *Store, r AdmissionReceipt) error {
			_, e := s.db.Exec(`UPDATE admission_records SET snapshot='{}' WHERE id=?`, r.Snapshot.Admission)
			return e
		},
		"hash": func(s *Store, r AdmissionReceipt) error {
			_, e := s.db.Exec(`UPDATE admission_records SET binding_hash=? WHERE id=?`, strings.Repeat("0", 64), r.Snapshot.Admission)
			return e
		},
		"audit-digest": func(s *Store, r AdmissionReceipt) error {
			_, e := s.db.Exec(`UPDATE admission_operations SET digest=? WHERE admission_id=? AND revision=0`, strings.Repeat("0", 64), r.Snapshot.Admission)
			return e
		},
	}
	for name, damage := range cases {
		t.Run(name, func(t *testing.T) {
			s, path := admissionStore(t)
			r := advanceAdmission(t, s, preparedAdmission(t, s), admission.Claim)
			if e := damage(s, r); e != nil {
				t.Fatal(e)
			}
			if e := s.RecoverAdmissions(context.Background()); e == nil {
				t.Fatal("damage accepted")
			}
			if s.AdmissionEpoch() != "" {
				t.Fatal("writes remained enabled")
			}
			s.Close()
			if reopened, e := Open(path); e == nil {
				reopened.Close()
				t.Fatal("corruption reopened")
			}
			if _, e := os.Stat(filepath.Join(path, "history.sqlite")); e != nil {
				t.Fatal("evidence deleted")
			}
		})
	}
}

func oldAdmissionDatabase(t *testing.T, version int) (string, *sql.DB) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "old-store")
	if e := os.Mkdir(path, 0700); e != nil {
		t.Fatal(e)
	}
	file := filepath.Join(path, "history.sqlite")
	f, e := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0600)
	if e != nil {
		t.Fatal(e)
	}
	f.Close()
	db, e := sql.Open("sqlite", file)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`PRAGMA foreign_keys=ON; CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,checksum TEXT NOT NULL)`); e != nil {
		t.Fatal(e)
	}
	for i, sqlText := range []string{migration, snapshotMigration} {
		if i >= version {
			break
		}
		if _, e = db.Exec(sqlText); e != nil {
			t.Fatal(e)
		}
		sum := sha256.Sum256([]byte(sqlText))
		if _, e = db.Exec(`INSERT INTO schema_migrations VALUES(?,?)`, i+1, hex.EncodeToString(sum[:])); e != nil {
			t.Fatal(e)
		}
	}
	return path, db
}
func TestAdmissionMigrationEveryVersionAndPreservedHistory(t *testing.T) {
	for _, version := range []int{0, 1, 2} {
		t.Run(fmt.Sprint(version), func(t *testing.T) {
			path, db := oldAdmissionDatabase(t, version)
			var run model.Run
			var ev model.Event
			var originalEvent []byte
			var snapID, branchID string
			var snapBody, branchBody []byte
			if version > 0 {
				old := &Store{db: db, writer: make(chan struct{}, 1)}
				run = model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created, CreatedAt: model.Now()}
				if e := old.Create(run); e != nil {
					t.Fatal(e)
				}
				ev = event(run, 1)
				if e := old.Append(ev); e != nil {
					t.Fatal(e)
				}
				if e := db.QueryRow(`SELECT data FROM events WHERE id=?`, ev.EventID).Scan(&originalEvent); e != nil {
					t.Fatal(e)
				}
				if version == 2 {
					snapID = model.ID()
					branchID = model.ID()
					snapBody, _ = json.Marshal(snapshot.Record{ID: snapID, RunID: run.ID, State: "failed", Workspace: "/synthetic/work", CreatedAt: model.Now()})
					branchBody, _ = json.Marshal(snapshot.Branch{ID: branchID, SnapshotID: snapID, RunID: run.ID, State: "failed", Kind: "branch", Workspace: "/synthetic/branch", CreatedAt: model.Now()})
					if _, e := db.Exec(`INSERT INTO snapshots(id,run_id,state,metadata) VALUES(?,?,'failed',?)`, snapID, run.ID, snapBody); e != nil {
						t.Fatal(e)
					}
					if _, e := db.Exec(`INSERT INTO branches(id,snapshot_id,run_id,state,metadata) VALUES(?,?,?,'failed',?)`, branchID, snapID, run.ID, branchBody); e != nil {
						t.Fatal(e)
					}
				}
			}
			db.Close()
			s, e := Open(path)
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			var n int
			if e = s.db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&n); e != nil || n != 3 {
				t.Fatal(n, e)
			}
			for i, text := range []string{migration, snapshotMigration} {
				var checksum string
				s.db.QueryRow(`SELECT checksum FROM schema_migrations WHERE version=?`, i+1).Scan(&checksum)
				sum := sha256.Sum256([]byte(text))
				if checksum != hex.EncodeToString(sum[:]) {
					t.Fatal("prior checksum rewritten")
				}
			}
			if version > 0 {
				var got []byte
				if e = s.db.QueryRow(`SELECT data FROM events WHERE id=?`, ev.EventID).Scan(&got); e != nil || string(got) != string(originalEvent) {
					t.Fatal("event bytes changed", e)
				}
				if _, e = s.GetEvent(context.Background(), run.ID, ev.EventID); e != nil {
					t.Fatal(e)
				}
			}
			if version == 2 {
				var got []byte
				s.db.QueryRow(`SELECT metadata FROM snapshots WHERE id=?`, snapID).Scan(&got)
				if string(got) != string(snapBody) {
					t.Fatal("snapshot changed")
				}
				s.db.QueryRow(`SELECT metadata FROM branches WHERE id=?`, branchID).Scan(&got)
				if string(got) != string(branchBody) {
					t.Fatal("branch changed")
				}
			}
			s.db.QueryRow(`SELECT count(*) FROM admission_records`).Scan(&n)
			if n != 0 {
				t.Fatal("fabricated admissions")
			}
		})
	}
}
func TestAdmissionMigrationAtomicFailureAndVersionRefusal(t *testing.T) {
	path, db := oldAdmissionDatabase(t, 2)
	if _, e := db.Exec(`CREATE TRIGGER reject_admission_migration BEFORE INSERT ON schema_migrations WHEN NEW.version=3 BEGIN SELECT RAISE(ABORT,'synthetic migration failure'); END`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	if s, e := Open(path); e == nil {
		s.Close()
		t.Fatal("migration failure ignored")
	}
	db, e := sql.Open("sqlite", filepath.Join(path, "history.sqlite"))
	if e != nil {
		t.Fatal(e)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE name LIKE 'admission_%'`).Scan(&n)
	if n != 0 {
		t.Fatal("partial migration", n)
	}
	db.QueryRow(`SELECT count(*) FROM schema_migrations`).Scan(&n)
	if n != 2 {
		t.Fatal("partial ledger")
	}
	if _, e = db.Exec(`DROP TRIGGER reject_admission_migration`); e != nil {
		t.Fatal(e)
	}
	db.Close()
	s, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.db.Exec(`UPDATE schema_migrations SET checksum='corrupted' WHERE version=3`); e != nil {
		t.Fatal(e)
	}
	s.Close()
	if other, e := Open(path); e == nil {
		other.Close()
		t.Fatal("checksum accepted")
	}
	s, path = admissionStore(t)
	s.db.Exec(`INSERT INTO schema_migrations VALUES(999,'future')`)
	s.Close()
	if other, e := Open(path); e == nil {
		other.Close()
		t.Fatal("future version accepted")
	}
}
func TestAdmissionSQLConstraintsAndUnknownOperation(t *testing.T) {
	s, _ := admissionStore(t)
	r := advanceAdmission(t, s, preparedAdmission(t, s), admission.Claim)
	ctx := context.Background()
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO admission_references VALUES('claim',?,?,?)`, []any{aid(), r.Snapshot.Admission, r.Revision}},
		{`INSERT INTO admission_references VALUES('revocation',?,?,999)`, []any{aid(), r.Snapshot.Admission}},
		{`INSERT INTO admission_operations SELECT admission_id,revision,?,request,digest,result FROM admission_operations WHERE admission_id=? AND revision=0`, []any{aid(), r.Snapshot.Admission}},
	}
	for _, q := range statements {
		if _, e := s.db.Exec(q.query, q.args...); e == nil {
			t.Fatal("constraint ignored")
		}
	}
	badSeed := admissionSeed(s)
	badSeed.Binding.ParentRun = aid()
	if _, e := s.CreateAdmission(ctx, AdmissionCreate{aid(), badSeed}); e == nil {
		t.Fatal("broken parent accepted")
	}
	var raw []byte
	if e := s.db.QueryRow(`SELECT request FROM admission_operations WHERE admission_id=? AND revision=0`, r.Snapshot.Admission).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var op admissionOperation
	if e := json.Unmarshal(raw, &op); e != nil {
		t.Fatal(e)
	}
	op.Version = 99
	raw, _ = json.Marshal(op)
	if _, e := s.db.Exec(`UPDATE admission_operations SET request=?,digest=? WHERE admission_id=? AND revision=0`, raw, admissionHash(raw), r.Snapshot.Admission); e != nil {
		t.Fatal(e)
	}
	if _, e := s.GetAdmission(ctx, r.Snapshot.Admission); storageCode(e) != "UNSUPPORTED_SCHEMA" {
		t.Fatal(e)
	}
}
func TestAdmissionStoppedBackupRestoreInvalidates(t *testing.T) {
	s, path := admissionStore(t)
	r := preparedAdmission(t, s)
	oldEpoch := s.AdmissionEpoch()
	s.Close()
	// Copy a stopped complete store into a distinct private directory. No live DB
	// copying or restored-positive-admission assertion is made.
	restore := filepath.Join(t.TempDir(), "restored")
	if e := os.Mkdir(restore, 0700); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"history.sqlite", "history.sqlite-wal", "history.sqlite-shm"} {
		data, e := os.ReadFile(filepath.Join(path, name))
		if errors.Is(e, os.ErrNotExist) {
			continue
		}
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(restore, name), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
	reopened, e := Open(restore)
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	got, e := reopened.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil || got.Snapshot.State != admission.Invalidated || reopened.AdmissionEpoch() == oldEpoch {
		t.Fatal(got, e)
	}
}

func TestAdmissionDatabaseContentionAndPrivateEnvelope(t *testing.T) {
	s, _ := admissionStore(t)
	r := preparedAdmission(t, s)
	request := admissionRequest(t, s, r, admission.Claim)
	tx, e := s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE admission_records SET revision=revision WHERE id=?`, r.Snapshot.Admission); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	_, e = s.ChangeAdmission(ctx, request)
	cancel()
	tx.Rollback()
	if e == nil {
		t.Fatal("concurrent SQLite writer unexpectedly committed")
	}
	got, e := s.GetAdmission(context.Background(), r.Snapshot.Admission)
	if e != nil || got.Revision != r.Revision {
		t.Fatal(got, e)
	}
	// Unused fields are not a storage channel for values ignored by the domain.
	bad := admissionRequest(t, s, r, admission.Refuse)
	bad.Command.Checks.Outcome = "unreviewed-data"
	if _, e = s.ChangeAdmission(context.Background(), bad); storageCode(e) != "INVALID_ARGUMENT" {
		t.Fatal(e)
	}
	if _, e = s.ChangeAdmission(context.Background(), request); e != nil {
		t.Fatal(e)
	}
}
func TestAdmissionOldEpochAndMissingForeignKey(t *testing.T) {
	s, path := admissionStore(t)
	seed := admissionSeed(s)
	if e := s.RecoverAdmissions(context.Background()); e != nil {
		t.Fatal(e)
	}
	if _, e := s.CreateAdmission(context.Background(), AdmissionCreate{aid(), seed}); storageCode(e) != "ADMISSION_EPOCH_MISMATCH" {
		t.Fatal(e)
	}
	r := createAdmission(t, s)
	conn, e := s.db.Conn(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	if _, e = conn.ExecContext(context.Background(), `PRAGMA foreign_keys=OFF`); e != nil {
		t.Fatal(e)
	}
	if _, e = conn.ExecContext(context.Background(), `UPDATE admission_records SET parent_run=? WHERE id=?`, model.ID(), r.Snapshot.Admission); e != nil {
		t.Fatal(e)
	}
	conn.Close()
	s.Close()
	if reopened, e := Open(path); e == nil {
		reopened.Close()
		t.Fatal("broken reference reopened")
	}
}
