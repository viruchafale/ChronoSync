export interface EngineVersion {
  protocol_version: 1;
  engine_version: string;
  capabilities: (
    | "version"
    | "stdio-recorder-v1"
    | "event-v2"
    | "timeline-v1"
    | "snapshots-v1"
    | "replay-fixtures-v1"
    | "determinism-analysis-v1"
    | "benchmark-governance-v1"
    | "benchmark-exchange-v1"
    | "key-custody-recovery-v1"
  )[];
}
