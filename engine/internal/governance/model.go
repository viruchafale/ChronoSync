// Package governance manages immutable local benchmark and retention evidence.
// It never launches commands, agents, providers, transports, or evaluators.
package governance

import "chronosync.local/engine/internal/analysis"

const (
	FormatVersion      = 1
	MaxDatasetBytes    = 2 * 1024 * 1024
	MaxDatasets        = 1000
	MaxCases           = 1000
	MaxLineages        = 1000
	MaxRetentionEvents = 10000
	MaxGovernanceBytes = 256 * 1024 * 1024
)

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

type DatasetEnvelope struct {
	Dataset Dataset `json:"dataset"`
	SHA256  string  `json:"sha256"`
}

type Dataset struct {
	FormatVersion         int                  `json:"format_version"`
	BenchmarkID           string               `json:"benchmark_id"`
	Version               uint64               `json:"version"`
	Name                  string               `json:"name"`
	CreatedAt             string               `json:"created_at"`
	Cases                 []Case               `json:"cases"`
	SupportedDimensions   []analysis.Dimension `json:"supported_dimensions"`
	RequiredDimensions    []analysis.Dimension `json:"required_dimensions"`
	CompatibilityModel    string               `json:"compatibility_model"`
	ProvenanceLimitations []string             `json:"provenance_limitations"`
}

type Case struct {
	CaseID                string               `json:"case_id"`
	SourceReportID        string               `json:"source_report_id"`
	ComparisonIndex       int                  `json:"comparison_index"`
	BaselineObservationID string               `json:"baseline_observation_id"`
	ObservedObservationID string               `json:"observed_observation_id"`
	ComparisonSHA256      string               `json:"comparison_sha256"`
	ExpectedStatus        analysis.Status      `json:"expected_status"`
	SupportedDimensions   []analysis.Dimension `json:"supported_dimensions"`
}

type CaseDraft struct {
	CaseID              string               `json:"case_id"`
	SourceReportID      string               `json:"source_report_id"`
	ComparisonIndex     int                  `json:"comparison_index"`
	SupportedDimensions []analysis.Dimension `json:"supported_dimensions"`
}

type CreateDataset struct {
	BenchmarkID           string               `json:"benchmark_id,omitempty"`
	Version               uint64               `json:"version"`
	Name                  string               `json:"name"`
	Cases                 []CaseDraft          `json:"cases"`
	SupportedDimensions   []analysis.Dimension `json:"supported_dimensions"`
	RequiredDimensions    []analysis.Dimension `json:"required_dimensions"`
	ProvenanceLimitations []string             `json:"provenance_limitations"`
}

type DatasetRecord struct {
	ContentID   string   `json:"content_id"`
	BenchmarkID string   `json:"benchmark_id,omitempty"`
	Version     uint64   `json:"version,omitempty"`
	Name        string   `json:"name,omitempty"`
	Cases       int      `json:"cases"`
	Bytes       int64    `json:"bytes"`
	Integrity   string   `json:"integrity"`
	Duplicate   bool     `json:"duplicate,omitempty"`
	Lifecycle   State    `json:"lifecycle"`
	Warnings    []string `json:"warnings"`
}

type CompatibilityStatus string

const (
	CompatibilityExact        CompatibilityStatus = "exact"
	CompatibilityExtension    CompatibilityStatus = "compatible_extension"
	CompatibilityIncompatible CompatibilityStatus = "incompatible"
	CompatibilityUnsupported  CompatibilityStatus = "unsupported"
)

type CompatibilityResult struct {
	FormatVersion int                 `json:"format_version"`
	BaselineID    string              `json:"baseline_content_id"`
	CandidateID   string              `json:"candidate_content_id"`
	Status        CompatibilityStatus `json:"status"`
	MissingCases  []string            `json:"missing_cases"`
	AddedCases    []string            `json:"added_cases"`
	ChangedCases  []string            `json:"changed_cases"`
	Reasons       []string            `json:"reasons"`
}

