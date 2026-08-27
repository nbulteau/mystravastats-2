import { expect, test, type Page, type Route } from "@playwright/test";

const dataQualityIssue = {
  id: "distance-spike-42",
  source: "FIT",
  activityId: 42,
  activityName: "Morning Ride",
  activityType: "Ride",
  year: "2026",
  severity: "warning",
  category: "distance",
  field: "distance",
  message: "Distance spike detected",
  excludedFromStats: false,
  correction: { available: true, safety: "safe", type: "distance-spike", description: "Remove the spike" },
};

const correction = {
  id: "correction-42",
  issueId: dataQualityIssue.id,
  source: "FIT",
  activityId: 42,
  activityName: "Morning Ride",
  activityType: "Ride",
  year: "2026",
  type: "distance-spike",
  safety: "safe",
  status: "active",
  pointIndexes: [2],
  modifiedFields: ["distance"],
  impact: { distanceDeltaMeters: -120, elevationDeltaMeters: 0 },
};

const dataQualityReport = {
  generatedAt: "2026-08-27T08:00:00Z",
  summary: {
    status: "warning",
    provider: "FIT",
    issueCount: 1,
    impactedActivities: 1,
    excludedActivities: 0,
    correctionCount: 0,
    safeCorrectionCount: 1,
    manualReviewCount: 0,
    bySeverity: { warning: 1 },
    byCategory: { distance: 1 },
    topIssues: [dataQualityIssue],
  },
  issues: [dataQualityIssue],
  exclusions: [],
  corrections: [],
};

const health = {
  timestamp: "2026-08-27T08:00:00Z",
  provider: "FIT",
  athleteId: "123",
  cacheRoot: "/data/fit",
  activities: 1,
  availableYearBins: ["2026"],
  refresh: { backgroundInProgress: false },
  runtimeConfig: {},
  files: {},
  routing: {
    status: "up",
    reachable: true,
    engine: "osrm",
    extractProfile: "bicycle.lua",
    effectiveProfile: "cycling",
    supportedRouteTypes: ["RIDE", "MTB", "GRAVEL"],
  },
  dataQuality: dataQualityReport.summary,
};

const dashboard = {
  nbActivitiesByYear: { "2026": 1 },
  activeDaysByYear: { "2026": 1 },
  consistencyByYear: { "2026": 1 },
  movingTimeByYear: { "2026": 5100 },
  totalDistanceByYear: { "2026": 42.5 },
  averageDistanceByYear: { "2026": 42.5 },
  maxDistanceByYear: { "2026": 42.5 },
  maxDistanceDateByYear: { "2026": "2026-08-27" },
  averageDistanceByActiveDayByYear: { "2026": 42.5 },
  maxDistanceByActiveDayByYear: { "2026": 42.5 },
  maxDistanceByActiveDayDateByYear: { "2026": "2026-08-27" },
  totalElevationByYear: { "2026": 520 },
  averageElevationByYear: { "2026": 520 },
  maxElevationByYear: { "2026": 520 },
  maxElevationDateByYear: { "2026": "2026-08-27" },
  averageElevationByActiveDayByYear: { "2026": 520 },
  maxElevationByActiveDayByYear: { "2026": 520 },
  maxElevationByActiveDayDateByYear: { "2026": "2026-08-27" },
  elevationEfficiencyByYear: { "2026": 12.2 },
  averageSpeedByYear: { "2026": 29.9 },
  maxSpeedByYear: { "2026": 54.7 },
  maxSpeedDateByYear: { "2026": "2026-08-27" },
  averageHeartRateByYear: { "2026": 142 },
  maxHeartRateByYear: { "2026": 171 },
  maxHeartRateDateByYear: { "2026": "2026-08-27" },
  averageWattsByYear: { "2026": 186 },
  maxWattsByYear: { "2026": 640 },
  maxWattsDateByYear: { "2026": "2026-08-27" },
  deviceAverageWattsByYear: { "2026": 186 },
  deviceMaxWattsByYear: { "2026": 640 },
  deviceMaxWattsDateByYear: { "2026": "2026-08-27" },
  averageCadenceByYear: [[2026, 82]],
};

