#!/usr/bin/env node
import { readFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import SwaggerParser from "@apidevtools/swagger-parser";
import Ajv from "ajv";
import addFormats from "ajv-formats";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");

// OpenAPI 3.0 nullable also applies to reference wrappers. Turn it into a JSON
// Schema union before validation, including nullable elements inside arrays.
function jsonSchema(value) {
  if (Array.isArray(value)) return value.map(jsonSchema);
  if (value === null || typeof value !== "object") return value;
  const { nullable, ...rest } = value;
  const schema = Object.fromEntries(Object.entries(rest).map(([key, child]) => [key, jsonSchema(child)]));
  return nullable ? { anyOf: [schema, { type: "null" }] } : schema;
}

export async function createResponseValidator() {
  const spec = await SwaggerParser.dereference(resolve(root, "docs/api/openapi.json"));
  const ajv = new Ajv({ strict: false, allErrors: true });
  addFormats(ajv);
  const operations = new Map();
  for (const path of Object.values(spec.paths)) {
    for (const operation of Object.values(path)) {
      if (operation?.operationId) operations.set(operation.operationId, operation);
    }
  }
  const compiled = new Map();
  return (record) => {
    const key = `${record.operationId}:${record.status}`;
    if (!compiled.has(key)) {
      const schema = operations.get(record.operationId)?.responses?.[record.status]?.content?.["application/json"]?.schema;
      if (!schema) throw new Error(`No JSON response schema for ${key}`);
      compiled.set(key, ajv.compile(jsonSchema(schema)));
    }
    const validate = compiled.get(key);
    if (!validate(record.body)) {
      throw new Error(`${record.name} (${key}): ${ajv.errorsText(validate.errors, { separator: "\n" })}`);
    }
  };
}

export async function validateBusinessResponseFixtures(actualPaths = []) {
  const fixtures = JSON.parse(await readFile(resolve(root, "test-fixtures/api/business-responses.json"), "utf8"));
  const validate = await createResponseValidator();
  for (const fixture of fixtures) validate({ ...fixture, body: fixture.expected });
  for (const path of actualPaths) {
    const records = JSON.parse(await readFile(resolve(path), "utf8"));
    const byName = new Map(records.map((record) => [record.name, record]));
    if (records.length !== fixtures.length || byName.size !== fixtures.length) {
      throw new Error(`${path}: expected exactly ${fixtures.length} distinct responses`);
    }
    for (const fixture of fixtures) {
      const record = byName.get(fixture.name);
      if (!record || record.operationId !== fixture.operationId || record.status !== fixture.status) {
        throw new Error(`${path}: missing or stale response for ${fixture.name}`);
      }
      validate(record);
    }
  }
  return fixtures.length;
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  const paths = process.argv.slice(2);
  const count = await validateBusinessResponseFixtures(paths);
  console.log(`Validated ${count} business fixtures and ${paths.length} backend response files against OpenAPI.`);
}
