package analysis

import (
	"fmt"
	"math"
	"sort"
	"strconv"
)

func Compare(baseline, observed Observation) (Comparison, error) {
	result := Comparison{
		FormatVersion: FormatVersion, BaselineObservation: baseline.ObservationID, ObservedObservation: observed.ObservationID,
		Findings: []Finding{}, Alignment: []Alignment{}, Limitations: []string{"Per-run sequence is observed order only; this comparison does not establish causation."},
		Metrics: Metrics{BaselineEvents: len(baseline.Events), ObservedEvents: len(observed.Events)},
	}
	if err := validateObservation(baseline); err != nil {
		return result, err
	}
	if err := validateObservation(observed); err != nil {
		return result, err
	}
	if baseline.SourceEventVersion != observed.SourceEventVersion {
		result.Status = Unsupported
		result.Summary = "Source event versions are incompatible."
		result.Findings = append(result.Findings, Finding{Kind: UnsupportedEvidence, Dimension: EventType, Expected: strconv.Itoa(baseline.SourceEventVersion), Actual: strconv.Itoa(observed.SourceEventVersion), Explanation: "Source event versions differ.", Causality: "unknown"})
		result.FirstDivergence = &result.Findings[0]
		return result, nil
	}
	queues := map[string][]int{}
	for index, event := range observed.Events {
		key := alignmentIdentity(event)
		queues[key] = append(queues[key], index)
	}
	used := make([]bool, len(observed.Events))
	lastObserved := -1
	comparable := 0
	equal := 0
	workspaceComparable := 0
	workspaceEqual := 0
	for _, expected := range baseline.Events {
		key := alignmentIdentity(expected)
		indices := queues[key]
		match := -1
		for len(indices) > 0 {
			candidate := indices[0]
			indices = indices[1:]
			if !used[candidate] {
				match = candidate
				break
			}
		}
		queues[key] = indices
		if match < 0 {
			result.Metrics.MissingEvents++
			finding := Finding{Kind: MissingEvent, Dimension: EventType, BaselineSequence: expected.Sequence, Expected: expected.Kind, Explanation: "An aligned observed event was not present.", Causality: "unknown"}
			result.Findings = append(result.Findings, finding)
			result.Alignment = append(result.Alignment, Alignment{BaselineSequence: expected.Sequence, BaselineKind: expected.Kind, State: "missing", Dimensions: []DimensionResult{}})
			continue
		}
		used[match] = true
		actual := observed.Events[match]
		result.Metrics.AlignedEvents++
		alignment := Alignment{BaselineSequence: expected.Sequence, ObservedSequence: actual.Sequence, BaselineKind: expected.Kind, ObservedKind: actual.Kind, State: "matched", Dimensions: []DimensionResult{}}
		if match < lastObserved {
			result.Metrics.ReorderedEvents++
			alignment.State = "reordered"
			result.Findings = append(result.Findings, Finding{Kind: ReorderedEvent, Dimension: Ordering, BaselineSequence: expected.Sequence, ObservedSequence: actual.Sequence, Explanation: "Aligned interaction appears in a different observed order.", Causality: "unknown"})
		}
		if match > lastObserved {
			lastObserved = match
		}
		genericExpected, genericActual := "", ""
		if expected.InputSHA256 == "" && expected.OutputSHA256 == "" && expected.ToolSHA256 == "" && expected.RequestSHA256 == "" && expected.ResponseSHA256 == "" && expected.WorkspaceSHA256 == "" && actual.InputSHA256 == "" && actual.OutputSHA256 == "" && actual.ToolSHA256 == "" && actual.RequestSHA256 == "" && actual.ResponseSHA256 == "" && actual.WorkspaceSHA256 == "" {
			genericExpected, genericActual = expected.EvidenceSHA256, actual.EvidenceSHA256
		}
		pairs := []struct {
			dimension Dimension
			expected  string
			actual    string
		}{
			{EventType, genericExpected, genericActual},
			{Input, expected.InputSHA256, actual.InputSHA256},
			{Output, expected.OutputSHA256, actual.OutputSHA256},
			{Tool, expected.ToolSHA256, actual.ToolSHA256},
			{Request, expected.RequestSHA256, actual.RequestSHA256},
			{Response, expected.ResponseSHA256, actual.ResponseSHA256},
			{Workspace, expected.WorkspaceSHA256, actual.WorkspaceSHA256},
		}
		for _, pair := range pairs {
			if pair.expected == "" && pair.actual == "" {
				continue
			}
			state := "match"
			comparable++
			if pair.dimension == Workspace {
				workspaceComparable++
			}
			if pair.expected == pair.actual && pair.expected != "" {
				equal++
				if pair.dimension == Workspace {
					workspaceEqual++
				}
			} else {
				state = "different"
				alignment.State = "changed"
				result.Metrics.ChangedDimensions++
				kind := ChangedFingerprint
				if pair.dimension == Workspace {
					kind = WorkspaceDifference
				}
				result.Findings = append(result.Findings, Finding{Kind: kind, Dimension: pair.dimension, BaselineSequence: expected.Sequence, ObservedSequence: actual.Sequence, Expected: pair.expected, Actual: pair.actual, Explanation: "Available observable fingerprints differ or one side is unavailable.", Causality: "unknown"})
			}
			alignment.Dimensions = append(alignment.Dimensions, DimensionResult{Dimension: pair.dimension, State: state, Expected: pair.expected, Actual: pair.actual})
		}
		result.Alignment = append(result.Alignment, alignment)
	}
	for index, event := range observed.Events {
		if used[index] {
			continue
		}
		result.Metrics.AdditionalEvents++
		result.Findings = append(result.Findings, Finding{Kind: AdditionalEvent, Dimension: EventType, ObservedSequence: event.Sequence, Actual: event.Kind, Explanation: "An observed event had no aligned baseline event.", Causality: "unknown"})
		result.Alignment = append(result.Alignment, Alignment{ObservedSequence: event.Sequence, ObservedKind: event.Kind, State: "additional", Dimensions: []DimensionResult{}})
	}
	compareConfiguration(&result, baseline.Configuration, observed.Configuration)
	compareSnapshots(&result, baseline.Snapshots, observed.Snapshots)
	result.Metrics.ComparableDimensions = comparable
	result.Metrics.EventAgreement = ratio(result.Metrics.AlignedEvents, max(len(baseline.Events), len(observed.Events)))
	result.Metrics.OrderAgreement = ratio(result.Metrics.AlignedEvents-result.Metrics.ReorderedEvents, result.Metrics.AlignedEvents)
	result.Metrics.DimensionAgreement = ratio(equal, comparable)
	if workspaceComparable > 0 {
		value := ratio(workspaceEqual, workspaceComparable)
		result.Metrics.WorkspaceAgreement = &value
	}
	for _, gap := range append(append([]Gap{}, baseline.Missing...), observed.Missing...) {
		result.Limitations = append(result.Limitations, fmt.Sprintf("Missing %s evidence: %s", gap.Dimension, gap.Reason))
	}
	for _, gap := range append(append([]Gap{}, baseline.Unsupported...), observed.Unsupported...) {
		result.Limitations = append(result.Limitations, fmt.Sprintf("Unsupported %s evidence: %s", gap.Dimension, gap.Reason))
	}
	if len(result.Findings) > 0 {
		result.FirstDivergence = firstFinding(result.Findings)
	}
	result.Status, result.Summary = classify(result, baseline, observed)
	return result, nil
}

