package storage

import (
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/snapshot"
	"context"
	"database/sql"
	"encoding/json"
)

const snapshotMigration = `
CREATE TABLE snapshots(id TEXT PRIMARY KEY,run_id TEXT NOT NULL REFERENCES runs(id),event_id TEXT REFERENCES events(id),branch_id TEXT REFERENCES branches(id),state TEXT NOT NULL CHECK(state IN ('preparing','committed','failed','damaged')),manifest_hash TEXT,metadata BLOB NOT NULL);
CREATE TABLE branches(id TEXT PRIMARY KEY,snapshot_id TEXT NOT NULL REFERENCES snapshots(id),parent_id TEXT REFERENCES branches(id),run_id TEXT NOT NULL REFERENCES runs(id),state TEXT NOT NULL CHECK(state IN ('preparing','ready','failed')),metadata BLOB NOT NULL);
CREATE TABLE objects(hash TEXT PRIMARY KEY CHECK(length(hash)=64),size INTEGER NOT NULL CHECK(size>=0 AND size<=8388608));
CREATE TABLE snapshot_objects(snapshot_id TEXT NOT NULL REFERENCES snapshots(id),hash TEXT NOT NULL REFERENCES objects(hash),PRIMARY KEY(snapshot_id,hash));
CREATE INDEX snapshots_run ON snapshots(run_id,id);
CREATE INDEX branches_run ON branches(run_id,id);
`

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
func (s *Store) PrepareSnapshot(ctx context.Context, r snapshot.Record) error {
	if !model.ValidID(r.ID) || !model.ValidID(r.RunID) || r.State != "preparing" {
		return invalid("snapshot record")
	}
	return s.write(ctx, func(tx *sql.Tx) error {
		raw, _ := json.Marshal(r)
		if r.EventID != "" {
			var run string
			if err := tx.QueryRowContext(ctx, `SELECT run_id FROM events WHERE id=?`, r.EventID).Scan(&run); err != nil {
				return err
			}
			if run != r.RunID {
				return invalid("event run mismatch")
			}
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO snapshots(id,run_id,event_id,branch_id,state,metadata) VALUES(?,?,?,?,?,?)`, r.ID, r.RunID, nullable(r.EventID), nullable(r.BranchID), r.State, raw)
		return err
	})
}
func (s *Store) CommitSnapshot(ctx context.Context, r snapshot.Record, refs []snapshot.Ref) error {
	if r.State != "committed" || r.Manifest == nil || !snapshot.ValidRef(*r.Manifest) {
		return invalid("snapshot commit")
	}
	return s.write(ctx, func(tx *sql.Tx) error {
		for _, ref := range refs {
			if !snapshot.ValidRef(ref) {
				return invalid("object reference")
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO objects(hash,size) VALUES(?,?) ON CONFLICT(hash) DO NOTHING`, ref.Hash, ref.Size); err != nil {
				return err
			}
			var size int64
			if err := tx.QueryRowContext(ctx, `SELECT size FROM objects WHERE hash=?`, ref.Hash).Scan(&size); err != nil {
				return err
			}
			if size != ref.Size {
				return invalid("object length conflict")
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO snapshot_objects VALUES(?,?) ON CONFLICT DO NOTHING`, r.ID, ref.Hash); err != nil {
				return err
			}
		}
		raw, _ := json.Marshal(r)
		result, err := tx.ExecContext(ctx, `UPDATE snapshots SET state='committed',manifest_hash=?,metadata=? WHERE id=? AND state='preparing'`, r.Manifest.Hash, raw, r.ID)
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return &Error{"CONFLICT", "snapshot commit state"}
		}
		return nil
	})
}
func (s *Store) Snapshot(ctx context.Context, id string) (snapshot.Record, error) {
	var r snapshot.Record
	if !model.ValidID(id) {
		return r, invalid("snapshot id")
	}
	var raw []byte
	var state, run string
	var hash sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT metadata,state,run_id,manifest_hash FROM snapshots WHERE id=?`, id).Scan(&raw, &state, &run, &hash)
	if err != nil {
		return r, failure("snapshot", err)
	}
	if len(raw) > 65536 || json.Unmarshal(raw, &r) != nil || r.ID != id || r.RunID != run || r.State != state || (state == "committed" && (r.Manifest == nil || r.Manifest.Hash != hash.String)) {
		return r, &Error{"CORRUPT_METADATA", "snapshot"}
	}
	return r, nil
}
func (s *Store) SnapshotState(ctx context.Context, id, state, diagnostic string) error {
	return s.write(ctx, func(tx *sql.Tx) error {
		r, err := s.Snapshot(ctx, id)
		if err != nil {
			return err
		}
		if !(r.State == "preparing" && state == "failed" || r.State == "committed" && state == "damaged") {
			return &Error{"CONFLICT", "snapshot state"}
		}
		r.State = state
		r.Diagnostic = diagnostic
		raw, _ := json.Marshal(r)
		_, err = tx.ExecContext(ctx, `UPDATE snapshots SET state=?,metadata=? WHERE id=?`, state, raw, id)
		return err
	})
}
func (s *Store) PrepareBranch(ctx context.Context, b snapshot.Branch) error {
	if !model.ValidID(b.ID) || !model.ValidID(b.SnapshotID) || b.State != "preparing" {
		return invalid("branch")
	}
	return s.write(ctx, func(tx *sql.Tx) error {
		raw, _ := json.Marshal(b)
		_, err := tx.ExecContext(ctx, `INSERT INTO branches VALUES(?,?,?,?,?,?)`, b.ID, b.SnapshotID, nullable(b.ParentID), b.RunID, b.State, raw)
		return err
	})
}
func (s *Store) Branch(ctx context.Context, id string) (snapshot.Branch, error) {
	var b snapshot.Branch
	if !model.ValidID(id) {
		return b, invalid("branch id")
	}
	var raw []byte
	var state, source, run string
	var parent sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT metadata,state,snapshot_id,run_id,parent_id FROM branches WHERE id=?`, id).Scan(&raw, &state, &source, &run, &parent)
	if err != nil {
		return b, failure("branch", err)
	}
	if len(raw) > 65536 || json.Unmarshal(raw, &b) != nil || b.ID != id || b.State != state || b.SnapshotID != source || b.RunID != run || b.ParentID != parent.String {
		return b, &Error{"CORRUPT_METADATA", "branch"}
	}
	return b, nil
}
func (s *Store) FinishBranch(ctx context.Context, id, state, diagnostic string) error {
	return s.write(ctx, func(tx *sql.Tx) error {
		b, err := s.Branch(ctx, id)
		if err != nil {
			return err
		}
		if b.State != "preparing" || (state != "ready" && state != "failed") {
			return &Error{"CONFLICT", "branch state"}
		}
		b.State = state
		b.Diagnostic = diagnostic
		raw, _ := json.Marshal(b)
		_, err = tx.ExecContext(ctx, `UPDATE branches SET state=?,metadata=? WHERE id=?`, state, raw, id)
		return err
	})
}
func (s *Store) snapshotIDs(ctx context.Context, table string, q snapshot.Query) ([]string, error) {
	if q.RunID != "" && !model.ValidID(q.RunID) || q.After != "" && !model.ValidID(q.After) {
		return nil, invalid("snapshot query")
	}
	limit, err := pageLimit(q.Limit)
	if err != nil {
		return nil, err
	}
	query := `SELECT id FROM ` + table + ` WHERE id>?`
	args := []any{q.After}
	if q.RunID != "" {
		query += ` AND run_id=?`
		args = append(args, q.RunID)
	}
	query += ` ORDER BY id LIMIT ?`
	args = append(args, limit+1)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, failure("snapshot list", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, failure("snapshot list", err)
		}
		ids = append(ids, id)
	}
	return ids, failure("snapshot list", rows.Err())
}
func (s *Store) Snapshots(ctx context.Context, q snapshot.Query) (snapshot.Page, error) {
	p := snapshot.Page{Snapshots: []snapshot.Record{}}
	ids, err := s.snapshotIDs(ctx, "snapshots", q)
	if err != nil {
		return p, err
	}
	limit, _ := pageLimit(q.Limit)
	for i, id := range ids {
		if i == limit {
			p.Next = ids[i-1]
			break
		}
		r, err := s.Snapshot(ctx, id)
		if err != nil {
			return p, err
		}
		p.Snapshots = append(p.Snapshots, r)
	}
	return p, nil
}
func (s *Store) Branches(ctx context.Context, q snapshot.Query) (snapshot.BranchPage, error) {
	p := snapshot.BranchPage{Branches: []snapshot.Branch{}}
	ids, err := s.snapshotIDs(ctx, "branches", q)
	if err != nil {
		return p, err
	}
	limit, _ := pageLimit(q.Limit)
	for i, id := range ids {
		if i == limit {
			p.Next = ids[i-1]
			break
		}
		b, err := s.Branch(ctx, id)
		if err != nil {
			return p, err
		}
		p.Branches = append(p.Branches, b)
	}
	return p, nil
}
func (s *Store) Referenced(ctx context.Context, r snapshot.Ref) (bool, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM snapshot_objects WHERE hash=?`, r.Hash).Scan(&n)
	return n > 0, failure("object references", err)
}
func (s *Store) RecoverSnapshots(ctx context.Context) error {
	for _, table := range []string{"snapshots", "branches"} {
		after := ""
		for {
			ids, err := s.snapshotIDs(ctx, table, snapshot.Query{After: after, Limit: 100})
			if err != nil {
				return err
			}
			if len(ids) > 100 {
				ids = ids[:100]
			}
			for _, id := range ids {
				after = id
				if table == "snapshots" {
					r, err := s.Snapshot(ctx, id)
					if err != nil {
						return err
					}
					if r.State == "preparing" {
						if err = s.SnapshotState(ctx, id, "failed", "interrupted_capture_objects_retained"); err != nil {
							return err
						}
					}
				} else {
					b, err := s.Branch(ctx, id)
					if err != nil {
						return err
					}
					if b.State == "preparing" {
						if err = s.FinishBranch(ctx, id, "failed", "interrupted_materialization_directory_retained"); err != nil {
							return err
						}
					}
				}
			}
			if len(ids) < 100 {
				break
			}
		}
	}
	return nil
}

var _ snapshot.Repository = (*Store)(nil)
