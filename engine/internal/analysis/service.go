package analysis

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

	"chronosync.local/engine/internal/model"
)

const maxReportStorage = 256 * 1024 * 1024

type Service struct {
	root    *os.Root
	builder Builder
	gate    chan struct{}
}

func New(base string, history HistorySource, snapshots SnapshotSource) (*Service, error) {
	if !filepath.IsAbs(base) || history == nil {
		return nil, &Error{"INVALID_ARGUMENT", "analysis root or history source"}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "analysis root"}
	}
	for _, directory := range []string{"analysis", "analysis/v1", "analysis/v1/reports"} {
		if err = root.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			root.Close()
			return nil, &Error{"STORAGE_ERROR", "analysis report directory"}
		}
		info, inspectErr := root.Lstat(directory)
		if inspectErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			root.Close()
			return nil, &Error{"UNSAFE_STORAGE", "analysis report directory"}
		}
	}
	service := &Service{root: root, builder: Builder{History: history, Snapshots: snapshots}, gate: make(chan struct{}, 1)}
	if err = service.syncDirectory(); err != nil {
		root.Close()
		return nil, err
	}
	return service, nil
}

func (s *Service) Close() error { return s.root.Close() }
func (s *Service) Observe(ctx context.Context, runID string) (Observation, error) {
	return s.builder.Observe(ctx, runID)
}

func (s *Service) Compare(ctx context.Context, runIDs []string, policy Policy) (Report, error) {
	experiment, err := Run(ctx, s.builder, runIDs)
	if err != nil {
		return Report{}, err
	}
	return s.save(ctx, experiment, policy)
}

func (s *Service) Reevaluate(ctx context.Context, reportID string, policy Policy) (Report, error) {
	report, err := s.Get(ctx, reportID)
	if err != nil {
		return Report{}, err
	}
	return s.save(ctx, report.Experiment, policy)
}

func (s *Service) save(ctx context.Context, experiment Experiment, policy Policy) (Report, error) {
	evaluation, err := Evaluate(experiment, policy)
	if err != nil {
		return Report{}, err
	}
	report := Report{FormatVersion: FormatVersion, ReportID: model.ID(), CreatedAt: model.Now(), Experiment: experiment, Policy: policy, Evaluation: evaluation}
	unsigned, _ := json.Marshal(report)
	report.SHA256 = hashBytes(unsigned)
	raw, err := json.Marshal(report)
	if err != nil || len(raw) == 0 || len(raw) > MaxReportBytes {
		return Report{}, &Error{"LIMIT_EXCEEDED", "analysis report bytes"}
	}
	if err = s.enter(); err != nil {
		return Report{}, err
	}
	defer s.leave()
	if ctx.Err() != nil {
		return Report{}, &Error{"CANCELLED", "analysis report save"}
	}
	summaries, total, err := s.inventory(ctx, false)
	if err != nil {
		return Report{}, err
	}
	if len(summaries) >= MaxReports || total+int64(len(raw)) > maxReportStorage {
		return Report{}, &Error{"LIMIT_EXCEEDED", "analysis report storage"}
	}
	name := reportName(report.ReportID)
	temp := "analysis/v1/reports/." + report.ReportID + "." + model.ID() + ".tmp"
	file, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Report{}, &Error{"STORAGE_ERROR", "analysis report temporary create"}
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
		return Report{}, &Error{"STORAGE_ERROR", "analysis report write"}
	}
	if err = s.root.Link(temp, name); err != nil {
		return Report{}, &Error{"STORAGE_ERROR", "analysis report publish"}
	}
	if err = s.root.Remove(temp); err != nil {
		return Report{}, &Error{"STORAGE_ERROR", "analysis report temporary cleanup"}
	}
	if err = s.syncDirectory(); err != nil {
		return Report{}, err
	}
	return report, nil
}

func (s *Service) List(ctx context.Context) ([]ReportSummary, error) {
	summaries, _, err := s.inventory(ctx, true)
	return summaries, err
}

func (s *Service) Get(ctx context.Context, id string) (Report, error) {
	if ctx.Err() != nil {
		return Report{}, &Error{"CANCELLED", "analysis report read"}
	}
	if !model.ValidID(id) {
		return Report{}, &Error{"INVALID_ARGUMENT", "analysis report identity"}
	}
	return s.read(reportName(id), id)
}