const detailedActivity = {
  id: 42,
  name: "Morning Ride",
  type: "Ride",
  sportType: "Ride",
  commute: false,
  link: "",
  distance: 42500,
  elapsedTime: 5400,
  movingTime: 5100,
  totalElevationGain: 520,
  totalDescent: 520,
  averageSpeed: 8.3,
  averageCadence: 82,
  averageHeartrate: 142,
  maxHeartrate: 171,
  averageWatts: 186,
  weightedAverageWatts: 201,
  deviceWatts: true,
  kilojoules: 950,
  maxSpeed: 15.2,
  maxWatts: 640,
  elevHigh: 620,
  startDate: "2026-08-27T06:00:00Z",
  startDateLocal: "2026-08-27T08:00:00+02:00",
  startLatlng: [48.85, 2.35],
  stream: {
    distance: [0, 1000, 2000],
    time: [0, 120, 240],
    heartrate: [120, 140, 150],
    cadence: [80, 82, 84],
    moving: [true, true, true],
    altitude: [45, 52, 60],
    latlng: [[48.85, 2.35], [48.86, 2.36], [48.87, 2.37]],
    watts: [170, 190, 205],
  },
  source: { primaryProvider: "FIT", primaryId: 42, streamProvider: "FIT", sources: [], conflicts: [], fieldSources: {} },
  activityEfforts: [],
  stravaSegmentEfforts: [],
};

test.beforeEach(async ({ page }) => {
  await installApiMocks(page);
});

