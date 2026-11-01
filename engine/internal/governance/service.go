package governance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/model"
)

type ReportRepository interface {
	List(context.Context) ([]analysis.ReportSummary, error)
	Get(context.Context, string) (analysis.Report, error)
	Reevaluate(context.Context, string, analysis.Policy) (analysis.Report, error)
	DeleteGoverned(context.Context, string) error
}

type Service struct {
	root    *os.Root
	reports ReportRepository
	gate    chan struct{}
}

func New(base string, reports ReportRepository) (*Service, error) {
	if !filepath.IsAbs(base) || reports == nil {
		return nil, &Error{"INVALID_ARGUMENT", "governance root or report repository"}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "governance root"}
	}
	directories := []string{"governance", "governance/v1", "governance/v1/datasets", "governance/v1/datasets/sha256", "governance/v1/lineage", "governance/v1/retention"}
	for _, directory := range directories {
		if err = root.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			root.Close()
			return nil, &Error{"STORAGE_ERROR", "governance directory"}
		}
		info, inspectErr := root.Lstat(directory)
		if inspectErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			root.Close()
			return nil, &Error{"UNSAFE_STORAGE", "governance directory"}
		}
	}
	service := &Service{root: root, reports: reports, gate: make(chan struct{}, 1)}
	for _, directory := range []string{"governance/v1/datasets/sha256", "governance/v1/lineage", "governance/v1/retention"} {
		if err = service.sync(directory); err != nil {
			root.Close()
			return nil, err
		}
	}
	return service, nil
}

func (s *Service) Close() error { return s.root.Close() }

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{FormatVersion: FormatVersion, Name: "manual-retain-v1", ArchiveAllowed: true, DeleteAllowed: true, Automatic: false}
}

func (s *Service) CreateDataset(ctx context.Context, spec CreateDataset) (DatasetRecord, error) {
	if err := s.enter(); err != nil {
		return DatasetRecord{}, err
	}
	defer s.leave()
	if ctx.Err() != nil {
		return DatasetRecord{}, &Error{"CANCELLED", "benchmark creation"}
	}
	if spec.BenchmarkID == "" {
		if spec.Version != 1 {
			return DatasetRecord{}, &Error{"INVALID_DATASET", "new benchmark family must start at version 1"}
		}
		spec.BenchmarkID = model.ID()
	} else if !model.ValidID(spec.BenchmarkID) {
		return DatasetRecord{}, &Error{"INVALID_DATASET", "benchmark family identity"}
	}
	dataset := Dataset{FormatVersion: FormatVersion, BenchmarkID: spec.BenchmarkID, Version: spec.Version, Name: spec.Name, CreatedAt: model.Now(), SupportedDimensions: normalizeDimensions(spec.SupportedDimensions), RequiredDimensions: normalizeDimensions(spec.RequiredDimensions), CompatibilityModel: "case-id-v1", ProvenanceLimitations: normalizeStrings(spec.ProvenanceLimitations), Cases: []Case{}}
	caseIDs := map[string]bool{}
	for _, draft := range spec.Cases {
		if caseIDs[draft.CaseID] {
			return DatasetRecord{}, &Error{"INVALID_DATASET", "duplicate case identity"}
		}
		caseIDs[draft.CaseID] = true
		report, err := s.reports.Get(ctx, draft.SourceReportID)
		if err != nil {
			return DatasetRecord{}, &Error{"MISSING_REFERENCE", "source report " + draft.SourceReportID}
		}
		if draft.ComparisonIndex < 0 || draft.ComparisonIndex >= len(report.Experiment.Comparisons) {
			return DatasetRecord{}, &Error{"MISSING_REFERENCE", "source comparison index"}
		}
		comparison := report.Experiment.Comparisons[draft.ComparisonIndex]
		dataset.Cases = append(dataset.Cases, Case{CaseID: draft.CaseID, SourceReportID: report.ReportID, ComparisonIndex: draft.ComparisonIndex, BaselineObservationID: comparison.BaselineObservation, ObservedObservationID: comparison.ObservedObservation, ComparisonSHA256: comparisonDigest(comparison), ExpectedStatus: comparison.Status, SupportedDimensions: normalizeDimensions(draft.SupportedDimensions)})
	}
	sort.Slice(dataset.Cases, func(i, j int) bool { return dataset.Cases[i].CaseID < dataset.Cases[j].CaseID })
	return s.storeDataset(ctx, dataset, nil)
}

func (s *Service) ImportDataset(ctx context.Context, raw []byte) (DatasetRecord, error) {
	if err := s.enter(); err != nil {
		return DatasetRecord{}, err
	}
	defer s.leave()
	dataset, contentID, err := decodeDataset(raw)
	if err != nil {
		return DatasetRecord{}, err
	}
	if err = s.validateDatasetReferences(ctx, dataset); err != nil {
		return DatasetRecord{}, err
	}
	return s.storeDataset(ctx, dataset, rawWithID{raw, contentID})
}

type rawWithID struct {
	raw []byte
	id  string
}

