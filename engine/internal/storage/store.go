// Package storage owns the private SQLite event repository and payload files.
package storage

import (
	"bytes"
	"chronosync.local/engine/internal/admission"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/storage/legacy"
	_ "modernc.org/sqlite"
)

const MaxEvents = 1000000
const inlineLimit = 8192

type Error struct {
	Code      string
	Operation string
}

func (e *Error) Error() string { return e.Code + ": " + e.Operation }
func failure(op string, err error) error {
	if err == nil {
		return nil
	}
	var typed *Error
	if errors.As(err, &typed) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return &Error{"NOT_FOUND", op}
	}
	return &Error{"STORAGE_ERROR", op}
}
func invalid(op string) error { return &Error{"INVALID_ARGUMENT", op} }

type Repository interface {
	Create(model.Run) error
	Save(model.Run) error
	Append(model.Event) error
	GetRun(context.Context, string) (model.Run, error)
}
type Inspector interface {
	List() ([]model.Run, error)
	GetRun(context.Context, string) (model.Run, error)
	QueryRuns(context.Context, RunQuery) (RunPage, error)
	QueryEvents(context.Context, EventQuery) (EventPage, error)
	GetEvent(context.Context, string, string) (model.Event, error)
}
type Store struct {
	root           *os.Root
	lease          *os.File
	db             *sql.DB
	writer         chan struct{}
	admissionMu    sync.RWMutex
	admissionEpoch admission.Epoch
	admissionReady bool
	admissionFault func(string) error // package-private deterministic test injection; nil in production
}
type History struct {
	Run      model.Run     `json:"run"`
	Events   []model.Event `json:"events"`
	Warnings []string      `json:"warnings"`
}

const migration = `
CREATE TABLE runs(id TEXT PRIMARY KEY CHECK(length(id)=32), created TEXT NOT NULL, state TEXT NOT NULL CHECK(state IN ('created','starting','running','completed','failed','cancelled','timed_out','interrupted')), last INTEGER NOT NULL CHECK(last>=0), metadata BLOB NOT NULL, integrity TEXT NOT NULL DEFAULT 'complete', diagnostic TEXT NOT NULL DEFAULT '');
CREATE TABLE events(id TEXT PRIMARY KEY CHECK(length(id)=32), run_id TEXT NOT NULL REFERENCES runs(id), sequence INTEGER NOT NULL CHECK(sequence>0), kind TEXT NOT NULL, observed TEXT NOT NULL, status TEXT NOT NULL, truncated INTEGER NOT NULL CHECK(truncated IN (0,1)), data BLOB, object TEXT, digest TEXT NOT NULL CHECK(length(digest)=64), size INTEGER NOT NULL CHECK(size>0 AND size<=262144), search TEXT NOT NULL, UNIQUE(run_id,sequence), CHECK((data IS NULL)!=(object IS NULL)));
CREATE INDEX events_kind_sequence ON events(run_id,kind,sequence);
CREATE INDEX runs_created ON runs(created DESC,id DESC);
CREATE TABLE legacy_imports(run_id TEXT PRIMARY KEY REFERENCES runs(id));
`

