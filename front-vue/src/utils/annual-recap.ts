import type { MapTrack } from "@/models/map.model";

export type AnnualRecapTheme = "light" | "dark";
export type AnnualRecapFormat = "portrait" | "square" | "story";
export type AnnualRecapPage = "overview" | "progress" | "consistency" | "exploration" | "highlights";

export const ANNUAL_RECAP_PAGES: ReadonlyArray<{ id: AnnualRecapPage; label: string }> = [
  { id: "overview", label: "Overview" },
  { id: "progress", label: "Progress" },
  { id: "consistency", label: "Consistency" },
  { id: "exploration", label: "Explore" },
  { id: "highlights", label: "Highlights" },
];

export const ANNUAL_RECAP_FORMATS: ReadonlyArray<{
  id: AnnualRecapFormat;
  label: string;
  width: number;
  height: number;
}> = [
  { id: "portrait", label: "Portrait · 4:5", width: 1080, height: 1350 },
  { id: "square", label: "Square · 1:1", width: 1080, height: 1080 },
  { id: "story", label: "Story · 9:16", width: 1080, height: 1920 },
];

export interface AnnualRecapMetrics {
  activities: number;
  activeDays: number;
  distanceKm: number;
  elevationM: number;
  movingTimeSeconds: number;
  longestActivityKm: number;
  longestActivityDate?: string;
}

export interface AnnualRecapInput {
  year: string;
  athleteName?: string;
  activityLabel: string;
  metrics: AnnualRecapMetrics;
  theme: AnnualRecapTheme;
  format: AnnualRecapFormat;
  includeMap: boolean;
  tracks?: MapTrack[];
  page?: AnnualRecapPage;
  previousMetrics?: AnnualRecapMetrics;
  consistencyPercent?: number;
  previousConsistencyPercent?: number;
  highlights?: string[];
  yearToDate?: boolean;
  daysInScope?: number;
}

