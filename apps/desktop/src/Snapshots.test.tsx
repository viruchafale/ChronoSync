// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Snapshots } from "./Snapshots";
import type {
  SnapshotAPI,
  Snapshot,
  Branch,
} from "../../../contracts/snapshot";
import type { Run } from "../../../contracts/recorder";
afterEach(cleanup);
const run: Run = {
  id: "run",
  schema_version: 1,
  workspace: "/synthetic",
  adapter_version: "synthetic",
  sandbox: "read-only",
  state: "completed",
  created_at: "",
  timeout_seconds: 1,
  last_sequence: "1",
};
const snapshot: Snapshot = {
  id: "snapshot",
  run_id: run.id,
  workspace: run.workspace,
  created_at: "",
  state: "committed",
  entries: 1,
  omitted: 0,
  bytes: 10,
};
const branch: Branch = {
  id: "branch",
  run_id: run.id,
  snapshot_id: snapshot.id,
  created_at: "",
  state: "ready",
  kind: "branch",
  workspace: "/managed/branch",
};
function fixture(overrides: Partial<SnapshotAPI> = {}): SnapshotAPI {
  return {
    capture: vi.fn(async () => snapshot),
    list: vi.fn(async () => ({ snapshots: [snapshot], next: "" })),
    branches: vi.fn(async () => ({ branches: [], next: "" })),
    validate: vi.fn(async () => snapshot),
    materialize: vi.fn(async () => branch),
    inventory: vi.fn(async () => ({
      objects: 1,
      bytes: 10,
      temporary: 0,
      unreferenced: 0,
      unreferenced_bytes: 0,
    })),
    ...overrides,
  };
}
it("requires source-byte consent and sends exclusions", async () => {
  const api = fixture();
  const user = userEvent.setup();
  render(<Snapshots run={run} api={api} />);
  await screen.findByRole("button", { name: "Branch snapshot" });
  const button = screen.getByRole("button", {
    name: "Capture snapshot",
  }) as HTMLButtonElement;
  expect(button.disabled).toBe(true);
  await user.type(screen.getByLabelText(/Snapshot exclusions/), "private");
  await user.click(screen.getByRole("checkbox"));
  await user.click(button);
  await waitFor(() =>
    expect(api.capture).toHaveBeenCalledWith({
      run_id: "run",
      branch_id: "",
      excludes: ["private"],
      consent: true,
    }),
  );
});
it("shows actual branch preparation then ready and source ancestry", async () => {
  let resolve!: (b: Branch) => void;
  let ready = false;
  const api = fixture({
    materialize: vi.fn(
      () =>
        new Promise<Branch>((r) => {
          resolve = r;
        }),
    ),
    branches: vi.fn(async () => ({
      branches: ready ? [branch] : [],
      next: "",
    })),
  });
  const user = userEvent.setup();
  render(<Snapshots run={run} api={api} />);
  await screen.findByRole("button", { name: "Branch snapshot" });
  await user.click(screen.getByRole("checkbox"));
  await user.click(screen.getByRole("button", { name: "Branch snapshot" }));
  expect(screen.getByText("Preparing branch…")).toBeTruthy();
  ready = true;
  resolve(branch);
  expect(
    await screen.findByText(/branch branch · ready · source snapshot/),
  ).toBeTruthy();
});
it("retains failure diagnostics and disables damaged restore", async () => {
  const api = fixture({
    list: vi.fn(async () => ({
      snapshots: [{ ...snapshot, state: "damaged" as const }],
      next: "",
    })),
    capture: vi.fn(async () => {
      throw new Error("WORKSPACE_CHANGED");
    }),
  });
  const user = userEvent.setup();
  render(<Snapshots run={run} api={api} />);
  const restore = await screen.findByRole("button", {
    name: "Restore snapshot",
  });
  expect((restore as HTMLButtonElement).disabled).toBe(true);
  await user.click(screen.getByRole("checkbox"));
  await user.click(screen.getByRole("button", { name: "Capture snapshot" }));
  expect(await screen.findByRole("alert")).toHaveProperty(
    "textContent",
    "Error: WORKSPACE_CHANGED",
  );
  await waitFor(() => expect(api.list).toHaveBeenCalledTimes(2));
  expect(screen.getByRole("alert").textContent).toContain("WORKSPACE_CHANGED");
});
