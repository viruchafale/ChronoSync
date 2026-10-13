import { spawnSync } from "node:child_process";
import { readFileSync } from "node:fs";
import Ajv from "ajv";
const result = spawnSync("go", ["run", "./cmd/chronosync-engine", "version"], {
  cwd: "engine",
  encoding: "utf8",
});
if (result.error || result.status !== 0)
  throw new Error(result.error?.message ?? result.stderr);
const validate = new Ajv({ strict: true }).compile(
  JSON.parse(readFileSync("contracts/version.schema.json", "utf8")),
);
if (!validate(JSON.parse(result.stdout)))
  throw new Error(JSON.stringify(validate.errors));
console.log(
  "PASS: Go executable output satisfies the frontend JSON schema. No live agent invoked.",
);