export function buildAnnualRecapSvg(input: AnnualRecapInput): string {
  if (input.page && input.page !== "overview") {
    return buildSecondaryAnnualRecapSvg(input, input.page);
  }
  const format = ANNUAL_RECAP_FORMATS.find((candidate) => candidate.id === input.format)
    ?? ANNUAL_RECAP_FORMATS[0];
  const { width, height } = format;
  const dark = input.theme === "dark";
  const palette = dark
    ? { background: "#111827", panel: "#1f2937", text: "#f9fafb", muted: "#aeb7c5", line: "#374151", accent: "#fc4c02", accentSoft: "#432519" }
    : { background: "#f7f4ef", panel: "#ffffff", text: "#20242b", muted: "#6d7480", line: "#e4ded6", accent: "#e8490b", accentSoft: "#fff0e8" };
  const compact = height <= 1080;
  const headerY = compact ? 82 : 100;
  const yearY = compact ? 190 : 225;
  const metricsY = compact ? 310 : 390;
  const metricHeight = compact ? 155 : 175;
  const metricGap = 18;
  const metricWidth = (width - 128 - metricGap) / 2;
  const mapY = metricsY + (metricHeight * 3) + (metricGap * 2) + (compact ? 36 : 48);
  const footerY = height - 54;
  const mapHeight = Math.max(120, footerY - mapY - 44);
  const athlete = input.athleteName?.trim() || "My activity year";
  const metrics = [
    ["DISTANCE", formatNumber(input.metrics.distanceKm, 0), "km"],
    ["ELEVATION", formatNumber(input.metrics.elevationM, 0), "m"],
    ["MOVING TIME", formatDuration(input.metrics.movingTimeSeconds), ""],
    ["ACTIVITIES", formatNumber(input.metrics.activities, 0), "sessions"],
    ["ACTIVE DAYS", formatNumber(input.metrics.activeDays, 0), "days"],
    ["LONGEST", formatNumber(input.metrics.longestActivityKm, 1), "km"],
  ];
  const metricCards = metrics.map(([label, value, unit], index) => {
    const column = index % 2;
    const row = Math.floor(index / 2);
    const x = 64 + column * (metricWidth + metricGap);
    const y = metricsY + row * (metricHeight + metricGap);
    return `
      <g transform="translate(${x} ${y})">
        <rect width="${metricWidth}" height="${metricHeight}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/>
        <text x="30" y="46" fill="${palette.muted}" font-size="19" font-weight="750" letter-spacing="2.4">${label}</text>
        <text x="30" y="112" fill="${palette.text}" font-size="48" font-weight="850">${value}</text>
        ${unit ? `<text x="${30 + Math.min(340, String(value).length * 29)}" y="112" fill="${palette.muted}" font-size="23" font-weight="700">${unit}</text>` : ""}
      </g>`;
  }).join("");
  const map = input.includeMap
    ? buildMapSvg(input.tracks ?? [], 64, mapY, width - 128, mapHeight, palette)
    : buildPrivacyPanel(64, mapY, width - 128, mapHeight, palette);
  const longestDate = formatDate(input.metrics.longestActivityDate);

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-labelledby="recap-title recap-description">
    <title id="recap-title">${escapeXml(input.year)} activity recap</title>
    <desc id="recap-description">Annual activity statistics for ${escapeXml(athlete)}</desc>
    <defs>
      <linearGradient id="recap-background" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0" stop-color="${palette.background}"/>
        <stop offset="1" stop-color="${dark ? "#0b1220" : "#eee7dd"}"/>
      </linearGradient>
    </defs>
    <rect width="${width}" height="${height}" fill="url(#recap-background)"/>
    <circle cx="1000" cy="70" r="250" fill="${palette.accent}" opacity=".08"/>
    <circle cx="20" cy="${height - 30}" r="210" fill="${palette.accent}" opacity=".06"/>
    <g font-family="Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif">
      <text x="64" y="${headerY}" fill="${palette.accent}" font-size="22" font-weight="850" letter-spacing="3.2">MYSTRAVASTATS · YEAR IN SPORT</text>
      <text x="64" y="${yearY}" fill="${palette.text}" font-size="118" font-weight="900" letter-spacing="-5">${escapeXml(input.year)}</text>
      <text x="${width - 64}" y="${yearY - 35}" text-anchor="end" fill="${palette.text}" font-size="28" font-weight="800">${escapeXml(athlete)}</text>
      <text x="${width - 64}" y="${yearY + 6}" text-anchor="end" fill="${palette.muted}" font-size="21" font-weight="650">${escapeXml(input.activityLabel)}</text>
      ${longestDate ? `<text x="64" y="${metricsY - 30}" fill="${palette.muted}" font-size="18">Longest activity · ${escapeXml(longestDate)}</text>` : ""}
      ${metricCards}
      ${map}
      <text x="64" y="${footerY}" fill="${palette.muted}" font-size="17">Generated locally · precise coordinates are never printed</text>
      <text x="${width - 64}" y="${footerY}" text-anchor="end" fill="${palette.accent}" font-size="19" font-weight="850">#mystravastats</text>
    </g>
  </svg>`;
}

export function buildAnnualRecapHighlights(
  metrics: AnnualRecapMetrics,
  previousMetrics?: AnnualRecapMetrics,
  consistencyPercent = 0,
  yearToDate = false,
): string[] {
  const highlights: string[] = [];
  const distanceDelta = percentageDelta(metrics.distanceKm, previousMetrics?.distanceKm);
  if (distanceDelta !== null) {
    highlights.push(yearToDate
      ? `Distance so far is ${Math.abs(distanceDelta).toFixed(0)}% ${distanceDelta >= 0 ? "above" : "below"} last year's total`
      : `${Math.abs(distanceDelta).toFixed(0)}% ${distanceDelta >= 0 ? "more" : "less"} distance than the previous year`);
  }
  highlights.push(`${formatNumber(metrics.activeDays, 0)} active days across ${formatNumber(metrics.activities, 0)} activities`);
  if (metrics.elevationM >= 8849) {
    highlights.push(`Climbed the equivalent of ${(metrics.elevationM / 8849).toFixed(1)} Everests`);
  } else if (metrics.elevationM > 0) {
    highlights.push(`${formatNumber(metrics.elevationM, 0)} metres of elevation collected`);
  }
  if (metrics.longestActivityKm > 0) {
    highlights.push(`Longest activity: ${formatNumber(metrics.longestActivityKm, 1)} km`);
  }
  if (consistencyPercent > 0) {
    highlights.push(`${formatNumber(consistencyPercent, 1)}% consistency across the year`);
  }
  return highlights.slice(0, 5);
}

