package snapshot

import (
	"chronosync.local/engine/internal/workspace"
	"context"
	"io"
	"io/fs"
	"os"
)

type scanResult struct {
	manifest   Manifest
	identities map[string]os.FileInfo
}

func scan(ctx context.Context, source string, excludes []string, objects Objects) (scanResult, error) {
	result := scanResult{Manifest{Version: 1, Policy: Policy, Excludes: excludes, Entries: []Entry{}, Omitted: []Omission{}}, map[string]os.FileInfo{}}
	root, err := os.OpenRoot(source)
	if err != nil {
		return result, fail("CAPTURE_FAILED", "workspace unavailable")
	}
	defer root.Close()
	count := 0
	var total int64
	err = walkBounded(ctx, root, func(p string, d fs.DirEntry, walkErr error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if walkErr != nil {
			return fail("CAPTURE_FAILED", "entry unreadable")
		}
		if p == "." {
			return nil
		}
		count++
		if count > MaxEntries {
			return fail("LIMIT_EXCEEDED", "entry count")
		}
		if !ValidPath(p) {
			return fail("INVALID_PATH", "workspace contains unsupported path")
		}
		if workspace.Excluded(p, excludes) {
			result.manifest.Omitted = append(result.manifest.Omitted, Omission{p, "excluded"})
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		info, err := root.Lstat(p)
		if err != nil {
			return fail("WORKSPACE_CHANGED", "entry unavailable")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			result.manifest.Omitted = append(result.manifest.Omitted, Omission{p, "symlink"})
			return nil
		}
		result.identities[p] = info
		if info.IsDir() {
			result.manifest.Entries = append(result.manifest.Entries, Entry{Path: p, Kind: "directory", Mode: 0700})
			return nil
		}
		if !info.Mode().IsRegular() {
			return fail("UNSUPPORTED_FILE", "special file")
		}
		if info.Mode().Perm()&0400 == 0 {
			return fail("CAPTURE_FAILED", "file lacks owner read permission")
		}
		if info.Size() > MaxFile {
			return fail("LIMIT_EXCEEDED", "file size")
		}
		f, err := openRegular(root, p)
		if err != nil {
			return fail("CAPTURE_FAILED", "file open")
		}
		opened, err := f.Stat()
		if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
			f.Close()
			return fail("WORKSPACE_CHANGED", "file replaced before read")
		}
		data, err := io.ReadAll(io.LimitReader(f, MaxFile+1))
		after, statErr := f.Stat()
		f.Close()
		current, pathErr := root.Lstat(p)
		if err != nil || statErr != nil || pathErr != nil || !os.SameFile(info, current) || !sameMetadata(info, after) || !sameMetadata(info, current) || int64(len(data)) != info.Size() {
			return fail("WORKSPACE_CHANGED", "file changed while reading")
		}
		total += int64(len(data))
		if total > MaxTotal {
			return fail("LIMIT_EXCEEDED", "workspace byte limit")
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ref := Digest(data)
		if objects != nil {
			ref, err = objects.Put(ctx, data)
			if err != nil {
				return err
			}
		}
		mode := uint32(0600)
		if info.Mode().Perm()&0111 != 0 {
			mode = 0700
		}
		result.manifest.Entries = append(result.manifest.Entries, Entry{Path: p, Kind: "file", Mode: mode, Object: &ref})
		return nil
	})
	if err != nil {
		return result, err
	}
	Sort(&result.manifest)
	return result, result.manifest.Validate()
}
func sameMetadata(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Mode() == b.Mode() && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

// Bounded directory batches avoid WalkDir's whole-directory allocation. Paths
// are validated by visit before descending; directory handles never follow links.
func walkBounded(ctx context.Context, root *os.Root, visit func(string, fs.DirEntry, error) error) error {
	var walk func(string) error
	walk = func(path string) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		before, err := root.Lstat(path)
		if err != nil || !before.IsDir() {
			return fail("WORKSPACE_CHANGED", "directory replaced")
		}
		dir, err := openRegular(root, path)
		if err != nil {
			return fail("CAPTURE_FAILED", "directory open")
		}
		defer dir.Close()
		opened, err := dir.Stat()
		if err != nil || !os.SameFile(before, opened) {
			return fail("WORKSPACE_CHANGED", "directory identity")
		}
		for {
			entries, readErr := dir.ReadDir(100)
			if readErr != nil && readErr != io.EOF {
				return fail("CAPTURE_FAILED", "directory read")
			}
			for _, entry := range entries {
				p := entry.Name()
				if path != "." {
					p = path + "/" + p
				}
				err := visit(p, entry, nil)
				if err == fs.SkipDir {
					continue
				}
				if err != nil {
					return err
				}
				if entry.IsDir() {
					if err = walk(p); err != nil {
						return err
					}
				}
			}
			if readErr == io.EOF {
				break
			}
		}
		after, err := root.Lstat(path)
		if err != nil || !sameMetadata(before, after) {
			return fail("WORKSPACE_CHANGED", "directory changed during scan")
		}
		return nil
	}
	return walk(".")
}
