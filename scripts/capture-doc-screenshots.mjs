#!/usr/bin/env node

import fs from "node:fs";
import path from "node:path";
import process from "node:process";
import { chromium } from "playwright";

const ACTIVITY_TO_BUTTON_ID = {
  Ride: "ride",
  MountainBikeRide: "mountain-bike-ride",
  Commute: "commute",
  GravelRide: "gravel-ride",
  VirtualRide: "virtual-ride",
  Run: "run",
  TrailRun: "trail-run",
  Hike: "hike",
  Walk: "walk",
  AlpineSki: "alpine-ski",
};

const ACTIVITY_GROUPS = {
  cycling: ["Ride", "MountainBikeRide", "Commute", "GravelRide", "VirtualRide"],
  running: ["Run", "TrailRun"],
  hiking: ["Hike", "Walk"],
  winter: ["AlpineSki"],
};

const SCREEN_SPECS = [
  { key: "dashboard", route: "/dashboard", file: "screen-dashboard.png", applyFilters: true },
  { key: "charts", route: "/charts", file: "screen-charts.png", applyFilters: true },
  { key: "heatmap", route: "/heatmap", file: "screen-heatmap.png", applyFilters: true },
  { key: "statistics", route: "/statistics", file: "screen-statistics.png", applyFilters: true },
  { key: "badges", route: "/badges", file: "screen-badges.png", applyFilters: true },
  { key: "activities", route: "/activities", file: "screen-activities.png", applyFilters: true },
  { key: "map", route: "/map", file: "screen-map.png", applyFilters: true },
  { key: "segments", route: "/segments", file: "screen-segments.png", applyFilters: true },
  { key: "detailed", route: "/activities/__DETAIL_ID__", file: "screen-detailed-activity.png", applyFilters: false },
];

function printHelp() {
  console.log(`
Capture documentation screenshots for MyStravaStats.

Usage:
  node scripts/capture-doc-screenshots.mjs [options]

Options:
  --base-url <url>            Front URL (default: http://localhost:8080)
  --out-dir <path>            Output directory (default: ./docs)
  --year <value>              Year filter (example: 2025 or "All years")
  --activities <list>         Activity selection (same group only).
                              Examples:
                              Ride
                              Run,TrailRun
                              Commute_GravelRide_MountainBikeRide_Ride_VirtualRide
  --detailed-activity-id <id> Activity id for detailed screenshot (default: 15340076302)
  --wait-ms <n>               Wait before each screenshot (default: 1800)
  --viewport <WxH>            Viewport size (default: 1720x1080)
  --full-page                 Capture full page screenshots
  --screens <list>            Comma list: dashboard,charts,heatmap,statistics,badges,activities,map,segments,detailed
  --help                      Show this help
`);
}

function parseArgs(argv) {
  const options = {
    baseUrl: "http://localhost:8080",
    outDir: path.resolve(process.cwd(), "docs"),
    year: undefined,
    activitiesRaw: undefined,
    detailedActivityId: "15340076302",
    waitMs: 1800,
    viewportWidth: 1720,
    viewportHeight: 1080,
    fullPage: false,
    screens: new Set(SCREEN_SPECS.map((screen) => screen.key)),
  };

  for (let i = 0; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--help" || arg === "-h") {
      printHelp();
      process.exit(0);
    }
    if (arg === "--full-page") {
      options.fullPage = true;
      continue;
    }
    if (!arg.startsWith("--")) {
      throw new Error(`Unexpected argument: ${arg}`);
    }
    const next = argv[i + 1];
    if (next === undefined || next.startsWith("--")) {
      throw new Error(`Missing value for ${arg}`);
    }
    i += 1;
    switch (arg) {
      case "--base-url":
        options.baseUrl = next.replace(/\/+$/, "");
        break;
      case "--out-dir":
        options.outDir = path.resolve(process.cwd(), next);
        break;
      case "--year":
        options.year = next;
        break;
      case "--activities":
        options.activitiesRaw = next;
        break;
      case "--detailed-activity-id":
        options.detailedActivityId = next;
        break;
      case "--wait-ms":
        options.waitMs = Number.parseInt(next, 10);
        if (!Number.isFinite(options.waitMs) || options.waitMs < 0) {
          throw new Error(`Invalid --wait-ms value: ${next}`);
        }
        break;
      case "--viewport": {
        const [w, h] = next.toLowerCase().split("x");
        const width = Number.parseInt(w, 10);
        const height = Number.parseInt(h, 10);
        if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
          throw new Error(`Invalid --viewport value: ${next}. Expected format WIDTHxHEIGHT`);
        }
        options.viewportWidth = width;
        options.viewportHeight = height;
        break;
      }
      case "--screens": {
        const values = next.split(",").map((value) => value.trim()).filter(Boolean);
        const allowed = new Set(SCREEN_SPECS.map((screen) => screen.key));
        if (values.length === 0) {
          throw new Error("Invalid --screens value: empty list");
        }
        for (const value of values) {
          if (!allowed.has(value)) {
            throw new Error(`Unknown screen key '${value}'. Allowed: ${[...allowed].join(", ")}`);
          }
        }
        options.screens = new Set(values);
        break;
      }
      default:
        throw new Error(`Unknown option: ${arg}`);
    }
  }
  return options;
}