func (s *Service) storeDataset(ctx context.Context, dataset Dataset, supplied any) (DatasetRecord, error) {
	if err := validateDataset(dataset); err != nil {
		return DatasetRecord{}, err
	}
	if dataset.Version > 1 {
		previous, found, err := s.findDatasetVersion(ctx, dataset.BenchmarkID, dataset.Version-1)
		if err != nil {
			return DatasetRecord{}, err
		}
		if !found {
			return DatasetRecord{}, &Error{"MISSING_REFERENCE", "previous benchmark version"}
		}
		_ = previous
	}
	if existing, found, err := s.findDatasetVersion(ctx, dataset.BenchmarkID, dataset.Version); err != nil {
		return DatasetRecord{}, err
	} else if found {
		raw, _, readErr := s.readDatasetRaw(existing.ContentID)
		if readErr != nil {
			return DatasetRecord{}, readErr
		}
		candidate, _, sealErr := sealDataset(dataset)
		if sealErr != nil {
			return DatasetRecord{}, sealErr
		}
		if !bytes.Equal(raw, candidate) {
			return DatasetRecord{}, &Error{"VERSION_CONFLICT", "benchmark identity/version already has different content"}
		}
		existing.Duplicate = true
		return existing, nil
	}
	raw, contentID, err := sealDataset(dataset)
	if value, ok := supplied.(rawWithID); ok {
		raw, contentID = value.raw, value.id
	}
	if err != nil {
		return DatasetRecord{}, err
	}
	records, total, err := s.datasetInventory(ctx)
	if err != nil {
		return DatasetRecord{}, err
	}
	if len(records) >= MaxDatasets || total+int64(len(raw)) > MaxGovernanceBytes {
		return DatasetRecord{}, &Error{"LIMIT_EXCEEDED", "benchmark storage"}
	}
	name := datasetName(contentID)
	if existing, readErr := s.root.Open(name); readErr == nil {
		existing.Close()
		record := recordForDataset(contentID, dataset, int64(len(raw)), true, s.statusOrDefault(DatasetObject, contentID))
		return record, nil
	}
	if err = s.publish("governance/v1/datasets/sha256", name, contentID, raw); err != nil {
		return DatasetRecord{}, err
	}
	if _, err = s.appendRetention(ctx, DatasetObject, contentID, Active, DefaultRetentionPolicy(), "dataset stored"); err != nil {
		return DatasetRecord{}, err
	}
	return recordForDataset(contentID, dataset, int64(len(raw)), false, RetentionStatus{ObjectKind: DatasetObject, ObjectID: contentID, State: Active, Policy: DefaultRetentionPolicy()}), nil
}

func (s *Service) ListDatasets(ctx context.Context) ([]DatasetRecord, error) {
	records, _, err := s.datasetInventory(ctx)
	return records, err
}

func (s *Service) GetDataset(ctx context.Context, contentID string) (Dataset, error) {
	if ctx.Err() != nil {
		return Dataset{}, &Error{"CANCELLED", "benchmark read"}
	}
	_, dataset, err := s.readDatasetRaw(contentID)
	if err != nil {
		return Dataset{}, err
	}
	if err = s.validateDatasetReferences(ctx, dataset); err != nil {
		return Dataset{}, err
	}
	return dataset, nil
}

func (s *Service) ExportDataset(ctx context.Context, contentID string, consent bool) ([]byte, DatasetRecord, error) {
	if !consent {
		return nil, DatasetRecord{}, &Error{"CONSENT_REQUIRED", "benchmark export"}
	}
	raw, dataset, err := s.readDatasetRaw(contentID)
	if err != nil {
		return nil, DatasetRecord{}, err
	}
	return append([]byte(nil), raw...), recordForDataset(contentID, dataset, int64(len(raw)), false, s.statusOrDefault(DatasetObject, contentID)), nil
}

func (s *Service) CompareDatasetVersions(ctx context.Context, baselineID, candidateID string) (CompatibilityResult, error) {
	baseline, err := s.GetDataset(ctx, baselineID)
	if err != nil {
		return CompatibilityResult{}, err
	}
	candidate, err := s.GetDataset(ctx, candidateID)
	if err != nil {
		return CompatibilityResult{}, err
	}
	return CompareDatasets(baselineID, baseline, candidateID, candidate), nil
}

func (s *Service) EvaluateDataset(ctx context.Context, contentID string, policy analysis.Policy, parentLineageID string) (Lineage, error) {
	if err := s.enter(); err != nil {
		return Lineage{}, err
	}
	defer s.leave()
	dataset, err := s.GetDataset(ctx, contentID)
	if err != nil {
		return Lineage{}, err
	}
	if parentLineageID != "" {
		parent, parentErr := s.GetLineage(ctx, parentLineageID)
		if parentErr != nil || parent.BenchmarkID != dataset.BenchmarkID {
			return Lineage{}, &Error{"MISSING_REFERENCE", "parent lineage"}
		}
	}
	sources := []string{}
	seen := map[string]bool{}
	for _, item := range dataset.Cases {
		if !seen[item.SourceReportID] {
			seen[item.SourceReportID] = true
			sources = append(sources, item.SourceReportID)
		}
	}
	sort.Strings(sources)
	evaluations := make([]string, 0, len(sources))
	for _, source := range sources {
		report, reevaluateErr := s.reports.Reevaluate(ctx, source, policy)
		if reevaluateErr != nil {
			return Lineage{}, &Error{"EVALUATION_FAILED", "policy reevaluation"}
		}
		evaluations = append(evaluations, report.ReportID)
	}
	lineage := Lineage{FormatVersion: FormatVersion, LineageID: model.ID(), CreatedAt: model.Now(), DatasetContentID: contentID, BenchmarkID: dataset.BenchmarkID, DatasetVersion: dataset.Version, SourceReportIDs: sources, EvaluationReportIDs: evaluations, Policy: policy, ParentLineageID: parentLineageID}
	unsigned, _ := json.Marshal(lineage)
	lineage.SHA256 = digest(unsigned)
	raw, _ := json.Marshal(lineage)
	if len(raw) > MaxDatasetBytes {
		return Lineage{}, &Error{"LIMIT_EXCEEDED", "lineage bytes"}
	}
	if err = s.validateLineage(ctx, lineage); err != nil {
		return Lineage{}, err
	}
	if err = s.publish("governance/v1/lineage", lineageName(lineage.LineageID), lineage.LineageID, raw); err != nil {
		return Lineage{}, err
	}
	if _, err = s.appendRetention(ctx, LineageObject, lineage.LineageID, Active, DefaultRetentionPolicy(), "lineage stored"); err != nil {
		return Lineage{}, err
	}
	return lineage, nil
}

