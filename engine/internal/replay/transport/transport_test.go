package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var syntheticIdentity = TransportIdentity{Name: "fixture", Protocol: "synthetic-rpc", Version: "1"}

func syntheticRequest(operation string, occurrence uint32) SyntheticRequest {
	return SyntheticRequest{
		Transport: syntheticIdentity, Operation: operation, Occurrence: occurrence,
		Metadata: []TransportMetadata{{Name: "content-type", Value: "application/json"}, {Name: "tool-name", Value: "fixture.read"}},
	}
}

func syntheticResponse(status int, result string) SyntheticResponse {
	return SyntheticResponse{
		Status:   status,
		Metadata: []TransportMetadata{{Name: "content-type", Value: "application/json"}, {Name: "result-kind", Value: result}},
	}
}

func transportRecording(t *testing.T) TransportRecording {
	t.Helper()
	recording, err := NewTransportRecording([]TransportInteractionDraft{
		{Sequence: 1, Occurrence: 1, Request: syntheticRequest("tools.read", 0), Response: syntheticResponse(200, "first")},
		{Sequence: 2, Occurrence: 1, Request: syntheticRequest("tools.list", 0), Response: syntheticResponse(200, "listing")},
		{Sequence: 3, Occurrence: 2, Request: syntheticRequest("tools.read", 0), Response: syntheticResponse(200, "second")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return recording
}

func TestSyntheticTransportStableMatchingAndReset(t *testing.T) {
	recording := transportRecording(t)
	first, err := recording.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	second, err := recording.Bytes()
	if err != nil || !bytes.Equal(first, second) {
		t.Fatal("transport fixture is not byte stable", err)
	}
	loaded, err := LoadTransportRecording(first)
	if err != nil {
		t.Fatal(err)
	}
	loadedBytes, err := loaded.Bytes()
	if err != nil || !bytes.Equal(first, loadedBytes) {
		t.Fatal("transport fixture changed after strict load", err)
	}
	matcher := NewTransportMatcher(loaded)
	var port SyntheticTransport = matcher

	ambiguous := port.Exchange(context.Background(), syntheticRequest("tools.read", 0))
	if ambiguous.Kind != TransportAmbiguous || ambiguous.Code != "OCCURRENCE_REQUIRED" {
		t.Fatal("repeated request was guessed", ambiguous)
	}
	request := syntheticRequest("tools.read", 1)
	matched := port.Exchange(context.Background(), request)
	if matched.Kind != TransportMatched || matched.Response.Status != 200 || matched.Response.Metadata[1].Value != "first" || matched.ResponseReference == "" || !validDigest(matched.ResponseFingerprint) {
		t.Fatal("first occurrence did not match", matched)
	}
	matched.Response.Metadata[0].Value = "mutated"
	request.Occurrence = 2
	matched = port.Exchange(context.Background(), request)
	if matched.Kind != TransportMatched || matched.Response.Metadata[0].Value != "application/json" || matched.Response.Metadata[1].Value != "second" {
		t.Fatal("returned response aliases stored state", matched)
	}
	if repeated := port.Exchange(context.Background(), request); repeated.Kind != TransportInvalid || repeated.Code != "INVALID_OCCURRENCE_ORDER" {
		t.Fatal("consumed occurrence accepted", repeated)
	}
	if unmatched := port.Exchange(context.Background(), syntheticRequest("tools.missing", 0)); unmatched.Kind != TransportUnmatched {
		t.Fatal("unrecorded request matched", unmatched)
	}
	matcher.Reset()
	request.Occurrence = 1
	if reset := matcher.Exchange(context.Background(), request); reset.Kind != TransportMatched || reset.Response.Metadata[1].Value != "first" {
		t.Fatal("reset did not create clean evaluation state", reset)
	}
}

func TestSyntheticTransportCancellationConsumesNothing(t *testing.T) {
	matcher := NewTransportMatcher(transportRecording(t))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	request := syntheticRequest("tools.read", 1)
	if outcome := matcher.Exchange(ctx, request); outcome.Kind != TransportCancelled {
		t.Fatal("cancelled request continued", outcome)
	}
	if outcome := matcher.Exchange(context.Background(), request); outcome.Kind != TransportMatched {
		t.Fatal("cancelled request consumed occurrence", outcome)
	}
}

func TestSyntheticTransportCanonicalFingerprintAndInputOwnership(t *testing.T) {
	request := syntheticRequest("tools.read", 0)
	response := syntheticResponse(200, "first")
	drafts := []TransportInteractionDraft{{Sequence: 1, Occurrence: 1, Request: request, Response: response}}
	recording, err := NewTransportRecording(drafts)
	if err != nil {
		t.Fatal(err)
	}
	before, err := recording.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	drafts[0].Request.Metadata[0].Value = "mutated"
	drafts[0].Response.Metadata[0].Value = "mutated"
	after, err := recording.Bytes()
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("recording aliases input buffers", err)
	}

	reorderedRequest := syntheticRequest("tools.read", 0)
	reorderedRequest.Metadata[0], reorderedRequest.Metadata[1] = reorderedRequest.Metadata[1], reorderedRequest.Metadata[0]
	reorderedResponse := syntheticResponse(200, "first")
	reorderedResponse.Metadata[0], reorderedResponse.Metadata[1] = reorderedResponse.Metadata[1], reorderedResponse.Metadata[0]
	reordered, err := NewTransportRecording([]TransportInteractionDraft{{Sequence: 1, Occurrence: 1, Request: reorderedRequest, Response: reorderedResponse}})
	if err != nil {
		t.Fatal(err)
	}
	reorderedBytes, err := reordered.Bytes()
	if err != nil || !bytes.Equal(before, reorderedBytes) {
		t.Fatal("canonical metadata order changed fingerprints", err)
	}
}

func TestSyntheticTransportRefusesSensitiveOrUnsupportedValues(t *testing.T) {
	valid := TransportInteractionDraft{Sequence: 1, Occurrence: 1, Request: syntheticRequest("tools.read", 0), Response: syntheticResponse(200, "ok")}
	tests := map[string]func(*TransportInteractionDraft){
		"request body":  func(draft *TransportInteractionDraft) { draft.Request.Body = []byte("private prompt") },
		"response body": func(draft *TransportInteractionDraft) { draft.Response.Body = []byte("private result") },
		"authorization": func(draft *TransportInteractionDraft) {
			draft.Request.Metadata = []TransportMetadata{{Name: "authorization", Value: "Bearer-synthetic"}}
		},
		"cookie": func(draft *TransportInteractionDraft) {
			draft.Response.Metadata = []TransportMetadata{{Name: "cookie", Value: "synthetic"}}
		},
		"secret pattern": func(draft *TransportInteractionDraft) {
			draft.Request.Operation = "api_key=synthetic-secret"
		},
		"prompt text": func(draft *TransportInteractionDraft) { draft.Request.Operation = "tell me private content" },
		"duplicate field": func(draft *TransportInteractionDraft) {
			draft.Request.Metadata = []TransportMetadata{{Name: "accept", Value: "application/json"}, {Name: "accept", Value: "text/plain"}}
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			draft := valid
			draft.Request.Metadata = append([]TransportMetadata(nil), valid.Request.Metadata...)
			draft.Response.Metadata = append([]TransportMetadata(nil), valid.Response.Metadata...)
			mutate(&draft)
			_, err := NewTransportRecording([]TransportInteractionDraft{draft})
			if err == nil || strings.Contains(err.Error(), "synthetic-secret") || strings.Contains(err.Error(), "private") {
				t.Fatal("unsafe interaction accepted or leaked", err)
			}
		})
	}

	recording, err := NewTransportRecording([]TransportInteractionDraft{valid})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := recording.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"tools.read", "fixture.read", "api_key", "authorization", "private prompt"} {
		if bytes.Contains(raw, []byte(forbidden)) {
			t.Fatalf("request content persisted: %q", forbidden)
		}
	}
	matcher := NewTransportMatcher(recording)
	requestWithBody := syntheticRequest("tools.read", 1)
	requestWithBody.Body = []byte("synthetic private body")
	if outcome := matcher.Exchange(context.Background(), requestWithBody); outcome.Kind != TransportUnsupported || outcome.Code != "UNSUPPORTED_INTERACTION" {
		t.Fatal("unsupported body was not typed", outcome)
	}
}

