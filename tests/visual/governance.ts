// Synthetic visual test evidence copied from Governance.test.tsx. Never imported by production.
export const dataset = {
  content_id: "a".repeat(64),
  benchmark_id: "b".repeat(32),
  version: 1,
  name: "Synthetic baseline",
  cases: 1,
  bytes: 512,
  integrity: "verified" as const,
  lifecycle: "active" as const,
  warnings: [],
};

export const audit = {
  format_version: 1 as const,
  usage: {
    reports: 1,
    report_bytes: 256,
    datasets: 1,
    dataset_bytes: 512,
    lineages: 1,
    lineage_bytes: 128,
    retention_events: 3,
    retention_bytes: 96,
    temporary_files: 0,
    temporary_bytes: 0,
    damaged_objects: 0,
    missing_references: 0,
    unmanaged_objects: 0,
    orphaned_metadata: 0,
    total_bytes: 992,
  },
  issues: [],
  retention: [],
  references: [
    {
      from_kind: "dataset" as const,
      from_id: dataset.content_id,
      to_kind: "report" as const,
      to_id: "c".repeat(32),
      relation: "source_report",
    },
  ],
};