func (s *Service) ListLineages(ctx context.Context) ([]LineageSummary, error) {
	lineages, err := s.lineageInventory(ctx, true)
	if err != nil {
		return nil, err
	}
	result := make([]LineageSummary, 0, len(lineages))
	for _, lineage := range lineages {
		result = append(result, LineageSummary{LineageID: lineage.LineageID, CreatedAt: lineage.CreatedAt, DatasetContentID: lineage.DatasetContentID, BenchmarkID: lineage.BenchmarkID, DatasetVersion: lineage.DatasetVersion, PolicyName: lineage.Policy.Name, Integrity: "verified"})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].CreatedAt > result[j].CreatedAt })
	return result, nil
}

func (s *Service) GetLineage(ctx context.Context, id string) (Lineage, error) {
	return s.readLineage(ctx, id, true)
}

func (s *Service) readLineage(ctx context.Context, id string, validateReferences bool) (Lineage, error) {
	if !model.ValidID(id) {
		return Lineage{}, &Error{"INVALID_ARGUMENT", "lineage identity"}
	}
	raw, err := s.readRegular(lineageName(id), MaxDatasetBytes)
	if errors.Is(err, os.ErrNotExist) {
		return Lineage{}, &Error{"NOT_FOUND", "lineage"}
	}
	if err != nil {
		return Lineage{}, err
	}
	var lineage Lineage
	if strictCanonical(raw, &lineage) != nil || lineage.FormatVersion != FormatVersion || lineage.LineageID != id {
		return Lineage{}, &Error{"CORRUPT_LINEAGE", "lineage encoding"}
	}
	digestValue := lineage.SHA256
	lineage.SHA256 = ""
	unsigned, _ := json.Marshal(lineage)
	lineage.SHA256 = digestValue
	if digestValue != digest(unsigned) {
		return Lineage{}, &Error{"CORRUPT_LINEAGE", "lineage integrity"}
	}
	if validateReferences {
		err = s.validateLineage(ctx, lineage)
	} else {
		err = validateLineageContract(lineage)
	}
	if err != nil {
		return Lineage{}, err
	}
	return lineage, nil
}

func (s *Service) SetRetention(ctx context.Context, kind ObjectKind, id string, state State, policy RetentionPolicy, reason string) (RetentionStatus, error) {
	if err := s.enter(); err != nil {
		return RetentionStatus{}, err
	}
	defer s.leave()
	if !s.objectExists(ctx, kind, id) {
		return RetentionStatus{}, &Error{"NOT_FOUND", "retention object"}
	}
	return s.appendRetention(ctx, kind, id, state, policy, reason)
}

func (s *Service) Delete(ctx context.Context, kind ObjectKind, id string, consent bool) error {
	if !consent {
		return &Error{"CONSENT_REQUIRED", "governed deletion"}
	}
	if err := s.enter(); err != nil {
		return err
	}
	defer s.leave()
	status := s.statusOrDefault(kind, id)
	if status.State == Deleted {
		return nil
	}
	if status.State != DeletionPending && (!status.Policy.DeleteAllowed || status.State == Unmanaged) {
		return &Error{"RETENTION_BLOCKED", "object policy does not permit deletion"}
	}
	references, err := s.referenceGraph(ctx, kind, id)
	if err != nil {
		return err
	}
	for _, reference := range references {
		if reference.ToKind == kind && reference.ToID == id {
			return &Error{"REFERENCED", "delete dependent objects first: " + string(reference.FromKind) + " " + reference.FromID}
		}
	}
	if status.State != DeletionPending {
		if _, err = s.appendRetention(ctx, kind, id, DeletionPending, status.Policy, "explicit deletion confirmed"); err != nil {
			return err
		}
	}
	if s.objectExists(ctx, kind, id) {
		switch kind {
		case ReportObject:
			err = s.reports.DeleteGoverned(ctx, id)
		case DatasetObject:
			err = s.remove(datasetName(id), "governance/v1/datasets/sha256")
		case LineageObject:
			err = s.remove(lineageName(id), "governance/v1/lineage")
		default:
			err = &Error{"INVALID_ARGUMENT", "object kind"}
		}
		if err != nil {
			return err
		}
	}
	_, err = s.appendRetention(ctx, kind, id, Deleted, status.Policy, "physical deletion completed")
	return err
}