func TestSyntheticTransportFixtureStrictValidation(t *testing.T) {
	raw, err := transportRecording(t).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = LoadTransportRecording(append(raw, '\n')); !transportErrorIs(err, "NON_CANONICAL_TRANSPORT_FIXTURE") {
		t.Fatal("noncanonical transport fixture accepted", err)
	}
	if _, err = LoadTransportRecording(make([]byte, MaxTransportFixtureBytes+1)); !transportErrorIs(err, "TRANSPORT_FIXTURE_TOO_LARGE") {
		t.Fatal("oversized transport fixture accepted", err)
	}
	if _, err = LoadTransportRecording([]byte(`{"fixture":`)); !transportErrorIs(err, "INVALID_TRANSPORT_FIXTURE") {
		t.Fatal("malformed transport fixture accepted", err)
	}

	var envelope transportFixtureEnvelope
	if err = json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	mutations := map[string]func(*transportFixtureWire){
		"version":            func(wire *transportFixtureWire) { wire.FormatVersion++ },
		"duplicate sequence": func(wire *transportFixtureWire) { wire.Entries[1].Sequence = 1 },
		"bad occurrence":     func(wire *transportFixtureWire) { wire.Entries[2].Occurrence = 1 },
		"missing reference":  func(wire *transportFixtureWire) { wire.Entries[0].ResponseReference = "rsp_missing" },
		"bad request hash":   func(wire *transportFixtureWire) { wire.Entries[0].RequestFingerprint = "bad" },
		"bad response hash":  func(wire *transportFixtureWire) { wire.Responses[0].Fingerprint = strings.Repeat("0", 64) },
		"response order": func(wire *transportFixtureWire) {
			wire.Responses[0], wire.Responses[1] = wire.Responses[1], wire.Responses[0]
		},
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			changed := cloneTransportWire(envelope.Fixture)
			mutate(&changed)
			changedBytes, marshalErr := json.Marshal(changed)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			sealed, marshalErr := json.Marshal(transportFixtureEnvelope{Fixture: changed, SHA256: digest(changedBytes)})
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, loadErr := LoadTransportRecording(sealed); loadErr == nil {
				t.Fatal("invalid transport fixture accepted")
			}
		})
	}

	var unknown map[string]any
	if err = json.Unmarshal(raw, &unknown); err != nil {
		t.Fatal(err)
	}
	unknown["unknown"] = true
	unknownBytes, err := json.Marshal(unknown)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = LoadTransportRecording(unknownBytes); !transportErrorIs(err, "INVALID_TRANSPORT_FIXTURE") {
		t.Fatal("unknown transport field accepted", err)
	}
}

