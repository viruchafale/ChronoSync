import { useEffect, useRef, useState } from "react";
import { SectionNav, Badge, CopyButton } from "./Workbench";
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
  const [pending, setPending] = useState<{
    id: string;
    branch: boolean;
  } | null>(null);
  const confirmation = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    if (pending) confirmation.current?.showModal();
    else confirmation.current?.close();
  }, [pending]);
  const generation = useRef(0);
  useEffect(() => {
    const token = ++generation.current;
    setBusy("Loading snapshots…");
    setConsent(false);
    setPending(null);
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
      <SectionNav
        items={[
          ["snapshot-capture", "Capture"],
          ["snapshot-history", "Snapshot history"],
          ["branch-history", "Materialized workspaces"],
        ]}
      />
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
      <label id="snapshot-capture">
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
      <ul id="snapshot-history" className="artifact-library">
        {snapshots.map((s) => (
          <li key={s.id}>
            <p>
              <code>{s.id}</code> · <Badge value={s.state} /> · {s.entries}{" "}
              entries · {s.omitted} omitted · {s.bytes} bytes{" "}
              {s.branch_id ? `· branch ${s.branch_id}` : ""} {s.diagnostic}
            </p>
            <CopyButton value={s.id} label="Copy snapshot ID" />
            <button
              aria-label={`Verify ${s.id}`}
              title={`Verify ${s.id}`}
              disabled={!!busy || s.state !== "committed"}
              onClick={() =>
                void perform("Verifying snapshot…", () => api.validate(s.id))
              }
            >
              Verify
            </button>
            <button
              aria-label={`Restore ${s.id}`}
              title={`Restore ${s.id}`}
              disabled={!!busy || !consent || s.state !== "committed"}
              onClick={() => setPending({ id: s.id, branch: false })}
            >
              Restore
            </button>
            <button
              aria-label={`Branch ${s.id}`}
              title={`Branch ${s.id}`}
              disabled={!!busy || !consent || s.state !== "committed"}
              onClick={() => setPending({ id: s.id, branch: true })}
            >
              Branch
            </button>
          </li>
        ))}
      </ul>
      <dialog
        ref={confirmation}
        className="command-dialog"
        aria-labelledby="snapshot-confirm-title"
        onCancel={() => setPending(null)}
      >
        <h2 id="snapshot-confirm-title">
          {pending?.branch ? "Create branch workspace" : "Restore snapshot"}
        </h2>
        <p>
          Create a separate managed directory from snapshot{" "}
          <code>{pending?.id}</code>. Source files stay unchanged; no commands
          will execute.
        </p>
        <button autoFocus onClick={() => setPending(null)}>
          Cancel
        </button>
        <button
          className="primary"
          disabled={!pending || !consent || !!busy}
          onClick={() => {
            if (!pending || !consent) return;
            const action = pending;
            setPending(null);
            void perform(
              action.branch ? "Preparing branch…" : "Restoring workspace…",
              () => api.materialize(action.id, action.branch, consent),
            );
          }}
        >
          Confirm workspace creation
        </button>
      </dialog>
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
      <h4 id="branch-history">Materialized workspaces</h4>
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
