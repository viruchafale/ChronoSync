import { useState } from "react";
import { SectionNav, CopyButton, Badge } from "./Workbench";
import type { Run } from "../../../contracts/recorder";
import type {
  FixtureRecord,
  ReplayAPI,
  UnifiedPlan,
  UnifiedResult,
} from "../../../contracts/replay";

export function Replay({ api, run }: { api: ReplayAPI; run: Run | null }) {
  const [fixtures, setFixtures] = useState<FixtureRecord[]>([]);
  const [inspected, setInspected] = useState<FixtureRecord | null>(null);
  const [eligibility, setEligibility] = useState<FixtureRecord | null>(null);
  const [regressionID, setRegressionID] = useState("");
  const [transportID, setTransportID] = useState("");
  const [planText, setPlanText] = useState(
    JSON.stringify(
      {
        version: 1,
        timeout_millis: 5000,
        actuals: [],
        transport_requests: [],
      } satisfies UnifiedPlan,
      null,
      2,
    ),
  );
  const [result, setResult] = useState<UnifiedResult | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");

  const refresh = async () => {
    const values = await api.list();
    setFixtures(values);
    if (!regressionID)
      setRegressionID(
        values.find((item) => item.kind === "regression-v1")?.id || "",
      );
    if (!transportID)
      setTransportID(
        values.find((item) => item.kind === "synthetic-transport-v1")?.id || "",
      );
  };
  const action = async (operation: () => Promise<void>) => {
    setError("");
    setMessage("");
    try {
      await operation();
    } catch (value) {
      setError(String(value));
    }
  };

  return (
    <section aria-labelledby="replay-heading">
      <h2 id="replay-heading">Regression fixtures</h2>
      <SectionNav
        items={[
          ["fixture-library", "Fixture library"],
          ["evaluation-heading", "Controlled evaluation"],
        ]}
      />
      <p>
        Fixtures contain inert fingerprints and sanitized metadata. Controlled
        evaluation never runs historical commands or contacts a provider.
      </p>
      {error && <p role="alert">{error}</p>}
      {message && <p role="status">{message}</p>}
      <button onClick={() => void action(refresh)}>Refresh fixtures</button>
      <button
        onClick={() =>
          void action(async () => {
            const imported = await api.importFixture();
            if ("cancelled" in imported) return;
            setMessage(
              imported.duplicate
                ? "Fixture already stored."
                : "Fixture imported.",
            );
            await refresh();
          })
        }
      >
        Import fixture…
      </button>
      {run?.state === "completed" && (
        <>
          <button
            onClick={() =>
              void action(async () => {
                setEligibility(await api.eligibility(run.id));
                setMessage("Execution is eligible for a regression fixture.");
              })
            }
          >
            Check selected execution eligibility
          </button>
          <button
            disabled={
              !eligibility || eligibility.source_execution_id !== run.id
            }
            onClick={() =>
              void action(async () => {
                const created = await api.create(run.id);
                setMessage(
                  created.duplicate
                    ? "Fixture already stored."
                    : "Regression fixture created.",
                );
                await refresh();
              })
            }
          >
            Create regression fixture
          </button>
        </>
      )}
      <div id="fixture-library" className="artifact-library">
        {fixtures.length === 0 ? (
          <p>No stored fixtures loaded.</p>
        ) : (
          <ul>
            {fixtures.map((fixture) => (
              <li key={fixture.id}>
                <strong>{fixture.kind}</strong> · {fixture.interactions}{" "}
                interactions · {fixture.bytes} bytes
                <button
                  onClick={() =>
                    void action(async () =>
                      setInspected(await api.inspect(fixture.id)),
                    )
                  }
                >
                  Inspect
                </button>
                <button
                  onClick={() =>
                    void action(async () => {
                      const exported = await api.exportFixture(fixture.id);
                      if (exported.exported) setMessage("Fixture exported.");
                    })
                  }
                >
                  Export…
                </button>
                <button
                  onClick={() =>
                    void action(async () => {
                      const deleted = await api.deleteFixture(fixture.id);
                      if (deleted.deleted) {
                        setMessage("Fixture deleted.");
                        if (inspected?.id === fixture.id) setInspected(null);
                        await refresh();
                      }
                    })
                  }
                >
                  Delete…
                </button>
              </li>
            ))}
          </ul>
        )}
      </div>
      {inspected && (
        <section aria-label="Fixture inspection">
          <h3>Fixture inspection</h3>
          <p>
            <code>{inspected.id}</code> · integrity{" "}
            <Badge value={inspected.integrity} /> · format{" "}
            {inspected.format_version}
          </p>
          <CopyButton value={inspected.id} label="Copy fixture ID" />
          {inspected.warnings.map((warning) => (
            <p key={warning}>{warning}</p>
          ))}
          <pre>{JSON.stringify(inspected.details, null, 2)}</pre>
        </section>
      )}
      <section aria-labelledby="evaluation-heading">
        <h3 id="evaluation-heading">Controlled evaluation</h3>
        <label>
          Regression fixture
          <select
            value={regressionID}
            onChange={(event) => setRegressionID(event.target.value)}
          >
            <option value="">Select fixture</option>
            {fixtures
              .filter((item) => item.kind === "regression-v1")
              .map((item) => (
                <option key={item.id} value={item.id}>
                  {item.id}
                </option>
              ))}
          </select>
        </label>
        <label>
          Synthetic transport fixture
          <select
            value={transportID}
            onChange={(event) => setTransportID(event.target.value)}
          >
            <option value="">Select fixture</option>
            {fixtures
              .filter((item) => item.kind === "synthetic-transport-v1")
              .map((item) => (
                <option key={item.id} value={item.id}>
                  {item.id}
                </option>
              ))}
          </select>
        </label>
        <label>
          Controlled evaluation plan (versioned JSON)
          <textarea
            value={planText}
            onChange={(event) => setPlanText(event.target.value)}
          />
        </label>
        <button
          disabled={!regressionID || !transportID}
          onClick={() =>
            void action(async () => {
              const plan = JSON.parse(planText) as UnifiedPlan;
              setResult(await api.evaluate(regressionID, transportID, plan));
            })
          }
        >
          Run controlled evaluation
        </button>
        {result && (
          <section aria-label="Evaluation result">
            <p role="status">
              {result.passed ? "Passed" : "Failed"}: {result.summary}
            </p>
            <pre>{JSON.stringify(result, null, 2)}</pre>
          </section>
        )}
      </section>
    </section>
  );
}
