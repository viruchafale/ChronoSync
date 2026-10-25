package ipc

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"chronosync.local/engine/internal/adapter"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/recorder"
	"chronosync.local/engine/internal/storage"
)

type forbiddenTrustAdapter struct{ calls atomic.Int32 }

func (a *forbiddenTrustAdapter) Version(context.Context, string) (string, error) {
	a.calls.Add(1)
	return "synthetic", nil
}
func (a *forbiddenTrustAdapter) Record(context.Context, model.StartSpec, adapter.Sink) adapter.Result {
	a.calls.Add(1)
	return adapter.Result{Completed: true}
}
func TestIPCRejectsForgedAuthorityBeforeRecorder(t *testing.T) {
	for _, field := range []string{"verified", "capability_evidence", "effective_policy", "provider_identity", "approved_at", "approval", "admission", "launch_token", "checks", "policy"} {
		t.Run(field, func(t *testing.T) {
			dir := t.TempDir()
			s, e := storage.Open(filepath.Join(dir, "store"))
			if e != nil {
				t.Fatal(e)
			}
			defer s.Close()
			a := &forbiddenTrustAdapter{}
			r := recorder.New(s, a)
			params := map[string]any{"workspace": dir, "prompt": "synthetic-private-canary", "codex_path": "/synthetic/not-executed", "sandbox": "read-only", "timeout_seconds": 1, "consent": true, "excludes": []string{}, field: map[string]any{"verified": true, "secret": "synthetic-private-canary"}}
			body, _ := json.Marshal(params)
			request, _ := json.Marshal(Request{1, "forged", "run.start", body})
			var out bytes.Buffer
			if e := Serve(strings.NewReader("{\"protocol_version\":1,\"id\":\"hello\",\"method\":\"hello\"}\n"+string(request)), &out, r, s); e != nil {
				t.Fatal(e)
			}
			lines := strings.Split(strings.TrimSpace(out.String()), "\n")
			var response Response
			if len(lines) != 2 || json.Unmarshal([]byte(lines[1]), &response) != nil || response.Error == nil {
				t.Fatal(out.String())
			}
			if a.calls.Load() != 0 || strings.Contains(out.String(), "canary") {
				t.Fatal("authority or private content escaped boundary")
			}
			runs, e := s.List()
			if e != nil || len(runs) != 0 {
				t.Fatal("rejected request persisted", runs, e)
			}
		})
	}
}
func TestIPCNoApprovalCommandsOrMalformedAuthority(t *testing.T) {
	s, e := storage.Open(filepath.Join(t.TempDir(), "store"))
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	r := recorder.New(s, &forbiddenTrustAdapter{})
	defer r.Close()
	for _, method := range []string{"approval.respond", "challenge.create", "provider.register", "admission.claim"} {
		if _, e := dispatch(Request{1, "test", method, json.RawMessage(`{"verified":true}`)}, r, s); e == nil {
			t.Fatal("new authority command", method)
		}
	}
	for _, raw := range []string{`{"consent":"true"}`, `{"consent":false,"consent":true}`, `{"consent":false,"Consent":true}`, `null`, `{"consent":true} {}`, `{"verified":true}`, `{"prompt":`, strings.Repeat("x", MaxFrame+1)} {
		var spec model.StartSpec
		if strictRequest([]byte(raw), &spec) == nil {
			t.Fatal("malformed or oversized input accepted")
		}
	}
	var req Request
	if strictRequest([]byte(`{"protocol_version":1,"id":"x","method":"hello","verified":true}`), &req) == nil {
		t.Fatal("envelope verification accepted")
	}
}
