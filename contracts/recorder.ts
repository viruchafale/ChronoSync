export type RunState =
  | "created"
  | "starting"
  | "running"
  | "completed"
  | "failed"
  | "cancelled"
  | "timed_out"
  | "interrupted";
export interface Run {
  adapter?: "codex";
  recorder_version?: string;
  schema_version: 1;
  id: string;
  workspace: string;
  adapter_version: string;
  sandbox: "read-only" | "workspace-write";
  state: RunState;
  created_at: string;
  finished_at?: string;
  started_at?: string;
  history?: "complete" | "partial" | "damaged";
  recovered?: boolean;
  diagnostic?: string;
  timeout_seconds: number;
  last_sequence: string;
  reason?: string;
  exit_code?: number;
}
export interface FileMeta {
  size: number;
  sha256: string;
  mode: number;
}
export interface Change {
  path: string;
  operation: string;
  before?: FileMeta;
  after?: FileMeta;
}
export interface Payload {
  truncations?: {
    field: string;
    original_bytes: number;
    limit_bytes: number;
  }[];
  message?: string;
  upstream_type?: string;
  item_id?: string;
  thread_id?: string;
  command?: string;
  output?: string;
  exit_code?: number;
  status?: RunState;
  reason?: string;
  changes?: Change[];
  source?: string;
}
export interface RecordedEvent {
  schema_version: 2;
  event_id: string;
  run_id: string;
  sequence: string;
  kind: string;
  observed_at: string;
  adapter: "codex";
  payload: Payload;
}
export interface StartSpec {
  workspace: string;
  prompt: string;
  codex_path: string;
  sandbox: "read-only" | "workspace-write";
  timeout_seconds: number;
  consent: boolean;
  excludes: string[];
}
export interface EventPage {
  run: Run;
  events: RecordedEvent[];
  warnings: string[];
  next: string;
}
export interface EventQuery {
  id: string;
  cursor: string;
  kind: string;
  search: string;
  min: string;
  max: string;
  limit: number;
}
export interface EventSummary {
  event_id: string;
  sequence: string;
  kind: string;
  observed_at: string;
  status: string;
  truncated: boolean;
}
export interface TimelinePage {
  run: Run;
  events: EventSummary[];
  next: string;
  high_water: string;
  count: number;
}
export interface RunPage {
  runs: Run[];
  next: string;
}
export interface RecorderAPI {
  queryRuns(cursor: string): Promise<RunPage>;
  query(query: EventQuery): Promise<TimelinePage>;
  detail(id: string, eventID: string): Promise<RecordedEvent>;
  list(): Promise<Run[]>;
  start(spec: StartSpec): Promise<Run>;
  cancel(id: string): Promise<unknown>;
  events(id: string, after: string): Promise<EventPage>;
}
export const isActive = (state: RunState) =>
  ["created", "starting", "running"].includes(state);
