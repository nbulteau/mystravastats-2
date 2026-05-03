#!/usr/bin/env node

import { spawn } from "node:child_process";
import { createServer } from "node:http";
import { mkdir, writeFile, chmod } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { createInterface } from "node:readline/promises";
import { stdin as input, stdout as output } from "node:process";
import { randomBytes } from "node:crypto";

const DEFAULT_SCOPE = "read_all,activity:read_all,profile:read_all";
const SETTINGS_URL = "https://www.strava.com/settings/api";
const TOKEN_URL = "https://www.strava.com/oauth/token";
const ATHLETE_URL = "https://www.strava.com/api/v3/athlete";

const args = parseArgs(process.argv.slice(2));

if (args.help) {
  printHelp();
  process.exit(0);
}

const rl = createInterface({ input, output });

try {
  await main();
} finally {
  rl.close();
}

async function main() {
  ensureFetch();

  console.log("Strava enrollment assistant");
  console.log("");
  console.log("Strava app creation is manual. This script automates the local setup after that step.");
  console.log(`Open or create the app here: ${SETTINGS_URL}`);
  console.log("Set Authorization Callback Domain to: 127.0.0.1");
  console.log("");

  if (!args.noBrowser) {
    openUrl(SETTINGS_URL);
  }

  const cacheDir = resolve(await ask("Strava cache directory", args.cache ?? process.env.STRAVA_CACHE_PATH ?? "strava-cache"));
  const clientId = await ask("Client ID", args.clientId ?? "");
  const clientSecret = await ask("Client Secret", args.clientSecret ?? "");
  const useCache = normalizeBoolean(await ask("Use cache-only mode? (true/false)", args.useCache ?? "false"));

  if (!/^\d+$/.test(clientId)) {
    throw new Error("Client ID must be numeric.");
  }
  if (!clientSecret.trim()) {
    throw new Error("Client Secret is required for live Strava OAuth.");
  }

  await mkdir(cacheDir, { recursive: true });
  await writePrivateFile(
    resolve(cacheDir, ".strava"),
    `clientId=${clientId.trim()}\nclientSecret=${clientSecret.trim()}\nuseCache=${useCache}\n`,
  );
  console.log(`Wrote ${resolve(cacheDir, ".strava")}`);

  if (args.skipOauth || useCache === "true") {
    console.log("OAuth skipped. Run again with useCache=false when you want live Strava refresh.");
    return;
  }

  const port = Number(args.port ?? 8090);
  const state = randomBytes(24).toString("hex");
  const redirectUri = `http://127.0.0.1:${port}/exchange_token`;
  const authorizeUrl = buildAuthorizeUrl({ clientId, redirectUri, state });

  const codePromise = waitForOAuthCode({ port, state });

  console.log("");
  console.log("Opening Strava OAuth authorization page.");
  console.log(authorizeUrl);
  if (!args.noBrowser) {
    openUrl(authorizeUrl);
  }

  const code = await codePromise;
  const token = await exchangeCode({ clientId, clientSecret, code });
  const athlete = await validateAthlete(token.access_token);
  const tokenPath = resolve(cacheDir, ".strava-token.json");

  await writePrivateJson(tokenPath, {
    token_type: token.token_type,
    access_token: token.access_token,
    refresh_token: token.refresh_token,
    expires_at: token.expires_at,
    expires_in: token.expires_in,
    scope: token.scope,
    athlete,
    created_at: new Date().toISOString(),
  });

  console.log(`Saved ${tokenPath}`);
  console.log(`Validated Strava athlete ${athlete.id}${athlete.username ? ` (${athlete.username})` : ""}.`);
  console.log("");
  console.log("Next launch can use this cache with:");
  console.log(`STRAVA_CACHE_PATH=${shellQuote(cacheDir)} ./mystravastats`);
}

function parseArgs(values) {
  const parsed = {};
  for (let index = 0; index < values.length; index += 1) {
    const value = values[index];
    switch (value) {
      case "--cache":
        parsed.cache = values[++index];
        break;
      case "--client-id":
        parsed.clientId = values[++index];
        break;
      case "--client-secret":
        parsed.clientSecret = values[++index];
        break;
      case "--use-cache":
        parsed.useCache = values[++index];
        break;
      case "--port":
        parsed.port = values[++index];
        break;
      case "--skip-oauth":
        parsed.skipOauth = true;
        break;
      case "--no-browser":
        parsed.noBrowser = true;
        break;
      case "--help":
      case "-h":
        parsed.help = true;
        break;
      default:
        throw new Error(`Unknown option: ${value}`);
    }
  }
  return parsed;
}

