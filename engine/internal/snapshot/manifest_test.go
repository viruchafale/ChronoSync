package snapshot

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func sample() Manifest {
	r := Digest([]byte("fixture"))
	return Manifest{Version: 1, Policy: Policy, Excludes: []string{}, Entries: []Entry{{Path: "dir", Kind: "directory", Mode: 0700}, {Path: "dir/file", Kind: "file", Mode: 0600, Object: &r}}, Omitted: []Omission{}}
}
func TestManifestRoundTrip(t *testing.T) {
	m := sample()
	b, e := m.Bytes()
	if e != nil {
		t.Fatal(e)
	}
	got, e := Decode(b)
	if e != nil || len(got.Entries) != 2 {
		t.Fatal(got, e)
	}
}
func TestPathPolicy(t *testing.T) {
	for _, p := range []string{"/a", "../a", "a/../b", "a//b", "a/./b", "a\\b", "C:/a", "a\x00b", "a.", "a ", "CON", "con.txt", "LPT1", "a/é", "a/é", strings.Repeat("a", 241)} {
		if ValidPath(p) {
			t.Fatal("accepted", p)
		}
	}
	for _, p := range []string{"file.txt", "dir/.hidden", "a-b_c/README"} {
		if !ValidPath(p) {
			t.Fatal("rejected", p)
		}
	}
}
func TestManifestRejects(t *testing.T) {
	tests := []func(*Manifest){func(m *Manifest) { m.Version = 2 }, func(m *Manifest) { m.Entries[1].Path = "../file" }, func(m *Manifest) { m.Entries[1].Path = "dir" }, func(m *Manifest) { m.Entries[1].Path = "DIR" }, func(m *Manifest) { m.Entries[1].Path = "DIR/file" }, func(m *Manifest) { m.Entries[1].Object.Hash = "bad" }, func(m *Manifest) { m.Entries[1].Object.Size = -1 }, func(m *Manifest) { m.Entries[1].Mode = 0777 }, func(m *Manifest) { m.Entries[1].Kind = "symlink" }, func(m *Manifest) { m.Entries[1].Object.Size = MaxFile + 1 }, func(m *Manifest) { m.Entries = make([]Entry, MaxEntries+1) }, func(m *Manifest) { m.Excludes = []string{"../escape"} }, func(m *Manifest) { m.Omitted = []Omission{{Path: "dir/file", Reason: "symlink"}} }}
	for i, change := range tests {
		m := sample()
		change(&m)
		if m.Validate() == nil {
			t.Fatal("invalid accepted", i)
		}
	}
	m := sample()
	raw, _ := json.Marshal(m)
	raw = append(raw, []byte("{}")...)
	if _, e := Decode(raw); e == nil {
		t.Fatal("trailing JSON")
	}
}

func TestAggregateAndExcludedEntries(t *testing.T) {
	m := sample()
	m.Entries = nil
	for i := 0; i < 9; i++ {
		r := Digest(nil)
		r.Size = MaxFile
		m.Entries = append(m.Entries, Entry{Path: fmt.Sprintf("file%d", i), Kind: "file", Mode: 0600, Object: &r})
	}
	if m.Validate() == nil {
		t.Fatal("aggregate bytes accepted")
	}
	m = sample()
	m.Entries[0].Path = ".git"
	m.Entries[1].Path = ".git/config"
	if m.Validate() == nil {
		t.Fatal("excluded metadata accepted")
	}
	if ValidPath(strings.Repeat("a/", 64) + "file") {
		t.Fatal("deep path accepted")
	}
}
