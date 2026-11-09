// Synthetic visual test evidence copied from Replay.test.tsx. Never imported by production.
import type { FixtureRecord } from "../../contracts/replay";
import type { Run } from "../../contracts/recorder";
export const regression: FixtureRecord = {
  id: "a".repeat(64),
  kind: "regression-v1",
  bytes: 500,
  format_version: 1,
  source_execution_id: "run-1",
  interactions: 1,
  supported: ["command.completed"],
  integrity: "verified",
  warnings: ["Equality leakage"],
  details: [
    {
      sequence: "4",
      type: "command.completed",
      expected_output_fingerprint: "b".repeat(64),
      expected_exit_code: 0,
    },
  ],
};
export const transport: FixtureRecord = {
  ...regression,
  id: "c".repeat(64),
  kind: "synthetic-transport-v1",
  supported: ["synthetic.request.v1"],
};
delete transport.source_execution_id;
export const run: Run = {
  schema_version: 1,
  id: "run-1",
  workspace: "/fixture",
  adapter_version: "synthetic",
  sandbox: "read-only",
  state: "completed",
  created_at: "2026-10-31T00:00:00Z",
  timeout_seconds: 30,
  last_sequence: "6",
};
