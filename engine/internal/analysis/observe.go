package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"chronosync.local/engine/internal/model"
	"chronosync.local/engine/internal/snapshot"
	"chronosync.local/engine/internal/storage"
)

type HistorySource interface {
	GetRun(context.Context, string) (model.Run, error)
	QueryEvents(context.Context, storage.EventQuery) (storage.EventPage, error)
	GetEvent(context.Context, string, string) (model.Event, error)
}

type SnapshotSource interface {
	Snapshots(context.Context, snapshot.Query) (snapshot.Page, error)
}

type Builder struct {
	History   HistorySource
	Snapshots SnapshotSource
}

func (b Builder) Observe(ctx context.Context, runID string) (Observation, error) {
	if b.History == nil || !model.ValidID(runID) {
		return Observation{}, &Error{"INVALID_ARGUMENT", "source execution"}
	}
	run, err := b.History.GetRun(ctx, runID)
	if err != nil {
		return Observation{}, &Error{"SOURCE_UNAVAILABLE", "execution metadata"}
	}
	observation := Observation{
		FormatVersion: FormatVersion, SourceExecutionID: run.ID, SourceEventVersion: model.EventVersion,
		SourceHighWater: run.LastSequence, SourceHistory: run.History, Events: []EventEvidence{},
		Snapshots: []SnapshotEvidence{}, Missing: []Gap{}, Unsupported: []Gap{},
		Configuration: Configuration{Adapter: run.Adapter, AdapterVersion: run.AdapterVersion,
			RecorderVersion: run.RecorderVersion, Sandbox: run.Sandbox, TimeoutSeconds: run.TimeoutSeconds,
			WorkspaceSHA256: hashText("workspace", run.Workspace)},
	}
	if run.History != "complete" {
		observation.Missing = append(observation.Missing, Gap{Dimension: Ordering, Reason: "source history is " + run.History})
	}
	last, parseErr := strconv.Atoi(run.LastSequence)
	if parseErr != nil || last < 0 || last > MaxEvents {
		return Observation{}, &Error{"UNSUPPORTED_SOURCE", "event count exceeds analysis boundary"}
	}
	cursor := ""
	want := 1
	for {
		if ctx.Err() != nil {
			return Observation{}, &Error{"CANCELLED", "observation build"}
		}
		page, queryErr := b.History.QueryEvents(ctx, storage.EventQuery{ID: run.ID, Cursor: cursor, Limit: 100})
		if queryErr != nil {
			return Observation{}, &Error{"SOURCE_UNAVAILABLE", "event index"}
		}
		for _, summary := range page.Events {
			if summary.Sequence != strconv.Itoa(want) {
				observation.Missing = append(observation.Missing, Gap{Dimension: Ordering, Sequence: strconv.Itoa(want), Reason: "sequence gap"})
				parsed, _ := strconv.Atoi(summary.Sequence)
				want = parsed
			}
			want++
			event, getErr := b.History.GetEvent(ctx, run.ID, summary.EventID)
			if getErr != nil {
				observation.Missing = append(observation.Missing, Gap{Dimension: EventType, Sequence: summary.Sequence, Reason: "event detail unavailable"})
				continue
			}
			if event.Sequence != summary.Sequence || event.Kind != summary.Kind {
				observation.Missing = append(observation.Missing, Gap{Dimension: EventType, Sequence: summary.Sequence, Reason: "event index/detail mismatch"})
				continue
			}
			observation.Events = append(observation.Events, evidence(event))
		}
		if page.Next == "" {
			break
		}
		cursor = page.Next
	}
	if want-1 != last {
		observation.Missing = append(observation.Missing, Gap{Dimension: Ordering, Reason: "source high-water mismatch"})
	}
	if len(observation.Events) == 0 {
		observation.Missing = append(observation.Missing, Gap{Dimension: EventType, Reason: "no readable events"})
	}
	if b.Snapshots == nil {
		observation.Unsupported = append(observation.Unsupported, Gap{Dimension: Snapshot, Reason: "snapshot repository unavailable"})
	} else {
		after := ""
		for {
			page, snapshotErr := b.Snapshots.Snapshots(ctx, snapshot.Query{RunID: run.ID, After: after, Limit: 100})
			if snapshotErr != nil {
				observation.Missing = append(observation.Missing, Gap{Dimension: Snapshot, Reason: "snapshot inventory unavailable"})
				break
			}
			for _, record := range page.Snapshots {
				item := SnapshotEvidence{SnapshotID: record.ID, EventID: record.EventID, State: record.State}
				if record.State == "committed" && record.Manifest != nil {
					item.ManifestSHA256 = record.Manifest.Hash
				} else {
					observation.Missing = append(observation.Missing, Gap{Dimension: Snapshot, Reason: "snapshot " + record.ID + " is not a readable committed identity"})
				}
				observation.Snapshots = append(observation.Snapshots, item)
			}
			if page.Next == "" {
				break
			}
			after = page.Next
		}
		if len(observation.Snapshots) == 0 {
			observation.Missing = append(observation.Missing, Gap{Dimension: Snapshot, Reason: "no workspace snapshot recorded"})
		}
	}
	observation.Unsupported = append(observation.Unsupported,
		Gap{Dimension: Request, Reason: "event-v2 does not record a controlled provider request"},
		Gap{Dimension: Response, Reason: "event-v2 does not record a controlled provider response"},
	)
	sort.Slice(observation.Snapshots, func(i, j int) bool { return observation.Snapshots[i].SnapshotID < observation.Snapshots[j].SnapshotID })
	copyForID := observation
	copyForID.ObservationID = ""
	raw, _ := json.Marshal(copyForID)
	observation.ObservationID = hashBytes(raw)
	return observation, nil
}

