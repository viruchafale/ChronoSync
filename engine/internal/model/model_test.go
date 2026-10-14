package model

import (
	"encoding/json"
	"testing"
)

func TestEventContract(t *testing.T) {
	e := Event{SchemaVersion: 2, EventID: ID(), RunID: ID(), Sequence: "1", Kind: "command.completed", ObservedAt: Now(), Adapter: "codex", Payload: Payload{Output: "synthetic"}}
	if err := e.Validate(); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(e)
	var got Event
	if json.Unmarshal(b, &got) != nil || got.Payload.Output != "synthetic" {
		t.Fatal("round trip")
	}
	for _, bad := range []Event{func() Event { x := e; x.SchemaVersion = 3; return x }(), func() Event { x := e; x.Sequence = "0"; return x }(), func() Event { x := e; x.ObservedAt = "invalid"; return x }(), func() Event { x := e; x.Kind = "invented"; return x }()} {
		if bad.Validate() == nil {
			t.Fatal("accepted invalid event")
		}
	}
	seen := map[string]bool{}
	for range 1000 {
		id := ID()
		if !ValidID(id) || seen[id] {
			t.Fatal("nonunique identity")
		}
		seen[id] = true
	}
}
func TestTransitions(t *testing.T) {
	for _, pair := range [][2]State{{Created, Starting}, {Starting, Running}, {Running, Completed}, {Running, Cancelled}, {Starting, Failed}, {Running, TimedOut}, {Running, Interrupted}} {
		if Transition(pair[0], pair[1]) != nil {
			t.Fatal(pair)
		}
	}
	for _, pair := range [][2]State{{Completed, Running}, {Created, Running}, {Running, Starting}, {Failed, Completed}} {
		if Transition(pair[0], pair[1]) == nil {
			t.Fatal(pair)
		}
	}
}
