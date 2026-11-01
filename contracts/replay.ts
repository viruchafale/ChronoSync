export type FixtureKind =
  "regression-v1" | "synthetic-transport-v1" | "damaged";

export interface TransportIdentity {
  name: string;
  protocol: string;
  version: string;
}

export interface FixtureInteractionDetail {
  sequence: string;
  type: string;
  command_fingerprint?: string;
  item_fingerprint?: string;
  expected_output_fingerprint?: string;
  expected_exit_code?: number;
  transport?: TransportIdentity;
  occurrence?: number;
  request_fingerprint?: string;
  response_reference?: string;
  response_fingerprint?: string;
  response_status?: number;
}

export interface FixtureRecord {
  id: string;
  kind: FixtureKind;
  bytes: number;
  format_version: number;
  source_execution_id?: string;
  interactions: number;
  supported: string[];
  integrity: "verified" | "damaged";
  warnings: string[];
  duplicate?: boolean;
  details: FixtureInteractionDetail[];
}

export interface SyntheticRequest {
  transport: TransportIdentity;
  operation: string;
  metadata: { name: string; value: string }[];
  occurrence: number;
}

export interface UnifiedPlan {
  version: 1;
  timeout_millis: number;
  actuals: {
    source_sequence: string;
    output_sha256: string;
    exit_code: number;
  }[];
  transport_requests: SyntheticRequest[];
}

export interface UnifiedResult {
  version: 1;
  status: "completed" | "failed" | "cancelled" | "unsupported";
  passed: boolean;
  summary: string;
  transport: {
    kind:
      | "matched"
      | "unmatched"
      | "ambiguous"
      | "unsupported"
      | "cancelled"
      | "invalid";
    response_reference?: string;
    response_fingerprint?: string;
    response: { status: number; metadata: { name: string; value: string }[] };
    code?: string;
  }[];
  regression: {
    replay_id: string;
    source_execution_id: string;
    format_version: number;
    status: "completed" | "failed" | "cancelled" | "unsupported";
    passed: boolean;
    evaluated: number;
    mismatches: {
      source_sequence: string;
      field: string;
      expected: string;
      actual: string;
    }[];
    unsupported: { source_sequence: string; reason: string }[];
    summary: string;
  };
}

export interface ReplayAPI {
  eligibility(runID: string): Promise<FixtureRecord>;
  create(runID: string): Promise<FixtureRecord>;
  importFixture(): Promise<FixtureRecord | { cancelled: true }>;
  list(): Promise<FixtureRecord[]>;
  inspect(id: string): Promise<FixtureRecord>;
  exportFixture(
    id: string,
  ): Promise<{ exported: boolean; cancelled?: boolean }>;
  deleteFixture(id: string): Promise<{ deleted: boolean; cancelled?: boolean }>;
  evaluate(
    regressionID: string,
    transportID: string,
    plan: UnifiedPlan,
  ): Promise<UnifiedResult>;
}
