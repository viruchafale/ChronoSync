// Synthetic visual test evidence copied from Exchange.test.tsx. Never imported by production.
import type {
  IdentityRecord,
  ExchangeVerification,
} from "../../contracts/exchange";
import type {
  BenchmarkDataset,
  DatasetRecord,
} from "../../contracts/governance";
export const identity: IdentityRecord = {
  identity: {
    format_version: 1,
    key_id: "a".repeat(64),
    algorithm: "ed25519",
    public_key: "synthetic-public-key",
    label: "Personal signer",
    created_at: "2026-11-01T00:00:00Z",
  },
  trust: "trusted",
  has_private_key: true,
  trust_revision: 1,
  custody_backend: "macos_keychain",
  custody_status: "unchecked",
  warnings: [],
};

export const datasetRecord: DatasetRecord = {
  content_id: "b".repeat(64),
  benchmark_id: "c".repeat(32),
  version: 1,
  name: "Portable benchmark",
  cases: 1,
  bytes: 512,
  integrity: "verified",
  lifecycle: "active",
  warnings: [],
};

export const dataset: BenchmarkDataset = {
  format_version: 1,
  benchmark_id: datasetRecord.benchmark_id!,
  version: 1,
  name: datasetRecord.name!,
  created_at: "2026-11-01T00:00:00Z",
  cases: [
    {
      case_id: "case-001",
      source_report_id: "d".repeat(32),
      comparison_index: 0,
      baseline_observation_id: "baseline",
      observed_observation_id: "observed",
      comparison_sha256: "e".repeat(64),
      expected_status: "partial_match",
      supported_dimensions: ["event_type", "ordering"],
    },
  ],
  supported_dimensions: ["event_type", "ordering"],
  required_dimensions: ["event_type", "ordering"],
  compatibility_model: "case-id-v1",
  provenance_limitations: ["Synthetic evidence only."],
};

export const verification: ExchangeVerification = {
  format_version: 1,
  status: "valid_untrusted",
  package_id: "f".repeat(64),
  signature_valid: true,
  trust: "untrusted",
  identity: { ...identity.identity, label: "External signer" },
  manifest: {
    format_version: 1,
    domain: "chronosync.benchmark-exchange",
    dataset_content_id: datasetRecord.content_id,
    benchmark_id: dataset.benchmark_id,
    benchmark_version: 1,
    compatibility_model: "case-id-v1",
    entries: [],
    included_report_ids: [dataset.cases[0]!.source_report_id],
    omitted_report_ids: [],
    privacy: {
      raw_commands: false,
      raw_outputs: false,
      workspace_paths: false,
      credentials: false,
      environment: false,
      private_keys: false,
      included_evidence: ["benchmark_dataset"],
      omitted_evidence: [],
    },
  },
  importable: true,
  warnings: ["Valid does not mean trusted or safe."],
  detail: "Signature valid; identity is not locally trusted.",
};