test("dashboard and activity detail render from the public API", async ({ page }) => {
  await page.goto("/dashboard?year=2026&activityType=Ride");
  await expect(page.getByText("Create your annual recap")).toBeVisible();
  await expect(page.getByRole("heading", { name: "Eddington number" })).toBeVisible();

  await page.goto("/activities/42");
  await expect(page.getByRole("heading", { name: "Morning Ride" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Activity Summary" })).toBeVisible();
  await expect(page.getByRole("heading", { name: "Data & Source" })).toBeVisible();
});

test("source onboarding previews, saves and synchronizes a FIT directory", async ({ page }) => {
  let previewCalls = 0;
  let applyCalls = 0;
  let synchronizeCalls = 0;
  page.on("request", (request) => {
    const url = new URL(request.url());
    if (request.method() === "POST" && url.pathname === "/api/source-modes/preview") previewCalls += 1;
    if (request.method() === "POST" && url.pathname === "/api/source-modes/apply") applyCalls += 1;
    if (request.method() === "POST" && url.pathname === "/api/source-sync/synchronize") synchronizeCalls += 1;
  });

  await page.goto("/diagnostics");
  await expect(page.getByRole("heading", { name: "System Status" })).toBeVisible();
  await page.getByText("Change data source", { exact: true }).click();
  await page.getByRole("tab", { name: "FIT" }).click();
  await page.locator(".source-path-field input").fill("/data/fit");
  await page.getByRole("button", { name: "Check directory" }).click();
  await expect.poll(() => previewCalls).toBeGreaterThanOrEqual(1);
  await page.getByRole("button", { name: "Use this source" }).click();
  await expect.poll(() => applyCalls).toBe(1);
  await expect(page.getByText(/FIT source saved/)).toBeVisible();

  await page.getByRole("button", { name: "Synchronize", exact: true }).click();
  await expect.poll(() => synchronizeCalls).toBe(1);
});

test("safe data-quality correction is reviewed before it is applied", async ({ page }) => {
  let applyCalls = 0;
  page.on("request", (request) => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/data-quality/corrections/safe") applyCalls += 1;
  });

  await page.goto("/diagnostics");
  await page.getByRole("button", { name: /Review safe fixes/ }).click();
  await expect(page.getByRole("heading", { name: "Review safe fixes" })).toBeVisible();
  await page.getByRole("button", { name: "Apply safe fixes" }).click();
  await expect.poll(() => applyCalls).toBe(1);
  await expect(page.getByText("Safe local corrections applied.")).toBeVisible();
});

test("GPS Art drawing generates and selects an OSRM proposal", async ({ page }) => {
  let generationCalls = 0;
  page.on("request", (request) => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/routes/generate/shape") generationCalls += 1;
  });

  await page.goto("/routes");
  await expect(page.getByRole("heading", { name: "GPS Art" })).toBeVisible();
  await page.getByRole("button", { name: "Draw", exact: true }).click();
  const map = page.locator(".routes-map");
  await map.click({ position: { x: 250, y: 220 } });
  await map.click({ position: { x: 330, y: 260 } });
  await page.locator(".routes-map-generate-btn").click();
  await expect.poll(() => generationCalls).toBe(1);
  await expect(page.getByText("Generated route", { exact: true })).toBeVisible();
  await expect(page.getByText("Mock Art Loop")).toBeVisible();
  await page.waitForTimeout(700);
});

async function installApiMocks(page: Page) {
  await page.route("**/api/**", async (route) => fulfillApiRoute(route));
}

async function fulfillApiRoute(route: Route) {
  const request = route.request();
  const { pathname } = new URL(request.url());
  const json = (body: unknown, status = 200) => route.fulfill({
    status,
    contentType: "application/json",
    body: JSON.stringify(body),
  });

  if (pathname === "/api/athletes/me") return json({ firstname: "Ada", lastname: "Rider", weight: 62, ftp: 235 });
  if (pathname === "/api/athletes/me/performance-settings") return json({ weightKg: 62, manualFtpBySport: {} });
  if (pathname === "/api/athletes/me/heart-rate-zones") return json({ maxHr: 185, thresholdHr: 170, reserveHr: 130 });
  if (pathname === "/api/health/details") return json(health);
  if (pathname === "/api/dashboard") return json(dashboard);
  if (pathname === "/api/dashboard/cumulative-data-per-year") return json({ distance: { "2026": { "2026-08-27": 42.5 } }, elevation: { "2026": { "2026-08-27": 520 } } });
  if (pathname === "/api/dashboard/eddington-number") return json({ number: 1, nextTarget: 2, qualifyingCount: 1, progress: 0.5, distribution: [] });
  if (pathname === "/api/dashboard/activity-heatmap") return json({});
  if (pathname === "/api/statistics/heart-rate-zones") return json({ distributions: [], activities: [], periods: [] });
  if (pathname === "/api/activities/42") return json(detailedActivity);
  if (pathname === "/api/data-quality/issues") return json(dataQualityReport);
  if (pathname === "/api/data-quality/corrections/safe/preview") return json({ generatedAt: "2026-08-27T08:00:00Z", mode: "safe", summary: { safeCorrectionCount: 1, manualReviewCount: 0, unsupportedIssueCount: 0, activityCount: 1, distanceDeltaMeters: -120, elevationDeltaMeters: 0, modifiedFields: ["distance"], potentiallyImpactsRecords: false }, corrections: [correction], warnings: [], blockingReasons: [] });
  if (pathname === "/api/data-quality/corrections/safe") return json({ ...dataQualityReport, summary: { ...dataQualityReport.summary, safeCorrectionCount: 0, correctionCount: 1 }, corrections: [correction] });
  if (pathname === "/api/source-sync/synchronize") return json({ status: "ok", message: "Synchronized", reloaded: true, completedAt: "2026-08-27T08:01:00Z" });
  if (pathname === "/api/source-modes/preview") return json(sourcePreview(false));
  if (pathname === "/api/source-modes/apply") return json({ status: "ok", message: "Saved", envFile: ".env", restartNeeded: true, preview: sourcePreview(false) });
  if (pathname === "/api/routes/generate/shape") return json({ routes: [{ routeId: "mock-art-loop", title: "Mock Art Loop", variantType: "SHAPE_MATCH", routeType: "RIDE", distanceKm: 12.4, elevationGainM: 180, durationSec: 2700, estimatedDurationSec: 2700, score: { global: 88, distance: 90, elevation: 80, duration: 85, direction: 100, shape: 92, roadFitness: 86 }, reasons: ["SHAPE_MODE:draw"], previewLatLng: [[48.85, 2.35], [48.86, 2.36], [48.87, 2.35]], start: { lat: 48.85, lng: 2.35 }, end: { lat: 48.87, lng: 2.35 }, isRoadGraphGenerated: true }], diagnostics: [] });
  if (pathname.startsWith("/api/activities")) return json([]);
  return json({});
}

function sourcePreview(active: boolean) {
  return { mode: "FIT", activeMode: "FIT", path: "/data/fit", configKey: "FIT_FILES_PATH", supported: true, active, configured: true, readable: true, validStructure: true, restartNeeded: !active, activationCommand: "FIT_FILES_PATH=/data/fit", fileCount: 1, validFileCount: 1, invalidFileCount: 0, activityCount: 1, years: [{ year: "2026", fileCount: 1, validFileCount: 1, activityCount: 1 }], missingFields: [], environment: [{ key: "FIT_FILES_PATH", value: "/data/fit", required: true }], errors: [], recommendations: [], stravaOAuth: null };
}
