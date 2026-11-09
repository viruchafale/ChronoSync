import { spawnSync } from "node:child_process";
import { mkdirSync } from "node:fs";
import { resolve } from "node:path";
const releaseSigning = process.env.CHRONOSYNC_RELEASE_SIGN_SIDECAR === "1";
const signingIdentity = process.env.APPLE_SIGNING_IDENTITY;
if (releaseSigning) {
  if (process.platform !== "darwin") {
    throw new Error("release sidecar signing is supported only on macOS");
  }
  if (!signingIdentity || signingIdentity === "-") {
    throw new Error(
      "release sidecar signing requires a non-ad-hoc APPLE_SIGNING_IDENTITY",
    );
  }
}
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
if (result.status !== 0) process.exit(result.status ?? 1);

if (releaseSigning) {
  const signed = spawnSync(
    "codesign",
    [
      "--force",
      "--timestamp",
      "--options",
      "runtime",
      "--identifier",
      "dev.chronosync.engine",
      "--sign",
      signingIdentity,
      output,
    ],
    { stdio: "inherit" },
  );
  if (signed.error) throw signed.error;
  if (signed.status !== 0) process.exit(signed.status ?? 1);
}
