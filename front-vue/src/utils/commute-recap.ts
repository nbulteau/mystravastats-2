import type { Activity } from "@/models/activity.model";
import type { MapTrack } from "@/models/map.model";
import { ANNUAL_RECAP_FORMATS, type AnnualRecapFormat, type AnnualRecapTheme } from "@/utils/annual-recap";

export type CommuteRecapPage = "overview" | "regularity" | "progress" | "impact" | "routes";

export const COMMUTE_RECAP_PAGES: ReadonlyArray<{ id: CommuteRecapPage; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "regularity", label: "Regularity" },
  { id: "progress", label: "Progress" },
  { id: "impact", label: "Impact" },
  { id: "routes", label: "Routes & highlights" },
];

export interface CommuteRecapStats {
  year: string;
  trips: number;
  activeDays: number;
  distanceKm: number;
  movingTimeSeconds: number;
  elevationM: number;
  averageDistanceKm: number;
  longestDistanceKm: number;
  activeWeeks: number;
  weeksInScope: number;
  longestWorkdayStreak: number;
  favoriteWeekday: string;
  busiestMonth: string;
  morningTrips: number;
  eveningTrips: number;
  weekdayCounts: number[];
  monthCounts: number[];
}

export interface CommuteImpactConfig {
  motorizedSharePercent: number;
  fuelConsumptionLitresPer100Km: number;
  fuelPricePerLitre: number;
  co2KgPerKm: number;
}

export interface CommuteImpact {
  substitutedKm: number;
  fuelLitres: number;
  fuelCost: number;
  co2Kg: number;
}

export interface CommuteRecapInput {
  page: CommuteRecapPage;
  year: string;
  athleteName?: string;
  stats: CommuteRecapStats;
  previousStats?: CommuteRecapStats;
  theme: AnnualRecapTheme;
  format: AnnualRecapFormat;
  impactEnabled: boolean;
  impactConfig: CommuteImpactConfig;
  includeMap: boolean;
  tracks?: MapTrack[];
  yearToDate?: boolean;
}

const WEEKDAY_LABELS = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const MONTH_LABELS = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

export function buildCommuteRecapStats(
  activities: Activity[],
  year: string,
  scopeDate = new Date(),
): CommuteRecapStats {
  const entries = activities
    .filter((activity) => activity.commute)
    .map((activity) => ({ activity, date: parseActivityDate(activity.date) }))
    .filter((entry): entry is { activity: Activity; date: Date } => entry.date !== null && String(entry.date.getUTCFullYear()) === year);
  const days = [...new Set(entries.map((entry) => isoDay(entry.date)))].sort();
  const weekdayCounts = Array<number>(7).fill(0);
  const monthCounts = Array<number>(12).fill(0);
  const weeks = new Set<string>();
  let morningTrips = 0;
  let eveningTrips = 0;
  for (const { date } of entries) {
    const weekdayIndex = (date.getUTCDay() + 6) % 7;
    weekdayCounts[weekdayIndex] = (weekdayCounts[weekdayIndex] ?? 0) + 1;
    monthCounts[date.getUTCMonth()] = (monthCounts[date.getUTCMonth()] ?? 0) + 1;
    weeks.add(isoWeekKey(date));
    if (date.getUTCHours() < 12) morningTrips += 1;
    else eveningTrips += 1;
  }
  const distanceKm = entries.reduce((total, { activity }) => total + positive(activity.distance) / 1000, 0);
  const movingTimeSeconds = entries.reduce((total, { activity }) => total + positive(activity.movingTime), 0);
  const elevationM = entries.reduce((total, { activity }) => total + positive(activity.totalElevationGain), 0);
  const endOfScope = year === String(scopeDate.getFullYear())
    ? new Date(Date.UTC(scopeDate.getFullYear(), scopeDate.getMonth(), scopeDate.getDate()))
    : new Date(Date.UTC(Number(year), 11, 31));
  return {
    year,
    trips: entries.length,
    activeDays: days.length,
    distanceKm,
    movingTimeSeconds,
    elevationM,
    averageDistanceKm: entries.length > 0 ? distanceKm / entries.length : 0,
    longestDistanceKm: entries.reduce((longest, { activity }) => Math.max(longest, positive(activity.distance) / 1000), 0),
    activeWeeks: weeks.size,
    weeksInScope: weeksBetween(new Date(Date.UTC(Number(year), 0, 1)), endOfScope),
    longestWorkdayStreak: workdayStreak(days),
    favoriteWeekday: labelOfMax(weekdayCounts, WEEKDAY_LABELS),
    busiestMonth: labelOfMax(monthCounts, MONTH_LABELS),
    morningTrips,
    eveningTrips,
    weekdayCounts,
    monthCounts,
  };
}

