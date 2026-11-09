import { invoke, isTauri } from "@tauri-apps/api/core";
import type { RecorderAPI } from "../../../contracts/recorder";
function desktop<T>(
  command: string,
  args?: Record<string, unknown>,
): Promise<T> {
  if (!isTauri())
    return Promise.reject(
      new Error(
        "Recording requires the native desktop app. Run npm run desktop:dev.",
      ),
    );
  return invoke<T>(command, args);
}
export const api: RecorderAPI = {
  queryRuns: (cursor) => desktop("run_query", { cursor }),
  query: (query) => desktop("event_query", { query }),
  detail: (id, eventId) => desktop("event_get", { id, eventId }),
  list: () => desktop("run_list"),
  start: (spec) => desktop("run_start", { spec }),
  cancel: (id) => desktop("run_cancel", { id }),
  events: (id, after) => desktop("event_list", { id, after }),
};

export const snapshotAPI: import("../../../contracts/snapshot").SnapshotAPI = {
  capture: (spec) => desktop("snapshot_capture", { spec }),
  list: (runId, after) => desktop("snapshot_list", { runId, after }),
  branches: (runId, after) => desktop("branch_list", { runId, after }),
  validate: (id) => desktop("snapshot_validate", { id }),
  materialize: (snapshotId, branch, consent) =>
    desktop("snapshot_materialize", { snapshotId, branch, consent }),
  inventory: () => desktop("object_inventory"),
};

export const replayAPI: import("../../../contracts/replay").ReplayAPI = {
  eligibility: (runId) => desktop("fixture_eligibility", { runId }),
  create: (runId) => desktop("fixture_create", { runId }),
  importFixture: () => desktop("fixture_import"),
  list: () => desktop("fixture_list"),
  inspect: (id) => desktop("fixture_inspect", { id }),
  exportFixture: (id) => desktop("fixture_export", { id }),
  deleteFixture: (id) => desktop("fixture_delete", { id }),
  evaluate: (regressionId, transportId, plan) =>
    desktop("replay_evaluate", { regressionId, transportId, plan }),
};

export const analysisAPI: import("../../../contracts/analysis").AnalysisAPI = {
  compare: (runIds, policy) => desktop("analysis_compare", { runIds, policy }),
  reports: () => desktop("analysis_report_list"),
  report: (id) => desktop("analysis_report_get", { id }),
  reevaluate: (id, policy) =>
    desktop("analysis_report_reevaluate", { id, policy }),
};

export const governanceAPI: import("../../../contracts/governance").GovernanceAPI =
  {
    create: (spec) => desktop("benchmark_create", { spec }),
    importDataset: () => desktop("benchmark_import"),
    list: () => desktop("benchmark_list"),
    get: (id) => desktop("benchmark_get", { id }),
    exportDataset: (id) => desktop("benchmark_export", { id }),
    compare: (baselineId, candidateId) =>
      desktop("benchmark_compare", { baselineId, candidateId }),
    evaluate: (id, policy, parentLineageId) =>
      desktop("benchmark_evaluate", { id, policy, parentLineageId }),
    lineages: () => desktop("lineage_list"),
    lineage: (id) => desktop("lineage_get", { id }),
    setRetention: (kind, id, state, policy, reason) =>
      desktop("retention_set", { kind, id, state, policy, reason }),
    audit: () => desktop("governance_audit"),
    deleteObject: (kind, id) => desktop("governance_delete", { kind, id }),
  };

export const exchangeAPI: import("../../../contracts/exchange").ExchangeAPI = {
  createIdentity: (label, supersedes) =>
    desktop("identity_create", { label, supersedes }),
  identities: () => desktop("identity_list"),
  migrations: () => desktop("identity_migration_list"),
  migrate: (keyId) => desktop("identity_migrate", { keyId }),
  custodyStatus: (keyId) => desktop("identity_custody_status", { keyId }),
  auditTrust: () => desktop("identity_trust_audit"),
  setTrust: (keyId, trustState, reason) =>
    desktop("identity_trust", { keyId, trustState, reason }),
  exportPackage: (datasetContentId, identityKeyId, includeReportIds) =>
    desktop("exchange_export", {
      datasetContentId,
      identityKeyId,
      includeReportIds,
    }),
  inspectPackage: () => desktop("exchange_inspect"),
  importPackage: () => desktop("exchange_import"),
};
