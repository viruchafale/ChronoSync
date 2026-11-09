package exchange

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/governance"
	"chronosync.local/engine/internal/model"
)

func TestExchangePackageHasNoExecutionNetworkOrAuthorizationImports(t *testing.T) {
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
		"chronosync.local/engine/internal/admission":  true,
		"chronosync.local/engine/internal/approval":   true,
		"chronosync.local/engine/internal/policy":     true,
		"chronosync.local/engine/internal/provider":   true,
		"chronosync.local/engine/internal/snapshot":   true,
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
				t.Fatalf("forbidden exchange import %q in %s", path, entry.Name())
			}
		}
	}
}

func TestTrustedUntrustedRevokedRotationAndRestart(t *testing.T) {
	ctx := context.Background()
	reportRaw, report, comparison := testReport(t)
	datasetRaw, datasetID, dataset := testDataset(t, report, comparison, false)
	source := newTestService(t, map[string][]byte{report.ReportID: reportRaw}, map[string][]byte{datasetID: datasetRaw})
	identity, err := source.CreateIdentity(ctx, "Personal test signer", "")
	if err != nil || identity.Trust != Trusted || !identity.HasPrivateKey {
		t.Fatal(identity, err)
	}
	exported, err := source.Export(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil || exported.Verification.Status != ValidTrusted || exported.PackageID == "" {
		t.Fatal(exported.Verification, err)
	}
	packageRaw, _ := base64.StdEncoding.DecodeString(exported.BytesBase64)
	targetRoot := privateRoot(t)
	targetReports := &memoryReports{values: map[string][]byte{}}
	targetBenchmarks := &memoryBenchmarks{values: map[string][]byte{}}
	target, err := New(targetRoot, targetReports, targetBenchmarks)
	if err != nil {
		t.Fatal(err)
	}
	verification, err := target.Inspect(ctx, packageRaw)
	if err != nil || verification.Status != ValidUntrusted || !verification.SignatureValid || verification.Trust != Untrusted {
		t.Fatal(verification, err)
	}
	if _, err = target.Import(ctx, packageRaw, false); !exchangeError(err, "UNTRUSTED_CONFIRMATION_REQUIRED") {
		t.Fatal("untrusted package imported without confirmation", err)
	}
	imported, err := target.Import(ctx, packageRaw, true)
	if err != nil || imported.Dataset.ContentID != datasetID || imported.Reports != 1 {
		t.Fatal(imported, err)
	}
	trusted, err := target.SetTrust(ctx, identity.Identity.KeyID, Trusted, "verified out of band")
	if err != nil || trusted.Trust != Trusted {
		t.Fatal(trusted, err)
	}
	target.Close()
	target, err = New(targetRoot, targetReports, targetBenchmarks)
	if err != nil {
		t.Fatal(err)
	}
	defer target.Close()
	verification, err = target.Inspect(ctx, packageRaw)
	if err != nil || verification.Status != ValidTrusted {
		t.Fatal("trust did not survive restart", verification, err)
	}
	if _, err = target.SetTrust(ctx, identity.Identity.KeyID, Revoked, "test revocation"); err != nil {
		t.Fatal(err)
	}
	target.Close()
	target, err = New(targetRoot, targetReports, targetBenchmarks)
	if err != nil {
		t.Fatal(err)
	}
	verification, err = target.Inspect(ctx, packageRaw)
	if err != nil || verification.Status != RevokedIdentity || !verification.SignatureValid {
		t.Fatal("revocation did not survive restart", verification, err)
	}
	rotated, err := source.CreateIdentity(ctx, "Rotated personal signer", identity.Identity.KeyID)
	if err != nil || rotated.Identity.KeyID == identity.Identity.KeyID || rotated.Identity.Supersedes != identity.Identity.KeyID {
		t.Fatal(rotated, err)
	}
	_ = dataset
}

func TestTypedSignatureFormatTamperingAndIncompleteProvenance(t *testing.T) {
	ctx := context.Background()
	reportRaw, report, comparison := testReport(t)
	datasetRaw, datasetID, _ := testDataset(t, report, comparison, false)
	service := newTestService(t, map[string][]byte{report.ReportID: reportRaw}, map[string][]byte{datasetID: datasetRaw})
	identity, err := service.CreateIdentity(ctx, "Signer", "")
	if err != nil {
		t.Fatal(err)
	}
	complete, err := service.Export(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := base64.StdEncoding.DecodeString(complete.BytesBase64)

	tampered := rewriteEntry(t, raw, benchmarkPath, func(value []byte) []byte {
		result := append([]byte(nil), value...)
		result[len(result)-2] ^= 1
		return result
	})
	assertStatus(t, service, tampered, MalformedPackage)

	badSignature := rewriteJSON[SignatureEnvelope](t, raw, signaturePath, func(value *SignatureEnvelope) {
		value.Signature = base64.StdEncoding.EncodeToString(make([]byte, 64))
	})
	assertStatus(t, service, badSignature, InvalidSignature)
	unknownAlgorithm := rewriteJSON[SignatureEnvelope](t, raw, signaturePath, func(value *SignatureEnvelope) {
		value.Algorithm = "future-signature"
	})
	assertStatus(t, service, unknownAlgorithm, UnsupportedAlgorithm)
	unknownVersion := rewriteJSON[Manifest](t, raw, manifestPath, func(value *Manifest) {
		value.FormatVersion = 99
	})
	assertStatus(t, service, unknownVersion, UnsupportedFormat)

	incomplete, err := service.Export(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{}})
	if err != nil || incomplete.Verification.Status != IncompleteProvenance || incomplete.Verification.Importable {
		t.Fatal(incomplete.Verification, err)
	}

	mismatchedRaw, mismatchedID, _ := testDataset(t, report, comparison, true)
	service.benchmarks.(*memoryBenchmarks).values[mismatchedID] = mismatchedRaw
	mismatched, err := service.Export(ctx, ExportRequest{DatasetContentID: mismatchedID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	if err != nil || mismatched.Verification.Status != IncompleteProvenance {
		t.Fatal(mismatched.Verification, err)
	}
	if _, err = service.SetTrust(ctx, identity.Identity.KeyID, Revoked, "verify before trust lookup"); err != nil {
		t.Fatal(err)
	}
	incompleteRaw, _ := base64.StdEncoding.DecodeString(incomplete.BytesBase64)
	verification, err := service.Inspect(ctx, incompleteRaw)
	if err != nil || verification.Status != IncompleteProvenance || verification.Trust != Untrusted {
		t.Fatal("trust was resolved before complete provenance verification", verification, err)
	}
}

func TestAdversarialArchiveAndConcurrentImport(t *testing.T) {
	ctx := context.Background()
	reportRaw, report, comparison := testReport(t)
	datasetRaw, datasetID, _ := testDataset(t, report, comparison, false)
	service := newTestService(t, map[string][]byte{report.ReportID: reportRaw}, map[string][]byte{datasetID: datasetRaw})
	identity, _ := service.CreateIdentity(ctx, "Signer", "")
	exported, _ := service.Export(ctx, ExportRequest{DatasetContentID: datasetID, IdentityKeyID: identity.Identity.KeyID, IncludeReportIDs: []string{report.ReportID}})
	raw, _ := base64.StdEncoding.DecodeString(exported.BytesBase64)

	assertStatus(t, service, append(raw, make([]byte, MaxPackageBytes)...), MalformedPackage)
	assertStatus(t, service, addEntry(t, raw, "../escape", []byte("x"), 0600, zip.Store), MalformedPackage)
	assertStatus(t, service, addEntry(t, raw, "reports/"+model.ID()+".json", []byte("x"), os.ModeSymlink|0777, zip.Store), MalformedPackage)
	assertStatus(t, service, rewriteArchiveMethod(t, raw, benchmarkPath, zip.Deflate), MalformedPackage)
	assertStatus(t, service, duplicateEntry(t, raw, manifestPath), MalformedPackage)
	excessive := archiveEntries(t, raw)
	for len(excessive) <= MaxEntries {
		excessive[reportPath(model.ID())] = []byte("x")
	}
	assertStatus(t, service, buildArchive(t, excessive, nil, zip.Store), MalformedPackage)
	duplicateJSON := rewriteEntry(t, raw, manifestPath, func([]byte) []byte { return []byte(`{"format_version":1,"format_version":1}`) })
	assertStatus(t, service, duplicateJSON, MalformedPackage)
	unknownField := rewriteEntry(t, raw, manifestPath, func(value []byte) []byte {
		return append(value[:len(value)-1], []byte(`,"authorize":true}`)...)
	})
	assertStatus(t, service, unknownField, MalformedPackage)

	targetReports := &memoryReports{values: map[string][]byte{}, failImport: true}
	targetBenchmarks := &memoryBenchmarks{values: map[string][]byte{}}
	target := newTestServiceWithRoot(t, privateRoot(t), targetReports, targetBenchmarks)
	if _, err := target.Import(ctx, raw, true); err == nil || len(targetBenchmarks.values) != 0 {
		t.Fatal("interrupted report import published dataset", err)
	}
	targetReports.failImport = false

	var wait sync.WaitGroup
	errorsSeen := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, importErr := target.Import(ctx, raw, true)
			errorsSeen <- importErr
		}()
	}
	wait.Wait()
	close(errorsSeen)
	successes := 0
	for importErr := range errorsSeen {
		if importErr == nil {
			successes++
		} else if !exchangeError(importErr, "BUSY") {
			t.Fatal(importErr)
		}
	}
	if successes < 1 || len(targetBenchmarks.values) != 1 {
		t.Fatal("concurrent imports did not converge", successes, len(targetBenchmarks.values))
	}
}

func TestEd25519RFC8032Vector(t *testing.T) {
	seed, _ := hex.DecodeString("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	expectedPublic, _ := hex.DecodeString("d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a")
	expectedSignature, _ := hex.DecodeString("e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b")
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	signature := ed25519.Sign(privateKey, nil)
	if !bytes.Equal(publicKey, expectedPublic) || !bytes.Equal(signature, expectedSignature) || !ed25519.Verify(publicKey, nil, signature) {
		t.Fatal("standard Ed25519 implementation does not match RFC 8032 test vector 1")
	}
}

type memoryReports struct {
	mu         sync.Mutex
	values     map[string][]byte
	failImport bool
}

func (m *memoryReports) ExportCanonical(_ context.Context, id string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	raw, ok := m.values[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return append([]byte(nil), raw...), nil
}

func (m *memoryReports) ImportCanonical(_ context.Context, raw []byte) (analysis.Report, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failImport {
		return analysis.Report{}, false, errors.New("injected interrupted import")
	}
	report, err := analysis.ValidateCanonical(raw)
	if err != nil {
		return analysis.Report{}, false, err
	}
	existing, duplicate := m.values[report.ReportID]
	if duplicate && !bytes.Equal(existing, raw) {
		return analysis.Report{}, false, errors.New("conflict")
	}
	m.values[report.ReportID] = append([]byte(nil), raw...)
	return report, duplicate, nil
}

type memoryBenchmarks struct {
	mu     sync.Mutex
	values map[string][]byte
}

func (m *memoryBenchmarks) ExportDataset(_ context.Context, id string, consent bool) ([]byte, governance.DatasetRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !consent {
		return nil, governance.DatasetRecord{}, errors.New("consent")
	}
	raw, ok := m.values[id]
	if !ok {
		return nil, governance.DatasetRecord{}, errors.New("not found")
	}
	dataset, contentID, _ := governance.ValidateDatasetBytes(raw)
	return append([]byte(nil), raw...), governance.DatasetRecord{ContentID: contentID, BenchmarkID: dataset.BenchmarkID, Version: dataset.Version}, nil
}

func (m *memoryBenchmarks) ImportDataset(_ context.Context, raw []byte) (governance.DatasetRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dataset, id, err := governance.ValidateDatasetBytes(raw)
	if err != nil {
		return governance.DatasetRecord{}, err
	}
	existing, duplicate := m.values[id]
	if duplicate && !bytes.Equal(existing, raw) {
		return governance.DatasetRecord{}, errors.New("conflict")
	}
	m.values[id] = append([]byte(nil), raw...)
	return governance.DatasetRecord{ContentID: id, BenchmarkID: dataset.BenchmarkID, Version: dataset.Version, Duplicate: duplicate}, nil
}

func (m *memoryBenchmarks) GetDataset(_ context.Context, id string) (governance.Dataset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	dataset, _, err := governance.ValidateDatasetBytes(m.values[id])
	return dataset, err
}

func (m *memoryBenchmarks) ListDatasets(_ context.Context) ([]governance.DatasetRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := []governance.DatasetRecord{}
	for id, raw := range m.values {
		dataset, _, _ := governance.ValidateDatasetBytes(raw)
		result = append(result, governance.DatasetRecord{ContentID: id, BenchmarkID: dataset.BenchmarkID, Version: dataset.Version})
	}
	return result, nil
}

func testReport(t *testing.T) ([]byte, analysis.Report, analysis.Comparison) {
	t.Helper()
	reportID := model.ID()
	base := analysis.Observation{FormatVersion: 1, ObservationID: "base-" + reportID, SourceExecutionID: model.ID(), SourceEventVersion: model.EventVersion, SourceHighWater: "1", SourceHistory: "complete", Configuration: analysis.Configuration{Adapter: "synthetic", AdapterVersion: "1", RecorderVersion: "1", Sandbox: "none", WorkspaceSHA256: hash([]byte("workspace"))}, Events: []analysis.EventEvidence{{Sequence: "1", EventID: model.ID(), Kind: "command.completed", OutputSHA256: hash([]byte("a")), EvidenceSHA256: hash([]byte("base")), Missing: []analysis.Gap{}}}, Snapshots: []analysis.SnapshotEvidence{}, Missing: []analysis.Gap{}, Unsupported: []analysis.Gap{}}
	observed := base
	observed.ObservationID = "observed-" + reportID
	observed.SourceExecutionID = model.ID()
	observed.Events = append([]analysis.EventEvidence(nil), base.Events...)
	observed.Events[0].OutputSHA256 = hash([]byte("b"))
	observed.Events[0].EvidenceSHA256 = hash([]byte("observed"))
	comparison, err := analysis.Compare(base, observed)
	if err != nil {
		t.Fatal(err)
	}
	experiment := analysis.Experiment{FormatVersion: 1, Observations: []analysis.Observation{base, observed}, Comparisons: []analysis.Comparison{comparison}, Aggregate: analysis.Aggregate{Observations: 2, Comparisons: 1, Divergent: 1, MeanEventAgreement: comparison.Metrics.EventAgreement, MeanDimensionAgreement: comparison.Metrics.DimensionAgreement}}
	policy := analysis.DefaultPolicy()
	evaluation, err := analysis.Evaluate(experiment, policy)
	if err != nil {
		t.Fatal(err)
	}
	report := analysis.Report{FormatVersion: 1, ReportID: reportID, CreatedAt: model.Now(), Experiment: experiment, Policy: policy, Evaluation: evaluation}
	unsigned, _ := json.Marshal(report)
	report.SHA256 = hash(unsigned)
	raw, _ := json.Marshal(report)
	if _, err = analysis.ValidateCanonical(raw); err != nil {
		t.Fatal(err)
	}
	return raw, report, comparison
}

func testDataset(t *testing.T, report analysis.Report, comparison analysis.Comparison, mismatch bool) ([]byte, string, governance.Dataset) {
	t.Helper()
	comparisonRaw, _ := json.Marshal(comparison)
	comparisonSHA := hash(comparisonRaw)
	if mismatch {
		comparisonSHA = stringsRepeat("0", 64)
	}
	dataset := governance.Dataset{FormatVersion: 1, BenchmarkID: model.ID(), Version: 1, Name: "Synthetic exchange benchmark", CreatedAt: model.Now(), Cases: []governance.Case{{CaseID: "case-001", SourceReportID: report.ReportID, ComparisonIndex: 0, BaselineObservationID: comparison.BaselineObservation, ObservedObservationID: comparison.ObservedObservation, ComparisonSHA256: comparisonSHA, ExpectedStatus: comparison.Status, SupportedDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering, analysis.Output}}}, SupportedDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering, analysis.Output}, RequiredDimensions: []analysis.Dimension{analysis.EventType, analysis.Ordering}, CompatibilityModel: "case-id-v1", ProvenanceLimitations: []string{"Synthetic observable evidence only."}}
	body, _ := json.Marshal(dataset)
	envelope := governance.DatasetEnvelope{Dataset: dataset, SHA256: hash(body)}
	raw, _ := json.Marshal(envelope)
	_, id, err := governance.ValidateDatasetBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	return raw, id, dataset
}

func newTestService(t *testing.T, reports map[string][]byte, benchmarks map[string][]byte) *Service {
	t.Helper()
	return newTestServiceWithRoot(t, privateRoot(t), &memoryReports{values: reports}, &memoryBenchmarks{values: benchmarks})
}

func newTestServiceWithRoot(t *testing.T, root string, reports *memoryReports, benchmarks *memoryBenchmarks) *Service {
	t.Helper()
	service, err := New(root, reports, benchmarks)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Close() })
	return service
}

