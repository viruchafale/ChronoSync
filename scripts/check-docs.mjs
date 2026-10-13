import { existsSync, readFileSync, readdirSync } from "node:fs";
import { dirname, resolve } from "node:path";
const required = [
  "README.md",
  "AGENTS.md",
  ...[
    "README",
    "PROJECT_SPECIFICATION",
    "ARCHITECTURE",
    "SYSTEM_DESIGN",
    "DATA_MODEL",
    "SECURITY_MODEL",
    "CODEX_INTEGRATION",
    "TESTING_STRATEGY",
    "IMPLEMENTATION_STATUS",
    "CHANGELOG",
  ].map((n) => `docs/${n}.md`),
  "docs/decisions/ADR-0001-initial-architecture.md",
];
for (const file of required)
  if (!existsSync(file)) throw new Error(`Missing ${file}`);
const plans = readdirSync("docs/phases").filter((n) =>
  /^PHASE-\d{2}-.*\.md$/.test(n),
);
if (plans.length !== 8) throw new Error("Expected eight phase plans");
for (const file of plans) {
  const text = readFileSync(`docs/phases/${file}`, "utf8");
  for (let i = 1; i <= 16; i++)
    if (!text.includes(`## ${i}. `))
      throw new Error(`${file}: missing section ${i}`);
  if (!/`(NOT_STARTED|IN_PROGRESS|BLOCKED|COMPLETE)`/.test(text))
    throw new Error(`${file}: missing status`);
}
const files = [
  ...required,
  ...plans.map((n) => `docs/phases/${n}`),
  ...readdirSync("docs")
    .filter((n) => n.endsWith(".md"))
    .map((n) => `docs/${n}`),
  "contracts/README.md",
];
for (const file of new Set(files)) {
  const text = readFileSync(file, "utf8");
  for (const match of text.matchAll(/\]\(([^)]+)\)/g)) {
    const link = match[1];
    if (/^(https?:|#)/.test(link)) continue;
    if (!existsSync(resolve(dirname(file), link.split("#")[0])))
      throw new Error(`${file}: broken link ${link}`);
  }
}
console.log(
  "PASS: required documents, eight structured phase plans, and local Markdown links",
);
