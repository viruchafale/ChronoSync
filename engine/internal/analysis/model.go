// Package analysis compares bounded recorded evidence. It never executes an
// event, command, provider request, or workspace.
package analysis

import "context"

const (
	FormatVersion   = 1
	MaxObservations = 20
	MaxEvents       = 10_000
	MaxReports      = 1_000
	// Leave headroom for the protocol-1 response envelope under IPC's 4 MiB frame.
	MaxReportBytes = 3 * 1024 * 1024
)

type Status string

const (
	Exact        Status = "exact_match"
	Partial      Status = "partial_match"
	Divergent    Status = "divergent"
	Insufficient Status = "insufficient_evidence"
	Unsupported  Status = "unsupported_comparison"
)

type Dimension string

const (
	EventType Dimension = "event_type"
	Ordering  Dimension = "ordering"
	Input     Dimension = "input_fingerprint"
	Output    Dimension = "output_fingerprint"
	Tool      Dimension = "tool_fingerprint"
	Request   Dimension = "request_fingerprint"
	Response  Dimension = "response_fingerprint"
	Workspace Dimension = "workspace_fingerprint"
	Snapshot  Dimension = "snapshot_identity"
	Config    Dimension = "configuration"
)

var dimensions = []Dimension{EventType, Ordering, Input, Output, Tool, Request, Response, Workspace, Snapshot, Config}

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

type Gap struct {
	Dimension Dimension `json:"dimension"`
	Sequence  string    `json:"sequence,omitempty"`
	Reason    string    `json:"reason"`
}