function buildSecondaryAnnualRecapSvg(input: AnnualRecapInput, page: Exclude<AnnualRecapPage, "overview">): string {
  const format = ANNUAL_RECAP_FORMATS.find((candidate) => candidate.id === input.format) ?? ANNUAL_RECAP_FORMATS[0];
  const { width, height } = format;
  const dark = input.theme === "dark";
  const palette = dark
    ? { background: "#111827", panel: "#1f2937", text: "#f9fafb", muted: "#aeb7c5", line: "#374151", accent: "#fc4c02", positive: "#4ade80", negative: "#fb7185" }
    : { background: "#f7f4ef", panel: "#ffffff", text: "#20242b", muted: "#6d7480", line: "#e4ded6", accent: "#e8490b", positive: "#16865c", negative: "#c43d52" };
  const athlete = input.athleteName?.trim() || "My activity year";
  const contentTop = height <= 1080 ? 270 : 330;
  const footerY = height - 54;
  const titles: Record<typeof page, [string, string]> = {
    progress: ["Year over year", `How ${input.year} compares`],
    consistency: ["Consistency", "Showing up is the real record"],
    exploration: ["Exploration", "Every trace tells part of the story"],
    highlights: ["Highlights", "The moments that defined the year"],
  };
  const [kicker, title] = titles[page];
  let content = "";
  if (page === "progress") {
    content = buildProgressPage(input, 64, contentTop, width - 128, footerY - contentTop - 48, palette);
  } else if (page === "consistency") {
    content = buildConsistencyPage(input, 64, contentTop, width - 128, footerY - contentTop - 48, palette);
  } else if (page === "exploration") {
    content = buildExplorationPage(input, 64, contentTop, width - 128, footerY - contentTop - 48, palette);
  } else {
    content = buildHighlightsPage(input, 64, contentTop, width - 128, footerY - contentTop - 48, palette);
  }
  return `<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}" role="img" aria-labelledby="recap-title recap-description">
    <title id="recap-title">${escapeXml(input.year)} ${escapeXml(kicker.toLowerCase())}</title>
    <desc id="recap-description">${escapeXml(title)} for ${escapeXml(athlete)}</desc>
    <defs><linearGradient id="recap-background" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="${palette.background}"/><stop offset="1" stop-color="${dark ? "#0b1220" : "#eee7dd"}"/></linearGradient></defs>
    <rect width="${width}" height="${height}" fill="url(#recap-background)"/><circle cx="1000" cy="70" r="250" fill="${palette.accent}" opacity=".08"/>
    <g font-family="Inter, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif">
      <text x="64" y="88" fill="${palette.accent}" font-size="21" font-weight="850" letter-spacing="3">${escapeXml(kicker.toUpperCase())} · ${escapeXml(input.year)}</text>
      <text x="64" y="175" fill="${palette.text}" font-size="56" font-weight="900" letter-spacing="-2">${escapeXml(title)}</text>
      <text x="64" y="220" fill="${palette.muted}" font-size="22">${escapeXml(athlete)} · ${escapeXml(input.activityLabel)}</text>
      ${content}
      <text x="64" y="${footerY}" fill="${palette.muted}" font-size="17">Generated locally · card ${ANNUAL_RECAP_PAGES.findIndex((candidate) => candidate.id === page) + 1} of ${ANNUAL_RECAP_PAGES.length}</text>
      <text x="${width - 64}" y="${footerY}" text-anchor="end" fill="${palette.accent}" font-size="19" font-weight="850">#mystravastats</text>
    </g>
  </svg>`;
}

