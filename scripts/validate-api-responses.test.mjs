import { test } from "node:test";
import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import { createResponseValidator, validateBusinessResponseFixtures } from "./validate-api-responses.mjs";

const fixtures = JSON.parse(await readFile(new URL("../test-fixtures/api/business-responses.json", import.meta.url), "utf8"));
const validate = await createResponseValidator();
function detail() {
  const fixture = structuredClone(fixtures.find((entry) => entry.name === "irregular sensor timestamps and real zero watts"));
  return { ...fixture, body: fixture.expected };
}

test("all business fixtures conform to the declared endpoint responses", async () => {
  assert.equal(await validateBusinessResponseFixtures(), fixtures.length);
});
test("rejects missing required nullable fields and wrong sensor types", () => {
  const missing = detail(); delete missing.body.sufferScore;
  assert.throws(() => validate(missing), /sufferScore/);
  const invalid = detail(); invalid.body.stream.watts[1] = "200 W";
  assert.throws(() => validate(invalid), /watts/);
});
test("preserves the distinction between a zero reading and an unavailable sample", () => {
  const record = detail(); record.body.stream.watts = [0, null, 200, 240];
  assert.doesNotThrow(() => validate(record));
  record.body.stream = null;
  assert.doesNotThrow(() => validate(record));
});
test("rejects undocumented response fields and numeric display statistics", () => {
  const record = detail(); record.body.undocumented = true;
  assert.throws(() => validate(record), /additional properties/);
  assert.throws(() => validate({ name: "wrong statistic", operationId: "getStatistics", status: 200,
    body: [{ label: "Distance", value: 1200 }] }), /string/);
});