function normalizeActivityName(rawValue) {
  const trimmed = rawValue.trim();
  if (!trimmed) return null;
  const normalized = trimmed.toLowerCase().replace(/[\s-]/g, "");
  return (
    Object.keys(ACTIVITY_TO_BUTTON_ID).find(
      (activity) => activity.toLowerCase().replace(/[\s-]/g, "") === normalized
    ) ?? null
  );
}

function parseActivities(raw) {
  if (!raw) return [];

  let tokens = raw
    .split(",")
    .flatMap((value) => value.split("_"))
    .map((value) => value.trim())
    .filter(Boolean);

  if (tokens.length === 1) {
    const alias = tokens[0].toLowerCase();
    if (alias === "all-cycling") tokens = [...ACTIVITY_GROUPS.cycling];
    if (alias === "all-running") tokens = [...ACTIVITY_GROUPS.running];
    if (alias === "all-hiking") tokens = [...ACTIVITY_GROUPS.hiking];
    if (alias === "all-winter") tokens = [...ACTIVITY_GROUPS.winter];
  }

  const activities = tokens.map((token) => {
    const normalized = normalizeActivityName(token);
    if (!normalized) {
      throw new Error(`Unknown activity '${token}'.`);
    }
    return normalized;
  });

  return [...new Set(activities)];
}

function findGroupName(activity) {
  for (const [groupName, groupActivities] of Object.entries(ACTIVITY_GROUPS)) {
    if (groupActivities.includes(activity)) return groupName;
  }
  return null;
}

function validateActivitiesWithinSingleGroup(activities) {
  if (activities.length <= 1) return;
  const groups = new Set(activities.map((activity) => findGroupName(activity)).filter(Boolean));
  if (groups.size > 1) {
    throw new Error(
      `Activity selection '${activities.join(", ")}' spans multiple groups. ` +
        "Please select activities from a single group (cycling, running, hiking, winter)."
    );
  }
}

async function waitForHeader(page) {
  await page.waitForSelector("#year", { timeout: 30000 });
  await page.waitForFunction(
    () => {
      const year = document.querySelector("#year");
      return !!year && year instanceof HTMLSelectElement && year.options.length > 0;
    },
    { timeout: 30000 }
  );
}

async function applyYearFilter(page, year) {
  if (!year) return;
  const availableYears = await page.$$eval("#year option", (options) =>
    options.map((option) => (option).value)
  );

  if (!availableYears.includes(year)) {
    console.warn(
      `Requested year '${year}' not available on this dataset. Available years: ${availableYears.join(", ")}`
    );
    return;
  }

  await page.selectOption("#year", { value: year });
  await page.waitForTimeout(400);
}

async function isButtonSelected(page, buttonId) {
  return page.$eval(
    `#${buttonId}`,
    (node) => node.classList.contains("btn-primary")
  );
}

async function clickButton(page, buttonId) {
  await page.click(`#${buttonId}`);
  await page.waitForTimeout(350);
}

async function applyActivityFilter(page, activities) {
  if (!activities.length) return;
  validateActivitiesWithinSingleGroup(activities);

  const groupName = findGroupName(activities[0]);
  if (!groupName) {
    throw new Error(`Cannot resolve group for activity '${activities[0]}'.`);
  }

  const groupActivities = ACTIVITY_GROUPS[groupName];
  const targetSet = new Set(activities);
  const primerButtonId = ACTIVITY_TO_BUTTON_ID[activities[0]];

  await clickButton(page, primerButtonId);

  for (const activity of groupActivities) {
    const buttonId = ACTIVITY_TO_BUTTON_ID[activity];
    const selected = await isButtonSelected(page, buttonId);
    const shouldBeSelected = targetSet.has(activity);
    if (selected !== shouldBeSelected) {
      await clickButton(page, buttonId);
    }
  }
}

function resolveScreenList(options) {
  return SCREEN_SPECS.filter((screen) => options.screens.has(screen.key)).map((screen) => {
    if (screen.key !== "detailed") return screen;
    return {
      ...screen,
      route: screen.route.replace("__DETAIL_ID__", options.detailedActivityId),
    };
  });
}

async function captureScreenshots(options) {
  fs.mkdirSync(options.outDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({
    viewport: {
      width: options.viewportWidth,
      height: options.viewportHeight,
    },
  });
  const page = await context.newPage();

  try {
    const screens = resolveScreenList(options);
    const requestedActivities = parseActivities(options.activitiesRaw);

    for (const screen of screens) {
      const targetUrl = `${options.baseUrl}${screen.route}`;
      const outputPath = path.join(options.outDir, screen.file);
      console.log(`Capturing ${screen.key}: ${targetUrl} -> ${outputPath}`);

      await page.goto(targetUrl, { waitUntil: "domcontentloaded", timeout: 60000 });
      await waitForHeader(page);

      if (screen.applyFilters) {
        await applyYearFilter(page, options.year);
        await applyActivityFilter(page, requestedActivities);
      }

      await page.waitForTimeout(options.waitMs);
      await page.screenshot({
        path: outputPath,
        fullPage: options.fullPage,
      });
    }
  } finally {
    await context.close();
    await browser.close();
  }
}

async function main() {
  try {
    const options = parseArgs(process.argv.slice(2));
    await captureScreenshots(options);
    console.log("Done.");
  } catch (error) {
    console.error(error instanceof Error ? error.message : String(error));
    process.exit(1);
  }
}

void main();
