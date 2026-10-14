package recorder

import (
	"chronosync.local/engine/internal/adapter"
	"chronosync.local/engine/internal/adapter/codex"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/storage"
	"chronosync.local/engine/internal/supervisor"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type fixture struct{ mode string }

func (f fixture) Version(context.Context, string) (string, error) { return "synthetic", nil }
func (f fixture) Record(ctx context.Context, s model.StartSpec, sink adapter.Sink) adapter.Result {
	if err := sink(adapter.Record{Kind: "agent.message", Payload: model.Payload{Message: "api_key=synthetic-secret"}}); err != nil {
		return adapter.Result{Err: err}
	}
	switch f.mode {
	case "wait":
		<-ctx.Done()
		return adapter.Result{ExitCode: -1, Err: ctx.Err()}
	case "failure":
		return adapter.Result{ExitCode: 7, Err: errors.New("synthetic failure")}
	case "malformed":
		return adapter.Result{Completed: true, Malformed: 1}
	case "missing":
		return adapter.Result{}
	case "write":
		_ = os.WriteFile(filepath.Join(s.Workspace, "observed.txt"), []byte("fixture"), 0600)
	}
	return adapter.Result{Completed: true}
}
func TestRecorderLifecycle(t *testing.T) {
	for _, tc := range []struct {
		mode  string
		state model.State
	}{{"success", model.Completed}, {"failure", model.Failed}, {"malformed", model.Failed}, {"missing", model.Failed}, {"write", model.Completed}, {"wait", model.TimedOut}, {"cancel", model.Cancelled}} {
		t.Run(tc.mode, func(t *testing.T) {
			s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			mode := tc.mode
			if mode == "cancel" {
				mode = "wait"
			}
			r := New(s, fixture{mode})
			defer r.Close()
			spec := model.StartSpec{Workspace: t.TempDir(), Prompt: "synthetic", CodexPath: "/synthetic/codex", Sandbox: "read-only", TimeoutSeconds: 1, Consent: true}
			run, e := r.Start(spec)
			if e != nil {
				t.Fatal(e)
			}
			if tc.mode == "cancel" {
				time.Sleep(20 * time.Millisecond)
				if e = r.Cancel(run.ID); e != nil {
					t.Fatal(e)
				}
			}
			r.Wait(run.ID)
			h, e := s.Read(run.ID)
			if e != nil || h.Run.State != tc.state {
				t.Fatalf("%+v %v", h, e)
			}
			seen := map[string]bool{}
			changed := false
			for i, event := range h.Events {
				if event.Sequence != strconv.Itoa(i+1) || seen[event.EventID] {
					t.Fatal("identity/order")
				}
				seen[event.EventID] = true
				if event.Payload.Message == "api_key=synthetic-secret" {
					t.Fatal("secret persisted")
				}
				if event.Kind == "workspace.changed" {
					changed = true
				}
			}
			if tc.mode == "write" && !changed {
				t.Fatal("missing workspace change")
			}
			if len(h.Events) < 4 {
				t.Fatal("partial history lost")
			}
		})
	}
}
func TestConsentAndSingleActive(t *testing.T) {
	s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := New(s, fixture{"wait"})
	defer r.Close()
	spec := model.StartSpec{Workspace: t.TempDir(), Prompt: "x", CodexPath: "/synthetic", Sandbox: "read-only", TimeoutSeconds: 10}
	if _, e = r.Start(spec); e == nil {
		t.Fatal("no consent")
	}
	spec.Consent = true
	run, e := r.Start(spec)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = r.Start(spec); e == nil {
		t.Fatal("second active")
	}
	_ = r.Cancel(run.ID)
	r.Wait(run.ID)
}
func TestActualCodexAdapterProcess(t *testing.T) {
	if !supervisor.Supported() {
		t.Skip("platform gate")
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "synthetic-codex")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codex-cli 0.163.0'; exit 0; fi\nprintf '%s\\n' '{\"type\":\"thread.started\",\"thread_id\":\"synthetic\"}' '{\"type\":\"item.completed\",\"item\":{\"id\":\"x\",\"type\":\"command_execution\",\"command\":\"fixture\",\"aggregated_output\":\"fixture output\",\"exit_code\":0}}' '{\"type\":\"turn.completed\"}'\n"
	if e := os.WriteFile(cli, []byte(script), 0700); e != nil {
		t.Fatal(e)
	}
	s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := New(s, codex.CLI{})
	defer r.Close()
	run, e := r.Start(model.StartSpec{Workspace: dir, Prompt: "synthetic", CodexPath: cli, Sandbox: "read-only", TimeoutSeconds: 5, Consent: true})
	if e != nil {
		t.Fatal(e)
	}
	r.Wait(run.ID)
	h, e := s.Read(run.ID)
	if e != nil || h.Run.State != model.Completed {
		t.Fatalf("%+v %v", h, e)
	}
	found := false
	for _, event := range h.Events {
		if event.Kind == "command.completed" && event.Payload.Output == "fixture output" {
			found = true
		}
	}
	if !found {
		t.Fatal("command output missing")
	}
}

type failingRepository struct{ storage.Repository }

func (s failingRepository) Append(e model.Event) error {
	if e.Kind == "command.completed" {
		return &storage.Error{Code: "STORAGE_ERROR", Operation: "synthetic failure"}
	}
	return s.Repository.Append(e)
}
func TestStorageFailureStopsActualSubprocess(t *testing.T) {
	if !supervisor.Supported() {
		t.Skip("platform gate")
	}
	dir := t.TempDir()
	cli := filepath.Join(dir, "synthetic-codex")
	script := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codex-cli 0.163.0'; exit 0; fi\nprintf '%s\\n' '{\"type\":\"item.completed\",\"item\":{\"id\":\"x\",\"type\":\"command_execution\",\"command\":\"fixture\",\"aggregated_output\":\"fixture output\",\"exit_code\":0}}'\nsleep 20\n"
	if err := os.WriteFile(cli, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	s, err := storage.Open(filepath.Join(t.TempDir(), "store"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	r := New(failingRepository{s}, codex.CLI{})
	defer r.Close()
	started := time.Now()
	run, err := r.Start(model.StartSpec{Workspace: dir, Prompt: "synthetic", CodexPath: cli, Sandbox: "read-only", TimeoutSeconds: 10, Consent: true})
	if err != nil {
		t.Fatal(err)
	}
	r.Wait(run.ID)
	if time.Since(started) > 3*time.Second {
		t.Fatal("storage failure did not promptly stop subprocess")
	}
	saved, err := s.GetRun(context.Background(), run.ID)
	if err != nil || saved.State != model.Interrupted || saved.Reason != "storage_failure" || saved.LastSequence != "3" || r.Error() == nil {
		t.Fatal(saved, err, r.Error())
	}
}