func (s *Service) Audit(ctx context.Context) (Audit, error) {
	audit := Audit{FormatVersion: FormatVersion, Issues: []AuditIssue{}, Retention: []RetentionStatus{}, References: []Reference{}}
	statuses, retentionBytes, tempCount, tempBytes, retentionErr := s.retentionInventory(ctx)
	if retentionErr != nil {
		audit.Issues = append(audit.Issues, AuditIssue{Code: "RETENTION_DAMAGED", Detail: retentionErr.Error()})
	} else {
		for _, status := range statuses {
			audit.Retention = append(audit.Retention, status)
			if !s.objectExists(ctx, status.ObjectKind, status.ObjectID) && status.State != Deleted && status.State != DeletionPending {
				audit.Usage.OrphanedMetadata++
				audit.Issues = append(audit.Issues, AuditIssue{Code: "ORPHANED_RETENTION", ObjectKind: status.ObjectKind, ObjectID: status.ObjectID, Detail: "retention metadata exists but the governed object is missing"})
			}
			if !s.objectExists(ctx, status.ObjectKind, status.ObjectID) && status.State == DeletionPending {
				audit.Issues = append(audit.Issues, AuditIssue{Code: "PENDING_DELETION", ObjectKind: status.ObjectKind, ObjectID: status.ObjectID, Detail: "physical deletion occurred but the tombstone is not committed"})
			}
		}
	}
	for _, status := range statuses {
		audit.Usage.RetentionEvents += int(status.Revision)
	}
	audit.Usage.RetentionBytes = retentionBytes
	audit.Usage.TemporaryFiles = tempCount
	audit.Usage.TemporaryBytes = tempBytes
	reportSummaries := []analysis.ReportSummary{}
	reportSeen := map[string]bool{}
	reportEntries, reportErr := s.readDirectory("analysis/v1/reports", analysis.MaxReports+1024)
	if reportErr != nil {
		audit.Issues = append(audit.Issues, AuditIssue{Code: "REPORT_INVENTORY_DAMAGED", ObjectKind: ReportObject, Detail: reportErr.Error()})
	} else {
		for _, entry := range reportEntries {
			info, infoErr := entry.Info()
			if infoErr != nil {
				continue
			}
			name := entry.Name()
			if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
				audit.Usage.TemporaryFiles++
				audit.Usage.TemporaryBytes += info.Size()
				continue
			}
			id := strings.TrimSuffix(name, ".json")
			if !model.ValidID(id) || !strings.HasSuffix(name, ".json") {
				audit.Issues = append(audit.Issues, AuditIssue{Code: "DAMAGED", ObjectKind: ReportObject, Detail: "unexpected report entry"})
				continue
			}
			report, getErr := s.reports.Get(ctx, id)
			if getErr != nil {
				audit.Usage.DamagedObjects++
				audit.Issues = append(audit.Issues, AuditIssue{Code: "DAMAGED", ObjectKind: ReportObject, ObjectID: id, Detail: getErr.Error()})
				continue
			}
			reportSummaries = append(reportSummaries, analysis.ReportSummary{ReportID: report.ReportID})
			reportSeen[report.ReportID] = true
			audit.Usage.Reports++
			audit.Usage.ReportBytes += info.Size()
		}
	}
	repositorySummaries, repositoryErr := s.reports.List(ctx)
	if repositoryErr != nil && len(reportEntries) == 0 {
		audit.Issues = append(audit.Issues, AuditIssue{Code: "REPORT_INVENTORY_DAMAGED", ObjectKind: ReportObject, Detail: repositoryErr.Error()})
	}
	for _, summary := range repositorySummaries {
		if reportSeen[summary.ReportID] {
			continue
		}
		report, getErr := s.reports.Get(ctx, summary.ReportID)
		if getErr != nil {
			continue
		}
		raw, _ := json.Marshal(report)
		reportSummaries = append(reportSummaries, summary)
		audit.Usage.Reports++
		audit.Usage.ReportBytes += int64(len(raw))
	}
	datasets, datasetBytes, datasetErr := s.datasetInventory(ctx)
	if datasetErr != nil {
		audit.Issues = append(audit.Issues, AuditIssue{Code: "DATASET_INVENTORY_DAMAGED", ObjectKind: DatasetObject, Detail: datasetErr.Error()})
	}
	for _, record := range datasets {
		audit.Usage.Datasets++
		audit.Usage.DatasetBytes += record.Bytes
		if record.Integrity != "verified" {
			audit.Usage.DamagedObjects++
			audit.Issues = append(audit.Issues, AuditIssue{Code: "DAMAGED", ObjectKind: DatasetObject, ObjectID: record.ContentID, Detail: "benchmark bytes failed validation"})
		}
	}
	_ = datasetBytes
	for _, directory := range []string{"governance/v1/datasets/sha256", "governance/v1/lineage"} {
		entries, entryErr := s.readDirectory(directory, MaxDatasets+MaxLineages+1024)
		if entryErr != nil {
			continue
		}
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), ".") && strings.HasSuffix(entry.Name(), ".tmp") {
				if info, infoErr := entry.Info(); infoErr == nil {
					audit.Usage.TemporaryFiles++
					audit.Usage.TemporaryBytes += info.Size()
				}
			}
		}
	}
	lineages, lineageErr := s.lineageInventory(ctx, false)
	if lineageErr != nil {
		audit.Issues = append(audit.Issues, AuditIssue{Code: "LINEAGE_DAMAGED", ObjectKind: LineageObject, Detail: lineageErr.Error()})
	}
	for _, lineage := range lineages {
		raw, _ := json.Marshal(lineage)
		audit.Usage.Lineages++
		audit.Usage.LineageBytes += int64(len(raw))
	}
	references, referenceErr := s.references(ctx)
	if referenceErr != nil {
		audit.Issues = append(audit.Issues, AuditIssue{Code: "REFERENCE_AUDIT_FAILED", Detail: referenceErr.Error()})
	} else {
		audit.References = references
		for _, reference := range references {
			if !s.objectExists(ctx, reference.ToKind, reference.ToID) {
				audit.Usage.MissingReferences++
				audit.Issues = append(audit.Issues, AuditIssue{Code: "MISSING_REFERENCE", ObjectKind: reference.ToKind, ObjectID: reference.ToID, Detail: string(reference.FromKind) + " " + reference.FromID + " requires this object"})
			}
		}
	}
	for _, object := range s.liveObjects(ctx, datasets, lineages, reportSummaries) {
		if _, ok := statuses[retentionKey(object.ObjectKind, object.ObjectID)]; !ok {
			audit.Usage.UnmanagedObjects++
			audit.Retention = append(audit.Retention, object)
		}
	}
	audit.Usage.TotalBytes = audit.Usage.ReportBytes + audit.Usage.DatasetBytes + audit.Usage.LineageBytes + audit.Usage.RetentionBytes + audit.Usage.TemporaryBytes
	sort.Slice(audit.Retention, func(i, j int) bool {
		return retentionKey(audit.Retention[i].ObjectKind, audit.Retention[i].ObjectID) < retentionKey(audit.Retention[j].ObjectKind, audit.Retention[j].ObjectID)
	})
	return audit, nil
}

