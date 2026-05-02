#!/usr/bin/env node

import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import { cp, mkdir, mkdtemp, rm } from "node:fs/promises";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { tmpdir } from "node:os";

const scriptDir = dirname(fileURLToPath(import.meta.url));
const repoRoot = resolve(scriptDir, "..");
const fixtureRoot = join(repoRoot, "test-fixtures", "source-modes");
const smokeYear = 2026;
const smokeActivityType = "Ride";
const supportedModes = ["STRAVA", "FIT", "GPX"];

main().catch((error) => {
  console.error(error.message);
  process.exit(1);
});

async function main() {
  const options = parseArgs(process.argv.slice(2));
  if (options.help) {
    printHelp();
    return;
  }

  validateFixtureTree();

  const tempRoot = await mkdtemp(join(tmpdir(), "mystravastats-source-modes-"));
  let backendContext = null;
  try {
    const fixturePaths = await copyRuntimeFixtures(tempRoot);
    if (!options.backendUrl && options.backend === "go" && !process.env.SOURCE_MODES_BACKEND_CMD) {
      backendContext = await buildGoBackend(tempRoot);
    }

    for (const [index, mode] of options.modes.entries()) {
      const port = options.portStart + index;
      const baseUrl = options.backendUrl || `http://127.0.0.1:${port}`;
      let processHandle = null;
      try {
        if (!options.backendUrl) {
          processHandle = launchBackend({
            backend: options.backend,
            backendContext,
            fixturePaths,
            mode,
            port,
            tempRoot,
          });
          await waitForHealthy(baseUrl, options.timeoutMs, processHandle);
        }
        const result = await checkSourceMode(baseUrl, mode, fixturePaths[mode], {
          expectActivePreview: !options.backendUrl,
        });
        console.log(
          `ok ${options.backendUrl ? "existing" : options.backend}/${mode}: ` +
            `${result.activities} activities, map tracks=${result.mapTracks}`,
        );
      } catch (error) {
        if (processHandle) {
          await delay(1000);
          throw new Error(`${error.message}\n${tailLogs(processHandle)}`);
        }
        throw error;
      } finally {
        if (processHandle) {
          await stopProcess(processHandle);
        }
      }
    }
  } finally {
    if (!options.keepTemp) {
      await rm(tempRoot, { recursive: true, force: true });
    } else {
      console.log(`kept temp directory: ${tempRoot}`);
    }
  }
}

function parseArgs(args) {
  const options = {
    backend: "go",
    backendUrl: "",
    modes: supportedModes,
    portStart: 19080,
    timeoutMs: 90000,
    keepTemp: false,
    help: false,
  };

  for (let index = 0; index < args.length; index++) {
    const arg = args[index];
    switch (arg) {
      case "--backend":
        options.backend = requireValue(args, ++index, arg);
        break;
      case "--backend-url":
        options.backendUrl = stripTrailingSlash(requireValue(args, ++index, arg));
        break;
      case "--modes":
        options.modes = requireValue(args, ++index, arg)
          .split(",")
          .map((mode) => mode.trim().toUpperCase())
          .filter(Boolean);
        break;
      case "--port-start":
        options.portStart = Number.parseInt(requireValue(args, ++index, arg), 10);
        break;
      case "--timeout-ms":
        options.timeoutMs = Number.parseInt(requireValue(args, ++index, arg), 10);
        break;
      case "--keep-temp":
        options.keepTemp = true;
        break;
      case "--help":
      case "-h":
        options.help = true;
        break;
      default:
        throw new Error(`Unknown option: ${arg}`);
    }
  }

  if (!["go", "kotlin"].includes(options.backend)) {
    throw new Error(`--backend must be "go" or "kotlin", got "${options.backend}"`);
  }
  if (!Number.isInteger(options.portStart) || options.portStart <= 0) {
    throw new Error("--port-start must be a positive integer");
  }
  if (!Number.isInteger(options.timeoutMs) || options.timeoutMs < 1000) {
    throw new Error("--timeout-ms must be an integer >= 1000");
  }
  if (options.modes.length === 0) {
    throw new Error("--modes must include at least one mode");
  }
  for (const mode of options.modes) {
    if (!supportedModes.includes(mode)) {
      throw new Error(`Unsupported mode "${mode}". Expected one of ${supportedModes.join(", ")}`);
    }
  }
  if (options.backendUrl && options.modes.length !== 1) {
    throw new Error("--backend-url validates one already-running mode; pass a single mode with --modes");
  }

  return options;
}

