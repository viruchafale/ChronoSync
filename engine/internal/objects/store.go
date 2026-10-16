// Package objects implements immutable exact-byte SHA-256 objects; it never
// deletes finalized objects or links them into mutable workspaces.
package objects

import (
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/snapshot"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Store struct {
	root *os.Root
	gate chan struct{}
}

func Open(path string) (*Store, error) {
	if !filepath.IsAbs(path) {
		return nil, errors.New("object root must be absolute")
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, errors.New("object directory unavailable")
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("unsafe object directory")
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, err
	}
	return &Store{root, make(chan struct{}, 1)}, nil
}
func (s *Store) Close() error { return s.root.Close() }
func (s *Store) Get(ctx context.Context, r snapshot.Ref) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if !snapshot.ValidRef(r) {
		return nil, &snapshot.Error{Code: "INVALID_OBJECT", Operation: "object reference"}
	}
	info, err := s.root.Lstat(r.Hash)
	if err != nil || !info.Mode().IsRegular() || info.Size() != r.Size || info.Mode().Perm()&0077 != 0 {
		return nil, &snapshot.Error{Code: "DAMAGED_OBJECT", Operation: "missing or unsafe object"}
	}
	f, err := s.root.Open(r.Hash)
	if err != nil {
		return nil, &snapshot.Error{Code: "DAMAGED_OBJECT", Operation: "object unreadable"}
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, &snapshot.Error{Code: "DAMAGED_OBJECT", Operation: "object replaced"}
	}
	b, err := io.ReadAll(io.LimitReader(f, snapshot.MaxFile+1))
	if err != nil || snapshot.Digest(b) != r {
		return nil, &snapshot.Error{Code: "DAMAGED_OBJECT", Operation: "object integrity mismatch"}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return b, nil
}
func (s *Store) Put(ctx context.Context, b []byte) (snapshot.Ref, error) {
	r := snapshot.Digest(b)
	if !snapshot.ValidRef(r) {
		return r, &snapshot.Error{Code: "LIMIT_EXCEEDED", Operation: "object size"}
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	select {
	case s.gate <- struct{}{}:
		defer func() { <-s.gate }()
	case <-ctx.Done():
		return r, ctx.Err()
	}
	if _, err := s.root.Lstat(r.Hash); err == nil {
		_, err = s.Get(ctx, r)
		return r, err
	} else if !errors.Is(err, os.ErrNotExist) {
		return r, errors.New("object lookup failed")
	}
	temp := "tmp-" + model.ID()
	f, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return r, errors.New("object temporary write failed")
	}
	defer s.root.Remove(temp)
	if _, err = f.Write(b); err == nil {
		err = f.Chmod(0400)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return r, errors.New("object sync failed")
	}
	if ctx.Err() != nil {
		return r, ctx.Err()
	}
	if err = s.root.Link(temp, r.Hash); err != nil {
		if errors.Is(err, os.ErrExist) {
			_, err = s.Get(ctx, r)
		}
		return r, err
	}
	if err = s.root.Remove(temp); err != nil {
		return r, errors.New("object temporary cleanup failed")
	}
	dir, err := s.root.Open(".")
	if err != nil {
		return r, err
	}
	defer dir.Close()
	if err = dir.Sync(); err != nil {
		return r, errors.New("object directory sync failed")
	}
	return r, nil
}
func (s *Store) Inventory(ctx context.Context, referenced func(snapshot.Ref) (bool, error)) (snapshot.Inventory, error) {
	out := snapshot.Inventory{}
	dir, err := s.root.Open(".")
	if err != nil {
		return out, err
	}
	defer dir.Close()
	for {
		entries, err := dir.ReadDir(100)
		if err != nil && err != io.EOF {
			return out, err
		}
		for _, e := range entries {
			if ctx.Err() != nil {
				return out, ctx.Err()
			}
			info, x := e.Info()
			if x != nil {
				return out, x
			}
			if strings.HasPrefix(e.Name(), "tmp-") {
				out.Temporary++
				continue
			}
			r := snapshot.Ref{Hash: e.Name(), Size: info.Size()}
			if !snapshot.ValidRef(r) || !info.Mode().IsRegular() {
				return out, errors.New("unexpected object-store entry; preserved")
			}
			out.Objects++
			out.Bytes += r.Size
			used, x := referenced(r)
			if x != nil {
				return out, x
			}
			if !used {
				out.Unreferenced++
				out.UnreferencedBytes += r.Size
			}
		}
		if err == io.EOF {
			return out, nil
		}
	}
}

// OpenIn creates only descriptor-relative private child directories in the
// already owned application-data root. It refuses symlinked directory entries.
func OpenIn(base string) (*Store, error) {
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, err
	}
	for _, name := range []string{"objects", "v1", "sha256"} {
		if err = root.Mkdir(name, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			root.Close()
			return nil, err
		}
		info, err := root.Lstat(name)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			root.Close()
			return nil, errors.New("unsafe object-store directory")
		}
		dir, err := root.Open(".")
		if err != nil {
			root.Close()
			return nil, err
		}
		err = dir.Sync()
		dir.Close()
		if err != nil {
			root.Close()
			return nil, err
		}
		child, err := root.OpenRoot(name)
		root.Close()
		if err != nil {
			return nil, err
		}
		root = child
	}
	return &Store{root, make(chan struct{}, 1)}, nil
}
