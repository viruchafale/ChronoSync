package exchange

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/governance"
	"chronosync.local/engine/internal/model"
)

type decodedPackage struct {
	manifestRaw []byte
	manifest    Manifest
	signature   SignatureEnvelope
	datasetRaw  []byte
	dataset     governance.Dataset
	reports     map[string][]byte
}

func (s *Service) Export(ctx context.Context, request ExportRequest) (ExportResult, error) {
	if err := s.enter(); err != nil {
		return ExportResult{}, err
	}
	defer s.leave()
	prepared, err := s.prepareExport(ctx, request)
	if err != nil {
		return ExportResult{}, err
	}
	privateKey, err := s.privateKey(prepared.identity.KeyID)
	if err != nil {
		return ExportResult{}, err
	}
	return s.finishPrepared(ctx, prepared, ed25519.Sign(privateKey, prepared.payload))
}

func (s *Service) BeginExport(ctx context.Context, request ExportRequest) (ExportBeginResult, error) {
	if err := s.enter(); err != nil {
		return ExportBeginResult{}, err
	}
	defer s.leave()
	prepared, err := s.prepareExport(ctx, request)
	if err != nil {
		return ExportBeginResult{}, err
	}
	backend, _ := s.custodyState(prepared.identity.KeyID)
	if backend == LegacyFile {
		privateKey, keyErr := s.privateKey(prepared.identity.KeyID)
		if keyErr != nil {
			return ExportBeginResult{}, keyErr
		}
		completed, finishErr := s.finishPrepared(ctx, prepared, ed25519.Sign(privateKey, prepared.payload))
		if finishErr != nil {
			return ExportBeginResult{}, finishErr
		}
		return ExportBeginResult{Mode: "legacy_completed", Completed: &completed}, nil
	}
	if backend != MacOSKeychain {
		return ExportBeginResult{}, &Error{"PRIVATE_KEY_UNAVAILABLE", "no usable signing custody"}
	}
	for id, session := range s.sessions {
		if time.Now().After(session.expires) {
			delete(s.sessions, id)
		}
	}
	if len(s.sessions) >= 16 {
		return ExportBeginResult{}, &Error{"BUSY", "signing session capacity"}
	}
	id := model.ID()
	prepared.expires = time.Now().Add(2 * time.Minute)
	s.sessions[id] = prepared
	return ExportBeginResult{Mode: "native_sign", SessionID: id, KeyID: prepared.keyID, PayloadBase64: base64.StdEncoding.EncodeToString(prepared.payload)}, nil
}

func (s *Service) FinishExport(ctx context.Context, sessionID, signatureBase64 string) (ExportResult, error) {
	if err := s.enter(); err != nil {
		return ExportResult{}, err
	}
	defer s.leave()
	if !validModelID(sessionID) {
		return ExportResult{}, &Error{"INVALID_ARGUMENT", "signing session"}
	}
	prepared, ok := s.sessions[sessionID]
	delete(s.sessions, sessionID)
	if !ok || time.Now().After(prepared.expires) {
		return ExportResult{}, &Error{"SIGNING_SESSION_EXPIRED", "one-use signing session"}
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(signatureBase64)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ExportResult{}, &Error{"INVALID_SIGNATURE", "native signing result"}
	}
	return s.finishPrepared(ctx, prepared, signature)
}