// ExportCanonical returns the exact validated report bytes for a signed
// exchange package. It is internal and does not expose a filesystem path.
func (s *Service) ExportCanonical(ctx context.Context, id string) ([]byte, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	file, err := s.root.Open(reportName(id))
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "analysis report export"}
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, MaxReportBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(raw) > MaxReportBytes {
		return nil, &Error{"CORRUPT_REPORT", "analysis report export"}
	}
	return raw, nil
}

// ValidateCanonical validates detached report bytes without publishing them.
func ValidateCanonical(raw []byte) (Report, error) {
	if len(raw) == 0 || len(raw) > MaxReportBytes {
		return Report{}, &Error{"LIMIT_EXCEEDED", "analysis report bytes"}
	}
	var report Report
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&report) != nil || decoder.Decode(&struct{}{}) != io.EOF || report.FormatVersion != FormatVersion || !model.ValidID(report.ReportID) {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis report validation"}
	}
	canonical, _ := json.Marshal(report)
	if !bytes.Equal(canonical, raw) {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis report is not canonical"}
	}
	digest := report.SHA256
	report.SHA256 = ""
	unsigned, _ := json.Marshal(report)
	report.SHA256 = digest
	if digest == "" || digest != hashBytes(unsigned) {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis report integrity"}
	}
	evaluation, err := Evaluate(report.Experiment, report.Policy)
	if err != nil {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis report contract"}
	}
	expected, _ := json.Marshal(evaluation)
	actual, _ := json.Marshal(report.Evaluation)
	if !bytes.Equal(expected, actual) {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis evaluation mismatch"}
	}
	return report, nil
}

// ImportCanonical publishes a fully preflighted signed report idempotently.
func (s *Service) ImportCanonical(ctx context.Context, raw []byte) (Report, bool, error) {
	report, err := ValidateCanonical(raw)
	if err != nil {
		return Report{}, false, err
	}
	if err = s.enter(); err != nil {
		return Report{}, false, err
	}
	defer s.leave()
	if ctx.Err() != nil {
		return Report{}, false, &Error{"CANCELLED", "analysis report import"}
	}
	name := reportName(report.ReportID)
	if existing, readErr := s.root.Open(name); readErr == nil {
		existingRaw, existingErr := io.ReadAll(io.LimitReader(existing, MaxReportBytes+1))
		closeErr := existing.Close()
		if existingErr != nil || closeErr != nil || !bytes.Equal(existingRaw, raw) {
			return Report{}, false, &Error{"REPORT_CONFLICT", "analysis report identity has different bytes"}
		}
		return report, true, nil
	}
	summaries, total, err := s.inventory(ctx, false)
	if err != nil {
		return Report{}, false, err
	}
	if len(summaries) >= MaxReports || total+int64(len(raw)) > maxReportStorage {
		return Report{}, false, &Error{"LIMIT_EXCEEDED", "analysis report storage"}
	}
	temp := "analysis/v1/reports/." + report.ReportID + "." + model.ID() + ".tmp"
	file, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Report{}, false, &Error{"STORAGE_ERROR", "analysis report import temporary create"}
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
		return Report{}, false, &Error{"STORAGE_ERROR", "analysis report import write"}
	}
	if err = s.root.Link(temp, name); err != nil {
		return Report{}, false, &Error{"STORAGE_ERROR", "analysis report import publish"}
	}
	if err = s.root.Remove(temp); err != nil {
		return Report{}, false, &Error{"STORAGE_ERROR", "analysis report import cleanup"}
	}
	if err = s.syncDirectory(); err != nil {
		return Report{}, false, err
	}
	return report, false, nil
}

// DeleteGoverned removes one report only after the governance service has
// recorded explicit intent and verified that no retained artifact references
// it. It is deliberately not exposed directly over IPC.
func (s *Service) DeleteGoverned(ctx context.Context, id string) error {
	if !model.ValidID(id) {
		return &Error{"INVALID_ARGUMENT", "analysis report identity"}
	}
	if err := s.enter(); err != nil {
		return err
	}
	defer s.leave()
	if ctx.Err() != nil {
		return &Error{"CANCELLED", "analysis report deletion"}
	}
	name := reportName(id)
	info, err := s.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return &Error{"NOT_FOUND", "analysis report"}
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return &Error{"CORRUPT_REPORT", "unsafe analysis report file"}
	}
	if err = s.root.Remove(name); err != nil {
		return &Error{"STORAGE_ERROR", "analysis report deletion"}
	}
	return s.syncDirectory()
}