function printHelp() {
  console.log(`Usage:
  node scripts/setup-strava-oauth.mjs [options]

Options:
  --cache <path>           Strava cache directory (default: STRAVA_CACHE_PATH or strava-cache)
  --client-id <id>         Strava application Client ID
  --client-secret <secret> Strava application Client Secret
  --use-cache <bool>       Write useCache=true/false (default: false)
  --port <port>            Local OAuth callback port (default: 8090)
  --skip-oauth             Only write .strava
  --no-browser             Print URLs without opening them
  --help                   Show this help
`);
}

async function ask(label, defaultValue) {
  const suffix = defaultValue ? ` [${defaultValue}]` : "";
  const value = await rl.question(`${label}${suffix}: `);
  return value.trim() || defaultValue;
}

function normalizeBoolean(value) {
  const normalized = String(value).trim().toLowerCase();
  if (["1", "true", "yes", "y", "on"].includes(normalized)) return "true";
  if (["0", "false", "no", "n", "off", ""].includes(normalized)) return "false";
  throw new Error(`Invalid boolean value: ${value}`);
}

function buildAuthorizeUrl({ clientId, redirectUri, state }) {
  const url = new URL("https://www.strava.com/oauth/authorize");
  url.searchParams.set("client_id", clientId.trim());
  url.searchParams.set("response_type", "code");
  url.searchParams.set("redirect_uri", redirectUri);
  url.searchParams.set("approval_prompt", "auto");
  url.searchParams.set("scope", DEFAULT_SCOPE);
  url.searchParams.set("state", state);
  return url.toString();
}

function waitForOAuthCode({ port, state }) {
  return new Promise((resolveCode, rejectCode) => {
    let settled = false;
    let timeout;

    const settle = (callback, value) => {
      if (settled) return;
      settled = true;
      clearTimeout(timeout);
      server.close(() => undefined);
      callback(value);
    };

    const server = createServer((request, response) => {
      try {
        const requestUrl = new URL(request.url ?? "/", `http://127.0.0.1:${port}`);
        const returnedState = requestUrl.searchParams.get("state") ?? "";
        const error = requestUrl.searchParams.get("error") ?? "";
        const code = requestUrl.searchParams.get("code") ?? "";

        if (returnedState !== state) {
          throw new Error("OAuth state mismatch. Please retry the setup.");
        }
        if (error) {
          throw new Error(`Strava OAuth failed: ${error}`);
        }
        if (!code) {
          throw new Error("Strava OAuth callback did not include an authorization code.");
        }

        response.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
        response.end("<h1>Access granted</h1><p>You can close this window.</p>");
        settle(resolveCode, code);
      } catch (error) {
        response.writeHead(400, { "Content-Type": "text/plain; charset=utf-8" });
        response.end(error instanceof Error ? error.message : String(error));
        settle(rejectCode, error);
      }
    });

    server.on("error", (error) => settle(rejectCode, error));
    server.listen(port, "127.0.0.1");

    timeout = setTimeout(() => {
      settle(rejectCode, new Error("Timed out waiting for Strava OAuth callback."));
    }, 5 * 60 * 1000).unref();
  });
}

async function exchangeCode({ clientId, clientSecret, code }) {
  const body = new URLSearchParams({
    client_id: clientId.trim(),
    client_secret: clientSecret.trim(),
    code,
    grant_type: "authorization_code",
  });

  const response = await fetch(TOKEN_URL, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body,
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(`Strava token exchange failed (${response.status}): ${JSON.stringify(payload)}`);
  }
  if (!payload.access_token || !payload.refresh_token) {
    throw new Error("Strava token response did not include access_token and refresh_token.");
  }
  return payload;
}

async function validateAthlete(accessToken) {
  const response = await fetch(ATHLETE_URL, {
    headers: { Authorization: `Bearer ${accessToken}` },
  });
  const payload = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(`Unable to validate Strava athlete (${response.status}): ${JSON.stringify(payload)}`);
  }
  return payload;
}

async function writePrivateFile(path, content) {
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, content, { mode: 0o600 });
  await chmod(path, 0o600).catch(() => undefined);
}

async function writePrivateJson(path, payload) {
  await writePrivateFile(path, `${JSON.stringify(payload, null, 2)}\n`);
}

function openUrl(url) {
  const command = process.platform === "darwin"
    ? "open"
    : process.platform === "win32"
      ? "cmd"
      : "xdg-open";
  const args = process.platform === "win32" ? ["/c", "start", "", url] : [url];
  const child = spawn(command, args, { detached: true, stdio: "ignore" });
  child.on("error", () => {
    console.log(`Open this URL manually: ${url}`);
  });
  child.unref();
}

function shellQuote(value) {
  return `'${String(value).replaceAll("'", "'\\''")}'`;
}

function ensureFetch() {
  if (typeof fetch !== "function") {
    throw new Error("This script needs Node.js with global fetch support.");
  }
}