func (s *Service) prepareExport(ctx context.Context, request ExportRequest) (exportSession, error) {
	if ctx.Err() != nil || !validDigest(request.DatasetContentID) || !validDigest(request.IdentityKeyID) {
		return exportSession{}, &Error{"INVALID_ARGUMENT", "exchange export request"}
	}
	requestedReports, ok := canonicalStrings(request.IncludeReportIDs)
	if !ok {
		return exportSession{}, &Error{"INVALID_ARGUMENT", "report selection"}
	}
	identity, err := s.getIdentity(request.IdentityKeyID)
	if err != nil {
		return exportSession{}, err
	}
	trust, err := s.trustState(ctx, identity.KeyID)
	if err != nil {
		return exportSession{}, err
	}
	if trust.State == Revoked {
		return exportSession{}, &Error{"REVOKED_IDENTITY", "signing identity"}
	}
	datasetRaw, _, err := s.benchmarks.ExportDataset(ctx, request.DatasetContentID, true)
	if err != nil {
		return exportSession{}, err
	}
	dataset, contentID, err := governance.ValidateDatasetBytes(datasetRaw)
	if err != nil || contentID != request.DatasetContentID {
		return exportSession{}, &Error{"CORRUPT_DATASET", "exchange source dataset"}
	}
	required := requiredReports(dataset)
	requested := map[string]bool{}
	for _, id := range requestedReports {
		if !required[id] {
			return exportSession{}, &Error{"INVALID_ARGUMENT", "selected report is not required by dataset"}
		}
		requested[id] = true
	}
	reportBytes := map[string][]byte{}
	omitted := []string{}
	for id := range required {
		if !requested[id] {
			omitted = append(omitted, id)
			continue
		}
		raw, exportErr := s.reports.ExportCanonical(ctx, id)
		if exportErr != nil {
			return exportSession{}, exportErr
		}
		report, validationErr := analysis.ValidateCanonical(raw)
		if validationErr != nil || report.ReportID != id {
			return exportSession{}, &Error{"CORRUPT_REPORT", "exchange source report"}
		}
		reportBytes[id] = raw
	}
	sort.Strings(omitted)
	entries := []Entry{{Path: benchmarkPath, Role: "benchmark", SHA256: digest(datasetRaw), Bytes: int64(len(datasetRaw))}}
	for _, id := range requestedReports {
		raw := reportBytes[id]
		entries = append(entries, Entry{Path: reportPath(id), Role: "analysis_report", SHA256: digest(raw), Bytes: int64(len(raw))})
	}
	manifest := Manifest{FormatVersion: FormatVersion, Domain: "chronosync.benchmark-exchange", DatasetContentID: contentID, BenchmarkID: dataset.BenchmarkID, BenchmarkVersion: dataset.Version, CompatibilityModel: dataset.CompatibilityModel, Entries: entries, IncludedReportIDs: requestedReports, OmittedReportIDs: omitted, Privacy: PrivacyManifest{RawCommands: false, RawOutputs: false, WorkspacePaths: false, Credentials: false, Environment: false, PrivateKeys: false, IncludedEvidence: append([]string{"benchmark_dataset"}, prefixedReports(requestedReports)...), OmittedEvidence: prefixedReports(omitted)}}
	manifestRaw, _ := json.Marshal(manifest)
	identityRaw, _ := json.Marshal(identity)
	payload := signingPayload(identityRaw, manifestRaw)
	return exportSession{keyID: identity.KeyID, identity: identity, manifestRaw: manifestRaw, datasetRaw: datasetRaw, reports: reportBytes, payload: payload}, nil
}

func (s *Service) finishPrepared(ctx context.Context, prepared exportSession, signature []byte) (ExportResult, error) {
	publicKey, _ := base64.StdEncoding.Strict().DecodeString(prepared.identity.PublicKey)
	if len(signature) != ed25519.SignatureSize || !ed25519.Verify(publicKey, prepared.payload, signature) {
		return ExportResult{}, &Error{"INVALID_SIGNATURE", "native signature did not match identity"}
	}
	envelope := SignatureEnvelope{FormatVersion: FormatVersion, Algorithm: AlgorithmEd25519, Identity: prepared.identity, Signature: base64.StdEncoding.EncodeToString(signature)}
	signatureRaw, _ := json.Marshal(envelope)
	packageRaw, err := writePackage(prepared.manifestRaw, prepared.datasetRaw, prepared.reports, signatureRaw)
	if err != nil {
		return ExportResult{}, err
	}
	verification, _, err := s.inspect(ctx, packageRaw)
	if err != nil || verification.Status != ValidTrusted && verification.Status != ValidUntrusted && verification.Status != IncompleteProvenance {
		return ExportResult{}, &Error{"EXPORT_VERIFICATION_FAILED", "generated package did not verify"}
	}
	return ExportResult{PackageID: verification.PackageID, BytesBase64: base64.StdEncoding.EncodeToString(packageRaw), Bytes: len(packageRaw), Verification: verification}, nil
}