func alignmentIdentity(event EventEvidence) string {
	identity := event.ToolSHA256
	if identity == "" {
		identity = event.InputSHA256
	}
	if identity == "" {
		identity = event.RequestSHA256
	}
	if identity == "" {
		identity = event.Kind
	}
	return event.Kind + "\x00" + identity
}

func compareConfiguration(result *Comparison, expected, actual Configuration) {
	values := []struct{ name, expected, actual string }{
		{"adapter", expected.Adapter, actual.Adapter}, {"adapter_version", expected.AdapterVersion, actual.AdapterVersion},
		{"recorder_version", expected.RecorderVersion, actual.RecorderVersion}, {"sandbox", expected.Sandbox, actual.Sandbox},
		{"timeout_seconds", strconv.Itoa(expected.TimeoutSeconds), strconv.Itoa(actual.TimeoutSeconds)}, {"workspace_sha256", expected.WorkspaceSHA256, actual.WorkspaceSHA256},
	}
	for _, value := range values {
		if value.expected != value.actual {
			result.Metrics.ChangedDimensions++
			result.Findings = append(result.Findings, Finding{Kind: ConfigurationDifference, Dimension: Config, Expected: value.expected, Actual: value.actual, Explanation: "Recorded configuration field " + value.name + " differs.", Causality: "unknown"})
		}
	}
}

func compareSnapshots(result *Comparison, baseline, observed []SnapshotEvidence) {
	expected := make([]string, 0, len(baseline))
	actual := make([]string, 0, len(observed))
	for _, item := range baseline {
		if item.ManifestSHA256 != "" {
			expected = append(expected, item.ManifestSHA256)
		}
	}
	for _, item := range observed {
		if item.ManifestSHA256 != "" {
			actual = append(actual, item.ManifestSHA256)
		}
	}
	sort.Strings(expected)
	sort.Strings(actual)
	if fmt.Sprint(expected) != fmt.Sprint(actual) {
		result.Metrics.ChangedDimensions++
		result.Findings = append(result.Findings, Finding{Kind: WorkspaceDifference, Dimension: Snapshot, Expected: fmt.Sprint(expected), Actual: fmt.Sprint(actual), Explanation: "Committed snapshot manifest identities differ.", Causality: "unknown"})
	}
}

func firstFinding(findings []Finding) *Finding {
	copyFindings := append([]Finding(nil), findings...)
	sort.SliceStable(copyFindings, func(i, j int) bool {
		return findingPosition(copyFindings[i]) < findingPosition(copyFindings[j])
	})
	return &copyFindings[0]
}

func findingPosition(f Finding) int {
	value := f.BaselineSequence
	if value == "" {
		value = f.ObservedSequence
	}
	position, err := strconv.Atoi(value)
	if err != nil {
		return math.MaxInt
	}
	return position
}

func classify(result Comparison, baseline, observed Observation) (Status, string) {
	for _, gap := range append(append([]Gap{}, baseline.Unsupported...), observed.Unsupported...) {
		if gap.Dimension == EventType || gap.Dimension == Ordering {
			return Unsupported, "The event or ordering evidence is unsupported."
		}
	}
	for _, gap := range append(append([]Gap{}, baseline.Missing...), observed.Missing...) {
		if gap.Dimension == EventType || gap.Dimension == Ordering {
			return Insufficient, "Event history is incomplete; divergence cannot be evaluated conclusively."
		}
	}
	if len(baseline.Events) == 0 || len(observed.Events) == 0 {
		return Insufficient, "No comparable event evidence is available."
	}
	if len(result.Findings) == 0 {
		return Exact, "All available bounded evidence matches exactly."
	}
	if result.Metrics.EventAgreement >= 0.5 {
		return Partial, "Some bounded evidence agrees, with explicit observed divergences."
	}
	return Divergent, "Available bounded evidence is substantially divergent."
}

func ratio(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