function requireValue(args, index, option) {
  const value = args[index];
  if (!value || value.startsWith("--")) {
    throw new Error(`${option} requires a value`);
  }
  return value;
}

function stripTrailingSlash(value) {
  return value.replace(/\/+$/, "");
}

function printHelp() {
  console.log(`Source mode smoke test

Usage:
  node scripts/smoke-source-modes.mjs [options]

Options:
  --backend <go|kotlin>    Backend to launch (default: go)
  --backend-url <url>      Validate one already-running backend instead of launching
  --modes <list>           Comma list: STRAVA,FIT,GPX (default: all)
  --port-start <port>      First temporary port when launching (default: 19080)
  --timeout-ms <ms>        Backend startup timeout (default: 90000)
  --keep-temp              Keep copied fixtures and built binary for inspection
  --help                   Show this help

Override launch command:
  SOURCE_MODES_BACKEND_CMD="path/to/server -port {port}" node scripts/smoke-source-modes.mjs --modes GPX
`);
}

function validateFixtureTree() {
  const required = [
    join(fixtureRoot, "strava", ".strava"),
    join(fixtureRoot, "strava", "strava-123456", "strava-123456-2026", "activities-123456-2026.json"),
    join(fixtureRoot, "fit", "2026", "smoke-ride.fit"),
    join(fixtureRoot, "gpx", "2026", "smoke-ride.gpx"),
  ];
  const missing = required.filter((path) => !existsSync(path));
  if (missing.length > 0) {
    throw new Error(
      "Missing source-mode fixtures:\n" +
        missing.map((path) => `- ${path}`).join("\n") +
        "\nRun: cd back-go && go run ../scripts/generate-source-mode-fit-fixture.go --out ../test-fixtures/source-modes/fit/2026/smoke-ride.fit",
    );
  }
}

async function copyRuntimeFixtures(tempRoot) {
  const runtimeRoot = join(tempRoot, "fixtures");
  await mkdir(runtimeRoot, { recursive: true });

  const paths = {};
  for (const mode of supportedModes) {
    const source = join(fixtureRoot, mode.toLowerCase());
    const destination = join(runtimeRoot, mode.toLowerCase());
    await cp(source, destination, { recursive: true });
    paths[mode] = destination;
  }
  return paths;
}

async function buildGoBackend(tempRoot) {
  const binary = join(tempRoot, "bin", "mystravastats");
  await mkdir(dirname(binary), { recursive: true });
  const env = { ...process.env, GOCACHE: process.env.GOCACHE || join(tempRoot, "go-build-cache") };
  await runCommand("go", ["build", "-o", binary, "."], {
    cwd: join(repoRoot, "back-go"),
    env,
    label: "go build backend",
  });
  return { binary };
}

function launchBackend({ backend, backendContext, fixturePaths, mode, port, tempRoot }) {
  const env = backendEnvironment(mode, fixturePaths, port, tempRoot);
  const override = process.env.SOURCE_MODES_BACKEND_CMD;
  if (override) {
    return spawnLogged(override.replaceAll("{port}", String(port)), [], {
      cwd: repoRoot,
      env,
      label: `${backend}/${mode}`,
      shell: true,
    });
  }

  if (backend === "go") {
    return spawnLogged(backendContext.binary, ["-host", "127.0.0.1", "-port", String(port)], {
      cwd: tempRoot,
      env,
      label: `go/${mode}`,
    });
  }

  return spawnLogged("./gradlew", ["bootRun", `--args=--server.address=127.0.0.1 --server.port=${port}`], {
    cwd: join(repoRoot, "back-kotlin"),
    env,
    label: `kotlin/${mode}`,
  });
}

