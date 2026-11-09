// @vitest-environment jsdom
import { afterEach, beforeAll, expect, it, vi } from "vitest";
import {
  cleanup,
  render,
  screen,
  within,
  fireEvent,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Workbench, CopyButton } from "./Workbench";
beforeAll(() => {
  HTMLDialogElement.prototype.showModal = function () {
    this.open = true;
  };
  HTMLDialogElement.prototype.close = function () {
    this.open = false;
  };
});
afterEach(() => {
  cleanup();
  localStorage.clear();
  vi.restoreAllMocks();
});
it("routes navigation and persists sidebar density", async () => {
  const user = userEvent.setup();
  const navigate = vi.fn();
  render(
    <Workbench
      page="overview"
      navigate={navigate}
      connection="Engine unavailable"
    >
      <p>Evidence</p>
    </Workbench>,
  );
  await user.click(
    within(screen.getByRole("navigation")).getByRole("button", {
      name: "Replay & regression",
    }),
  );
  expect(navigate).toHaveBeenCalledWith("replay");
  expect(
    screen
      .getByRole("button", { name: "Overview" })
      .getAttribute("aria-current"),
  ).toBe("page");
  await user.click(screen.getByRole("button", { name: "Collapse sidebar" }));
  expect(localStorage.getItem("chronosync.sidebar")).toBe("compact");
  expect(screen.getByText("Engine unavailable")).toBeTruthy();
});
it("resets document scroll when the destination changes", () => {
  const navigate = vi.fn();
  const { rerender } = render(
    <Workbench
      page="analysis"
      navigate={navigate}
      connection="Engine unavailable"
    >
      <p>Analysis evidence</p>
    </Workbench>,
  );
  document.documentElement.scrollTop = 640;
  document.documentElement.scrollLeft = 96;
  rerender(
    <Workbench
      page="governance"
      navigate={navigate}
      connection="Engine unavailable"
    >
      <p>Governance evidence</p>
    </Workbench>,
  );
  expect(document.documentElement.scrollTop).toBe(0);
  expect(document.documentElement.scrollLeft).toBe(0);
});
it("opens, searches and closes the command palette with focus restoration", async () => {
  const user = userEvent.setup();
  const navigate = vi.fn();
  render(
    <Workbench
      page="overview"
      navigate={navigate}
      connection="Engine unavailable"
    >
      <p>Evidence</p>
    </Workbench>,
  );
  await user.keyboard("{Control>}k{/Control}");
  const search = screen.getByRole("textbox", { name: "Search commands" });
  expect(document.activeElement).toBe(search);
  await user.type(search, "trust");
  await user.click(
    within(screen.getByRole("dialog")).getByRole("button", {
      name: /Exchange & trust/,
    }),
  );
  expect(navigate).toHaveBeenCalledWith("exchange");
  expect(document.activeElement).toBe(
    screen.getByRole("button", { name: /Search commands/ }),
  );
  await user.click(screen.getByRole("button", { name: /Search commands/ }));
  fireEvent(screen.getByRole("dialog"), new Event("cancel", { bubbles: true }));
  expect(screen.queryByRole("dialog")).toBeNull();
});
it("copies exact evidence and reports clipboard failures", async () => {
  const user = userEvent.setup();
  const copy = vi.spyOn(navigator.clipboard, "writeText").mockResolvedValue();
  render(<CopyButton value="evidence-id" label="Copy ID" />);
  await user.click(screen.getByRole("button", { name: "Copy ID" }));
  expect(copy).toHaveBeenCalledWith("evidence-id");
  expect(await screen.findByText("Copied")).toBeTruthy();
  copy.mockRejectedValue(new Error("denied"));
  await user.click(screen.getByRole("button", { name: "Copy ID" }));
  expect(await screen.findByText("Clipboard unavailable")).toBeTruthy();
});
