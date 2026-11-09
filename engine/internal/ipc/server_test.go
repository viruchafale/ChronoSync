package ipc

import (
	"bytes"
	"chronosync.local/engine/internal/adapter"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/recorder"
	"chronosync.local/engine/internal/storage"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type idleAdapter struct{}

func (idleAdapter) Version(context.Context, string) (string, error) { return "synthetic", nil }
func (idleAdapter) Record(context.Context, model.StartSpec, adapter.Sink) adapter.Result {
	return adapter.Result{Completed: true}
}
func TestNegotiationAndErrors(t *testing.T) {
	s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := recorder.New(s, idleAdapter{})
	input := strings.Join([]string{`{"protocol_version":1,"id":"1","method":"run.list","params":{}}`, `{"protocol_version":2,"id":"2","method":"hello"}`, `{"protocol_version":1,"id":"3","method":"hello"}`, `{"protocol_version":1,"id":"4","method":"run.list"}`, `{"protocol_version":1,"id":"5","method":"event.list","params":{"id":"../../outside","after":"0"}}`, `{"protocol_version":1,"id":"6","method":"unknown"}`}, "\n")
	var out bytes.Buffer
	if e = Serve(strings.NewReader(input), &out, r, s); e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 6 {
		t.Fatal(out.String())
	}
	for i, line := range lines {
		var response Response
		if e = json.Unmarshal([]byte(line), &response); e != nil {
			t.Fatal(e)
		}
		wantErr := i != 2 && i != 3
		if (response.Error != nil) != wantErr {
			t.Fatalf("response %d: %s", i, line)
		}
	}
}
func TestOversizeFrameAndEOF(t *testing.T) {
	s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := recorder.New(s, idleAdapter{})
	var out bytes.Buffer
	if Serve(strings.NewReader(strings.Repeat("x", MaxFrame+1)), &out, r, s) == nil {
		t.Fatal("unbounded input")
	}
}

func TestLegacyAndSnapshotParamsRejectUnknownFields(t *testing.T) {
	s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := recorder.New(s, idleAdapter{})
	input := strings.Join([]string{
		`{"protocol_version":1,"id":"hello","method":"hello"}`,
		`{"protocol_version":1,"id":"query","method":"run.query","params":{"limit":1,"unreviewed":true}}`,
		`{"protocol_version":1,"id":"cancel","method":"run.cancel","params":{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","approval":true}}`,
		`{"protocol_version":1,"id":"event","method":"event.list","params":{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","after":"0","filter":"ignored"}}`,
	}, "\n")
	var out bytes.Buffer
	if e = Serve(strings.NewReader(input), &out, r, s); e != nil {
		t.Fatal(e)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 4 {
		t.Fatal(out.String())
	}
	for index, line := range lines {
		var response Response
		if e = json.Unmarshal([]byte(line), &response); e != nil {
			t.Fatal(e)
		}
		if index == 0 && response.Error != nil {
			t.Fatalf("hello failed: %s", line)
		}
		if index > 0 && response.Error == nil {
			t.Fatalf("unknown params accepted: %s", line)
		}
	}

	unknown := json.RawMessage(`{"unreviewed":true}`)
	for _, method := range []string{"snapshot.capture", "snapshot.list", "branch.list", "snapshot.get", "snapshot.restore", "object.inventory"} {
		if _, e = snapshotDispatch(Request{Method: method, Params: unknown}, nil, nil); e == nil {
			t.Fatalf("%s accepted unknown params", method)
		}
	}
}