function backendEnvironment(mode, fixturePaths, port, tempRoot) {
  const env = {
    ...process.env,
    OPEN_BROWSER: "false",
    OSM_ROUTING_ENABLED: "false",
    OSRM_CONTROL_ENABLED: "false",
    STRAVA_CACHE_PATH: fixturePaths.STRAVA,
    SERVER_HOST: "127.0.0.1",
    HOST: "127.0.0.1",
    SERVER_ADDRESS: "127.0.0.1",
    PORT: String(port),
    SERVER_PORT: String(port),
    GOCACHE: process.env.GOCACHE || join(tempRoot, "go-build-cache"),
  };

  delete env.FIT_FILES_PATH;
  delete env.GPX_FILES_PATH;

  if (mode === "FIT") {
    env.FIT_FILES_PATH = fixturePaths.FIT;
  }
  if (mode === "GPX") {
    env.GPX_FILES_PATH = fixturePaths.GPX;
  }

  return env;
}

function spawnLogged(command, args, { cwd, env, label, shell = false }) {
  const child = spawn(command, args, {
    cwd,
    env,
    shell,
    stdio: ["ignore", "pipe", "pipe"],
  });
  const handle = { child, label, logs: [] };
  child.stdout.on("data", (chunk) => appendLog(handle, chunk));
  child.stderr.on("data", (chunk) => appendLog(handle, chunk));
  child.on("exit", (code, signal) => {
    appendLog(handle, `${label} exited with code=${code} signal=${signal}\n`);
  });
  return handle;
}

function appendLog(handle, chunk) {
  handle.logs.push(chunk.toString());
  if (handle.logs.length > 200) {
    handle.logs.splice(0, handle.logs.length - 200);
  }
}

async function waitForHealthy(baseUrl, timeoutMs, processHandle) {
  const deadline = Date.now() + timeoutMs;
  let lastError = null;
  while (Date.now() < deadline) {
    if (processHandle.child.exitCode !== null) {
      throw new Error(
        `${processHandle.label} exited before becoming healthy\n${tailLogs(processHandle)}`,
      );
    }
    try {
      return await fetchJSON(`${baseUrl}/api/health/details`, { label: "health" });
    } catch (error) {
      lastError = error;
      await delay(500);
    }
  }
  throw new Error(
    `Timed out waiting for ${baseUrl}/api/health/details: ${lastError?.message ?? "no response"}\n` +
      tailLogs(processHandle),
  );
}

async function checkSourceMode(baseUrl, mode, fixturePath, { expectActivePreview }) {
  const health = await fetchJSON(`${baseUrl}/api/health/details`, { label: "health" });
  const provider = nested(health, ["runtimeConfig", "data", "provider"]) ?? health.provider;
  assertEqual(String(provider).toUpperCase(), mode, `health provider for ${mode}`);

  const preview = await fetchJSON(`${baseUrl}/api/source-modes/preview`, {
    label: "source preview",
    method: "POST",
    body: JSON.stringify({ mode, path: fixturePath }),
    headers: { "Content-Type": "application/json" },
  });
  assertEqual(preview.mode, mode, "preview mode");
  assertEqual(preview.activeMode, mode, "preview activeMode");
  assertTruthy(preview.supported, "preview supported");
  assertTruthy(preview.readable, "preview readable");
  assertTruthy(preview.validStructure, "preview validStructure");
  if (expectActivePreview) {
    assertTruthy(preview.active, "preview active");
  }
  assertNumberAtLeast(preview.activityCount, 1, "preview activityCount");

  await fetchJSON(`${baseUrl}/api/dashboard?activityType=${smokeActivityType}`, { label: "dashboard" });

  const activities = await fetchJSON(
    `${baseUrl}/api/activities?year=${smokeYear}&activityType=${smokeActivityType}`,
    { label: "activities" },
  );
  if (!Array.isArray(activities) || activities.length < 1) {
    throw new Error(`${mode} activities endpoint returned no activities`);
  }
  const activityId = activities[0]?.id;
  assertTruthy(activityId, `${mode} first activity id`);

  const detail = await fetchJSON(`${baseUrl}/api/activities/${activityId}`, { label: "activity detail" });
  assertEqual(Number(detail.id), Number(activityId), `${mode} detail id`);

  const mapTracks = await fetchJSON(
    `${baseUrl}/api/maps/gpx?year=${smokeYear}&activityType=${smokeActivityType}`,
    { label: "maps gpx" },
  );
  if (!Array.isArray(mapTracks) || mapTracks.length < 1) {
    throw new Error(`${mode} maps GPX endpoint returned no tracks`);
  }

  await fetchJSON(`${baseUrl}/api/data-quality/issues`, { label: "data quality" });
  return { activities: activities.length, mapTracks: mapTracks.length };
}

