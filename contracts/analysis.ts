export type AnalysisStatus =
  | "exact_match"
  | "partial_match"
  | "divergent"
  | "insufficient_evidence"
  | "unsupported_comparison";

export type EvidenceDimension =
  | "event_type"
  | "ordering"
  | "input_fingerprint"
  | "output_fingerprint"
  | "tool_fingerprint"
  | "request_fingerprint"
  | "response_fingerprint"
  | "workspace_fingerprint"
  | "snapshot_identity"
  | "configuration";

export interface EvidenceGap {
  dimension: EvidenceDimension;
  sequence?: string;
  reason: string;
}

export interface ObservationEvent {
  sequence: string;
  event_id: string;
  kind: string;
  input_sha256?: string;
  output_sha256?: string;
  tool_sha256?: string;
  request_sha256?: string;
  response_sha256?: string;
  workspace_sha256?: string;
  evidence_sha256: string;
  missing: EvidenceGap[];
}

export interface Observation {
  format_version: 1;
  observation_id: string;
  source_execution_id: string;
  source_event_version: number;
  source_high_water: string;
  source_history: string;
  configuration: {
    adapter: string;
    adapter_version: string;
    recorder_version: string;
    sandbox: string;
    timeout_seconds: number;
    workspace_sha256: string;
  };
  events: ObservationEvent[];
  snapshots: {
    snapshot_id: string;
    manifest_sha256: string;
    event_id?: string;
    state: string;
  }[];
  missing: EvidenceGap[];
  unsupported: EvidenceGap[];
}

export interface Finding {
  kind:
    | "missing_event"
    | "additional_event"
    | "changed_fingerprint"
    | "reordered_event"
    | "workspace_difference"
    | "unsupported_evidence"
    | "configuration_difference";
  dimension: EvidenceDimension;
  baseline_sequence?: string;
  observed_sequence?: string;
  expected?: string;
  actual?: string;
  explanation: string;
  causality: "unknown";
}

export interface Comparison {
  format_version: 1;
  baseline_observation: string;
  observed_observation: string;
  status: AnalysisStatus;
  summary: string;
  metrics: {
    baseline_events: number;
    observed_events: number;
    aligned_events: number;
    missing_events: number;
    additional_events: number;
    reordered_events: number;
    changed_dimensions: number;
    comparable_dimensions: number;
    event_agreement: number;
    order_agreement: number;
    dimension_agreement: number;
    workspace_agreement?: number;
  };
  first_divergence?: Finding;
  findings: Finding[];
  alignment: {
    baseline_sequence?: string;
    observed_sequence?: string;
    baseline_kind?: string;
    observed_kind?: string;
    state: string;
    dimensions: {
      dimension: EvidenceDimension;
      state: string;
      expected?: string;
      actual?: string;
    }[];
  }[];
  limitations: string[];
}

export interface AnalysisPolicy {
  format_version: 1;
  name: string;
  exact_min_event_agreement: number;
  partial_min_event_agreement: number;
  min_dimension_agreement: number;
  max_missing_events: number;
  max_additional_events: number;
  max_reordered_events: number;
  max_changed_dimensions: number;
  required_dimensions: EvidenceDimension[];
  unsupported_handling: "fail" | "incomparable";
}

export interface AnalysisReport {
  format_version: 1;
  report_id: string;
  created_at: string;
  experiment: {
    format_version: 1;
    observations: Observation[];
    comparisons: Comparison[];
    aggregate: {
      observations: number;
      comparisons: number;
      exact: number;
      partial: number;
      divergent: number;
      insufficient: number;
      unsupported: number;
      mean_event_agreement: number;
      mean_dimension_agreement: number;
    };
  };
  policy: AnalysisPolicy;
  evaluation: {
    format_version: 1;
    decision: "pass" | "fail" | "incomparable";
    passed: boolean;
    summary: string;
    rules: {
      rule: string;
      passed: boolean;
      actual: string;
      required: string;
      reason: string;
    }[];
  };
  sha256: string;
}

export interface AnalysisReportSummary {
  report_id: string;
  created_at: string;
  policy_name: string;
  decision: string;
  passed: boolean;
  observations: number;
}

export interface AnalysisAPI {
  compare(runIDs: string[], policy: AnalysisPolicy): Promise<AnalysisReport>;
  reports(): Promise<AnalysisReportSummary[]>;
  report(id: string): Promise<AnalysisReport>;
  reevaluate(id: string, policy: AnalysisPolicy): Promise<AnalysisReport>;
}

export const defaultAnalysisPolicy: AnalysisPolicy = {
  format_version: 1,
  name: "strict-observable-v1",
  exact_min_event_agreement: 1,
  partial_min_event_agreement: 0.8,
  min_dimension_agreement: 1,
  max_missing_events: 0,
  max_additional_events: 0,
  max_reordered_events: 0,
  max_changed_dimensions: 0,
  required_dimensions: ["event_type", "ordering"],
  unsupported_handling: "incomparable",
};
