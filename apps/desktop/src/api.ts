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
