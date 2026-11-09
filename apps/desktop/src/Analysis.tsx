import { useEffect, useState } from "react";
import { SectionNav, CopyButton, Badge } from "./Workbench";
import type { Run } from "../../../contracts/recorder";
import {
  defaultAnalysisPolicy,
  type AnalysisAPI,
  type AnalysisPolicy,
  type AnalysisReport,
  type AnalysisReportSummary,
} from "../../../contracts/analysis";

export function Analysis({ api, runs }: { api: AnalysisAPI; runs: Run[] }) {
  const [baseline, setBaseline] = useState("");
  const [targets, setTargets] = useState<string[]>([]);
  const [policyText, setPolicyText] = useState(
    JSON.stringify(defaultAnalysisPolicy, null, 2),
  );
  const [report, setReport] = useState<AnalysisReport | null>(null);
  const [reports, setReports] = useState<AnalysisReportSummary[]>([]);
  const [comparisonIndex, setComparisonIndex] = useState(0);
  const [focusedFinding, setFocusedFinding] = useState(0);
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  useEffect(() => {
    if (!baseline && runs.length > 0) setBaseline(runs[0]!.id);
  }, [baseline, runs]);

  const action = async (operation: () => Promise<void>) => {
    setError("");
    setMessage("");
    try {
      await operation();
    } catch (value) {
      setError(String(value));
    }
  };
  const policy = () => JSON.parse(policyText) as AnalysisPolicy;
  const comparison = report?.experiment.comparisons[comparisonIndex];

  return (
    <section aria-labelledby="analysis-heading">
      <h2 id="analysis-heading">Determinism evidence analysis</h2>
      <SectionNav
        items={[
          ["analysis-configuration", "Comparison setup"],
          ["analysis-results", "Evidence & findings"],
        ]}
      />
      <p>
        Compare recorded evidence only. This does not rerun an execution,
        contact a provider, reveal hidden model state, or establish causation.
      </p>
      {error && <p role="alert">{error}</p>}
      {message && <p role="status">{message}</p>}
      <div id="analysis-configuration" className="comparison-configuration">
        <label>
          Baseline execution
          <select
            value={baseline}
            onChange={(event) => {
              setBaseline(event.target.value);
              setTargets((current) =>
                current.filter((id) => id !== event.target.value),
              );
            }}
          >
            <option value="">Select baseline</option>
            {runs.map((run) => (
              <option key={run.id} value={run.id}>
                {run.created_at} · {run.id}
              </option>
            ))}
          </select>
        </label>
        <fieldset>
          <legend>Comparison executions</legend>
          {runs
            .filter((run) => run.id !== baseline)
            .map((run) => (
              <label key={run.id}>
                <input
                  type="checkbox"
                  checked={targets.includes(run.id)}
                  onChange={(event) =>
                    setTargets((current) =>
                      event.target.checked
                        ? [...current, run.id]
                        : current.filter((id) => id !== run.id),
                    )
                  }
                />
                {run.created_at} · {run.id}
              </label>
            ))}
        </fieldset>
      </div>
      <label>
        Versioned evaluation policy
        <textarea
          value={policyText}
          onChange={(event) => setPolicyText(event.target.value)}
        />
      </label>
      <button
        disabled={!baseline || targets.length === 0}
        onClick={() =>
          void action(async () => {
            const value = await api.compare([baseline, ...targets], policy());
            setReport(value);
            setComparisonIndex(0);
            setFocusedFinding(0);
            setMessage("Analysis report stored.");
          })
        }
      >
        Compare recorded evidence
      </button>
      <button
        onClick={() =>
          void action(async () => {
            setReports(await api.reports());
            setMessage("Historical reports loaded.");
          })
        }
      >
        Load historical reports
      </button>
      <div id="analysis-results">
        {reports.length > 0 && (
          <section aria-label="Historical analysis reports">
            <h3>Historical reports</h3>
            <ul>
              {reports.map((item) => (
                <li key={item.report_id}>
                  {item.created_at} · {item.policy_name} · {item.decision}
                  <button
                    onClick={() =>
                      void action(async () => {
                        setReport(await api.report(item.report_id));
                        setComparisonIndex(0);
                        setFocusedFinding(0);
                      })
                    }
                  >
                    Open report
                  </button>
                </li>
              ))}
            </ul>
          </section>
        )}
        {report && (
          <section aria-label="Analysis report">
            <h3>
              Evaluation: <Badge value={report.evaluation.decision} />
            </h3>
            <CopyButton value={report.report_id} label="Copy report ID" />
            <p role="status">{report.evaluation.summary}</p>
            <p>
              {report.experiment.aggregate.observations} observations ·{" "}
              {report.experiment.aggregate.exact} exact ·{" "}
              {report.experiment.aggregate.partial} partial ·{" "}
              {report.experiment.aggregate.divergent} divergent ·{" "}
              {report.experiment.aggregate.insufficient} insufficient ·{" "}
              {report.experiment.aggregate.unsupported} unsupported
            </p>
            <button
              onClick={() =>
                void action(async () => {
                  const value = await api.reevaluate(
                    report.report_id,
                    policy(),
                  );
                  setReport(value);
                  setComparisonIndex(0);
                  setFocusedFinding(0);
                  setMessage(
                    "Preserved observations reevaluated in a new report.",
                  );
                })
              }
            >
              Reevaluate preserved evidence
            </button>
            <section aria-label="Evidence quality">
              <h4>Evidence quality</h4>
              <ul>
                {report.experiment.observations.map((observation) => (
                  <li key={observation.observation_id}>
                    {observation.source_execution_id}:{" "}
                    {observation.events.length} events,{" "}
                    {observation.missing.length} missing,{" "}
                    {observation.unsupported.length} unsupported
                  </li>
                ))}
              </ul>
            </section>
            {report.experiment.comparisons.length > 1 && (
              <label>
                Comparison result
                <select
                  value={comparisonIndex}
                  onChange={(event) => {
                    setComparisonIndex(Number(event.target.value));
                    setFocusedFinding(0);
                  }}
                >
                  {report.experiment.comparisons.map((item, index) => (
                    <option key={item.observed_observation} value={index}>
                      Baseline vs observation {index + 2}
                    </option>
                  ))}
                </select>
              </label>
            )}
            {comparison && (
              <section aria-label="Selected comparison">
                <h4>{comparison.status}</h4>
                <p>{comparison.summary}</p>
                <p>
                  Event agreement{" "}
                  {comparison.metrics.event_agreement.toFixed(3)} · order
                  agreement {comparison.metrics.order_agreement.toFixed(3)} ·
                  fingerprint agreement{" "}
                  {comparison.metrics.dimension_agreement.toFixed(3)}
                </p>
                {comparison.first_divergence && (
                  <button onClick={() => setFocusedFinding(0)}>
                    First observed divergence:{" "}
                    {comparison.first_divergence.kind}
                  </button>
                )}
                <section aria-label="Divergence details">
                  <h5>Divergence details</h5>
                  {comparison.findings.length === 0 ? (
                    <p>No divergence in the available comparison boundary.</p>
                  ) : (
                    <ol>
                      {comparison.findings.map((finding, index) => (
                        <li
                          key={`${finding.kind}-${index}`}
                          aria-current={index === focusedFinding}
                        >
                          <button onClick={() => setFocusedFinding(index)}>
                            {finding.kind} · {finding.dimension}
                          </button>{" "}
                          {finding.explanation} Causality: {finding.causality}.
                        </li>
                      ))}
                    </ol>
                  )}
                </section>
                <section aria-label="Aligned timelines">
                  <h5>Aligned timelines</h5>
                  <table>
                    <thead>
                      <tr>
                        <th>Baseline</th>
                        <th>Observed</th>
                        <th>Evidence</th>
                      </tr>
                    </thead>
                    <tbody>
                      {comparison.alignment.map((item, index) => (
                        <tr
                          key={`${item.baseline_sequence}-${item.observed_sequence}-${index}`}
                        >
                          <td>
                            {item.baseline_sequence || "—"} {item.baseline_kind}
                          </td>
                          <td>
                            {item.observed_sequence || "—"} {item.observed_kind}
                          </td>
                          <td>{item.state}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </section>
                {comparison.findings.some(
                  (finding) =>
                    finding.dimension === "workspace_fingerprint" ||
                    finding.dimension === "snapshot_identity",
                ) && (
                  <p>
                    Workspace evidence differs within the recorded boundary.
                  </p>
                )}
                {comparison.limitations.map((limitation) => (
                  <p key={limitation}>{limitation}</p>
                ))}
              </section>
            )}
            <section aria-label="Evaluation policy rules">
              <h4>Policy rules</h4>
              <ul>
                {report.evaluation.rules.map((rule) => (
                  <li key={rule.rule}>
                    {rule.passed ? "Pass" : "Fail"}: {rule.rule} · actual{" "}
                    {rule.actual} · required {rule.required}
                  </li>
                ))}
              </ul>
            </section>
          </section>
        )}
      </div>
    </section>
  );
}
