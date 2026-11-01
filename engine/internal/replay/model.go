// Package replay exports and evaluates bounded, inert regression fixtures.
// It never launches a process or calls a provider.
package replay

import "context"

const (
	FormatVersion    = 1
	MaxFixtureBytes  = 2 * 1024 * 1024
	MaxInteractions  = 10_000
	RedactionProfile = "chronosync-redaction-v1"
	InteractionType  = "command.completed"
)

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Detail }

type Envelope struct {
	Fixture Fixture `json:"fixture"`
	SHA256  string  `json:"sha256"`
}

type Fixture struct {
	FormatVersion      int           `json:"format_version"`
	ReplayID           string        `json:"replay_id"`
	SourceExecutionID  string        `json:"source_execution_id"`
	SourceEventVersion int           `json:"source_event_version"`
	SourceEventCount   int           `json:"source_event_count"`
	SourceLastSequence string        `json:"source_last_sequence"`
	Adapter            string        `json:"adapter"`
	RedactionProfile   string        `json:"redaction_profile"`
	SupportedTypes     []string      `json:"supported_types"`
	Interactions       []Interaction `json:"interactions"`
}

type Interaction struct {
	SourceEventID  string   `json:"source_event_id"`
	SourceSequence string   `json:"source_sequence"`
	Type           string   `json:"type"`
	Input          Input    `json:"input"`
	Expected       Expected `json:"expected"`
}

// Input contains fingerprints only. Historical command text is intentionally
// absent, so an evaluator cannot execute it through this contract.
type Input struct {
	SourceSequence string `json:"source_sequence"`
	CommandSHA256  string `json:"command_sha256"`
	ItemIDSHA256   string `json:"item_id_sha256"`
}

type Expected struct {
	OutputSHA256 string `json:"output_sha256"`
	ExitCode     int    `json:"exit_code"`
}

type Actual struct {
	OutputSHA256 string `json:"output_sha256"`
	ExitCode     int    `json:"exit_code"`
}

type Status string

const (
	Completed   Status = "completed"
	Failed      Status = "failed"
	Cancelled   Status = "cancelled"
	Unsupported Status = "unsupported"
)

type Mismatch struct {
	SourceSequence string `json:"source_sequence"`
	Field          string `json:"field"`
	Expected       string `json:"expected"`
	Actual         string `json:"actual"`
}

type UnsupportedInteraction struct {
	SourceSequence string `json:"source_sequence"`
	Reason         string `json:"reason"`
}

type Result struct {
	ReplayID          string                   `json:"replay_id"`
	SourceExecutionID string                   `json:"source_execution_id"`
	FormatVersion     int                      `json:"format_version"`
	Status            Status                   `json:"status"`
	Passed            bool                     `json:"passed"`
	Evaluated         int                      `json:"evaluated"`
	Mismatches        []Mismatch               `json:"mismatches"`
	Unsupported       []UnsupportedInteraction `json:"unsupported"`
	Summary           string                   `json:"summary"`
}

type Evaluator interface {
	Evaluate(context.Context, Input) (Actual, error)
}

type UnsupportedError struct{ Reason string }

func (e *UnsupportedError) Error() string { return "unsupported interaction: " + e.Reason }
