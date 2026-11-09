// @vitest-environment jsdom
import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Analysis } from "./Analysis";
import {
  defaultAnalysisPolicy,
  type AnalysisReport,
} from "../../../contracts/analysis";
import type { Run } from "../../../contracts/recorder";

const runs: Run[] = [
  {
    schema_version: 1,
    id: "11111111111111111111111111111111",
    workspace: "/one",
    adapter_version: "test",
    sandbox: "read-only",
    state: "completed",
    created_at: "2026-10-31T00:00:00Z",
    timeout_seconds: 30,
    last_sequence: "2",
  },
  {
    schema_version: 1,
    id: "22222222222222222222222222222222",
    workspace: "/two",
    adapter_version: "test",
    sandbox: "read-only",
    state: "completed",
    created_at: "2026-10-31T00:01:00Z",
    timeout_seconds: 30,
    last_sequence: "2",
  },
];

const report: AnalysisReport = {
  format_version: 1,
  report_id: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
  created_at: "2026-10-31T00:02:00Z",
  experiment: {
    format_version: 1,
    observations: runs.map((run, index) => ({
      format_version: 1,
      observation_id: `observation-${index}`,
      source_execution_id: run.id,
      source_event_version: 2,
      source_high_water: "2",
      source_history: "complete",
      configuration: {
        adapter: "codex",
        adapter_version: "test",
        recorder_version: "test",
        sandbox: "read-only",
        timeout_seconds: 30,
        workspace_sha256: "hash",
      },
      events: [],
      snapshots: [],
      missing: [],
      unsupported: [],
    })),
    comparisons: [
      {
        format_version: 1,
        baseline_observation: "observation-0",
        observed_observation: "observation-1",
        status: "partial_match",
        summary: "One observable output differs.",
        metrics: {
          baseline_events: 2,
          observed_events: 2,
          aligned_events: 2,
          missing_events: 0,
          additional_events: 0,
          reordered_events: 0,
          changed_dimensions: 1,
          comparable_dimensions: 2,
          event_agreement: 1,
          order_agreement: 1,
          dimension_agreement: 0.5,
        },
        first_divergence: {
          kind: "changed_fingerprint",
          dimension: "output_fingerprint",
          baseline_sequence: "2",
          observed_sequence: "2",
          explanation: "Available fingerprints differ.",
          causality: "unknown",
        },
        findings: [
          {
            kind: "changed_fingerprint",
            dimension: "output_fingerprint",
            baseline_sequence: "2",
            observed_sequence: "2",
            explanation: "Available fingerprints differ.",
            causality: "unknown",
          },
        ],
        alignment: [
          {
            baseline_sequence: "2",
            observed_sequence: "2",
            baseline_kind: "command.completed",
            observed_kind: "command.completed",
            state: "changed",
            dimensions: [],
          },
        ],
        limitations: ["Per-run order does not establish causation."],
      },
    ],
    aggregate: {
      observations: 2,
      comparisons: 1,
      exact: 0,
      partial: 1,
      divergent: 0,
      insufficient: 0,
      unsupported: 0,
      mean_event_agreement: 1,
      mean_dimension_agreement: 0.5,
    },
  },
  policy: defaultAnalysisPolicy,
  evaluation: {
    format_version: 1,
    decision: "fail",
    passed: false,
    summary: "One explicit rule failed.",
    rules: [
      {
        rule: "changed_dimensions",
        passed: false,
        actual: "1",
        required: "<=0",
        reason: "Explicit threshold.",
      },
    ],
  },
  sha256: "report-integrity",
};

afterEach(cleanup);

describe("Analysis", () => {
  it("submits selected recorded evidence and renders explainable divergence", async () => {
    const compare = vi.fn().mockResolvedValue(report);
    render(
      <Analysis
        runs={runs}
        api={{
          compare,
          reports: vi.fn().mockResolvedValue([]),
          report: vi.fn().mockResolvedValue(report),
          reevaluate: vi.fn().mockResolvedValue(report),
        }}
      />,
    );
    await waitFor(() =>
      expect(
        (screen.getByLabelText("Baseline execution") as HTMLSelectElement)
          .value,
      ).toBe(runs[0]!.id),
    );
    fireEvent.click(screen.getByRole("checkbox"));
    fireEvent.click(
      screen.getByRole("button", { name: "Compare recorded evidence" }),
    );
    await waitFor(() => expect(compare).toHaveBeenCalled());
    expect(compare.mock.calls[0]![0]).toEqual([runs[0]!.id, runs[1]!.id]);
    expect(
      await screen.findByRole("heading", { name: "Evaluation: fail" }),
    ).not.toBeNull();
    expect(screen.getByText(/Causality: unknown/)).not.toBeNull();
    expect(screen.getByRole("table").textContent).toContain(
      "command.completed",
    );
  });

  it("loads historical reports and reevaluates preserved evidence", async () => {
    const reevaluate = vi.fn().mockResolvedValue({
      ...report,
      report_id: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
    });
    render(
      <Analysis
        runs={runs}
        api={{
          compare: vi.fn().mockResolvedValue(report),
          reports: vi.fn().mockResolvedValue([
            {
              report_id: report.report_id,
              created_at: report.created_at,
              policy_name: report.policy.name,
              decision: report.evaluation.decision,
              passed: false,
              observations: 2,
            },
          ]),
          report: vi.fn().mockResolvedValue(report),
          reevaluate,
        }}
      />,
    );
    fireEvent.click(
      screen.getByRole("button", { name: "Load historical reports" }),
    );
    fireEvent.click(await screen.findByRole("button", { name: "Open report" }));
    expect(
      await screen.findByRole("heading", { name: "Evaluation: fail" }),
    ).not.toBeNull();
    fireEvent.click(
      screen.getByRole("button", { name: "Reevaluate preserved evidence" }),
    );
    await waitFor(() =>
      expect(reevaluate).toHaveBeenCalledWith(
        report.report_id,
        defaultAnalysisPolicy,
      ),
    );
  });
});