func (s *Service) Inspect(ctx context.Context, raw []byte) (Verification, error) {
	verification, _, err := s.inspect(ctx, raw)
	return verification, err
}

func (s *Service) Import(ctx context.Context, raw []byte, allowUntrusted bool) (ImportResult, error) {
	if err := s.enter(); err != nil {
		return ImportResult{}, err
	}
	defer s.leave()
	verification, decoded, err := s.inspect(ctx, raw)
	if err != nil {
		return ImportResult{}, err
	}
	if verification.Status == ValidUntrusted && !allowUntrusted {
		return ImportResult{}, &Error{"UNTRUSTED_CONFIRMATION_REQUIRED", "valid untrusted package"}
	}
	if verification.Status != ValidTrusted && verification.Status != ValidUntrusted {
		return ImportResult{}, &Error{"IMPORT_REFUSED", string(verification.Status)}
	}
	if _, err = s.observeIdentity(decoded.signature.Identity); err != nil {
		return ImportResult{}, err
	}
	duplicates := true
	for _, id := range decoded.manifest.IncludedReportIDs {
		_, duplicate, importErr := s.reports.ImportCanonical(ctx, decoded.reports[id])
		if importErr != nil {
			return ImportResult{}, importErr
		}
		duplicates = duplicates && duplicate
	}
	dataset, err := s.benchmarks.ImportDataset(ctx, decoded.datasetRaw)
	if err != nil {
		return ImportResult{}, err
	}
	duplicates = duplicates && dataset.Duplicate
	return ImportResult{Verification: verification, Dataset: dataset, Reports: len(decoded.reports), Duplicate: duplicates}, nil
}

func (s *Service) inspect(ctx context.Context, raw []byte) (Verification, *decodedPackage, error) {
	if ctx.Err() != nil {
		return Verification{}, nil, &Error{"CANCELLED", "package inspection"}
	}
	decoded, verification := decodePackage(raw)
	if decoded == nil {
		return verification, nil, nil
	}
	if decoded.manifest.FormatVersion != FormatVersion || decoded.signature.FormatVersion != FormatVersion {
		verification.Status = UnsupportedFormat
		verification.Detail = "exchange format version is unsupported"
		return verification, decoded, nil
	}
	if decoded.signature.Algorithm != AlgorithmEd25519 || decoded.signature.Identity.Algorithm != AlgorithmEd25519 {
		verification.Status = UnsupportedAlgorithm
		verification.Detail = "signature algorithm is unsupported"
		return verification, decoded, nil
	}
	if err := validateIdentity(decoded.signature.Identity); err != nil {
		verification.Status = MalformedPackage
		verification.Detail = "signing identity is malformed"
		return verification, decoded, nil
	}
	verification.Identity = &decoded.signature.Identity
	publicKey, _ := base64.StdEncoding.Strict().DecodeString(decoded.signature.Identity.PublicKey)
	signature, signatureErr := base64.StdEncoding.Strict().DecodeString(decoded.signature.Signature)
	identityRaw, _ := json.Marshal(decoded.signature.Identity)
	if signatureErr != nil || len(signature) != ed25519.SignatureSize || !ed25519.Verify(publicKey, signingPayload(identityRaw, decoded.manifestRaw), signature) {
		verification.Status = InvalidSignature
		verification.Detail = "Ed25519 signature verification failed"
		return verification, decoded, nil
	}
	verification.SignatureValid = true
	if err := validateProvenance(decoded); err != nil {
		verification.Status = IncompleteProvenance
		verification.Detail = err.Error()
		verification.Warnings = append(verification.Warnings, "The signature is valid, but required benchmark provenance is incomplete or mismatched.")
		return verification, decoded, nil
	}
	trust, err := s.trustState(ctx, decoded.signature.Identity.KeyID)
	if err != nil {
		return Verification{}, nil, err
	}
	verification.Trust = trust.State
	if trust.State == Revoked {
		verification.Status = RevokedIdentity
		verification.Detail = "signing identity is locally revoked"
		return verification, decoded, nil
	}
	verification.Importable = true
	if trust.State == Trusted {
		verification.Status = ValidTrusted
		verification.Detail = "signature and complete provenance verified; identity is locally trusted"
	} else {
		verification.Status = ValidUntrusted
		verification.Detail = "signature and complete provenance verified; identity is not locally trusted"
		verification.Warnings = append(verification.Warnings, "A valid signature does not establish trust, safety, quality, or execution authorization.")
	}
	return verification, decoded, nil
}

