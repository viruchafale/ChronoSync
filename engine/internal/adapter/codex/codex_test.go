package codex

import (
	"bufio"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestFixtureMapping(t *testing.T) {
	f, e := os.Open("testdata/events.jsonl")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	expected := []string{"agent.thread", "agent.turn_started", "command.started", "command.completed", "file.reported", "agent.message", "adapter.unknown", "agent.turn_completed"}
	i := 0
	for s.Scan() {
		rec, terminal, err := Parse(s.Bytes())
		if err != nil || rec.Kind != expected[i] {
			t.Fatalf("%d: %+v %v", i, rec, err)
		}
		if i == 3 && (rec.Payload.Output != "fixture" || rec.Payload.ExitCode == nil || *rec.Payload.ExitCode != 0) {
			t.Fatal("lost command output")
		}
		if i == 6 {
			b, _ := json.Marshal(rec)
			if strings.Contains(string(b), "omit-this") {
				t.Fatal("unknown raw payload retained")
			}
		}
		if i == 7 && terminal != "completed" {
			t.Fatal("terminal missing")
		}
		i++
	}
	if i != len(expected) {
		t.Fatal("fixture incomplete")
	}
}
func TestMalformed(t *testing.T) {
	for _, b := range []string{"{", "null", "{}", `{"type":"item.completed"}`, `{"type":"thread.started"}`, `{"type":"item.completed","item":{"id":"x","type":"command_execution","exit_code":"zero"}}`} {
		if _, _, e := Parse([]byte(b)); e == nil {
			t.Fatalf("accepted %s", b)
		}
	}
}
func TestFailureEvents(t *testing.T) {
	for _, b := range []string{`{"type":"turn.failed","error":{"message":"fixture"}}`, `{"type":"error","message":"fixture"}`} {
		_, term, err := Parse([]byte(b))
		if err != nil || term != "failed" {
			t.Fatal("lost failure")
		}
	}
}
