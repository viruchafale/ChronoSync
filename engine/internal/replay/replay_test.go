package replay

import (
	"context"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/storage"
)

type evaluatorFunc func(context.Context, Input) (Actual, error)

func (f evaluatorFunc) Evaluate(ctx context.Context, input Input) (Actual, error) {
	return f(ctx, input)
}

func errorIs(err error, code string) bool {
	var typedErr *Error
	return errors.As(err, &typedErr) && typedErr.Code == code
}

type unavailableDetailSource struct{ HistorySource }

func (unavailableDetailSource) GetEvent(context.Context, string, string) (model.Event, error) {
	return model.Event{}, errors.New("synthetic unavailable detail")
}

func appendEvent(t *testing.T, store *storage.Store, run model.Run, sequence int, kind string, payload model.Payload) model.Event {
	t.Helper()
	event := model.Event{SchemaVersion: model.EventVersion, EventID: model.ID(), RunID: run.ID, Sequence: strconv.Itoa(sequence), Kind: kind, ObservedAt: model.Now(), Adapter: "codex", Payload: payload}
	if err := store.Append(event); err != nil {
		t.Fatal(err)
	}
	return event
}

func completedHistory(t *testing.T, commands int, command, output string) (*storage.Store, model.Run) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	run := model.Run{Adapter: "codex", RecorderVersion: "test", SchemaVersion: 1, ID: model.ID(), Workspace: "/synthetic", AdapterVersion: "codex-cli 0.163.0", Sandbox: "read-only", State: model.Created, CreatedAt: model.Now(), TimeoutSeconds: 30}
	if err = store.Create(run); err != nil {
		t.Fatal(err)
	}
	appendEvent(t, store, run, 1, "execution.state", model.Payload{Status: model.Created})
	appendEvent(t, store, run, 2, "execution.state", model.Payload{Status: model.Starting})
	appendEvent(t, store, run, 3, "execution.state", model.Payload{Status: model.Running})
	sequence := 4
	for n := range commands {
		exit := n
		appendEvent(t, store, run, sequence, InteractionType, model.Payload{ItemID: "item-" + strconv.Itoa(n), Command: command + strconv.Itoa(n), Output: output + strconv.Itoa(n), ExitCode: &exit, UpstreamType: "item.completed:command_execution"})
		sequence++
	}
	appendEvent(t, store, run, sequence, "agent.turn_completed", model.Payload{UpstreamType: "turn.completed"})
	sequence++
	appendEvent(t, store, run, sequence, "execution.state", model.Payload{Status: model.Completed, Reason: "turn_and_process_completed"})
	return store, run
}