export function calculateCommuteImpact(stats: CommuteRecapStats, config: CommuteImpactConfig): CommuteImpact {
  const substitutedKm = stats.distanceKm * clamp(config.motorizedSharePercent, 0, 100) / 100;
  const fuelLitres = substitutedKm * positive(config.fuelConsumptionLitresPer100Km) / 100;
  return {
    substitutedKm,
    fuelLitres,
    fuelCost: fuelLitres * positive(config.fuelPricePerLitre),
    co2Kg: substitutedKm * positive(config.co2KgPerKm),
  };
}

export function buildCommuteHighlights(
  stats: CommuteRecapStats,
  previousStats?: CommuteRecapStats,
): string[] {
  const result = [
    `${formatNumber(stats.activeDays, 0)} commute days and ${formatNumber(stats.trips, 0)} trips`,
    `${formatNumber(stats.activeWeeks, 0)} active weeks out of ${formatNumber(stats.weeksInScope, 0)}`,
  ];
  if (stats.longestWorkdayStreak > 0) result.push(`Best streak: ${stats.longestWorkdayStreak} consecutive workdays`);
  if (stats.favoriteWeekday !== "—") result.push(`${stats.favoriteWeekday} was the favorite commute day`);
  const delta = percentageDelta(stats.distanceKm, previousStats?.distanceKm);
  if (delta !== null) result.push(`${Math.abs(delta).toFixed(0)}% ${delta >= 0 ? "more" : "less"} commute distance than the previous year`);
  return result.slice(0, 5);
}

