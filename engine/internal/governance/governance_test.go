package governance

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/model"
)

func TestDatasetCreationImportCompatibilityAndReferenceValidation(t *testing.T) {
	root := privateRoot(t)
	reports := newFakeReports(t)
	service, err := New(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()

	first := createDataset(t, service, reports.first, "case-a", "")
	raw, _, err := service.ExportDataset(context.Background(), first.ContentID, true)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := service.ImportDataset(context.Background(), raw)
	if err != nil || !duplicate.Duplicate || duplicate.ContentID != first.ContentID {
		t.Fatal(duplicate, err)
	}
	secondSpec := datasetSpec(reports.first, "case-a")
	secondSpec.BenchmarkID = first.BenchmarkID
	secondSpec.Version = 2
	secondSpec.Cases = append(secondSpec.Cases, CaseDraft{CaseID: "case-b", SourceReportID: reports.first, ComparisonIndex: 0, SupportedDimensions: []analysis.Dimension{analysis.Output}})
	second, err := service.CreateDataset(context.Background(), secondSpec)
	if err != nil {
		t.Fatal(err)
	}
	compatible, err := service.CompareDatasetVersions(context.Background(), first.ContentID, second.ContentID)
	if err != nil || compatible.Status != CompatibilityExtension || len(compatible.AddedCases) != 1 {
		t.Fatal(compatible, err)
	}
	thirdSpec := datasetSpec(reports.second, "case-a")
	thirdSpec.BenchmarkID = first.BenchmarkID
	thirdSpec.Version = 3
	thirdSpec.Cases = append(thirdSpec.Cases, CaseDraft{CaseID: "case-b", SourceReportID: reports.first, ComparisonIndex: 0, SupportedDimensions: []analysis.Dimension{analysis.Output}})
	third, err := service.CreateDataset(context.Background(), thirdSpec)
	if err != nil {
		t.Fatal(err)
	}
	incompatible, err := service.CompareDatasetVersions(context.Background(), second.ContentID, third.ContentID)
	if err != nil || incompatible.Status != CompatibilityIncompatible || len(incompatible.ChangedCases) != 1 {
		t.Fatal(incompatible, err)
	}
	conflictSpec := secondSpec
	conflictSpec.Name = "changed definition"
	if _, err = service.CreateDataset(context.Background(), conflictSpec); !errorIs(err, "VERSION_CONFLICT") {
		t.Fatal("identity/version conflict accepted", err)
	}

	missingDataset, _, _ := decodeDataset(raw)
	missingDataset.BenchmarkID = model.ID()
	missingDataset.Cases[0].SourceReportID = model.ID()
	missingRaw, _, _ := sealDataset(missingDataset)
	if _, err = service.ImportDataset(context.Background(), missingRaw); !errorIs(err, "MISSING_REFERENCE") {
		t.Fatal("missing report reference accepted", err)
	}
	var envelope DatasetEnvelope
	_ = json.Unmarshal(raw, &envelope)
	envelope.Dataset.FormatVersion = 99
	body, _ := json.Marshal(envelope.Dataset)
	envelope.SHA256 = digest(body)
	unsupportedRaw, _ := json.Marshal(envelope)
	if _, err = service.ImportDataset(context.Background(), unsupportedRaw); !errorIs(err, "UNSUPPORTED_DATASET") {
		t.Fatal("unsupported version accepted", err)
	}
	corrupt := append([]byte(nil), raw...)
	corrupt[len(corrupt)-2] ^= 1
	if _, err = service.ImportDataset(context.Background(), corrupt); err == nil {
		t.Fatal("corrupted dataset accepted")
	}
}

func TestLineageRetentionAuditDeletionAndRestart(t *testing.T) {
	root := privateRoot(t)
	reports := newFakeReports(t)
	service, err := New(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	dataset := createDataset(t, service, reports.first, "case-a", "")
	lineage, err := service.EvaluateDataset(context.Background(), dataset.ContentID, analysis.DefaultPolicy(), "")
	if err != nil || len(lineage.EvaluationReportIDs) != 1 {
		t.Fatal(lineage, err)
	}
	if _, err = reports.Get(context.Background(), reports.first); err != nil {
		t.Fatal("historical source report was rewritten", err)
	}
	policy := DefaultRetentionPolicy()
	if _, err = service.SetRetention(context.Background(), ReportObject, reports.first, Active, policy, "manage historical report"); err != nil {
		t.Fatal(err)
	}
	if err = service.Delete(context.Background(), ReportObject, reports.first, true); !errorIs(err, "REFERENCED") {
		t.Fatal("referenced report deletion accepted", err)
	}
	if err = service.Delete(context.Background(), DatasetObject, dataset.ContentID, true); !errorIs(err, "REFERENCED") {
		t.Fatal("referenced dataset deletion accepted", err)
	}
	audit, err := service.Audit(context.Background())
	if err != nil || audit.Usage.Datasets != 1 || audit.Usage.Lineages != 1 || audit.Usage.Reports < 3 || len(audit.References) < 3 || audit.Usage.TotalBytes == 0 {
		t.Fatal(audit, err)
	}
	service.Close()
	service, err = New(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	if _, err = service.GetDataset(context.Background(), dataset.ContentID); err != nil {
		t.Fatal("dataset unavailable after restart", err)
	}
	if _, err = service.GetLineage(context.Background(), lineage.LineageID); err != nil {
		t.Fatal("lineage unavailable after restart", err)
	}
	if err = service.Delete(context.Background(), LineageObject, lineage.LineageID, true); err != nil {
		t.Fatal(err)
	}
	if err = service.Delete(context.Background(), DatasetObject, dataset.ContentID, true); err != nil {
		t.Fatal(err)
	}
	if err = service.Delete(context.Background(), ReportObject, reports.first, true); err != nil {
		t.Fatal(err)
	}
	status := service.statusOrDefault(ReportObject, reports.first)
	if status.State != Deleted || status.Revision != 3 {
		t.Fatal(status)
	}
}

func TestDamageAccountingInterruptedWriteAndExplicitRecovery(t *testing.T) {
	root := privateRoot(t)
	reports := newFakeReports(t)
	service, err := New(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	dataset := createDataset(t, service, reports.first, "case-a", "")
	tempPath := filepath.Join(root, "governance", "v1", "datasets", "sha256", ".interrupted.tmp")
	if err = os.WriteFile(tempPath, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	datasetPath := filepath.Join(root, datasetName(dataset.ContentID))
	if err = os.WriteFile(datasetPath, []byte("corrupt"), 0600); err != nil {
		t.Fatal(err)
	}
	records, err := service.ListDatasets(context.Background())
	if err != nil || len(records) != 1 || records[0].Integrity != "damaged" {
		t.Fatal(records, err)
	}
	audit, err := service.Audit(context.Background())
	if err != nil || audit.Usage.DamagedObjects == 0 || audit.Usage.TemporaryFiles == 0 {
		t.Fatal(audit, err)
	}
	if _, err = service.SetRetention(context.Background(), DatasetObject, dataset.ContentID, Damaged, DefaultRetentionPolicy(), "integrity audit failed"); err != nil {
		t.Fatal(err)
	}
	if err = service.Delete(context.Background(), DatasetObject, dataset.ContentID, true); err != nil {
		t.Fatal("explicit damaged dataset deletion failed", err)
	}
}

func TestConcurrentDuplicateImportIsBounded(t *testing.T) {
	root := privateRoot(t)
	reports := newFakeReports(t)
	service, err := New(root, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	created := createDataset(t, service, reports.first, "case-a", "")
	raw, _, _ := service.ExportDataset(context.Background(), created.ContentID, true)
	service.Delete(context.Background(), DatasetObject, created.ContentID, true)
	// Import under a fresh family because deleted identities remain historical.
	dataset, _, _ := decodeDataset(raw)
	dataset.BenchmarkID = model.ID()
	raw, _, _ = sealDataset(dataset)
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, importErr := service.ImportDataset(context.Background(), raw)
			results <- importErr
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	for result := range results {
		if result == nil {
			successes++
		} else if !errorIs(result, "BUSY") {
			t.Fatal(result)
		}
	}
	if successes < 1 {
		t.Fatal("no concurrent import succeeded")
	}
	records, err := service.ListDatasets(context.Background())
	if err != nil || len(records) != 1 {
		t.Fatal(records, err)
	}
}

func privateRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func datasetSpec(reportID, caseID string) CreateDataset {
	return CreateDataset{Version: 1, Name: "Synthetic benchmark", Cases: []CaseDraft{{CaseID: caseID, SourceReportID: reportID, ComparisonIndex: 0, SupportedDimensions: []analysis.Dimension{analysis.Output}}}, SupportedDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering, analysis.Output}, RequiredDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering}, ProvenanceLimitations: []string{"Synthetic observable evidence only."}}
}

func createDataset(t *testing.T, service *Service, reportID, caseID, benchmarkID string) DatasetRecord {
	t.Helper()
	spec := datasetSpec(reportID, caseID)
	spec.BenchmarkID = benchmarkID
	record, err := service.CreateDataset(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

type fakeReports struct {
	mu     sync.Mutex
	values map[string]analysis.Report
	first  string
	second string
}

func newFakeReports(t *testing.T) *fakeReports {
	t.Helper()
	values := &fakeReports{values: map[string]analysis.Report{}}
	values.first = model.ID()
	values.second = model.ID()
	values.values[values.first] = syntheticReport(t, values.first, "output-a")
	values.values[values.second] = syntheticReport(t, values.second, "output-b")
	return values
}

func syntheticReport(t *testing.T, id, output string) analysis.Report {
	t.Helper()
	base := analysis.Observation{FormatVersion: 1, ObservationID: "observation-base-" + id, SourceExecutionID: model.ID(), SourceEventVersion: model.EventVersion, SourceHighWater: "1", SourceHistory: "complete", Configuration: analysis.Configuration{Adapter: "synthetic", AdapterVersion: "1", RecorderVersion: "1", Sandbox: "none", WorkspaceSHA256: digest([]byte("workspace"))}, Events: []analysis.EventEvidence{{Sequence: "1", EventID: model.ID(), Kind: "command.completed", OutputSHA256: "output-a", EvidenceSHA256: digest([]byte("base")), Missing: []analysis.Gap{}}}, Snapshots: []analysis.SnapshotEvidence{}, Missing: []analysis.Gap{}, Unsupported: []analysis.Gap{}}
	observed := base
	observed.ObservationID = "observation-observed-" + id
	observed.SourceExecutionID = model.ID()
	observed.Events = append([]analysis.EventEvidence(nil), base.Events...)
	observed.Events[0].OutputSHA256 = output
	observed.Events[0].EvidenceSHA256 = digest([]byte(output))
	comparison, err := analysis.Compare(base, observed)
	if err != nil {
		t.Fatal(err)
	}
	experiment := analysis.Experiment{FormatVersion: 1, Observations: []analysis.Observation{base, observed}, Comparisons: []analysis.Comparison{comparison}, Aggregate: analysis.Aggregate{Observations: 2, Comparisons: 1}}
	policy := analysis.DefaultPolicy()
	evaluation, _ := analysis.Evaluate(experiment, policy)
	return analysis.Report{FormatVersion: 1, ReportID: id, CreatedAt: model.Now(), Experiment: experiment, Policy: policy, Evaluation: evaluation, SHA256: digest([]byte(id))}
}

func (f *fakeReports) List(context.Context) ([]analysis.ReportSummary, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := make([]analysis.ReportSummary, 0, len(f.values))
	for id := range f.values {
		result = append(result, analysis.ReportSummary{ReportID: id})
	}
	return result, nil
}

func (f *fakeReports) Get(_ context.Context, id string) (analysis.Report, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.values[id]
	if !ok {
		return analysis.Report{}, errors.New("not found")
	}
	return value, nil
}

func (f *fakeReports) Reevaluate(_ context.Context, id string, policy analysis.Policy) (analysis.Report, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	value, ok := f.values[id]
	if !ok {
		return analysis.Report{}, errors.New("not found")
	}
	value.ReportID = model.ID()
	value.Policy = policy
	value.Evaluation, _ = analysis.Evaluate(value.Experiment, policy)
	f.values[value.ReportID] = value
	return value, nil
}

func (f *fakeReports) DeleteGoverned(_ context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.values[id]; !ok {
		return errors.New("not found")
	}
	delete(f.values, id)
	return nil
}

func errorIs(err error, code string) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Code == code
}
