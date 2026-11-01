package analysis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/storage"
)

func TestComparisonScenariosAndNoCausalClaim(t *testing.T) {
	baseline := syntheticObservation("baseline", []EventEvidence{syntheticEvent("1", "command.completed", "tool-a", "out-a", ""), syntheticEvent("2", "tool.completed", "tool-b", "out-b", "")})
	tests := []struct {
		name        string
		observation Observation
		status      Status
		kind        FindingKind
	}{
		{"identical", cloneObservation(baseline, "identical"), Exact, ""},
		{"changed output", syntheticObservation("changed", []EventEvidence{syntheticEvent("1", "command.completed", "tool-a", "different", ""), syntheticEvent("2", "tool.completed", "tool-b", "out-b", "")}), Partial, ChangedFingerprint},
		{"missing interaction", syntheticObservation("missing", []EventEvidence{syntheticEvent("1", "command.completed", "tool-a", "out-a", "")}), Partial, MissingEvent},
		{"additional interaction", syntheticObservation("additional", []EventEvidence{syntheticEvent("1", "command.completed", "tool-a", "out-a", ""), syntheticEvent("2", "tool.completed", "tool-b", "out-b", ""), syntheticEvent("3", "tool.completed", "tool-c", "out-c", "")}), Partial, AdditionalEvent},
		{"reordered interaction", syntheticObservation("reordered", []EventEvidence{syntheticEvent("1", "tool.completed", "tool-b", "out-b", ""), syntheticEvent("2", "command.completed", "tool-a", "out-a", "")}), Partial, ReorderedEvent},
		{"workspace divergence", syntheticObservation("workspace", []EventEvidence{syntheticEvent("1", "command.completed", "tool-a", "out-a", "workspace-other"), syntheticEvent("2", "tool.completed", "tool-b", "out-b", "")}), Partial, WorkspaceDifference},
	}
	baseline.Events[0].WorkspaceSHA256 = "workspace-base"
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.name == "identical" {
				test.observation = cloneObservation(baseline, "identical")
			}
			comparison, err := Compare(baseline, test.observation)
			if err != nil || comparison.Status != test.status {
				t.Fatal(comparison.Status, err, comparison.Summary)
			}
			if test.kind != "" && !hasFinding(comparison, test.kind) {
				t.Fatalf("missing finding %s: %#v", test.kind, comparison.Findings)
			}
			for _, finding := range comparison.Findings {
				if finding.Causality != "unknown" {
					t.Fatal("causality invented", finding)
				}
			}
		})
	}
}

func TestIncompleteUnsupportedRepeatedAndPolicy(t *testing.T) {
	baseline := syntheticObservation("base", []EventEvidence{syntheticEvent("1", "command.completed", "tool", "out", "")})
	incomplete := cloneObservation(baseline, "incomplete")
	incomplete.SourceHistory = "partial"
	incomplete.Missing = append(incomplete.Missing, Gap{Dimension: Ordering, Reason: "test gap"})
	comparison, err := Compare(baseline, incomplete)
	if err != nil || comparison.Status != Insufficient {
		t.Fatal(comparison.Status, err)
	}
	unsupported := cloneObservation(baseline, "unsupported")
	unsupported.Unsupported = append(unsupported.Unsupported, Gap{Dimension: EventType, Reason: "test version"})
	comparison, err = Compare(baseline, unsupported)
	if err != nil || comparison.Status != Unsupported {
		t.Fatal(comparison.Status, err)
	}
	observer := &memoryObserver{values: map[string]Observation{
		"a": baseline,
		"b": cloneObservation(baseline, "repeat-b"),
		"c": syntheticObservation("repeat-c", []EventEvidence{syntheticEvent("1", "command.completed", "tool", "changed", "")}),
	}}
	experiment, err := Run(context.Background(), observer, []string{"a", "b", "c"})
	if err != nil || len(experiment.Observations) != 3 || len(experiment.Comparisons) != 2 || experiment.Aggregate.Exact != 1 || experiment.Aggregate.Partial != 1 {
		t.Fatal(experiment.Aggregate, err)
	}
	policy := DefaultPolicy()
	evaluation, err := Evaluate(experiment, policy)
	if err != nil || evaluation.Passed || evaluation.Decision != "fail" || len(evaluation.Rules) == 0 {
		t.Fatal(evaluation, err)
	}
	policy.RequiredDimensions = append(policy.RequiredDimensions, Request)
	evaluation, err = Evaluate(experiment, policy)
	if err != nil || evaluation.Passed || evaluation.Decision != "incomparable" {
		t.Fatal(evaluation, err)
	}
}

