package governance

import (
	"encoding/json"
	"sort"

	"chronosync.local/engine/internal/analysis"
)

func CompareDatasets(baselineID string, baseline Dataset, candidateID string, candidate Dataset) CompatibilityResult {
	result := CompatibilityResult{FormatVersion: FormatVersion, BaselineID: baselineID, CandidateID: candidateID, MissingCases: []string{}, AddedCases: []string{}, ChangedCases: []string{}, Reasons: []string{}}
	if baselineID == candidateID {
		result.Status = CompatibilityExact
		return result
	}
	if baseline.FormatVersion != FormatVersion || candidate.FormatVersion != FormatVersion {
		result.Status = CompatibilityUnsupported
		result.Reasons = append(result.Reasons, "A dataset format version is unsupported.")
		return result
	}
	if baseline.BenchmarkID != candidate.BenchmarkID {
		result.Status = CompatibilityIncompatible
		result.Reasons = append(result.Reasons, "Benchmark family identities differ.")
		return result
	}
	if candidate.Version <= baseline.Version {
		result.Reasons = append(result.Reasons, "Candidate version is not newer than the baseline version.")
	}
	if !sameDimensions(baseline.RequiredDimensions, candidate.RequiredDimensions) {
		result.Reasons = append(result.Reasons, "Required evidence dimensions changed.")
	}
	baseCases := map[string]Case{}
	candidateCases := map[string]Case{}
	for _, item := range baseline.Cases {
		baseCases[item.CaseID] = item
	}
	for _, item := range candidate.Cases {
		candidateCases[item.CaseID] = item
	}
	for id, item := range baseCases {
		other, ok := candidateCases[id]
		if !ok {
			result.MissingCases = append(result.MissingCases, id)
		} else if caseDefinitionDigest(item) != caseDefinitionDigest(other) {
			result.ChangedCases = append(result.ChangedCases, id)
		}
	}
	for id := range candidateCases {
		if _, ok := baseCases[id]; !ok {
			result.AddedCases = append(result.AddedCases, id)
		}
	}
	sort.Strings(result.MissingCases)
	sort.Strings(result.AddedCases)
	sort.Strings(result.ChangedCases)
	if len(result.MissingCases) > 0 {
		result.Reasons = append(result.Reasons, "Existing benchmark cases are missing.")
	}
	if len(result.ChangedCases) > 0 {
		result.Reasons = append(result.Reasons, "Existing benchmark case definitions changed.")
	}
	if candidate.Version <= baseline.Version || !sameDimensions(baseline.RequiredDimensions, candidate.RequiredDimensions) || len(result.MissingCases) > 0 || len(result.ChangedCases) > 0 {
		result.Status = CompatibilityIncompatible
		return result
	}
	result.Status = CompatibilityExtension
	if len(result.AddedCases) == 0 {
		result.Reasons = append(result.Reasons, "Metadata or provenance changed without changing existing case definitions.")
	} else {
		result.Reasons = append(result.Reasons, "Only new case identities were added under unchanged required dimensions.")
	}
	return result
}

func sameDimensions(a, b []analysis.Dimension) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func caseDefinitionDigest(value Case) string {
	semantic := struct {
		CaseID, Baseline, Observed, Comparison, Status string
		Dimensions                                     []analysis.Dimension
	}{value.CaseID, value.BaselineObservationID, value.ObservedObservationID, value.ComparisonSHA256, string(value.ExpectedStatus), value.SupportedDimensions}
	raw, _ := json.Marshal(semantic)
	return digest(raw)
}
