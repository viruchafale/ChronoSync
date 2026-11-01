package governance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"time"

	"chronosync.local/engine/internal/analysis"
	"chronosync.local/engine/internal/model"
)

func sealDataset(dataset Dataset) ([]byte, string, error) {
	if err := validateDataset(dataset); err != nil {
		return nil, "", err
	}
	body, _ := json.Marshal(dataset)
	envelope := DatasetEnvelope{Dataset: dataset, SHA256: digest(body)}
	raw, _ := json.Marshal(envelope)
	if len(raw) > MaxDatasetBytes {
		return nil, "", &Error{"LIMIT_EXCEEDED", "benchmark dataset bytes"}
	}
	return raw, digest(raw), nil
}

func decodeDataset(raw []byte) (Dataset, string, error) {
	if len(raw) == 0 || len(raw) > MaxDatasetBytes {
		return Dataset{}, "", &Error{"LIMIT_EXCEEDED", "benchmark dataset bytes"}
	}
	var envelope DatasetEnvelope
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&envelope) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		return Dataset{}, "", &Error{"INVALID_DATASET", "strict JSON envelope"}
	}
	canonical, _ := json.Marshal(envelope)
	if !bytes.Equal(canonical, raw) {
		return Dataset{}, "", &Error{"INVALID_DATASET", "dataset is not canonical JSON"}
	}
	if envelope.Dataset.FormatVersion != FormatVersion {
		return Dataset{}, "", &Error{"UNSUPPORTED_DATASET", "dataset format version"}
	}
	body, _ := json.Marshal(envelope.Dataset)
	if envelope.SHA256 != digest(body) {
		return Dataset{}, "", &Error{"CORRUPT_DATASET", "dataset integrity"}
	}
	if err := validateDataset(envelope.Dataset); err != nil {
		return Dataset{}, "", err
	}
	return envelope.Dataset, digest(raw), nil
}

func validateDataset(dataset Dataset) error {
	if dataset.FormatVersion != FormatVersion || !model.ValidID(dataset.BenchmarkID) || dataset.Version < 1 || len(dataset.Name) < 1 || len(dataset.Name) > 128 || dataset.CompatibilityModel != "case-id-v1" || len(dataset.Cases) < 1 || len(dataset.Cases) > MaxCases || len(dataset.ProvenanceLimitations) < 1 || len(dataset.ProvenanceLimitations) > 32 {
		return &Error{"INVALID_DATASET", "version, identity, name, cases, compatibility, or provenance"}
	}
	created, err := time.Parse(time.RFC3339Nano, dataset.CreatedAt)
	if err != nil || created.Location() != time.UTC {
		return &Error{"INVALID_DATASET", "creation timestamp"}
	}
	if err = validateDimensions(dataset.SupportedDimensions); err != nil {
		return err
	}
	if err = validateDimensions(dataset.RequiredDimensions); err != nil {
		return err
	}
	supported := map[analysis.Dimension]bool{}
	for _, dimension := range dataset.SupportedDimensions {
		supported[dimension] = true
	}
	for _, dimension := range dataset.RequiredDimensions {
		if !supported[dimension] {
			return &Error{"INVALID_DATASET", "required dimension is not supported"}
		}
	}
	last := ""
	for _, item := range dataset.Cases {
		if !validCaseID(item.CaseID) || item.CaseID <= last || !model.ValidID(item.SourceReportID) || item.ComparisonIndex < 0 || item.BaselineObservationID == "" || item.ObservedObservationID == "" || !validDigest(item.ComparisonSHA256) || !validAnalysisStatus(item.ExpectedStatus) {
			return &Error{"INVALID_DATASET", "case identity, order, reference, digest, or status"}
		}
		if err = validateDimensions(item.SupportedDimensions); err != nil {
			return err
		}
		for _, dimension := range item.SupportedDimensions {
			if !supported[dimension] {
				return &Error{"INVALID_DATASET", "case dimension is not dataset-supported"}
			}
		}
		last = item.CaseID
	}
	last = ""
	for _, limitation := range dataset.ProvenanceLimitations {
		if limitation == "" || len(limitation) > 512 || limitation <= last || strings.ContainsRune(limitation, 0) {
			return &Error{"INVALID_DATASET", "provenance limitations"}
		}
		last = limitation
	}
	return nil
}

func validateDimensions(values []analysis.Dimension) error {
	if len(values) == 0 || len(values) > 10 {
		return &Error{"INVALID_DATASET", "evidence dimensions"}
	}
	last := ""
	for _, value := range values {
		if !knownDimension(value) || string(value) <= last {
			return &Error{"INVALID_DATASET", "unknown, duplicate, or unsorted dimension"}
		}
		last = string(value)
	}
	return nil
}

func knownDimension(value analysis.Dimension) bool {
	switch value {
	case analysis.EventType, analysis.Ordering, analysis.Input, analysis.Output, analysis.Tool, analysis.Request, analysis.Response, analysis.Workspace, analysis.Snapshot, analysis.Config:
		return true
	default:
		return false
	}
}

func validAnalysisStatus(value analysis.Status) bool {
	return value == analysis.Exact || value == analysis.Partial || value == analysis.Divergent || value == analysis.Insufficient || value == analysis.Unsupported
}

func validCaseID(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= '0' && char <= '9' || char == '-' || char == '_' || char == '.') {
			return false
		}
	}
	return true
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

func digest(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func comparisonDigest(value analysis.Comparison) string {
	raw, _ := json.Marshal(value)
	return digest(raw)
}

func normalizeDimensions(values []analysis.Dimension) []analysis.Dimension {
	result := append([]analysis.Dimension(nil), values...)
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func normalizeStrings(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	return result
}
