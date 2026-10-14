// @vitest-environment jsdom
import { afterEach, expect, it, vi } from "vitest";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { App } from "./App";
import type {
  RecorderAPI,
  Run,
  RecordedEvent,
} from "../../../contracts/recorder";
afterEach(cleanup);
const run: Run = {
  schema_version: 1,
  id: "synthetic",
  workspace: "/project",
  adapter_version: "synthetic",
  sandbox: "read-only",
  state: "completed",
  created_at: "2026-10-21T00:00:00Z",
  timeout_seconds: 120,
  last_sequence: "2",
};
const events: RecordedEvent[] = [
  {
    schema_version: 2,
    event_id: "a",
    run_id: run.id,
    sequence: "1",
    kind: "command.completed",
    observed_at: run.created_at,
    adapter: "codex",
    payload: { output: "synthetic output" },
  },
  {
    schema_version: 2,
    event_id: "b",
    run_id: run.id,
    sequence: "2",
    kind: "execution.state",
    observed_at: run.created_at,
    adapter: "codex",
    payload: { status: "completed" },
  },
];
function fixture(overrides: Partial<RecorderAPI> = {}): RecorderAPI {
  return {
    list: vi.fn(async () => [run]),
    queryRuns: vi.fn(async () => ({
      runs: overrides.list ? await overrides.list() : [run],
      next: "",
    })),
    query: vi.fn(async () => {
      const page = overrides.events
        ? await overrides.events(run.id, "0")
        : { run, events };
      return {
        run: page.run,
        events: page.events.map((e) => ({
          ...e,
          status: e.payload.status || "",
          truncated: false,
        })),
        next: "",
        high_water: "2",
        count: 2,
      };
    }),
    detail: vi.fn(async (_id, eventID) =>
      events.find((e) => e.event_id === eventID)!,
    ),
    start: vi.fn(async () => run),
    cancel: vi.fn(async () => ({ accepted: true })),
    events: vi.fn(async () => ({ run, events, warnings: [], next: "2" })),
    ...overrides,
  };
}
it("loads persisted history in sequence and shows command detail", async () => {
  const user = userEvent.setup();
  render(<App api={fixture()} />);
  await user.click(
    await screen.findByRole("button", { name: /completed · \/project/ }),
  );
  await screen.findByRole("button", { name: /#1 command.completed/ });
  const rows = screen
    .getAllByRole("button")
    .filter((b) => b.textContent?.startsWith("#"));
  expect(rows.map((b) => b.textContent?.slice(0, 2))).toEqual(["#1", "#2"]);
  await user.click(rows[0]!);
  expect(await screen.findByText(/synthetic output/)).toBeTruthy();
  expect(screen.getByRole("status").textContent).toContain("completed");
});
it("requires explicit consent and sends the selected workspace/policy", async () => {
  const api = fixture({ list: vi.fn(async () => []) });
  const user = userEvent.setup();
  render(<App api={api} />);
  const start = screen.getByRole("button", {
    name: "Start recording",
  }) as HTMLButtonElement;
  expect(start.disabled).toBe(true);
  await user.type(screen.getByLabelText(/Project directory/), "/project");
  await user.type(
    screen.getByRole("textbox", { name: "Codex executable (absolute path)" }),
    "/usr/local/bin/codex",
  );
  await user.type(screen.getByLabelText("Task"), "Inspect fixture");
  await user.click(screen.getByRole("checkbox"));
  await user.click(start);
  await waitFor(() =>
    expect(api.start).toHaveBeenCalledWith(
      expect.objectContaining({
        workspace: "/project",
        consent: true,
        sandbox: "read-only",
        prompt: "Inspect fixture",
      }),
    ),
  );
});
it("requests cancellation and waits for confirmed terminal state", async () => {
  let cancelled = false;
  const current = { ...run, state: "running" as const };
  const api = fixture({
    list: vi.fn(async () => [current]),
    cancel: vi.fn(async () => {
      cancelled = true;
      return { accepted: true };
    }),
    events: vi.fn(async () => ({
      run: cancelled ? { ...run, state: "cancelled" as const } : current,
      events: [],
      warnings: [],
      next: "0",
    })),
  });
  const user = userEvent.setup();
  render(<App api={api} />);
  await user.click(
    await screen.findByRole("button", { name: /running · \/project/ }),
  );
  await user.click(
    await screen.findByRole("button", { name: "Cancel execution" }),
  );
  expect(api.cancel).toHaveBeenCalledWith(run.id);
  await waitFor(() =>
    expect(screen.getByRole("status").textContent).toContain("cancelled"),
  );
});
it("surfaces native errors without claiming recording success", async () => {
  render(
    <App
      api={fixture({
        list: vi.fn(async () => {
          throw new Error("Engine disconnected");
        }),
      })}
    />,
  );
  expect((await screen.findByRole("alert")).textContent).toContain(
    "Engine disconnected",
  );
  expect(screen.queryByRole("status")).toBeNull();
});

it("filters and paginates without retaining previous events", async () => {
  const api = fixture({
    query: vi.fn(async (q) => ({
      run,
      events: [
        {
          event_id: q.cursor ? "b" : "a",
          sequence: q.cursor ? "2" : "1",
          kind: "agent.message",
          observed_at: run.created_at,
          status: "",
          truncated: !q.cursor,
        },
      ],
      next: q.cursor ? "" : "cursor-2",
      high_water: "2",
      count: 2,
    })),
  });
  const user = userEvent.setup();
  render(<App api={api} />);
  await user.click(
    await screen.findByRole("button", { name: /completed · \/project/ }),
  );
  await screen.findByRole("button", { name: /#1 agent.message.*Truncated/ });
  await user.click(screen.getByRole("button", { name: "Next page" }));
  await screen.findByRole("button", { name: /#2 agent.message/ });
  expect(screen.queryByRole("button", { name: /#1 agent.message/ })).toBeNull();
  await user.click(screen.getByRole("button", { name: "Previous page" }));
  await screen.findByRole("button", { name: /#1 agent.message/ });
  await user.type(screen.getByLabelText("Event type"), "agent.message");
  await user.type(screen.getByLabelText("Search diagnostics"), "needle");
  await user.type(screen.getByLabelText("From sequence"), "1");
  await user.type(screen.getByLabelText("Through sequence"), "20");
  await user.click(screen.getByRole("button", { name: "Apply filters" }));
  await waitFor(() =>
    expect(api.query).toHaveBeenLastCalledWith(
      expect.objectContaining({
        cursor: "",
        kind: "agent.message",
        search: "needle",
        min: "1",
        max: "20",
        limit: 100,
      }),
    ),
  );
});
it("shows interrupted history and unavailable payload without fabricating details", async () => {
  const interrupted = {
    ...run,
    state: "interrupted" as const,
    history: "damaged" as const,
    diagnostic: "Missing payload",
  };
  const api = fixture({
    queryRuns: vi.fn(async () => ({ runs: [interrupted], next: "" })),
    query: vi.fn(async () => ({
      run: interrupted,
      events: [
        {
          event_id: "a",
          sequence: "1",
          kind: "command.completed",
          observed_at: run.created_at,
          status: "",
          truncated: true,
        },
      ],
      next: "",
      high_water: "1",
      count: 1,
    })),
    detail: vi.fn(async () => {
      throw new Error("CORRUPT_PAYLOAD");
    }),
  });
  const user = userEvent.setup();
  render(<App api={api} />);
  await user.click(
    await screen.findByRole("button", { name: /interrupted · \/project/ }),
  );
  await user.click(
    await screen.findByRole("button", { name: /#1 command.completed/ }),
  );
  expect(
    await screen.findByText(/Payload unavailable:.*CORRUPT_PAYLOAD/),
  ).toBeTruthy();
  expect(screen.getByRole("status").textContent).toContain("interrupted");
  expect(screen.queryByRole("region", { name: "Event details" })).toBeNull();
});
it("shows loading then an honest empty history", async () => {
  let resolve!: (value: Awaited<ReturnType<RecorderAPI["query"]>>) => void;
  const api = fixture({
    query: vi.fn(
      () =>
        new Promise<Awaited<ReturnType<RecorderAPI["query"]>>>((done) => {
          resolve = done;
        }),
    ),
  });
  const user = userEvent.setup();
  render(<App api={api} />);
  await user.click(
    await screen.findByRole("button", { name: /completed · \/project/ }),
  );
  expect(await screen.findByText("Loading events…")).toBeTruthy();
  resolve({ run, events: [], next: "", high_water: "0", count: 0 });
  expect(await screen.findByText(/No recorded events match/)).toBeTruthy();
});
