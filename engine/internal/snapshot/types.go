package snapshot

import (
	"chronosync.local/engine/internal/model"
	"context"
)

type Record struct {
	ID         string `json:"id"`
	RunID      string `json:"run_id"`
	EventID    string `json:"event_id,omitempty"`
	BranchID   string `json:"branch_id,omitempty"`
	Workspace  string `json:"workspace"`
	CreatedAt  string `json:"created_at"`
	State      string `json:"state"`
	Diagnostic string `json:"diagnostic,omitempty"`
	Manifest   *Ref   `json:"manifest,omitempty"`
	Entries    int    `json:"entries"`
	Omitted    int    `json:"omitted"`
	Bytes      int64  `json:"bytes"`
}
type Branch struct {
	ID         string `json:"id"`
	SnapshotID string `json:"snapshot_id"`
	ParentID   string `json:"parent_id,omitempty"`
	RunID      string `json:"run_id"`
	CreatedAt  string `json:"created_at"`
	Kind       string `json:"kind"`
	State      string `json:"state"`
	Workspace  string `json:"workspace"`
	Diagnostic string `json:"diagnostic,omitempty"`
}
type CaptureSpec struct {
	RunID    string   `json:"run_id"`
	EventID  string   `json:"event_id,omitempty"`
	BranchID string   `json:"branch_id,omitempty"`
	Excludes []string `json:"excludes"`
	Consent  bool     `json:"consent"`
}
type MaterializeSpec struct {
	SnapshotID string `json:"snapshot_id"`
	ParentID   string `json:"parent_id,omitempty"`
	Consent    bool   `json:"consent"`
}
type Query struct {
	RunID string `json:"run_id"`
	After string `json:"after"`
	Limit int    `json:"limit"`
}
type Page struct {
	Snapshots []Record `json:"snapshots"`
	Next      string   `json:"next"`
}
type BranchPage struct {
	Branches []Branch `json:"branches"`
	Next     string   `json:"next"`
}
type Inventory struct {
	Objects           int   `json:"objects"`
	Bytes             int64 `json:"bytes"`
	Temporary         int   `json:"temporary"`
	Unreferenced      int   `json:"unreferenced"`
	UnreferencedBytes int64 `json:"unreferenced_bytes"`
}
type Objects interface {
	Put(context.Context, []byte) (Ref, error)
	Get(context.Context, Ref) ([]byte, error)
	Inventory(context.Context, func(Ref) (bool, error)) (Inventory, error)
}
type Repository interface {
	GetRun(context.Context, string) (model.Run, error)
	GetEvent(context.Context, string, string) (model.Event, error)
	PrepareSnapshot(context.Context, Record) error
	CommitSnapshot(context.Context, Record, []Ref) error
	Snapshot(context.Context, string) (Record, error)
	Snapshots(context.Context, Query) (Page, error)
	SnapshotState(context.Context, string, string, string) error
	PrepareBranch(context.Context, Branch) error
	FinishBranch(context.Context, string, string, string) error
	Branch(context.Context, string) (Branch, error)
	Branches(context.Context, Query) (BranchPage, error)
	Referenced(context.Context, Ref) (bool, error)
	RecoverSnapshots(context.Context) error
}