export function buildCommuteRecapSvg(input: CommuteRecapInput): string {
  const format = ANNUAL_RECAP_FORMATS.find((candidate) => candidate.id === input.format) ?? ANNUAL_RECAP_FORMATS[0];
  const { width, height } = format;
  const palette = input.theme === "dark"
    ? { background: "#081a17", backgroundEnd: "#0d2822", panel: "#12352e", text: "#f4fbf8", muted: "#a8c4ba", line: "#285249", accent: "#2dd4a0", warning: "#fbbf24" }
    : { background: "#eef8f3", backgroundEnd: "#dcefe6", panel: "#ffffff", text: "#18332b", muted: "#61776f", line: "#cde0d8", accent: "#16865c", warning: "#a66b00" };
  const athlete = input.athleteName?.trim() || "My daily mobility";
  const contentTop = height <= 1080 ? 268 : 330;
  const footerY = height - 54;
  const contentHeight = footerY - contentTop - 44;
  const titles: Record<CommuteRecapPage, [string, string]> = {
    overview: ["Commute recap", "My year in daily motion"],
    regularity: ["Regularity", "The rhythm behind every trip"],
    progress: ["Year over year", `How ${input.year} compares`],
    impact: ["Estimated impact", "What those kilometres may have replaced"],
    routes: ["Routes & highlights", "The patterns of a year on the move"],
  };
  const [kicker, title] = titles[input.page];
  const cardIndex = COMMUTE_RECAP_PAGES.findIndex((page) => page.id === input.page) + 1;
  let content = "";
  switch (input.page) {
    case "overview": content = overviewContent(input, 64, contentTop, width - 128, contentHeight, palette); break;
    case "regularity": content = regularityContent(input, 64, contentTop, width - 128, contentHeight, palette); break;
    case "progress": content = progressContent(input, 64, contentTop, width - 128, contentHeight, palette); break;
    case "impact": content = impactContent(input, 64, contentTop, width - 128, contentHeight, palette); break;
    case "routes": content = routesContent(input, 64, contentTop, width - 128, contentHeight, palette); break;
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-labelledby="commute-title commute-description"><title id="commute-title">${escapeXml(input.year)} commute recap</title><desc id="commute-description">${escapeXml(title)} for ${escapeXml(athlete)}</desc><defs><linearGradient id="commute-background" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${palette.background}"/><stop offset="1" stop-color="${palette.backgroundEnd}"/></linearGradient></defs><rect width="${width}" height="${height}" fill="url(#commute-background)"/><path d="M-80 170 C220 30 400 300 720 130 S1120 30 1210 190" fill="none" stroke="${palette.accent}" stroke-width="28" opacity=".08"/><g font-family="Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"><text x="64" y="88" fill="${palette.accent}" font-size="21" font-weight="850" letter-spacing="3">${escapeXml(kicker.toUpperCase())} · ${escapeXml(input.year)}</text><text x="64" y="175" fill="${palette.text}" font-size="56" font-weight="900" letter-spacing="-2">${escapeXml(title)}</text><text x="64" y="220" fill="${palette.muted}" font-size="22">${escapeXml(athlete)} · explicitly tagged commute activities</text>${content}<text x="64" y="${footerY}" fill="${palette.muted}" font-size="17">Generated locally · commute card ${cardIndex} of ${COMMUTE_RECAP_PAGES.length}</text><text x="${width - 64}" y="${footerY}" text-anchor="end" fill="${palette.accent}" font-size="19" font-weight="850">#commuterecap</text></g></svg>`;
}

type Palette = { panel: string; text: string; muted: string; line: string; accent: string; warning: string };

function overviewContent(input: CommuteRecapInput, x: number, y: number, width: number, height: number, palette: Palette): string {
  const values: Array<[string, string]> = [
    ["TRIPS", formatNumber(input.stats.trips, 0)], ["COMMUTE DAYS", formatNumber(input.stats.activeDays, 0)],
    ["DISTANCE", `${formatNumber(input.stats.distanceKm, 0)} km`], ["MOVING TIME", formatDuration(input.stats.movingTimeSeconds)],
    ["AVERAGE TRIP", `${formatNumber(input.stats.averageDistanceKm, 1)} km`], ["ELEVATION", `${formatNumber(input.stats.elevationM, 0)} m`],
  ];
  return metricGrid(values, x, y, width, height, palette);
}

function regularityContent(input: CommuteRecapInput, x: number, y: number, width: number, height: number, palette: Palette): string {
  const barTop = 180;
  const barHeight = Math.max(120, height - barTop - 150);
  const maxCount = Math.max(1, ...input.stats.weekdayCounts);
  const barWidth = (width - 80) / 7;
  const bars = input.stats.weekdayCounts.map((count, index) => {
    const visualHeight = barHeight * count / maxCount;
    const bx = 40 + index * barWidth;
    return `<rect x="${bx}" y="${barTop + barHeight - visualHeight}" width="${barWidth - 12}" height="${visualHeight}" rx="10" fill="${palette.accent}" opacity="${count === maxCount ? 1 : .42}"/><text x="${bx + (barWidth - 12) / 2}" y="${barTop + barHeight + 30}" text-anchor="middle" fill="${palette.muted}" font-size="17">${WEEKDAY_LABELS[index]}</text><text x="${bx + (barWidth - 12) / 2}" y="${barTop + barHeight - visualHeight - 10}" text-anchor="middle" fill="${palette.text}" font-size="17" font-weight="800">${count}</text>`;
  }).join("");
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="30" fill="${palette.panel}" stroke="${palette.line}"/><text x="35" y="52" fill="${palette.text}" font-size="38" font-weight="850">${input.stats.activeWeeks} active weeks</text><text x="35" y="88" fill="${palette.muted}" font-size="19">out of ${input.stats.weeksInScope} · best workday streak: ${input.stats.longestWorkdayStreak}</text><text x="${width - 35}" y="52" text-anchor="end" fill="${palette.text}" font-size="30" font-weight="800">${input.stats.favoriteWeekday}</text><text x="${width - 35}" y="84" text-anchor="end" fill="${palette.muted}" font-size="17">favorite day · busiest month ${input.stats.busiestMonth}</text>${bars}<text x="35" y="${height - 38}" fill="${palette.muted}" font-size="19">${input.stats.morningTrips} morning trips · ${input.stats.eveningTrips} afternoon/evening trips</text></g>`;
}

function progressContent(input: CommuteRecapInput, x: number, y: number, width: number, height: number, palette: Palette): string {
  const previous = input.previousStats;
  const rows: Array<[string, number, number, string]> = [
    ["Distance", input.stats.distanceKm, previous?.distanceKm ?? 0, "km"], ["Trips", input.stats.trips, previous?.trips ?? 0, ""],
    ["Commute days", input.stats.activeDays, previous?.activeDays ?? 0, ""], ["Active weeks", input.stats.activeWeeks, previous?.activeWeeks ?? 0, ""],
  ];
  const gap = 16;
  const rowHeight = (height - gap * 3) / 4;
  const note = input.yearToDate ? `${input.year} to date vs full ${Number(input.year) - 1} totals` : `Full-year totals vs ${Number(input.year) - 1}`;
  const cards = rows.map(([label, current, old, unit], index) => {
    const delta = percentageDelta(current, old);
    return `<g transform="translate(0 ${index * (rowHeight + gap)})"><rect width="${width}" height="${rowHeight}" rx="24" fill="${palette.panel}" stroke="${palette.line}"/><text x="28" y="40" fill="${palette.muted}" font-size="18" font-weight="800">${label.toUpperCase()}</text><text x="28" y="${rowHeight - 28}" fill="${palette.text}" font-size="42" font-weight="850">${formatNumber(current, label === "Distance" ? 0 : 0)}${unit ? ` ${unit}` : ""}</text><text x="${width - 28}" y="${rowHeight - 28}" text-anchor="end" fill="${delta !== null && delta >= 0 ? palette.accent : palette.warning}" font-size="28" font-weight="850">${delta === null ? "No previous data" : `${delta >= 0 ? "+" : ""}${delta.toFixed(0)}%`}</text></g>`;
  }).join("");
  return `<g transform="translate(${x} ${y})"><text x="0" y="-16" fill="${palette.muted}" font-size="17">${note}</text>${cards}</g>`;
}

function impactContent(input: CommuteRecapInput, x: number, y: number, width: number, height: number, palette: Palette): string {
  if (!input.impactEnabled) {
    return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="30" fill="${palette.panel}" stroke="${palette.line}"/><circle cx="${width / 2}" cy="${height / 2 - 60}" r="64" fill="${palette.accent}" opacity=".12"/><path d="M${width / 2 - 28} ${height / 2 - 60}h56M${width / 2} ${height / 2 - 88}v56" stroke="${palette.accent}" stroke-width="9" stroke-linecap="round"/><text x="${width / 2}" y="${height / 2 + 50}" text-anchor="middle" fill="${palette.text}" font-size="35" font-weight="850">Impact estimates are off by default</text><text x="${width / 2}" y="${height / 2 + 92}" text-anchor="middle" fill="${palette.muted}" font-size="19">Enable them only if these commutes replaced motorized trips.</text></g>`;
  }
  const impact = calculateCommuteImpact(input.stats, input.impactConfig);
  const values: Array<[string, string]> = [
    ["POTENTIALLY REPLACED", `${formatNumber(impact.substitutedKm, 0)} km`], ["ESTIMATED CO₂", `${formatNumber(impact.co2Kg, 0)} kg`],
    ["ESTIMATED FUEL", `${formatNumber(impact.fuelLitres, 1)} L`], ["ESTIMATED FUEL COST", `${formatNumber(impact.fuelCost, 0)} €`],
  ];
  const grid = metricGrid(values, 0, 0, width, height - 70, palette);
  return `<g transform="translate(${x} ${y})">${grid}<text x="0" y="${height - 28}" fill="${palette.muted}" font-size="16">Estimate · ${input.impactConfig.motorizedSharePercent}% substituted · ${input.impactConfig.fuelConsumptionLitresPer100Km} L/100 km · ${input.impactConfig.co2KgPerKm} kg CO₂/km</text></g>`;
}

function routesContent(input: CommuteRecapInput, x: number, y: number, width: number, height: number, palette: Palette): string {
  const highlights = buildCommuteHighlights(input.stats, input.previousStats).slice(0, 3);
  const headerHeight = 180;
  const lines = highlights.map((text, index) => `<text x="35" y="${48 + index * 42}" fill="${index === 0 ? palette.text : palette.muted}" font-size="${index === 0 ? 25 : 20}" font-weight="${index === 0 ? 800 : 650}">${escapeXml(text)}</text>`).join("");
  const fingerprint = input.includeMap
    ? fingerprintSvg(input.tracks ?? [], 0, headerHeight + 16, width, height - headerHeight - 16, palette)
    : privacySvg(0, headerHeight + 16, width, height - headerHeight - 16, palette);
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${headerHeight}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/>${lines}<text x="${width - 35}" y="48" text-anchor="end" fill="${palette.accent}" font-size="24" font-weight="850">${formatNumber(input.stats.longestDistanceKm, 1)} km longest</text>${fingerprint}</g>`;
}

function metricGrid(values: Array<[string, string]>, x: number, y: number, width: number, height: number, palette: Palette): string {
  const columns = 2;
  const rows = Math.ceil(values.length / columns);
  const gap = 16;
  const cellWidth = (width - gap) / 2;
  const cellHeight = (height - gap * (rows - 1)) / rows;
  const cells = values.map(([label, value], index) => `<g transform="translate(${(index % 2) * (cellWidth + gap)} ${Math.floor(index / 2) * (cellHeight + gap)})"><rect width="${cellWidth}" height="${cellHeight}" rx="26" fill="${palette.panel}" stroke="${palette.line}"/><text x="28" y="43" fill="${palette.muted}" font-size="17" font-weight="800" letter-spacing="2">${label}</text><text x="28" y="${cellHeight - 32}" fill="${palette.text}" font-size="43" font-weight="850">${value}</text></g>`).join("");
  return `<g transform="translate(${x} ${y})">${cells}</g>`;
}

function fingerprintSvg(tracks: MapTrack[], x: number, y: number, width: number, height: number, palette: Palette): string {
  const candidates = tracks.map((track) => track.coordinates.filter(validCoordinate)).filter((points) => points.length >= 2).slice(0, 80);
  if (candidates.length === 0) return privacySvg(x, y, width, height, palette, "No commute GPS traces available");
  const paths = candidates.map((coordinates, trackIndex) => {
    const step = Math.max(1, Math.ceil(coordinates.length / 220));
    const sampled = coordinates.filter((_, index) => index % step === 0 || index === coordinates.length - 1);
    const bounds = sampled.reduce((result, [lat, lng]) => ({ minLat: Math.min(result.minLat, lat), maxLat: Math.max(result.maxLat, lat), minLng: Math.min(result.minLng, lng), maxLng: Math.max(result.maxLng, lng) }), { minLat: Infinity, maxLat: -Infinity, minLng: Infinity, maxLng: -Infinity });
    const scale = Math.min((width - 70) / Math.max(bounds.maxLng - bounds.minLng, .000001), (height - 100) / Math.max(bounds.maxLat - bounds.minLat, .000001));
    const usedWidth = (bounds.maxLng - bounds.minLng) * scale;
    const usedHeight = (bounds.maxLat - bounds.minLat) * scale;
    const offsetX = 35 + (width - 70 - usedWidth) / 2;
    const offsetY = 60 + (height - 90 - usedHeight) / 2;
    const path = sampled.map(([lat, lng], index) => `${index === 0 ? "M" : "L"}${(offsetX + (lng - bounds.minLng) * scale).toFixed(1)} ${(offsetY + (bounds.maxLat - lat) * scale).toFixed(1)}`).join(" ");
    return `<path d="${path}" fill="none" stroke="${palette.accent}" stroke-width="${trackIndex % 8 === 0 ? 3 : 2}" opacity="${trackIndex % 8 === 0 ? .5 : .14}" stroke-linecap="round"/>`;
  }).join("");
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/><text x="28" y="38" fill="${palette.muted}" font-size="16" font-weight="800">ANONYMISED COMMUTE FINGERPRINT · ${candidates.length} TRACES</text>${paths}</g>`;
}

function privacySvg(x: number, y: number, width: number, height: number, palette: Palette, message = "Route fingerprint hidden for privacy"): string {
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/><text x="${width / 2}" y="${height / 2}" text-anchor="middle" fill="${palette.muted}" font-size="23" font-weight="750">${escapeXml(message)}</text></g>`;
}

function parseActivityDate(value: string): Date | null {
  const match = /^(\d{4})-(\d{2})-(\d{2})(?:T(\d{2}):(\d{2}))?/.exec(value);
  if (!match) return null;
  const date = new Date(Date.UTC(Number(match[1]), Number(match[2]) - 1, Number(match[3]), Number(match[4] ?? 0), Number(match[5] ?? 0)));
  return Number.isNaN(date.getTime()) ? null : date;
}

function isoDay(date: Date): string { return date.toISOString().slice(0, 10); }
function isoWeekKey(date: Date): string {
  const copy = new Date(Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()));
  const day = copy.getUTCDay() || 7;
  copy.setUTCDate(copy.getUTCDate() + 4 - day);
  const yearStart = new Date(Date.UTC(copy.getUTCFullYear(), 0, 1));
  const week = Math.ceil((((copy.getTime() - yearStart.getTime()) / 86400000) + 1) / 7);
  return `${copy.getUTCFullYear()}-${String(week).padStart(2, "0")}`;
}
function weeksBetween(start: Date, end: Date): number {
  if (end < start) return 0;
  const weeks = new Set<string>();
  const cursor = new Date(start);
  while (cursor <= end) { weeks.add(isoWeekKey(cursor)); cursor.setUTCDate(cursor.getUTCDate() + 1); }
  return weeks.size;
}
function workdayStreak(days: string[]): number {
  const workdays = days.map((day) => new Date(`${day}T00:00:00Z`)).filter((date) => date.getUTCDay() >= 1 && date.getUTCDay() <= 5);
  let best = 0;
  let current = 0;
  let previous: Date | null = null;
  for (const date of workdays) {
    const expected = previous ? nextWorkday(previous) : null;
    current = expected && isoDay(expected) === isoDay(date) ? current + 1 : 1;
    best = Math.max(best, current);
    previous = date;
  }
  return best;
}
function nextWorkday(value: Date): Date {
  const next = new Date(value);
  do next.setUTCDate(next.getUTCDate() + 1); while (next.getUTCDay() === 0 || next.getUTCDay() === 6);
  return next;
}
function labelOfMax(values: number[], labels: readonly string[]): string {
  const max = Math.max(0, ...values);
  return max > 0 ? labels[values.indexOf(max)] ?? "—" : "—";
}
function percentageDelta(current: number, previous: number | undefined): number | null { return previous && previous > 0 ? ((current - previous) / previous) * 100 : null; }
function validCoordinate(value: number[]): value is [number, number] { return value.length >= 2 && Number.isFinite(value[0]) && Number.isFinite(value[1]) && Math.abs(value[0]) <= 90 && Math.abs(value[1]) <= 180; }
function positive(value: number): number { return Number.isFinite(value) && value > 0 ? value : 0; }
function clamp(value: number, min: number, max: number): number { return Math.min(max, Math.max(min, Number.isFinite(value) ? value : min)); }
function formatNumber(value: number, digits: number): string { return positive(value).toLocaleString("en-US", { maximumFractionDigits: digits }); }
function formatDuration(seconds: number): string { return `${Math.round(positive(seconds) / 3600).toLocaleString("en-US")} h`; }
function escapeXml(value: string): string { return value.replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", "\"": "&quot;", "'": "&apos;" })[char] ?? char); }
