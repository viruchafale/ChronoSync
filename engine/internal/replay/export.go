package replay

import (
	"context"
	"reflect"
	"strconv"
	"strings"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/redact"
	"chronosync.local/engine/internal/storage"
)

type HistorySource interface {
	GetRun(context.Context, string) (model.Run, error)
	QueryEvents(context.Context, storage.EventQuery) (storage.EventPage, error)
	GetEvent(context.Context, string, string) (model.Event, error)
}

type Exporter struct{ Source HistorySource }

var allowedBoundaryKinds = map[string]bool{
	"execution.state":      true,
	"execution.warning":    true,
	"agent.thread":         true,
	"agent.turn_started":   true,
	"agent.turn_completed": true,
	"command.started":      true,
	"command.updated":      true,
	InteractionType:        true,
}

func (e Exporter) Export(ctx context.Context, sourceExecutionID string) ([]byte, error) {
	if e.Source == nil || !model.ValidID(sourceExecutionID) {
		return nil, typed("INVALID_SOURCE", "history source or execution identity")
	}
	if ctx.Err() != nil {
		return nil, typed("CANCELLED", "fixture export")
	}
	run, err := e.Source.GetRun(ctx, sourceExecutionID)
	if err != nil {
		return nil, typed("SOURCE_UNAVAILABLE", "execution metadata")
	}
	if run.State != model.Completed || run.History != "complete" {
		return nil, typed("INCOMPLETE_HISTORY", "execution is not completed with complete history")
	}
	last, err := strconv.Atoi(run.LastSequence)
	if err != nil || last < 1 || last > MaxInteractions*10 {
		return nil, typed("INCOMPLETE_HISTORY", "source event count")
	}
	fixture := Fixture{
		FormatVersion:      FormatVersion,
		SourceExecutionID:  run.ID,
		SourceEventVersion: model.EventVersion,
		SourceEventCount:   last,
		SourceLastSequence: run.LastSequence,
		Adapter:            run.Adapter,
		RedactionProfile:   RedactionProfile,
		SupportedTypes:     []string{InteractionType},
		Interactions:       []Interaction{},
	}
	if fixture.Adapter != "codex" {
		return nil, typed("UNSUPPORTED_SOURCE", "adapter")
	}
	var cursor string
	wantSequence := 1
	for {
		if err = ctx.Err(); err != nil {
			return nil, typed("CANCELLED", "fixture export")
		}
		page, queryErr := e.Source.QueryEvents(ctx, storage.EventQuery{ID: run.ID, Cursor: cursor, Limit: 100})
		if queryErr != nil {
			return nil, typed("SOURCE_UNAVAILABLE", "event index")
		}
		for _, summary := range page.Events {
			if summary.Sequence != strconv.Itoa(wantSequence) {
				return nil, typed("INCOMPLETE_HISTORY", "event sequence gap")
			}
			wantSequence++
			if !allowedBoundaryKinds[summary.Kind] {
				return nil, typed("UNSUPPORTED_EVENT", "source sequence "+summary.Sequence+" kind "+summary.Kind)
			}
			event, getErr := e.Source.GetEvent(ctx, run.ID, summary.EventID)
			if getErr != nil {
				return nil, typed("SOURCE_UNAVAILABLE", "event detail")
			}
			if event.Sequence != summary.Sequence || event.Kind != summary.Kind {
				return nil, typed("INCOMPLETE_HISTORY", "event index mismatch")
			}
			if event.Kind == "execution.warning" {
				clean := redact.Payload(event.Payload)
				if clean.Message != event.Payload.Message || len(event.Payload.Truncations) != 0 || !strings.HasPrefix(event.Payload.Message, "Metadata-only observation complete; excluded/skipped entries before=") {
					return nil, typed("UNSUPPORTED_EVENT", "source sequence "+summary.Sequence+" warning")
				}
			}
			if event.Kind == InteractionType {
				interaction, interactionErr := exportInteraction(event)
				if interactionErr != nil {
					return nil, interactionErr
				}
				fixture.Interactions = append(fixture.Interactions, interaction)
				if len(fixture.Interactions) > MaxInteractions {
					return nil, typed("FIXTURE_TOO_LARGE", "interaction count")
				}
			}
		}
		if page.Next == "" {
			break
		}
		cursor = page.Next
	}
	if wantSequence-1 != last {
		return nil, typed("INCOMPLETE_HISTORY", "source high water mismatch")
	}
	if len(fixture.Interactions) == 0 {
		return nil, typed("UNSUPPORTED_EVENT", "no supported command completion")
	}
	return seal(fixture)
}

func exportInteraction(event model.Event) (Interaction, error) {
	payload := event.Payload
	clean := redact.Payload(payload)
	if !reflect.DeepEqual(clean, payload) {
		return Interaction{}, typed("REDACTED_OR_INCOMPLETE", "payload requires additional redaction")
	}
	if len(payload.Truncations) > 0 || containsMarker(payload.Command) || containsMarker(payload.Output) || containsMarker(payload.ItemID) {
		return Interaction{}, typed("REDACTED_OR_INCOMPLETE", "payload is redacted or truncated")
	}
	if payload.Command == "" || payload.ItemID == "" || payload.ExitCode == nil {
		return Interaction{}, typed("INCOMPLETE_HISTORY", "command completion fields")
	}
	return Interaction{
		SourceEventID:  event.EventID,
		SourceSequence: event.Sequence,
		Type:           InteractionType,
		Input: Input{
			SourceSequence: event.Sequence,
			CommandSHA256:  digest([]byte(payload.Command)),
			ItemIDSHA256:   digest([]byte(payload.ItemID)),
		},
		Expected: Expected{OutputSHA256: digest([]byte(payload.Output)), ExitCode: *payload.ExitCode},
	}, nil
}

func containsMarker(value string) bool {
	return strings.Contains(value, "[REDACTED]") || strings.Contains(value, "[TRUNCATED]")
}
