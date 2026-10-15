package storage

import (
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/storage/legacy"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func fixture(t *testing.T) (*Store, model.Run, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private")
	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { s.Close() })
	r := model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created, CreatedAt: model.Now()}
	if e = s.Create(r); e != nil {
		t.Fatal(e)
	}
	return s, r, dir
}
func event(r model.Run, n int) model.Event {
	return model.Event{SchemaVersion: 2, EventID: model.ID(), RunID: r.ID, Sequence: strconv.Itoa(n), Kind: "agent.message", ObservedAt: model.Now(), Adapter: "codex", Payload: model.Payload{Message: "synthetic searchable diagnostic"}}
}
func TestQueriesRestartRecovery(t *testing.T) {
	s, r, dir := fixture(t)
	ctx := context.Background()
	batch := []model.Event{}
	for n := 1; n <= 100; n++ {
		batch = append(batch, event(r, n))
	}
	if e := s.AppendBatch(ctx, batch); e != nil {
		t.Fatal(e)
	}
	p, e := s.QueryEvents(ctx, EventQuery{ID: r.ID, Limit: 17})
	if e != nil || len(p.Events) != 17 || p.Count != 100 {
		t.Fatalf("%+v %v", p, e)
	}
	if e = s.Append(event(r, 101)); e != nil {
		t.Fatal(e)
	}
	got := len(p.Events)
	cursor := p.Next
	for cursor != "" {
		p, e = s.QueryEvents(ctx, EventQuery{ID: r.ID, Cursor: cursor, Limit: 17})
		if e != nil {
			t.Fatal(e)
		}
		for _, row := range p.Events {
			got++
			if row.Sequence != strconv.Itoa(got) {
				t.Fatal("unstable ordering")
			}
		}
		cursor = p.Next
	}
	if got != 100 {
		t.Fatal(got)
	}
	p, e = s.QueryEvents(ctx, EventQuery{ID: r.ID, Min: "20", Max: "25", Kind: "agent.message", Search: "SEARCHABLE"})
	if e != nil || len(p.Events) != 6 {
		t.Fatal(p, e)
	}
	for _, q := range []EventQuery{{ID: r.ID, Limit: 101}, {ID: r.ID, Cursor: "garbage"}, {ID: r.ID, Kind: "unknown"}, {ID: r.ID, Min: "03"}, {ID: r.ID, Min: "5", Max: "2"}, {ID: "../../etc"}} {
		if _, e = s.QueryEvents(ctx, q); e == nil {
			t.Fatal("invalid query accepted", q)
		}
	}
	p, _ = s.QueryEvents(ctx, EventQuery{ID: r.ID, Limit: 1})
	if _, e = s.QueryEvents(ctx, EventQuery{ID: r.ID, Cursor: p.Next, Search: "different"}); e == nil {
		t.Fatal("cursor filter mismatch")
	}
	if _, e = s.GetRun(ctx, model.ID()); e == nil {
		t.Fatal("unknown accepted")
	}
	s.Close()
	s, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	for range 2 {
		if e = s.Recover(); e != nil {
			t.Fatal(e)
		}
	}
	recovered, e := s.GetRun(ctx, r.ID)
	if e != nil || recovered.State != model.Interrupted || !recovered.Recovered || recovered.LastSequence != "101" {
		t.Fatal(recovered, e)
	}
}
func TestAtomicLifecycleAndRollback(t *testing.T) {
	s, r, _ := fixture(t)
	ctx := context.Background()
	e := event(r, 1)
	e.Kind = "execution.state"
	e.Payload = model.Payload{Status: model.Starting}
	if err := s.Append(e); err != nil {
		t.Fatal(err)
	}
	second := event(r, 2)
	third := event(r, 4)
	if s.AppendBatch(ctx, []model.Event{second, third}) == nil {
		t.Fatal("gap accepted")
	}
	p, _ := s.QueryEvents(ctx, EventQuery{ID: r.ID})
	if len(p.Events) != 1 || p.Run.LastSequence != "1" || p.Run.State != model.Starting {
		t.Fatal(p)
	}
	r.State = model.Completed
	r.LastSequence = "999"
	if err := s.Save(r); err != nil {
		t.Fatal(err)
	}
	saved, _ := s.GetRun(ctx, r.ID)
	if saved.State != model.Starting || saved.LastSequence != "1" {
		t.Fatal("metadata fabricated success")
	}
	second.Kind = "execution.state"
	second.Payload = model.Payload{Status: model.Completed}
	if err := s.Append(second); err != nil {
		t.Fatal(err)
	}
	saved, _ = s.GetRun(ctx, r.ID)
	if saved.State != model.Completed || saved.FinishedAt != second.ObservedAt {
		t.Fatal(saved)
	}
	if s.Append(event(r, 3)) == nil {
		t.Fatal("terminal append accepted")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if s.AppendBatch(ctx, []model.Event{event(r, 3)}) == nil {
		t.Fatal("cancelled write accepted")
	}
	if _, err := s.db.Exec(`INSERT INTO legacy_imports VALUES(?)`, model.ID()); err == nil {
		t.Fatal("foreign key disabled")
	}
}
func TestPayloadIntegrityPermissionsAndMigrations(t *testing.T) {
	s, r, dir := fixture(t)
	e := event(r, 1)
	e.Payload.Output = strings.Repeat("x", 10000)
	if err := s.Append(e); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetEvent(context.Background(), r.ID, e.EventID)
	if err != nil || got.Payload.Output != e.Payload.Output {
		t.Fatal(err)
	}
	var ref string
	if err = s.db.QueryRow(`SELECT object FROM events WHERE id=?`, e.EventID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"history.sqlite", "history.sqlite-wal", "history.sqlite-shm", "payloads/" + ref + ".json"} {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.Mode().Perm()&0077 != 0 {
			t.Fatal("unsafe mode", name, err)
		}
	}
	if _, err = Open(dir); err == nil {
		t.Fatal("second owner")
	}
	if err = os.Remove(filepath.Join(dir, "payloads", ref+".json")); err != nil {
		t.Fatal(err)
	}
	if err = s.Recover(); err != nil {
		t.Fatal(err)
	}
	run, _ := s.GetRun(context.Background(), r.ID)
	if run.History != "damaged" || run.State != model.Interrupted {
		t.Fatal(run)
	}
	if _, err = s.GetEvent(context.Background(), r.ID, e.EventID); err == nil {
		t.Fatal("missing payload fabricated")
	}
	if _, err = s.db.Exec(`UPDATE schema_migrations SET checksum='modified'`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if other, err := Open(dir); err == nil {
		other.Close()
		t.Fatal("modified migration accepted")
	}
}
func TestLegacyImportIdempotentPreservesSource(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "private")
	old, err := legacy.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created, CreatedAt: model.Now()}
	if err = old.Create(r); err != nil {
		t.Fatal(err)
	}
	e := event(r, 1)
	if err = old.Append(e); err != nil {
		t.Fatal(err)
	}
	old.Close()
	file := filepath.Join(dir, r.ID, "events.jsonl")
	raw, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	raw = append(raw, []byte("torn")...)
	if err = os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		s, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		p, err := s.QueryEvents(context.Background(), EventQuery{ID: r.ID})
		if err != nil || len(p.Events) != 1 || p.Run.History != "damaged" {
			t.Fatal(p, err)
		}
		s.Close()
	}
	after, _ := os.ReadFile(file)
	if string(after) != string(raw) {
		t.Fatal("legacy evidence modified")
	}
}
func TestConcurrentProducersReaders(t *testing.T) {
	started := time.Now()
	var retries atomic.Int64
	s, r, _ := fixture(t)
	var wg sync.WaitGroup
	ctx := context.Background()
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 25 {
				for {
					run, err := s.GetRun(ctx, r.ID)
					if err != nil {
						t.Error(err)
						return
					}
					n, _ := strconv.Atoi(run.LastSequence)
					err = s.Append(event(r, n+1))
					if err == nil {
						break
					}
					typed, ok := err.(*Error)
					if !ok || (typed.Code != "CONFLICT" && typed.Code != "STORE_BUSY") {
						t.Error(err)
						return
					}
					retries.Add(1)
					time.Sleep(time.Millisecond)
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 50 {
			if _, err := s.QueryEvents(ctx, EventQuery{ID: r.ID, Limit: 10}); err != nil {
				t.Error(err)
			}
		}
	}()
	wg.Wait()
	p, err := s.QueryEvents(ctx, EventQuery{ID: r.ID})
	t.Logf("4 producers x25, concurrent reader x50: %s, retries=%d", time.Since(started), retries.Load())
	if err != nil || p.Count != 100 || len(p.Events) != 100 {
		t.Fatal(p, err)
	}
}

// Opt-in reproducible workload; timings are observations, not CI speed assertions.
func TestLoad100k(t *testing.T) {
	if os.Getenv("CHRONOSYNC_LOAD") != "1" {
		t.Skip("set CHRONOSYNC_LOAD=1 for 100k-event measurement")
	}
	s, r, dir := fixture(t)
	ctx := context.Background()
	start := time.Now()
	var at10k int64
	single := []time.Duration{}
	for n := 1; n <= 100; n++ {
		before := time.Now()
		if err := s.Append(event(r, n)); err != nil {
			t.Fatal(err)
		}
		single = append(single, time.Since(before))
	}
	sort.Slice(single, func(i, j int) bool { return single[i] < single[j] })
	for first := 101; first <= 100000; first += 100 {
		batch := make([]model.Event, 100)
		for i := range batch {
			batch[i] = event(r, first+i)
		}
		if err := s.AppendBatch(ctx, batch); err != nil {
			t.Fatal(err)
		}
		if first == 9901 {
			s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
			st, _ := os.Stat(filepath.Join(dir, "history.sqlite"))
			at10k = st.Size()
		}
	}
	insert := time.Since(start)
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start = time.Now()
	p, err := s.QueryEvents(ctx, EventQuery{ID: r.ID, Limit: 100})
	query := time.Since(start)
	runtime.ReadMemStats(&after)
	if err != nil || len(p.Events) != 100 || p.Count != 100000 {
		t.Fatal(p, err)
	}
	start = time.Now()
	if err = s.Recover(); err != nil {
		t.Fatal(err)
	}
	recovery := time.Since(start)
	s.db.Exec(`PRAGMA wal_checkpoint(TRUNCATE)`)
	st, _ := os.Stat(filepath.Join(dir, "history.sqlite"))
	var version string
	s.db.QueryRow(`SELECT sqlite_version()`).Scan(&version)
	report := map[string]any{"single_p50_us": single[49].Microseconds(), "single_p95_us": single[94].Microseconds(), "go": runtime.Version(), "os": runtime.GOOS, "arch": runtime.GOARCH, "sqlite": version, "events": 100000, "batch": 100, "insert_ms": insert.Milliseconds(), "query_us": query.Microseconds(), "page_alloc_bytes": after.TotalAlloc - before.TotalAlloc, "db_10k_bytes": at10k, "db_100k_bytes": st.Size(), "recovery_ms": recovery.Milliseconds()}
	b, _ := json.Marshal(report)
	fmt.Println(string(b))
}

func TestAbruptExitRecovery(t *testing.T) {
	if dir := os.Getenv("CHRONOSYNC_CRASH_STORE"); dir != "" {
		s, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		r := model.Run{SchemaVersion: 1, ID: strings.Repeat("a", 32), State: model.Created, CreatedAt: model.Now()}
		if err = s.Create(r); err != nil {
			t.Fatal(err)
		}
		if err = s.Append(event(r, 1)); err != nil {
			t.Fatal(err)
		}
		tx, err := s.db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		if err = s.insert(tx, event(r, 2)); err != nil {
			t.Fatal(err)
		}
		os.Exit(0) // deliberate synthetic crash: no rollback or Close
	}
	dir := filepath.Join(t.TempDir(), "crash")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, exe, "-test.run=^TestAbruptExitRecovery$")
	command.Env = append(os.Environ(), "CHRONOSYNC_CRASH_STORE="+dir)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("crash helper: %v %s", err, output)
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err = s.Recover(); err != nil {
		t.Fatal(err)
	}
	p, err := s.QueryEvents(ctx, EventQuery{ID: strings.Repeat("a", 32)})
	if err != nil || len(p.Events) != 1 || p.Run.State != model.Interrupted || p.Run.LastSequence != "1" {
		t.Fatal(p, err)
	}
}
func TestFailedSQLWriteAndFutureSchema(t *testing.T) {
	s, r, dir := fixture(t)
	if _, err := s.db.Exec(`CREATE TRIGGER fail_event BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT,'synthetic failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if s.Append(event(r, 1)) == nil {
		t.Fatal("failed SQL write accepted")
	}
	got, err := s.GetRun(context.Background(), r.ID)
	if err != nil || got.LastSequence != "0" {
		t.Fatal(got, err)
	}
	if _, err = s.db.Exec(`INSERT INTO schema_migrations VALUES(999,'future')`); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if opened, err := Open(dir); err == nil {
		opened.Close()
		t.Fatal("future schema accepted")
	}
}
func TestRunPaginationAndSearchInjection(t *testing.T) {
	s, r, _ := fixture(t)
	for range 4 {
		next := r
		next.ID = model.ID()
		if err := s.Create(next); err != nil {
			t.Fatal(err)
		}
	}
	cursor := ""
	seen := map[string]bool{}
	for {
		p, err := s.QueryRuns(context.Background(), RunQuery{Cursor: cursor, Limit: 2})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range p.Runs {
			if seen[r.ID] {
				t.Fatal("duplicate run")
			}
			seen[r.ID] = true
		}
		cursor = p.Next
		if cursor == "" {
			break
		}
	}
	if len(seen) != 5 {
		t.Fatal(seen)
	}
	if err := s.Append(event(r, 1)); err != nil {
		t.Fatal(err)
	}
	p, err := s.QueryEvents(context.Background(), EventQuery{ID: r.ID, Search: "' OR 1=1 --"})
	if err != nil || len(p.Events) != 0 {
		t.Fatal("search is not literal", err)
	}
	e := event(r, 2)
	e.Payload.Output = strings.Repeat("x", model.MaxEventBytes)
	if s.Append(e) == nil {
		t.Fatal("oversize accepted")
	}
}

func TestExistingUnsafePayloadPreserved(t *testing.T) {
	s, r, dir := fixture(t)
	e := event(r, 1)
	e.Payload.Output = strings.Repeat("x", 10000)
	raw, _ := json.Marshal(e)
	hash := sha256.Sum256(raw)
	name := filepath.Join(dir, "payloads", hex.EncodeToString(hash[:])+".json")
	if err := os.Symlink("missing-evidence", name); err != nil {
		t.Fatal(err)
	}
	if s.Append(e) == nil {
		t.Fatal("unsafe object overwritten")
	}
	target, err := os.Readlink(name)
	if err != nil || target != "missing-evidence" {
		t.Fatal("evidence changed", err)
	}
	saved, err := s.GetRun(context.Background(), r.ID)
	if err != nil || saved.LastSequence != "0" {
		t.Fatal(saved, err)
	}
}
