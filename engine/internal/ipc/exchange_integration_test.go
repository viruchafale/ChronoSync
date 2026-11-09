package ipc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"testing"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/exchange"
	"chronosync.local/engine/internal/governance"
	"chronosync.local/engine/internal/storage"
)

func TestSignedExchangeCleanMachineImportRestartAndReexport(t *testing.T) {
	ctx := context.Background()
	sourceBase := filepath.Join(t.TempDir(), "source")
	sourceStore, sourceAnalysis, sourceGovernance, sourceExchange := exchangeStack(t, sourceBase)
	baseline := addAnalysisRun(t, sourceStore, "stable-output", false)
	changed := addAnalysisRun(t, sourceStore, "changed-output", true)
	report := analysisRequest(t, "analysis.compare", map[string]any{"run_ids": []string{baseline, changed}, "policy": analysis.DefaultPolicy()}, sourceAnalysis).(analysis.Report)
	dataset := governanceRequest(t, "benchmark.create", governance.CreateDataset{Version: 1, Name: "Portable synthetic benchmark", Cases: []governance.CaseDraft{{CaseID: "case-001", SourceReportID: report.ReportID, ComparisonIndex: 0, SupportedDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering}}}, SupportedDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering}, RequiredDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering}, ProvenanceLimitations: []string{"Synthetic exchange integration evidence only."}}, sourceGovernance).(governance.DatasetRecord)
	identity := exchangeRequest(t, "identity.create", map[string]string{"label": "Source signer", "supersedes": ""}, sourceExchange).(exchange.IdentityRecord)
	exported := exchangeRequest(t, "exchange.export", exchange.ExportRequest{DatasetContentID: dataset.ContentID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}}, sourceExchange).(exchange.ExportResult)
	if exported.Verification.Status != exchange.ValidTrusted {
		t.Fatal(exported.Verification)
	}

	targetBase := filepath.Join(t.TempDir(), "target")
	targetStore, targetAnalysis, targetGovernance, targetExchange := exchangeStack(t, targetBase)
	inspection := exchangeRequest(t, "exchange.inspect", map[string]any{"bytes_base64": exported.BytesBase64}, targetExchange).(exchange.Verification)
	if inspection.Status != exchange.ValidUntrusted || !inspection.SignatureValid {
		t.Fatal(inspection)
	}
	if _, err := exchangeDispatch(exchangeIPCRequest(t, "exchange.import", map[string]any{"bytes_base64": exported.BytesBase64, "allow_untrusted": false}), targetExchange); err == nil {
		t.Fatal("valid untrusted package imported without explicit confirmation")
	}
	imported := exchangeRequest(t, "exchange.import", map[string]any{"bytes_base64": exported.BytesBase64, "allow_untrusted": true}, targetExchange).(exchange.ImportResult)
	if imported.Dataset.ContentID != dataset.ContentID || imported.Reports != 1 {
		t.Fatal(imported)
	}
	if _, err := targetAnalysis.Get(ctx, report.ReportID); err != nil {
		t.Fatal("imported report unavailable", err)
	}
	if _, err := targetGovernance.GetDataset(ctx, dataset.ContentID); err != nil {
		t.Fatal("imported benchmark unavailable", err)
	}
	exchangeRequest(t, "identity.trust", map[string]any{"key_id": identity.Identity.KeyID, "state": exchange.Trusted, "reason": "integration trust decision"}, targetExchange)
	targetExchange.Close()
	targetExchange, err := exchange.New(targetBase, targetAnalysis, targetGovernance)
	if err != nil {
		t.Fatal(err)
	}
	inspection = exchangeRequest(t, "exchange.inspect", map[string]any{"bytes_base64": exported.BytesBase64}, targetExchange).(exchange.Verification)
	if inspection.Status != exchange.ValidTrusted {
		t.Fatal("trust state unavailable after restart", inspection)
	}
	targetIdentity := exchangeRequest(t, "identity.create", map[string]string{"label": "Target re-export signer", "supersedes": ""}, targetExchange).(exchange.IdentityRecord)
	reexported := exchangeRequest(t, "exchange.export", exchange.ExportRequest{DatasetContentID: dataset.ContentID, IdentityKeyID: targetIdentity.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}}, targetExchange).(exchange.ExportResult)
	if reexported.PackageID != exported.PackageID || reexported.Verification.Status != exchange.ValidTrusted {
		t.Fatal("re-export changed content identity or failed independent signing", reexported)
	}

	incomplete := exchangeRequest(t, "exchange.export", exchange.ExportRequest{DatasetContentID: dataset.ContentID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{}}, sourceExchange).(exchange.ExportResult)
	if incomplete.Verification.Status != exchange.IncompleteProvenance {
		t.Fatal(incomplete.Verification)
	}
	if _, err = exchangeDispatch(exchangeIPCRequest(t, "exchange.import", map[string]any{"bytes_base64": incomplete.BytesBase64, "allow_untrusted": true}), targetExchange); err == nil {
		t.Fatal("clean-machine package with omitted provenance imported")
	}

	packageRaw, _ := base64.StdEncoding.DecodeString(exported.BytesBase64)
	if len(packageRaw) == 0 {
		t.Fatal("exported binary package unavailable")
	}
	targetExchange.Close()
	targetGovernance.Close()
	targetAnalysis.Close()
	targetStore.Close()
	sourceExchange.Close()
	sourceGovernance.Close()
	sourceAnalysis.Close()
	sourceStore.Close()
}

func TestExchangeIPCRejectsUnknownTrustAuthority(t *testing.T) {
	base := filepath.Join(t.TempDir(), "private")
	store, analysisService, governanceService, exchangeService := exchangeStack(t, base)
	defer store.Close()
	defer analysisService.Close()
	defer governanceService.Close()
	defer exchangeService.Close()
	raw := json.RawMessage(`{"key_id":"1111111111111111111111111111111111111111111111111111111111111111","state":"trusted","reason":"test","organization_admin":true}`)
	if _, err := exchangeDispatch(Request{ProtocolVersion: 1, ID: "test", Method: "identity.trust", Params: raw}, exchangeService); err == nil {
		t.Fatal("unknown organization authority field accepted")
	}
}

func exchangeStack(t *testing.T, base string) (*storage.Store, *analysis.Service, *governance.Service, *exchange.Service) {
	t.Helper()
	store, err := storage.Open(base)
	if err != nil {
		t.Fatal(err)
	}
	analysisService, err := analysis.New(base, store, store)
	if err != nil {
		t.Fatal(err)
	}
	governanceService, err := governance.New(base, analysisService)
	if err != nil {
		t.Fatal(err)
	}
	exchangeService, err := exchange.New(base, analysisService, governanceService)
	if err != nil {
		t.Fatal(err)
	}
	return store, analysisService, governanceService, exchangeService
}

func exchangeRequest(t *testing.T, method string, params any, service *exchange.Service) any {
	t.Helper()
	result, err := exchangeDispatch(exchangeIPCRequest(t, method, params), service)
	if err != nil {
		t.Fatal(method, err)
	}
	return result
}

func exchangeIPCRequest(t *testing.T, method string, params any) Request {
	t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}
	return Request{ProtocolVersion: 1, ID: "test", Method: method, Params: raw}
}
