package fixtures

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/replay"
	"chronosync.local/engine/internal/replay/transport"
)

const (
	MaxFixtures     = 1000
	MaxFixtureBytes = replay.MaxFixtureBytes
	MaxTotalBytes   = 256 * 1024 * 1024
)

type Kind string

const (
	RegressionV1 Kind = "regression-v1"
	TransportV1  Kind = "synthetic-transport-v1"
	Damaged      Kind = "damaged"
)

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

type Record struct {
	ID                string              `json:"id"`
	Kind              Kind                `json:"kind"`
	Bytes             int64               `json:"bytes"`
	FormatVersion     int                 `json:"format_version"`
	SourceExecutionID string              `json:"source_execution_id,omitempty"`
	Interactions      int                 `json:"interactions"`
	Supported         []string            `json:"supported"`
	Integrity         string              `json:"integrity"`
	Warnings          []string            `json:"warnings"`
	Duplicate         bool                `json:"duplicate,omitempty"`
	Details           []InteractionDetail `json:"details"`
}

type InteractionDetail struct {
	Sequence                  string                       `json:"sequence"`
	Type                      string                       `json:"type"`
	CommandFingerprint        string                       `json:"command_fingerprint,omitempty"`
	ItemFingerprint           string                       `json:"item_fingerprint,omitempty"`
	ExpectedOutputFingerprint string                       `json:"expected_output_fingerprint,omitempty"`
	ExpectedExitCode          *int                         `json:"expected_exit_code,omitempty"`
	Transport                 *transport.TransportIdentity `json:"transport,omitempty"`
	Occurrence                uint32                       `json:"occurrence,omitempty"`
	RequestFingerprint        string                       `json:"request_fingerprint,omitempty"`
	ResponseReference         string                       `json:"response_reference,omitempty"`
	ResponseFingerprint       string                       `json:"response_fingerprint,omitempty"`
	ResponseStatus            int                          `json:"response_status,omitempty"`
}

type Service struct {
	root   *os.Root
	source replay.HistorySource
	gate   chan struct{}
}

func New(base string, source replay.HistorySource) (*Service, error) {
	if !filepath.IsAbs(base) {
		return nil, &Error{"INVALID_ARGUMENT", "fixture root"}
	}
	root, err := os.OpenRoot(base)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "fixture root"}
	}
	for _, directory := range []string{"fixtures", "fixtures/v1", "fixtures/v1/sha256"} {
		if err = root.Mkdir(directory, 0700); err != nil && !errors.Is(err, os.ErrExist) {
			root.Close()
			return nil, &Error{"STORAGE_ERROR", "fixture directory"}
		}
		info, inspectErr := root.Lstat(directory)
		if inspectErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			root.Close()
			return nil, &Error{"UNSAFE_STORAGE", "fixture directory"}
		}
	}
	directory, err := root.Open("fixtures/v1/sha256")
	if err != nil {
		root.Close()
		return nil, &Error{"STORAGE_ERROR", "fixture directory sync"}
	}
	err = directory.Sync()
	directory.Close()
	if err != nil {
		root.Close()
		return nil, &Error{"STORAGE_ERROR", "fixture directory sync"}
	}
	return &Service{root: root, source: source, gate: make(chan struct{}, 1)}, nil
}

func (s *Service) Close() error { return s.root.Close() }

func (s *Service) Eligibility(ctx context.Context, runID string) (Record, error) {
	if s.source == nil {
		return Record{}, &Error{"UNAVAILABLE", "history source"}
	}
	raw, err := (replay.Exporter{Source: s.source}).Export(ctx, runID)
	if err != nil {
		return Record{}, err
	}
	return inspectBytes(raw)
}

func (s *Service) CreateFromRun(ctx context.Context, runID string) (Record, error) {
	if s.source == nil {
		return Record{}, &Error{"UNAVAILABLE", "history source"}
	}
	raw, err := (replay.Exporter{Source: s.source}).Export(ctx, runID)
	if err != nil {
		return Record{}, err
	}
	return s.Import(ctx, raw)
}

