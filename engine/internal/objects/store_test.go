package objects

import (
	"chronosync.local/engine/internal/snapshot"
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestObjects(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "objects")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	ctx := context.Background()
	data := []byte("synthetic bytes")
	ref := snapshot.Digest(data)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, e := s.Put(ctx, data)
			if e != nil || r != ref {
				t.Error(r, e)
			}
		}()
	}
	wg.Wait()
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatal("no dedup", len(entries))
	}
	got, err := s.Get(ctx, ref)
	if err != nil || string(got) != string(data) {
		t.Fatal(err)
	}
	if _, err = s.Get(ctx, snapshot.Ref{Hash: "../escape", Size: 1}); err == nil {
		t.Fatal("traversal")
	}
	if err = os.Chmod(filepath.Join(dir, ref.Hash), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, ref.Hash), []byte("corruption"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Put(ctx, data); err == nil {
		t.Fatal("corrupt object overwritten")
	}
	if _, err = s.Get(ctx, ref); err == nil {
		t.Fatal("corrupt accepted")
	}
	os.Remove(filepath.Join(dir, ref.Hash))
	if _, err = s.Get(ctx, ref); err == nil {
		t.Fatal("missing accepted")
	}
	os.WriteFile(filepath.Join(dir, "tmp-synthetic"), []byte("incomplete"), 0600)
	inv, err := s.Inventory(ctx, func(snapshot.Ref) (bool, error) { return false, nil })
	if err != nil || inv.Temporary != 1 {
		t.Fatal(inv, err)
	}
}
