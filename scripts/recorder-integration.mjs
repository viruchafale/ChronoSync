import { spawn, spawnSync } from "node:child_process";
import { setTimeout, clearTimeout } from "node:timers";
import { mkdtempSync, mkdirSync, writeFileSync, readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { createInterface } from "node:readline";
import assert from "node:assert/strict";
import Ajv from "ajv";
const live = process.argv.includes("--live");
const writeFixture = process.argv.includes("--write-fixture");
if (writeFixture && !live) throw new Error("--write-fixture requires --live");
if (live && !process.argv.includes("--consent"))
  throw new Error(
    "Live test requires --consent: a disposable workspace, provider access and local recording.",
  );
if (!["darwin", "linux"].includes(process.platform))
  throw new Error(
    "Recorder integration requires supported POSIX process groups.",
  );
const temporary = mkdtempSync(join(tmpdir(), "chronosync-recorder-"));
const workspace = join(temporary, "workspace");
mkdirSync(workspace, { mode: 0o700 });
writeFileSync(
  join(workspace, "README.txt"),
  "ChronoSync public test fixture.\n",
  { mode: 0o600 },
);
const store = join(temporary, "recordings");
const fixture = join(temporary, "synthetic-codex");
writeFileSync(
  fixture,
  `#!/bin/sh
if [ "$1" = "--version" ]; then echo 'codex-cli 0.163.0'; exit 0; fi
printf '%s\\n' '{"type":"thread.started","thread_id":"synthetic"}' '{"type":"turn.started"}'
printf 'changed by synthetic fixture\\n' > observed.txt
printf '%s\\n' '{"type":"item.completed","item":{"id":"one","type":"command_execution","command":"synthetic","aggregated_output":"fixture output","exit_code":0}}' '{"type":"future.event","secret":"not-preserved"}' '{"type":"turn.completed"}'
`,
  { mode: 0o700 },
);
if (live) {
  const git = spawnSync("git", ["init", "--quiet", workspace], {
    encoding: "utf8",
  });
  if (git.status !== 0) throw new Error(git.stderr);
}
const executable = resolve("bin/chronosync-engine");
const validate = new Ajv({ strict: true }).compile(
  JSON.parse(readFileSync("contracts/recorded-event.schema.json", "utf8")),
);
function connect() {
  const child = spawn(executable, ["serve", "--data-dir", store], {
    stdio: ["pipe", "pipe", "pipe"],
  });
  const pending = new Map();
  let next = 0;
  let stderr = "";
  child.stderr.on("data", (b) => {
    stderr = (stderr + b.toString()).slice(-4096);
  });
  createInterface({ input: child.stdout }).on("line", (line) => {
    const response = JSON.parse(line);
    const entry = pending.get(response.id);
    if (!entry) return;
    pending.delete(response.id);
    clearTimeout(entry.timer);
    if (response.error) entry.reject(new Error(response.error.message));
    else entry.resolve(response.result);
  });
  child.on("exit", () => {
    for (const entry of pending.values()) {
      clearTimeout(entry.timer);
      entry.reject(new Error("Engine exited: " + stderr));
    }
    pending.clear();
  });
  const request = (method, params = {}) =>
    new Promise((resolve, reject) => {
      const id = String(++next);
      const timer = setTimeout(() => {
        pending.delete(id);
        reject(new Error("IPC timeout"));
      }, 10000);
      pending.set(id, { resolve, reject, timer });
      child.stdin.write(
        JSON.stringify({ protocol_version: 1, id, method, params }) + "\n",
      );
    });
  const close = () =>
    new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        child.kill("SIGTERM");
        reject(new Error("engine EOF cleanup timed out"));
      }, 8000);
      child.once("exit", (code) => {
        clearTimeout(timer);
        if (code !== 0) reject(new Error("Engine failure: " + stderr));
        else resolve();
      });
      child.stdin.end();
    });
  return { request, close, child };
}
let engine = connect();
try {
  const hello = await engine.request("hello");
  assert.equal(hello.protocol_version, 1);
  const run = await engine.request("run.start", {
    workspace,
    prompt: writeFixture
      ? "Use a shell command to read README.txt, then create generated.txt containing exactly ChronoSync fixture. Do not change any other file or access any other directory."
      : "Use a shell command to read README.txt, then report its exact contents. Do not modify files or access any other directory.",
    codex_path: live ? process.env.CHRONOSYNC_LIVE_CODEX : fixture,
    sandbox: writeFixture ? "workspace-write" : "read-only",
    timeout_seconds: live ? 120 : 5,
    consent: true,
    excludes: [],
  });
  let page;
  const deadline = Date.now() + (live ? 130000 : 10000);
  do {
    page = await engine.request("event.list", { id: run.id, after: "0" });
    if (!["created", "starting", "running"].includes(page.run.state)) break;
    await new Promise((resolve) => setTimeout(resolve, 50));
  } while (Date.now() < deadline);
  for (const [i, event] of page.events.entries()) {
    assert.ok(validate(event), JSON.stringify(validate.errors));
    assert.equal(event.sequence, String(i + 1));
  }
  console.log(
    JSON.stringify({
      live,
      run_id: run.id,
      state: page.run.state,
      reason: page.run.reason,
      event_count: page.events.length,
      kinds: [...new Set(page.events.map((e) => e.kind))],
      recordings: store,
    }),
  );
  assert.equal(
    page.run.state,
    "completed",
    `Run ended ${page.run.state}; inspect local journal at ${store}`,
  );
  assert.ok(
    page.events.some((e) => e.kind === "command.completed"),
    "No real command result observed",
  );
  if (writeFixture)
    assert.ok(
      page.events.some(
        (e) =>
          e.kind === "workspace.changed" &&
          e.payload.changes?.some((c) => c.path === "generated.txt"),
      ),
      "Live workspace change missing",
    );
  if (!live) {
    assert.ok(page.events.some((e) => e.kind === "workspace.changed"));
    assert.ok(page.events.some((e) => e.kind === "adapter.unknown"));
    assert.ok(!JSON.stringify(page).includes("not-preserved"));
  }
  await engine.close();
  engine = connect();
  await engine.request("hello");
  const runs = await engine.request("run.list");
  assert.equal(runs[0].id, run.id);
  assert.equal(runs[0].state, "completed");
  const query = {
    id: run.id,
    kind: "",
    search: "",
    min: "",
    max: "",
    limit: 3,
    cursor: "",
  };
  const seen = [];
  do {
    const timeline = await engine.request("event.query", query);
    assert.ok(timeline.events.length <= 3);
    seen.push(...timeline.events.map((event) => event.event_id));
    query.cursor = timeline.next;
  } while (query.cursor);
  assert.deepEqual(
    seen,
    page.events.map((event) => event.event_id),
  );
  const command = page.events.find(
    (event) => event.kind === "command.completed",
  );
  const detail = await engine.request("event.get", {
    id: run.id,
    event_id: command.event_id,
  });
  assert.deepEqual(detail, command);
  const filtered = await engine.request("event.query", {
    ...query,
    kind: "command.completed",
    min: command.sequence,
    max: command.sequence,
  });
  assert.equal(filtered.events.length, 1);
  const runPage = await engine.request("run.query", { limit: 1, cursor: "" });
  assert.equal(runPage.runs[0].id, run.id);
  if (!live) {
    const originalBytes = readFileSync(join(workspace, "README.txt"));
    const snapshot = await engine.request("snapshot.capture", {
      run_id: run.id,
      consent: true,
      excludes: [],
    });
    assert.equal(snapshot.state, "committed");
    const restore = await engine.request("snapshot.restore", {
      snapshot_id: snapshot.id,
      consent: true,
    });
    assert.equal(restore.state, "ready");
    assert.deepEqual(
      readFileSync(join(restore.workspace, "README.txt")),
      originalBytes,
    );
    const branch = await engine.request("branch.create", {
      snapshot_id: snapshot.id,
      consent: true,
    });
    writeFileSync(
      join(branch.workspace, "README.txt"),
      "changed synthetic branch bytes",
    );
    const changed = await engine.request("snapshot.capture", {
      run_id: run.id,
      branch_id: branch.id,
      consent: true,
      excludes: [],
    });
    const child = await engine.request("branch.create", {
      snapshot_id: changed.id,
      parent_id: branch.id,
      consent: true,
    });
    assert.equal(child.parent_id, branch.id);
    assert.deepEqual(
      readFileSync(join(workspace, "README.txt")),
      originalBytes,
    );
    assert.equal(
      (await engine.request("snapshot.validate", { id: snapshot.id })).state,
      "committed",
    );
    await engine.close();
    engine = connect();
    await engine.request("hello");
    assert.equal(
      (await engine.request("snapshot.get", { id: snapshot.id })).state,
      "committed",
    );
    assert.equal(
      (await engine.request("branch.get", { id: child.id })).state,
      "ready",
    );
    const storedSnapshots = await engine.request("snapshot.list", {
      run_id: run.id,
      limit: 100,
    });
    assert.equal(storedSnapshots.snapshots.length, 2);
    const unchanged = await engine.request("event.query", {
      id: run.id,
      limit: 100,
    });
    assert.equal(unchanged.count, page.events.length);
    console.log(
      "PASS: snapshot capture, restore bytes, branch ancestry/isolation and restart through actual engine IPC.",
    );
  }
  console.log(
    "PASS: recorder IPC, event v2 schema, command output, terminal state and persistence, timeline paging/filter/detail after restart." +
      (live ? " Live Codex used." : " Synthetic subprocess only."),
  );
} finally {
  await engine.close();
}
