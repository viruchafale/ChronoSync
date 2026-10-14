# Codex integration — Phase 1

## Verified interface

Inspected installed `codex-cli 0.163.0`, `codex exec --help`, `codex app-server --help` and `codex login status` on 2026-10-21. Login reported ChatGPT without reading credential contents. [Official exec documentation](https://developers.openai.com/codex/noninteractive) establishes structured JSONL output; [app-server documentation](https://developers.openai.com/codex/app-server) describes a richer bidirectional interface. Exec is the narrower boundary for this single-run milestone; app-server is not used, and no terminal presentation scraping or internal API is involved.

Actual argv: `exec --json --sandbox <read-only|workspace-write> --ignore-user-config --ephemeral --color never -C <selected-root> -`. The prompt goes to stdin, never argv or the journal. The absolute executable is user-selected; it must be a trusted installation. A supervised `--version` probe requires codex-cli 0.163.0 or newer. This is a compatibility check, not executable authenticity verification. Only 0.163.0 has actual live evidence.

## Boundaries and event mapping

`internal/adapter.Adapter` exposes `Version(context,path)` and `Record(context,StartSpec,Sink) Result`. Result separates exit code, terminal-turn evidence, malformed count, bounded stderr and error. The recorder owns IDs, sequence, redaction, lifecycle, persistence and observation. Vendor parsing stays in `adapter/codex`.

| Upstream record                 | Internal event                      | Captured fields                                                     |
| ------------------------------- | ----------------------------------- | ------------------------------------------------------------------- |
| thread.started                  | agent.thread                        | thread ID                                                           |
| turn.started/completed/failed   | agent.turn_started/completed/failed | type; failure message when supplied                                 |
| item.* command_execution        | command.started/updated/completed   | item ID, command, aggregate output, exit code when supplied         |
| item.* agent_message            | agent.message                       | emitted text                                                        |
| item.* reasoning                | agent.reasoning_summary             | emitted summary only                                                |
| item.* mcp_tool_call/web_search | tool.started/updated/completed      | item/type/lifecycle; raw arguments and results deliberately omitted |
| item.* file_change              | file.reported                       | vendor-reported paths and operations; source=codex                  |
| error                           | adapter.error                       | bounded/redacted message                                            |
| future/unsupported types        | adapter.unknown                     | type and item ID where available; raw JSON omitted                  |
| malformed line                  | adapter.warning                     | omission notice without raw input                                   |

ChronoSync additionally creates execution.state, execution.diagnostic, execution.warning and workspace.changed from its own process/store/scanner observations. Vendor file reports and independently observed metadata changes have distinct source labels. Unknown records do not automatically fail a run; malformed records prevent a successful terminal outcome. Empty/partial streams without a completed turn fail even if exit code is zero. An upstream error/failed turn cannot be overwritten by a later completed turn.

## Supervision and persistence

macOS/Linux use an owned process group, concurrent pipe drains, 128 KiB line cap and 8 MiB combined output budget. Cancellation/deadline sends SIGTERM then SIGKILL after 500 ms; the remaining ordinary process group is killed after leader exit. Pipe draining is bounded after exit. Prompt pipe is closed. Store failures stop capture and cause an interrupted result if metadata remains writable. Final workspace inspection adds at most a 5-second context deadline; filesystem operations can still stall at the OS level.

The single recorder assigns local decimal sequences and UTC observation times. Event v2 has no Lamport/causal guarantee. Staging JSONL records are checksummed/fsynced; see DATA_MODEL.md. EOF from the host cancels and waits. Hard process death can bypass graceful cleanup; restart labels unfinished runs interrupted and never resumes a model automatically.

## Environment and privacy

Only HOME, USER, TMPDIR, LANG, LC_ALL and CODEX_HOME are inherited; PATH is an explicit standard installation list and NO_COLOR=1. API keys, arbitrary proxy/shell variables and unrelated environment are not forwarded. Codex reads its own authentication. `--ignore-user-config` does not prove all project rules/configuration/hooks are absent; use trusted projects and inspect applicable configuration. No dangerous bypass flags or hook-trust bypass are used. CLI login is not proof of model access.

Secrets in common authorization/API-key/password/token/private-key formats are redacted before persistence, with field limits and explicit truncation markers for text. Unknown raw payloads are omitted. Redaction cannot guarantee secrecy, nor can it preserve replay fidelity. MCP bodies, provider request/response internals, hidden reasoning, full model context and every file mutation are not captured.

## Evidence and remaining gates

The read-only live harness succeeded with run `801419b60519f32eee535f9253041c6d`: 12 persisted events included a thread, turn, actual command start/completion/output and a terminal completed state. Engine restart preserved that history. The first sandboxed attempt (`809cef4fd2f86f18b3a5e76435715129`) correctly failed and retained diagnostics when Codex initialization returned Operation not permitted. An approved retry outside the outer sandbox retained Codex's read-only policy and passed.

Synthetic adapter/subprocess tests cover failures, missing terminal, malformed/unknown records, cancellation, timeouts and changes; they are not live evidence. The fixtures under `engine/internal/adapter/codex/testdata` are clearly labeled synthetic. Live write-fixture outcome and final validation gates are tracked in IMPLEMENTATION_STATUS.md. Native GUI automation remains unverified after a tool timeout; Windows recording is intentionally blocked.

Live write-fixture run a7a6cc2ed9076936e09f694f4b86e28d also passed: 15 events, actual fixture output, one observed generated.txt creation and reopened completed history. Read-only run had zero observed changes. These journals remain in temporary local directories and are not committed fixtures.