async function fetchJSON(url, init = {}) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 10000);
  try {
    const response = await fetch(url, { ...init, signal: controller.signal });
    const text = await response.text();
    if (!response.ok) {
      throw new Error(`${init.label ?? url} returned HTTP ${response.status}: ${text.slice(0, 500)}`);
    }
    if (/\bNaN\b|\bInfinity\b|-Infinity\b/.test(text)) {
      throw new Error(`${init.label ?? url} contains a non-finite JSON token`);
    }
    const json = text ? JSON.parse(text) : null;
    assertSerializable(json, init.label ?? url);
    return json;
  } finally {
    clearTimeout(timer);
  }
}

function assertSerializable(value, label, path = "$") {
  if (typeof value === "number" && !Number.isFinite(value)) {
    throw new Error(`${label} contains non-finite number at ${path}`);
  }
  if (Array.isArray(value)) {
    value.forEach((item, index) => assertSerializable(item, label, `${path}[${index}]`));
    return;
  }
  if (value && typeof value === "object") {
    for (const [key, item] of Object.entries(value)) {
      assertSerializable(item, label, `${path}.${key}`);
    }
  }
}

async function runCommand(command, args, { cwd, env, label }) {
  const handle = spawnLogged(command, args, { cwd, env, label });
  const code = await new Promise((resolve) => {
    handle.child.on("close", resolve);
  });
  if (code !== 0) {
    throw new Error(`${label} failed with exit code ${code}\n${tailLogs(handle)}`);
  }
}

async function stopProcess(processHandle) {
  if (processHandle.child.exitCode !== null) {
    return;
  }
  processHandle.child.kill("SIGTERM");
  const stopped = await Promise.race([
    new Promise((resolve) => processHandle.child.once("close", () => resolve(true))),
    delay(5000).then(() => false),
  ]);
  if (!stopped && processHandle.child.exitCode === null) {
    processHandle.child.kill("SIGKILL");
    await new Promise((resolve) => processHandle.child.once("close", resolve));
  }
}

function tailLogs(handle) {
  return handle.logs.join("").split("\n").slice(-120).join("\n");
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function nested(object, keys) {
  return keys.reduce((current, key) => (current == null ? undefined : current[key]), object);
}

function assertTruthy(value, label) {
  if (!value) {
    throw new Error(`Expected ${label} to be truthy, got ${JSON.stringify(value)}`);
  }
}

function assertEqual(actual, expected, label) {
  if (actual !== expected) {
    throw new Error(`Expected ${label}=${expected}, got ${actual}`);
  }
}

function assertNumberAtLeast(value, min, label) {
  if (typeof value !== "number" || value < min) {
    throw new Error(`Expected ${label} >= ${min}, got ${JSON.stringify(value)}`);
  }
}
