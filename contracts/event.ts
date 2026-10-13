/** Proposed recording contract; no event capture is implemented in Phase 0. */
export interface EventEnvelope {
  schema_version: 1;
  event_id: string;
  run_id: string;
  producer_id: string;
  producer_seq: string;
  ingest_seq: string;
  lamport: string;
  observed_at: string;
  kind: string;
  causes: string[];
  payload_ref: string | null;
  redaction_version: number;
  adapter_type: string;
  adapter_version: string;
}
