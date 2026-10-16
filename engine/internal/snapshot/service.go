package snapshot

import (
	"bytes"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/workspace"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	Repo      Repository
	Objects   Objects
	root      *os.Root
	directory string
	gate      chan struct{}
}

func New(repo Repository, objects Objects, storePath string) (*Service, error) {
	absolute, err := filepath.Abs(storePath)
	if err != nil {
		return nil, err
	}
	absolute, err = filepath.EvalSymlinks(absolute)
	if err != nil {
		return nil, fail("STORAGE_ERROR", "storage root resolution")
	}
	root, err := os.OpenRoot(absolute)
	if err != nil {
		return nil, err
	}
	if err = root.Mkdir("managed-workspaces", 0700); err != nil && !errors.Is(err, os.ErrExist) {
		root.Close()
		return nil, fail("STORAGE_ERROR", "managed workspace directory")
	}
	info, err := root.Lstat("managed-workspaces")
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		root.Close()
		return nil, fail("UNSAFE_DESTINATION", "managed root")
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
	return &Service{repo, objects, root, absolute, make(chan struct{}, 1)}, nil
}
func (s *Service) Close() error { return s.root.Close() }
func (s *Service) enter() error {
	select {
	case s.gate <- struct{}{}:
		return nil
	default:
		return fail("BUSY", "snapshot operation active")
	}
}
func (s *Service) leave() { <-s.gate }
func (s *Service) Capture(ctx context.Context, spec CaptureSpec) (r Record, err error) {
	if !spec.Consent {
		return r, fail("CONSENT_REQUIRED", "snapshot captures source bytes")
	}
	if err = s.enter(); err != nil {
		return r, err
	}
	defer s.leave()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	run, err := s.Repo.GetRun(ctx, spec.RunID)
	if err != nil {
		return r, err
	}
	if !run.State.Terminal() {
		return r, fail("RUN_ACTIVE", "capture requires stopped execution")
	}
	source := run.Workspace
	if spec.BranchID != "" {
		b, e := s.Repo.Branch(ctx, spec.BranchID)
		if e != nil {
			return r, e
		}
		if b.State != "ready" || b.RunID != run.ID || b.Workspace != s.destination(b.ID) {
			return r, fail("INVALID_BRANCH", "capture source")
		}
		source = b.Workspace
	}
	source, err = workspace.Validate(source, spec.Excludes)
	if err != nil {
		return r, fail("INVALID_WORKSPACE", "capture source or exclusions")
	}
	inside, _ := filepath.Rel(s.directory, source)
	if spec.BranchID == "" && (inside == "." || inside != ".." && !strings.HasPrefix(inside, ".."+string(filepath.Separator))) {
		return r, fail("INVALID_WORKSPACE", "cannot capture application data")
	}
	excludes := append([]string{}, spec.Excludes...)
	relative, _ := filepath.Rel(source, s.directory)
	if relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		excludes = append(excludes, filepath.ToSlash(relative))
	}
	for _, x := range excludes {
		if !ValidPath(x) {
			return r, fail("INVALID_PATH", "exclusion path")
		}
	}
	if spec.EventID != "" {
		if _, err = s.Repo.GetEvent(ctx, run.ID, spec.EventID); err != nil {
			return r, err
		}
	}
	r = Record{ID: model.ID(), RunID: run.ID, EventID: spec.EventID, BranchID: spec.BranchID, Workspace: source, CreatedAt: model.Now(), State: "preparing"}
	if err = s.Repo.PrepareSnapshot(ctx, r); err != nil {
		return r, err
	}
	defer func() {
		if err != nil {
			_ = s.Repo.SnapshotState(context.Background(), r.ID, "failed", "capture_failed_or_interrupted_objects_retained")
			r.State = "failed"
			r.Diagnostic = "capture_failed_or_interrupted_objects_retained"
		}
	}()
	first, err := scan(ctx, source, excludes, s.Objects)
	if err != nil {
		return r, err
	}
	second, err := scan(ctx, source, excludes, nil)
	if err != nil {
		return r, err
	}
	a, _ := first.manifest.Bytes()
	b, _ := second.manifest.Bytes()
	if !bytes.Equal(a, b) {
		return r, fail("WORKSPACE_CHANGED", "two scans disagree")
	}
	for p, info := range first.identities {
		other, ok := second.identities[p]
		if !ok || !sameMetadata(info, other) {
			return r, fail("WORKSPACE_CHANGED", "entry identity changed")
		}
	}
	manifest, err := s.Objects.Put(ctx, a)
	if err != nil {
		return r, err
	}
	refs := []Ref{manifest}
	for _, e := range first.manifest.Entries {
		if e.Object != nil {
			refs = append(refs, *e.Object)
			r.Bytes += e.Object.Size
		}
	}
	for _, ref := range refs {
		if _, err = s.Objects.Get(ctx, ref); err != nil {
			return r, err
		}
	}
	r.Manifest = &manifest
	r.Entries = len(first.manifest.Entries)
	r.Omitted = len(first.manifest.Omitted)
	r.State = "committed"
	if err = s.Repo.CommitSnapshot(ctx, r, refs); err != nil {
		return r, err
	}
	return r, nil
}
func (s *Service) Validate(ctx context.Context, id string) (Record, Manifest, error) {
	r, err := s.Repo.Snapshot(ctx, id)
	if err != nil {
		return r, Manifest{}, err
	}
	if r.State != "committed" || r.Manifest == nil {
		return r, Manifest{}, fail("INVALID_SNAPSHOT", "snapshot is not committed")
	}
	damaged := func(err error) (Record, Manifest, error) {
		if ctx.Err() != nil {
			return r, Manifest{}, ctx.Err()
		}
		if updateErr := s.Repo.SnapshotState(context.Background(), id, "damaged", "manifest_or_object_integrity_failed"); updateErr != nil {
			return r, Manifest{}, updateErr
		}
		r.State = "damaged"
		return r, Manifest{}, err
	}
	raw, err := s.Objects.Get(ctx, *r.Manifest)
	if err != nil {
		return damaged(err)
	}
	m, err := Decode(raw)
	if err != nil {
		return damaged(err)
	}
	var total int64
	for _, e := range m.Entries {
		if e.Object != nil {
			total += e.Object.Size
			if _, err = s.Objects.Get(ctx, *e.Object); err != nil {
				return damaged(err)
			}
		}
	}
	if r.Entries != len(m.Entries) || r.Omitted != len(m.Omitted) || r.Bytes != total {
		return damaged(fail("INVALID_SNAPSHOT", "manifest metadata mismatch"))
	}
	return r, m, nil
}
func (s *Service) destination(id string) string {
	return filepath.Join(s.directory, "managed-workspaces", id)
}
func (s *Service) Materialize(ctx context.Context, spec MaterializeSpec, kind string) (b Branch, err error) {
	if !spec.Consent {
		return b, fail("CONSENT_REQUIRED", "materialization creates a private workspace")
	}
	if kind != "restore" && kind != "branch" {
		return b, fail("INVALID_ARGUMENT", "materialization kind")
	}
	if err = s.enter(); err != nil {
		return b, err
	}
	defer s.leave()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	source, m, err := s.Validate(ctx, spec.SnapshotID)
	if err != nil {
		return b, err
	}
	parent := spec.ParentID
	if kind == "branch" && parent == "" {
		parent = source.BranchID
	}
	if parent != "" {
		p, e := s.Repo.Branch(ctx, parent)
		if e != nil {
			return b, e
		}
		if p.State != "ready" || p.RunID != source.RunID || source.BranchID != parent {
			return b, fail("INVALID_BRANCH", "parent must own source snapshot")
		}
	}
	b = Branch{ID: model.ID(), SnapshotID: source.ID, ParentID: parent, RunID: source.RunID, CreatedAt: model.Now(), Kind: kind, State: "preparing"}
	b.Workspace = s.destination(b.ID)
	if err = s.Repo.PrepareBranch(ctx, b); err != nil {
		return b, err
	}
	defer func() {
		if err != nil {
			_ = s.Repo.FinishBranch(context.Background(), b.ID, "failed", "materialization_failed_directory_retained")
			b.State = "failed"
			b.Diagnostic = "materialization_failed_directory_retained"
		}
	}()
	if err = s.restore(ctx, b.ID, m); err != nil {
		return b, err
	}
	if err = s.Repo.FinishBranch(ctx, b.ID, "ready", ""); err != nil {
		return b, err
	}
	b.State = "ready"
	return b, nil
}
func (s *Service) Recover(ctx context.Context) error {
	if err := s.Repo.RecoverSnapshots(ctx); err != nil {
		return err
	}
	after := ""
	for {
		p, err := s.Repo.Snapshots(ctx, Query{After: after, Limit: 100})
		if err != nil {
			return err
		}
		for _, r := range p.Snapshots {
			if r.State == "committed" {
				_, _, err = s.Validate(ctx, r.ID)
				if err != nil {
					if ctx.Err() != nil {
						return ctx.Err()
					}
					saved, e := s.Repo.Snapshot(ctx, r.ID)
					if e != nil {
						return e
					}
					if saved.State != "damaged" {
						return err
					}
				}
			}
		}
		if p.Next == "" {
			return nil
		}
		after = p.Next
	}
}
func (s *Service) Inventory(ctx context.Context) (Inventory, error) {
	if err := s.enter(); err != nil {
		return Inventory{}, err
	}
	defer s.leave()
	return s.Objects.Inventory(ctx, func(r Ref) (bool, error) { return s.Repo.Referenced(ctx, r) })
}