func TestBuilderHashesRecordedEvidenceWithoutRawPayload(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private")
	store, err := storage.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	runID := model.ID()
	run := model.Run{SchemaVersion: 1, ID: runID, Workspace: "/private/workspace-name", Adapter: "codex", AdapterVersion: "test-adapter", RecorderVersion: "test-recorder", Sandbox: "read-only", State: model.Created, CreatedAt: model.Now(), TimeoutSeconds: 30}
	if err = store.Create(run); err != nil {
		t.Fatal(err)
	}
	events := []model.Event{
		recordedEvent(runID, "1", "execution.state", model.Payload{Status: model.Created}),
		recordedEvent(runID, "2", "execution.state", model.Payload{Status: model.Starting}),
		recordedEvent(runID, "3", "execution.state", model.Payload{Status: model.Running}),
		recordedEvent(runID, "4", "command.completed", model.Payload{ItemID: "private-item", Command: "private-command", Output: "private-output", ExitCode: intPtr(0)}),
		recordedEvent(runID, "5", "workspace.changed", model.Payload{Changes: []model.Change{{Path: "secret-name.txt", Operation: "modified", After: &model.FileMeta{Size: 3, SHA256: strings.Repeat("a", 64), Mode: 0600}}}}),
		recordedEvent(runID, "6", "execution.state", model.Payload{Status: model.Completed}),
	}
	if err = store.AppendBatch(context.Background(), events); err != nil {
		t.Fatal(err)
	}
	observation, err := (Builder{History: store, Snapshots: store}).Observe(context.Background(), runID)
	if err != nil || len(observation.Events) != len(events) || observation.Configuration.WorkspaceSHA256 == "" {
		t.Fatal(observation, err)
	}
	raw, _ := json.Marshal(observation)
	for _, secret := range []string{"private-item", "private-command", "private-output", "secret-name.txt", "/private/workspace-name"} {
		if strings.Contains(string(raw), secret) {
			t.Fatal("raw recorded content escaped into observation", secret)
		}
	}
}

func TestReportPersistenceReevaluationRestartAndCorruption(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	history := &emptyHistory{}
	service, err := New(root, history, nil)
	if err != nil {
		t.Fatal(err)
	}
	base := syntheticObservation("base", []EventEvidence{syntheticEvent("1", "command.completed", "tool", "out", "")})
	experiment := Experiment{FormatVersion: FormatVersion, Observations: []Observation{base, cloneObservation(base, "other")}}
	comparison, _ := Compare(experiment.Observations[0], experiment.Observations[1])
	experiment.Comparisons = []Comparison{comparison}
	experiment.Aggregate = aggregate(experiment)
	report, err := service.save(context.Background(), experiment, DefaultPolicy())
	if err != nil || !report.Evaluation.Passed {
		t.Fatal(report.Evaluation, err)
	}
	loose := DefaultPolicy()
	loose.Name = "revised-policy"
	revised, err := service.Reevaluate(context.Background(), report.ReportID, loose)
	if err != nil || revised.ReportID == report.ReportID || revised.Experiment.Observations[0].ObservationID != report.Experiment.Observations[0].ObservationID {
		t.Fatal(revised, err)
	}
	service.Close()
	service, err = New(root, history, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if err = os.WriteFile(filepath.Join(root, "analysis", "v1", "reports", ".interrupted.tmp"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	summaries, err := service.List(context.Background())
	if err != nil || len(summaries) != 2 {
		t.Fatal(summaries, err)
	}
	path := filepath.Join(root, reportName(report.ReportID))
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Get(context.Background(), report.ReportID); !errorCode(err, "CORRUPT_REPORT") {
		t.Fatal("noncanonical report accepted", err)
	}
}

func syntheticObservation(id string, events []EventEvidence) Observation {
	return Observation{FormatVersion: FormatVersion, ObservationID: id, SourceExecutionID: id, SourceEventVersion: model.EventVersion, SourceHighWater: strconvItoa(len(events)), SourceHistory: "complete", Configuration: Configuration{Adapter: "synthetic", AdapterVersion: "v1", RecorderVersion: "v1", Sandbox: "none", WorkspaceSHA256: "workspace"}, Events: events, Snapshots: []SnapshotEvidence{}, Missing: []Gap{}, Unsupported: []Gap{}}
}

func syntheticEvent(sequence, kind, tool, output, workspace string) EventEvidence {
	return EventEvidence{Sequence: sequence, EventID: "event-" + sequence, Kind: kind, InputSHA256: "input-" + tool, OutputSHA256: output, ToolSHA256: tool, WorkspaceSHA256: workspace, EvidenceSHA256: hashText("evidence", kind+"|"+tool+"|"+output+"|"+workspace), Missing: []Gap{}}
}

func cloneObservation(value Observation, id string) Observation {
	raw, _ := json.Marshal(value)
	var cloned Observation
	_ = json.Unmarshal(raw, &cloned)
	cloned.ObservationID = id
	cloned.SourceExecutionID = id
	return cloned
}

func hasFinding(comparison Comparison, kind FindingKind) bool {
	for _, finding := range comparison.Findings {
		if finding.Kind == kind {
			return true
		}
	}
	return false
}

type memoryObserver struct{ values map[string]Observation }

func (m *memoryObserver) Observe(_ context.Context, id string) (Observation, error) {
	value, ok := m.values[id]
	if !ok {
		return Observation{}, errors.New("missing")
	}
	return value, nil
}

type emptyHistory struct{}

func (*emptyHistory) GetRun(context.Context, string) (model.Run, error) {
	return model.Run{}, errors.New("unused")
}
func (*emptyHistory) QueryEvents(context.Context, storage.EventQuery) (storage.EventPage, error) {
	return storage.EventPage{}, errors.New("unused")
}
func (*emptyHistory) GetEvent(context.Context, string, string) (model.Event, error) {
	return model.Event{}, errors.New("unused")
}

func recordedEvent(runID, sequence, kind string, payload model.Payload) model.Event {
	return model.Event{SchemaVersion: model.EventVersion, EventID: model.ID(), RunID: runID, Sequence: sequence, Kind: kind, ObservedAt: model.Now(), Adapter: "codex", Payload: payload}
}
func intPtr(value int) *int        { return &value }
func strconvItoa(value int) string { return fmt.Sprintf("%d", value) }
func errorCode(err error, code string) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Code == code
}