func (s *Service) CreateFromEvaluation(ctx context.Context, raw []byte, result replay.Result) (Record, error) {
	eligible, err := replay.RegressionFixture(raw, result)
	if err != nil {
		return Record{}, err
	}
	return s.Import(ctx, eligible)
}

func (s *Service) Import(ctx context.Context, raw []byte) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, &Error{"CANCELLED", "fixture import"}
	}
	record, err := inspectBytes(raw)
	if err != nil {
		return Record{}, err
	}
	if len(raw) > MaxFixtureBytes {
		return Record{}, &Error{"LIMIT_EXCEEDED", "fixture bytes"}
	}
	if err = s.enter(); err != nil {
		return Record{}, err
	}
	defer s.leave()
	records, total, err := s.inventory(ctx)
	if err != nil {
		return Record{}, err
	}
	name := fixtureName(record.ID)
	if existing, readErr := s.read(name, record.ID); readErr == nil {
		if !bytes.Equal(existing, raw) {
			return Record{}, &Error{"CORRUPT_FIXTURE", "existing hash differs"}
		}
		record.Duplicate = true
		return record, nil
	} else if !errors.Is(readErr, os.ErrNotExist) {
		var typed *Error
		if errors.As(readErr, &typed) {
			return Record{}, readErr
		}
	}
	if len(records) >= MaxFixtures || total+int64(len(raw)) > MaxTotalBytes {
		return Record{}, &Error{"LIMIT_EXCEEDED", "fixture storage"}
	}
	temp := "fixtures/v1/sha256/." + record.ID + "." + model.ID() + ".tmp"
	file, err := s.root.OpenFile(temp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return Record{}, &Error{"STORAGE_ERROR", "fixture temporary file"}
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
		return Record{}, &Error{"STORAGE_ERROR", "fixture write"}
	}
	if err = s.root.Link(temp, name); err != nil {
		if existing, readErr := s.read(name, record.ID); readErr == nil && bytes.Equal(existing, raw) {
			record.Duplicate = true
			return record, nil
		}
		return Record{}, &Error{"STORAGE_ERROR", "fixture publish"}
	}
	if err = s.root.Remove(temp); err != nil {
		return Record{}, &Error{"STORAGE_ERROR", "fixture temporary cleanup"}
	}
	if err = s.syncDirectory(); err != nil {
		return Record{}, err
	}
	return record, nil
}

func (s *Service) List(ctx context.Context) ([]Record, error) {
	records, _, err := s.inventory(ctx)
	return records, err
}

func (s *Service) Inspect(ctx context.Context, id string) (Record, error) {
	raw, err := s.load(ctx, id)
	if err != nil {
		return Record{}, err
	}
	record, err := inspectBytes(raw)
	if err != nil || record.ID != id {
		return Record{}, &Error{"CORRUPT_FIXTURE", "fixture validation"}
	}
	return record, nil
}

func (s *Service) Export(ctx context.Context, id string, consent bool) ([]byte, Record, error) {
	if !consent {
		return nil, Record{}, &Error{"CONSENT_REQUIRED", "fixture export"}
	}
	raw, err := s.load(ctx, id)
	if err != nil {
		return nil, Record{}, err
	}
	record, err := inspectBytes(raw)
	if err != nil || record.ID != id {
		return nil, Record{}, &Error{"CORRUPT_FIXTURE", "fixture validation"}
	}
	return append([]byte(nil), raw...), record, nil
}

// ReadForEvaluation is an internal application-data read. It never selects an
// external destination and still performs the same integrity validation.
func (s *Service) ReadForEvaluation(ctx context.Context, id string) ([]byte, Record, error) {
	raw, err := s.load(ctx, id)
	if err != nil {
		return nil, Record{}, err
	}
	record, err := inspectBytes(raw)
	if err != nil || record.ID != id {
		return nil, Record{}, &Error{"CORRUPT_FIXTURE", "fixture validation"}
	}
	return append([]byte(nil), raw...), record, nil
}

