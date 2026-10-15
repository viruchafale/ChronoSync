// Package storage implements the bounded Phase 1 staging journal, not SQLite.
package legacy

import (
	"bufio"
	"chronosync.local/engine/internal/model"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"
)

const MaxEvents = 10000
const MaxRunBytes = 32 * 1024 * 1024

type Store struct {
	root  *os.Root
	lease *os.File
	mu    sync.Mutex
}
type record struct {
	Event model.Event `json:"event"`
	Hash  string      `json:"sha256"`
}
type History struct {
	Run      model.Run     `json:"run"`
	Events   []model.Event `json:"events"`
	Warnings []string      `json:"warnings"`
}

func Open(path string) (*Store, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("store root must be absolute")
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, errors.New("store root must be a real directory")
	}
	if info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("store root must have private 0700 permissions")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	f, err := root.OpenFile(".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		root.Close()
		return nil, err
	}
	if err = lock(f); err != nil {
		f.Close()
		root.Close()
		return nil, errors.New("store already owned by another engine or locking unavailable")
	}
	s := &Store{root: root, lease: f}
	return s, nil
}
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.root.Close()
	return s.lease.Close()
}
func (s *Store) save(run model.Run) error {
	if !model.ValidID(run.ID) {
		return errors.New("invalid run id")
	}
	b, err := json.Marshal(run)
	if err != nil {
		return err
	}
	name := run.ID + "/run.tmp"
	f, err := s.root.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = f.Write(b)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err = s.root.Rename(name, run.ID+"/run.json"); err != nil {
		return err
	}
	d, err := s.root.Open(run.ID)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
func (s *Store) Create(run model.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !model.ValidID(run.ID) {
		return errors.New("invalid run id")
	}
	if err := s.root.Mkdir(run.ID, 0700); err != nil {
		return err
	}
	f, err := s.root.OpenFile(run.ID+"/events.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_ = f.Close()
	return s.save(run)
}
func (s *Store) Save(run model.Run) error { s.mu.Lock(); defer s.mu.Unlock(); return s.save(run) }
func (s *Store) Append(e model.Event) error {
	if err := e.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(e)
	sum := sha256.Sum256(b)
	frame, _ := json.Marshal(record{e, hex.EncodeToString(sum[:])})
	f, err := s.root.OpenFile(e.RunID+"/events.jsonl", os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size()+int64(len(frame))+1 > MaxRunBytes {
		return errors.New("run storage budget exceeded")
	}
	if _, err = f.Write(append(frame, '\n')); err != nil {
		return err
	}
	return f.Sync()
}
func (s *Store) read(id string) (History, error) {
	h := History{Events: []model.Event{}, Warnings: []string{}}
	warn := func(message string) {
		if len(h.Warnings) < 100 {
			h.Warnings = append(h.Warnings, message)
		} else if len(h.Warnings) == 100 {
			h.Warnings = append(h.Warnings, "Additional journal warnings omitted")
		}
	}
	if !model.ValidID(id) {
		return h, errors.New("invalid run id")
	}
	f, err := s.root.Open(id + "/run.json")
	if err != nil {
		return h, err
	}
	b, err := io.ReadAll(io.LimitReader(f, 65537))
	_ = f.Close()
	if err != nil || len(b) > 65536 {
		return h, errors.New("invalid run metadata")
	}
	if json.Unmarshal(b, &h.Run) != nil || h.Run.SchemaVersion != 1 || h.Run.ID != id {
		return h, errors.New("unsupported or invalid run metadata")
	}
	saved := h.Run
	f, err = s.root.Open(id + "/events.jsonl")
	if err != nil {
		return h, err
	}
	defer f.Close()
	reader := bufio.NewReader(io.LimitReader(f, MaxRunBytes+1))
	var total int
	var frames int
	var last int64
	seen := map[string]bool{}
	for {
		line, e := reader.ReadBytes('\n')
		frames++
		if frames > MaxEvents*2 {
			warn("Journal frame count limit exceeded")
			break
		}
		total += len(line)
		if total > MaxRunBytes {
			return h, errors.New("journal size limit exceeded")
		}
		if len(line) == 0 && e == io.EOF {
			break
		}
		if e == io.EOF {
			warn("Incomplete final journal record ignored")
			break
		}
		if e != nil {
			return h, e
		}
		var frame record
		if len(line) > model.MaxEventBytes+1024 || json.Unmarshal(line, &frame) != nil || frame.Event.Validate() != nil {
			warn("Invalid journal record skipped")
			continue
		}
		raw, _ := json.Marshal(frame.Event)
		sum := sha256.Sum256(raw)
		if frame.Hash != hex.EncodeToString(sum[:]) {
			warn("Journal integrity mismatch; record skipped")
			continue
		}
		seq, _ := strconv.ParseInt(frame.Event.Sequence, 10, 64)
		if frame.Event.RunID != id || seq <= last || seen[frame.Event.EventID] {
			warn("Invalid journal order or identity; record skipped")
			continue
		}
		if seq != last+1 {
			warn("Journal sequence gap")
		}
		seen[frame.Event.EventID] = true
		last = seq
		h.Events = append(h.Events, frame.Event)
		if len(h.Events) > MaxEvents {
			return h, errors.New("event count limit exceeded")
		}
		if frame.Event.Kind == "execution.state" {
			h.Run.State = frame.Event.Payload.Status
			h.Run.Reason = frame.Event.Payload.Reason
			h.Run.ExitCode = frame.Event.Payload.ExitCode
			if h.Run.State.Terminal() {
				h.Run.FinishedAt = frame.Event.ObservedAt
			}
		}
	}
	RecoveryState(&h, saved)
	h.Run.LastSequence = strconv.FormatInt(last, 10)
	if len(h.Warnings) > 0 && h.Run.State == model.Completed {
		h.Run.State = model.Interrupted
		h.Run.Reason = "history_incomplete"
	}
	return h, nil
}
func (s *Store) Read(id string) (History, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.read(id) }
func (s *Store) List() ([]model.Run, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := s.root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(-1)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	runs := []model.Run{}
	for _, entry := range entries {
		if !entry.IsDir() || !model.ValidID(entry.Name()) {
			continue
		}
		h, e := s.read(entry.Name())
		if e != nil {
			// An incomplete/corrupt run must not prevent inspection of unrelated runs.
			// Keep the damaged files untouched; detail retrieval returns the actual error.
			runs = append(runs, model.Run{SchemaVersion: 1, ID: entry.Name(), State: model.Interrupted, Reason: "metadata_or_journal_unreadable", LastSequence: "0"})
			continue
		}
		runs = append(runs, h.Run)
	}
	sort.Slice(runs, func(i, j int) bool { return runs[i].CreatedAt > runs[j].CreatedAt })
	return runs, nil
}
func (s *Store) Recover() error {
	runs, err := s.List()
	if err != nil {
		return err
	}
	for _, run := range runs {
		if !run.State.Terminal() {
			run.State = model.Interrupted
			run.Reason = "engine_restarted_process_cleanup_unverified"
			run.FinishedAt = model.Now()
			// Do not append after a possibly torn line. Metadata is authoritative for this recovery state.
			if err = s.Save(run); err != nil {
				return err
			}
		}
	}
	return nil
}

// Keep recovery metadata authoritative over nonterminal journal entries.
func RecoveryState(h *History, saved model.Run) {
	if saved.State == model.Interrupted && !h.Run.State.Terminal() {
		h.Run.State = saved.State
		h.Run.Reason = saved.Reason
		h.Run.FinishedAt = saved.FinishedAt
	}
}

// Inspect borrows the already locked root during the one-time Phase 2 import.
// The caller owns root and must not call Close on the returned reader.
func Inspect(root *os.Root) *Store { return &Store{root: root} }
