// Package workspace observes metadata; it does not sandbox child processes.
package workspace

import (
	"chronosync.local/engine/internal/model"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Snapshot struct {
	Files   map[string]model.FileMeta
	Skipped int
}

var excluded = map[string]bool{".git": true, ".codex": true, ".aws": true, ".ssh": true, "node_modules": true, "vendor": true, "target": true, "dist": true, "build": true, ".cache": true, ".next": true, "coverage": true, "__pycache__": true, ".venv": true, "venv": true, ".chronosync": true}

func Validate(path string, extra []string) (string, error) {
	if !filepath.IsAbs(path) {
		return "", errors.New("workspace must be absolute")
	}
	root, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", errors.New("workspace cannot be resolved")
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() || root == string(filepath.Separator) {
		return "", errors.New("select a project directory, not a filesystem root")
	}
	if len(extra) > 100 {
		return "", errors.New("too many exclusions")
	}
	for _, p := range extra {
		if !fs.ValidPath(p) || p == "." {
			return "", errors.New("exclusions must be relative file/directory paths")
		}
	}
	return root, nil
}
func Scan(ctx context.Context, path string, extra []string) (Snapshot, error) {
	result := Snapshot{Files: map[string]model.FileMeta{}}
	root, err := os.OpenRoot(path)
	if err != nil {
		return result, err
	}
	defer root.Close()
	var count int
	var total int64
	err = fs.WalkDir(root.FS(), ".", func(p string, d fs.DirEntry, e error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e != nil {
			return errors.New("workspace scan could not read an entry")
		}
		if p == "." {
			return nil
		}
		count++
		if count > 20000 {
			return errors.New("workspace entry budget exceeded")
		}
		skip := Excluded(p, extra)
		if skip {
			result.Skipped++
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			result.Skipped++
			return nil
		}
		if d.IsDir() {
			return nil
		}
		before, e := root.Lstat(p)
		if e != nil {
			return errors.New("workspace changed during inspection")
		}
		if !before.Mode().IsRegular() || before.Size() > 1024*1024 {
			result.Skipped++
			return nil
		}
		f, e := root.Open(p)
		if e != nil {
			return errors.New("workspace file could not be safely opened")
		}
		defer f.Close()
		opened, e := f.Stat()
		if e != nil || !os.SameFile(before, opened) {
			return errors.New("workspace entry changed during open")
		}
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(f, 1024*1024+1))
		if e != nil {
			return errors.New("workspace read failed")
		}
		after, e := f.Stat()
		_ = f.Close()
		if e != nil || n > 1024*1024 || n != before.Size() || !after.ModTime().Equal(before.ModTime()) {
			return errors.New("workspace file changed during hash")
		}
		total += n
		if total > 64*1024*1024 {
			return errors.New("workspace byte budget exceeded")
		}
		result.Files[p] = model.FileMeta{Size: n, SHA256: hex.EncodeToString(h.Sum(nil)), Mode: uint32(before.Mode().Perm())}
		return nil
	})
	return result, err
}
func Diff(a, b Snapshot) []model.Change {
	changes := []model.Change{}
	for p, old := range a.Files {
		now, ok := b.Files[p]
		if !ok {
			o := old
			changes = append(changes, model.Change{Path: p, Operation: "deleted", Before: &o})
		} else if old != now {
			o, n := old, now
			changes = append(changes, model.Change{Path: p, Operation: "modified", Before: &o, After: &n})
		}
	}
	for p, now := range b.Files {
		if _, ok := a.Files[p]; !ok {
			n := now
			changes = append(changes, model.Change{Path: p, Operation: "created", After: &n})
		}
	}
	sort.Slice(changes, func(i, j int) bool { return changes[i].Path < changes[j].Path })
	return changes
}

// Excluded is the shared version-one observation/capture filename policy.
func Excluded(p string, extra []string) bool {
	name := filepath.Base(p)
	skip := excluded[name] || strings.HasPrefix(name, ".env") || strings.HasSuffix(name, ".pem") || strings.HasSuffix(name, ".key") || name == "credentials" || name == "auth.json"
	for _, x := range extra {
		if p == x || strings.HasPrefix(p, x+"/") {
			skip = true
		}
	}
	return skip
}