type Configuration struct {
	Adapter         string `json:"adapter"`
	AdapterVersion  string `json:"adapter_version"`
	RecorderVersion string `json:"recorder_version"`
	Sandbox         string `json:"sandbox"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
	WorkspaceSHA256 string `json:"workspace_sha256"`
}

type EventEvidence struct {
	Sequence        string `json:"sequence"`
	EventID         string `json:"event_id"`
	Kind            string `json:"kind"`
	InputSHA256     string `json:"input_sha256,omitempty"`
	OutputSHA256    string `json:"output_sha256,omitempty"`
	ToolSHA256      string `json:"tool_sha256,omitempty"`
	RequestSHA256   string `json:"request_sha256,omitempty"`
	ResponseSHA256  string `json:"response_sha256,omitempty"`
	WorkspaceSHA256 string `json:"workspace_sha256,omitempty"`
	EvidenceSHA256  string `json:"evidence_sha256"`
	Missing         []Gap  `json:"missing"`
}

type SnapshotEvidence struct {
	SnapshotID     string `json:"snapshot_id"`
	ManifestSHA256 string `json:"manifest_sha256"`
	EventID        string `json:"event_id,omitempty"`
	State          string `json:"state"`
}

type Observation struct {
	FormatVersion      int                `json:"format_version"`
	ObservationID      string             `json:"observation_id"`
	SourceExecutionID  string             `json:"source_execution_id"`
	SourceEventVersion int                `json:"source_event_version"`
	SourceHighWater    string             `json:"source_high_water"`
	SourceHistory      string             `json:"source_history"`
	Configuration      Configuration      `json:"configuration"`
	Events             []EventEvidence    `json:"events"`
	Snapshots          []SnapshotEvidence `json:"snapshots"`
	Missing            []Gap              `json:"missing"`
	Unsupported        []Gap              `json:"unsupported"`
}

type FindingKind string

const (
	MissingEvent            FindingKind = "missing_event"
	AdditionalEvent         FindingKind = "additional_event"
	ChangedFingerprint      FindingKind = "changed_fingerprint"
	ReorderedEvent          FindingKind = "reordered_event"
	WorkspaceDifference     FindingKind = "workspace_difference"
	UnsupportedEvidence     FindingKind = "unsupported_evidence"
	ConfigurationDifference FindingKind = "configuration_difference"
)

type Finding struct {
	Kind             FindingKind `json:"kind"`
	Dimension        Dimension   `json:"dimension"`
	BaselineSequence string      `json:"baseline_sequence,omitempty"`
	ObservedSequence string      `json:"observed_sequence,omitempty"`
	Expected         string      `json:"expected,omitempty"`
	Actual           string      `json:"actual,omitempty"`
	Explanation      string      `json:"explanation"`
	Causality        string      `json:"causality"`
}

type Alignment struct {
	BaselineSequence string            `json:"baseline_sequence,omitempty"`
	ObservedSequence string            `json:"observed_sequence,omitempty"`
	BaselineKind     string            `json:"baseline_kind,omitempty"`
	ObservedKind     string            `json:"observed_kind,omitempty"`
	State            string            `json:"state"`
	Dimensions       []DimensionResult `json:"dimensions"`
}

type DimensionResult struct {
	Dimension Dimension `json:"dimension"`
	State     string    `json:"state"`
	Expected  string    `json:"expected,omitempty"`
	Actual    string    `json:"actual,omitempty"`
}

type Metrics struct {
	BaselineEvents       int      `json:"baseline_events"`
	ObservedEvents       int      `json:"observed_events"`
	AlignedEvents        int      `json:"aligned_events"`
	MissingEvents        int      `json:"missing_events"`
	AdditionalEvents     int      `json:"additional_events"`
	ReorderedEvents      int      `json:"reordered_events"`
	ChangedDimensions    int      `json:"changed_dimensions"`
	ComparableDimensions int      `json:"comparable_dimensions"`
	EventAgreement       float64  `json:"event_agreement"`
	OrderAgreement       float64  `json:"order_agreement"`
	DimensionAgreement   float64  `json:"dimension_agreement"`
	WorkspaceAgreement   *float64 `json:"workspace_agreement,omitempty"`
}

type Comparison struct {
	FormatVersion       int         `json:"format_version"`
	BaselineObservation string      `json:"baseline_observation"`
	ObservedObservation string      `json:"observed_observation"`
	Status              Status      `json:"status"`
	Summary             string      `json:"summary"`
	Metrics             Metrics     `json:"metrics"`
	FirstDivergence     *Finding    `json:"first_divergence,omitempty"`
	Findings            []Finding   `json:"findings"`
	Alignment           []Alignment `json:"alignment"`
	Limitations         []string    `json:"limitations"`
}

type Aggregate struct {
	Observations           int     `json:"observations"`
	Comparisons            int     `json:"comparisons"`
	Exact                  int     `json:"exact"`
	Partial                int     `json:"partial"`
	Divergent              int     `json:"divergent"`
	Insufficient           int     `json:"insufficient"`
	Unsupported            int     `json:"unsupported"`
	MeanEventAgreement     float64 `json:"mean_event_agreement"`
	MeanDimensionAgreement float64 `json:"mean_dimension_agreement"`
}

type Experiment struct {
	FormatVersion int           `json:"format_version"`
	Observations  []Observation `json:"observations"`
	Comparisons   []Comparison  `json:"comparisons"`
	Aggregate     Aggregate     `json:"aggregate"`
}

type Policy struct {
	FormatVersion            int         `json:"format_version"`
	Name                     string      `json:"name"`
	ExactMinEventAgreement   float64     `json:"exact_min_event_agreement"`
	PartialMinEventAgreement float64     `json:"partial_min_event_agreement"`
	MinDimensionAgreement    float64     `json:"min_dimension_agreement"`
	MaxMissingEvents         int         `json:"max_missing_events"`
	MaxAdditionalEvents      int         `json:"max_additional_events"`
	MaxReorderedEvents       int         `json:"max_reordered_events"`
	MaxChangedDimensions     int         `json:"max_changed_dimensions"`
	RequiredDimensions       []Dimension `json:"required_dimensions"`
	UnsupportedHandling      string      `json:"unsupported_handling"`
}

type RuleResult struct {
	Rule     string `json:"rule"`
	Passed   bool   `json:"passed"`
	Actual   string `json:"actual"`
	Required string `json:"required"`
	Reason   string `json:"reason"`
}

type Evaluation struct {
	FormatVersion int          `json:"format_version"`
	Decision      string       `json:"decision"`
	Passed        bool         `json:"passed"`
	Summary       string       `json:"summary"`
	Rules         []RuleResult `json:"rules"`
}

type Report struct {
	FormatVersion int        `json:"format_version"`
	ReportID      string     `json:"report_id"`
	CreatedAt     string     `json:"created_at"`
	Experiment    Experiment `json:"experiment"`
	Policy        Policy     `json:"policy"`
	Evaluation    Evaluation `json:"evaluation"`
	SHA256        string     `json:"sha256"`
}

type ReportSummary struct {
	ReportID     string `json:"report_id"`
	CreatedAt    string `json:"created_at"`
	PolicyName   string `json:"policy_name"`
	Decision     string `json:"decision"`
	Passed       bool   `json:"passed"`
	Observations int    `json:"observations"`
}

type Observer interface {
	Observe(context.Context, string) (Observation, error)
}