func exportFixture(t *testing.T, commands int) ([]byte, Fixture) {
	t.Helper()
	store, run := completedHistory(t, commands, "printf-safe-", "observed-output-")
	raw, err := (Exporter{Source: store}).Export(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, fixture
}

func TestExportStableValidateAndEvaluate(t *testing.T) {
	store, run := completedHistory(t, 2, "printf-safe-", "observed-output-")
	exporter := Exporter{Source: store}
	first, err := exporter.Export(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	second, err := exporter.Export(context.Background(), run.ID)
	if err != nil || string(first) != string(second) {
		t.Fatal("export is not stable", err)
	}
	for _, forbidden := range []string{"printf-safe-", "observed-output-", "item-"} {
		if strings.Contains(string(first), forbidden) {
			t.Fatalf("raw recorded value exported: %q", forbidden)
		}
	}
	fixture, err := Load(first)
	if err != nil || len(fixture.Interactions) != 2 || fixture.SourceEventCount != 7 {
		t.Fatal(fixture, err)
	}
	called := 0
	evaluator := evaluatorFunc(func(_ context.Context, input Input) (Actual, error) {
		called++
		if strings.Contains(input.CommandSHA256, "printf-safe") || strings.Contains(input.ItemIDSHA256, "item-") {
			t.Fatal("raw historical input crossed replay boundary")
		}
		n := called - 1
		return Actual{OutputSHA256: digest([]byte("observed-output-" + strconv.Itoa(n))), ExitCode: n}, nil
	})
	result, err := Evaluate(context.Background(), first, evaluator)
	if err != nil || result.Status != Completed || !result.Passed || result.Evaluated != 2 || called != 2 || len(result.Mismatches) != 0 {
		t.Fatal(result, err)
	}
	saved, err := RegressionFixture(first, result)
	if err != nil || string(saved) != string(first) {
		t.Fatal("successful regression fixture was not preserved", err)
	}
	saved[0] = 'x'
	if first[0] == 'x' {
		t.Fatal("regression fixture aliases caller bytes")
	}
	again, err := Evaluate(context.Background(), first, evaluatorFunc(func(_ context.Context, input Input) (Actual, error) {
		for n, interaction := range fixture.Interactions {
			if interaction.Input == input {
				return Actual{OutputSHA256: digest([]byte("observed-output-" + strconv.Itoa(n))), ExitCode: n}, nil
			}
		}
		return Actual{}, errors.New("unexpected input")
	}))
	if err != nil || !reflect.DeepEqual(result, again) {
		t.Fatal("evaluation result is not stable", result, again, err)
	}
}

func TestExportReadsIntegrityCheckedExternalPayload(t *testing.T) {
	output := strings.Repeat("external-payload-", 800)
	store, run := completedHistory(t, 1, "safe-command-", output)
	raw, err := (Exporter{Source: store}).Export(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := Load(raw)
	if err != nil || fixture.Interactions[0].Expected.OutputSHA256 != digest([]byte(output+"0")) || strings.Contains(string(raw), "external-payload") {
		t.Fatal("external payload was not safely fingerprinted", err)
	}
}

func TestMismatchFailureCancellationAndUnsupported(t *testing.T) {
	raw, fixture := exportFixture(t, 1)
	wrong, err := Evaluate(context.Background(), raw, evaluatorFunc(func(context.Context, Input) (Actual, error) {
		return Actual{OutputSHA256: digest([]byte("different")), ExitCode: 99}, nil
	}))
	if err != nil || wrong.Status != Completed || wrong.Passed || len(wrong.Mismatches) != 2 || wrong.Mismatches[0].Field != "output_sha256" || wrong.Mismatches[1].Field != "exit_code" {
		t.Fatal(wrong, err)
	}
	if _, err = RegressionFixture(raw, wrong); !errorIs(err, "EVALUATION_NOT_PASSED") {
		t.Fatal("failed evaluation saved as regression fixture", err)
	}
	failed, err := Evaluate(context.Background(), raw, evaluatorFunc(func(context.Context, Input) (Actual, error) {
		return Actual{}, errors.New("synthetic evaluator failure with private detail")
	}))
	if err != nil || failed.Status != Failed || strings.Contains(failed.Summary, "private detail") {
		t.Fatal(failed, err)
	}
	unsupported, err := Evaluate(context.Background(), raw, evaluatorFunc(func(context.Context, Input) (Actual, error) {
		return Actual{}, &UnsupportedError{Reason: "controlled fixture has no matching fingerprint"}
	}))
	if err != nil || unsupported.Status != Unsupported || len(unsupported.Unsupported) != 1 || unsupported.Unsupported[0].SourceSequence != fixture.Interactions[0].SourceSequence {
		t.Fatal(unsupported, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cancelled, err := Evaluate(ctx, raw, evaluatorFunc(func(context.Context, Input) (Actual, error) {
		t.Fatal("evaluator called after cancellation")
		return Actual{}, nil
	}))
	if err != nil || cancelled.Status != Cancelled || cancelled.Evaluated != 0 {
		t.Fatal(cancelled, err)
	}
	invalid, err := Evaluate(context.Background(), raw, evaluatorFunc(func(context.Context, Input) (Actual, error) {
		return Actual{OutputSHA256: "not-a-digest"}, nil
	}))
	if err != nil || invalid.Status != Failed {
		t.Fatal(invalid, err)
	}
}

func TestFixtureIntegrityVersionsOrderingAndBounds(t *testing.T) {
	raw, fixture := exportFixture(t, 2)
	tampered := append([]byte(nil), raw...)
	for i := range tampered {
		if tampered[i] == 'a' {
			tampered[i] = 'b'
			break
		}
	}
	if _, err := Load(tampered); err == nil {
		t.Fatal("altered fixture accepted")
	}
	var badDigest Envelope
	if err := json.Unmarshal(raw, &badDigest); err != nil {
		t.Fatal(err)
	}
	badDigest.SHA256 = strings.Repeat("0", 64)
	invalidDigest, _ := json.Marshal(badDigest)
	if _, err := Load(invalidDigest); !errorIs(err, "INTEGRITY_MISMATCH") {
		t.Fatal("invalid envelope digest accepted", err)
	}
	if _, err := Load(append(raw, '\n')); !errorIs(err, "NON_CANONICAL_FIXTURE") {
		t.Fatal("noncanonical fixture accepted", err)
	}
	if _, err := Load([]byte(`{"fixture":`)); !errorIs(err, "INVALID_FIXTURE") {
		t.Fatal("malformed fixture accepted", err)
	}
	if _, err := Load(make([]byte, MaxFixtureBytes+1)); !errorIs(err, "FIXTURE_TOO_LARGE") {
		t.Fatal("oversized fixture accepted", err)
	}
	for name, mutate := range map[string]func(*Fixture){
		"version":    func(f *Fixture) { f.FormatVersion++ },
		"unknown":    func(f *Fixture) { f.Interactions[0].Type = "provider.response" },
		"missing":    func(f *Fixture) { f.Interactions = nil },
		"reordered":  func(f *Fixture) { f.Interactions[0], f.Interactions[1] = f.Interactions[1], f.Interactions[0] },
		"duplicated": func(f *Fixture) { f.Interactions[1] = f.Interactions[0] },
		"sequence":   func(f *Fixture) { f.Interactions[0].Input.SourceSequence = "999" },
		"digest":     func(f *Fixture) { f.Interactions[0].Expected.OutputSHA256 = "bad" },
		"range":      func(f *Fixture) { f.SourceEventCount-- },
	} {
		t.Run(name, func(t *testing.T) {
			changed := fixture
			changed.SupportedTypes = append([]string(nil), fixture.SupportedTypes...)
			changed.Interactions = append([]Interaction(nil), fixture.Interactions...)
			mutate(&changed)
			changed.ReplayID = ""
			changedRaw, sealErr := seal(changed)
			if sealErr != nil {
				// Oversized or structurally impossible mutations may be rejected while sealing.
				return
			}
			if _, loadErr := Load(changedRaw); loadErr == nil {
				t.Fatal("invalid mutation accepted")
			}
		})
	}
	var envelope map[string]any
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	envelope["unknown"] = true
	unknown, _ := json.Marshal(envelope)
	if _, err := Load(unknown); !errorIs(err, "INVALID_FIXTURE") {
		t.Fatal("unknown field accepted", err)
	}
}

func TestExportRejectsUnsupportedIncompleteAndSensitiveHistory(t *testing.T) {
	store, err := storage.Open(filepath.Join(t.TempDir(), "incomplete"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	incomplete := model.Run{Adapter: "codex", SchemaVersion: 1, ID: model.ID(), State: model.Created, CreatedAt: model.Now()}
	if err = store.Create(incomplete); err != nil {
		t.Fatal(err)
	}
	if _, err = (Exporter{Source: store}).Export(context.Background(), incomplete.ID); !errorIs(err, "INCOMPLETE_HISTORY") {
		t.Fatal("active history exported", err)
	}

	unsupportedStore, unsupportedRun := completedHistoryWithPayload(t, "agent.message", model.Payload{Message: "not replayable"})
	if _, err = (Exporter{Source: unsupportedStore}).Export(context.Background(), unsupportedRun.ID); !errorIs(err, "UNSUPPORTED_EVENT") {
		t.Fatal("unsupported event exported", err)
	}
	for name, payload := range map[string]model.Payload{
		"needs-redaction": {ItemID: "item", Command: "api_key=synthetic-secret", Output: "ok", ExitCode: intPointer(0)},
		"marked-redacted": {ItemID: "item", Command: "echo [REDACTED]", Output: "ok", ExitCode: intPointer(0)},
		"truncated":       {ItemID: "item", Command: "echo safe", Output: "partial", ExitCode: intPointer(0), Truncations: []model.Truncation{{Field: "output", OriginalBytes: 100, LimitBytes: 10}}},
		"missing":         {ItemID: "item", Command: "echo safe", Output: "ok"},
	} {
		t.Run(name, func(t *testing.T) {
			sensitiveStore, sensitiveRun := completedHistoryWithPayload(t, InteractionType, payload)
			if _, exportErr := (Exporter{Source: sensitiveStore}).Export(context.Background(), sensitiveRun.ID); exportErr == nil {
				t.Fatal("ineligible payload exported")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	validStore, validRun := completedHistory(t, 1, "safe-", "output-")
	if _, err = (Exporter{Source: unavailableDetailSource{HistorySource: validStore}}).Export(context.Background(), validRun.ID); !errorIs(err, "SOURCE_UNAVAILABLE") {
		t.Fatal("unavailable event detail accepted", err)
	}
	if _, err = (Exporter{Source: validStore}).Export(ctx, validRun.ID); !errorIs(err, "CANCELLED") {
		t.Fatal("cancelled export continued", err)
	}
}

func completedHistoryWithPayload(t *testing.T, kind string, payload model.Payload) (*storage.Store, model.Run) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "private"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	run := model.Run{Adapter: "codex", SchemaVersion: 1, ID: model.ID(), State: model.Created, CreatedAt: model.Now()}
	if err = store.Create(run); err != nil {
		t.Fatal(err)
	}
	appendEvent(t, store, run, 1, "execution.state", model.Payload{Status: model.Created})
	appendEvent(t, store, run, 2, "execution.state", model.Payload{Status: model.Starting})
	appendEvent(t, store, run, 3, "execution.state", model.Payload{Status: model.Running})
	appendEvent(t, store, run, 4, kind, payload)
	appendEvent(t, store, run, 5, "execution.state", model.Payload{Status: model.Completed})
	return store, run
}

func intPointer(value int) *int { return &value }

func TestReplayPackageHasNoExecutionOrNetworkImports(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller unavailable")
	}
	directory := filepath.Dir(file)
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]bool{
		"net": true, "net/http": true, "os/exec": true,
		"chronosync.local/engine/internal/adapter":    true,
		"chronosync.local/engine/internal/supervisor": true,
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
			if forbidden[path] || strings.HasPrefix(path, "net/") {
				t.Fatalf("forbidden replay import %q in %s", path, entry.Name())
			}
		}
	}
}
