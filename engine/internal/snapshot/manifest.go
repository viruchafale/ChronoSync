// Package snapshot defines portable manifests and snapshot orchestration ports.
package snapshot

import (
	"bytes"
	"chronosync.local/engine/internal/workspace"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"path"
	"sort"
	"strings"
)

const MaxFile = 8 * 1024 * 1024
const MaxTotal = 64 * 1024 * 1024
const MaxEntries = 10000
const MaxManifest = 2 * 1024 * 1024
const Policy = "portable-ascii-v1"

type Error struct {
	Code      string
	Operation string
}

func (e *Error) Error() string   { return e.Code + ": " + e.Operation }
func fail(code, op string) error { return &Error{code, op} }

type Ref struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

func ValidRef(r Ref) bool {
	b, e := hex.DecodeString(r.Hash)
	return e == nil && len(b) == 32 && hex.EncodeToString(b) == r.Hash && r.Size >= 0 && r.Size <= MaxFile
}
func Digest(b []byte) Ref {
	sum := sha256.Sum256(b)
	return Ref{hex.EncodeToString(sum[:]), int64(len(b))}
}

type Entry struct {
	Path   string `json:"path"`
	Kind   string `json:"kind"`
	Mode   uint32 `json:"mode"`
	Object *Ref   `json:"object,omitempty"`
}
type Omission struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}
type Manifest struct {
	Version  int        `json:"version"`
	Policy   string     `json:"policy"`
	Excludes []string   `json:"excludes"`
	Entries  []Entry    `json:"entries"`
	Omitted  []Omission `json:"omitted"`
}

func ValidPath(p string) bool {
	if p == "" || strings.Count(p, "/") >= 64 || len(p) > 1024 || strings.HasPrefix(p, "/") {
		return false
	}
	for _, c := range []byte(p) {
		if c < 32 || c > 126 || strings.ContainsRune(`\:*?"<>|`, rune(c)) {
			return false
		}
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." || len(part) > 240 || strings.HasSuffix(part, ".") || strings.HasSuffix(part, " ") {
			return false
		}
		base := strings.ToUpper(strings.Split(part, ".")[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '0' && base[3] <= '9' {
			return false
		}
	}
	return true
}
func (m Manifest) Validate() error {
	bad := func() error { return fail("INVALID_MANIFEST", "unsupported or ambiguous manifest") }
	if m.Version != 1 || m.Policy != Policy || len(m.Entries)+len(m.Omitted) > MaxEntries || len(m.Excludes) > 101 {
		return bad()
	}
	seen := map[string]string{}
	kinds := map[string]string{}
	var total int64
	last := ""
	for _, x := range m.Excludes {
		if !ValidPath(x) {
			return bad()
		}
	}
	check := func(p, kind string) bool {
		if !ValidPath(p) || seen[strings.ToLower(p)] != "" {
			return false
		}
		parent := path.Dir(p)
		if parent != "." && kinds[parent] != "directory" {
			return false
		}
		seen[strings.ToLower(p)] = p
		kinds[p] = kind
		return true
	}
	for _, e := range m.Entries {
		if e.Path <= last || workspace.Excluded(e.Path, m.Excludes) || !check(e.Path, e.Kind) {
			return bad()
		}
		last = e.Path
		switch e.Kind {
		case "directory":
			if e.Mode != 0700 || e.Object != nil {
				return bad()
			}
		case "file":
			if e.Mode != 0600 && e.Mode != 0700 || e.Object == nil || !ValidRef(*e.Object) {
				return bad()
			}
			total += e.Object.Size
		default:
			return bad()
		}
	}
	last = ""
	for _, e := range m.Omitted {
		if e.Path <= last || !check(e.Path, "omitted") || (e.Reason != "excluded" && e.Reason != "symlink") {
			return bad()
		}
		last = e.Path
	}
	if total > MaxTotal {
		return bad()
	}
	b, _ := json.Marshal(m)
	if len(b) > MaxManifest {
		return bad()
	}
	return nil
}
func (m Manifest) Bytes() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}
func Decode(b []byte) (Manifest, error) {
	var m Manifest
	if len(b) > MaxManifest {
		return m, fail("INVALID_MANIFEST", "manifest too large")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if d.Decode(&m) != nil {
		return m, fail("INVALID_MANIFEST", "manifest JSON")
	}
	var extra any
	if !errors.Is(d.Decode(&extra), io.EOF) {
		return m, fail("INVALID_MANIFEST", "trailing data")
	}
	return m, m.Validate()
}
func Sort(m *Manifest) {
	sort.Slice(m.Entries, func(i, j int) bool { return m.Entries[i].Path < m.Entries[j].Path })
	sort.Slice(m.Omitted, func(i, j int) bool { return m.Omitted[i].Path < m.Omitted[j].Path })
	sort.Strings(m.Excludes)
}
