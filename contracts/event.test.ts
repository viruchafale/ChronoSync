import { expect, it } from "vitest";
import Ajv from "ajv";
import schema from "./event.schema.json";
import type { EventEnvelope } from "./event";
const validate = new Ajv({ strict: true }).compile(schema);
const event: EventEnvelope = {
  schema_version: 1,
  event_id: "synthetic-event",
  run_id: "synthetic-run",
  producer_id: "test",
  producer_seq: "1",
  ingest_seq: "1",
  lamport: "9007199254740993",
  observed_at: "2026-10-21T00:00:00Z",
  kind: "adapter.unknown",
  causes: [],
  payload_ref: null,
  redaction_version: 1,
  adapter_type: "synthetic",
  adapter_version: "0.0.0",
};
it("preserves counters above JavaScript integer precision as decimal strings", () => {
  expect(validate(JSON.parse(JSON.stringify(event)))).toBe(true);
  expect(JSON.parse(JSON.stringify(event)).lamport).toBe("9007199254740993");
});
it.each([
  { ...event, lamport: 9007199254740992 },
  { ...event, ingest_seq: "0" },
  { ...event, payload_ref: "../../secret" },
  { ...event, causes: ["x", "x"] },
  { ...event, schema_version: 2 },
  { ...event, observed_at: "yesterday" },
])("rejects incompatible event wire shapes", (value) =>
  expect(validate(value)).toBe(false),
);
