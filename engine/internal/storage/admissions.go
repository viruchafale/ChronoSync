package storage

import (
	"bytes"
	"chronosync.local/engine/internal/admission"
	"chronosync.local/engine/internal/model"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"
)

const admissionMigration = `
CREATE TABLE admission_records(id TEXT PRIMARY KEY CHECK(length(id)=32),version INTEGER NOT NULL CHECK(version>0),binding BLOB NOT NULL CHECK(length(binding)<=1048576),binding_hash TEXT NOT NULL CHECK(length(binding_hash)=64),created INTEGER NOT NULL,expires INTEGER NOT NULL CHECK(expires>created),parent_run TEXT REFERENCES runs(id),revision INTEGER NOT NULL CHECK(revision>=0 AND revision<64),snapshot BLOB NOT NULL CHECK(length(snapshot)<=65536));
CREATE TABLE admission_operations(admission_id TEXT NOT NULL REFERENCES admission_records(id),revision INTEGER NOT NULL CHECK(revision>=0 AND revision<64),operation_id TEXT NOT NULL CHECK(length(operation_id)=32),request BLOB NOT NULL CHECK(length(request)<=1048576),digest TEXT NOT NULL CHECK(length(digest)=64),result BLOB NOT NULL CHECK(length(result)<=65536),PRIMARY KEY(admission_id,revision),UNIQUE(admission_id,operation_id));
CREATE TABLE admission_references(kind TEXT NOT NULL CHECK(kind IN ('challenge','approval','claim','revocation')),identity TEXT NOT NULL CHECK(length(identity)=32),admission_id TEXT NOT NULL REFERENCES admission_records(id),revision INTEGER NOT NULL,PRIMARY KEY(kind,identity),UNIQUE(admission_id,kind),FOREIGN KEY(admission_id,revision) REFERENCES admission_operations(admission_id,revision));
`
const admissionStorageVersion = 1
const maxAdmissionOperations = 64

