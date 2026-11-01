package ipc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/governance"
	"chronosync.local/engine/internal/storage"
)

func TestGovernanceWorkflowPersistsAndRefusesReferencedDeletion(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	baseline := addAnalysisRun(t, store, "stable-output", false)
	changed := addAnalysisRun(t, store, "changed-output", true)
	reports, err := analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	report := analysisRequest(t, "analysis.compare", map[string]any{
		"run_ids": []string{baseline, changed},
		"policy":  analysis.DefaultPolicy(),
	}, reports).(analysis.Report)
	service, err := governance.New(base, reports)
	if err != nil {
		t.Fatal(err)
	}
	dataset := governanceRequest(t, "benchmark.create", governance.CreateDataset{
		Version:               1,
		Name:                  "Synthetic benchmark",
		Cases:                 []governance.CaseDraft{{CaseID: "case-001", SourceReportID: report.ReportID, ComparisonIndex: 0, SupportedDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering}}},
		SupportedDimensions:   []analysis.Dimension{analysis.EventType, analysis.Ordering},
		RequiredDimensions:    []analysis.Dimension{analysis.EventType, analysis.Ordering},
		ProvenanceLimitations: []string{"Synthetic recorded observations only."},
	}, service).(governance.DatasetRecord)
	lineage := governanceRequest(t, "benchmark.evaluate", map[string]any{
		"id":                dataset.ContentID,
		"policy":            analysis.DefaultPolicy(),
		"parent_lineage_id": "",
	}, service).(governance.Lineage)
	if len(lineage.EvaluationReportIDs) != 1 || lineage.SourceReportIDs[0] != report.ReportID {
		t.Fatal("lineage did not preserve source and new report identities", lineage)
	}
	_, err = governanceDispatch(governanceIPCRequest(t, "governance.delete", map[string]any{
		"kind": governance.DatasetObject, "id": dataset.ContentID, "consent": true,
	}), service)
	var governanceErr *governance.Error
	if err == nil || !asGovernanceError(err, &governanceErr) || governanceErr.Code != "REFERENCED" {
		t.Fatal("referenced dataset deletion was not refused", err)
	}
	audit := governanceRequest(t, "governance.audit", struct{}{}, service).(governance.Audit)
	if audit.Usage.Datasets != 1 || audit.Usage.Lineages != 1 || audit.Usage.Reports != 2 || len(audit.References) < 4 {
		t.Fatal("unexpected governed accounting", audit.Usage, audit.References)
	}
	service.Close()
	reports.Close()
	store.Close()

	store, err = storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	reports, err = analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	defer reports.Close()
	service, err = governance.New(base, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	loaded := governanceRequest(t, "lineage.get", map[string]string{"id": lineage.LineageID}, service).(governance.Lineage)
	if loaded.DatasetContentID != dataset.ContentID || loaded.EvaluationReportIDs[0] != lineage.EvaluationReportIDs[0] {
		t.Fatal("lineage did not survive restart", loaded)
	}
}

func TestGovernanceAuditsAndExplicitlyDeletesDamagedReport(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	baseline := addAnalysisRun(t, store, "stable-output", false)
	changed := addAnalysisRun(t, store, "changed-output", false)
	reports, err := analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	defer reports.Close()
	report := analysisRequest(t, "analysis.compare", map[string]any{
		"run_ids": []string{baseline, changed},
		"policy":  analysis.DefaultPolicy(),
	}, reports).(analysis.Report)
	service, err := governance.New(base, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	path := filepath.Join(base, "analysis", "v1", "reports", report.ReportID+".json")
	if err = os.WriteFile(path, []byte(`{"damaged":true}`), 0600); err != nil {
		t.Fatal(err)
	}
	audit := governanceRequest(t, "governance.audit", struct{}{}, service).(governance.Audit)
	if audit.Usage.DamagedObjects != 1 {
		t.Fatal("damaged report was not accounted", audit)
	}
	if _, err = service.SetRetention(t.Context(), governance.ReportObject, report.ReportID, governance.Damaged, governance.DefaultRetentionPolicy(), "integrity audit failed"); err != nil {
		t.Fatal(err)
	}
	if err = service.Delete(t.Context(), governance.ReportObject, report.ReportID, true); err != nil {
		t.Fatal(err)
	}
	if _, err = os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("damaged report remains after explicit governed deletion", err)
	}
}

func TestGovernanceIPCRejectsUnknownAuthorityFields(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	reports, err := analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	defer reports.Close()
	service, err := governance.New(base, reports)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()
	raw := json.RawMessage(`{"kind":"report","id":"11111111111111111111111111111111","state":"active","policy":{"format_version":1,"name":"manual","archive_allowed":true,"delete_allowed":true,"automatic":false},"reason":"manual","cascade":true}`)
	if _, err = governanceDispatch(Request{ProtocolVersion: 1, ID: "test", Method: "retention.set", Params: raw}, service); err == nil {
		t.Fatal("unknown cascading-deletion authority accepted")
	}
}

func governanceRequest(t *testing.T, method string, params any, service *governance.Service) any {
	t.Helper()
	result, err := governanceDispatch(governanceIPCRequest(t, method, params), service)
	if err != nil {
		t.Fatal(method, err)
	}
	return result
}

func governanceIPCRequest(t *testing.T, method string, params any) Request {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	return Request{ProtocolVersion: 1, ID: "test", Method: method, Params: raw}
}

func asGovernanceError(err error, target **governance.Error) bool {
	value, ok := err.(*governance.Error)
	if ok {
		*target = value
	}
	return ok
}