type Lineage struct {
	FormatVersion       int             `json:"format_version"`
	LineageID           string          `json:"lineage_id"`
	CreatedAt           string          `json:"created_at"`
	DatasetContentID    string          `json:"dataset_content_id"`
	BenchmarkID         string          `json:"benchmark_id"`
	DatasetVersion      uint64          `json:"dataset_version"`
	SourceReportIDs     []string        `json:"source_report_ids"`
	EvaluationReportIDs []string        `json:"evaluation_report_ids"`
	Policy              analysis.Policy `json:"policy"`
	ParentLineageID     string          `json:"parent_lineage_id,omitempty"`
	SHA256              string          `json:"sha256"`
}

type LineageSummary struct {
	LineageID        string `json:"lineage_id"`
	CreatedAt        string `json:"created_at"`
	DatasetContentID string `json:"dataset_content_id"`
	BenchmarkID      string `json:"benchmark_id"`
	DatasetVersion   uint64 `json:"dataset_version"`
	PolicyName       string `json:"policy_name"`
	Integrity        string `json:"integrity"`
}

type ObjectKind string

const (
	ReportObject  ObjectKind = "report"
	DatasetObject ObjectKind = "dataset"
	LineageObject ObjectKind = "lineage"
)

type State string

const (
	Unmanaged       State = "unmanaged"
	Active          State = "active"
	Archived        State = "archived"
	Expired         State = "expired"
	Damaged         State = "damaged"
	DeletionPending State = "deletion_pending"
	Deleted         State = "deleted"
)

type RetentionPolicy struct {
	FormatVersion  int    `json:"format_version"`
	Name           string `json:"name"`
	ExpiresAt      string `json:"expires_at,omitempty"`
	ArchiveAllowed bool   `json:"archive_allowed"`
	DeleteAllowed  bool   `json:"delete_allowed"`
	Automatic      bool   `json:"automatic"`
}

type RetentionEvent struct {
	FormatVersion int             `json:"format_version"`
	EventID       string          `json:"event_id"`
	ObjectKind    ObjectKind      `json:"object_kind"`
	ObjectID      string          `json:"object_id"`
	Revision      uint64          `json:"revision"`
	PreviousState State           `json:"previous_state"`
	State         State           `json:"state"`
	Policy        RetentionPolicy `json:"policy"`
	Reason        string          `json:"reason"`
	CreatedAt     string          `json:"created_at"`
	SHA256        string          `json:"sha256"`
}

type RetentionStatus struct {
	ObjectKind ObjectKind      `json:"object_kind"`
	ObjectID   string          `json:"object_id"`
	Revision   uint64          `json:"revision"`
	State      State           `json:"state"`
	Policy     RetentionPolicy `json:"policy"`
	Reason     string          `json:"reason,omitempty"`
	UpdatedAt  string          `json:"updated_at,omitempty"`
}

type Reference struct {
	FromKind ObjectKind `json:"from_kind"`
	FromID   string     `json:"from_id"`
	ToKind   ObjectKind `json:"to_kind"`
	ToID     string     `json:"to_id"`
	Relation string     `json:"relation"`
}

type AuditIssue struct {
	Code       string     `json:"code"`
	ObjectKind ObjectKind `json:"object_kind"`
	ObjectID   string     `json:"object_id"`
	Detail     string     `json:"detail"`
}

type StorageUsage struct {
	Reports           int   `json:"reports"`
	ReportBytes       int64 `json:"report_bytes"`
	Datasets          int   `json:"datasets"`
	DatasetBytes      int64 `json:"dataset_bytes"`
	Lineages          int   `json:"lineages"`
	LineageBytes      int64 `json:"lineage_bytes"`
	RetentionEvents   int   `json:"retention_events"`
	RetentionBytes    int64 `json:"retention_bytes"`
	TemporaryFiles    int   `json:"temporary_files"`
	TemporaryBytes    int64 `json:"temporary_bytes"`
	DamagedObjects    int   `json:"damaged_objects"`
	MissingReferences int   `json:"missing_references"`
	UnmanagedObjects  int   `json:"unmanaged_objects"`
	OrphanedMetadata  int   `json:"orphaned_metadata"`
	TotalBytes        int64 `json:"total_bytes"`
}

type Audit struct {
	FormatVersion int               `json:"format_version"`
	Usage         StorageUsage      `json:"usage"`
	Issues        []AuditIssue      `json:"issues"`
	Retention     []RetentionStatus `json:"retention"`
	References    []Reference       `json:"references"`
}