func decodePackage(raw []byte) (*decodedPackage, Verification) {
	failure := func(status VerificationStatus, detail string) (*decodedPackage, Verification) {
		return nil, Verification{FormatVersion: FormatVersion, Status: status, Trust: Untrusted, Warnings: []string{}, Detail: detail}
	}
	if len(raw) == 0 || len(raw) > MaxPackageBytes {
		return failure(MalformedPackage, "package byte limit")
	}
	reader, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil || len(reader.File) < 3 || len(reader.File) > MaxEntries {
		return failure(MalformedPackage, "ZIP structure or entry count")
	}
	files := map[string][]byte{}
	var total int64
	for _, file := range reader.File {
		if !validArchivePath(file.Name) || file.Method != zip.Store || file.FileInfo().IsDir() || !file.Mode().IsRegular() || file.UncompressedSize64 != file.CompressedSize64 || file.UncompressedSize64 > uint64(maxEntryBytes(file.Name)) {
			return failure(MalformedPackage, "unsafe archive entry")
		}
		if _, duplicate := files[file.Name]; duplicate {
			return failure(MalformedPackage, "duplicate archive entry")
		}
		stream, openErr := file.Open()
		if openErr != nil {
			return failure(MalformedPackage, "archive entry open")
		}
		entryRaw, readErr := io.ReadAll(io.LimitReader(stream, maxEntryBytes(file.Name)+1))
		closeErr := stream.Close()
		if readErr != nil || closeErr != nil || int64(len(entryRaw)) != int64(file.UncompressedSize64) || int64(len(entryRaw)) > maxEntryBytes(file.Name) {
			return failure(MalformedPackage, "archive entry read")
		}
		total += int64(len(entryRaw))
		if total > MaxPackageBytes {
			return failure(MalformedPackage, "archive aggregate size")
		}
		files[file.Name] = entryRaw
	}
	manifestRaw, manifestOK := files[manifestPath]
	signatureRaw, signatureOK := files[signaturePath]
	datasetRaw, datasetOK := files[benchmarkPath]
	if !manifestOK || !signatureOK || !datasetOK {
		return failure(MalformedPackage, "required archive entries")
	}
	var manifest Manifest
	var signature SignatureEnvelope
	if strictCanonical(manifestRaw, &manifest) != nil || strictCanonical(signatureRaw, &signature) != nil {
		return failure(MalformedPackage, "canonical manifest or signature")
	}
	verification := Verification{FormatVersion: FormatVersion, Trust: Untrusted, Manifest: &manifest, PackageID: digest([]byte(packageDomain), manifestRaw), Warnings: []string{"Package signatures establish covered-content key possession only, not authorship, safety, quality, or execution authorization."}}
	if manifest.Domain != "chronosync.benchmark-exchange" || !validDigest(manifest.DatasetContentID) || !model.ValidID(manifest.BenchmarkID) || manifest.BenchmarkVersion < 1 || manifest.CompatibilityModel != "case-id-v1" || !validPrivacy(manifest.Privacy) {
		verification.Status = MalformedPackage
		verification.Detail = "manifest contract"
		return nil, verification
	}
	if err = validateManifestEntries(manifest, files); err != nil {
		verification.Status = MalformedPackage
		verification.Detail = err.Error()
		return nil, verification
	}
	dataset, contentID, datasetErr := governance.ValidateDatasetBytes(datasetRaw)
	if datasetErr != nil || contentID != manifest.DatasetContentID || dataset.BenchmarkID != manifest.BenchmarkID || dataset.Version != manifest.BenchmarkVersion || dataset.CompatibilityModel != manifest.CompatibilityModel {
		verification.Status = MalformedPackage
		verification.Detail = "benchmark manifest mismatch"
		return nil, verification
	}
	reports := map[string][]byte{}
	for _, id := range manifest.IncludedReportIDs {
		reports[id] = files[reportPath(id)]
	}
	return &decodedPackage{manifestRaw: manifestRaw, manifest: manifest, signature: signature, datasetRaw: datasetRaw, dataset: dataset, reports: reports}, verification
}

