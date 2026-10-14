import { Snapshots } from "./Snapshots";
import { useEffect, useRef, useState } from "react";
import { api as nativeAPI } from "./api";
import {
  isActive,
  type RecorderAPI,
  type Run,
  type RecordedEvent,
  type EventSummary,
  type StartSpec,
} from "../../../contracts/recorder";
export function App({ api = nativeAPI }: { api?: RecorderAPI }) {
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
        }
      })
      .catch((e) => {
        if (!disposed) setError(String(e));
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
    <main>
      <header>
        <h1>ChronoSync</h1>
        <p>Codex history and workspace snapshots · Phase 3</p>
      </header>
      <p>
        Runs use your Codex account and may contact the model provider.
        Recordings stay on this computer. Redaction is limited; captured output
        can contain private data.
      </p>
      {error && <p role="alert">{error}</p>}
      <section aria-labelledby="start-heading">
        <h2 id="start-heading">Start an execution</h2>
        <form onSubmit={(e) => void start(e)}>
          <label>
            Project directory (absolute path)
            <input
              required
              value={spec.workspace}
              onChange={(e) =>
                setSpec({ ...spec, workspace: e.target.value, consent: false })
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
                setSpec({ ...spec, codex_path: e.target.value, consent: false })
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
            Workspace-write changes the selected project directly; no branch or
            restore is provided. File observation skips sensitive and generated
            paths and does not restrict Codex access. Process groups are
            execution controls, not a security sandbox. Windows recording is
            currently unavailable.
          </p>
          <label className="consent">
            <input
              type="checkbox"
              checked={spec.consent}
              onChange={(e) => setSpec({ ...spec, consent: e.target.checked })}
            />
            I authorize this Codex executable and task in this workspace under
            the selected policy, and local recording of observable output.
          </label>
          <button disabled={busy || active || !spec.consent} type="submit">
            {busy ? "Starting…" : "Start recording"}
          </button>
        </form>
      </section>
      <section aria-labelledby="runs-heading">
        <h2 id="runs-heading">Recorded executions</h2>
        <button
          onClick={() => {
            setError("");
            void api
              .queryRuns("")
              .then((page) => {
                setRuns(page.runs);
                setRunNext(page.next);
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
        {runs.length === 0 && <p>No recorded executions loaded.</p>}
        <ul>
          {runs.map((run) => (
            <li key={run.id}>
              <button
                onClick={() => {
                  setError("");
                  setCursor("");
                  setPrevious([]);
                  setSelected(run);
                }}
                aria-pressed={run.id === selectedID}
              >
                {run.created_at} · {run.state} · {run.workspace}
              </button>
            </li>
          ))}
        </ul>
      </section>
      {selected && (
        <section aria-labelledby="timeline-heading">
          <h2 id="timeline-heading">Execution timeline</h2>
          <Snapshots key={selected.id} run={selected} />
          <p role="status">
            Status: {selected.state}
            {selected.reason ? ` — ${selected.reason}` : ""}
          </p>
          <p>
            Run {selected.id} · {selected.adapter_version} · {selected.sandbox}
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
          <ol>
            {events.map((event) => (
              <li key={event.event_id}>
                <button
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
                  #{event.sequence} {event.kind} · {event.observed_at}{" "}
                  {event.status} {event.truncated ? "· Truncated payload" : ""}
                </button>
              </li>
            ))}
          </ol>
          {detail && (
            <section aria-label="Event details">
              <h3>
                Event #{detail.sequence}: {detail.kind}
              </h3>
              {Object.keys(detail.payload).length === 0 && (
                <p>This event was recorded with an empty payload.</p>
              )}
              {detail.payload.truncations?.map((t) => (
                <p key={t.field}>
                  Truncated {t.field}: original {t.original_bytes} bytes, limit{" "}
                  {t.limit_bytes} bytes.
                </p>
              ))}
              <pre>{JSON.stringify(detail.payload, null, 2)}</pre>
            </section>
          )}
        </section>
      )}
    </main>
  );
}
