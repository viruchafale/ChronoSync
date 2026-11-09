import type { DatasetRecord, BenchmarkDataset } from "./governance";

export type TrustState = "untrusted" | "trusted" | "revoked";
export type CustodyBackend =
  "none" | "legacy_file" | "macos_keychain" | "unsupported";
export type CustodyStatus =
  | "unchecked"
  | "available"
  | "denied"
  | "missing"
  | "corrupt"
  | "unsupported"
  | "recovery_required";
export type MigrationState =
  | "legacy_detected"
  | "migration_pending"
  | "keychain_created"
  | "identity_verified"
  | "legacy_retirement_pending"
  | "migrated"
  | "migration_failed"
  | "recovery_required";
export type VerificationStatus =
  | "valid_trusted"
  | "valid_untrusted"
  | "invalid_signature"
  | "revoked_identity"
  | "unsupported_algorithm"
  | "unsupported_format"
  | "malformed_package"
  | "incomplete_provenance";

export interface SigningIdentity {
  format_version: 1;
  key_id: string;
  algorithm: "ed25519";
  public_key: string;
  label: string;
  created_at: string;
  supersedes?: string;
}

export interface IdentityRecord {
  identity: SigningIdentity;
  trust: TrustState;
  has_private_key: boolean;
  trust_revision: number;
  trust_updated_at?: string;
  warnings: string[];
  custody_backend: CustodyBackend;
  custody_status: CustodyStatus;
  migration_state?: MigrationState;
}

export interface MigrationRecord {
  key_id: string;
  state: MigrationState;
  revision: number;
  updated_at: string;
  eligible: boolean;
  custody_backend: CustodyBackend;
  detail: string;
}

export interface TrustAudit {
  format_version: 1;
  verified: boolean;
  identities: number;
  events: number;
  revoked: number;
  issues: string[];
}

export interface ExchangeManifest {
  format_version: 1;
  domain: "chronosync.benchmark-exchange";
  dataset_content_id: string;
  benchmark_id: string;
  benchmark_version: number;
  compatibility_model: "case-id-v1";
  entries: {
    path: string;
    role: "benchmark" | "analysis_report";
    sha256: string;
    bytes: number;
  }[];
  included_report_ids: string[];
  omitted_report_ids: string[];
  privacy: {
    raw_commands: false;
    raw_outputs: false;
    workspace_paths: false;
    credentials: false;
    environment: false;
    private_keys: false;
    included_evidence: string[];
    omitted_evidence: string[];
  };
}

export interface ExchangeVerification {
  format_version: 1;
  status: VerificationStatus;
  package_id?: string;
  signature_valid: boolean;
  trust: TrustState;
  identity?: SigningIdentity;
  manifest?: ExchangeManifest;
  importable: boolean;
  warnings: string[];
  detail: string;
}

export interface ExchangeImportResult {
  verification: ExchangeVerification;
  dataset: DatasetRecord;
  reports: number;
  duplicate: boolean;
}

export interface ExchangeAPI {
  createIdentity(
    label: string,
    supersedes: string,
  ): Promise<IdentityRecord | { cancelled: true }>;
  identities(): Promise<IdentityRecord[]>;
  migrations(): Promise<MigrationRecord[]>;
  migrate(keyID: string): Promise<MigrationRecord | { cancelled: true }>;
  custodyStatus(keyID: string): Promise<CustodyStatus>;
  auditTrust(): Promise<TrustAudit>;
  setTrust(
    keyID: string,
    state: TrustState,
    reason: string,
  ): Promise<IdentityRecord | { cancelled: true }>;
  exportPackage(
    datasetContentID: string,
    identityKeyID: string,
    includeReportIDs: string[],
  ): Promise<
    | {
        exported: true;
        package_id: string;
        bytes: number;
        verification: ExchangeVerification;
      }
    | { cancelled: true }
  >;
  inspectPackage(): Promise<ExchangeVerification | { cancelled: true }>;
  importPackage(): Promise<
    | ExchangeImportResult
    | {
        imported?: false;
        cancelled?: true;
        verification?: ExchangeVerification;
      }
  >;
}

export interface ExchangeViewDependencies {
  listDatasets(): Promise<DatasetRecord[]>;
  getDataset(id: string): Promise<BenchmarkDataset>;
}