type AdmissionRepository interface {
	AdmissionEpoch() admission.Epoch
	CreateAdmission(context.Context, AdmissionCreate) (AdmissionReceipt, error)
	ChangeAdmission(context.Context, AdmissionChange) (AdmissionReceipt, error)
	GetAdmission(context.Context, admission.ID) (AdmissionRecord, error)
	RecoverAdmissions(context.Context) error
}
type AdmissionSeed struct {
	Binding              admission.BindingSpec
	CreatedAt, ExpiresAt int64
}
type AdmissionCreate struct {
	Operation admission.ID
	Seed      AdmissionSeed
}
type AdmissionChange struct {
	ID               admission.ID
	Operation        admission.ID
	ExpectedRevision int64
	Command          admission.Command
}
type AdmissionReceipt struct {
	Revision int64
	Snapshot admission.Snapshot
	Replayed bool
}
type AdmissionRecord struct {
	Binding  admission.BindingSpec
	Revision int64
	Snapshot admission.Snapshot
	Audit    []AdmissionAudit
}
type AdmissionAudit struct {
	Revision  int64
	Operation admission.ID
	Action    admission.Action
	Snapshot  admission.Snapshot
}
type admissionOperation struct {
	Version          int
	ID               admission.ID
	ExpectedRevision int64
	Seed             *AdmissionSeed     `json:",omitempty"`
	Command          *admission.Command `json:",omitempty"`
}
type storedAdmission struct {
	domain     admission.Admission
	binding    admission.Binding
	revision   int64
	operations []admissionOperation
	results    []admission.Snapshot
	raw        [][]byte
}
type admissionQuery interface {
	rowQuery
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func admissionCorrupt() error  { return &Error{"CORRUPT_ADMISSION", "admission integrity"} }
func admissionConflict() error { return &Error{"CONFLICT", "admission revision or operation"} }
func admissionID(id admission.ID) bool {
	return model.ValidID(string(id)) && string(id) != "00000000000000000000000000000000"
}
func admissionHash(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func admissionJSON(v any, limit int) ([]byte, error) {
	b, e := json.Marshal(v)
	if e != nil || len(b) > limit {
		return nil, invalid("admission body bound")
	}
	return b, nil
}
func admissionDecode(b []byte, v any, limit int) error {
	if len(b) == 0 || len(b) > limit {
		return admissionCorrupt()
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		return admissionCorrupt()
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return admissionCorrupt()
	}
	// Persisted bodies must use the exact typed encoding. This also rejects
	// duplicate JSON keys, alternative casing and ignored/missing fields.
	canonical, e := json.Marshal(v)
	if e != nil || !bytes.Equal(b, canonical) {
		return admissionCorrupt()
	}
	return nil
}
func (s *Store) AdmissionEpoch() admission.Epoch {
	s.admissionMu.RLock()
	defer s.admissionMu.RUnlock()
	if !s.admissionReady {
		return ""
	}
	return s.admissionEpoch
}
func (s *Store) admissionFaultAt(stage string) error {
	if s.admissionFault != nil {
		return s.admissionFault(stage)
	}
	return nil
}

// Reject irrelevant command fields rather than persisting arbitrary unused data.
// This is a storage envelope rule; legal domain transitions still use Apply.
func admissionCommandShape(c admission.Command) bool {
	base := admission.Command{Action: c.Action, Epoch: c.Epoch, Order: c.Order, Now: c.Now}
	switch c.Action {
	case admission.IssueChallenge:
		base.Challenge = c.Challenge
	case admission.AcceptApproval:
		base.Approval = c.Approval
	case admission.Prepare, admission.FinalValidate, admission.BeginAttempt:
		base.Checks = c.Checks
	case admission.Claim:
		base.Checks = c.Checks
		base.Attempt = c.Attempt
	case admission.Revoke:
		base.Revocation = c.Revocation
	case admission.ObserveNotStarted, admission.ObserveStarted, admission.ObserveUnknown, admission.Refuse, admission.Expire, admission.Recover:
	default:
		return false
	}
	return c == base
}

func admissionReference(c *admission.Command) (string, admission.ID) {
	if c == nil {
		return "", ""
	}
	switch c.Action {
	case admission.IssueChallenge:
		return "challenge", c.Challenge.ID
	case admission.AcceptApproval:
		return "approval", c.Approval.ID
	case admission.Claim:
		return "claim", c.Attempt
	case admission.Revoke:
		return "revocation", c.Revocation.ID
	}
	return "", ""
}

// Replay uses the domain as the sole transition/canonicalization authority.
func loadAdmission(ctx context.Context, q admissionQuery, id admission.ID) (storedAdmission, error) {
	var out storedAdmission
	var version int
	var raw, projection []byte
	var hash string
	var created, expires int64
	var parent sql.NullString
	e := q.QueryRowContext(ctx, `SELECT version,binding,binding_hash,created,expires,parent_run,revision,snapshot FROM admission_records WHERE id=?`, id).Scan(&version, &raw, &hash, &created, &expires, &parent, &out.revision, &projection)
	if e != nil {
		return out, failure("read admission", e)
	}
	if version != admissionStorageVersion {
		return out, &Error{"UNSUPPORTED_SCHEMA", "admission storage version"}
	}
	if out.revision < 0 || out.revision >= maxAdmissionOperations {
		return out, admissionCorrupt()
	}
	var spec admission.BindingSpec
	if e = admissionDecode(raw, &spec, 1048576); e != nil {
		return out, e
	}
	out.binding, e = admission.NewBinding(spec)
	if e != nil {
		return out, admissionCorrupt()
	}
	if out.binding.Hash() != admission.Digest(hash) || spec.Admission != id || string(spec.ParentRun) != parent.String || parent.Valid != (spec.ParentRun != "") {
		return out, admissionCorrupt()
	}
	out.domain, e = admission.New(out.binding, created, expires)
	if e != nil {
		return out, admissionCorrupt()
	}
	rows, e := q.QueryContext(ctx, `SELECT revision,operation_id,request,digest,result FROM admission_operations WHERE admission_id=? ORDER BY revision`, id)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	expectedRefs := map[string]struct {
		id       admission.ID
		revision int64
	}{}
	for rows.Next() {
		if len(out.operations) >= maxAdmissionOperations {
			return out, admissionCorrupt()
		}
		var revision int64
		var opID, checksum string
		var body, result []byte
		if e = rows.Scan(&revision, &opID, &body, &checksum, &result); e != nil {
			return out, e
		}
		if revision != int64(len(out.operations)) || admissionHash(body) != checksum {
			return out, admissionCorrupt()
		}
		var op admissionOperation
		if e = admissionDecode(body, &op, 1048576); e != nil {
			return out, e
		}
		if op.Version != admissionStorageVersion {
			return out, &Error{"UNSUPPORTED_SCHEMA", "admission operation version"}
		}
		if !admissionID(op.ID) || op.ID != admission.ID(opID) || op.ExpectedRevision != revision-1 {
			return out, admissionCorrupt()
		}
		if revision == 0 {
			if op.Seed == nil || op.Command != nil || op.Seed.CreatedAt != created || op.Seed.ExpiresAt != expires || !reflect.DeepEqual(op.Seed.Binding, spec) {
				return out, admissionCorrupt()
			}
		} else {
			if op.Seed != nil || op.Command == nil || len(body) > 65536 || op.Command.Order != uint64(revision) || !admissionCommandShape(*op.Command) {
				return out, admissionCorrupt()
			}
			out.domain, e = admission.Apply(out.domain, *op.Command)
			if e != nil {
				return out, admissionCorrupt()
			}
			if kind, ref := admissionReference(op.Command); kind != "" {
				if _, ok := expectedRefs[kind]; ok {
					return out, admissionCorrupt()
				}
				expectedRefs[kind] = struct {
					id       admission.ID
					revision int64
				}{ref, revision}
			}
		}
		var snapshot admission.Snapshot
		if e = admissionDecode(result, &snapshot, 65536); e != nil {
			return out, e
		}
		if snapshot != out.domain.Snapshot() {
			return out, admissionCorrupt()
		}
		out.operations = append(out.operations, op)
		out.results = append(out.results, snapshot)
		out.raw = append(out.raw, body)
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	rows.Close()
	if int64(len(out.operations)) != out.revision+1 {
		return out, admissionCorrupt()
	}
	var snapshot admission.Snapshot
	if e = admissionDecode(projection, &snapshot, 65536); e != nil {
		return out, e
	}
	if snapshot != out.domain.Snapshot() {
		return out, admissionCorrupt()
	}
	refs, e := q.QueryContext(ctx, `SELECT kind,identity,revision FROM admission_references WHERE admission_id=?`, id)
	if e != nil {
		return out, e
	}
	defer refs.Close()
	count := 0
	for refs.Next() {
		var kind, identity string
		var revision int64
		if e = refs.Scan(&kind, &identity, &revision); e != nil {
			return out, e
		}
		want, ok := expectedRefs[kind]
		if !ok || want.id != admission.ID(identity) || want.revision != revision {
			return out, admissionCorrupt()
		}
		count++
	}
	if e = refs.Err(); e != nil {
		return out, e
	}
	if count != len(expectedRefs) {
		return out, admissionCorrupt()
	}
	return out, nil
}
func admissionRetry(a storedAdmission, op admissionOperation, raw []byte) (AdmissionReceipt, bool, error) {
	for i, old := range a.operations {
		if old.ID == op.ID {
			if !bytes.Equal(a.raw[i], raw) {
				return AdmissionReceipt{}, true, admissionConflict()
			}
			return AdmissionReceipt{int64(i), a.results[i], true}, true, nil
		}
	}
	return AdmissionReceipt{}, false, nil
}
func (s *Store) CreateAdmission(ctx context.Context, r AdmissionCreate) (AdmissionReceipt, error) {
	s.admissionMu.RLock()
	defer s.admissionMu.RUnlock()
	var receipt AdmissionReceipt
	if !s.admissionReady {
		return receipt, &Error{"ADMISSION_RECOVERY_REQUIRED", "admission writes disabled"}
	}
	if !admissionID(r.Operation) {
		return receipt, invalid("admission operation id")
	}
	b, e := admission.NewBinding(r.Seed.Binding)
	if e != nil {
		return receipt, invalid("admission binding")
	}
	r.Seed.Binding = b.Spec()
	a, e := admission.New(b, r.Seed.CreatedAt, r.Seed.ExpiresAt)
	if e != nil {
		return receipt, invalid("admission initial state")
	}
	op := admissionOperation{Version: 1, ID: r.Operation, ExpectedRevision: -1, Seed: &r.Seed}
	raw, e := admissionJSON(op, 1048576)
	if e != nil {
		return receipt, e
	}
	e = s.write(ctx, func(tx *sql.Tx) error {
		old, err := loadAdmission(ctx, tx, r.Seed.Binding.Admission)
		if err == nil {
			var ok bool
			receipt, ok, err = admissionRetry(old, op, raw)
			if ok {
				return err
			}
			return admissionConflict()
		}
		var typed *Error
		if !errors.As(err, &typed) || typed.Code != "NOT_FOUND" {
			return err
		}
		if b.Spec().Epoch != s.admissionEpoch {
			return &Error{"ADMISSION_EPOCH_MISMATCH", "new admission epoch"}
		}
		binding, err := admissionJSON(b.Spec(), 1048576)
		if err != nil {
			return err
		}
		snap, err := admissionJSON(a.Snapshot(), 65536)
		if err != nil {
			return err
		}
		var parent any
		if b.Spec().ParentRun != "" {
			parent = b.Spec().ParentRun
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO admission_records VALUES(?,?,?,?,?,?,?,?,?)`, b.Spec().Admission, 1, binding, b.Hash(), r.Seed.CreatedAt, r.Seed.ExpiresAt, parent, 0, snap)
		if err != nil {
			return err
		}
		if err = s.appendAdmission(ctx, tx, b.Spec().Admission, 0, op, raw, snap); err != nil {
			return err
		}
		receipt = AdmissionReceipt{0, a.Snapshot(), false}
		return s.admissionFaultAt("before_commit")
	})
	if e != nil {
		return AdmissionReceipt{}, e
	}
	if e = s.admissionFaultAt("after_commit"); e != nil {
		return AdmissionReceipt{}, &Error{"ADMISSION_RESULT_UNKNOWN", "reconcile operation id"}
	}
	return receipt, nil
}
func (s *Store) appendAdmission(ctx context.Context, tx *sql.Tx, id admission.ID, revision int64, op admissionOperation, raw, result []byte) error {
	if e := s.admissionFaultAt("audit_write"); e != nil {
		return e
	}
	if _, e := tx.ExecContext(ctx, `INSERT INTO admission_operations VALUES(?,?,?,?,?,?)`, id, revision, op.ID, raw, admissionHash(raw), result); e != nil {
		return e
	}
	if kind, identity := admissionReference(op.Command); kind != "" {
		_, e := tx.ExecContext(ctx, `INSERT INTO admission_references VALUES(?,?,?,?)`, kind, identity, id, revision)
		return e
	}
	return nil
}
func (s *Store) changeAdmissionTx(ctx context.Context, tx *sql.Tx, r AdmissionChange, recovery bool) (AdmissionReceipt, error) {
	var receipt AdmissionReceipt
	a, e := loadAdmission(ctx, tx, r.ID)
	if e != nil {
		return receipt, e
	}
	r.Command.Order = uint64(r.ExpectedRevision + 1)
	op := admissionOperation{Version: 1, ID: r.Operation, ExpectedRevision: r.ExpectedRevision, Command: &r.Command}
	raw, e := admissionJSON(op, 65536)
	if e != nil {
		return receipt, e
	}
	if cached, ok, err := admissionRetry(a, op, raw); ok {
		return cached, err
	}
	if a.revision != r.ExpectedRevision {
		return receipt, admissionConflict()
	}
	if a.revision+1 >= maxAdmissionOperations {
		return receipt, &Error{"ADMISSION_LIMIT", "audit bound"}
	}
	if !recovery && (a.domain.Snapshot().Epoch != s.admissionEpoch || r.Command.Epoch != s.admissionEpoch) {
		return receipt, &Error{"ADMISSION_EPOCH_MISMATCH", "admission not current"}
	}
	next, e := admission.Apply(a.domain, r.Command)
	if e != nil {
		return receipt, &Error{"ADMISSION_TRANSITION", "domain refused transition"}
	}
	result, e := admissionJSON(next.Snapshot(), 65536)
	if e != nil {
		return receipt, e
	}
	updated, e := tx.ExecContext(ctx, `UPDATE admission_records SET revision=?,snapshot=? WHERE id=? AND revision=?`, a.revision+1, result, r.ID, r.ExpectedRevision)
	if e != nil {
		return receipt, e
	}
	n, e := updated.RowsAffected()
	if e != nil {
		return receipt, e
	}
	if n != 1 {
		return receipt, admissionConflict()
	}
	if e = s.appendAdmission(ctx, tx, r.ID, a.revision+1, op, raw, result); e != nil {
		return receipt, e
	}
	return AdmissionReceipt{a.revision + 1, next.Snapshot(), false}, s.admissionFaultAt("before_commit")
}
func (s *Store) ChangeAdmission(ctx context.Context, r AdmissionChange) (AdmissionReceipt, error) {
	s.admissionMu.RLock()
	defer s.admissionMu.RUnlock()
	var out AdmissionReceipt
	if !s.admissionReady {
		return out, &Error{"ADMISSION_RECOVERY_REQUIRED", "admission writes disabled"}
	}
	if !admissionID(r.ID) || !admissionID(r.Operation) || r.ExpectedRevision < 0 || r.ExpectedRevision >= maxAdmissionOperations || r.Command.Order != 0 || r.Command.Action == admission.Recover || !admissionCommandShape(r.Command) {
		return out, invalid("admission transition request")
	}
	e := s.write(ctx, func(tx *sql.Tx) error { var err error; out, err = s.changeAdmissionTx(ctx, tx, r, false); return err })
	if e != nil {
		return AdmissionReceipt{}, e
	}
	if e = s.admissionFaultAt("after_commit"); e != nil {
		return AdmissionReceipt{}, &Error{"ADMISSION_RESULT_UNKNOWN", "reconcile operation id"}
	}
	return out, nil
}
func (s *Store) GetAdmission(ctx context.Context, id admission.ID) (AdmissionRecord, error) {
	var out AdmissionRecord
	if !admissionID(id) {
		return out, invalid("admission id")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if e != nil {
		return out, failure("admission read", e)
	}
	defer tx.Rollback()
	a, e := loadAdmission(ctx, tx, id)
	if e != nil {
		return out, failure("admission read", e)
	}
	out = AdmissionRecord{Binding: a.binding.Spec(), Revision: a.revision, Snapshot: a.domain.Snapshot()}
	for i, op := range a.operations {
		action := admission.Action("create")
		if op.Command != nil {
			action = op.Command.Action
		}
		out.Audit = append(out.Audit, AdmissionAudit{int64(i), op.ID, action, a.results[i]})
	}
	return out, nil
}

// Recovery is serialized against all admission mutations. It never launches or
// kills and never reactivates a prior epoch. Failure keeps writes disabled.
func (s *Store) RecoverAdmissions(ctx context.Context) error {
	s.admissionMu.Lock()
	defer s.admissionMu.Unlock()
	s.admissionReady = false
	epoch, err := admission.NewEpoch(admission.ID(model.ID()))
	if err != nil {
		return err
	}
	s.admissionEpoch = epoch
	var integrity string
	if e := s.db.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&integrity); e != nil || integrity != "ok" {
		return &Error{"CORRUPT_DATABASE", "admission recovery integrity check"}
	}
	fk, e := s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if e != nil {
		return failure("admission foreign keys", e)
	}
	bad := fk.Next()
	e = fk.Err()
	fk.Close()
	if e != nil || bad {
		return &Error{"CORRUPT_DATABASE", "admission recovery foreign keys"}
	}
	after := ""
	for {
		rows, e := s.db.QueryContext(ctx, `SELECT id FROM admission_records WHERE id>? ORDER BY id LIMIT 100`, after)
		if e != nil {
			return failure("admission inventory", e)
		}
		var ids []admission.ID
		for rows.Next() {
			var id admission.ID
			if e = rows.Scan(&id); e != nil {
				break
			}
			ids = append(ids, id)
		}
		if e == nil {
			e = rows.Err()
		}
		rows.Close()
		if e != nil {
			return failure("admission inventory", e)
		}
		if len(ids) == 0 {
			break
		}
		for _, id := range ids {
			e = s.write(ctx, func(tx *sql.Tx) error {
				a, e := loadAdmission(ctx, tx, id)
				if e != nil {
					return e
				}
				v := a.domain.Snapshot()
				if v.State == admission.Refused || v.State == admission.ExpiredState || v.State == admission.Invalidated || (v.State == admission.Claimed && v.Outcome != admission.Pending) {
					return nil
				}
				// Monotonic clamp is for conservative invalidation/unknown only, never
				// fresh readiness or evidence. Caller clocks cannot renew an admission.
				now := max(time.Now().UnixMilli(), v.LastTime)
				_, e = s.changeAdmissionTx(ctx, tx, AdmissionChange{ID: id, Operation: admission.ID(model.ID()), ExpectedRevision: a.revision, Command: admission.Command{Action: admission.Recover, Epoch: epoch, Now: now}}, true)
				return e
			})
			if e != nil {
				return e
			}
		}
		after = string(ids[len(ids)-1])
	}
	s.admissionReady = true
	return nil
}
