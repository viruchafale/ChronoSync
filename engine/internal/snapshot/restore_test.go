package snapshot

import (
	"chronosync.local/engine/internal/model"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRestoreRefusesExistingAndMalicious(t *testing.T) {
	path := t.TempDir()
	root, err := os.OpenRoot(path)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	root.Mkdir("managed-workspaces", 0700)
	id := model.ID()
	root.Mkdir("managed-workspaces/"+id, 0700)
	marker := filepath.Join(path, "managed-workspaces", id, "user-file")
	os.WriteFile(marker, []byte("preserved"), 0600)
	s := &Service{root: root, directory: path}
	m := Manifest{Version: 1, Policy: Policy, Excludes: []string{}, Entries: []Entry{}, Omitted: []Omission{}}
	if s.restore(context.Background(), id, m) == nil {
		t.Fatal("existing destination accepted")
	}
	b, _ := os.ReadFile(marker)
	if string(b) != "preserved" {
		t.Fatal("destination changed")
	}
	m.Entries = []Entry{{Path: "../escape", Kind: "directory", Mode: 0700}}
	if s.restore(context.Background(), model.ID(), m) == nil {
		t.Fatal("traversal accepted")
	}
	os.Rename(filepath.Join(path, "managed-workspaces"), filepath.Join(path, "retained"))
	os.Symlink(t.TempDir(), filepath.Join(path, "managed-workspaces"))
	m.Entries = nil
	if s.restore(context.Background(), model.ID(), m) == nil {
		t.Fatal("root symlink accepted")
	}
}
