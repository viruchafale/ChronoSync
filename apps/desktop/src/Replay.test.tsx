// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Replay } from "./Replay";
import type { ReplayAPI, FixtureRecord } from "../../../contracts/replay";
import type { Run } from "../../../contracts/recorder";

afterEach(cleanup);

const regression: FixtureRecord = {
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
const transport: FixtureRecord = {
  ...regression,
  id: "c".repeat(64),
  kind: "synthetic-transport-v1",
  supported: ["synthetic.request.v1"],
};
delete transport.source_execution_id;
const run: Run = {
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

function api(): ReplayAPI {
  const mismatch: Awaited<ReturnType<ReplayAPI["evaluate"]>> = {
    version: 1,
    status: "completed",
    passed: false,
    summary: "regression mismatch",
    transport: [],
    regression: {
      replay_id: "replay",
      source_execution_id: run.id,
      format_version: 1,
      status: "completed",
      passed: false,
      evaluated: 1,
      mismatches: [
        {
          source_sequence: "4",
          field: "exit_code",
          expected: "0",
          actual: "7",
        },
      ],
      unsupported: [],
      summary: "regression mismatch",
    },
  };
  return {
    eligibility: vi.fn(async () => regression),
    create: vi.fn(async () => regression),
    importFixture: vi.fn(async () => transport),
    list: vi.fn(async () => [regression, transport]),
    inspect: vi.fn(async () => regression),
    exportFixture: vi.fn(async () => ({ exported: true })),
    deleteFixture: vi.fn(async () => ({ deleted: true })),
    evaluate: vi.fn(async () => mismatch),
  };
}

it("creates, inspects, imports, exports and deletes real fixtures", async () => {
  const replay = api();
  const user = userEvent.setup();
  render(<Replay api={replay} run={run} />);
  await user.click(screen.getByRole("button", { name: "Refresh fixtures" }));
  expect(await screen.findByText(/regression-v1/)).toBeTruthy();
  await user.click(
    screen.getByRole("button", {
      name: "Check selected execution eligibility",
    }),
  );
  await user.click(
    await screen.findByRole("button", { name: "Create regression fixture" }),
  );
  expect(replay.create).toHaveBeenCalledWith(run.id);
  await user.click(screen.getAllByRole("button", { name: "Inspect" })[0]!);
  expect(
    await screen.findByRole("region", { name: "Fixture inspection" }),
  ).toBeTruthy();
  await user.click(screen.getByRole("button", { name: "Import fixture…" }));
  expect(replay.importFixture).toHaveBeenCalled();
  await user.click(screen.getAllByRole("button", { name: "Export…" })[0]!);
  expect(replay.exportFixture).toHaveBeenCalledWith(regression.id);
  await user.click(screen.getAllByRole("button", { name: "Delete…" })[0]!);
  expect(replay.deleteFixture).toHaveBeenCalledWith(regression.id);
});

it("shows controlled mismatch details", async () => {
  const replay = api();
  const user = userEvent.setup();
  render(<Replay api={replay} run={run} />);
  await user.click(screen.getByRole("button", { name: "Refresh fixtures" }));
  await waitFor(() => expect(screen.getAllByRole("combobox").length).toBe(2));
  await user.selectOptions(screen.getAllByRole("combobox")[0]!, regression.id);
  await user.selectOptions(screen.getAllByRole("combobox")[1]!, transport.id);
  await user.click(
    screen.getByRole("button", { name: "Run controlled evaluation" }),
  );
  expect(await screen.findByText(/Failed: regression mismatch/)).toBeTruthy();
  expect(
    screen.getByRole("region", { name: "Evaluation result" }).textContent,
  ).toContain("exit_code");
});