func (s *Service) Delete(ctx context.Context, id string, consent bool) error {
	if !consent {
		return &Error{"CONSENT_REQUIRED", "fixture deletion"}
	}
	if !validID(id) {
		return &Error{"INVALID_ARGUMENT", "fixture identity"}
	}
	if err := s.enter(); err != nil {
		return err
	}
	defer s.leave()
	if err := ctx.Err(); err != nil {
		return &Error{"CANCELLED", "fixture deletion"}
	}
	info, err := s.root.Lstat(fixtureName(id))
	if errors.Is(err, os.ErrNotExist) {
		return &Error{"NOT_FOUND", "fixture"}
	}
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return &Error{"CORRUPT_FIXTURE", "unsafe fixture file"}
	}
	if err = s.root.Remove(fixtureName(id)); err != nil {
		return &Error{"STORAGE_ERROR", "fixture deletion"}
	}
	return s.syncDirectory()
}

func (s *Service) load(ctx context.Context, id string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, &Error{"CANCELLED", "fixture load"}
	}
	if !validID(id) {
		return nil, &Error{"INVALID_ARGUMENT", "fixture identity"}
	}
	raw, err := s.read(fixtureName(id), id)
	if errors.Is(err, os.ErrNotExist) {
		return nil, &Error{"NOT_FOUND", "fixture"}
	}
	return raw, err
}

func (s *Service) read(name, id string) ([]byte, error) {
	info, err := s.root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() < 1 || info.Size() > MaxFixtureBytes {
		return nil, &Error{"CORRUPT_FIXTURE", "unsafe fixture file"}
	}
	file, err := s.root.Open(name)
	if err != nil {
		return nil, &Error{"STORAGE_ERROR", "fixture open"}
	}
	raw, readErr := io.ReadAll(io.LimitReader(file, MaxFixtureBytes+1))
	closeErr := file.Close()
	if readErr != nil || closeErr != nil || len(raw) > MaxFixtureBytes || digest(raw) != id {
		return nil, &Error{"CORRUPT_FIXTURE", "fixture integrity"}
	}
	return raw, nil
}

