# Isolated visual evidence harness

Run `npx vite --config tests/visual/vite.config.ts`, then open `http://127.0.0.1:1421/`.

This separate Vite root renders the actual application components with disposable synthetic evidence through existing typed API props. It is not imported by the production entry or build, has no Tauri host or engine connection, and has no production mode switch. The fixed amber label identifies all screenshots as synthetic visual tests. Missing operations reject with `VISUAL_TEST_READ_ONLY`; capture, materialization, signing, import/export and deletion cannot change data. Clicking controls can display those honest test refusal states. Comparison returns a pre-existing synthetic test report, not a new evaluation result.

The analysis, exchange, governance and replay evidence modules copy the constants from their corresponding existing component tests. Recorder/snapshot examples are inert and include long IDs, output, warning and damaged states. These values prove presentation only: no cryptography, integrity, custody, replay, provider or engine acceptance follows from them.

Typecheck independently with `npx tsc --project tests/visual/tsconfig.json --noEmit`. Repository ESLint and Prettier also inspect this directory. Production `npm run build` continues to use `apps/desktop/index.html`.

For a populated review: select an execution in Overview; select its first timeline row; visit Snapshots. On Replay use Refresh fixtures and Inspect. On Analysis select a comparison execution and Compare recorded evidence. On Governance and Exchange use their inventory refresh controls. Inspect signed package returns a labeled test verification. Restore/branch confirmation can be opened and cancelled; confirming produces a read-only test refusal.
