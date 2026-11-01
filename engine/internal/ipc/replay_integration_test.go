package ipc

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"chronosync.local/engine/internal/adapter"
	"chronosync.local/engine/internal/fixtures"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/recorder"
	"chronosync.local/engine/internal/replay/transport"
	"chronosync.local/engine/internal/replay/unified"
	"chronosync.local/engine/internal/storage"
)

type replayFixtureAdapter struct{}

func (replayFixtureAdapter) Version(context.Context, string) (string, error) {
	return "synthetic", nil
}

func (replayFixtureAdapter) Record(_ context.Context, _ model.StartSpec, sink adapter.Sink) adapter.Result {
	exit := 0
	if err := sink(adapter.Record{Kind: "command.completed", Payload: model.Payload{ItemID: "item", Command: "safe-command", Output: "safe-output", ExitCode: &exit}}); err != nil {
		return adapter.Result{Err: err}
	}
	if err := sink(adapter.Record{Kind: "agent.turn_completed", Payload: model.Payload{UpstreamType: "turn.completed"}}); err != nil {
		return adapter.Result{Err: err}
	}
	return adapter.Result{Completed: true}
}

func fixtureRequest(t *testing.T, method string, params any, service *fixtures.Service) any {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	result, err := fixtureDispatch(Request{ProtocolVersion: 1, ID: "test", Method: method, Params: raw}, service)
	if err != nil {
		t.Fatal(method, err)
	}
	return result
}

func TestSyntheticRegressionWorkflowPersistsAcrossRestart(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	workspace := filepath.Join(t.TempDir(), "workspace")
	if err := os.Mkdir(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	recorderService := recorder.New(store, replayFixtureAdapter{})
	run, err := recorderService.Start(model.StartSpec{Workspace: workspace, Prompt: "synthetic", CodexPath: "/synthetic/codex", Sandbox: "read-only", TimeoutSeconds: 10, Consent: true})
	if err != nil {
		t.Fatal(err)
	}
	recorderService.Wait(run.ID)
	fixtureService, err := fixtures.New(base, store)
	if err != nil {
		t.Fatal(err)
	}
	regression := fixtureRequest(t, "fixture.create", map[string]string{"run_id": run.ID}, fixtureService).(fixtures.Record)
	if regression.Kind != fixtures.RegressionV1 || regression.Interactions != 1 {
		t.Fatal(regression)
	}
	identity := transport.TransportIdentity{Name: "fixture", Protocol: "synthetic-rpc", Version: "1"}
	request := transport.SyntheticRequest{Transport: identity, Operation: "tools.read", Occurrence: 1}
	recording, err := transport.NewTransportRecording([]transport.TransportInteractionDraft{{Sequence: 1, Occurrence: 1, Request: request, Response: transport.SyntheticResponse{Status: 200}}})
	if err != nil {
		t.Fatal(err)
	}
	transportRaw, _ := recording.Bytes()
	transportRecord := fixtureRequest(t, "fixture.import", map[string]string{"bytes": string(transportRaw)}, fixtureService).(fixtures.Record)
	actual := regression.Details[0]
	plan := unified.Plan{Version: unified.Version, TimeoutMillis: 1000, Actuals: []unified.Actual{{SourceSequence: actual.Sequence, OutputSHA256: actual.ExpectedOutputFingerprint, ExitCode: *actual.ExpectedExitCode}}, TransportRequests: []transport.SyntheticRequest{request}}
	passed := fixtureRequest(t, "replay.evaluate", map[string]any{"regression_id": regression.ID, "transport_id": transportRecord.ID, "plan": plan}, fixtureService).(unified.Result)
	if !passed.Passed || passed.Status != unified.Completed {
		t.Fatal(passed)
	}
	plan.Actuals[0].ExitCode = 9
	mismatch := fixtureRequest(t, "replay.evaluate", map[string]any{"regression_id": regression.ID, "transport_id": transportRecord.ID, "plan": plan}, fixtureService).(unified.Result)
	if mismatch.Passed || len(mismatch.Regression.Mismatches) != 1 {
		t.Fatal(mismatch)
	}
	exported := fixtureRequest(t, "fixture.export", map[string]any{"id": transportRecord.ID, "consent": true}, fixtureService)
	exportedJSON, _ := json.Marshal(exported)
	var exportBody struct {
		Bytes string `json:"bytes"`
	}
	if json.Unmarshal(exportedJSON, &exportBody) != nil || exportBody.Bytes != string(transportRaw) {
		t.Fatal("export mismatch")
	}
	fixtureRequest(t, "fixture.delete", map[string]any{"id": transportRecord.ID, "consent": true}, fixtureService)
	reimported := fixtureRequest(t, "fixture.import", map[string]string{"bytes": exportBody.Bytes}, fixtureService).(fixtures.Record)
	if reimported.ID != transportRecord.ID {
		t.Fatal(reimported)
	}
	cancelledContext, cancel := context.WithCancel(context.Background())
	cancel()
	regressionRaw, _, _ := fixtureService.ReadForEvaluation(context.Background(), regression.ID)
	cancelled, err := unified.Evaluate(cancelledContext, regressionRaw, transportRaw, plan)
	if err != nil || cancelled.Status != unified.Cancelled {
		t.Fatal(cancelled, err)
	}
	recorderService.Close()
	fixtureService.Close()
	store.Close()

	store, err = storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	fixtureService, err = fixtures.New(base, store)
	if err != nil {
		t.Fatal(err)
	}
	defer fixtureService.Close()
	list := fixtureRequest(t, "fixture.list", struct{}{}, fixtureService).([]fixtures.Record)
	if len(list) != 2 {
		t.Fatal("fixtures unavailable after restart", list)
	}
	path := filepath.Join(base, "fixtures", "v1", "sha256", reimported.ID+".fixture")
	if err = os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = fixtureService.Inspect(context.Background(), reimported.ID); err == nil {
		t.Fatal("corrupted fixture accepted")
	}
}