func TestSyntheticTransportRejectsMalformedRecording(t *testing.T) {
	if _, err := (TransportRecording{}).Bytes(); err == nil {
		t.Fatal("zero recording serialized")
	}
	request := syntheticRequest("tools.read", 0)
	response := syntheticResponse(200, "ok")
	tests := []struct {
		name   string
		drafts []TransportInteractionDraft
	}{
		{"empty", nil},
		{"sequence gap", []TransportInteractionDraft{{Sequence: 2, Occurrence: 1, Request: request, Response: response}}},
		{"occurrence gap", []TransportInteractionDraft{{Sequence: 1, Occurrence: 2, Request: request, Response: response}}},
		{"invalid response", []TransportInteractionDraft{{Sequence: 1, Occurrence: 1, Request: request, Response: SyntheticResponse{Status: 42}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewTransportRecording(test.drafts); err == nil {
				t.Fatal("malformed recording accepted")
			}
		})
	}
}

func TestSyntheticTransportProductionGraphHasNoNetworkOrLauncher(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	engineRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	pending := []string{filepath.Dir(currentFile)}
	visited := map[string]bool{}
	for len(pending) > 0 {
		directory := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[directory] {
			continue
		}
		visited[directory] = true
		entries, err := os.ReadDir(directory)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			parsed, parseErr := parser.ParseFile(token.NewFileSet(), filepath.Join(directory, entry.Name()), nil, parser.ImportsOnly)
			if parseErr != nil {
				t.Fatal(parseErr)
			}
			for _, spec := range parsed.Imports {
				path := strings.Trim(spec.Path.Value, `"`)
				if path == "net" || strings.HasPrefix(path, "net/") || path == "os/exec" || strings.Contains(path, "/internal/adapter") || strings.Contains(path, "/internal/supervisor") || strings.Contains(path, "/internal/storage") {
					t.Fatalf("forbidden production dependency %q in %s", path, entry.Name())
				}
				const modulePrefix = "chronosync.local/engine/"
				if strings.HasPrefix(path, modulePrefix) {
					pending = append(pending, filepath.Join(engineRoot, strings.TrimPrefix(path, modulePrefix)))
				} else if strings.Contains(path, ".") {
					t.Fatalf("third-party production dependency %q in %s", path, entry.Name())
				}
			}
		}
	}
}

func transportErrorIs(err error, code string) bool {
	var typedError *Error
	return errors.As(err, &typedError) && typedError.Code == code
}

func cloneTransportWire(wire transportFixtureWire) transportFixtureWire {
	wire.Entries = append([]transportEntryWire(nil), wire.Entries...)
	wire.Responses = append([]transportResponseWire(nil), wire.Responses...)
	for index := range wire.Responses {
		wire.Responses[index].Metadata = append([]TransportMetadata(nil), wire.Responses[index].Metadata...)
	}
	return wire
}
