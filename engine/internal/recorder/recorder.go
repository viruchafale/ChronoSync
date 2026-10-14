package recorder

import (
	"chronosync.local/engine/internal/adapter"
	"chronosync.local/engine/internal/contract"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/redact"
	"chronosync.local/engine/internal/storage"
	"chronosync.local/engine/internal/workspace"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type active struct {
	id         string
	cancel     context.CancelFunc
	done       chan struct{}
	userCancel bool
}
type Recorder struct {
	store     storage.Repository
	adapter   adapter.Adapter
	mu        sync.Mutex
	active    *active
	lastError error
	closed    bool
}

func New(s storage.Repository, a adapter.Adapter) *Recorder { return &Recorder{store: s, adapter: a} }
func (r *Recorder) Start(spec model.StartSpec) (model.Run, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return model.Run{}, errors.New("engine is shutting down")
	}
	if r.active != nil {
		return model.Run{}, errors.New("an execution is already active")
	}
	if !spec.Consent {
		return model.Run{}, errors.New("explicit execution consent required")
	}
	if len(strings.TrimSpace(spec.Prompt)) == 0 || len(spec.Prompt) > 32768 {
		return model.Run{}, errors.New("prompt must contain 1–32768 bytes")
	}
	if spec.TimeoutSeconds < 1 || spec.TimeoutSeconds > 3600 {
		return model.Run{}, errors.New("timeout must be 1–3600 seconds")
	}
	if spec.Sandbox != "read-only" && spec.Sandbox != "workspace-write" {
		return model.Run{}, errors.New("unsupported sandbox policy")
	}
	path, err := workspace.Validate(spec.Workspace, spec.Excludes)
	if err != nil {
		return model.Run{}, err
	}
	spec.Workspace = path
	if !filepath.IsAbs(spec.CodexPath) {
		return model.Run{}, errors.New("Codex path must be absolute")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(spec.TimeoutSeconds)*time.Second)
	run := model.Run{Adapter: "codex", RecorderVersion: contract.CurrentVersion().EngineVersion, SchemaVersion: 1, ID: model.ID(), Workspace: path, AdapterVersion: "pending", Sandbox: spec.Sandbox, State: model.Created, CreatedAt: model.Now(), TimeoutSeconds: spec.TimeoutSeconds, LastSequence: "0"}
	if err = r.store.Create(run); err != nil {
		cancel()
		return model.Run{}, err
	}
	a := &active{id: run.ID, cancel: cancel, done: make(chan struct{})}
	r.active = a
	r.lastError = nil
	go r.execute(ctx, a, run, spec)
	return run, nil
}
func (r *Recorder) execute(ctx context.Context, a *active, run model.Run, spec model.StartSpec) {
	defer func() { a.cancel(); r.mu.Lock(); r.active = nil; close(a.done); r.mu.Unlock() }()
	var sequence int64
	appendEvent := func(kind string, p model.Payload) error {
		if sequence >= storage.MaxEvents-5 && kind != "execution.state" {
			return errors.New("event budget exceeded")
		}
		nextSequence := sequence + 1
		e := model.Event{SchemaVersion: 2, EventID: model.ID(), RunID: run.ID, Sequence: strconv.FormatInt(nextSequence, 10), Kind: kind, ObservedAt: model.Now(), Adapter: "codex", Payload: redact.Payload(p)}
		if err := r.store.Append(e); err != nil {
			return err
		}
		sequence = nextSequence
		run.LastSequence = e.Sequence
		return nil
	}
	state := func(next model.State, reason string, exit *int) error {
		if err := model.Transition(run.State, next); err != nil {
			return err
		}
		if err := appendEvent("execution.state", model.Payload{Status: next, Reason: reason, ExitCode: exit}); err != nil {
			return err
		}
		run.State = next
		run.Reason = reason
		run.ExitCode = exit
		if next.Terminal() {
			run.FinishedAt = model.Now()
		}
		return r.store.Save(run)
	}
	failStorage := func(err error) {
		r.mu.Lock()
		r.lastError = fmt.Errorf("recording storage failed: %w", err)
		r.mu.Unlock()
		run.State = model.Interrupted
		run.Reason = "storage_failure"
		run.FinishedAt = model.Now()
		_ = r.store.Save(run)
	}
	if err := appendEvent("execution.state", model.Payload{Status: model.Created}); err != nil {
		failStorage(err)
		return
	}
	if err := state(model.Starting, "", nil); err != nil {
		failStorage(err)
		return
	}
	finish := func(next model.State, reason string, exit *int) {
		if ctx.Err() != nil {
			r.mu.Lock()
			user := a.userCancel
			r.mu.Unlock()
			if user {
				next = model.Cancelled
				reason = "user_cancelled"
			} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				next = model.TimedOut
				reason = "deadline_exceeded"
			} else {
				next = model.Interrupted
				reason = "engine_shutdown"
			}
		}
		if err := state(next, reason, exit); err != nil {
			failStorage(err)
		}
	}
	version, err := r.adapter.Version(ctx, spec.CodexPath)
	if err != nil {
		if writeErr := appendEvent("execution.diagnostic", model.Payload{Message: redact.Text(err.Error())}); writeErr != nil {
			failStorage(writeErr)
			return
		}
		finish(model.Failed, "adapter_unavailable", nil)
		return
	}
	run.AdapterVersion = version
	before, err := workspace.Scan(ctx, spec.Workspace, spec.Excludes)
	if err != nil {
		if writeErr := appendEvent("execution.warning", model.Payload{Message: err.Error()}); writeErr != nil {
			failStorage(writeErr)
			return
		}
		finish(model.Failed, "workspace_scan_failed", nil)
		return
	}
	if err = state(model.Running, "", nil); err != nil {
		failStorage(err)
		return
	}
	var sinkErr error
	result := r.adapter.Record(ctx, spec, func(rec adapter.Record) error { sinkErr = appendEvent(rec.Kind, rec.Payload); return sinkErr })
	if sinkErr != nil {
		failStorage(sinkErr)
		return
	}
	if result.Diagnostic != "" {
		if err = appendEvent("execution.diagnostic", model.Payload{Message: result.Diagnostic}); err != nil {
			failStorage(err)
			return
		}
	}
	// Post-run observation has its own bound so cancellation still preserves changes.
	scanCtx, scanCancel := context.WithTimeout(context.Background(), 5*time.Second)
	after, scanErr := workspace.Scan(scanCtx, spec.Workspace, spec.Excludes)
	scanCancel()
	if scanErr != nil {
		err = appendEvent("execution.warning", model.Payload{Message: "Final workspace observation incomplete"})
	} else {
		for _, change := range workspace.Diff(before, after) {
			if err = appendEvent("workspace.changed", model.Payload{Source: "workspace_scan", Changes: []model.Change{change}}); err != nil {
				break
			}
		}
		if err == nil {
			err = appendEvent("execution.warning", model.Payload{Message: fmt.Sprintf("Metadata-only observation complete; excluded/skipped entries before=%d after=%d. Transient or external changes are not attributed to Codex.", before.Skipped, after.Skipped)})
		}
	}
	if err != nil {
		failStorage(err)
		return
	}
	next, reason := model.Completed, "turn_and_process_completed"
	if scanErr != nil {
		next, reason = model.Failed, "workspace_observation_incomplete"
	} else if result.Err != nil || result.ExitCode != 0 {
		next, reason = model.Failed, "process_failure"
	} else if result.Malformed > 0 {
		next, reason = model.Failed, "malformed_upstream_records"
	} else if result.Failed || !result.Completed {
		next, reason = model.Failed, "missing_or_failed_terminal_turn"
	}
	finish(next, reason, &result.ExitCode)
}
func (r *Recorder) Cancel(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active == nil {
		run, err := r.store.GetRun(context.Background(), id)
		if err == nil && run.State.Terminal() {
			return nil
		}
		return errors.New("run is not active")
	}
	if r.active.id != id {
		return errors.New("run is not active")
	}
	r.active.userCancel = true
	r.active.cancel()
	return nil
}
func (r *Recorder) Close() {
	r.mu.Lock()
	r.closed = true
	a := r.active
	if a != nil {
		a.cancel()
	}
	r.mu.Unlock()
	if a != nil {
		<-a.done
	}
}
func (r *Recorder) Wait(id string) {
	r.mu.Lock()
	a := r.active
	r.mu.Unlock()
	if a != nil && a.id == id {
		<-a.done
	}
}
func (r *Recorder) Error() error { r.mu.Lock(); defer r.mu.Unlock(); return r.lastError }

// WhileIdle prevents a new run from starting during a source capture operation.
func (r *Recorder) WhileIdle(operation func() (any, error)) (any, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active != nil || r.closed {
		return nil, errors.New("snapshot operation requires an idle recorder")
	}
	return operation()
}