function buildProgressPage(
  input: AnnualRecapInput,
  x: number,
  y: number,
  width: number,
  height: number,
  palette: { panel: string; text: string; muted: string; line: string; positive: string; negative: string },
): string {
  const previousYear = String(Number(input.year) - 1);
  const comparisonNote = input.yearToDate
    ? `${input.year} to date compared with the full ${previousYear} total`
    : `Full-year totals compared with ${previousYear}`;
  const rows: Array<[string, number, number, string]> = [
    ["Distance", input.metrics.distanceKm, input.previousMetrics?.distanceKm ?? 0, "km"],
    ["Elevation", input.metrics.elevationM, input.previousMetrics?.elevationM ?? 0, "m"],
    ["Activities", input.metrics.activities, input.previousMetrics?.activities ?? 0, ""],
    ["Active days", input.metrics.activeDays, input.previousMetrics?.activeDays ?? 0, ""],
  ];
  const gap = 16;
  const rowHeight = (height - gap * 3) / 4;
  const cards = rows.map(([label, current, previous, unit], index) => {
    const delta = percentageDelta(current, previous);
    const deltaLabel = delta === null ? "No previous data" : `${delta >= 0 ? "+" : ""}${delta.toFixed(0)}%`;
    const deltaColor = delta === null ? palette.muted : delta >= 0 ? palette.positive : palette.negative;
    const rowY = index * (rowHeight + gap);
    return `<g transform="translate(0 ${rowY})"><rect width="${width}" height="${rowHeight}" rx="25" fill="${palette.panel}" stroke="${palette.line}"/><text x="28" y="40" fill="${palette.muted}" font-size="18" font-weight="800" letter-spacing="2">${label.toUpperCase()}</text><text x="28" y="${rowHeight - 28}" fill="${palette.text}" font-size="40" font-weight="850">${formatNumber(current, label === "Distance" ? 0 : 0)}${unit ? ` ${unit}` : ""}</text><text x="${width - 150}" y="${rowHeight - 28}" text-anchor="end" fill="${palette.muted}" font-size="18">${previousYear}: ${formatNumber(previous, 0)}</text><text x="${width - 28}" y="${rowHeight - 28}" text-anchor="end" fill="${deltaColor}" font-size="27" font-weight="850">${deltaLabel}</text></g>`;
  }).join("");
  return `<g transform="translate(${x} ${y})"><text x="0" y="-16" fill="${palette.muted}" font-size="17">${escapeXml(comparisonNote)}</text>${cards}</g>`;
}

function buildConsistencyPage(
  input: AnnualRecapInput,
  x: number,
  y: number,
  width: number,
  height: number,
  palette: { panel: string; text: string; muted: string; line: string; accent?: string; positive: string },
): string {
  const consistency = Math.max(0, Math.min(100, input.consistencyPercent ?? 0));
  const daysInYear = isLeapYear(Number(input.year)) ? 366 : 365;
  const daysInScope = Math.max(1, input.daysInScope ?? daysInYear);
  const activeRate = Math.min(100, (input.metrics.activeDays / daysInScope) * 100);
  const previousDelta = percentageDelta(consistency, input.previousConsistencyPercent);
  const circleRadius = Math.min(170, height * .22);
  const circumference = 2 * Math.PI * circleRadius;
  const dash = circumference * (consistency / 100);
  const centerX = width / 2;
  const centerY = circleRadius + 30;
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="30" fill="${palette.panel}" stroke="${palette.line}"/><circle cx="${centerX}" cy="${centerY}" r="${circleRadius}" fill="none" stroke="${palette.line}" stroke-width="28"/><circle cx="${centerX}" cy="${centerY}" r="${circleRadius}" fill="none" stroke="${palette.positive}" stroke-width="28" stroke-linecap="round" stroke-dasharray="${dash} ${circumference - dash}" transform="rotate(-90 ${centerX} ${centerY})"/><text x="${centerX}" y="${centerY + 12}" text-anchor="middle" fill="${palette.text}" font-size="70" font-weight="900">${formatNumber(consistency, 1)}%</text><text x="${centerX}" y="${centerY + 50}" text-anchor="middle" fill="${palette.muted}" font-size="18">CONSISTENCY SCORE</text><text x="42" y="${height - 112}" fill="${palette.text}" font-size="44" font-weight="850">${formatNumber(input.metrics.activeDays, 0)} active days</text><text x="42" y="${height - 70}" fill="${palette.muted}" font-size="21">Active on ${formatNumber(activeRate, 1)}% of days ${input.yearToDate ? "so far" : "in the year"}</text><text x="${width - 42}" y="${height - 70}" text-anchor="end" fill="${previousDelta !== null && previousDelta >= 0 ? palette.positive : palette.muted}" font-size="21" font-weight="750">${previousDelta === null ? "No previous score" : `${previousDelta >= 0 ? "+" : ""}${previousDelta.toFixed(0)}% vs previous year`}</text></g>`;
}

function buildExplorationPage(
  input: AnnualRecapInput,
  x: number,
  y: number,
  width: number,
  height: number,
  palette: { panel: string; text: string; muted: string; line: string; accent?: string },
): string {
  const statsHeight = 118;
  const map = input.includeMap
    ? buildMapSvg(input.tracks ?? [], 0, statsHeight + 18, width, height - statsHeight - 18, { ...palette, accent: palette.accent ?? "#fc4c02" })
    : buildPrivacyPanel(0, statsHeight + 18, width, height - statsHeight - 18, { ...palette, accent: palette.accent ?? "#fc4c02" });
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${statsHeight}" rx="25" fill="${palette.panel}" stroke="${palette.line}"/><text x="30" y="42" fill="${palette.muted}" font-size="17" font-weight="800" letter-spacing="2">THE YEAR IN MOTION</text><text x="30" y="90" fill="${palette.text}" font-size="34" font-weight="850">${formatNumber(input.metrics.distanceKm, 0)} km · ${formatNumber(input.metrics.elevationM, 0)} m climbed · ${formatNumber(input.metrics.activities, 0)} activities</text>${map}</g>`;
}

