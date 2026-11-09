import { createRoot } from "react-dom/client";
import { App } from "../../apps/desktop/src/App";
import "../../apps/desktop/src/style.css";
import type { RecorderAPI, RecordedEvent } from "../../contracts/recorder";
import type { SnapshotAPI } from "../../contracts/snapshot";
import type { ReplayAPI } from "../../contracts/replay";
import type { AnalysisAPI } from "../../contracts/analysis";
import type { GovernanceAPI } from "../../contracts/governance";
import type { ExchangeAPI } from "../../contracts/exchange";
import { runs, report } from "./analysis";
import { dataset, audit } from "./governance";
import {
  identity,
  datasetRecord,
  dataset as portableDataset,
  verification,
} from "./exchange";
import { regression, transport } from "./replay";

// Only this separate Vite entry imports test evidence. No production mode switch.
// Unsupported/write/native operations reject; nothing launches or persists.
function readOnly<T extends object>(reads: Partial<T>): T {
  return new Proxy(reads, {
    get(target, key) {
      return (
        Reflect.get(target, key) ??
        (() =>
          Promise.reject(
            new Error(
              "VISUAL_TEST_READ_ONLY: native and write operations are disabled",
            ),
          ))
      );
    },
  }) as T;
}
const selected = runs[0]!;
const events: RecordedEvent[] = [
  {
    schema_version: 2,
    event_id: "a".repeat(64),
    run_id: selected.id,
    sequence: "1",
    kind: "command.completed",
    observed_at: selected.created_at,
    adapter: "codex",
    payload: {
      command: "synthetic command — inert recorded text",
      output:
        "Synthetic output\nA long technical identifier: " +
        "0123456789abcdef".repeat(12),
      exit_code: 7,
    },
  },
  {
    schema_version: 2,
    event_id: "b".repeat(64),
    run_id: selected.id,
    sequence: "2",
    kind: "execution.warning",
    observed_at: selected.created_at,
    adapter: "codex",
    payload: {
      message: "Synthetic missing evidence warning; no causality claim.",
    },
  },
];
const recorder = readOnly<RecorderAPI>({
  list: async () => runs,
  queryRuns: async () => ({ runs, next: "" }),
  query: async (query) => ({
    run: runs.find((r) => r.id === query.id)!,
    events: events
      .filter(
        (e) =>
          (!query.kind || e.kind === query.kind) &&
          (!query.search || JSON.stringify(e.payload).includes(query.search)),
      )
      .map((e) => ({
        event_id: e.event_id,
        sequence: e.sequence,
        kind: e.kind,
        observed_at: e.observed_at,
        status: e.payload.exit_code === 7 ? "failed" : "",
        truncated: false,
      })),
    next: "",
    high_water: "2",
    count: 2,
  }),
  detail: async (_id, id) => events.find((e) => e.event_id === id)!,
});
const snapshot = {
  id: "c".repeat(64),
  run_id: selected.id,
  workspace: "/synthetic/source",
  created_at: selected.created_at,
  state: "committed" as const,
  entries: 12,
  omitted: 2,
  bytes: 8192,
};
const snapshots = readOnly<SnapshotAPI>({
  list: async () => ({
    snapshots: [
      snapshot,
      {
        ...snapshot,
        id: "d".repeat(64),
        state: "damaged",
        diagnostic: "Synthetic integrity refusal",
      },
    ],
    next: "",
  }),
  branches: async () => ({
    branches: [
      {
        id: "branch-synthetic",
        snapshot_id: snapshot.id,
        run_id: selected.id,
        created_at: selected.created_at,
        kind: "branch",
        state: "ready",
        workspace: "/synthetic/managed/branch",
      },
    ],
    next: "",
  }),
  inventory: async () => ({
    objects: 12,
    bytes: 8192,
    temporary: 0,
    unreferenced: 0,
    unreferenced_bytes: 0,
  }),
});
const replay = readOnly<ReplayAPI>({
  list: async () => [regression, transport],
  inspect: async () => regression,
  eligibility: async () => regression,
});
const analysis = readOnly<AnalysisAPI>({
  compare: async () => report,
  reports: async () => [],
  report: async () => report,
});
const governance = readOnly<GovernanceAPI>({
  list: async () => [dataset, datasetRecord],
  get: async () => portableDataset,
  audit: async () => audit,
  lineages: async () => [],
});
const exchange = readOnly<ExchangeAPI>({
  identities: async () => [identity],
  migrations: async () => [],
  inspectPackage: async () => verification,
});
createRoot(document.getElementById("root")!).render(
  <>
    <div
      style={{
        position: "fixed",
        bottom: 0,
        right: 0,
        zIndex: 20,
        background: "#47391f",
        color: "#ffe0a3",
        padding: "3px 8px",
        fontSize: 10,
      }}
    >
      SYNTHETIC VISUAL TEST · no engine connection · writes disabled
    </div>
    <App
      api={recorder}
      snapshots={snapshots}
      replay={replay}
      analysis={analysis}
      governance={governance}
      exchange={exchange}
    />
  </>,
);
