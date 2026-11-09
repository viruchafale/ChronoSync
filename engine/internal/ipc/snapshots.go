package ipc

import (
	"chronosync.local/engine/internal/recorder"
	"chronosync.local/engine/internal/snapshot"
	"context"
	"time"
)

func snapshotDispatch(req Request, r *recorder.Recorder, s *snapshot.Service) (any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	bad := func() error { return &snapshot.Error{Code: "INVALID_ARGUMENT", Operation: "snapshot request"} }
	switch req.Method {
	case "snapshot.capture":
		var p snapshot.CaptureSpec
		if strictRequest(req.Params, &p) != nil {
			return nil, bad()
		}
		return r.WhileIdle(func() (any, error) { return s.Capture(ctx, p) })
	case "snapshot.list":
		var p snapshot.Query
		if strictRequest(req.Params, &p) != nil {
			return nil, bad()
		}
		return s.Repo.Snapshots(ctx, p)
	case "branch.list":
		var p snapshot.Query
		if strictRequest(req.Params, &p) != nil {
			return nil, bad()
		}
		return s.Repo.Branches(ctx, p)
	case "snapshot.get", "snapshot.validate", "branch.get":
		var p struct {
			ID string `json:"id"`
		}
		if strictRequest(req.Params, &p) != nil {
			return nil, bad()
		}
		if req.Method == "branch.get" {
			return s.Repo.Branch(ctx, p.ID)
		}
		if req.Method == "snapshot.get" {
			return s.Repo.Snapshot(ctx, p.ID)
		}
		record, _, err := s.Validate(ctx, p.ID)
		return record, err
	case "snapshot.restore", "branch.create":
		var p snapshot.MaterializeSpec
		if strictRequest(req.Params, &p) != nil {
			return nil, bad()
		}
		kind := "restore"
		if req.Method == "branch.create" {
			kind = "branch"
		}
		return r.WhileIdle(func() (any, error) { return s.Materialize(ctx, p, kind) })
	case "object.inventory":
		var p struct{}
		if strictRequest(req.Params, &p) != nil {
			return nil, bad()
		}
		return s.Inventory(ctx)
	default:
		return nil, bad()
	}
}
