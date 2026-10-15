package legacy

import (
	"chronosync.local/engine/internal/model"
	"os"
	"path/filepath"
	"testing"
)

func TestJournalRecoveryAndIntegrity(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	run := model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created, CreatedAt: model.Now()}
	if e = s.Create(run); e != nil {
		t.Fatal(e)
	}
	event := model.Event{SchemaVersion: 2, EventID: model.ID(), RunID: run.ID, Sequence: "1", Kind: "execution.state", ObservedAt: model.Now(), Adapter: "codex", Payload: model.Payload{Status: model.Running}}
	if e = s.Append(event); e != nil {
		t.Fatal(e)
	}
	if _, e = Open(dir); e == nil {
		t.Fatal("second writer allowed")
	}
	s.Close()
	f, e := os.OpenFile(filepath.Join(dir, run.ID, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, _ = f.WriteString("not-json\n{\"partial\":")
	f.Close()
	s, e = Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if e = s.Recover(); e != nil {
		t.Fatal(e)
	}
	h, e := s.Read(run.ID)
	if e != nil || len(h.Events) != 1 || len(h.Warnings) != 2 || h.Run.State != model.Interrupted {
		t.Fatalf("%+v %v", h, e)
	}
	if _, e = s.Read("../../outside"); e == nil {
		t.Fatal("path traversal")
	}
}
func TestRejectUnsupportedSchemaAndCorruption(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	s, e := Open(dir)
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	run := model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created}
	if e = s.Create(run); e != nil {
		t.Fatal(e)
	}
	event := model.Event{SchemaVersion: 3, EventID: model.ID(), RunID: run.ID, Sequence: "1", Kind: "agent.message", ObservedAt: model.Now(), Adapter: "codex"}
	if s.Append(event) == nil {
		t.Fatal("future schema accepted")
	}
	event.SchemaVersion = 2
	if e = s.Append(event); e != nil {
		t.Fatal(e)
	}
	p := filepath.Join(dir, run.ID, "events.jsonl")
	b, e := os.ReadFile(p)
	if e != nil {
		t.Fatal(e)
	}
	for i := range b {
		if b[i] == '2' {
			b[i] = '1'
			break
		}
	}
	if e = os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
	h, e := s.Read(run.ID)
	if e != nil || len(h.Events) != 0 || len(h.Warnings) == 0 {
		t.Fatal("corruption not surfaced")
	}
}

func TestDamagedRunDoesNotBlockOtherHistory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	good := model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created}
	if err = s.Create(good); err != nil {
		t.Fatal(err)
	}
	damaged := model.ID()
	if err = os.Mkdir(filepath.Join(dir, damaged), 0700); err != nil {
		t.Fatal(err)
	}
	if err = s.Recover(); err != nil {
		t.Fatal(err)
	}
	runs, err := s.List()
	if err != nil || len(runs) != 2 {
		t.Fatalf("unrelated history unavailable: %+v %v", runs, err)
	}
	if _, err = s.Read(good.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Read(damaged); err == nil {
		t.Fatal("damaged detail falsely accepted")
	}
}

func TestMalformedLinesPreserveLaterValidEventsAndBoundWarnings(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "store")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	run := model.Run{SchemaVersion: 1, ID: model.ID(), State: model.Created}
	if err = s.Create(run); err != nil {
		t.Fatal(err)
	}
	first := model.Event{SchemaVersion: 2, EventID: model.ID(), RunID: run.ID, Sequence: "1", Kind: "agent.message", ObservedAt: model.Now(), Adapter: "codex"}
	if err = s.Append(first); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(filepath.Join(dir, run.ID, "events.jsonl"), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	for range 150 {
		if _, err = f.WriteString("invalid\n"); err != nil {
			t.Fatal(err)
		}
	}
	f.Close()
	second := first
	second.EventID = model.ID()
	second.Sequence = "2"
	if err = s.Append(second); err != nil {
		t.Fatal(err)
	}
	h, err := s.Read(run.ID)
	if err != nil || len(h.Events) != 2 || h.Events[1].Sequence != "2" || len(h.Warnings) != 101 {
		t.Fatalf("history or bounded warnings: %d %d %v", len(h.Events), len(h.Warnings), err)
	}
}