function buildHighlightsPage(
  input: AnnualRecapInput,
  x: number,
  y: number,
  width: number,
  height: number,
  palette: { panel: string; text: string; muted: string; line: string; accent?: string },
): string {
  const highlights = (input.highlights?.length ? input.highlights : buildAnnualRecapHighlights(input.metrics, input.previousMetrics, input.consistencyPercent, input.yearToDate)).slice(0, 5);
  const gap = 16;
  const itemHeight = (height - gap * Math.max(0, highlights.length - 1)) / Math.max(1, highlights.length);
  const items = highlights.map((highlight, index) => `<g transform="translate(0 ${index * (itemHeight + gap)})"><rect width="${width}" height="${itemHeight}" rx="25" fill="${palette.panel}" stroke="${palette.line}"/><circle cx="45" cy="${itemHeight / 2}" r="21" fill="${palette.accent ?? "#fc4c02"}" opacity=".15"/><text x="45" y="${itemHeight / 2 + 8}" text-anchor="middle" fill="${palette.accent ?? "#fc4c02"}" font-size="22" font-weight="900">${index + 1}</text><text x="85" y="${itemHeight / 2 + 9}" fill="${palette.text}" font-size="28" font-weight="780">${escapeXml(highlight)}</text></g>`).join("");
  return `<g transform="translate(${x} ${y})">${items || `<text x="0" y="40" fill="${palette.muted}" font-size="24">No highlights available for this selection.</text>`}</g>`;
}

export async function annualRecapSvgToPng(svg: string, width: number, height: number): Promise<Blob> {
  const svgBlob = new Blob([svg], { type: "image/svg+xml;charset=utf-8" });
  const url = URL.createObjectURL(svgBlob);
  try {
    const image = await loadImage(url);
    const canvas = document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d");
    if (!context) throw new Error("PNG export is not supported by this browser.");
    context.drawImage(image, 0, 0, width, height);
    return await new Promise<Blob>((resolve, reject) => {
      canvas.toBlob((blob) => blob ? resolve(blob) : reject(new Error("Unable to create the PNG file.")), "image/png", 1);
    });
  } finally {
    URL.revokeObjectURL(url);
  }
}

