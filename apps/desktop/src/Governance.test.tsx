// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Governance } from "./Governance";
import type { GovernanceAPI } from "../../../contracts/governance";

const dataset = {
  content_id: "a".repeat(64),
  benchmark_id: "b".repeat(32),
  version: 1,
  name: "Synthetic baseline",
  cases: 1,
  bytes: 512,
  integrity: "verified" as const,
  lifecycle: "active" as const,
  warnings: [],
};

const audit = {
  format_version: 1 as const,
  usage: {
    reports: 1,
    report_bytes: 256,
    datasets: 1,
    dataset_bytes: 512,
    lineages: 1,
    lineage_bytes: 128,
    retention_events: 3,
    retention_bytes: 96,
    temporary_files: 0,
    temporary_bytes: 0,
    damaged_objects: 0,
    missing_references: 0,
    unmanaged_objects: 0,
    orphaned_metadata: 0,
    total_bytes: 992,
  },
  issues: [],
  retention: [],
  references: [
    {
      from_kind: "dataset" as const,
      from_id: dataset.content_id,
      to_kind: "report" as const,
      to_id: "c".repeat(32),
      relation: "source_report",
    },
  ],
};

function mockAPI(overrides: Partial<GovernanceAPI> = {}): GovernanceAPI {
  return {
    create: vi.fn(),
    importDataset: vi.fn(),
    list: vi.fn().mockResolvedValue([dataset]),
    get: vi.fn(),
    exportDataset: vi.fn(),
    compare: vi.fn(),
    evaluate: vi.fn(),
    lineages: vi.fn().mockResolvedValue([
      {
        lineage_id: "d".repeat(32),
        created_at: "2026-10-31T00:00:00Z",
        dataset_content_id: dataset.content_id,
        benchmark_id: dataset.benchmark_id,
        dataset_version: 1,
        policy_name: "strict-observable-v1",
        integrity: "verified",
      },
    ]),
    lineage: vi.fn(),
    setRetention: vi.fn(),
    audit: vi.fn().mockResolvedValue(audit),
    deleteObject: vi.fn(),
    ...overrides,
  };
}

afterEach(cleanup);

describe("Governance", () => {
  it("renders bounded accounting, references, lifecycle, and lineage history", async () => {
    render(<Governance api={mockAPI()} />);
    fireEvent.click(
      screen.getByRole("button", { name: "Refresh governance inventory" }),
    );
    expect(await screen.findByText(/Synthetic baseline/)).not.toBeNull();
    expect(screen.getByText(/992 total bytes/)).not.toBeNull();
    expect(screen.getByText("1 tracked evidence references.")).not.toBeNull();
    expect(
      screen.getByLabelText("Evaluation lineage history").textContent,
    ).toContain("strict-observable-v1");
  });

  it("shows explicit added, missing, and changed compatibility results", async () => {
    const compare = vi.fn().mockResolvedValue({
      format_version: 1,
      baseline_content_id: dataset.content_id,
      candidate_content_id: "e".repeat(64),
      status: "incompatible",
      missing_cases: ["case-old"],
      added_cases: ["case-new"],
      changed_cases: ["case-changed"],
      reasons: ["Existing case definitions changed."],
    });
    render(<Governance api={mockAPI({ compare })} />);
    fireEvent.change(screen.getByLabelText("Baseline content ID"), {
      target: { value: dataset.content_id },
    });
    fireEvent.change(screen.getByLabelText("Candidate content ID"), {
      target: { value: "e".repeat(64) },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Compare benchmark versions" }),
    );
    expect(await screen.findByText("incompatible")).not.toBeNull();
    expect(screen.getByText("Added: case-new")).not.toBeNull();
    expect(screen.getByText("Missing: case-old")).not.toBeNull();
    expect(screen.getByText("Changed: case-changed")).not.toBeNull();
  });

  it("surfaces reference-based deletion refusal without claiming deletion", async () => {
    const deleteObject = vi
      .fn()
      .mockRejectedValue(
        new Error("REFERENCED: delete dependent objects first"),
      );
    render(<Governance api={mockAPI({ deleteObject })} />);
    fireEvent.change(screen.getByLabelText("Object ID"), {
      target: { value: dataset.content_id },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Delete governed object" }),
    );
    await waitFor(() => expect(deleteObject).toHaveBeenCalled());
    expect((await screen.findByRole("alert")).textContent).toContain(
      "REFERENCED",
    );
    expect(screen.queryByText(/tombstoned/)).toBeNull();
  });
});
