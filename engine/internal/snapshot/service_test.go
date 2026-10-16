package snapshot_test

import (
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/objects"
	"chronosync.local/engine/internal/snapshot"
	"chronosync.local/engine/internal/storage"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

type fixture struct {
	s            *snapshot.Service
	db           *storage.Store
	obj          *objects.Store
	run          model.Run
	source, root string
}

func setup(t *testing.T) fixture {
	t.Helper()
	source := t.TempDir()
	root := filepath.Join(t.TempDir(), "store")
	db, err := storage.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	obj, err := objects.Open(filepath.Join(root, "objects", "v1", "sha256"))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := snapshot.New(db, obj, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { svc.Close(); obj.Close(); db.Close() })
	run := model.Run{ID: model.ID(), SchemaVersion: 1, Workspace: source, CreatedAt: model.Now(), State: model.Created}
	if err = db.Create(run); err != nil {
		t.Fatal(err)
	}
	if err = db.Append(model.Event{SchemaVersion: 2, EventID: model.ID(), RunID: run.ID, Sequence: "1", Kind: "execution.state", ObservedAt: model.Now(), Adapter: "codex", Payload: model.Payload{Status: model.Completed}}); err != nil {
		t.Fatal(err)
	}
	return fixture{svc, db, obj, run, source, root}
}
func write(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}
func capture(t *testing.T, f fixture, branch string) snapshot.Record {
	t.Helper()
	r, err := f.s.Capture(context.Background(), snapshot.CaptureSpec{RunID: f.run.ID, BranchID: branch, Consent: true, Excludes: []string{"omit"}})
	if err != nil {
		t.Fatal(r, err)
	}
	return r
}
func TestCaptureRestoreBranchIsolation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	os.Mkdir(filepath.Join(f.source, "empty"), 0700)
	write(t, filepath.Join(f.source, "file"), "one", 0600)
	write(t, filepath.Join(f.source, "duplicate"), "one", 0600)
	write(t, filepath.Join(f.source, "exec"), "#!/bin/sh\nexit 99", 0700)
	write(t, filepath.Join(f.source, "zero"), "", 0600)
	write(t, filepath.Join(f.source, ".env"), "synthetic secret", 0600)
	write(t, filepath.Join(f.source, "omit"), "excluded", 0600)
	os.Symlink("/outside", filepath.Join(f.source, "link"))
	os.Symlink("cycle", filepath.Join(f.source, "cycle"))
	os.Symlink("missing", filepath.Join(f.source, "broken"))
	original := capture(t, f, "")
	if original.State != "committed" || original.Omitted != 5 {
		t.Fatal(original)
	}
	_, m, err := f.s.Validate(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Entries) != 5 {
		t.Fatal(m)
	}
	inv, err := f.s.Inventory(ctx)
	if err != nil || inv.Objects != 4 || inv.Unreferenced != 0 {
		t.Fatal(inv, err)
	}
	restored, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: original.ID, Consent: true}, "restore")
	if err != nil || restored.State != "ready" {
		t.Fatal(restored, err)
	}
	raw, _ := os.ReadFile(filepath.Join(restored.Workspace, "file"))
	if string(raw) != "one" {
		t.Fatal("restore bytes")
	}
	info, _ := os.Stat(filepath.Join(restored.Workspace, "exec"))
	if info.Mode().Perm() != 0700 {
		t.Fatal("exec mode")
	}
	if _, err = os.Stat(filepath.Join(restored.Workspace, ".env")); !os.IsNotExist(err) {
		t.Fatal("excluded restored")
	}
	branch, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: original.ID, Consent: true}, "branch")
	if err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(branch.Workspace, "file"), "changed", 0600)
	next := capture(t, f, branch.ID)
	if next.BranchID != branch.ID {
		t.Fatal(next)
	}
	child, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: next.ID, ParentID: branch.ID, Consent: true}, "branch")
	if err != nil || child.ParentID != branch.ID {
		t.Fatal(child, err)
	}
	raw, _ = os.ReadFile(filepath.Join(f.source, "file"))
	if string(raw) != "one" {
		t.Fatal("source modified")
	}
	_, old, err := f.s.Validate(ctx, original.ID)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.Bytes()
	b, _ := old.Bytes()
	if string(a) != string(b) {
		t.Fatal("ancestor mutated")
	}
	events, err := f.db.QueryEvents(ctx, storage.EventQuery{ID: f.run.ID})
	if err != nil || events.Count != 1 {
		t.Fatal("original history changed")
	}
}

type changingObjects struct {
	snapshot.Objects
	path string
	once sync.Once
}

