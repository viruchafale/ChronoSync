package contract

// EventEnvelope reserves the v1 recording wire shape. No recorder exists yet.
// Consumers must validate schema, timestamp, counter bounds and reference integrity.
type EventEnvelope struct {
	SchemaVersion    int      `json:"schema_version"`
	EventID          string   `json:"event_id"`
	RunID            string   `json:"run_id"`
	ProducerID       string   `json:"producer_id"`
	ProducerSeq      string   `json:"producer_seq"`
	IngestSeq        string   `json:"ingest_seq"`
	Lamport          string   `json:"lamport"`
	ObservedAt       string   `json:"observed_at"`
	Kind             string   `json:"kind"`
	Causes           []string `json:"causes"`
	PayloadRef       *string  `json:"payload_ref"`
	RedactionVersion int      `json:"redaction_version"`
	AdapterType      string   `json:"adapter_type"`
	AdapterVersion   string   `json:"adapter_version"`
}