func (s *Service) validateDatasetReferences(ctx context.Context, dataset Dataset) error {
	for _, item := range dataset.Cases {
		report, err := s.reports.Get(ctx, item.SourceReportID)
		if err != nil || item.ComparisonIndex < 0 || item.ComparisonIndex >= len(report.Experiment.Comparisons) {
			return &Error{"MISSING_REFERENCE", "dataset source report or comparison"}
		}
		comparison := report.Experiment.Comparisons[item.ComparisonIndex]
		if comparison.BaselineObservation != item.BaselineObservationID || comparison.ObservedObservation != item.ObservedObservationID || comparisonDigest(comparison) != item.ComparisonSHA256 || comparison.Status != item.ExpectedStatus {
			return &Error{"REFERENCE_MISMATCH", "dataset case provenance"}
		}
		for _, dimension := range item.SupportedDimensions {
			if !reportSupportsDimension(report, comparison, dimension) {
				return &Error{"REFERENCE_MISMATCH", "declared case dimension is unavailable"}
			}
		}
	}
	return nil
}

func (s *Service) validateLineage(ctx context.Context, lineage Lineage) error {
	if err := validateLineageContract(lineage); err != nil {
		return err
	}
	dataset, err := s.GetDataset(ctx, lineage.DatasetContentID)
	if err != nil || dataset.BenchmarkID != lineage.BenchmarkID || dataset.Version != lineage.DatasetVersion {
		return &Error{"MISSING_REFERENCE", "lineage dataset"}
	}
	for index, source := range lineage.SourceReportIDs {
		if !model.ValidID(source) || !model.ValidID(lineage.EvaluationReportIDs[index]) {
			return &Error{"CORRUPT_LINEAGE", "lineage report identity"}
		}
		if _, err = s.reports.Get(ctx, source); err != nil {
			return &Error{"MISSING_REFERENCE", "lineage source report"}
		}
		sourceReport, _ := s.reports.Get(ctx, source)
		result, resultErr := s.reports.Get(ctx, lineage.EvaluationReportIDs[index])
		sourceExperiment, _ := json.Marshal(sourceReport.Experiment)
		resultExperiment, _ := json.Marshal(result.Experiment)
		lineagePolicy, _ := json.Marshal(lineage.Policy)
		resultPolicy, _ := json.Marshal(result.Policy)
		if resultErr != nil || !bytes.Equal(lineagePolicy, resultPolicy) || !bytes.Equal(sourceExperiment, resultExperiment) {
			return &Error{"MISSING_REFERENCE", "lineage evaluation report"}
		}
	}
	if lineage.ParentLineageID != "" && !model.ValidID(lineage.ParentLineageID) {
		return &Error{"CORRUPT_LINEAGE", "parent lineage identity"}
	}
	if lineage.ParentLineageID != "" {
		parent, parentErr := s.readLineage(ctx, lineage.ParentLineageID, false)
		if parentErr != nil || parent.BenchmarkID != lineage.BenchmarkID {
			return &Error{"MISSING_REFERENCE", "parent lineage"}
		}
	}
	return nil
}

func reportSupportsDimension(report analysis.Report, comparison analysis.Comparison, dimension analysis.Dimension) bool {
	if dimension == analysis.EventType || dimension == analysis.Ordering || dimension == analysis.Config {
		return true
	}
	matched := 0
	for _, observation := range report.Experiment.Observations {
		if observation.ObservationID != comparison.BaselineObservation && observation.ObservationID != comparison.ObservedObservation {
			continue
		}
		matched++
		for _, gap := range append(append([]analysis.Gap{}, observation.Missing...), observation.Unsupported...) {
			if gap.Dimension == dimension {
				return false
			}
		}
		if dimension == analysis.Snapshot && len(observation.Snapshots) == 0 {
			return false
		}
		if dimension != analysis.Snapshot {
			found := false
			for _, event := range observation.Events {
				switch dimension {
				case analysis.Input:
					found = found || event.InputSHA256 != ""
				case analysis.Output:
					found = found || event.OutputSHA256 != ""
				case analysis.Tool:
					found = found || event.ToolSHA256 != ""
				case analysis.Request:
					found = found || event.RequestSHA256 != ""
				case analysis.Response:
					found = found || event.ResponseSHA256 != ""
				case analysis.Workspace:
					found = found || event.WorkspaceSHA256 != ""
				}
			}
			if !found {
				return false
			}
		}
	}
	return matched == 2
}

func validateLineageContract(lineage Lineage) error {
	if lineage.FormatVersion != FormatVersion || !model.ValidID(lineage.LineageID) || !validDigest(lineage.DatasetContentID) || !model.ValidID(lineage.BenchmarkID) || lineage.DatasetVersion < 1 || len(lineage.SourceReportIDs) == 0 || len(lineage.SourceReportIDs) != len(lineage.EvaluationReportIDs) {
		return &Error{"CORRUPT_LINEAGE", "lineage contract"}
	}
	created, err := time.Parse(time.RFC3339Nano, lineage.CreatedAt)
	if err != nil || created.Location() != time.UTC {
		return &Error{"CORRUPT_LINEAGE", "lineage timestamp"}
	}
	lastSource := ""
	seenEvaluations := map[string]bool{}
	for index, source := range lineage.SourceReportIDs {
		evaluation := lineage.EvaluationReportIDs[index]
		if !model.ValidID(source) || source <= lastSource || !model.ValidID(evaluation) || seenEvaluations[evaluation] {
			return &Error{"CORRUPT_LINEAGE", "lineage report identity/order"}
		}
		lastSource = source
		seenEvaluations[evaluation] = true
	}
	if lineage.ParentLineageID != "" && !model.ValidID(lineage.ParentLineageID) {
		return &Error{"CORRUPT_LINEAGE", "parent lineage identity"}
	}
	return nil
}

func recordForDataset(id string, dataset Dataset, size int64, duplicate bool, state RetentionStatus) DatasetRecord {
	return DatasetRecord{ContentID: id, BenchmarkID: dataset.BenchmarkID, Version: dataset.Version, Name: dataset.Name, Cases: len(dataset.Cases), Bytes: size, Integrity: "verified", Duplicate: duplicate, Lifecycle: state.State, Warnings: []string{"SHA-256 provides integrity and equality only; it does not prove authorship or authenticity.", "Source reports contain bounded observable evidence and explicit provenance limitations."}}
}