func (s *Service) inventory(ctx context.Context, collect bool) ([]ReportSummary, int64, error) {
	directory, err := s.root.Open("analysis/v1/reports")
	if err != nil {
		return nil, 0, &Error{"STORAGE_ERROR", "analysis report inventory"}
	}
	defer directory.Close()
	const maxEntries = MaxReports + 1024
	entries, err := directory.ReadDir(maxEntries)
	if err != nil && err != io.EOF {
		return nil, 0, &Error{"STORAGE_ERROR", "analysis report inventory"}
	}
	if len(entries) == maxEntries && err == nil {
		return nil, 0, &Error{"LIMIT_EXCEEDED", "analysis report directory entries"}
	}
	summaries := []ReportSummary{}
	var total int64
	reports := 0
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, 0, &Error{"CANCELLED", "analysis report inventory"}
		}
		name := entry.Name()
		if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !strings.HasSuffix(name, ".json") || !model.ValidID(id) {
			return nil, 0, &Error{"CORRUPT_REPORT", "unexpected analysis report entry"}
		}
		info, infoErr := entry.Info()
		if infoErr != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > MaxReportBytes {
			return nil, 0, &Error{"CORRUPT_REPORT", "unsafe analysis report file"}
		}
		reports++
		if reports > MaxReports {
			return nil, 0, &Error{"LIMIT_EXCEEDED", "analysis report count"}
		}
		total += info.Size()
		if total > maxReportStorage {
			return nil, 0, &Error{"LIMIT_EXCEEDED", "analysis report storage"}
		}
		if collect {
			report, readErr := s.read("analysis/v1/reports/"+name, id)
			if readErr != nil {
				return nil, 0, readErr
			}
			summaries = append(summaries, summarize(report))
		}
	}
	// Capacity-only callers still need an accurate count without reading and
	// exposing every report body.
	if !collect {
		summaries = make([]ReportSummary, reports)
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].CreatedAt == summaries[j].CreatedAt {
			return summaries[i].ReportID > summaries[j].ReportID
		}
		return summaries[i].CreatedAt > summaries[j].CreatedAt
	})
	return summaries, total, nil
}

func (s *Service) read(name, id string) (Report, error) {
	info, err := s.root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return Report{}, &Error{"NOT_FOUND", "analysis report"}
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > MaxReportBytes {
		return Report{}, &Error{"CORRUPT_REPORT", "unsafe analysis report file"}
	}
	file, err := s.root.Open(name)
	if err != nil {
		return Report{}, &Error{"STORAGE_ERROR", "analysis report open"}
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, MaxReportBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(raw) > MaxReportBytes {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis report read"}
	}
	report, err := ValidateCanonical(raw)
	if err != nil || report.ReportID != id {
		return Report{}, &Error{"CORRUPT_REPORT", "analysis report validation"}
	}
	return report, nil
}

func summarize(report Report) ReportSummary {
	return ReportSummary{ReportID: report.ReportID, CreatedAt: report.CreatedAt, PolicyName: report.Policy.Name, Decision: report.Evaluation.Decision, Passed: report.Evaluation.Passed, Observations: len(report.Experiment.Observations)}
}

func reportName(id string) string { return "analysis/v1/reports/" + id + ".json" }
func (s *Service) syncDirectory() error {
	directory, err := s.root.Open("analysis/v1/reports")
	if err != nil {
		return &Error{"STORAGE_ERROR", "analysis report directory sync"}
	}
	err = directory.Sync()
	directory.Close()
	if err != nil {
		return &Error{"STORAGE_ERROR", "analysis report directory sync"}
	}
	return nil
}
func (s *Service) enter() error {
	select {
	case s.gate <- struct{}{}:
		return nil
	default:
		return &Error{"BUSY", "analysis report writer"}
	}
}
func (s *Service) leave() { <-s.gate }

var _ Observer = Builder{}
