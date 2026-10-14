package workspace

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, data string) {
	t.Helper()
	if e := os.WriteFile(filepath.Join(dir, name), []byte(data), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestDiffExclusionsAndSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	write(t, outside, "secret", "outside")
	write(t, dir, "modified", "before")
	write(t, dir, "deleted", "before")
	write(t, dir, ".env", "secret")
	write(t, dir, "private", "secret")
	if e := os.Mkdir(filepath.Join(dir, "node_modules"), 0700); e != nil {
		t.Fatal(e)
	}
	write(t, filepath.Join(dir, "node_modules"), "dep", "ignored")
	if e := os.Symlink(outside, filepath.Join(dir, "escape")); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(filepath.Join(outside, "secret"), filepath.Join(dir, "link")); e != nil {
		t.Fatal(e)
	}
	before, e := Scan(context.Background(), dir, []string{"private"})
	if e != nil {
		t.Fatal(e)
	}
	if len(before.Files) != 2 {
		t.Fatalf("read excluded files: %+v", before)
	}
	write(t, dir, "modified", "after")
	write(t, dir, "created", "new")
	if e = os.Remove(filepath.Join(dir, "deleted")); e != nil {
		t.Fatal(e)
	}
	after, e := Scan(context.Background(), dir, []string{"private"})
	if e != nil {
		t.Fatal(e)
	}
	diff := Diff(before, after)
	if len(diff) != 3 || diff[0].Operation != "created" || diff[1].Operation != "deleted" || diff[2].Operation != "modified" {
		t.Fatalf("%+v", diff)
	}
}
func TestInvalidAndCancelled(t *testing.T) {
	for _, p := range []string{"../outside", "/absolute", "."} {
		if _, e := Validate(t.TempDir(), []string{p}); e == nil {
			t.Fatal(p)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := Scan(ctx, t.TempDir(), nil); e == nil {
		t.Fatal("cancel ignored")
	}
}
