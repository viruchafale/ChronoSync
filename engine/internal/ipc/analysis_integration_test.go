package ipc

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"testing"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/storage"
)

func TestAnalysisWorkflowPersistsAcrossRestart(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	baseline := addAnalysisRun(t, store, "stable-output", false)
	changed := addAnalysisRun(t, store, "changed-output", true)
	service, err := analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	policy := analysis.DefaultPolicy()
	result := analysisRequest(t, "analysis.compare", map[string]any{"run_ids": []string{baseline, changed}, "policy": policy}, service)
	report, ok := result.(analysis.Report)
	if !ok || report.Evaluation.Passed || report.Evaluation.Decision != "fail" || len(report.Experiment.Comparisons) != 1 || report.Experiment.Comparisons[0].FirstDivergence == nil {
		t.Fatal(report)
	}
	listed := analysisRequest(t, "analysis.report.list", struct{}{}, service).([]analysis.ReportSummary)
	if len(listed) != 1 || listed[0].ReportID != report.ReportID {
		t.Fatal(listed)
	}
	policy.Name = "revised-observable-v1"
	policy.MaxAdditionalEvents = 1
	policy.MaxChangedDimensions = 3
	revised := analysisRequest(t, "analysis.report.reevaluate", map[string]any{"id": report.ReportID, "policy": policy}, service).(analysis.Report)
	if revised.ReportID == report.ReportID || revised.Experiment.Observations[0].ObservationID != report.Experiment.Observations[0].ObservationID {
		t.Fatal(revised)
	}
	service.Close()
	store.Close()

	store, err = storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service, err = analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	loaded := analysisRequest(t, "analysis.report.get", map[string]string{"id": report.ReportID}, service).(analysis.Report)
	if loaded.ReportID != report.ReportID || len(loaded.Experiment.Observations) != 2 {
		t.Fatal(loaded)
	}
	listed = analysisRequest(t, "analysis.report.list", struct{}{}, service).([]analysis.ReportSummary)
	if len(listed) != 2 {
		t.Fatal("historical reports unavailable after restart", listed)
	}
}

func TestAnalysisIPCRejectsUnknownPolicyFields(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	service, err := analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	raw := json.RawMessage(`{"run_ids":["11111111111111111111111111111111","22222222222222222222222222222222"],"policy":{"format_version":1,"name":"x","exact_min_event_agreement":1,"partial_min_event_agreement":0.8,"min_dimension_agreement":1,"max_missing_events":0,"max_additional_events":0,"max_reordered_events":0,"max_changed_dimensions":0,"required_dimensions":[],"unsupported_handling":"fail","trusted":true}}`)
	if _, err = analysisDispatch(Request{ProtocolVersion: 1, ID: "test", Method: "analysis.compare", Params: raw}, service); err == nil {
		t.Fatal("unknown policy authority accepted")
	}
}

func analysisRequest(t *testing.T, method string, params any, service *analysis.Service) any {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	result, err := analysisDispatch(Request{ProtocolVersion: 1, ID: "test", Method: method, Params: raw}, service)
	if err != nil {
		t.Fatal(method, err)
	}
	return result
}

func addAnalysisRun(t *testing.T, store *storage.Store, output string, additional bool) string {
	t.Helper()
	runID := model.ID()
	run := model.Run{SchemaVersion: 1, ID: runID, Workspace: "/synthetic/workspace", Adapter: "codex", AdapterVersion: "synthetic", RecorderVersion: "synthetic", Sandbox: "read-only", State: model.Created, CreatedAt: model.Now(), TimeoutSeconds: 30}
	if err := store.Create(run); err != nil {
		t.Fatal(err)
	}
	exit := 0
	events := []model.Event{
		analysisEvent(runID, "1", "execution.state", model.Payload{Status: model.Created}),
		analysisEvent(runID, "2", "execution.state", model.Payload{Status: model.Starting}),
		analysisEvent(runID, "3", "execution.state", model.Payload{Status: model.Running}),
		analysisEvent(runID, "4", "command.completed", model.Payload{ItemID: "item", Command: "synthetic-command", Output: output, ExitCode: &exit}),
	}
	if additional {
		events = append(events, analysisEvent(runID, "5", "tool.completed", model.Payload{ItemID: "tool", Command: "synthetic-tool", Output: "tool-output", ExitCode: &exit}))
	}
	events = append(events, analysisEvent(runID, strconv.Itoa(len(events)+1), "execution.state", model.Payload{Status: model.Completed}))
	if err := store.AppendBatch(context.Background(), events); err != nil {
		t.Fatal(err)
	}
	return runID
}

func analysisEvent(runID, sequence, kind string, payload model.Payload) model.Event {
	return model.Event{SchemaVersion: model.EventVersion, EventID: model.ID(), RunID: runID, Sequence: sequence, Kind: kind, ObservedAt: model.Now(), Adapter: "codex", Payload: payload}
}