func validateManifestEntries(manifest Manifest, files map[string][]byte) error {
	included, ok := canonicalStrings(manifest.IncludedReportIDs)
	if !ok || !equalStrings(included, manifest.IncludedReportIDs) {
		return errors.New("included report identities")
	}
	omitted, ok := canonicalStrings(manifest.OmittedReportIDs)
	if !ok || !equalStrings(omitted, manifest.OmittedReportIDs) {
		return errors.New("omitted report identities")
	}
	seen := map[string]bool{manifestPath: true, signaturePath: true}
	last := ""
	for _, entry := range manifest.Entries {
		if entry.Path <= last || seen[entry.Path] || entry.Bytes < 1 || !validDigest(entry.SHA256) {
			return errors.New("manifest entry order or identity")
		}
		if entry.Path == benchmarkPath && entry.Role != "benchmark" || strings.HasPrefix(entry.Path, "reports/") && entry.Role != "analysis_report" {
			return errors.New("manifest entry role")
		}
		raw, exists := files[entry.Path]
		if !exists || int64(len(raw)) != entry.Bytes || digest(raw) != entry.SHA256 {
			return errors.New("manifest entry integrity")
		}
		seen[entry.Path] = true
		last = entry.Path
	}
	if !seen[benchmarkPath] || len(seen) != len(files) {
		return errors.New("manifest/archive entry mismatch")
	}
	for _, id := range included {
		if !validDigest(id) && !model.ValidID(id) {
			return errors.New("included report identity")
		}
		if !seen[reportPath(id)] {
			return errors.New("included report entry")
		}
	}
	return nil
}

func validateProvenance(decoded *decodedPackage) error {
	required := requiredReports(decoded.dataset)
	included := map[string]bool{}
	for _, id := range decoded.manifest.IncludedReportIDs {
		if !required[id] {
			return errors.New("package includes a report not referenced by the benchmark")
		}
		included[id] = true
	}
	omitted := map[string]bool{}
	for _, id := range decoded.manifest.OmittedReportIDs {
		if !required[id] || included[id] {
			return errors.New("package omission declaration is inconsistent")
		}
		omitted[id] = true
	}
	for id := range required {
		if !included[id] && !omitted[id] {
			return errors.New("required report is not declared")
		}
	}
	if len(omitted) > 0 {
		return errors.New("required reports were explicitly omitted")
	}
	parsed := map[string]analysis.Report{}
	for id, raw := range decoded.reports {
		report, err := analysis.ValidateCanonical(raw)
		if err != nil || report.ReportID != id {
			return errors.New("included report is invalid")
		}
		parsed[id] = report
	}
	for _, item := range decoded.dataset.Cases {
		report, ok := parsed[item.SourceReportID]
		if !ok || item.ComparisonIndex < 0 || item.ComparisonIndex >= len(report.Experiment.Comparisons) {
			return errors.New("benchmark source report or comparison is unavailable")
		}
		comparison := report.Experiment.Comparisons[item.ComparisonIndex]
		comparisonRaw, _ := json.Marshal(comparison)
		if comparison.BaselineObservation != item.BaselineObservationID || comparison.ObservedObservation != item.ObservedObservationID || digest(comparisonRaw) != item.ComparisonSHA256 || comparison.Status != item.ExpectedStatus {
			return errors.New("benchmark provenance does not match included report")
		}
	}
	return nil
}

