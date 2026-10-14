// Package model contains provider-independent Phase 1 wire records.
package model

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"
)

const EventVersion = 2
const MaxEventBytes = 256 * 1024

type State string

const (
	Created     State = "created"
	Starting    State = "starting"
	Running     State = "running"
	Completed   State = "completed"
	Failed      State = "failed"
	Cancelled   State = "cancelled"
	TimedOut    State = "timed_out"
	Interrupted State = "interrupted"
)

func (s State) Terminal() bool {
	return s == Completed || s == Failed || s == Cancelled || s == TimedOut || s == Interrupted
}
func Transition(from, to State) error {
	valid := (from == Created && to == Starting) || (from == Starting && to == Running) || ((from == Created || from == Starting || from == Running) && to.Terminal())
	if !valid {
		return fmt.Errorf("invalid state transition %s -> %s", from, to)
	}
	return nil
}
func ID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}
func ValidID(s string) bool {
	b, e := hex.DecodeString(s)
	return e == nil && len(b) == 16 && s == hex.EncodeToString(b)
}
func Now() string { return time.Now().UTC().Format(time.RFC3339Nano) }

type FileMeta struct {
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	Mode   uint32 `json:"mode"`
}
type Change struct {
	Path      string    `json:"path"`
	Operation string    `json:"operation"`
	Before    *FileMeta `json:"before,omitempty"`
	After     *FileMeta `json:"after,omitempty"`
}

// Payload is deliberately bounded and typed: no raw vendor JSON is persisted.
type Truncation struct {
	Field         string `json:"field"`
	OriginalBytes int    `json:"original_bytes"`
	LimitBytes    int    `json:"limit_bytes"`
}
type Payload struct {
	Truncations  []Truncation `json:"truncations,omitempty"`
	Message      string       `json:"message,omitempty"`
	UpstreamType string       `json:"upstream_type,omitempty"`
	ItemID       string       `json:"item_id,omitempty"`
	ThreadID     string       `json:"thread_id,omitempty"`
	Command      string       `json:"command,omitempty"`
	Output       string       `json:"output,omitempty"`
	ExitCode     *int         `json:"exit_code,omitempty"`
	Status       State        `json:"status,omitempty"`
	Reason       string       `json:"reason,omitempty"`
	Changes      []Change     `json:"changes,omitempty"`
	Source       string       `json:"source,omitempty"`
}
type Event struct {
	SchemaVersion int     `json:"schema_version"`
	EventID       string  `json:"event_id"`
	RunID         string  `json:"run_id"`
	Sequence      string  `json:"sequence"`
	Kind          string  `json:"kind"`
	ObservedAt    string  `json:"observed_at"`
	Adapter       string  `json:"adapter"`
	Payload       Payload `json:"payload"`
}

var Kinds = map[string]bool{
	"execution.state": true, "execution.diagnostic": true, "execution.warning": true,
	"agent.thread": true, "agent.turn_started": true, "agent.turn_completed": true, "agent.turn_failed": true, "agent.message": true, "agent.reasoning_summary": true,
	"command.started": true, "command.updated": true, "command.completed": true, "tool.started": true, "tool.updated": true, "tool.completed": true,
	"file.reported": true, "workspace.changed": true, "adapter.unknown": true, "adapter.warning": true, "adapter.error": true,
}

func (e Event) Validate() error {
	n, err := strconv.ParseInt(e.Sequence, 10, 64)
	if e.SchemaVersion != EventVersion || !ValidID(e.EventID) || !ValidID(e.RunID) || err != nil || n < 1 || strconv.FormatInt(n, 10) != e.Sequence || !Kinds[e.Kind] || e.Adapter != "codex" {
		return errors.New("invalid event envelope")
	}
	t, err := time.Parse(time.RFC3339Nano, e.ObservedAt)
	if err != nil || t.Location() != time.UTC {
		return errors.New("invalid UTC timestamp")
	}
	if e.Kind == "execution.state" && !(e.Payload.Status == Created || e.Payload.Status == Starting || e.Payload.Status == Running || e.Payload.Status.Terminal()) {
		return errors.New("invalid event state")
	}
	if e.Payload.Status != "" && !(e.Payload.Status == Created || e.Payload.Status == Starting || e.Payload.Status == Running || e.Payload.Status.Terminal()) {
		return errors.New("invalid payload state")
	}
	for _, v := range []string{e.Payload.Message, e.Payload.Command, e.Payload.Output, e.Payload.Reason, e.Payload.Source, e.Payload.ItemID, e.Payload.ThreadID, e.Payload.UpstreamType} {
		if !utf8.ValidString(v) {
			return errors.New("invalid payload text")
		}
	}
	if len(e.Payload.Changes) > 1000 || len(e.Payload.Truncations) > 100 {
		return errors.New("payload collection limit")
	}
	for _, change := range e.Payload.Changes {
		if len(change.Path) == 0 || len(change.Path) > 32780 || len(change.Operation) == 0 || len(change.Operation) > 128 || !utf8.ValidString(change.Path) {
			return errors.New("invalid file change")
		}
		for _, meta := range []*FileMeta{change.Before, change.After} {
			if meta != nil {
				hash, err := hex.DecodeString(meta.SHA256)
				if meta.Size < 0 || err != nil || len(hash) != 32 || hex.EncodeToString(hash) != meta.SHA256 {
					return errors.New("invalid file metadata")
				}
			}
		}
	}
	for _, truncation := range e.Payload.Truncations {
		if truncation.Field == "" || truncation.LimitBytes < 1 || truncation.OriginalBytes <= truncation.LimitBytes {
			return errors.New("invalid truncation metadata")
		}
	}
	b, err := json.Marshal(e)
	if err != nil || len(b) > MaxEventBytes {
		return errors.New("event exceeds size limit")
	}
	return nil
}

type Run struct {
	Adapter         string `json:"adapter,omitempty"`
	RecorderVersion string `json:"recorder_version,omitempty"`
	StartedAt       string `json:"started_at,omitempty"`
	History         string `json:"history,omitempty"`
	Recovered       bool   `json:"recovered,omitempty"`
	Diagnostic      string `json:"diagnostic,omitempty"`
	SchemaVersion   int    `json:"schema_version"`
	ID              string `json:"id"`
	Workspace       string `json:"workspace"`
	AdapterVersion  string `json:"adapter_version"`
	Sandbox         string `json:"sandbox"`
	State           State  `json:"state"`
	CreatedAt       string `json:"created_at"`
	FinishedAt      string `json:"finished_at,omitempty"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	LastSequence    string `json:"last_sequence"`
	Reason          string `json:"reason,omitempty"`
	ExitCode        *int   `json:"exit_code,omitempty"`
}
type StartSpec struct {
	Workspace      string   `json:"workspace"`
	Prompt         string   `json:"prompt"`
	CodexPath      string   `json:"codex_path"`
	Sandbox        string   `json:"sandbox"`
	TimeoutSeconds int      `json:"timeout_seconds"`
	Consent        bool     `json:"consent"`
	Excludes       []string `json:"excludes"`
}