func evidence(event model.Event) EventEvidence {
	result := EventEvidence{Sequence: event.Sequence, EventID: event.EventID, Kind: event.Kind, Missing: []Gap{}}
	payload := event.Payload
	if len(payload.Truncations) > 0 {
		for _, truncation := range payload.Truncations {
			result.Missing = append(result.Missing, Gap{Dimension: dimensionForField(truncation.Field), Sequence: event.Sequence, Reason: "recorded value was truncated"})
		}
	}
	if strings.HasPrefix(event.Kind, "command.") || strings.HasPrefix(event.Kind, "tool.") {
		if payload.ItemID == "" && payload.Command == "" {
			result.Missing = append(result.Missing, Gap{Dimension: Input, Sequence: event.Sequence, Reason: "interaction input identity unavailable"})
		} else {
			result.InputSHA256 = hashJSON("input", struct{ Item, Command string }{payload.ItemID, payload.Command})
			result.ToolSHA256 = hashJSON("tool", struct{ Kind, Item, Command string }{event.Kind, payload.ItemID, payload.Command})
		}
		if strings.HasSuffix(event.Kind, ".completed") {
			if payload.ExitCode == nil && payload.Output == "" {
				result.Missing = append(result.Missing, Gap{Dimension: Output, Sequence: event.Sequence, Reason: "completion output unavailable"})
			} else {
				result.OutputSHA256 = hashJSON("output", struct {
					Output string
					Exit   *int
				}{payload.Output, payload.ExitCode})
			}
		}
	}
	if event.Kind == "workspace.changed" || event.Kind == "file.reported" {
		if len(payload.Changes) == 0 {
			result.Missing = append(result.Missing, Gap{Dimension: Workspace, Sequence: event.Sequence, Reason: "workspace change details unavailable"})
		} else {
			result.WorkspaceSHA256 = hashJSON("workspace-change", payload.Changes)
		}
	}
	stable := struct {
		Kind, Input, Output, Tool, Workspace, Status, Reason, Source, Upstream string
	}{event.Kind, result.InputSHA256, result.OutputSHA256, result.ToolSHA256, result.WorkspaceSHA256, string(payload.Status), payload.Reason, payload.Source, payload.UpstreamType}
	result.EvidenceSHA256 = hashJSON("event-evidence", stable)
	return result
}

func dimensionForField(field string) Dimension {
	field = strings.ToLower(field)
	if strings.Contains(field, "output") || strings.Contains(field, "exit") {
		return Output
	}
	if strings.Contains(field, "change") || strings.Contains(field, "path") {
		return Workspace
	}
	return Input
}

func hashText(domain, value string) string { return hashBytes([]byte(domain + "\x00" + value)) }
func hashJSON(domain string, value any) string {
	raw, _ := json.Marshal(value)
	return hashBytes(append([]byte(domain+"\x00"), raw...))
}
func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func validateObservation(observation Observation) error {
	if observation.FormatVersion != FormatVersion || observation.ObservationID == "" || observation.SourceExecutionID == "" || len(observation.Events) > MaxEvents {
		return &Error{"UNSUPPORTED_OBSERVATION", "version, identity, or event bounds"}
	}
	for _, event := range observation.Events {
		if event.Sequence == "" || event.Kind == "" || event.EvidenceSHA256 == "" {
			return &Error{"INVALID_OBSERVATION", fmt.Sprintf("event evidence at %q", event.Sequence)}
		}
	}
	return nil
}
