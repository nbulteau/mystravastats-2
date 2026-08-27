#!/usr/bin/env node

import { readdir, readFile } from "node:fs/promises";
import { dirname, extname, relative, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const repository = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const failures = [];
const productionFiles = [];

await checkKotlinDomainBoundary();
await checkGoLayerBoundaries();
await checkFrontendApiBoundary();
await checkModuleSizeRatchet();

if (failures.length > 0) {
  console.error(`Architecture check failed with ${failures.length} violation(s):`);
  failures.forEach((failure) => console.error(`- ${failure}`));
  process.exit(1);
}

console.log("Architecture boundaries and module-size ratchet are valid.");

async function checkKotlinDomainBoundary() {
  const root = resolve(repository, "back-kotlin/src/main/kotlin/me/nicolas/stravastats/domain");
  for (const path of await walk(root, new Set([".kt"]))) {
    const source = await readFile(path, "utf8");
    for (const match of source.matchAll(/^import me\.nicolas\.stravastats\.(api|adapters)\b.*$/gm)) {
      failures.push(`${projectPath(path)}: domain code imports an outer ${match[1]} layer (${match[0]}).`);
    }
  }
}

async function checkGoLayerBoundaries() {
  const root = resolve(repository, "back-go/internal");
  for (const path of await walk(root, new Set([".go"]))) {
    const normalized = projectPath(path);
    if (!normalized.includes("/application/") && !normalized.includes("/domain/")) continue;
    const source = await readFile(path, "utf8");
    for (const match of source.matchAll(/["`]mystravastats\/internal\/[^"`]+\/infrastructure(?:\/[^"`]*)?["`]/g)) {
      failures.push(`${normalized}: inner layer imports infrastructure (${match[0]}).`);
    }
  }
}

async function checkFrontendApiBoundary() {
  const root = resolve(repository, "front-vue/src");
  for (const path of await walk(root, new Set([".ts", ".vue"]))) {
    const normalized = projectPath(path);
    if (isTestOrGenerated(normalized)) continue;
    const source = await readFile(path, "utf8");
    if (normalized !== "front-vue/src/services/http-client.ts" && /\bfetch\s*\(/.test(source)) {
      failures.push(`${normalized}: direct fetch bypasses services/http-client.ts.`);
    }
    if (/['"`]\/api\//.test(source)) {
      failures.push(`${normalized}: raw API path bypasses services/api-url.ts.`);
    }
    if (source.includes("@/stores/api")) {
      failures.push(`${normalized}: transport must not be owned by the stores layer.`);
    }
  }
}

async function checkModuleSizeRatchet() {
  const baselinePath = resolve(repository, "docs/architecture/module-size-baseline.json");
  const baseline = JSON.parse(await readFile(baselinePath, "utf8"));
  const roots = [
    resolve(repository, "back-go"),
    resolve(repository, "back-kotlin/src/main/kotlin"),
    resolve(repository, "front-vue/src"),
  ];
  for (const root of roots) {
    productionFiles.push(...await walk(root, new Set([".go", ".kt", ".ts", ".vue"])));
  }
  const known = new Set(Object.keys(baseline.modules));
  for (const path of productionFiles) {
    const normalized = projectPath(path);
    if (isTestOrGenerated(normalized) || normalized.startsWith("back-go/docs/")) continue;
    const lines = lineCount(await readFile(path, "utf8"));
    const ceiling = baseline.modules[normalized];
    if (ceiling !== undefined && lines > ceiling) {
      failures.push(`${normalized}: ${lines} lines exceeds its ratchet ceiling of ${ceiling}.`);
    } else if (ceiling === undefined && lines > baseline.maximumNewModuleLines) {
      failures.push(`${normalized}: new oversized module has ${lines} lines (maximum ${baseline.maximumNewModuleLines}).`);
    }
  }
  for (const path of known) {
    if (!productionFiles.some((candidate) => projectPath(candidate) === path)) {
      failures.push(`${path}: stale module-size baseline entry.`);
    }
  }
}

async function walk(root, extensions) {
  const files = [];
  for (const entry of await readdir(root, { withFileTypes: true })) {
    const path = resolve(root, entry.name);
    if (entry.isDirectory()) files.push(...await walk(path, extensions));
    else if (entry.isFile() && extensions.has(extname(entry.name))) files.push(path);
  }
  return files;
}

function isTestOrGenerated(path) {
  return path.includes("/__tests__/")
    || path.includes("/src/test/")
    || path.endsWith(".spec.ts")
    || path.endsWith("_test.go")
    || path.includes("/generated/")
    || path.endsWith("GeneratedApiContract.kt")
    || path.endsWith("generated_contract.go");
}

function projectPath(path) {
  return relative(repository, path).replaceAll("\\", "/");
}

function lineCount(source) {
  return source.length === 0 ? 0 : source.split("\n").length - (source.endsWith("\n") ? 1 : 0);
}
