import { useState } from "react";
import { SectionNav } from "./Workbench";
import {
  defaultRetentionPolicy,
  type BenchmarkDataset,
  type CompatibilityResult,
  type DatasetRecord,
  type GovernanceAPI,
  type GovernanceAudit,
  type GovernanceObjectKind,
  type Lineage,
  type LineageSummary,
  type RetentionState,
} from "../../../contracts/governance";
import {
  defaultAnalysisPolicy,
  type AnalysisPolicy,
} from "../../../contracts/analysis";

export function Governance({ api }: { api: GovernanceAPI }) {
  const [datasets, setDatasets] = useState<DatasetRecord[]>([]);
  const [lineages, setLineages] = useState<LineageSummary[]>([]);
  const [selected, setSelected] = useState<BenchmarkDataset | null>(null);
  const [lineage, setLineage] = useState<Lineage | null>(null);
  const [comparison, setComparison] = useState<CompatibilityResult | null>(
    null,
  );
  const [audit, setAudit] = useState<GovernanceAudit | null>(null);
  const [baselineID, setBaselineID] = useState("");
  const [candidateID, setCandidateID] = useState("");
  const [evaluationID, setEvaluationID] = useState("");
  const [parentLineageID, setParentLineageID] = useState("");
  const [policyText, setPolicyText] = useState(
    JSON.stringify(defaultAnalysisPolicy, null, 2),
  );
  const [reportID, setReportID] = useState("");
  const [datasetName, setDatasetName] = useState("Local benchmark");
  const [objectKind, setObjectKind] = useState<GovernanceObjectKind>("dataset");
  const [objectID, setObjectID] = useState("");
  const [retentionState, setRetentionState] =
    useState<RetentionState>("active");
  const [error, setError] = useState("");
  const [message, setMessage] = useState("");

  async function action(operation: () => Promise<void>) {
    setError("");
    setMessage("");
    try {
      await operation();
    } catch (value) {
      setError(String(value));
    }
  }

  async function refresh() {
    const [nextDatasets, nextLineages, nextAudit] = await Promise.all([
      api.list(),
      api.lineages(),
      api.audit(),
    ]);
    setDatasets(nextDatasets);
    setLineages(nextLineages);
    setAudit(nextAudit);
  }

  const policy = () => JSON.parse(policyText) as AnalysisPolicy;

  return (
    <section aria-labelledby="governance-heading">
      <h2 id="governance-heading">Evidence retention and benchmarks</h2>
      <SectionNav
        items={[
          ["benchmark-create-heading", "Create dataset"],
          ["dataset-library", "Dataset library"],
          ["compatibility-heading", "Compatibility"],
          ["reevaluation-heading", "Evaluation lineage"],
          ["retention-heading", "Retention"],
        ]}
      />
      <p>
        Govern local, bounded analysis evidence. Integrity fingerprints reveal
        equality; they do not prove authorship, authenticity, causality, or
        whole-run determinism. No benchmark action executes a command or
        contacts a provider.
      </p>
      {error && <p role="alert">{error}</p>}
      {message && <p role="status">{message}</p>}
      <button
        onClick={() =>
          void action(async () => {
            await refresh();
            setMessage("Governance inventory refreshed.");
          })
        }
      >
        Refresh governance inventory
      </button>
      <button
        onClick={() =>
          void action(async () => {
            const result = await api.importDataset();
            if ("cancelled" in result) {
              setMessage("Benchmark import cancelled.");
              return;
            }
            await refresh();
            setMessage(
              result.duplicate
                ? "Identical benchmark already exists."
                : "Benchmark imported and retained.",
            );
          })
        }
      >
        Import benchmark file
      </button>

      <section aria-labelledby="benchmark-create-heading">
        <h3 id="benchmark-create-heading">Create a benchmark dataset</h3>
        <label>
          Dataset name
          <input
            value={datasetName}
            onChange={(event) => setDatasetName(event.target.value)}
          />
        </label>
        <label>
          Source analysis report ID
          <input
            value={reportID}
            onChange={(event) => setReportID(event.target.value)}
          />
        </label>
        <button
          disabled={!reportID || !datasetName}
          onClick={() =>
            void action(async () => {
              const record = await api.create({
                version: 1,
                name: datasetName,
                cases: [
                  {
                    case_id: "case-001",
                    source_report_id: reportID,
                    comparison_index: 0,
                    supported_dimensions: ["event_type", "ordering"],
                  },
                ],
                supported_dimensions: ["event_type", "ordering"],
                required_dimensions: ["event_type", "ordering"],
                provenance_limitations: [
                  "Derived from bounded recorded observations; hidden provider state and causality are unavailable.",
                ],
              });
              setEvaluationID(record.content_id);
              await refresh();
              setMessage("Versioned benchmark stored with active retention.");
            })
          }
        >
          Create benchmark
        </button>
      </section>

      <section id="dataset-library" aria-label="Benchmark datasets">
        <h3>Benchmark datasets</h3>
        {datasets.length === 0 ? (
          <p>No benchmark datasets are loaded.</p>
        ) : (
          <table>
            <thead>
              <tr>
                <th>Name and version</th>
                <th>Integrity</th>
                <th>Lifecycle</th>
                <th>Storage</th>
                <th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {datasets.map((dataset) => (
                <tr key={dataset.content_id}>
                  <td>
                    {dataset.name || "Damaged dataset"} v
                    {dataset.version || "?"}
                  </td>
                  <td>{dataset.integrity}</td>
                  <td>{dataset.lifecycle}</td>
                  <td>{dataset.bytes} bytes</td>
                  <td>
                    <button
                      disabled={dataset.integrity !== "verified"}
                      onClick={() =>
                        void action(async () => {
                          setSelected(await api.get(dataset.content_id));
                        })
                      }
                    >
                      Inspect
                    </button>
                    <button
                      disabled={dataset.integrity !== "verified"}
                      onClick={() => void api.exportDataset(dataset.content_id)}
                    >
                      Export
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {selected && (
        <section aria-label="Selected benchmark">
          <h3>{selected.name}</h3>
          <p>
            Family {selected.benchmark_id} · version {selected.version} ·{" "}
            {selected.cases.length} cases
          </p>
          <ul>
            {selected.provenance_limitations.map((limitation) => (
              <li key={limitation}>{limitation}</li>
            ))}
          </ul>
        </section>
      )}

      <section aria-labelledby="compatibility-heading">
        <h3 id="compatibility-heading">Version compatibility</h3>
        <label>
          Baseline content ID
          <input
            value={baselineID}
            onChange={(event) => setBaselineID(event.target.value)}
          />
        </label>
        <label>
          Candidate content ID
          <input
            value={candidateID}
            onChange={(event) => setCandidateID(event.target.value)}
          />
        </label>
        <button
          disabled={!baselineID || !candidateID}
          onClick={() =>
            void action(async () => {
              setComparison(await api.compare(baselineID, candidateID));
            })
          }
        >
          Compare benchmark versions
        </button>
        {comparison && (
          <section aria-label="Compatibility result">
            <h4>{comparison.status}</h4>
            <p>Added: {comparison.added_cases.join(", ") || "none"}</p>
            <p>Missing: {comparison.missing_cases.join(", ") || "none"}</p>
            <p>Changed: {comparison.changed_cases.join(", ") || "none"}</p>
            {comparison.reasons.map((reason) => (
              <p key={reason}>{reason}</p>
            ))}
          </section>
        )}
      </section>

      <section aria-labelledby="reevaluation-heading">
        <h3 id="reevaluation-heading">Re-evaluate preserved evidence</h3>
        <label>
          Dataset content ID
          <input
            value={evaluationID}
            onChange={(event) => setEvaluationID(event.target.value)}
          />
        </label>
        <label>
          Optional parent lineage ID
          <input
            value={parentLineageID}
            onChange={(event) => setParentLineageID(event.target.value)}
          />
        </label>
        <label>
          Versioned analysis policy
          <textarea
            value={policyText}
            onChange={(event) => setPolicyText(event.target.value)}
          />
        </label>
        <button
          disabled={!evaluationID}
          onClick={() =>
            void action(async () => {
              const result = await api.evaluate(
                evaluationID,
                policy(),
                parentLineageID,
              );
              setLineage(result);
              await refresh();
              setMessage(
                "New evaluation reports and immutable lineage stored; source history was preserved.",
              );
            })
          }
        >
          Evaluate benchmark
        </button>
        {lineage && (
          <p>
            Lineage {lineage.lineage_id}: {lineage.evaluation_report_ids.length}{" "}
            new reports under {lineage.policy.name}.
          </p>
        )}
        <ul aria-label="Evaluation lineage history">
          {lineages.map((item) => (
            <li key={item.lineage_id}>
              {item.created_at} · dataset v{item.dataset_version} ·{" "}
              {item.policy_name}
              <button
                onClick={() =>
                  void action(async () => {
                    setLineage(await api.lineage(item.lineage_id));
                  })
                }
              >
                Inspect lineage
              </button>
            </li>
          ))}
        </ul>
      </section>

      <section aria-labelledby="retention-heading">
        <h3 id="retention-heading">Explicit retention and deletion</h3>
        <p>
          Nothing expires or cascades automatically. Referenced objects are
          refused until dependents are explicitly handled.
        </p>
        <label>
          Object kind
          <select
            value={objectKind}
            onChange={(event) =>
              setObjectKind(event.target.value as GovernanceObjectKind)
            }
          >
            <option value="report">Report</option>
            <option value="dataset">Dataset</option>
            <option value="lineage">Lineage</option>
          </select>
        </label>
        <label>
          Object ID
          <input
            value={objectID}
            onChange={(event) => setObjectID(event.target.value)}
          />
        </label>
        <label>
          Lifecycle state
          <select
            value={retentionState}
            onChange={(event) =>
              setRetentionState(event.target.value as RetentionState)
            }
          >
            <option value="active">Active</option>
            <option value="archived">Archived</option>
            <option value="expired">Expired</option>
            <option value="damaged">Damaged</option>
          </select>
        </label>
        <button
          disabled={!objectID}
          onClick={() =>
            void action(async () => {
              await api.setRetention(
                objectKind,
                objectID,
                retentionState,
                defaultRetentionPolicy,
                "explicit desktop lifecycle change",
              );
              await refresh();
              setMessage("Retention event appended.");
            })
          }
        >
          Record lifecycle state
        </button>
        <button
          disabled={!objectID}
          onClick={() =>
            void action(async () => {
              const result = await api.deleteObject(objectKind, objectID);
              if (result.cancelled) {
                setMessage("Deletion cancelled.");
                return;
              }
              await refresh();
              setMessage("Governed object deleted and tombstoned.");
            })
          }
        >
          Delete governed object
        </button>
      </section>

      {audit && (
        <section aria-label="Governance audit">
          <h3>Storage accounting and reference audit</h3>
          <p>
            {audit.usage.reports} reports · {audit.usage.datasets} datasets ·{" "}
            {audit.usage.lineages} lineages · {audit.usage.total_bytes} total
            bytes
          </p>
          <p>
            {audit.usage.unmanaged_objects} unmanaged ·{" "}
            {audit.usage.damaged_objects} damaged ·{" "}
            {audit.usage.missing_references} missing references ·{" "}
            {audit.usage.temporary_files} temporary files
          </p>
          {audit.issues.length === 0 ? (
            <p>No audit issue was observed in the bounded inventory.</p>
          ) : (
            <ul>
              {audit.issues.map((issue, index) => (
                <li key={`${issue.code}-${issue.object_id}-${index}`}>
                  {issue.code}: {issue.detail}
                </li>
              ))}
            </ul>
          )}
          <p>{audit.references.length} tracked evidence references.</p>
        </section>
      )}
    </section>
  );
}
