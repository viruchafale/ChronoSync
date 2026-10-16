package snapshot

import (
	"chronosync.local/engine/internal/model"
	"context"
	"os"
)

func (s *Service) restore(ctx context.Context, id string, m Manifest) error {
	if !model.ValidID(id) {
		return fail("INVALID_ARGUMENT", "materialization id")
	}
	if err := m.Validate(); err != nil {
		return err
	}
	info, err := s.root.Lstat("managed-workspaces")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fail("UNSAFE_DESTINATION", "managed root changed")
	}
	managed, err := s.root.OpenRoot("managed-workspaces")
	if err != nil {
		return fail("UNSAFE_DESTINATION", "managed root unavailable")
	}
	defer managed.Close()
	if err = managed.Mkdir(id, 0700); err != nil {
		return fail("DESTINATION_EXISTS", "new destination required")
	}
	target, err := managed.OpenRoot(id)
	if err != nil {
		return fail("RESTORE_FAILED", "destination open")
	}
	defer target.Close()
	for _, e := range m.Entries {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if e.Kind == "directory" {
			if err = target.Mkdir(e.Path, 0700); err != nil {
				return fail("RESTORE_FAILED", "directory creation")
			}
			continue
		}
		data, err := s.Objects.Get(ctx, *e.Object)
		if err != nil {
			return err
		}
		f, err := target.OpenFile(e.Path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(e.Mode))
		if err != nil {
			return fail("RESTORE_FAILED", "file creation")
		}
		if _, err = f.Write(data); err == nil {
			err = f.Chmod(os.FileMode(e.Mode))
		}
		if err == nil {
			err = f.Sync()
		}
		ce := f.Close()
		if err == nil {
			err = ce
		}
		if err != nil {
			return fail("RESTORE_FAILED", "file sync")
		}
	}
	// Verify exact materialized entries independently before publishing ready.
	restored, err := scan(ctx, s.destination(id), m.Excludes, nil)
	if err != nil {
		return err
	}
	if len(restored.manifest.Entries) != len(m.Entries) || len(restored.manifest.Omitted) != 0 {
		return fail("RESTORE_FAILED", "verification entry mismatch")
	}
	for i, e := range m.Entries {
		other := restored.manifest.Entries[i]
		if e.Path != other.Path || e.Kind != other.Kind || e.Mode != other.Mode || (e.Object != nil && (other.Object == nil || *e.Object != *other.Object)) {
			return fail("RESTORE_FAILED", "verification mismatch")
		}
	}
	for i := len(m.Entries) - 1; i >= 0; i-- {
		e := m.Entries[i]
		if e.Kind == "directory" {
			f, err := target.Open(e.Path)
			if err != nil {
				return err
			}
			err = f.Sync()
			f.Close()
			if err != nil {
				return fail("RESTORE_FAILED", "directory sync")
			}
		}
	}
	for _, root := range []*os.Root{target, managed} {
		f, err := root.Open(".")
		if err != nil {
			return err
		}
		err = f.Sync()
		f.Close()
		if err != nil {
			return fail("RESTORE_FAILED", "root sync")
		}
	}
	return nil
}
