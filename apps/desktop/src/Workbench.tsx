import { useEffect, useRef, useState, type ReactNode } from "react";

export const destinations = [
  [
    "overview",
    "Overview",
    "Workspace",
    "M3 3h7v7H3z M14 3h7v7h-7z M3 14h7v7H3z M14 14h7v7h-7z",
  ],
  ["capture", "New capture", "Workspace", "M12 5v14 M5 12h14"],
  [
    "timeline",
    "Execution timeline",
    "Workspace",
    "M6 3v18 M10 6h10 M10 12h7 M10 18h10",
  ],
  [
    "snapshots",
    "Snapshots & branches",
    "Workspace",
    "M6 3v12a4 4 0 0 0 4 4h8 M6 8h8a4 4 0 0 0 4-4 M3 3h6 M15 3h6 M15 19h6",
  ],
  ["replay", "Replay & regression", "Evaluation", "M5 4l15 8-15 8z"],
  ["analysis", "Compare & analyze", "Evaluation", "M4 4v16h6V4z M14 4v16h6V4z"],
  [
    "governance",
    "Benchmarks & retention",
    "Governance",
    "M4 5h16v14H4z M4 10h16 M9 10v9",
  ],
  ["exchange", "Exchange & trust", "Governance", "M3 8h17l-4-4 M21 16H4l4 4"],
  [
    "settings",
    "Security & settings",
    "System",
    "M12 3l8 4v6c0 4-8 8-8 8s-8-4-8-8V7z M8 12l3 3 5-6",
  ],
] as const;
export type Page = (typeof destinations)[number][0];

export function SectionNav({
  items,
}: {
  items: readonly (readonly [string, string])[];
}) {
  return (
    <div className="section-nav" aria-label="Page sections">
      {items.map(([id, label]) => (
        <a key={id} href={`#${id}`}>
          {label}
        </a>
      ))}
    </div>
  );
}

export function Badge({ value }: { value: string }) {
  return (
    <span className="badge" data-state={value}>
      {value}
    </span>
  );
}

export function Icon({ path }: { path: string }) {
  return (
    <svg
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
    >
      <path d={path} />
    </svg>
  );
}

export function CopyButton({
  value,
  label = "Copy",
}: {
  value: string;
  label?: string;
}) {
  const [feedback, setFeedback] = useState("");
  return (
    <span className="copy-control">
      <button
        title={label}
        onClick={() => {
          if (!navigator.clipboard) {
            setFeedback("Clipboard unavailable");
            return;
          }
          void navigator.clipboard.writeText(value).then(
            () => setFeedback("Copied"),
            () => setFeedback("Clipboard unavailable"),
          );
        }}
      >
        {label}
      </button>
      <span aria-live="polite">{feedback}</span>
    </span>
  );
}

export function Workbench({
  page,
  navigate,
  connection,
  context,
  children,
}: {
  page: Page;
  navigate: (page: Page) => void;
  connection: string;
  context?: string | undefined;
  children: ReactNode;
}) {
  const [compact, setCompact] = useState(() => {
    try {
      return localStorage.getItem("chronosync.sidebar") === "compact";
    } catch {
      return false;
    }
  });
  const [palette, setPalette] = useState(false);
  const [query, setQuery] = useState("");
  const dialog = useRef<HTMLDialogElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const input = useRef<HTMLInputElement>(null);
  const title = destinations.find((item) => item[0] === page)![1];
  useEffect(() => {
    document.documentElement.scrollTop = 0;
    document.documentElement.scrollLeft = 0;
  }, [page]);
  useEffect(() => {
    const shortcut = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
        event.preventDefault();
        setPalette((value) => !value);
      }
    };
    window.addEventListener("keydown", shortcut);
    return () => window.removeEventListener("keydown", shortcut);
  }, []);
  useEffect(() => {
    if (palette) {
      dialog.current?.showModal();
      input.current?.focus();
    } else if (dialog.current?.open) {
      dialog.current.close();
      trigger.current?.focus();
    }
  }, [palette]);
  return (
    <div className={`workbench ${compact ? "compact" : ""}`}>
      <a href="#workspace-content" className="skip-link">
        Skip to workspace
      </a>
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">
            C<span>↗</span>
          </span>
          <span className="nav-label">
            ChronoSync<small>ENGINEERING WORKBENCH</small>
          </span>
        </div>
        <nav aria-label="Workspace navigation">
          {destinations.map(([id, label, group, path], index) => (
            <div key={id}>
              {(index === 0 || destinations[index - 1]![2] !== group) && (
                <div className="nav-group nav-label">{group}</div>
              )}
              <button
                title={label}
                aria-label={label}
                aria-current={page === id ? "page" : undefined}
                onClick={() => navigate(id)}
              >
                <Icon path={path} />
                <span className="nav-label">{label}</span>
              </button>
            </div>
          ))}
        </nav>
        <div className="sidebar-footer">
          <span className="nav-label">Local evidence. Clear provenance.</span>
          <button
            title={compact ? "Expand sidebar" : "Collapse sidebar"}
            aria-label={compact ? "Expand sidebar" : "Collapse sidebar"}
            onClick={() => {
              setCompact(!compact);
              try {
                localStorage.setItem(
                  "chronosync.sidebar",
                  !compact ? "compact" : "expanded",
                );
              } catch {
                /* Preferences are optional. */
              }
            }}
          >
            <Icon path="M4 4h16v16H4z M9 4v16" />
          </button>
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="breadcrumbs">
            <span>Workspace</span>
            <span>/</span>
            <strong>{title}</strong>
          </div>
          <button
            ref={trigger}
            className="command-trigger"
            onClick={() => {
              setQuery("");
              setPalette(true);
            }}
          >
            Search commands <kbd>⌘ / Ctrl K</kbd>
          </button>
        </header>
        <main id="workspace-content" tabIndex={-1}>
          <div className="page-heading">
            <div>
              <div className="eyebrow">LOCAL-FIRST / OBSERVABLE EVIDENCE</div>
              <h1>{title}</h1>
            </div>
            {context && (
              <code className="context-path" title={context}>
                {context}
              </code>
            )}
          </div>
          {children}
        </main>
        <footer className="statusbar">
          <span
            className={`connection ${connection === "Engine responded" ? "connected" : ""}`}
          >
            {connection}
          </span>
          <span>Private stdio · Evidence stays local</span>
        </footer>
      </div>
      <dialog
        ref={dialog}
        aria-labelledby="command-title"
        className="command-dialog"
        onCancel={() => setPalette(false)}
        onClick={(event) => {
          if (event.target === dialog.current) setPalette(false);
        }}
      >
        <div className="dialog-heading">
          <h2 id="command-title">Go to workspace</h2>
          <button
            onClick={() => setPalette(false)}
            aria-label="Close command palette"
          >
            Esc
          </button>
        </div>
        <input
          ref={input}
          aria-label="Search commands"
          placeholder="Find a workspace…"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <div className="command-results">
          {destinations
            .filter((item) =>
              item[1].toLowerCase().includes(query.toLowerCase()),
            )
            .map(([id, label, group, path]) => (
              <button
                key={id}
                onClick={() => {
                  navigate(id);
                  setPalette(false);
                }}
              >
                <Icon path={path} />
                <span>{label}</span>
                <small>{group}</small>
              </button>
            ))}
          {!destinations.some((item) =>
            item[1].toLowerCase().includes(query.toLowerCase()),
          ) && <p>No matching workspaces.</p>}
        </div>
      </dialog>
    </div>
  );
}
