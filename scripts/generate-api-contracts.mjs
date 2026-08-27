#!/usr/bin/env node

import { readdir, readFile, writeFile } from "node:fs/promises";
import { execFileSync } from "node:child_process";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const specPath = resolve(repository, "docs/api/openapi.json");
const checkOnly = process.argv.includes("--check");
const httpMethods = new Set(["get", "post", "put", "delete", "patch"]);

const spec = JSON.parse(await readFile(specPath, "utf8"));
const operations = [];
const operationIds = new Set();

for (const [path, pathItem] of Object.entries(spec.paths ?? {})) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (!httpMethods.has(method)) continue;
    if (!operation.operationId) throw new Error(`${method.toUpperCase()} ${path} has no operationId`);
    if (operationIds.has(operation.operationId)) throw new Error(`Duplicate operationId: ${operation.operationId}`);
    if (!operation.responses || Object.keys(operation.responses).length === 0) {
      throw new Error(`${operation.operationId} has no response contract`);
    }
    operationIds.add(operation.operationId);
    operations.push({ method: method.toUpperCase(), path, operationId: operation.operationId });
  }
}
operations.sort((left, right) => left.path.localeCompare(right.path) || left.method.localeCompare(right.method));

await assertGoRouteParity(operations);
await assertKotlinRouteParity(operations);

const schemas = spec.components?.schemas ?? {};
const outputs = new Map([
  [resolve(repository, "front-vue/src/generated/api-contract.ts"), renderTypeScript(operations, schemas)],
  [resolve(repository, "back-go/api/dto/generated_contract.go"), formatGo(renderGo(operations, schemas))],
  [resolve(repository, "back-kotlin/src/main/kotlin/me/nicolas/stravastats/api/dto/GeneratedApiContract.kt"), renderKotlin(operations, schemas)],
]);

for (const [path, content] of outputs) {
  if (checkOnly) {
    const current = await readFile(path, "utf8").catch(() => "");
    if (current !== content) throw new Error(`${path} is stale; run node scripts/generate-api-contracts.mjs`);
  } else {
    await writeFile(path, content, "utf8");
  }
}

console.log(`${checkOnly ? "Checked" : "Generated"} ${operations.length} API operations and ${Object.keys(schemas).length} shared schemas.`);

async function assertGoRouteParity(contractOperations) {
  const source = await readFile(resolve(repository, "back-go/api/routes.go"), "utf8");
  const routePattern = /Method:\s*"([A-Z]+)",\s*Pattern:\s*"([^"]+)"/g;
  const implemented = new Set();
  for (const match of source.matchAll(routePattern)) implemented.add(`${match[1]} ${match[2]}`);
  const contracted = new Set(contractOperations.map(({ method, path }) => `${method} ${path}`));
  const missing = [...implemented].filter((route) => !contracted.has(route));
  const stale = [...contracted].filter((route) => !implemented.has(route));
  if (missing.length || stale.length) {
    throw new Error(`OpenAPI/Go route mismatch. Missing: ${missing.join(", ") || "none"}. Stale: ${stale.join(", ") || "none"}.`);
  }
}

async function assertKotlinRouteParity(contractOperations) {
  const controllersDirectory = resolve(repository, "back-kotlin/src/main/kotlin/me/nicolas/stravastats/api/controllers");
  const implemented = new Set();
  for (const entry of await readdir(controllersDirectory, { withFileTypes: true })) {
    if (!entry.isFile() || !entry.name.endsWith("Controller.kt")) continue;
    const source = await readFile(resolve(controllersDirectory, entry.name), "utf8");
    const basePath = source.match(/@RequestMapping\(\s*"([^"]*)"/)?.[1];
    if (basePath === undefined) continue;
    const mappingPattern = /@(Get|Post|Put|Delete|Patch)Mapping(?:\(([^)]*)\))?/g;
    for (const match of source.matchAll(mappingPattern)) {
      const relativePath = match[2]?.match(/^\s*"([^"]*)"/)?.[1] ?? "";
      implemented.add(`${match[1].toUpperCase()} /api${basePath}${relativePath}`);
    }
  }
  assertRouteParity("Kotlin", implemented, contractOperations);
}

function assertRouteParity(runtime, implemented, contractOperations) {
  const contracted = new Set(contractOperations.map(({ method, path }) => `${method} ${path}`));
  const missing = [...implemented].filter((route) => !contracted.has(route));
  const stale = [...contracted].filter((route) => !implemented.has(route));
  if (missing.length || stale.length) {
    throw new Error(`OpenAPI/${runtime} route mismatch. Missing: ${missing.join(", ") || "none"}. Stale: ${stale.join(", ") || "none"}.`);
  }
}

