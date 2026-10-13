import { describe, expect, it } from "vitest";
import Ajv from "ajv";
import schema from "./version.schema.json";
import type { EngineVersion } from "./version";
const validate = new Ajv({ strict: true }).compile(schema);
const version: EngineVersion = {
  protocol_version: 1,
  engine_version: "0.0.0",
  capabilities: ["version"],
};
describe("foundation compatibility contract", () => {
  it("accepts the typed v1 report", () => expect(validate(version)).toBe(true));
  it.each([
    { ...version, protocol_version: 2 },
    { ...version, capabilities: ["record"] },
    { ...version, engine_version: null },
    { ...version, secret: "unexpected" },
  ])("rejects incompatible reports", (value) =>
    expect(validate(value)).toBe(false),
  );
});