func (s *Service) datasetInventory(ctx context.Context) ([]DatasetRecord, int64, error) {
	entries, err := s.readDirectory("governance/v1/datasets/sha256", MaxDatasets+1024)
	if err != nil {
		return nil, 0, err
	}
	records := []DatasetRecord{}
	var total int64
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, 0, &Error{"CANCELLED", "dataset inventory"}
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
			continue
		}
		id := strings.TrimSuffix(name, ".benchmark")
		info, infoErr := entry.Info()
		if infoErr != nil || !validDigest(id) || !strings.HasSuffix(name, ".benchmark") {
			return nil, 0, &Error{"CORRUPT_DATASET", "unexpected dataset entry"}
		}
		total += info.Size()
		raw, dataset, readErr := s.readDatasetRaw(id)
		if readErr != nil {
			records = append(records, DatasetRecord{ContentID: id, Bytes: info.Size(), Integrity: "damaged", Lifecycle: s.statusOrDefault(DatasetObject, id).State, Warnings: []string{readErr.Error()}})
			continue
		}
		records = append(records, recordForDataset(id, dataset, int64(len(raw)), false, s.statusOrDefault(DatasetObject, id)))
	}
	if len(records) > MaxDatasets || total > MaxGovernanceBytes {
		return nil, 0, &Error{"LIMIT_EXCEEDED", "dataset inventory"}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ContentID < records[j].ContentID })
	return records, total, nil
}

func (s *Service) findDatasetVersion(ctx context.Context, benchmarkID string, version uint64) (DatasetRecord, bool, error) {
	records, _, err := s.datasetInventory(ctx)
	if err != nil {
		return DatasetRecord{}, false, err
	}
	for _, record := range records {
		if record.Integrity == "verified" && record.BenchmarkID == benchmarkID && record.Version == version {
			return record, true, nil
		}
	}
	return DatasetRecord{}, false, nil
}

func (s *Service) readDatasetRaw(id string) ([]byte, Dataset, error) {
	if !validDigest(id) {
		return nil, Dataset{}, &Error{"INVALID_ARGUMENT", "dataset content identity"}
	}
	raw, err := s.readRegular(datasetName(id), MaxDatasetBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, Dataset{}, &Error{"NOT_FOUND", "dataset"}
	}
	if err != nil {
		return nil, Dataset{}, err
	}
	dataset, actual, err := decodeDataset(raw)
	if err != nil || actual != id {
		return nil, Dataset{}, &Error{"CORRUPT_DATASET", "dataset content identity"}
	}
	return raw, dataset, nil
}

func (s *Service) lineageInventory(ctx context.Context, validate bool) ([]Lineage, error) {
	entries, err := s.readDirectory("governance/v1/lineage", MaxLineages+1024)
	if err != nil {
		return nil, err
	}
	result := []Lineage{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !model.ValidID(id) || !strings.HasSuffix(name, ".json") {
			return nil, &Error{"CORRUPT_LINEAGE", "unexpected lineage entry"}
		}
		lineage, getErr := s.readLineage(ctx, id, validate)
		if getErr != nil {
			return nil, getErr
		}
		result = append(result, lineage)
	}
	if len(result) > MaxLineages {
		return nil, &Error{"LIMIT_EXCEEDED", "lineage count"}
	}
	return result, nil
}

func (s *Service) references(ctx context.Context) ([]Reference, error) {
	return s.referenceGraph(ctx, "", "")
}

func (s *Service) referenceGraph(ctx context.Context, deletingKind ObjectKind, deletingID string) ([]Reference, error) {
	references := []Reference{}
	datasets, _, err := s.datasetInventory(ctx)
	if err != nil {
		return nil, err
	}
	for _, record := range datasets {
		if record.Integrity != "verified" {
			if deletingKind == DatasetObject && deletingID == record.ContentID {
				continue
			}
			return nil, &Error{"CORRUPT_REFERENCE_GRAPH", "damaged dataset may contain unknown references"}
		}
		_, dataset, readErr := s.readDatasetRaw(record.ContentID)
		if readErr != nil {
			continue
		}
		for _, item := range dataset.Cases {
			references = append(references, Reference{FromKind: DatasetObject, FromID: record.ContentID, ToKind: ReportObject, ToID: item.SourceReportID, Relation: "source_report"})
		}
	}
	lineages, lineageErr := s.lineagesForGraph(ctx, func(id string) bool {
		return deletingKind == LineageObject && deletingID == id
	})
	if lineageErr != nil {
		return nil, lineageErr
	}
	for _, lineage := range lineages {
		references = append(references, Reference{FromKind: LineageObject, FromID: lineage.LineageID, ToKind: DatasetObject, ToID: lineage.DatasetContentID, Relation: "evaluated_dataset"})
		for _, id := range lineage.SourceReportIDs {
			references = append(references, Reference{FromKind: LineageObject, FromID: lineage.LineageID, ToKind: ReportObject, ToID: id, Relation: "source_report"})
		}
		for _, id := range lineage.EvaluationReportIDs {
			references = append(references, Reference{FromKind: LineageObject, FromID: lineage.LineageID, ToKind: ReportObject, ToID: id, Relation: "evaluation_report"})
		}
		if lineage.ParentLineageID != "" {
			references = append(references, Reference{FromKind: LineageObject, FromID: lineage.LineageID, ToKind: LineageObject, ToID: lineage.ParentLineageID, Relation: "parent_lineage"})
		}
	}
	sort.Slice(references, func(i, j int) bool {
		return string(references[i].FromKind)+references[i].FromID+references[i].Relation < string(references[j].FromKind)+references[j].FromID+references[j].Relation
	})
	return references, nil
}

