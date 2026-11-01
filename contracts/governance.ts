import type {
  AnalysisPolicy,
  AnalysisStatus,
  EvidenceDimension,
} from "./analysis";

export type GovernanceObjectKind = "report" | "dataset" | "lineage";
export type RetentionState =
  | "unmanaged"
  | "active"
  | "archived"
  | "expired"
  | "damaged"
  | "deletion_pending"
  | "deleted";

export interface RetentionPolicy {
  format_version: 1;
  name: string;
  expires_at?: string;
  archive_allowed: boolean;
  delete_allowed: boolean;
  automatic: false;
}

export interface DatasetCaseDraft {
  case_id: string;
  source_report_id: string;
  comparison_index: number;
  supported_dimensions: EvidenceDimension[];
}

export interface CreateDataset {
  benchmark_id?: string;
  version: number;
  name: string;
  cases: DatasetCaseDraft[];
  supported_dimensions: EvidenceDimension[];
  required_dimensions: EvidenceDimension[];
  provenance_limitations: string[];
}

export interface BenchmarkDataset {
  format_version: 1;
  benchmark_id: string;
  version: number;
  name: string;
  created_at: string;
  cases: {
    case_id: string;
    source_report_id: string;
    comparison_index: number;
    baseline_observation_id: string;
    observed_observation_id: string;
    comparison_sha256: string;
    expected_status: AnalysisStatus;
    supported_dimensions: EvidenceDimension[];
  }[];
  supported_dimensions: EvidenceDimension[];
  required_dimensions: EvidenceDimension[];
  compatibility_model: "case-id-v1";
  provenance_limitations: string[];
}

export interface DatasetRecord {
  content_id: string;
  benchmark_id?: string;
  version?: number;
  name?: string;
  cases: number;
  bytes: number;
  integrity: "verified" | "damaged";
  duplicate?: boolean;
  lifecycle: RetentionState;
  warnings: string[];
}

export interface CompatibilityResult {
  format_version: 1;
  baseline_content_id: string;
  candidate_content_id: string;
  status: "exact" | "compatible_extension" | "incompatible" | "unsupported";
  missing_cases: string[];
  added_cases: string[];
  changed_cases: string[];
  reasons: string[];
}

export interface Lineage {
  format_version: 1;
  lineage_id: string;
  created_at: string;
  dataset_content_id: string;
  benchmark_id: string;
  dataset_version: number;
  source_report_ids: string[];
  evaluation_report_ids: string[];
  policy: AnalysisPolicy;
  parent_lineage_id?: string;
  sha256: string;
}

export interface LineageSummary {
  lineage_id: string;
  created_at: string;
  dataset_content_id: string;
  benchmark_id: string;
  dataset_version: number;
  policy_name: string;
  integrity: string;
}

export interface RetentionStatus {
  object_kind: GovernanceObjectKind;
  object_id: string;
  revision: number;
  state: RetentionState;
  policy: RetentionPolicy;
  reason?: string;
  updated_at?: string;
}

export interface GovernanceAudit {
  format_version: 1;
  usage: {
    reports: number;
    report_bytes: number;
    datasets: number;
    dataset_bytes: number;
    lineages: number;
    lineage_bytes: number;
    retention_events: number;
    retention_bytes: number;
    temporary_files: number;
    temporary_bytes: number;
    damaged_objects: number;
    missing_references: number;
    unmanaged_objects: number;
    orphaned_metadata: number;
    total_bytes: number;
  };
  issues: {
    code: string;
    object_kind?: GovernanceObjectKind;
    object_id?: string;
    detail: string;
  }[];
  retention: RetentionStatus[];
  references: {
    from_kind: GovernanceObjectKind;
    from_id: string;
    to_kind: GovernanceObjectKind;
    to_id: string;
    relation: string;
  }[];
}

export interface GovernanceAPI {
  create(spec: CreateDataset): Promise<DatasetRecord>;
  importDataset(): Promise<DatasetRecord | { cancelled: true }>;
  list(): Promise<DatasetRecord[]>;
  get(id: string): Promise<BenchmarkDataset>;
  exportDataset(
    id: string,
  ): Promise<{ exported: boolean; cancelled?: boolean }>;
  compare(
    baselineID: string,
    candidateID: string,
  ): Promise<CompatibilityResult>;
  evaluate(
    id: string,
    policy: AnalysisPolicy,
    parentLineageID: string,
  ): Promise<Lineage>;
  lineages(): Promise<LineageSummary[]>;
  lineage(id: string): Promise<Lineage>;
  setRetention(
    kind: GovernanceObjectKind,
    id: string,
    state: RetentionState,
    policy: RetentionPolicy,
    reason: string,
  ): Promise<RetentionStatus>;
  audit(): Promise<GovernanceAudit>;
  deleteObject(
    kind: GovernanceObjectKind,
    id: string,
  ): Promise<{ deleted: boolean; cancelled?: boolean }>;
}

export const defaultRetentionPolicy: RetentionPolicy = {
  format_version: 1,
  name: "manual-retain-v1",
  archive_allowed: true,
  delete_allowed: true,
  automatic: false,
};