func (c *changingObjects) Put(ctx context.Context, b []byte) (snapshot.Ref, error) {
	r, e := c.Objects.Put(ctx, b)
	c.once.Do(func() { os.WriteFile(c.path, []byte("new content"), 0600) })
	return r, e
}
func TestChangedAndInvalidSources(t *testing.T) {
	f := setup(t)
	path := filepath.Join(f.source, "file")
	write(t, path, "old", 0600)
	old := capture(t, f, "")
	f.s.Objects = &changingObjects{Objects: f.obj, path: path}
	if r, err := f.s.Capture(context.Background(), snapshot.CaptureSpec{RunID: f.run.ID, Consent: true}); err == nil || r.State != "failed" {
		t.Fatal("changing source committed", r, err)
	}
	if _, _, err := f.s.Validate(context.Background(), old.ID); err != nil {
		t.Fatal(err)
	}
	f.s.Objects = f.obj
	write(t, filepath.Join(f.source, "FILE"), "collision", 0600) // case-insensitive filesystems may alias, tested by manifest policy separately
	os.Remove(filepath.Join(f.source, "FILE"))
	write(t, filepath.Join(f.source, "unicode-é"), "unsupported", 0600)
	if _, err := f.s.Capture(context.Background(), snapshot.CaptureSpec{RunID: f.run.ID, Consent: true}); err == nil {
		t.Fatal("Unicode source accepted")
	}
	os.Remove(filepath.Join(f.source, "unicode-é"))
	write(t, filepath.Join(f.source, "unreadable"), "x", 0000)
	if _, err := f.s.Capture(context.Background(), snapshot.CaptureSpec{RunID: f.run.ID, Consent: true}); err == nil {
		t.Fatal("unreadable accepted")
	}
	os.Remove(filepath.Join(f.source, "unreadable"))
	write(t, filepath.Join(f.source, "large"), strings.Repeat("x", snapshot.MaxFile+1), 0600)
	if _, err := f.s.Capture(context.Background(), snapshot.CaptureSpec{RunID: f.run.ID, Consent: true}); err == nil {
		t.Fatal("large accepted")
	}
}
func TestRecoveryMissingAndCancelled(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	write(t, filepath.Join(f.source, "file"), "fixture", 0600)
	r := capture(t, f, "")
	preparing := snapshot.Record{ID: model.ID(), RunID: f.run.ID, State: "preparing", Workspace: f.source, CreatedAt: model.Now()}
	if err := f.db.PrepareSnapshot(ctx, preparing); err != nil {
		t.Fatal(err)
	}
	b := snapshot.Branch{ID: model.ID(), SnapshotID: r.ID, RunID: f.run.ID, State: "preparing", Kind: "branch"}
	if err := f.db.PrepareBranch(ctx, b); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := f.s.Recover(ctx); err != nil {
			t.Fatal(err)
		}
	}
	got, _ := f.db.Snapshot(ctx, preparing.ID)
	branch, _ := f.db.Branch(ctx, b.ID)
	if got.State != "failed" || branch.State != "failed" {
		t.Fatal(got, branch)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err := f.s.Validate(cancelled, r.ID); err == nil {
		t.Fatal("cancel ignored")
	}
	got, _ = f.db.Snapshot(ctx, r.ID)
	if got.State != "committed" {
		t.Fatal("cancellation marked damage")
	}
	os.Remove(filepath.Join(f.root, "objects", "v1", "sha256", r.Manifest.Hash))
	if _, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: r.ID, Consent: true}, "branch"); err == nil {
		t.Fatal("missing manifest accepted")
	}
	got, _ = f.db.Snapshot(ctx, r.ID)
	if got.State != "damaged" {
		t.Fatal(got)
	}
}
func TestConcurrentBranchAdmission(t *testing.T) {
	f := setup(t)
	write(t, filepath.Join(f.source, "file"), "data", 0600)
	r := capture(t, f, "")
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, e := f.s.Materialize(context.Background(), snapshot.MaterializeSpec{SnapshotID: r.ID, Consent: true}, "branch")
			if e != nil {
				typed, ok := e.(*snapshot.Error)
				if !ok || typed.Code != "BUSY" {
					t.Error(e)
				}
			} else if b.State != "ready" {
				t.Error(b)
			}
		}()
	}
	wg.Wait()
	if _, _, err := f.s.Validate(context.Background(), r.ID); err != nil {
		t.Fatal(err)
	}
}
func TestSnapshotPerformance(t *testing.T) {
	if os.Getenv("CHRONOSYNC_SNAPSHOT_LOAD") != "1" {
		t.Skip("opt-in synthetic workload")
	}
	for _, workload := range []struct{ count, size, groups int }{{10, 1024, 10}, {1000, 1024, 10}, {16, 128 * 1024, 16}} {
		count := workload.count
		t.Run(fmt.Sprintf("%d-%d", count, workload.size), func(t *testing.T) {
			f := setup(t)
			for i := 0; i < count; i++ {
				write(t, filepath.Join(f.source, fmt.Sprintf("file-%04d", i)), strings.Repeat(fmt.Sprintf("group-%02d", i%workload.groups), workload.size/8), 0600)
			}
			ctx := context.Background()
			start := time.Now()
			r := capture(t, f, "")
			captureTime := time.Since(start)
			start = time.Now()
			restored, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: r.ID, Consent: true}, "restore")
			if err != nil {
				t.Fatal(err)
			}
			restoreTime := time.Since(start)
			start = time.Now()
			_, err = f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: r.ID, Consent: true}, "branch")
			if err != nil {
				t.Fatal(err)
			}
			branchTime := time.Since(start)
			start = time.Now()
			if _, _, err = f.s.Validate(ctx, r.ID); err != nil {
				t.Fatal(err)
			}
			verifyTime := time.Since(start)
			inv, err := f.s.Inventory(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if restored.Workspace == f.source {
				t.Fatal("source destination")
			}
			t.Logf("%s/%s %s files=%d logical_bytes=%d objects=%d object_bytes=%d manifest_bytes=%d capture=%s restore=%s branch=%s verify=%s", runtime.GOOS, runtime.GOARCH, runtime.Version(), count, r.Bytes, inv.Objects, inv.Bytes, r.Manifest.Size, captureTime, restoreTime, branchTime, verifyTime)
		})
	}
}