func (s *Service) lineagesForGraph(ctx context.Context, ignore func(string) bool) ([]Lineage, error) {
	entries, err := s.readDirectory("governance/v1/lineage", MaxLineages+1024)
	if err != nil {
		return nil, err
	}
	result := []Lineage{}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if ignore != nil && ignore(id) {
			continue
		}
		lineage, readErr := s.readLineage(ctx, id, false)
		if readErr != nil {
			return nil, &Error{"CORRUPT_REFERENCE_GRAPH", "damaged lineage may contain unknown references"}
		}
		result = append(result, lineage)
	}
	return result, nil
}

func (s *Service) appendRetention(ctx context.Context, kind ObjectKind, id string, state State, policy RetentionPolicy, reason string) (RetentionStatus, error) {
	if err := validateRetentionPolicy(policy); err != nil || !validObjectID(kind, id) || len(reason) < 1 || len(reason) > 256 {
		return RetentionStatus{}, &Error{"INVALID_RETENTION", "policy, object, or reason"}
	}
	statuses, _, _, _, err := s.retentionInventory(ctx)
	if err != nil {
		return RetentionStatus{}, err
	}
	current, ok := statuses[retentionKey(kind, id)]
	if !ok {
		current = RetentionStatus{ObjectKind: kind, ObjectID: id, State: Unmanaged, Policy: policy}
	}
	if !allowedTransition(current.State, state, policy) {
		return RetentionStatus{}, &Error{"INVALID_RETENTION", "lifecycle transition"}
	}
	event := RetentionEvent{FormatVersion: FormatVersion, EventID: model.ID(), ObjectKind: kind, ObjectID: id, Revision: current.Revision + 1, PreviousState: current.State, State: state, Policy: policy, Reason: reason, CreatedAt: model.Now()}
	unsigned, _ := json.Marshal(event)
	event.SHA256 = digest(unsigned)
	raw, _ := json.Marshal(event)
	if err = s.publish("governance/v1/retention", retentionName(event.EventID), event.EventID, raw); err != nil {
		return RetentionStatus{}, err
	}
	return statusFromEvent(event), nil
}

func (s *Service) retentionInventory(ctx context.Context) (map[string]RetentionStatus, int64, int, int64, error) {
	entries, err := s.readDirectory("governance/v1/retention", MaxRetentionEvents+1024)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	statuses := map[string]RetentionStatus{}
	retentionEvents := []RetentionEvent{}
	var total, tempBytes int64
	temps := 0
	events := 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, 0, 0, 0, &Error{"CANCELLED", "retention inventory"}
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return nil, 0, 0, 0, &Error{"CORRUPT_RETENTION", "retention metadata"}
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
			temps++
			tempBytes += info.Size()
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !model.ValidID(id) || !strings.HasSuffix(name, ".json") {
			return nil, 0, 0, 0, &Error{"CORRUPT_RETENTION", "unexpected retention entry"}
		}
		raw, readErr := s.readRegular("governance/v1/retention/"+name, 64*1024)
		if readErr != nil {
			return nil, 0, 0, 0, readErr
		}
		var event RetentionEvent
		if strictCanonical(raw, &event) != nil || event.EventID != id || event.FormatVersion != FormatVersion {
			return nil, 0, 0, 0, &Error{"CORRUPT_RETENTION", "retention encoding"}
		}
		digestValue := event.SHA256
		event.SHA256 = ""
		unsigned, _ := json.Marshal(event)
		event.SHA256 = digestValue
		if digestValue != digest(unsigned) || validateRetentionPolicy(event.Policy) != nil || !validObjectID(event.ObjectKind, event.ObjectID) {
			return nil, 0, 0, 0, &Error{"CORRUPT_RETENTION", "retention integrity"}
		}
		retentionEvents = append(retentionEvents, event)
		total += int64(len(raw))
		events++
	}
	if events > MaxRetentionEvents || total > MaxGovernanceBytes {
		return nil, 0, 0, 0, &Error{"LIMIT_EXCEEDED", "retention inventory"}
	}
	sort.Slice(retentionEvents, func(i, j int) bool {
		left := retentionKey(retentionEvents[i].ObjectKind, retentionEvents[i].ObjectID)
		right := retentionKey(retentionEvents[j].ObjectKind, retentionEvents[j].ObjectID)
		if left == right {
			if retentionEvents[i].Revision == retentionEvents[j].Revision {
				return retentionEvents[i].EventID < retentionEvents[j].EventID
			}
			return retentionEvents[i].Revision < retentionEvents[j].Revision
		}
		return left < right
	})
	for _, event := range retentionEvents {
		key := retentionKey(event.ObjectKind, event.ObjectID)
		current, ok := statuses[key]
		if event.Revision != current.Revision+1 || ok && event.PreviousState != current.State || !ok && event.PreviousState != Unmanaged || !allowedTransition(event.PreviousState, event.State, event.Policy) {
			return nil, 0, 0, 0, &Error{"CORRUPT_RETENTION", "retention revision or transition"}
		}
		statuses[key] = statusFromEvent(event)
	}
	return statuses, total, temps, tempBytes, nil
}

func validateRetentionPolicy(policy RetentionPolicy) error {
	if policy.FormatVersion != FormatVersion || policy.Name == "" || len(policy.Name) > 128 || policy.Automatic {
		return &Error{"INVALID_RETENTION", "policy version, name, or automatic behavior"}
	}
	if policy.ExpiresAt != "" {
		value, err := time.Parse(time.RFC3339Nano, policy.ExpiresAt)
		if err != nil || value.Location() != time.UTC {
			return &Error{"INVALID_RETENTION", "policy expiration"}
		}
	}
	return nil
}

