import { useEffect, useRef, useState } from "react";
import type { Run } from "../../../contracts/recorder";
import { isActive } from "../../../contracts/recorder";
import type {
  SnapshotAPI,
  Snapshot,
  Branch,
} from "../../../contracts/snapshot";
import { snapshotAPI } from "./api";
export function Snapshots({
  run,
  api = snapshotAPI,
}: {
  run: Run;
  api?: SnapshotAPI;
}) {
  const [snapshots, setSnapshots] = useState<Snapshot[]>([]);
  const [branches, setBranches] = useState<Branch[]>([]);
  const [next, setNext] = useState("");
  const [branchNext, setBranchNext] = useState("");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");
  const [consent, setConsent] = useState(false);
  const [excludes, setExcludes] = useState("");
  const [branchID, setBranchID] = useState("");
  const [notice, setNotice] = useState("");
  const [reload, setReload] = useState(0);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current;
    setBusy("Loading snapshots…");
    setConsent(false);
    setBranchID("");
    setSnapshots([]);
    setBranches([]);
    void Promise.all([api.list(run.id, ""), api.branches(run.id, "")])
      .then(([s, b]) => {
        if (token !== generation.current) return;
        setSnapshots(s.snapshots);
        setNext(s.next);
        setBranches(b.branches);
        setBranchNext(b.next);
      })
      .catch((e) => {
        if (token === generation.current) setError(String(e));
      })
      .finally(() => {
        if (token === generation.current) setBusy("");
      });
    return () => {
      generation.current++;
    };
  }, [run.id, api, reload]);
  async function perform(label: string, operation: () => Promise<unknown>) {
    const token = generation.current;
    setBusy(label);
    setError("");
    setNotice("");
    try {
      await operation();
      if (token === generation.current) {
        setNotice("Operation completed. Inspect the recorded status below.");
        setReload((v) => v + 1);
      }
    } catch (e) {
      if (token === generation.current) {
        setError(String(e));
        setReload((v) => v + 1);
      }
    } finally {
      if (token === generation.current) setBusy("");
    }
  }
  return (
    <section aria-label="Workspace snapshots">
      <h3>Workspace snapshots and branches</h3>
      <p>
        Capture reads current project bytes after recording has stopped; it does
        not reconstruct the workspace at an earlier event. Default exclusions
        omit credentials, Git metadata, dependencies and build outputs. All
        symlinks are omitted. Only portable ASCII paths are currently supported.
      </p>
      <p>
        Source bytes remain private on this computer without encryption or
        complete secret detection. Restore creates a separate managed directory;
        no commands execute and the original workspace is not overwritten.
      </p>
      {error && <p role="alert">{error}</p>}
      {busy && <p aria-live="polite">{busy}</p>}
      {notice && <p>{notice}</p>}
      <label>
        Snapshot source
        <select
          disabled={!!busy}
          value={branchID}
          onChange={(e) => {
            setBranchID(e.target.value);
            setConsent(false);
          }}
        >
          <option value="">Original project: {run.workspace}</option>
          {branches
            .filter((b) => b.state === "ready")
            .map((b) => (
              <option key={b.id} value={b.id}>
                {b.kind} {b.id}: {b.workspace}
              </option>
            ))}
        </select>
      </label>
      <label>
        Snapshot exclusions (relative paths, one per line)
        <textarea
          value={excludes}
          onChange={(e) => {
            setExcludes(e.target.value);
            setConsent(false);
          }}
        />
      </label>
      <label>
        <input
          type="checkbox"
          checked={consent}
          onChange={(e) => setConsent(e.target.checked)}
        />
        I authorize storing source bytes with these exclusions and creating
        separate managed workspaces.
      </label>
      <button
        disabled={!!busy || !consent || isActive(run.state)}
        onClick={() =>
          void perform("Capturing snapshot…", () =>
            api.capture({
              run_id: run.id,
              branch_id: branchID,
              excludes: excludes.split("\n").filter(Boolean),
              consent,
            }),
          )
        }
      >
        Capture snapshot
      </button>
      <button disabled={!!busy} onClick={() => setReload((v) => v + 1)}>
        Refresh snapshots
      </button>
      {!busy && snapshots.length === 0 && <p>No snapshots on this page.</p>}
      <ul>
        {snapshots.map((s) => (
          <li key={s.id}>
            <p>
              {s.id} · {s.state} · {s.entries} entries · {s.omitted} omitted ·{" "}
              {s.bytes} bytes {s.branch_id ? `· branch ${s.branch_id}` : ""}{" "}
              {s.diagnostic}
            </p>
            <button
              disabled={!!busy || s.state !== "committed"}
              onClick={() =>
                void perform("Verifying snapshot…", () => api.validate(s.id))
              }
            >
              Verify {s.id}
            </button>
            <button
              disabled={!!busy || !consent || s.state !== "committed"}
              onClick={() =>
                void perform("Restoring workspace…", () =>
                  api.materialize(s.id, false, consent),
                )
              }
            >
              Restore {s.id}
            </button>
            <button
              disabled={!!busy || !consent || s.state !== "committed"}
              onClick={() =>
                void perform("Preparing branch…", () =>
                  api.materialize(s.id, true, consent),
                )
              }
            >
              Branch {s.id}
            </button>
          </li>
        ))}
      </ul>
      {next && (
        <button
          disabled={!!busy}
          onClick={() => {
            setBusy("Loading snapshots…");
            void api
              .list(run.id, next)
              .then((p) => {
                setSnapshots(p.snapshots);
                setNext(p.next);
              })
              .catch((e) => setError(String(e)))
              .finally(() => setBusy(""));
          }}
        >
          More snapshots
        </button>
      )}
      <h4>Materialized workspaces</h4>
      <ul>
        {branches.map((b) => (
          <li key={b.id}>
            {b.kind} {b.id} · {b.state} · source {b.snapshot_id} · parent{" "}
            {b.parent_id || "none"}
            <p>{b.workspace}</p>
            {b.diagnostic && <p>{b.diagnostic}</p>}
          </li>
        ))}
      </ul>
      {branchNext && (
        <button
          disabled={!!busy}
          onClick={() => {
            setBusy("Loading branches…");
            void api
              .branches(run.id, branchNext)
              .then((p) => {
                setBranches(p.branches);
                setBranchNext(p.next);
              })
              .catch((e) => setError(String(e)))
              .finally(() => setBusy(""));
          }}
        >
          More branches
        </button>
      )}
      <button
        disabled={!!busy}
        onClick={() => {
          setBusy("Inspecting object inventory…");
          void api
            .inventory()
            .then((v) =>
              setNotice(
                `${v.objects} objects; ${v.unreferenced} unreferenced; ${v.temporary} temporary. All retained; no garbage collection performed.`,
              ),
            )
            .catch((e) => setError(String(e)))
            .finally(() => setBusy(""));
        }}
      >
        Inspect retained objects
      </button>
    </section>
  );
}