func TestInterruptedCaptureProcess(t *testing.T) {
	if root := os.Getenv("CHRONOSYNC_SNAPSHOT_CRASH"); root != "" {
		db, err := storage.Open(root)
		if err != nil {
			t.Fatal(err)
		}
		run := model.Run{ID: strings.Repeat("a", 32), SchemaVersion: 1, State: model.Created, CreatedAt: model.Now()}
		if err = db.Create(run); err != nil {
			t.Fatal(err)
		}
		record := snapshot.Record{ID: strings.Repeat("b", 32), RunID: run.ID, State: "preparing", CreatedAt: model.Now()}
		if err = db.PrepareSnapshot(context.Background(), record); err != nil {
			t.Fatal(err)
		}
		obj, err := objects.OpenIn(root)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = obj.Put(context.Background(), []byte("published but unreferenced")); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(root, "objects", "v1", "sha256", "tmp-interrupted"), []byte("partial"), 0600)
		os.Exit(0)
	}
	root := filepath.Join(t.TempDir(), "store")
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, exe, "-test.run=^TestInterruptedCaptureProcess$")
	command.Env = append(os.Environ(), "CHRONOSYNC_SNAPSHOT_CRASH="+root)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("%v %s", err, output)
	}
	db, err := storage.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	obj, err := objects.OpenIn(root)
	if err != nil {
		t.Fatal(err)
	}
	defer obj.Close()
	svc, err := snapshot.New(db, obj, root)
	if err != nil {
		t.Fatal(err)
	}
	defer svc.Close()
	if err = svc.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	r, err := db.Snapshot(ctx, strings.Repeat("b", 32))
	if err != nil || r.State != "failed" {
		t.Fatal(r, err)
	}
	inv, err := svc.Inventory(ctx)
	if err != nil || inv.Unreferenced != 1 || inv.Temporary != 1 {
		t.Fatal(inv, err)
	}
}

type failingReads struct {
	snapshot.Objects
	count, failAt int
}

func (f *failingReads) Get(ctx context.Context, r snapshot.Ref) ([]byte, error) {
	f.count++
	if f.count == f.failAt {
		return nil, &snapshot.Error{Code: "DAMAGED_OBJECT", Operation: "synthetic read failure"}
	}
	return f.Objects.Get(ctx, r)
}
func TestFailedMaterializationRetainsPartialDirectory(t *testing.T) {
	f := setup(t)
	write(t, filepath.Join(f.source, "a"), "a", 0600)
	write(t, filepath.Join(f.source, "b"), "b", 0600)
	r := capture(t, f, "")
	f.s.Objects = &failingReads{Objects: f.obj, failAt: 5}
	b, err := f.s.Materialize(context.Background(), snapshot.MaterializeSpec{SnapshotID: r.ID, Consent: true}, "branch")
	if err == nil || b.State != "failed" {
		t.Fatal(b, err)
	}
	saved, e := f.db.Branch(context.Background(), b.ID)
	if e != nil || saved.State != "failed" {
		t.Fatal(saved, e)
	}
	data, e := os.ReadFile(filepath.Join(b.Workspace, "a"))
	if e != nil || string(data) != "a" {
		t.Fatal("partial evidence removed", e)
	}
	f.s.Objects = f.obj
	if _, _, e = f.s.Validate(context.Background(), r.ID); e != nil {
		t.Fatal("source damaged", e)
	}
}

func TestConsentSourceAndParentBoundaries(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	if _, err := f.s.Capture(ctx, snapshot.CaptureSpec{RunID: f.run.ID}); err == nil {
		t.Fatal("consent bypass")
	}
	r := capture(t, f, "")
	if _, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: r.ID}, "branch"); err == nil {
		t.Fatal("restore consent bypass")
	}
	b, err := f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: r.ID, Consent: true}, "branch")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Materialize(ctx, snapshot.MaterializeSpec{SnapshotID: r.ID, ParentID: b.ID, Consent: true}, "branch"); err == nil {
		t.Fatal("unrelated parent accepted")
	}
	active := model.Run{ID: model.ID(), SchemaVersion: 1, State: model.Created, Workspace: f.source, CreatedAt: model.Now()}
	if err = f.db.Create(active); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Capture(ctx, snapshot.CaptureSpec{RunID: active.ID, Consent: true}); err == nil {
		t.Fatal("active run captured")
	}
}