func allowedTransition(from, to State, policy RetentionPolicy) bool {
	if to == DeletionPending && !policy.DeleteAllowed || to == Archived && !policy.ArchiveAllowed || to == Deleted && from != DeletionPending {
		return false
	}
	switch from {
	case Unmanaged:
		return to == Active || to == Damaged
	case Active:
		return to == Archived || to == Expired || to == Damaged || to == DeletionPending
	case Archived:
		return to == Active || to == Expired || to == Damaged || to == DeletionPending
	case Expired:
		return to == Active || to == Archived || to == Damaged || to == DeletionPending
	case Damaged:
		return to == Active || to == DeletionPending
	case DeletionPending:
		return to == Deleted || to == Damaged
	default:
		return false
	}
}

func (s *Service) statusOrDefault(kind ObjectKind, id string) RetentionStatus {
	statuses, _, _, _, err := s.retentionInventory(context.Background())
	if err == nil {
		if status, ok := statuses[retentionKey(kind, id)]; ok {
			return status
		}
	}
	return RetentionStatus{ObjectKind: kind, ObjectID: id, State: Unmanaged, Policy: DefaultRetentionPolicy()}
}

func statusFromEvent(event RetentionEvent) RetentionStatus {
	return RetentionStatus{ObjectKind: event.ObjectKind, ObjectID: event.ObjectID, Revision: event.Revision, State: event.State, Policy: event.Policy, Reason: event.Reason, UpdatedAt: event.CreatedAt}
}

func (s *Service) objectExists(ctx context.Context, kind ObjectKind, id string) bool {
	switch kind {
	case ReportObject:
		if !model.ValidID(id) {
			return false
		}
		if _, err := s.reports.Get(ctx, id); err == nil {
			return true
		}
		info, err := s.root.Lstat("analysis/v1/reports/" + id + ".json")
		return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0
	case DatasetObject:
		info, err := s.root.Lstat(datasetName(id))
		return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0
	case LineageObject:
		info, err := s.root.Lstat(lineageName(id))
		return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0077 == 0
	default:
		return false
	}
}

func validObjectID(kind ObjectKind, id string) bool {
	return kind == DatasetObject && validDigest(id) || (kind == ReportObject || kind == LineageObject) && model.ValidID(id)
}

func (s *Service) liveObjects(_ context.Context, datasets []DatasetRecord, lineages []Lineage, reports []analysis.ReportSummary) []RetentionStatus {
	result := []RetentionStatus{}
	for _, report := range reports {
		result = append(result, RetentionStatus{ObjectKind: ReportObject, ObjectID: report.ReportID, State: Unmanaged, Policy: DefaultRetentionPolicy()})
	}
	for _, dataset := range datasets {
		result = append(result, RetentionStatus{ObjectKind: DatasetObject, ObjectID: dataset.ContentID, State: Unmanaged, Policy: DefaultRetentionPolicy()})
	}
	for _, lineage := range lineages {
		result = append(result, RetentionStatus{ObjectKind: LineageObject, ObjectID: lineage.LineageID, State: Unmanaged, Policy: DefaultRetentionPolicy()})
	}
	return result
}

func (s *Service) publish(directory, name, identity string, raw []byte) error {
	temp := directory + "/." + identity + "." + model.ID() + ".tmp"
	file, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return &Error{"STORAGE_ERROR", "temporary create"}
	}
	defer s.root.Remove(temp)
	if _, err = file.Write(raw); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		return &Error{"STORAGE_ERROR", "temporary write"}
	}
	if err = s.root.Link(temp, name); err != nil {
		return &Error{"STORAGE_ERROR", "no-replace publication"}
	}
	if err = s.root.Remove(temp); err != nil {
		return &Error{"STORAGE_ERROR", "temporary cleanup"}
	}
	return s.sync(directory)
}

func (s *Service) remove(name, directory string) error {
	info, err := s.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return &Error{"UNSAFE_STORAGE", "deletion target"}
	}
	if err = s.root.Remove(name); err != nil {
		return &Error{"STORAGE_ERROR", "physical deletion"}
	}
	return s.sync(directory)
}

func (s *Service) readRegular(name string, limit int64) ([]byte, error) {
	info, err := s.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > limit {
		return nil, &Error{"UNSAFE_STORAGE", "governance file"}
	}
	file, err := s.root.Open(name)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "governance file open"}
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, limit+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || int64(len(raw)) > limit {
		return nil, &Error{"STORAGE_ERROR", "governance file read"}
	}
	return raw, nil
}

func (s *Service) readDirectory(name string, limit int) ([]os.DirEntry, error) {
	directory, err := s.root.Open(name)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "governance inventory"}
	}
	defer directory.Close()
	entries, err := directory.ReadDir(limit)
	if err != nil && err != io.EOF {
		return nil, &Error{"STORAGE_ERROR", "governance inventory"}
	}
	if len(entries) == limit && err == nil {
		return nil, &Error{"LIMIT_EXCEEDED", "governance directory entries"}
	}
	return entries, nil
}

func strictCanonical(raw []byte, value any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return errors.New("strict JSON")
	}
	canonical, _ := json.Marshal(value)
	if !bytes.Equal(canonical, raw) {
		return errors.New("noncanonical JSON")
	}
	return nil
}

func (s *Service) sync(directory string) error {
	file, err := s.root.Open(directory)
	if err != nil {
		return &Error{"STORAGE_ERROR", "directory sync"}
	}
	err = file.Sync()
	file.Close()
	if err != nil {
		return &Error{"STORAGE_ERROR", "directory sync"}
	}
	return nil
}

func datasetName(id string) string                   { return "governance/v1/datasets/sha256/" + id + ".benchmark" }
func lineageName(id string) string                   { return "governance/v1/lineage/" + id + ".json" }
func retentionName(id string) string                 { return "governance/v1/retention/" + id + ".json" }
func retentionKey(kind ObjectKind, id string) string { return string(kind) + "\x00" + id }

func (s *Service) enter() error {
	select {
	case s.gate <- struct{}{}:
		return nil
	default:
		return &Error{"BUSY", "governance writer"}
	}
}

func (s *Service) leave() { <-s.gate }