func writePackage(manifestRaw, datasetRaw []byte, reports map[string][]byte, signatureRaw []byte) ([]byte, error) {
	entries := map[string][]byte{manifestPath: manifestRaw, benchmarkPath: datasetRaw, signaturePath: signatureRaw}
	for id, raw := range reports {
		entries[reportPath(id)] = raw
	}
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	buffer := &bytes.Buffer{}
	archive := zip.NewWriter(buffer)
	for _, name := range names {
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetMode(0600)
		header.SetModTime(time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC))
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return nil, &Error{"PACKAGE_ERROR", "archive entry create"}
		}
		if _, err = writer.Write(entries[name]); err != nil {
			return nil, &Error{"PACKAGE_ERROR", "archive entry write"}
		}
	}
	if err := archive.Close(); err != nil {
		return nil, &Error{"PACKAGE_ERROR", "archive close"}
	}
	if buffer.Len() > MaxPackageBytes {
		return nil, &Error{"LIMIT_EXCEEDED", "exchange package bytes"}
	}
	return buffer.Bytes(), nil
}

func validArchivePath(name string) bool {
	if name == "" || strings.Contains(name, "\\") || strings.HasPrefix(name, "/") || path.Clean(name) != name || strings.Contains(name, "..") {
		return false
	}
	if name == manifestPath || name == benchmarkPath || name == signaturePath {
		return true
	}
	if !strings.HasPrefix(name, "reports/") || !strings.HasSuffix(name, ".json") {
		return false
	}
	id := strings.TrimSuffix(strings.TrimPrefix(name, "reports/"), ".json")
	return model.ValidID(id)
}

func maxEntryBytes(name string) int64 {
	switch name {
	case manifestPath:
		return MaxManifestBytes
	case signaturePath:
		return MaxSignatureBytes
	case benchmarkPath:
		return governance.MaxDatasetBytes
	default:
		return analysis.MaxReportBytes
	}
}

func requiredReports(dataset governance.Dataset) map[string]bool {
	result := map[string]bool{}
	for _, item := range dataset.Cases {
		result[item.SourceReportID] = true
	}
	return result
}

func prefixedReports(ids []string) []string {
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		result = append(result, reportPath(id))
	}
	return result
}

func reportPath(id string) string { return "reports/" + id + ".json" }

func signingPayload(identityRaw, manifestRaw []byte) []byte {
	result := make([]byte, 0, len(signatureDomain)+len(identityRaw)+1+len(manifestRaw))
	result = append(result, []byte(signatureDomain)...)
	result = append(result, identityRaw...)
	result = append(result, 0)
	return append(result, manifestRaw...)
}

func validPrivacy(value PrivacyManifest) bool {
	if value.RawCommands || value.RawOutputs || value.WorkspacePaths || value.Credentials || value.Environment || value.PrivateKeys {
		return false
	}
	included, ok := canonicalStrings(value.IncludedEvidence)
	if !ok || !equalStrings(included, value.IncludedEvidence) {
		return false
	}
	omitted, ok := canonicalStrings(value.OmittedEvidence)
	return ok && equalStrings(omitted, value.OmittedEvidence)
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func (v Verification) String() string { return fmt.Sprintf("%s: %s", v.Status, v.Detail) }