func (s *Service) inventory(ctx context.Context) ([]Record, int64, error) {
	directory, err := s.root.Open("fixtures/v1/sha256")
	if err != nil {
		return nil, 0, &Error{"STORAGE_ERROR", "fixture inventory"}
	}
	defer directory.Close()
	entries, err := directory.ReadDir(MaxFixtures + 1024)
	if err != nil && err != io.EOF {
		return nil, 0, &Error{"STORAGE_ERROR", "fixture inventory"}
	}
	records := make([]Record, 0)
	var total int64
	for _, entry := range entries {
		if ctx.Err() != nil {
			return nil, 0, &Error{"CANCELLED", "fixture inventory"}
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".tmp") {
			continue
		}
		if !strings.HasSuffix(name, ".fixture") || !validID(strings.TrimSuffix(name, ".fixture")) {
			return nil, 0, &Error{"CORRUPT_FIXTURE", "unexpected fixture entry"}
		}
		id := strings.TrimSuffix(name, ".fixture")
		raw, readErr := s.read("fixtures/v1/sha256/"+name, id)
		if readErr != nil {
			info, infoErr := entry.Info()
			if infoErr != nil {
				return nil, 0, &Error{"CORRUPT_FIXTURE", "fixture inventory metadata"}
			}
			records = append(records, Record{ID: id, Kind: Damaged, Bytes: info.Size(), Integrity: "damaged", Supported: []string{}, Warnings: []string{"Stored fixture failed integrity validation; it was not repaired."}, Details: []InteractionDetail{}})
			total += info.Size()
			continue
		}
		record, inspectErr := inspectBytes(raw)
		if inspectErr != nil || record.ID != id {
			return nil, 0, &Error{"CORRUPT_FIXTURE", "fixture validation"}
		}
		records = append(records, record)
		total += record.Bytes
		if len(records) > MaxFixtures || total > MaxTotalBytes {
			return nil, 0, &Error{"LIMIT_EXCEEDED", "fixture inventory"}
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	return records, total, nil
}

func inspectBytes(raw []byte) (Record, error) {
	if len(raw) == 0 || len(raw) > MaxFixtureBytes {
		return Record{}, &Error{"LIMIT_EXCEEDED", "fixture bytes"}
	}
	id := digest(raw)
	if fixture, err := replay.Load(raw); err == nil {
		record := Record{
			ID: id, Kind: RegressionV1, Bytes: int64(len(raw)), FormatVersion: fixture.FormatVersion,
			SourceExecutionID: fixture.SourceExecutionID, Interactions: len(fixture.Interactions),
			Supported: append([]string(nil), fixture.SupportedTypes...), Integrity: "verified",
			Warnings: []string{"Contains equality-revealing unkeyed fingerprints; review before sharing.", "Does not prove origin or authenticity."},
			Details:  make([]InteractionDetail, 0, len(fixture.Interactions)),
		}
		for _, interaction := range fixture.Interactions {
			exitCode := interaction.Expected.ExitCode
			record.Details = append(record.Details, InteractionDetail{
				Sequence: interaction.SourceSequence, Type: interaction.Type,
				CommandFingerprint: interaction.Input.CommandSHA256, ItemFingerprint: interaction.Input.ItemIDSHA256,
				ExpectedOutputFingerprint: interaction.Expected.OutputSHA256, ExpectedExitCode: &exitCode,
			})
		}
		return record, nil
	}
	if inspection, err := transport.Inspect(raw); err == nil {
		record := Record{
			ID: id, Kind: TransportV1, Bytes: int64(len(raw)), FormatVersion: inspection.FormatVersion,
			Interactions: inspection.Interactions, Supported: append([]string(nil), inspection.Supported...), Integrity: "verified",
			Warnings: []string{"Contains equality-revealing unkeyed fingerprints and sanitized response metadata; review before sharing.", "Does not prove origin or authenticity."},
			Details:  make([]InteractionDetail, 0, len(inspection.Entries)),
		}
		for _, entry := range inspection.Entries {
			identity := entry.Transport
			record.Details = append(record.Details, InteractionDetail{
				Sequence: strconv.FormatUint(entry.Sequence, 10), Type: "synthetic.request.v1", Transport: &identity,
				Occurrence: entry.Occurrence, RequestFingerprint: entry.RequestFingerprint,
				ResponseReference: entry.ResponseReference, ResponseFingerprint: entry.ResponseFingerprint, ResponseStatus: entry.ResponseStatus,
			})
		}
		return record, nil
	}
	return Record{}, &Error{"UNSUPPORTED_FIXTURE", "fixture format or integrity"}
}

func (s *Service) enter() error {
	select {
	case s.gate <- struct{}{}:
		return nil
	default:
		return &Error{"BUSY", "fixture operation"}
	}
}

func (s *Service) leave() { <-s.gate }

func (s *Service) syncDirectory() error {
	directory, err := s.root.Open("fixtures/v1/sha256")
	if err != nil {
		return &Error{"STORAGE_ERROR", "fixture directory sync"}
	}
	defer directory.Close()
	if err = directory.Sync(); err != nil {
		return &Error{"STORAGE_ERROR", "fixture directory sync"}
	}
	return nil
}

func fixtureName(id string) string { return "fixtures/v1/sha256/" + id + ".fixture" }

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func validID(id string) bool {
	decoded, err := hex.DecodeString(id)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == id
}
