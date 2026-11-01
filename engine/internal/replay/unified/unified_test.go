package unified

import (
	"bytes"
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/replay"
	"chronosync.local/engine/internal/replay/transport"
	"chronosync.local/engine/internal/storage"
)

func unifiedFixtures(t *testing.T, repeated bool) ([]byte, []byte, Plan) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	run := model.Run{Adapter: "codex", RecorderVersion: "test", SchemaVersion: 1, ID: model.ID(), Workspace: "/synthetic", AdapterVersion: "synthetic", Sandbox: "read-only", State: model.Created, CreatedAt: model.Now(), TimeoutSeconds: 30}
	if err = store.Create(run); err != nil {
		t.Fatal(err)
	}
	exit := 0
	events := []struct {
		kind    string
		payload model.Payload
	}{{"execution.state", model.Payload{Status: model.Created}}, {"execution.state", model.Payload{Status: model.Starting}}, {"execution.state", model.Payload{Status: model.Running}}, {replay.InteractionType, model.Payload{ItemID: "item", Command: "safe", Output: "result", ExitCode: &exit}}, {"agent.turn_completed", model.Payload{UpstreamType: "turn.completed"}}, {"execution.state", model.Payload{Status: model.Completed}}}
	for index, item := range events {
		if err = store.Append(model.Event{SchemaVersion: model.EventVersion, EventID: model.ID(), RunID: run.ID, Sequence: strconv.Itoa(index + 1), Kind: item.kind, ObservedAt: model.Now(), Adapter: "codex", Payload: item.payload}); err != nil {
			t.Fatal(err)
		}
	}
	regressionRaw, err := (replay.Exporter{Source: store}).Export(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	regressionFixture, _ := replay.Load(regressionRaw)
	identity := transport.TransportIdentity{Name: "fixture", Protocol: "synthetic-rpc", Version: "1"}
	request := transport.SyntheticRequest{Transport: identity, Operation: "tools.read", Occurrence: 1}
	drafts := []transport.TransportInteractionDraft{{Sequence: 1, Occurrence: 1, Request: request, Response: transport.SyntheticResponse{Status: 200}}}
	requests := []transport.SyntheticRequest{request}
	if repeated {
		second := request
		second.Occurrence = 2
		drafts = append(drafts, transport.TransportInteractionDraft{Sequence: 2, Occurrence: 2, Request: second, Response: transport.SyntheticResponse{Status: 201}})
		requests = append(requests, second)
	}
	recording, err := transport.NewTransportRecording(drafts)
	if err != nil {
		t.Fatal(err)
	}
	transportRaw, err := recording.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	plan := Plan{Version: Version, TimeoutMillis: 1000, TransportRequests: requests, Actuals: []Actual{{SourceSequence: regressionFixture.Interactions[0].SourceSequence, OutputSHA256: regressionFixture.Interactions[0].Expected.OutputSHA256, ExitCode: 0}}}
	return regressionRaw, transportRaw, plan
}

func TestUnifiedEvaluationPassMismatchAndStableSerialization(t *testing.T) {
	regressionRaw, transportRaw, plan := unifiedFixtures(t, true)
	result, err := Evaluate(context.Background(), regressionRaw, transportRaw, plan)
	if err != nil || result.Status != Completed || !result.Passed || len(result.Transport) != 2 || !result.Regression.Passed {
		t.Fatal(result, err)
	}
	first, err := MarshalResult(result)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadResult(first)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := MarshalResult(loaded)
	if !bytes.Equal(first, second) {
		t.Fatal("result serialization is unstable")
	}
	plan.Actuals[0].ExitCode = 7
	mismatch, err := Evaluate(context.Background(), regressionRaw, transportRaw, plan)
	if err != nil || mismatch.Status != Completed || mismatch.Passed || len(mismatch.Regression.Mismatches) != 1 || mismatch.Regression.Mismatches[0].Field != "exit_code" {
		t.Fatal(mismatch, err)
	}
}

func TestUnifiedEvaluationAmbiguousUnmatchedUnsupportedAndFreshState(t *testing.T) {
	regressionRaw, transportRaw, plan := unifiedFixtures(t, true)
	plan.TransportRequests[0].Occurrence = 0
	ambiguous, err := Evaluate(context.Background(), regressionRaw, transportRaw, plan)
	if err != nil || ambiguous.Passed || ambiguous.Transport[0].Kind != transport.TransportAmbiguous {
		t.Fatal(ambiguous, err)
	}
	regressionRaw, transportRaw, singlePlan := unifiedFixtures(t, false)
	singlePlan.TransportRequests[0].Operation = "tools.missing"
	unmatched, err := Evaluate(context.Background(), regressionRaw, transportRaw, singlePlan)
	if err != nil || unmatched.Passed || unmatched.Transport[0].Kind != transport.TransportUnmatched {
		t.Fatal(unmatched, err)
	}
	regressionRaw, transportRaw, plan = unifiedFixtures(t, false)
	plan.TransportRequests[0].Body = []byte("unsupported")
	unsupported, err := Evaluate(context.Background(), regressionRaw, transportRaw, plan)
	if err != nil || unsupported.Status != Unsupported || unsupported.Passed {
		t.Fatal(unsupported, err)
	}
	plan.TransportRequests[0].Body = nil
	first, err := Evaluate(context.Background(), regressionRaw, transportRaw, plan)
	second, secondErr := Evaluate(context.Background(), regressionRaw, transportRaw, plan)
	if err != nil || secondErr != nil || !first.Passed || !second.Passed {
		t.Fatal("matcher state leaked between evaluations", first, second, err, secondErr)
	}
}

func TestUnifiedEvaluationCancellationAndValidation(t *testing.T) {
	regressionRaw, transportRaw, plan := unifiedFixtures(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := Evaluate(ctx, regressionRaw, transportRaw, plan)
	if err != nil || result.Status != Cancelled || result.Passed {
		t.Fatal(result, err)
	}
	plan.TimeoutMillis = 0
	if _, err = Evaluate(context.Background(), regressionRaw, transportRaw, plan); err == nil {
		t.Fatal("invalid timeout accepted")
	}
	plan.TimeoutMillis = 1000
	if _, err = Evaluate(context.Background(), []byte("bad"), transportRaw, plan); err == nil {
		t.Fatal("malformed regression fixture accepted")
	}
	if _, err = Evaluate(context.Background(), regressionRaw, []byte("bad"), plan); err == nil {
		t.Fatal("malformed transport fixture accepted")
	}
	if _, err = LoadResult(append([]byte(`{"version":1}`), '\n')); err == nil {
		t.Fatal("noncanonical result accepted")
	}
}
