import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";
const directory = resolve("apps/desktop/src-tauri/binaries");
mkdirSync(directory, { recursive: true });
const output = resolve(
  directory,
  process.platform === "win32" ? "chronosync-engine.exe" : "chronosync-engine",
);
const result = spawnSync(
  "go",
  ["build", "-trimpath", "-o", output, "./cmd/chronosync-engine"],
  { cwd: "engine", stdio: "inherit" },
);
if (result.error) throw result.error;
process.exit(result.status ?? 1);