function buildMapSvg(
  tracks: MapTrack[],
  x: number,
  y: number,
  width: number,
  height: number,
  palette: { panel: string; line: string; muted: string; accent: string },
): string {
  const validTracks = tracks
    .map((track) => track.coordinates.filter(isCoordinate).slice(0, 1200))
    .filter((coordinates) => coordinates.length >= 2)
    .slice(0, 80);
  if (validTracks.length === 0) {
    return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/><text x="32" y="52" fill="${palette.muted}" font-size="20" font-weight="700">No GPS traces available for this selection</text></g>`;
  }
  const paddingX = 38;
  const top = 62;
  const bottom = 28;
  const drawableWidth = width - paddingX * 2;
  const drawableHeight = Math.max(24, height - top - bottom);
  const paths = validTracks.map((coordinates, trackIndex) => {
    const step = Math.max(1, Math.ceil(coordinates.length / 240));
    const sampled = coordinates.filter((_, index) => index % step === 0 || index === coordinates.length - 1);
    const bounds = sampled.reduce((result, [latitude, longitude]) => ({
      minLat: Math.min(result.minLat, latitude),
      maxLat: Math.max(result.maxLat, latitude),
      minLng: Math.min(result.minLng, longitude),
      maxLng: Math.max(result.maxLng, longitude),
    }), { minLat: Infinity, maxLat: -Infinity, minLng: Infinity, maxLng: -Infinity });
    const latSpan = Math.max(bounds.maxLat - bounds.minLat, 0.000001);
    const lngSpan = Math.max(bounds.maxLng - bounds.minLng, 0.000001);
    const scale = Math.min(drawableWidth / lngSpan, drawableHeight / latSpan);
    const usedWidth = lngSpan * scale;
    const usedHeight = latSpan * scale;
    const offsetX = paddingX + (drawableWidth - usedWidth) / 2;
    const offsetY = top + (drawableHeight - usedHeight) / 2;
    const points = sampled.map(([latitude, longitude], index) => {
      const px = offsetX + (longitude - bounds.minLng) * scale;
      const py = offsetY + (bounds.maxLat - latitude) * scale;
      return `${index === 0 ? "M" : "L"}${px.toFixed(1)} ${py.toFixed(1)}`;
    }).join(" ");
    const opacity = trackIndex % 9 === 0 ? ".46" : ".16";
    return `<path d="${points}" fill="none" stroke="${palette.accent}" stroke-width="${trackIndex % 9 === 0 ? 3 : 2}" stroke-linecap="round" stroke-linejoin="round" opacity="${opacity}"/>`;
  }).join("");
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/><text x="30" y="40" fill="${palette.muted}" font-size="17" font-weight="750" letter-spacing="2">ACTIVITY FINGERPRINT · ${validTracks.length} TRACES</text>${paths}</g>`;
}

function buildPrivacyPanel(
  x: number,
  y: number,
  width: number,
  height: number,
  palette: { panel: string; line: string; muted: string; accent: string },
): string {
  const centerY = Math.max(64, height / 2);
  return `<g transform="translate(${x} ${y})"><rect width="${width}" height="${height}" rx="28" fill="${palette.panel}" stroke="${palette.line}"/><circle cx="50" cy="${centerY}" r="18" fill="${palette.accent}" opacity=".16"/><path d="M43 ${centerY}h14M50 ${centerY - 7}v14" stroke="${palette.accent}" stroke-width="3" stroke-linecap="round"/><text x="82" y="${centerY - 4}" fill="${palette.muted}" font-size="20" font-weight="750">Map hidden for privacy</text><text x="82" y="${centerY + 25}" fill="${palette.muted}" font-size="16">Enable it before export to add an abstract trace fingerprint.</text></g>`;
}

function loadImage(url: string): Promise<HTMLImageElement> {
  return new Promise((resolve, reject) => {
    const image = new Image();
    image.onload = () => resolve(image);
    image.onerror = () => reject(new Error("Unable to render the annual recap."));
    image.src = url;
  });
}

function isCoordinate(value: number[]): value is [number, number] {
  return value.length >= 2
    && Number.isFinite(value[0])
    && Number.isFinite(value[1])
    && Math.abs(value[0]) <= 90
    && Math.abs(value[1]) <= 180;
}

function formatNumber(value: number, maximumFractionDigits: number): string {
  return Math.max(0, Number.isFinite(value) ? value : 0).toLocaleString("en-US", { maximumFractionDigits });
}

function formatDuration(totalSeconds: number): string {
  const hours = Math.max(0, Math.round((Number.isFinite(totalSeconds) ? totalSeconds : 0) / 3600));
  return `${hours.toLocaleString("en-US")} h`;
}

function formatDate(value: string | undefined): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value ?? "");
  return match ? `${match[3]}/${match[2]}/${match[1]}` : "";
}

function percentageDelta(current: number, previous: number | undefined): number | null {
  if (!Number.isFinite(current) || !Number.isFinite(previous) || !previous || previous <= 0) return null;
  return ((current - previous) / previous) * 100;
}

function isLeapYear(year: number): boolean {
  return Number.isInteger(year) && year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
}

function escapeXml(value: string): string {
  return value.replace(/[&<>"']/g, (character) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    "\"": "&quot;",
    "'": "&apos;",
  })[character] ?? character);
}