func privateRoot(t *testing.T) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	return root
}

func hash(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func assertStatus(t *testing.T, service *Service, raw []byte, expected VerificationStatus) {
	t.Helper()
	verification, err := service.Inspect(context.Background(), raw)
	if err != nil || verification.Status != expected {
		t.Fatalf("status %s, expected %s: %v (%s)", verification.Status, expected, err, verification.Detail)
	}
}

func rewriteJSON[T any](t *testing.T, raw []byte, name string, mutate func(*T)) []byte {
	t.Helper()
	return rewriteEntry(t, raw, name, func(value []byte) []byte {
		var decoded T
		if err := json.Unmarshal(value, &decoded); err != nil {
			t.Fatal(err)
		}
		mutate(&decoded)
		result, _ := json.Marshal(decoded)
		return result
	})
}

func rewriteEntry(t *testing.T, raw []byte, target string, mutate func([]byte) []byte) []byte {
	t.Helper()
	entries := archiveEntries(t, raw)
	entries[target] = mutate(entries[target])
	return buildArchive(t, entries, nil, zip.Store)
}

func rewriteArchiveMethod(t *testing.T, raw []byte, target string, method uint16) []byte {
	t.Helper()
	return buildArchive(t, archiveEntries(t, raw), map[string]uint16{target: method}, zip.Store)
}

func addEntry(t *testing.T, raw []byte, name string, value []byte, mode os.FileMode, method uint16) []byte {
	t.Helper()
	entries := archiveEntries(t, raw)
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for entryName, entryRaw := range entries {
		writeArchiveEntry(t, writer, entryName, entryRaw, 0600, zip.Store)
	}
	writeArchiveEntry(t, writer, name, value, mode, method)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func duplicateEntry(t *testing.T, raw []byte, name string) []byte {
	t.Helper()
	entries := archiveEntries(t, raw)
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	for entryName, entryRaw := range entries {
		writeArchiveEntry(t, writer, entryName, entryRaw, 0600, zip.Store)
	}
	writeArchiveEntry(t, writer, name, entries[name], 0600, zip.Store)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func archiveEntries(t *testing.T, raw []byte) map[string][]byte {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		t.Fatal(err)
	}
	result := map[string][]byte{}
	for _, file := range reader.File {
		stream, _ := file.Open()
		value, _ := io.ReadAll(stream)
		stream.Close()
		result[file.Name] = value
	}
	return result
}

func buildArchive(t *testing.T, entries map[string][]byte, methods map[string]uint16, fallback uint16) []byte {
	t.Helper()
	buffer := &bytes.Buffer{}
	writer := zip.NewWriter(buffer)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		method := fallback
		if configured, ok := methods[name]; ok {
			method = configured
		}
		writeArchiveEntry(t, writer, name, entries[name], 0600, method)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func writeArchiveEntry(t *testing.T, writer *zip.Writer, name string, value []byte, mode os.FileMode, method uint16) {
	t.Helper()
	header := &zip.FileHeader{Name: name, Method: method}
	header.SetMode(mode)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = entry.Write(value); err != nil {
		t.Fatal(err)
	}
}

func exchangeError(err error, code string) bool {
	var typed *Error
	return errors.As(err, &typed) && typed.Code == code
}

func stringsRepeat(value string, count int) string {
	return string(bytes.Repeat([]byte(value), count))
}