func (s *Store) migrate(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, checksum TEXT NOT NULL)`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT version,checksum FROM schema_migrations ORDER BY version`)
	if err != nil {
		return err
	}
	migrations := []string{migration, snapshotMigration, admissionMigration}
	applied := 0
	for rows.Next() {
		var v int
		var hash string
		if err = rows.Scan(&v, &hash); err != nil {
			rows.Close()
			return err
		}
		if v != applied+1 || v > len(migrations) {
			rows.Close()
			return &Error{"UNSUPPORTED_SCHEMA", "migration order or version"}
		}
		sum := sha256.Sum256([]byte(migrations[v-1]))
		if hash != hex.EncodeToString(sum[:]) {
			rows.Close()
			return &Error{"UNSUPPORTED_SCHEMA", "migration checksum"}
		}
		applied = v
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for i := applied; i < len(migrations); i++ {
		if _, err = tx.ExecContext(ctx, migrations[i]); err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(migrations[i]))
		if _, err = tx.ExecContext(ctx, `INSERT INTO schema_migrations VALUES(?,?)`, i+1, hex.EncodeToString(sum[:])); err != nil {
			return err
		}
	}

	return tx.Commit()
}
func Open(path string) (_ *Store, err error) {
	if !filepath.IsAbs(path) {
		return nil, invalid("store root must be absolute")
	}
	if err = os.MkdirAll(path, 0700); err != nil {
		return nil, failure("create store", err)
	}
	info, e := os.Lstat(path)
	if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, invalid("store root must be a private real directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, failure("open root", err)
	}
	s := &Store{root: root, writer: make(chan struct{}, 1)}
	defer func() {
		if err != nil {
			s.Close()
		}
	}()
	for _, name := range []string{".lock", "history.sqlite", "history.sqlite-wal", "history.sqlite-shm"} {
		if st, e := root.Lstat(name); e == nil && (!st.Mode().IsRegular() || st.Mode().Perm()&0077 != 0) {
			return nil, invalid("unsafe storage file")
		}
	}
	s.lease, err = root.OpenFile(".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, failure("open lease", err)
	}
	if err = lock(s.lease); err != nil {
		return nil, &Error{"STORE_BUSY", "another owner or locking unavailable"}
	}
	f, err := root.OpenFile("history.sqlite", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, failure("create database", err)
	}
	f.Close()
	if err = root.Mkdir("payloads", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, failure("create payload directory", err)
	}
	st, e := root.Lstat("payloads")
	if e != nil || !st.IsDir() || st.Mode()&os.ModeSymlink != 0 || st.Mode().Perm()&0077 != 0 {
		return nil, invalid("unsafe payload directory")
	}
	u := url.URL{Scheme: "file", Path: filepath.Join(path, "history.sqlite")}
	q := u.Query()
	for _, p := range []string{"foreign_keys(1)", "busy_timeout(1000)", "synchronous(FULL)", "trusted_schema(OFF)"} {
		q.Add("_pragma", p)
	}
	u.RawQuery = q.Encode()
	s.db, err = sql.Open("sqlite", u.String())
	if err != nil {
		return nil, failure("open database", err)
	}
	s.db.SetMaxOpenConns(4)
	s.db.SetMaxIdleConns(4)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var mode string
	if err = s.db.QueryRowContext(ctx, `PRAGMA journal_mode=WAL`).Scan(&mode); err != nil || mode != "wal" {
		return nil, &Error{"STORAGE_ERROR", "enable WAL"}
	}
	if err = s.migrate(ctx); err != nil {
		return nil, failure("migrate", err)
	}
	if err = s.importLegacy(context.Background()); err != nil {
		return nil, err
	}
	if err = s.RecoverAdmissions(ctx); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error {
	var err error
	if s.db != nil {
		err = s.db.Close()
	}
	if s.root != nil {
		s.root.Close()
	}
	if s.lease != nil {
		s.lease.Close()
	}
	return err
}
func (s *Store) write(ctx context.Context, fn func(*sql.Tx) error) error {
	select {
	case s.writer <- struct{}{}:
		defer func() { <-s.writer }()
	default:
		return &Error{"STORE_BUSY", "writer backpressure"}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return failure("begin transaction", err)
	}
	defer tx.Rollback()
	if err = fn(tx); err != nil {
		return failure("write transaction", err)
	}
	return failure("commit transaction", tx.Commit())
}
func (s *Store) Create(run model.Run) error {
	if !model.ValidID(run.ID) || run.SchemaVersion != 1 || run.State != model.Created {
		return invalid("new run")
	}
	run.LastSequence = "0"
	run.History = "complete"
	return s.write(context.Background(), func(tx *sql.Tx) error {
		b, _ := json.Marshal(run)
		_, err := tx.Exec(`INSERT INTO runs(id,created,state,last,metadata) VALUES(?,?,?,0,?)`, run.ID, sortTime(run.CreatedAt), run.State, b)
		return err
	})
}

// Save updates adapter metadata only. A failure annotation may interrupt an active
// run, but success always requires a committed execution.state event.
func (s *Store) Save(run model.Run) error {
	return s.write(context.Background(), func(tx *sql.Tx) error {
		current, err := getRun(context.Background(), tx, run.ID)
		if err != nil {
			return err
		}
		current.AdapterVersion = run.AdapterVersion
		if run.State == model.Interrupted && !current.State.Terminal() {
			current.State = model.Interrupted
			current.Reason = run.Reason
			current.FinishedAt = run.FinishedAt
			current.History = "partial"
			current.Recovered = true
		}
		return saveRun(tx, current)
	})
}
func saveRun(tx *sql.Tx, r model.Run) error {
	b, _ := json.Marshal(r)
	_, err := tx.Exec(`UPDATE runs SET metadata=?,state=?,last=?,integrity=? WHERE id=?`, b, r.State, r.LastSequence, r.History, r.ID)
	return err
}
func (s *Store) Append(e model.Event) error {
	return s.AppendBatch(context.Background(), []model.Event{e})
}

// AppendBatch is bounded; caller supplies strictly consecutive sequences. Concurrent
// callers receive STORE_BUSY or a sequence conflict and must retry explicitly.
func (s *Store) AppendBatch(ctx context.Context, events []model.Event) error {
	if len(events) == 0 || len(events) > 100 {
		return invalid("batch size 1..100")
	}
	total := 0
	for _, e := range events {
		if err := e.Validate(); err != nil {
			return invalid("event validation")
		}
		b, _ := json.Marshal(e)
		total += len(b)
	}
	if total > 2*1024*1024 {
		return invalid("batch byte limit")
	}
	return s.write(ctx, func(tx *sql.Tx) error {
		for _, e := range events {
			r, err := getRun(ctx, tx, e.RunID)
			if err != nil {
				return err
			}
			n, _ := strconv.ParseInt(e.Sequence, 10, 64)
			last, _ := strconv.ParseInt(r.LastSequence, 10, 64)
			if n != last+1 || n > MaxEvents || r.State.Terminal() {
				return &Error{"CONFLICT", "sequence or terminal run"}
			}
			if e.Kind == "execution.state" {
				if !(n == 1 && e.Payload.Status == model.Created) {
					if err = model.Transition(r.State, e.Payload.Status); err != nil {
						return &Error{"CONFLICT", "lifecycle transition"}
					}
				}
				r.State = e.Payload.Status
				r.Reason = e.Payload.Reason
				r.ExitCode = e.Payload.ExitCode
				if r.State == model.Running {
					r.StartedAt = e.ObservedAt
				}
				if r.State.Terminal() {
					r.FinishedAt = e.ObservedAt
				}
			}
			if err = s.insert(tx, e); err != nil {
				return err
			}
			r.LastSequence = e.Sequence
			if err = saveRun(tx, r); err != nil {
				return err
			}
		}
		return nil
	})
}
func (s *Store) insert(tx *sql.Tx, e model.Event) error {
	b, _ := json.Marshal(e)
	sum := sha256.Sum256(b)
	hash := hex.EncodeToString(sum[:])
	var data any = b
	var object any
	if len(b) > inlineLimit {
		name := "payloads/" + hash + ".json"
		if err := s.publish(name, b); err != nil {
			return err
		}
		data = nil
		object = hash
	}
	// Index only the first 1024 bytes of sanitized diagnostic text. Outputs and
	// commands remain detail-only and are deliberately not duplicated in search.
	search := e.Payload.Message + " " + e.Payload.Reason + " " + e.Payload.UpstreamType
	if len(search) > 1024 {
		search = search[:1024]
	}
	truncated := strings.Contains(string(b), "[TRUNCATED]") || len(e.Payload.Truncations) > 0
	_, err := tx.Exec(`INSERT INTO events(id,run_id,sequence,kind,observed,status,truncated,data,object,digest,size,search) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, e.EventID, e.RunID, e.Sequence, e.Kind, e.ObservedAt, e.Payload.Status, truncated, data, object, hash, len(b), search)
	return err
}
func (s *Store) publish(name string, b []byte) error {
	// Hash names are generated here, never accepted as filesystem paths from callers.
	if info, err := s.root.Lstat(name); err == nil {
		if !info.Mode().IsRegular() {
			return &Error{"CORRUPT_PAYLOAD", "unsafe existing object; preserved"}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if existing, err := s.root.Open(name); err == nil {
		raw, e := io.ReadAll(io.LimitReader(existing, model.MaxEventBytes+1))
		existing.Close()
		if e != nil || string(raw) != string(b) {
			return &Error{"CORRUPT_PAYLOAD", "existing object differs"}
		}
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	temp := "payloads/" + model.ID() + ".tmp"
	f, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer s.root.Remove(temp)
	if _, err = f.Write(b); err == nil {
		err = f.Sync()
	}
	ce := f.Close()
	if err == nil {
		err = ce
	}
	if err != nil {
		return err
	}
	if err = s.root.Rename(temp, name); err != nil {
		return err
	}
	dir, err := s.root.Open("payloads")
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

type rowQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getRun(ctx context.Context, db rowQuery, id string) (model.Run, error) {
	var r model.Run
	if !model.ValidID(id) {
		return r, invalid("run id")
	}
	var raw []byte
	var state, history, diagnostic string
	var last int64
	err := db.QueryRowContext(ctx, `SELECT metadata,state,last,integrity,diagnostic FROM runs WHERE id=?`, id).Scan(&raw, &state, &last, &history, &diagnostic)
	if err != nil {
		return r, failure("get run", err)
	}
	if len(raw) > 65536 || json.Unmarshal(raw, &r) != nil || r.ID != id || r.SchemaVersion != 1 || string(r.State) != state || r.LastSequence != strconv.FormatInt(last, 10) {
		return r, &Error{"CORRUPT_METADATA", "run metadata"}
	}
	r.History = history
	r.Diagnostic = diagnostic
	if history == "damaged" && r.State == model.Completed {
		r.State = model.Interrupted
		r.Reason = "history_damaged"
	}
	return r, nil
}
func (s *Store) GetRun(ctx context.Context, id string) (model.Run, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return getRun(ctx, s.db, id)
}
func (s *Store) Read(id string) (History, error) {
	h := History{Events: []model.Event{}, Warnings: []string{}}
	var err error
	h.Run, err = s.GetRun(context.Background(), id)
	if err != nil {
		return h, err
	}
	if h.Run.History != "complete" {
		h.Warnings = append(h.Warnings, h.Run.History+": "+h.Run.Diagnostic)
	}
	var cursor string
	for {
		p, e := s.QueryEvents(context.Background(), EventQuery{ID: id, Cursor: cursor, Limit: 100})
		if e != nil {
			return h, e
		}
		for _, row := range p.Events {
			event, e := s.GetEvent(context.Background(), id, row.EventID)
			if e != nil {
				h.Warnings = append(h.Warnings, e.Error())
				continue
			}
			h.Events = append(h.Events, event)
			if len(h.Events) > 10000 {
				return h, invalid("use paginated queries for large histories")
			}
		}
		if p.Next == "" {
			break
		}
		cursor = p.Next
	}
	return h, nil
}
func (s *Store) List() ([]model.Run, error) {
	p, err := s.QueryRuns(context.Background(), RunQuery{Limit: 100})
	return p.Runs, err
}
func (s *Store) importLegacy(ctx context.Context) error {
	dir, err := s.root.Open(".")
	if err != nil {
		return failure("legacy inventory", err)
	}
	defer dir.Close()
	reader := legacy.Inspect(s.root)
	for {
		entries, e := dir.ReadDir(100)
		if e != nil && e != io.EOF {
			return failure("legacy inventory", e)
		}
		for _, entry := range entries {
			if !entry.IsDir() || !model.ValidID(entry.Name()) {
				continue
			}
			var count int
			if err = s.db.QueryRowContext(ctx, `SELECT count(*) FROM runs WHERE id=?`, entry.Name()).Scan(&count); err != nil {
				return failure("import lookup", err)
			}
			if count > 0 {
				continue
			}
			h, readErr := reader.Read(entry.Name())
			r := h.Run
			if readErr != nil {
				r = model.Run{SchemaVersion: 1, ID: entry.Name(), State: model.Interrupted, LastSequence: "0", Reason: "legacy_unreadable"}
				h.Events = nil
				h.Warnings = []string{"Legacy metadata or journal unreadable; source preserved"}
			}
			if r.State != model.Created && r.State != model.Starting && r.State != model.Running && !r.State.Terminal() {
				r.State = model.Interrupted
				r.Reason = "legacy_invalid_state"
				h.Warnings = append(h.Warnings, "Invalid legacy state; source preserved")
			}
			r.History = "complete"
			if len(h.Warnings) > 0 {
				r.History = "damaged"
			}
			diagnostic := strings.Join(h.Warnings, "; ")
			if len(diagnostic) > 8192 {
				diagnostic = diagnostic[:8192]
			}
			r.Diagnostic = diagnostic
			if err = s.write(ctx, func(tx *sql.Tx) error {
				if r.LastSequence == "" {
					r.LastSequence = "0"
				}
				raw, _ := json.Marshal(r)
				_, e := tx.Exec(`INSERT INTO runs(id,created,state,last,metadata,integrity,diagnostic) VALUES(?,?,?,?,?,?,?)`, r.ID, sortTime(r.CreatedAt), r.State, r.LastSequence, raw, r.History, diagnostic)
				if e != nil {
					return e
				}
				for _, event := range h.Events {
					var exists int
					if e = tx.QueryRow(`SELECT count(*) FROM events WHERE id=?`, event.EventID).Scan(&exists); e != nil {
						return e
					}
					if exists > 0 {
						r.History = "damaged"
						r.Diagnostic = "Legacy event identity conflict; source preserved"
						continue
					}
					if e = s.insert(tx, event); e != nil {
						return e
					}
				}
				if e = saveRun(tx, r); e != nil {
					return e
				}
				if _, e = tx.Exec(`UPDATE runs SET diagnostic=? WHERE id=?`, r.Diagnostic, r.ID); e != nil {
					return e
				}
				_, e = tx.Exec(`INSERT INTO legacy_imports VALUES(?)`, r.ID)
				return e
			}); err != nil {
				return err
			}
		}
		if e == io.EOF {
			return nil
		}
	}
}

// Recover is a startup operation, before exposing the recorder. It never appends
// invented lifecycle events or launches/kills a process from persisted metadata.
func (s *Store) Recover() error {
	ctx := context.Background()
	var result string
	if err := s.db.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&result); err != nil || result != "ok" {
		return &Error{"CORRUPT_DATABASE", "quick_check failed; evidence preserved"}
	}
	rows, err := s.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return failure("relationship check", err)
	}
	bad := rows.Next()
	err = rows.Err()
	rows.Close()
	if bad || err != nil {
		return &Error{"CORRUPT_DATABASE", "relationship check failed"}
	}
	// Verify aggregate relationships without inventing absent records.
	if _, err = s.db.Exec(`UPDATE runs SET integrity='damaged',diagnostic='Sequence or lifecycle relationship mismatch; evidence preserved' WHERE last != COALESCE((SELECT max(sequence) FROM events WHERE run_id=runs.id),0) OR last != (SELECT count(*) FROM events WHERE run_id=runs.id) OR (state='completed' AND COALESCE((SELECT status FROM events WHERE run_id=runs.id AND kind='execution.state' ORDER BY sequence DESC LIMIT 1),'') != 'completed')`); err != nil {
		return failure("relationship validation", err)
	}
	// Stream IDs; each payload is read and released separately (no history array).
	rows, err = s.db.QueryContext(ctx, `SELECT id,run_id FROM events ORDER BY run_id,sequence`)
	if err != nil {
		return failure("integrity scan", err)
	}
	for rows.Next() {
		var id, run string
		if err = rows.Scan(&id, &run); err != nil {
			break
		}
		if _, e := s.GetEvent(ctx, run, id); e != nil {
			_, err = s.db.Exec(`UPDATE runs SET integrity='damaged',diagnostic='Payload validation failed; inspect event details' WHERE id=?`, run)
			if err != nil {
				break
			}
		}
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		return failure("integrity scan", err)
	}
	for {
		processed := 0
		err = s.write(ctx, func(tx *sql.Tx) error {
			rows, err := tx.Query(`SELECT id FROM runs WHERE state IN ('created','starting','running') LIMIT 100`)
			if err != nil {
				return err
			}
			ids := []string{}
			for rows.Next() {
				var id string
				if err = rows.Scan(&id); err != nil {
					rows.Close()
					return err
				}
				ids = append(ids, id)
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			processed = len(ids)
			for _, id := range ids {
				r, e := getRun(ctx, tx, id)
				if e != nil {
					return e
				}
				r.State = model.Interrupted
				r.Reason = "engine_restarted_process_cleanup_unverified"
				r.FinishedAt = model.Now()
				r.Recovered = true
				if r.History == "complete" {
					r.History = "partial"
				}
				if e = saveRun(tx, r); e != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
		if processed == 0 {
			return nil
		}
	}
}
func (s *Store) GetEvent(ctx context.Context, run, id string) (model.Event, error) {
	var e model.Event
	if !model.ValidID(id) || !model.ValidID(run) {
		return e, invalid("event identity")
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var b []byte
	var object sql.NullString
	var hash string
	var size int
	var seq, kind, observed, status string
	var truncated bool
	err := s.db.QueryRowContext(ctx, `SELECT data,object,digest,size,CAST(sequence AS TEXT),kind,observed,status,truncated FROM events WHERE id=? AND run_id=?`, id, run).Scan(&b, &object, &hash, &size, &seq, &kind, &observed, &status, &truncated)
	if err != nil {
		return e, failure("get event", err)
	}
	corrupt := func() (model.Event, error) {
		_, _ = s.db.ExecContext(ctx, `UPDATE runs SET integrity='damaged',diagnostic='Payload validation failed; inspect event details' WHERE id=?`, run)
		return model.Event{}, &Error{"CORRUPT_PAYLOAD", "event payload unavailable or invalid"}
	}
	if size < 1 || size > model.MaxEventBytes {
		return corrupt()
	}
	if object.Valid {
		decoded, err := hex.DecodeString(object.String)
		if err != nil || len(decoded) != 32 || hex.EncodeToString(decoded) != object.String || object.String != hash {
			return corrupt()
		}
		f, err := s.root.Open("payloads/" + object.String + ".json")
		if err != nil {
			return corrupt()
		}
		b, err = io.ReadAll(io.LimitReader(f, model.MaxEventBytes+1))
		f.Close()
		if err != nil {
			return corrupt()
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(b))
	decoder.DisallowUnknownFields()
	decodeErr := decoder.Decode(&e)
	var extra any
	if decodeErr == nil && decoder.Decode(&extra) != io.EOF {
		decodeErr = errors.New("extra data")
	}
	digest := sha256.Sum256(b)
	if len(b) != size || hex.EncodeToString(digest[:]) != hash || decodeErr != nil || e.Validate() != nil || e.RunID != run || e.EventID != id || e.Sequence != seq || e.Kind != kind || e.ObservedAt != observed || string(e.Payload.Status) != status || truncated != (strings.Contains(string(b), "[TRUNCATED]") || len(e.Payload.Truncations) > 0) {
		return corrupt()
	}
	return e, nil
}

var _ Repository = (*Store)(nil)
