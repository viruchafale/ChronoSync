import { Snapshots } from "./Snapshots";
import { Workbench, CopyButton, Badge, type Page } from "./Workbench";
import { Replay } from "./Replay";
import { Analysis } from "./Analysis";
import { Governance } from "./Governance";
import { Exchange } from "./Exchange";
import { useEffect, useRef, useState } from "react";
import {
  analysisAPI as nativeAnalysisAPI,
  api as nativeAPI,
  governanceAPI as nativeGovernanceAPI,
  exchangeAPI as nativeExchangeAPI,
  replayAPI as nativeReplayAPI,
} from "./api";
import type { ReplayAPI } from "../../../contracts/replay";
import type { AnalysisAPI } from "../../../contracts/analysis";
import type { GovernanceAPI } from "../../../contracts/governance";
import type { ExchangeAPI } from "../../../contracts/exchange";
import type { SnapshotAPI } from "../../../contracts/snapshot";
import {
  isActive,
  type RecorderAPI,
  type Run,
  type RecordedEvent,
  type EventSummary,
  type StartSpec,
} from "../../../contracts/recorder";
export function App({
  api = nativeAPI,
  replay = nativeReplayAPI,
  analysis = nativeAnalysisAPI,
  governance = nativeGovernanceAPI,
  exchange = nativeExchangeAPI,
  snapshots,
}: {
  api?: RecorderAPI;
  replay?: ReplayAPI;
  analysis?: AnalysisAPI;
  governance?: GovernanceAPI;
  exchange?: ExchangeAPI;
  snapshots?: SnapshotAPI;
}) {
  const [page, setPage] = useState<Page>("overview");
  const [connection, setConnection] = useState("Connecting to engine…");
  const [runSearch, setRunSearch] = useState("");
  const [timelineWidth, setTimelineWidth] = useState(38);
  const [runs, setRuns] = useState<Run[]>([]);
  const [selected, setSelected] = useState<Run | null>(null);
  const [events, setEvents] = useState<EventSummary[]>([]);
  const [detail, setDetail] = useState<RecordedEvent | null>(null);
  const [warnings, setWarnings] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const [cancelling, setCancelling] = useState(false);
  const [spec, setSpec] = useState<StartSpec>({
    workspace: "",
    prompt: "",
    codex_path: "",
    sandbox: "read-only",
    timeout_seconds: 120,
    consent: false,
    excludes: [],
  });
  const [cursor, setCursor] = useState("");
  const [previous, setPrevious] = useState<string[]>([]);
  const [next, setNext] = useState("");
  const [runNext, setRunNext] = useState("");
  const [loading, setLoading] = useState(false);
  const [count, setCount] = useState(0);
  const [filters, setFilters] = useState({
    kind: "",
    search: "",
    min: "",
    max: "",
  });
  const [draft, setDraft] = useState(filters);
  const [refresh, setRefresh] = useState(0);
  const detailGeneration = useRef(0);
  const generation = useRef(0);
  const selectedID = selected?.id;
  useEffect(() => {
    let disposed = false;
    api
      .queryRuns("")
      .then((r) => {
        if (!disposed) {
          setRuns(r.runs);
          setRunNext(r.next);
          setConnection("Engine responded");
        }
      })
      .catch((e) => {
        if (!disposed) {
          setError(String(e));
          setConnection("Engine unavailable");
        }
      });
    return () => {
      disposed = true;
    };
  }, [api]);
  useEffect(() => {
    if (!selectedID) return;
    const token = ++generation.current;
    let disposed = false;
    detailGeneration.current++;
    setLoading(true);
    let timer: ReturnType<typeof setTimeout>;
    setEvents([]);
    setDetail(null);
    setWarnings([]);
    setCancelling(false);
    async function poll() {
      try {
        const page = await api.query({
          id: selectedID!,
          cursor,
          ...filters,
          limit: 100,
        });
        if (disposed || token !== generation.current) return;
        setSelected(page.run);
        setConnection("Engine responded");
        setLoading(false);
        setWarnings(
          page.run.history && page.run.history !== "complete"
            ? [
                page.run.history +
                  ": " +
                  (page.run.diagnostic || "History is incomplete"),
              ]
            : [],
        );
        setNext(page.next);
        setCount(page.count);
        setEvents(page.events);
        setRuns((old) => [
          page.run,
          ...old.filter((r) => r.id !== page.run.id),
        ]);
        if (isActive(page.run.state))
          timer = setTimeout(() => void poll(), 1000);
      } catch (e) {
        if (!disposed) {
          setError(String(e));
          setConnection("Engine request failed");
          setLoading(false);
          setCancelling(false);
        }
      }
    }
    void poll();
    return () => {
      disposed = true;
      clearTimeout(timer);
    };
  }, [api, selectedID, cursor, filters, refresh]);
  const active =
    runs.some((r) => isActive(r.state)) ||
    (selected !== null && isActive(selected.state));
  async function start(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const run = await api.start(spec);
      setRuns((old) => [run, ...old]);
      setCursor("");
      setPrevious([]);
      setSelected(run);
      setPage("timeline");
      setSpec((old) => ({ ...old, prompt: "", consent: false }));
    } catch (e) {
      setError(String(e));
    } finally {
      setBusy(false);
    }
  }
  async function cancel() {
    if (!selected) return;
    setCancelling(true);
    setError("");
    try {
      await api.cancel(selected.id);
    } catch (e) {
      setError(String(e));
      setCancelling(false);
    }
  }
  return (
    <Workbench
      page={page}
      navigate={setPage}
      connection={connection}
      context={selected?.workspace}
    >
      {page === "overview" && (
        <div className="overview-intro">
          <div className="eyebrow">YOUR DEBUGGING STARTING POINT</div>
          <h2>Every execution leaves a trail.</h2>
          <p>
            Pick up a recorded execution, inspect its events, and follow the
            evidence.
          </p>
          <button className="primary" onClick={() => setPage("capture")}>
            New capture <span aria-hidden="true">↗</span>
          </button>
        </div>
      )}
      <div hidden={page !== "capture"}>
        <p className="boundary-note">
          Runs use your Codex account and may contact the model provider.
          Recordings stay on this computer. Redaction is limited; captured
          output can contain private data.
        </p>
        <section
          className="panel capture-panel"
          aria-labelledby="start-heading"
        >
          <h2 id="start-heading">Start an execution</h2>
          <form onSubmit={(e) => void start(e)}>
            <label>
              Project directory (absolute path)
              <input
                required
                value={spec.workspace}
                onChange={(e) =>
                  setSpec({
                    ...spec,
                    workspace: e.target.value,
                    consent: false,
                  })
                }
              />
            </label>
            <label>
              Codex executable (absolute path)
              <input
                required
                placeholder="/opt/homebrew/bin/codex"
                value={spec.codex_path}
                onChange={(e) =>
                  setSpec({
                    ...spec,
                    codex_path: e.target.value,
                    consent: false,
                  })
                }
              />
            </label>
            <label>
              Task
              <textarea
                required
                maxLength={32768}
                value={spec.prompt}
                onChange={(e) =>
                  setSpec({ ...spec, prompt: e.target.value, consent: false })
                }
              />
            </label>
            <label>
              Codex sandbox policy
              <select
                value={spec.sandbox}
                onChange={(e) =>
                  setSpec({
                    ...spec,
                    sandbox: e.target.value as StartSpec["sandbox"],
                    consent: false,
                  })
                }
              >
                <option value="read-only">Read only</option>
                <option value="workspace-write">
                  Workspace write — may modify project files
                </option>
              </select>
            </label>
            <label>
              Timeout (seconds)
              <input
                type="number"
                min="1"
                max="3600"
                required
                value={spec.timeout_seconds}
                onChange={(e) =>
                  setSpec({
                    ...spec,
                    timeout_seconds: Number(e.target.value),
                    consent: false,
                  })
                }
              />
            </label>
            <label>
              Additional excluded paths (one relative path per line)
              <textarea
                value={spec.excludes.join("\n")}
                onChange={(e) =>
                  setSpec({
                    ...spec,
                    excludes: e.target.value.split("\n").filter(Boolean),
                    consent: false,
                  })
                }
              />
            </label>
            <p>
              Workspace-write changes the selected project directly; no branch
              or restore is provided. File observation skips sensitive and
              generated paths and does not restrict Codex access. Process groups
              are execution controls, not a security sandbox. Windows recording
              is currently unavailable.
            </p>
            <label className="consent">
              <input
                type="checkbox"
                checked={spec.consent}
                onChange={(e) =>
                  setSpec({ ...spec, consent: e.target.checked })
                }
              />
              I authorize this Codex executable and task in this workspace under
              the selected policy, and local recording of observable output.
            </label>
            <button disabled={busy || active || !spec.consent} type="submit">
              {busy ? "Starting…" : "Start recording"}
            </button>
          </form>
        </section>
      </div>
      {error && ["overview", "capture", "timeline"].includes(page) && (
        <p role="alert">{error}</p>
      )}
      <div hidden={page !== "overview"}>
        <section className="panel runs-panel" aria-labelledby="runs-heading">
          <h2 id="runs-heading">Recorded executions</h2>
          <label className="run-search">
            Filter loaded executions
            <input
              placeholder="Workspace, status, or execution ID…"
              value={runSearch}
              onChange={(event) => setRunSearch(event.target.value)}
            />
          </label>
          <button
            onClick={() => {
              setError("");
              void api
                .queryRuns("")
                .then((page) => {
                  setRuns(page.runs);
                  setRunNext(page.next);
                  setConnection("Engine responded");
                })
                .catch((e) => setError(String(e)));
            }}
          >
            Refresh executions
          </button>
          {runNext && (
            <button
              onClick={() =>
                void api
                  .queryRuns(runNext)
                  .then((page) => {
                    setRuns(page.runs);
                    setRunNext(page.next);
                  })
                  .catch((e) => setError(String(e)))
              }
            >
              Older executions
            </button>
          )}
          {runs.length === 0 && (
            <div className="empty-state">
              <h3>Your next investigation starts here</h3>
              <p>No recorded executions loaded.</p>
              <p>
                Create a capture in the desktop app, or refresh to load local
                history.
              </p>
            </div>
          )}
          {runs.length > 0 && (
            <div className="list-caption">
              EXECUTION / STATUS / WORKSPACE <span>{runs.length} loaded</span>
            </div>
          )}
          <ul className="run-list">
            {runs
              .filter((run) =>
                `${run.workspace} ${run.state} ${run.id}`
                  .toLowerCase()
                  .includes(runSearch.toLowerCase()),
              )
              .map((run) => (
                <li key={run.id}>
                  <button
                    onClick={() => {
                      setError("");
                      setCursor("");
                      setPrevious([]);
                      setSelected(run);
                      setPage("timeline");
                    }}
                    aria-pressed={run.id === selectedID}
                  >
                    <time>{run.created_at}</time> · <Badge value={run.state} />{" "}
                    · <span>{run.workspace}</span>
                  </button>
                </li>
              ))}
          </ul>
          {runs.length > 0 &&
            !runs.some((run) =>
              `${run.workspace} ${run.state} ${run.id}`
                .toLowerCase()
                .includes(runSearch.toLowerCase()),
            ) && <p>No loaded executions match this filter.</p>}
        </section>
      </div>
      <div hidden={page !== "replay"} className="feature-page">
        <Replay api={replay} run={selected} />
      </div>
      <div hidden={page !== "analysis"} className="feature-page">
        <Analysis api={analysis} runs={runs} />
      </div>
      <div hidden={page !== "governance"} className="feature-page">
        <Governance api={governance} />
      </div>
      <div hidden={page !== "exchange"} className="feature-page">
        <Exchange
          api={exchange}
          evidence={{
            listDatasets: governance.list,
            getDataset: governance.get,
          }}
        />
      </div>
      {(page === "timeline" || page === "snapshots") && !selected && (
        <section className="empty-state panel">
          <h2>Select an execution to inspect</h2>
          <p>
            Timeline events and workspace snapshots belong to a recorded
            execution.
          </p>
          <button onClick={() => setPage("overview")}>Browse executions</button>
        </section>
      )}
      {page === "snapshots" && selected && (
        <div className="feature-page">
          <Snapshots
            key={selected.id}
            run={selected}
            {...(snapshots ? { api: snapshots } : {})}
          />
        </div>
      )}
      {selected && page === "timeline" && (
        <section className="timeline-panel" aria-labelledby="timeline-heading">
          <h2 id="timeline-heading">Execution timeline</h2>
          <p role="status">
            Status: {selected.state}
            {selected.reason ? ` — ${selected.reason}` : ""}
          </p>
          <p>
            Run {selected.id} · {selected.adapter_version} · {selected.sandbox}
            <CopyButton value={selected.id} label="Copy run ID" />
          </p>
          {isActive(selected.state) && (
            <button disabled={cancelling} onClick={() => void cancel()}>
              {cancelling ? "Cancellation requested…" : "Cancel execution"}
            </button>
          )}
          {warnings.map((warning, i) => (
            <p role="alert" key={i}>
              {warning}
            </p>
          ))}
          <p>
            Started: {selected.started_at || "Not recorded"} · Ended:{" "}
            {selected.finished_at || "Not recorded"} · {count} recorded events
          </p>
          <form
            className="timeline-filters"
            onSubmit={(e) => {
              e.preventDefault();
              setError("");
              setCursor("");
              setPrevious([]);
              setFilters({ ...draft });
            }}
          >
            <label>
              Event type
              <input
                value={draft.kind}
                placeholder="e.g. command.completed"
                onChange={(e) => setDraft({ ...draft, kind: e.target.value })}
              />
            </label>
            <label>
              Search diagnostics
              <input
                value={draft.search}
                maxLength={256}
                onChange={(e) => setDraft({ ...draft, search: e.target.value })}
              />
            </label>
            <label>
              From sequence
              <input
                value={draft.min}
                inputMode="numeric"
                onChange={(e) => setDraft({ ...draft, min: e.target.value })}
              />
            </label>
            <label>
              Through sequence
              <input
                value={draft.max}
                inputMode="numeric"
                onChange={(e) => setDraft({ ...draft, max: e.target.value })}
              />
            </label>
            <button type="submit">Apply filters</button>
          </form>
          <p>
            Search covers the first 1024 bytes of message, reason and upstream
            type; command output is available in details.
          </p>
          <button
            onClick={() => {
              setCursor("");
              setPrevious([]);
              setRefresh((n) => n + 1);
            }}
          >
            Refresh timeline
          </button>
          <button
            disabled={loading || previous.length === 0}
            onClick={() => {
              setCursor(previous[previous.length - 1]!);
              setPrevious(previous.slice(0, -1));
            }}
          >
            Previous page
          </button>
          <button
            disabled={loading || !next}
            onClick={() => {
              setPrevious([...previous, cursor]);
              setCursor(next);
            }}
          >
            Next page
          </button>
          {loading && <p>Loading events…</p>}
          {!loading && events.length === 0 && (
            <p>
              No recorded events match this query. This does not establish that
              no activity occurred.
            </p>
          )}
          <label className="panel-width">
            Timeline width
            <input
              aria-label="Timeline width"
              type="range"
              min="28"
              max="60"
              value={timelineWidth}
              onChange={(event) => setTimelineWidth(Number(event.target.value))}
            />
          </label>
          <div className="debugger-split">
            <div
              className="event-navigation"
              style={{ width: `${timelineWidth}%` }}
            >
              <div className="list-caption">
                EVENT STREAM <span>{events.length} on page</span>
              </div>
              <ol className="event-list">
                {events.map((event) => (
                  <li key={event.event_id}>
                    <button
                      data-kind={event.kind}
                      aria-pressed={detail?.event_id === event.event_id}
                      onClick={() => {
                        const token = ++detailGeneration.current;
                        setDetail(null);
                        setError("");
                        void api
                          .detail(selected.id, event.event_id)
                          .then((value) => {
                            if (token === detailGeneration.current)
                              setDetail(value);
                          })
                          .catch((e) => {
                            if (token === detailGeneration.current) {
                              setError("Payload unavailable: " + String(e));
                              setRefresh((n) => n + 1);
                            }
                          });
                      }}
                    >
                      <span className="event-title">
                        #{event.sequence} {event.kind}
                      </span>
                      <span className="event-caption">
                        <time>{event.observed_at}</time>{" "}
                        {event.status && <Badge value={event.status} />}
                        {event.truncated ? " · Truncated payload" : ""}
                      </span>
                    </button>
                  </li>
                ))}
              </ol>
            </div>
            <div className="event-inspection">
              {!detail && (
                <div className="empty-state">
                  <h3>Inspect an event</h3>
                  <p>
                    Select a timeline row to load its recorded payload and
                    metadata.
                  </p>
                </div>
              )}
              {detail && (
                <section aria-label="Event details">
                  <h3>
                    Event #{detail.sequence}: {detail.kind}
                  </h3>
                  <div className="inspector-meta">
                    <span>
                      Event ID <code>{detail.event_id}</code>
                    </span>
                    <span>
                      Observed <code>{detail.observed_at}</code>
                    </span>
                    <span>
                      Adapter <code>{detail.adapter}</code>
                    </span>
                  </div>
                  <CopyButton value={detail.event_id} label="Copy event ID" />
                  <CopyButton
                    value={JSON.stringify(detail.payload, null, 2)}
                    label="Copy payload"
                  />
                  {Object.keys(detail.payload).length === 0 && (
                    <p>This event was recorded with an empty payload.</p>
                  )}
                  {detail.payload.truncations?.map((t) => (
                    <p key={t.field}>
                      Truncated {t.field}: original {t.original_bytes} bytes,
                      limit {t.limit_bytes} bytes.
                    </p>
                  ))}
                  <pre>{JSON.stringify(detail.payload, null, 2)}</pre>
                </section>
              )}
            </div>
          </div>
        </section>
      )}
      {page === "settings" && (
        <section className="panel security-panel">
          <h2>Security boundaries</h2>
          <p>
            Capabilities and limitations of this implementation. This page does
            not perform a signed-release assessment.
          </p>
          <dl>
            <dt>Execution policy</dt>
            <dd>
              Positive policy-controlled launch is unavailable. Recorder consent
              remains required.
            </dd>
            <dt>Replay & analysis</dt>
            <dd>
              Synthetic evaluation of preserved evidence. No historical command
              execution or provider fallback.
            </dd>
            <dt>Signing custody</dt>
            <dd>
              macOS login Keychain. Protected per-sign user presence is
              unavailable in the development build.
            </dd>
            <dt>Release acceptance</dt>
            <dd>
              Not assessed by the renderer. Developer ID, user-presence,
              notarization and Gatekeeper acceptance require the authorized
              release process.
            </dd>
            <dt>Stored evidence</dt>
            <dd>
              Local data is not encrypted by ChronoSync. Redaction is limited.
            </dd>
          </dl>
          <h3>Workspace preferences</h3>
          <p>
            Sidebar density is saved on this device. Use the sidebar toggle to
            change it. Navigation and capture fields are kept only in memory.
          </p>
        </section>
      )}
    </Workbench>
  );
}
