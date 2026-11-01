package fixtures

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/replay"
	"chronosync.local/engine/internal/replay/transport"
	"chronosync.local/engine/internal/storage"
)

type evaluatorFunc func(context.Context, replay.Input) (replay.Actual, error)

func (function evaluatorFunc) Evaluate(ctx context.Context, input replay.Input) (replay.Actual, error) {
	return function(ctx, input)
}

func fixtureHistory(t *testing.T, root string) (*storage.Store, model.Run, []byte) {
	t.Helper()
	store, err := storage.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	run := model.Run{Adapter: "codex", RecorderVersion: "test", SchemaVersion: 1, ID: model.ID(), Workspace: "/synthetic", AdapterVersion: "synthetic", Sandbox: "read-only", State: model.Created, CreatedAt: model.Now(), TimeoutSeconds: 30}
	if err = store.Create(run); err != nil {
		t.Fatal(err)
	}
	events := []struct {
		kind    string
		payload model.Payload
	}{
		{"execution.state", model.Payload{Status: model.Created}},
		{"execution.state", model.Payload{Status: model.Starting}},
		{"execution.state", model.Payload{Status: model.Running}},
		{replay.InteractionType, model.Payload{ItemID: "item", Command: "safe-command", Output: "safe-output", ExitCode: intPointer(0)}},
		{"agent.turn_completed", model.Payload{UpstreamType: "turn.completed"}},
		{"execution.state", model.Payload{Status: model.Completed}},
	}
	for index, item := range events {
		event := model.Event{SchemaVersion: model.EventVersion, EventID: model.ID(), RunID: run.ID, Sequence: strconv.Itoa(index + 1), Kind: item.kind, ObservedAt: model.Now(), Adapter: "codex", Payload: item.payload}
		if err = store.Append(event); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := (replay.Exporter{Source: store}).Export(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	return store, run, raw
}

func TestFixtureLifecycleRestartDuplicateExportAndDelete(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private")
	store, run, raw := fixtureHistory(t, root)
	service, err := New(root, store)
	if err != nil {
		t.Fatal(err)
	}
	record, err := service.Eligibility(context.Background(), run.ID)
	if err != nil || record.Kind != RegressionV1 || record.Interactions != 1 {
		t.Fatal(record, err)
	}
	record, err = service.CreateFromRun(context.Background(), run.ID)
	if err != nil || record.Duplicate || record.ID != digest(raw) {
		t.Fatal(record, err)
	}
	duplicate, err := service.Import(context.Background(), raw)
	if err != nil || !duplicate.Duplicate || duplicate.ID != record.ID {
		t.Fatal(duplicate, err)
	}
	if _, _, err = service.Export(context.Background(), record.ID, false); !fixtureErrorIs(err, "CONSENT_REQUIRED") {
		t.Fatal("export without consent", err)
	}
	exported, inspected, err := service.Export(context.Background(), record.ID, true)
	if err != nil || inspected.Integrity != "verified" || string(exported) != string(raw) || len(inspected.Warnings) == 0 {
		t.Fatal(inspected, err)
	}
	exported[0] = 'x'
	again, _, err := service.Export(context.Background(), record.ID, true)
	if err != nil || again[0] == 'x' {
		t.Fatal("export aliases storage", err)
	}
	if err = service.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(root, store)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	list, err := reopened.List(context.Background())
	if err != nil || len(list) != 1 || list[0].ID != record.ID {
		t.Fatal(list, err)
	}
	if err = reopened.Delete(context.Background(), record.ID, false); !fixtureErrorIs(err, "CONSENT_REQUIRED") {
		t.Fatal("delete without consent", err)
	}
	if err = reopened.Delete(context.Background(), record.ID, true); err != nil {
		t.Fatal(err)
	}
	list, err = reopened.List(context.Background())
	if err != nil || len(list) != 0 {
		t.Fatal(list, err)
	}
	store.Close()
}

func TestFixtureSuccessfulEvaluationAndTransportImport(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private")
	store, _, raw := fixtureHistory(t, root)
	defer store.Close()
	service, err := New(root, store)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	fixture, err := replay.Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	result, err := replay.Evaluate(context.Background(), raw, evaluatorFunc(func(context.Context, replay.Input) (replay.Actual, error) {
		return replay.Actual{OutputSHA256: fixture.Interactions[0].Expected.OutputSHA256, ExitCode: fixture.Interactions[0].Expected.ExitCode}, nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.CreateFromEvaluation(context.Background(), raw, replay.Result{}); err == nil {
		t.Fatal("failed evaluation persisted")
	}
	if _, err = service.CreateFromEvaluation(context.Background(), raw, result); err != nil {
		t.Fatal(err)
	}
	identity := transport.TransportIdentity{Name: "fixture", Protocol: "synthetic-rpc", Version: "1"}
	recording, err := transport.NewTransportRecording([]transport.TransportInteractionDraft{{
		Sequence: 1, Occurrence: 1,
		Request:  transport.SyntheticRequest{Transport: identity, Operation: "tools.read"},
		Response: transport.SyntheticResponse{Status: 200},
	}})
	if err != nil {
		t.Fatal(err)
	}
	transportRaw, err := recording.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	imported, err := service.Import(context.Background(), transportRaw)
	if err != nil || imported.Kind != TransportV1 || imported.Interactions != 1 {
		t.Fatal(imported, err)
	}
}

func TestFixtureRejectsTraversalPartialUnsupportedAndCorruption(t *testing.T) {
	root := filepath.Join(t.TempDir(), "private")
	store, run, _ := fixtureHistory(t, root)
	defer store.Close()
	service, err := New(root, store)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err = service.Import(context.Background(), []byte(`{"unknown":true}`)); !fixtureErrorIs(err, "UNSUPPORTED_FIXTURE") {
		t.Fatal("unsupported fixture imported", err)
	}
	if _, err = service.Inspect(context.Background(), "../../history.sqlite"); !fixtureErrorIs(err, "INVALID_ARGUMENT") {
		t.Fatal("traversal identity accepted", err)
	}
	if err = os.WriteFile(filepath.Join(root, "fixtures", "v1", "sha256", ".partial.tmp"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if list, listErr := service.List(context.Background()); listErr != nil || len(list) != 0 {
		t.Fatal("partial file treated as fixture", list, listErr)
	}
	record, err := service.CreateFromRun(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, fixtureName(record.ID))
	if err = os.WriteFile(path, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = service.Inspect(context.Background(), record.ID); !fixtureErrorIs(err, "CORRUPT_FIXTURE") {
		t.Fatal("corrupt fixture accepted", err)
	}
	list, err := service.List(context.Background())
	if err != nil || len(list) != 1 || list[0].Kind != Damaged || list[0].Integrity != "damaged" {
		t.Fatal("corruption hidden from inventory", list, err)
	}
	if err = service.Delete(context.Background(), record.ID, true); err != nil {
		t.Fatal("explicit corrupt fixture deletion failed", err)
	}
}

func fixtureErrorIs(err error, code string) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Code == code
}

func intPointer(value int) *int { return &value }