function renderTypeScript(apiOperations, apiSchemas) {
  const models = Object.entries(apiSchemas).map(([name, schema]) => {
    const required = new Set(schema.required ?? []);
    const properties = Object.entries(schema.properties ?? {}).map(([propertyName, property]) =>
      `  ${propertyName}${required.has(propertyName) ? "" : "?"}: ${tsType(property)};`
    ).join("\n");
    return `export interface ${name} {\n${properties}\n}`;
  }).join("\n\n");
  const endpointRows = apiOperations.map(({ method, path, operationId }) =>
    `  ${operationId}: { method: "${method}", path: "${path}" },`
  ).join("\n");
  return `// Code generated from docs/api/openapi.json. DO NOT EDIT.\n\n${models}\n\nexport const apiOperations = {\n${endpointRows}\n} as const;\n\nexport type ApiOperationId = keyof typeof apiOperations;\n`;
}

function renderGo(apiOperations, apiSchemas) {
  const models = Object.entries(apiSchemas).map(([name, schema]) => {
    const required = new Set(schema.required ?? []);
    const fields = Object.entries(schema.properties ?? {}).map(([propertyName, property]) => {
      const optional = !required.has(propertyName) || property.nullable;
      const fieldType = goType(property, optional);
      return `\t${pascalCase(propertyName)} ${fieldType} \`json:"${propertyName}${optional ? ",omitempty" : ""}"\``;
    }).join("\n");
    return `type Contract${name} struct {\n${fields}\n}`;
  }).join("\n\n");
  const rows = apiOperations.map(({ method, path, operationId }) =>
    `\t"${operationId}": {Method: "${method}", Path: "${path}"},`
  ).join("\n");
  return `// Code generated from docs/api/openapi.json. DO NOT EDIT.\n\npackage dto\n\n${models}\n\ntype ContractOperation struct {\n\tMethod string\n\tPath string\n}\n\nvar ContractOperations = map[string]ContractOperation{\n${rows}\n}\n`;
}

function renderKotlin(apiOperations, apiSchemas) {
  const models = Object.entries(apiSchemas).map(([name, schema]) => {
    const required = new Set(schema.required ?? []);
    const fields = Object.entries(schema.properties ?? {}).map(([propertyName, property]) => {
      const optional = !required.has(propertyName) || property.nullable;
      return `    val ${propertyName}: ${kotlinType(property)}${optional ? "? = null" : ""}`;
    }).join(",\n");
    return `data class Contract${name}(\n${fields},\n)`;
  }).join("\n\n");
  const rows = apiOperations.map(({ method, path, operationId }) =>
    `    "${operationId}" to ContractOperation("${method}", "${path}"),`
  ).join("\n");
  return `// Code generated from docs/api/openapi.json. DO NOT EDIT.\n\npackage me.nicolas.stravastats.api.dto\n\n${models}\n\ndata class ContractOperation(val method: String, val path: String)\n\nval contractOperations: Map<String, ContractOperation> = mapOf(\n${rows}\n)\n`;
}

function formatGo(content) {
  return execFileSync("gofmt", { input: content, encoding: "utf8" });
}

function tsType(schema) {
  if (schema.enum) return schema.enum.map((value) => JSON.stringify(value)).join(" | ");
  if (schema.type === "integer" || schema.type === "number") return "number";
  if (schema.type === "boolean") return "boolean";
  if (schema.type === "array") return `${tsType(schema.items ?? {})}[]`;
  if (schema.type === "object") return "Record<string, unknown>";
  return "string";
}

function goType(schema, optional) {
  let value;
  if (schema.type === "integer") value = "int64";
  else if (schema.type === "number") value = "float64";
  else if (schema.type === "boolean") value = "bool";
  else if (schema.type === "array") value = `[]${goType(schema.items ?? {}, false)}`;
  else if (schema.type === "object") value = "map[string]any";
  else value = "string";
  return optional && !value.startsWith("[]") && !value.startsWith("map[") ? `*${value}` : value;
}

function kotlinType(schema) {
  if (schema.type === "integer") return "Long";
  if (schema.type === "number") return "Double";
  if (schema.type === "boolean") return "Boolean";
  if (schema.type === "array") return `List<${kotlinType(schema.items ?? {})}>`;
  if (schema.type === "object") return "Map<String, Any>";
  return "String";
}

function pascalCase(value) {
  return value.replace(/(^|[-_])(\w)/g, (_match, _separator, character) => character.toUpperCase());
}
