export interface Snapshot {
  id: string;
  run_id: string;
  event_id?: string;
  branch_id?: string;
  workspace: string;
  created_at: string;
  state: "preparing" | "committed" | "failed" | "damaged";
  diagnostic?: string;
  manifest?: { hash: string; size: number };
  entries: number;
  omitted: number;
  bytes: number;
}
export interface Branch {
  id: string;
  snapshot_id: string;
  parent_id?: string;
  run_id: string;
  created_at: string;
  kind: "restore" | "branch";
  state: "preparing" | "ready" | "failed";
  workspace: string;
  diagnostic?: string;
}
export interface SnapshotAPI {
  capture(spec: {
    run_id: string;
    branch_id: string;
    excludes: string[];
    consent: boolean;
  }): Promise<Snapshot>;
  list(
    runID: string,
    after: string,
  ): Promise<{ snapshots: Snapshot[]; next: string }>;
  branches(
    runID: string,
    after: string,
  ): Promise<{ branches: Branch[]; next: string }>;
  validate(id: string): Promise<Snapshot>;
  materialize(
    snapshotID: string,
    branch: boolean,
    consent: boolean,
  ): Promise<Branch>;
  inventory(): Promise<{
    objects: number;
    bytes: number;
    temporary: number;
    unreferenced: number;
    unreferenced_bytes: number;
  }>;
}
