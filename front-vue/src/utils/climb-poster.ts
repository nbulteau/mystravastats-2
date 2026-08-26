import type { ClimbDetails } from "@/models/badge-check-result.model";

export type ClimbPosterDesign = "alpine-index" | "massif-atlas" | "profile-wall";

export interface ClimbPosterEntry {
  label: string;
  category?: string | null;
  details: ClimbDetails;
}

export interface ClimbPosterOptions {
  design: ClimbPosterDesign;
  climbs: ClimbPosterEntry[];
  yearLabel: string;
  athleteName?: string;
}

export interface ClimbGradientSegment {
  startKm: number;
  endKm: number;
  startElevation: number;
  endElevation: number;
  averageGradient: number;
}

export const CLIMB_POSTER_DESIGNS: Array<{
  id: ClimbPosterDesign;
  name: string;
  description: string;
  maxClimbs: number;
}> = [
  {
    id: "alpine-index",
    name: "Alpine Index",
    description: "Minimal atlas with fine profile lines",
    maxClimbs: 50,
  },
  {
    id: "massif-atlas",
    name: "Massif Atlas",
    description: "Grouped by massif with restrained colour",
    maxClimbs: 50,
  },
  {
    id: "profile-wall",
    name: "Profile Wall",
    description: "Visual gallery of climb silhouettes",
    maxClimbs: 50,
  },
];

const STANDARD_POSTER_WIDTH = 1200;
const STANDARD_POSTER_HEIGHT = 1680;
const LARGE_POSTER_WIDTH = 2000;
const LARGE_POSTER_HEIGHT = 3000;
const DENSE_COLUMN_COUNT = 5;

export function posterDesignMaxClimbs(design: ClimbPosterDesign): number {
  return CLIMB_POSTER_DESIGNS.find((candidate) => candidate.id === design)?.maxClimbs ?? 50;
}

export function buildClimbPosterSvg(options: ClimbPosterOptions): string {
  const maxClimbs = posterDesignMaxClimbs(options.design);
  const climbs = options.climbs.slice(0, maxClimbs);
  if (climbs.length === 0) {
    throw new Error("Select at least one completed climb before generating a poster.");
  }
  const largePoster = climbs.length > 25;
  const posterWidth = largePoster ? LARGE_POSTER_WIDTH : STANDARD_POSTER_WIDTH;
  const posterHeight = largePoster ? LARGE_POSTER_HEIGHT : STANDARD_POSTER_HEIGHT;

  const content = options.design === "alpine-index"
    ? buildAlpineIndexDesign(climbs, options)
    : options.design === "massif-atlas"
      ? buildMassifAtlasDesign(climbs, options)
      : buildProfileWallDesign(climbs, options);

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${posterWidth}" height="${posterHeight}" viewBox="0 0 ${posterWidth} ${posterHeight}" role="img" aria-labelledby="poster-title poster-description">
  <title id="poster-title">${escapeXml(posterTitle(options.design))}</title>
  <desc id="poster-description">Cycling climb poster with altitude profiles, gradients, best ascent dates and times, and ascent counts.</desc>
  ${content}
</svg>`;
}

function posterTitle(design: ClimbPosterDesign): string {
  return CLIMB_POSTER_DESIGNS.find((candidate) => candidate.id === design)?.name ?? "Climb poster";
}

function posterClimbName(climb: ClimbPosterEntry): string {
  const label = climb.label.trim();
  if (label.length === 0) {
    return climb.details.name;
  }
  return label;
}

function buildAlpineIndexDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  const geometry = densePosterGeometry(climbs, 260, 1260);
  const { largePoster, width, height, margin, right, top, contentHeight } = geometry;
  const columnCount = posterColumnCount(climbs.length);
  const columnGap = largePoster ? 30 : 16;
  const columnWidth = (width - margin * 2 - columnGap * (columnCount - 1)) / columnCount;
  const rowCount = Math.ceil(climbs.length / columnCount);
  const rowHeight = contentHeight / rowCount;
  const rows = climbs.map((climb, index) => {
    const column = index % columnCount;
    const row = Math.floor(index / columnCount);
    const x = margin + column * (columnWidth + columnGap);
    const y = top + row * rowHeight;
    const profileTop = y + (largePoster ? 64 : 52);
    const tileBottom = y + rowHeight - (largePoster ? 28 : 20);
    const ascentY = tileBottom - 8;
    const statsY = ascentY - (largePoster ? 27 : 23);
    const profileBottom = statsY - 17;
    const profileHeight = Math.max(74, profileBottom - profileTop);
    const name = posterClimbName(climb).toUpperCase();
    const summitLine = `MAX ALT ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTY ${formatInteger(climb.details.difficulty)} PTS`;
    const statsLine = `${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M · ${formatDecimal(climb.details.averageGradient)} % AVG · ${formatMaximumGradient(climb.details.maximumGradient)} % MAX`;
    return `<g data-grid-column="${column}" data-grid-row="${row}" data-profile-height="${round(profileHeight)}">
      ${column > 0 ? `<line x1="${round(x - columnGap / 2)}" y1="${round(y)}" x2="${round(x - columnGap / 2)}" y2="${round(tileBottom)}" class="alpine-grid-rule"/>` : ""}
      ${row > 0 ? `<line x1="${round(x)}" y1="${round(y - 16)}" x2="${round(x + columnWidth)}" y2="${round(y - 16)}" class="alpine-grid-rule"/>` : ""}
      <text x="${round(x)}" y="${round(y + 20)}" class="alpine-index">${String(index + 1).padStart(2, "0")}</text>
      <text x="${round(x + 34)}" y="${round(y + 20)}" class="alpine-name"${fitTextAttributes(name, largePoster ? 22 : 16, columnWidth - 40, largePoster ? 18 : 15, 8.5, 0.58)}>${escapeXml(name)}</text>
      <text x="${round(x)}" y="${round(y + 46)}" class="alpine-summit"${fitTextAttributes(summitLine, 39, columnWidth, largePoster ? 12 : 10, 7, 0.53)}>${escapeXml(summitLine)}</text>
      ${profileMarkup(climb.details, x, profileTop, columnWidth, profileHeight, "alpine-profile", { maxSegments: largePoster ? 14 : 10, labelScale: largePoster ? 0.9 : 0.72 })}
      <text x="${round(x)}" y="${round(statsY)}" class="alpine-stat"${fitTextAttributes(statsLine, 50, columnWidth, largePoster ? 12 : 10, 7, 0.52)}>${escapeXml(statsLine)}</text>
      ${ascentSummaryMarkup(climb.details, x, ascentY, columnWidth, "alpine-ascent")}
    </g>`;
  }).join("");

  return `<style>
    .alpine-paper{fill:#f7f8f6}.alpine-ink{fill:#15191c}.alpine-muted{fill:#5b6569}.alpine-hairline{stroke:#15191c;stroke-width:2}.alpine-grid-rule{stroke:#c9d0cd;stroke-width:1}.alpine-index{font:600 13px ui-monospace,monospace;fill:#a44531;letter-spacing:0}.alpine-name{font:600 16px ui-sans-serif,system-ui;fill:#15191c;letter-spacing:0}.alpine-summit{font:600 10px ui-sans-serif,system-ui;fill:#5b6569;letter-spacing:0}.alpine-stat{font:500 10px ui-sans-serif,system-ui;fill:#293136;letter-spacing:0}.alpine-ascent{font:400 10px ui-monospace,monospace;fill:#5b6569;letter-spacing:0}.alpine-profile{fill:none;stroke:#15191c;stroke-width:2.4;stroke-linecap:round;stroke-linejoin:round}.profile-base{stroke:#9aa5a1;stroke-width:1}.profile-missing{font:500 11px ui-sans-serif,system-ui;fill:#7a8380;letter-spacing:0}
  </style>
  <rect width="${width}" height="${height}" class="alpine-paper"/>
  <text x="${margin}" y="${largePoster ? 108 : 78}" class="alpine-muted" style="font:600 ${largePoster ? 18 : 14}px ui-sans-serif,system-ui;letter-spacing:0">ALPINE INDEX · ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="${margin}" y="${largePoster ? 224 : 164}" class="alpine-ink" style="font:600 ${largePoster ? 92 : 64}px ui-sans-serif,system-ui;letter-spacing:0">CLIMBS ${String(climbs.length).padStart(2, "0")}</text>
  <text x="${right}" y="${largePoster ? 210 : 154}" text-anchor="end" class="alpine-ink" style="font:500 ${largePoster ? 34 : 24}px ui-sans-serif,system-ui;letter-spacing:0">${escapeXml(options.yearLabel)}</text>
  <line x1="${margin}" y1="${top - 34}" x2="${right}" y2="${top - 34}" class="alpine-hairline"/>
  ${rows}
  ${footerMarkup(options, climbs, "#5b6569")}`;
}

function buildMassifAtlasDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  const arrangedClimbs = climbsByMassif(climbs);
  const geometry = densePosterGeometry(arrangedClimbs, 270, 1245);
  const { largePoster, width, height, margin, right, top, contentHeight } = geometry;
  const columnCount = posterColumnCount(arrangedClimbs.length);
  const columnGap = largePoster ? 28 : 16;
  const columnWidth = (width - margin * 2 - columnGap * (columnCount - 1)) / columnCount;
  const rowCount = Math.ceil(arrangedClimbs.length / columnCount);
  const rowHeight = contentHeight / rowCount;
  const entries = arrangedClimbs.map((climb, index) => {
    const column = index % columnCount;
    const row = Math.floor(index / columnCount);
    const x = margin + column * (columnWidth + columnGap);
    const y = top + row * rowHeight;
    const massif = climb.details.massif || "Unknown";
    const color = massifColor(massif);
    const titleLines = splitPosterTitle(posterClimbName(climb).toUpperCase(), largePoster ? 31 : 22);
    const titleFontSize = fittedTextFontSize(
      Math.max(...titleLines.map((line) => line.length)),
      columnWidth - 18,
      largePoster ? 16 : 13,
      7.5,
      0.58,
    );
    const profileTop = y + (titleLines.length > 1 ? 74 : 60);
    const tileBottom = y + rowHeight - (largePoster ? 26 : 20);
    const ascentY = tileBottom - 8;
    const summitY = ascentY - (largePoster ? 24 : 20);
    const metricsY = summitY - (largePoster ? 20 : 17);
    const profileBottom = metricsY - 16;
    const profileHeight = Math.max(70, profileBottom - profileTop);
    const metrics = `${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M · ${formatDecimal(climb.details.averageGradient)} % AVG · ${formatMaximumGradient(climb.details.maximumGradient)} % MAX`;
    const summit = `MAX ALT ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTY ${formatInteger(climb.details.difficulty)} PTS`;
    return `<g data-grid-column="${column}" data-grid-row="${row}" data-massif="${escapeXml(massif)}" data-profile-height="${round(profileHeight)}">
      <rect x="${round(x)}" y="${round(y - 5)}" width="${round(columnWidth)}" height="${round(rowHeight - 13)}" rx="6" class="atlas-entry"/>
      <rect x="${round(x)}" y="${round(y - 5)}" width="${round(columnWidth)}" height="5" rx="2" fill="${color}"/>
      <text x="${round(x)}" y="${round(y + 22)}" class="atlas-index">${String(index + 1).padStart(2, "0")}</text>
      <text x="${round(x + 34)}" y="${round(y + 22)}" class="atlas-name" style="font-size:${round(titleFontSize)}px">${titleLines.map((line, lineIndex) => `<tspan x="${round(x + 34)}" y="${round(y + 22 + lineIndex * (largePoster ? 16 : 13))}">${escapeXml(line)}</tspan>`).join("")}</text>
      <text x="${round(x + 34)}" y="${round(y + (titleLines.length > 1 ? 54 : 42))}" class="atlas-location">${escapeXml(locationLabel(climb))}</text>
      ${profileMarkup(climb.details, x + 8, profileTop, columnWidth - 16, profileHeight, "atlas-profile", { maxSegments: largePoster ? 12 : 9, showLabels: false })}
      <text x="${round(x + 8)}" y="${round(metricsY)}" class="atlas-metrics"${fitTextAttributes(metrics, 42, columnWidth - 16, largePoster ? 11 : 9, 6.5, 0.5)}>${escapeXml(metrics)}</text>
      <text x="${round(x + 8)}" y="${round(summitY)}" class="atlas-summit"${fitTextAttributes(summit, 39, columnWidth - 16, largePoster ? 11 : 9, 6.5, 0.5)}>${escapeXml(summit)}</text>
      ${ascentSummaryMarkup(climb.details, x + 8, ascentY, columnWidth - 16, "atlas-ascent")}
    </g>`;
  }).join("");

  return `<style>
    .atlas-paper{fill:#edf3ef}.atlas-ink{fill:#172321}.atlas-muted{fill:#5a6d67}.atlas-rule{stroke:#31423d;stroke-width:2}.atlas-entry{fill:#f9fbf8;stroke:#b8c8c0;stroke-width:1}.atlas-index{font:700 13px ui-monospace,monospace;fill:#172321;letter-spacing:0}.atlas-name{font:650 14px ui-sans-serif,system-ui;fill:#172321;letter-spacing:0}.atlas-location{font:700 9px ui-sans-serif,system-ui;fill:#5a6d67;letter-spacing:0}.atlas-metrics{font:600 10px ui-sans-serif,system-ui;fill:#172321;letter-spacing:0}.atlas-summit{font:600 10px ui-sans-serif,system-ui;fill:#5a6d67;letter-spacing:0}.atlas-ascent{font:400 9.5px ui-monospace,monospace;fill:#5a6d67;letter-spacing:0}.atlas-profile{fill:none;stroke:#172321;stroke-width:2.1;stroke-linecap:round;stroke-linejoin:round}.profile-base{stroke:#8da099;stroke-width:1}.profile-missing{font:500 10px ui-sans-serif,system-ui;fill:#6f817b;letter-spacing:0}
  </style>
  <rect width="${width}" height="${height}" class="atlas-paper"/>
  <text x="${margin}" y="${largePoster ? 108 : 80}" class="atlas-muted" style="font:600 ${largePoster ? 18 : 14}px ui-sans-serif,system-ui;letter-spacing:0">MASSIF ATLAS · ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="${margin}" y="${largePoster ? 224 : 164}" class="atlas-ink" style="font:650 ${largePoster ? 90 : 62}px ui-sans-serif,system-ui;letter-spacing:0">${escapeXml(massifSummary(climbs))}</text>
  <text x="${right}" y="${largePoster ? 210 : 154}" text-anchor="end" class="atlas-muted" style="font:600 ${largePoster ? 28 : 20}px ui-sans-serif,system-ui;letter-spacing:0">${posterCountLabel(climbs.length)}</text>
  <line x1="${margin}" y1="${top - 34}" x2="${right}" y2="${top - 34}" class="atlas-rule"/>
  ${entries}
  ${footerMarkup(options, climbs, "#5a6d67")}`;
}

function buildProfileWallDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  const geometry = densePosterGeometry(climbs, 270, 1240);
  const { largePoster, width, height, margin, right, top, contentHeight } = geometry;
  const columnCount = posterColumnCount(climbs.length);
  const columnGap = largePoster ? 24 : 16;
  const columnWidth = (width - margin * 2 - columnGap * (columnCount - 1)) / columnCount;
  const rowCount = Math.ceil(climbs.length / columnCount);
  const rowHeight = contentHeight / rowCount;
  const tiles = climbs.map((climb, index) => {
    const column = index % columnCount;
    const row = Math.floor(index / columnCount);
    const x = margin + column * (columnWidth + columnGap);
    const y = top + row * rowHeight;
    const profileTop = y + (largePoster ? 52 : 42);
    const tileBottom = y + rowHeight - (largePoster ? 26 : 20);
    const ascentY = tileBottom - 7;
    const summitY = ascentY - (largePoster ? 22 : 19);
    const metricsY = summitY - (largePoster ? 20 : 17);
    const profileBottom = metricsY - 18;
    const profileHeight = Math.max(78, profileBottom - profileTop);
    const title = posterClimbName(climb).toUpperCase();
    const metrics = `${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M · ${formatDecimal(climb.details.averageGradient)} % AVG · ${formatMaximumGradient(climb.details.maximumGradient)} % MAX`;
    const summit = `MAX ALT ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTY ${formatInteger(climb.details.difficulty)} PTS`;
    return `<g data-grid-column="${column}" data-grid-row="${row}" data-profile-height="${round(profileHeight)}">
      <rect x="${round(x - 4)}" y="${round(y - 6)}" width="${round(columnWidth + 8)}" height="${round(rowHeight - 10)}" rx="4" class="wall-tile"/>
      <text x="${round(x)}" y="${round(y + 19)}" class="wall-index">${String(index + 1).padStart(2, "0")}</text>
      <text x="${round(x + 35)}" y="${round(y + 19)}" class="wall-name"${fitTextAttributes(title, largePoster ? 22 : 16, columnWidth - 42, largePoster ? 16 : 13, 7.5, 0.58)}>${escapeXml(title)}</text>
      ${profileMarkup(climb.details, x, profileTop, columnWidth, profileHeight, "wall-profile", { maxSegments: largePoster ? 18 : 13, showLabels: false })}
      <text x="${round(x)}" y="${round(metricsY)}" class="wall-metrics"${fitTextAttributes(metrics, 42, columnWidth, largePoster ? 12 : 10, 7, 0.5)}>${escapeXml(metrics)}</text>
      <text x="${round(x)}" y="${round(summitY)}" class="wall-summit"${fitTextAttributes(summit, 39, columnWidth, largePoster ? 11 : 9.5, 6.5, 0.5)}>${escapeXml(summit)}</text>
      ${ascentSummaryMarkup(climb.details, x, ascentY, columnWidth, "wall-ascent")}
    </g>`;
  }).join("");

  return `<style>
    .wall-paper{fill:#121416}.wall-ink{fill:#f4f2e9}.wall-muted{fill:#aeb7b2}.wall-rule{stroke:#d7ddd8;stroke-opacity:.34;stroke-width:2}.wall-tile{fill:#1a1e21;stroke:#343c3f;stroke-width:1}.wall-index{font:700 13px ui-monospace,monospace;fill:#ef7d4c;letter-spacing:0}.wall-name{font:650 15px ui-sans-serif,system-ui;fill:#f4f2e9;letter-spacing:0}.wall-metrics{font:600 10px ui-sans-serif,system-ui;fill:#f4f2e9;letter-spacing:0}.wall-summit{font:600 10px ui-sans-serif,system-ui;fill:#aeb7b2;letter-spacing:0}.wall-ascent{font:400 9.5px ui-monospace,monospace;fill:#aeb7b2;letter-spacing:0}.wall-profile{fill:none;stroke:#f4f2e9;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round}.profile-base{stroke:#6d7775;stroke-width:1}.profile-missing{font:500 10px ui-sans-serif,system-ui;fill:#aeb7b2;letter-spacing:0}
  </style>
  <rect width="${width}" height="${height}" class="wall-paper"/>
  <text x="${margin}" y="${largePoster ? 108 : 80}" class="wall-muted" style="font:600 ${largePoster ? 18 : 14}px ui-sans-serif,system-ui;letter-spacing:0">PROFILE WALL · ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="${margin}" y="${largePoster ? 224 : 164}" class="wall-ink" style="font:650 ${largePoster ? 90 : 62}px ui-sans-serif,system-ui;letter-spacing:0">${posterCountLabel(climbs.length)}</text>
  <text x="${right}" y="${largePoster ? 210 : 154}" text-anchor="end" class="wall-muted" style="font:600 ${largePoster ? 28 : 20}px ui-sans-serif,system-ui;letter-spacing:0">${escapeXml(options.yearLabel)}</text>
  <line x1="${margin}" y1="${top - 34}" x2="${right}" y2="${top - 34}" class="wall-rule"/>
  ${tiles}
  ${footerMarkup(options, climbs, "#aeb7b2")}`;
}

function climbsByMassif(climbs: ClimbPosterEntry[]): ClimbPosterEntry[] {
  return climbs
    .map((climb, index) => ({ climb, index }))
    .sort((left, right) => (
      (left.climb.details.massif || "").localeCompare(right.climb.details.massif || "") ||
      right.climb.details.difficulty - left.climb.details.difficulty ||
      left.index - right.index
    ))
    .map(({ climb }) => climb);
}

function locationLabel(climb: ClimbPosterEntry): string {
  return `${climb.details.country} · ${climb.details.massif}`.toUpperCase();
}

function massifSummary(climbs: ClimbPosterEntry[]): string {
  const count = new Set(climbs.map((climb) => climb.details.massif).filter(Boolean)).size;
  return `${count} MASSIF${count === 1 ? "" : "S"}`;
}

function posterCountLabel(count: number): string {
  return `${String(count).padStart(2, "0")} CLIMB${count === 1 ? "" : "S"}`;
}

function massifColor(massif: string): string {
  const palette = ["#2c6e60", "#b64b37", "#416c9c", "#8a6f2a", "#7b4d8b", "#4e7d3a", "#9c5b31"];
  const hash = [...massif].reduce((sum, character) => sum + character.charCodeAt(0), 0);
  return palette[hash % palette.length] ?? palette[0];
}

function posterColumnCount(climbCount: number): number {
  return climbCount <= 3 ? Math.max(1, climbCount) : DENSE_COLUMN_COUNT;
}

function densePosterGeometry(climbs: ClimbPosterEntry[], standardTop: number, standardContentHeight: number) {
  const largePoster = climbs.length > 25;
  const width = largePoster ? LARGE_POSTER_WIDTH : STANDARD_POSTER_WIDTH;
  const height = largePoster ? LARGE_POSTER_HEIGHT : STANDARD_POSTER_HEIGHT;
  const margin = largePoster ? 84 : 52;
  const top = largePoster ? 310 : standardTop;
  const contentHeight = largePoster ? 2390 : standardContentHeight;
  return {
    largePoster,
    width,
    height,
    margin,
    right: width - margin,
    top,
    contentHeight,
  };
}

function profileMarkup(
  details: ClimbDetails,
  x: number,
  y: number,
  width: number,
  height: number,
  lineClass: string,
  options: { maxSegments?: number; labelScale?: number; showLabels?: boolean } = {},
): string {
  const points = normalizedProfilePoints(details.profile);
  const baseY = y + height;
  if (points.length < 2) {
    return `<line x1="${round(x)}" y1="${round(baseY)}" x2="${round(x + width)}" y2="${round(baseY)}" class="profile-base"/><text x="${round(x)}" y="${round(y + height / 2)}" class="profile-missing">PROFILE UNAVAILABLE</text>`;
  }
  const minDistance = points[0]?.distanceKm ?? 0;
  const maxDistance = points.at(-1)?.distanceKm ?? details.lengthKm;
  const distanceRange = Math.max(0.001, maxDistance - minDistance);
  const profileElevations = points.map((point) => point.elevation);
  const referenceMinimum = Number.isFinite(details.minimumAltitude) && details.minimumAltitude < details.summitAltitude
    ? details.minimumAltitude
    : Math.min(...profileElevations);
  const referenceMaximum = Number.isFinite(details.summitAltitude) && details.summitAltitude > referenceMinimum
    ? details.summitAltitude
    : Math.max(...profileElevations);
  const minElevation = referenceMinimum;
  const maxElevation = referenceMaximum;
  const elevationRange = Math.max(1, maxElevation - minElevation);
  const axisHeight = height >= 80 ? 18 : 13;
  const plotHeight = Math.max(20, height - axisHeight);
  const plotBaseY = y + plotHeight;
  const profileX = (distanceKm: number): number => x + ((distanceKm - minDistance) / distanceRange) * width;
  const profileY = (elevation: number): number => {
    const clampedElevation = Math.min(maxElevation, Math.max(minElevation, elevation));
    return y + plotHeight - ((clampedElevation - minElevation) / elevationRange) * (plotHeight - 8);
  };
  const maxSegments = options.maxSegments ?? Math.min(20, Math.max(4, Math.floor(width / 28)));
  const labelScale = options.labelScale ?? 1;
  const showLabels = options.showLabels ?? true;
  const segments = buildClimbGradientSegments(points, maxSegments);
  const steepestGradient = Math.max(...segments.map((segment) => segment.averageGradient));
  const segmentMarkup = segments.map((segment, index) => {
    const startX = profileX(segment.startKm);
    const endX = profileX(segment.endKm);
    const startY = profileY(segment.startElevation);
    const endY = profileY(segment.endElevation);
    const segmentWidth = Math.max(1, endX - startX);
    const gradient = round(segment.averageGradient);
    const color = gradientColor(gradient);
    const textColor = gradientTextColor(gradient);
    const isSteepest = Math.abs(segment.averageGradient - steepestGradient) < 0.001;
    const gradeLabel = `${formatCompactDecimal(gradient)}%`;
    const distanceLabel = `${formatCompactDecimal(segment.endKm - minDistance)}${segment === segments.at(-1) ? " KM" : ""}`;
    const labelX = startX + segmentWidth / 2;
    const labelFontSize = (height >= 100 ? 9 : 7.5) * labelScale;
    const gradeText = showLabels && segmentWidth >= 27
      ? `<text x="${round(labelX)}" y="${round(plotBaseY - 4)}" text-anchor="middle" style="font:700 ${labelFontSize}px ui-monospace,monospace;fill:${textColor}">${gradeLabel}</text>`
      : showLabels && segmentWidth >= 13 && height >= 70
        ? `<text transform="translate(${round(labelX)} ${round(plotBaseY - 4)}) rotate(-90)" text-anchor="start" style="font:700 ${labelFontSize}px ui-monospace,monospace;fill:${textColor}">${gradeLabel}</text>`
        : "";
    const distanceText = showLabels && segmentWidth >= 24
      ? `<text x="${round(endX - 1)}" y="${round(baseY - 2)}" text-anchor="end" style="font:500 ${(height >= 80 ? 7.5 : 6.5) * labelScale}px ui-monospace,monospace;fill:#4d5961;fill-opacity:.9">${distanceLabel}</text>`
      : "";
    return `<g data-profile-segment="${round(segment.startKm)}-${round(segment.endKm)}" data-gradient="${gradient}">
      <path d="M ${round(startX)},${round(startY)} L ${round(endX)},${round(endY)} L ${round(endX)},${round(plotBaseY)} L ${round(startX)},${round(plotBaseY)} Z" fill="${color}"${isSteepest ? ' stroke="#2a2526" stroke-width="1.2"' : ' stroke="#ffffff" stroke-opacity=".55" stroke-width=".7"'}>
        <title>${formatCompactDecimal(segment.startKm - minDistance)}–${formatCompactDecimal(segment.endKm - minDistance)} km · ${gradeLabel}</title>
      </path>
      ${index < segments.length - 1 ? `<line x1="${round(endX)}" y1="${round(endY)}" x2="${round(endX)}" y2="${round(plotBaseY)}" style="stroke:#243038;stroke-opacity:.28;stroke-width:.8"/>` : ""}
      ${distanceText}${gradeText}
    </g>`;
  }).join("");
  const lineCoordinates = segments.length > 0
    ? [
      `${round(profileX(segments[0]?.startKm ?? minDistance))},${round(profileY(segments[0]?.startElevation ?? minElevation))}`,
      ...segments.map((segment) => `${round(profileX(segment.endKm))},${round(profileY(segment.endElevation))}`),
    ]
    : points.map((point) => `${round(profileX(point.distanceKm))},${round(profileY(point.elevation))}`);
  const linePath = `M ${lineCoordinates.join(" L ")}`;
  const altitudeBoundaries = segments.length > 0
    ? [
      { distanceKm: segments[0]?.startKm ?? minDistance, elevation: details.minimumAltitude || segments[0]?.startElevation || minElevation },
      ...segments.map((segment, index) => ({
        distanceKm: segment.endKm,
        elevation: index === segments.length - 1 ? details.summitAltitude : segment.endElevation,
      })),
    ]
    : [];
  const desiredAltitudeLabels = Math.max(2, Math.floor(width / 84));
  const altitudeStep = Math.max(1, Math.ceil((altitudeBoundaries.length - 1) / desiredAltitudeLabels));
  const minimumAltitudeLabelGap = height >= 100 ? 52 : 44;
  const altitudeLabelBoundaries = altitudeBoundaries.length > 0
    ? [
      altitudeBoundaries[0],
      altitudeBoundaries.at(-1),
      ...altitudeBoundaries.filter((_, index) => (
        index > 0 &&
        index < altitudeBoundaries.length - 1 &&
        index % altitudeStep === 0
      )),
    ].reduce<typeof altitudeBoundaries>((selected, boundary) => {
      if (!boundary || selected.includes(boundary)) {
        return selected;
      }
      const boundaryX = profileX(boundary.distanceKm);
      if (selected.every((candidate) => Math.abs(profileX(candidate.distanceKm) - boundaryX) >= minimumAltitudeLabelGap)) {
        selected.push(boundary);
      }
      return selected;
    }, []).sort((left, right) => left.distanceKm - right.distanceKm)
    : [];
  const altitudeMarkup = showLabels && height >= 70
    ? altitudeLabelBoundaries.map((boundary) => {
      const index = altitudeBoundaries.indexOf(boundary);
      const isLast = index === altitudeBoundaries.length - 1;
      const px = profileX(boundary.distanceKm);
      const py = profileY(boundary.elevation);
      const anchor = index === 0 ? "start" : isLast ? "end" : "middle";
      const offsetX = index === 0 ? 2 : isLast ? -2 : 0;
      const altitudeOffsetY = index === 0 ? (height >= 100 ? 18 : 15) : 4;
      return `<text data-profile-altitude-label="${formatInteger(boundary.elevation)}" x="${round(px + offsetX)}" y="${round(Math.max(y + 9, py - altitudeOffsetY))}" text-anchor="${anchor}" style="font:600 ${(height >= 100 ? 8 : 7) * labelScale}px ui-monospace,monospace;fill:#283238;paint-order:stroke;stroke:#ffffff;stroke-width:2px;stroke-opacity:.72">${formatInteger(boundary.elevation)} M</text>`;
    }).join("")
    : "";
  return `<line x1="${round(x)}" y1="${round(plotBaseY)}" x2="${round(x + width)}" y2="${round(plotBaseY)}" class="profile-base"/>${segmentMarkup}<path d="${linePath}" class="${lineClass}"/>${altitudeMarkup}`;
}

export function buildDetailedClimbProfileSvg(details: ClimbDetails): string {
  const width = 1200;
  const height = 360;
  const horizontalPadding = 28;
  const profileHeight = 318;
  const kilometerSegments = Math.max(1, Math.ceil(details.lengthKm));
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ${width} ${height}" role="img" aria-label="Kilometre profile of the climb" preserveAspectRatio="xMidYMid meet">
    <rect width="${width}" height="${height}" rx="18" fill="#fbf8f1"/>
    <style>.detail-profile{fill:none;stroke:#172129;stroke-width:4;stroke-linecap:round;stroke-linejoin:round}.profile-base{stroke:#9f9588;stroke-width:1.5}.profile-missing{font:500 18px ui-sans-serif,system-ui;fill:#756c62;letter-spacing:0}</style>
    ${profileMarkup(
      details,
      horizontalPadding,
      18,
      width - horizontalPadding * 2,
      profileHeight,
      "detail-profile",
      { maxSegments: kilometerSegments, labelScale: 1.45 },
    )}
  </svg>`;
}

export function buildClimbGradientSegments(
  profile: ClimbDetails["profile"],
  maxSegments = 16,
): ClimbGradientSegment[] {
  const points = normalizedProfilePoints(profile);
  if (points.length < 2) {
    return [];
  }
  const startDistance = points[0]?.distanceKm ?? 0;
  const endDistance = points.at(-1)?.distanceKm ?? startDistance;
  const distance = endDistance - startDistance;
  if (distance <= 0) {
    return [];
  }

  const segmentLimit = Math.max(1, Math.floor(maxSegments));
  const minimumStep = distance <= segmentLimit / 2 ? 0.5 : 1;
  const stepKm = Math.max(minimumStep, Math.ceil((distance / segmentLimit) * 2) / 2);
  const segments: ClimbGradientSegment[] = [];
  let segmentStart = startDistance;
  while (segmentStart < endDistance - 0.0001) {
    const segmentEnd = Math.min(endDistance, segmentStart + stepKm);
    const startElevation = interpolateElevation(points, segmentStart);
    const endElevation = interpolateElevation(points, segmentEnd);
    const segmentDistance = segmentEnd - segmentStart;
    segments.push({
      startKm: round(segmentStart),
      endKm: round(segmentEnd),
      startElevation: round(startElevation),
      endElevation: round(endElevation),
      averageGradient: round((endElevation - startElevation) / (segmentDistance * 10)),
    });
    segmentStart = segmentEnd;
  }
  return segments;
}

function normalizedProfilePoints(profile: ClimbDetails["profile"]): ClimbDetails["profile"] {
  return [...profile]
    .filter((point) => Number.isFinite(point.distanceKm) && Number.isFinite(point.elevation))
    .sort((left, right) => left.distanceKm - right.distanceKm)
    .filter((point, index, points) => index === 0 || point.distanceKm > (points[index - 1]?.distanceKm ?? -Infinity));
}

function interpolateElevation(points: ClimbDetails["profile"], distanceKm: number): number {
  const first = points[0];
  const last = points.at(-1);
  if (!first || !last) {
    return 0;
  }
  if (distanceKm <= first.distanceKm) {
    return first.elevation;
  }
  if (distanceKm >= last.distanceKm) {
    return last.elevation;
  }
  for (let index = 1; index < points.length; index += 1) {
    const previous = points[index - 1];
    const current = points[index];
    if (!previous || !current || current.distanceKm < distanceKm) {
      continue;
    }
    const span = current.distanceKm - previous.distanceKm;
    const ratio = span > 0 ? (distanceKm - previous.distanceKm) / span : 0;
    return previous.elevation + (current.elevation - previous.elevation) * ratio;
  }
  return last.elevation;
}

function gradientColor(gradient: number): string {
  if (gradient < 0) return "#f4f1e9";
  if (gradient < 4) return "#6eb35d";
  if (gradient < 6) return "#4f9fc4";
  if (gradient < 8) return "#f1cf45";
  if (gradient < 10) return "#df553f";
  return "#3e3a3d";
}

function gradientTextColor(gradient: number): string {
  return gradient >= 8 ? "#ffffff" : "#20262a";
}

function ascentSummaryMarkup(
  details: ClimbDetails,
  x: number,
  y: number,
  width: number,
  className: string,
): string {
  if (details.ascentCount <= 0) {
    return `<text x="${round(x)}" y="${round(y)}" class="${className}">ASCENT DETAILS UNAVAILABLE</text>`;
  }
  const ascentLabel = `${details.ascentCount} ASCENT${details.ascentCount === 1 ? "" : "S"}`;
  const summary = details.bestAscent
    ? `BEST ${formatAscent(details.bestAscent.date, details.bestAscent.durationSeconds)} · ${ascentLabel}`
    : `${ascentLabel} · BEST TIME UNAVAILABLE`;
  const maxCharacters = Math.max(28, Math.floor(width / 9));
  const fitAttributes = summary.length > maxCharacters
    ? ` textLength="${round(width)}" lengthAdjust="spacingAndGlyphs"`
    : "";
  return `<text x="${round(x)}" y="${round(y)}" class="${className}"${fitAttributes}>${escapeXml(summary)}</text>`;
}

function footerMarkup(options: ClimbPosterOptions, climbs: ClimbPosterEntry[], color: string): string {
  const athlete = options.athleteName?.trim() || "MY ACTIVITY STATS";
  const climbLabel = `${climbs.length} CLIMB${climbs.length === 1 ? "" : "S"}`;
  const ascentCount = totalAscents(climbs);
  const ascentLabel = `${ascentCount} ASCENT${ascentCount === 1 ? "" : "S"}`;
  const largePoster = climbs.length > 25;
  const width = largePoster ? LARGE_POSTER_WIDTH : STANDARD_POSTER_WIDTH;
  const height = largePoster ? LARGE_POSTER_HEIGHT : STANDARD_POSTER_HEIGHT;
  const margin = largePoster ? 84 : 82;
  const lineY = height - 110;
  const textY = height - 62;
  const fontSize = largePoster ? 19 : 16;
  return `${gradientLegendMarkup(color, width / 2, height - 136, largePoster)}
  <line x1="${margin}" y1="${lineY}" x2="${width - margin}" y2="${lineY}" style="stroke:${color};stroke-width:2"/>
  <text x="${margin}" y="${textY}" style="font:500 ${fontSize}px ui-sans-serif,system-ui;fill:${color};letter-spacing:0">${escapeXml(athlete.toUpperCase())}</text>
  <text x="${width / 2}" y="${textY}" text-anchor="middle" style="font:500 ${fontSize}px ui-sans-serif,system-ui;fill:${color};letter-spacing:0">${climbLabel} · ${ascentLabel}</text>
  <text x="${width - margin}" y="${textY}" text-anchor="end" style="font:500 ${fontSize}px ui-sans-serif,system-ui;fill:${color};letter-spacing:0">MYSTRAVASTATS</text>`;
}

function gradientLegendMarkup(color: string, centerX: number, y: number, largePoster: boolean): string {
  const entries = [
    { label: "DESC.", gradient: -1 },
    { label: "<4%", gradient: 2 },
    { label: "4–6%", gradient: 5 },
    { label: "6–8%", gradient: 7 },
    { label: "8–10%", gradient: 9 },
    { label: ">10%", gradient: 11 },
  ];
  const itemWidth = largePoster ? 92 : 70;
  const startX = centerX - (entries.length * itemWidth) / 2;
  const swatchWidth = largePoster ? 19 : 15;
  const swatchHeight = largePoster ? 15 : 12;
  const labelFontSize = largePoster ? 10 : 8;
  return `<g aria-label="Gradient colour legend">
    <text x="${round(startX - (largePoster ? 68 : 52))}" y="${y}" style="font:600 ${largePoster ? 11 : 9}px ui-monospace,monospace;fill:${color};letter-spacing:0">GRADE</text>
    ${entries.map((entry, index) => {
      const x = startX + index * itemWidth;
      return `<rect x="${round(x)}" y="${round(y - swatchHeight + 2)}" width="${swatchWidth}" height="${swatchHeight}" rx="1" fill="${gradientColor(entry.gradient)}" stroke="#283238" stroke-opacity=".2"/><text x="${round(x + swatchWidth + 6)}" y="${y}" style="font:600 ${labelFontSize}px ui-monospace,monospace;fill:${color}">${escapeXml(entry.label)}</text>`;
    }).join("")}
  </g>`;
}

function totalAscents(climbs: ClimbPosterEntry[]): number {
  return climbs.reduce((sum, climb) => sum + climb.details.ascentCount, 0);
}

function formatAscent(date: string, durationSeconds: number): string {
  return `${formatDate(date)}  ${formatDuration(durationSeconds)}`;
}

function formatDate(value: string): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (!match) {
    return value.slice(0, 10);
  }
  return `${match[3]}.${match[2]}.${match[1].slice(2)}`;
}

function formatDuration(totalSeconds: number): string {
  if (!Number.isFinite(totalSeconds) || totalSeconds <= 0) {
    return "—";
  }
  const seconds = Math.round(totalSeconds);
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const remainingSeconds = seconds % 60;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, "0")}:${String(remainingSeconds).padStart(2, "0")}`
    : `${minutes}:${String(remainingSeconds).padStart(2, "0")}`;
}

function formatDecimal(value: number): string {
  return Number.isFinite(value) ? value.toFixed(1) : "—";
}

function formatCompactDecimal(value: number): string {
  if (!Number.isFinite(value)) {
    return "—";
  }
  const rounded = Math.round(value * 10) / 10;
  return Number.isInteger(rounded) ? rounded.toFixed(0) : rounded.toFixed(1);
}

function formatMaximumGradient(value?: number | null): string {
  return value != null && Number.isFinite(value) ? value.toFixed(1) : "—";
}

function formatInteger(value: number): string {
  return Number.isFinite(value) ? Math.round(value).toLocaleString("en-US") : "—";
}

function fitTextAttributes(
  value: string,
  maxCharacters: number,
  width: number,
  baseFontSize: number,
  minimumFontSize: number,
  averageGlyphWidth = 0.56,
): string {
  if (value.length <= maxCharacters) {
    return "";
  }
  const fittedFontSize = Math.max(
    minimumFontSize,
    Math.min(baseFontSize, width / Math.max(1, value.length * averageGlyphWidth)),
  );
  return fittedFontSize < baseFontSize ? ` style="font-size:${round(fittedFontSize)}px"` : "";
}

function fittedTextFontSize(
  characterCount: number,
  width: number,
  baseFontSize: number,
  minimumFontSize: number,
  averageGlyphWidth = 0.56,
): number {
  return Math.max(
    minimumFontSize,
    Math.min(baseFontSize, width / Math.max(1, characterCount * averageGlyphWidth)),
  );
}

function splitPosterTitle(value: string, preferredLineLength: number): string[] {
  if (value.length <= preferredLineLength) {
    return [value];
  }
  const words = value.trim().split(/\s+/);
  if (words.length < 2) {
    return [value];
  }

  let bestSplit = 1;
  let bestScore = Number.POSITIVE_INFINITY;
  for (let index = 1; index < words.length; index += 1) {
    const firstLength = words.slice(0, index).join(" ").length;
    const secondLength = words.slice(index).join(" ").length;
    const overflow = Math.max(0, firstLength - preferredLineLength) + Math.max(0, secondLength - preferredLineLength);
    const score = overflow * 4 + Math.abs(firstLength - secondLength);
    if (score < bestScore) {
      bestScore = score;
      bestSplit = index;
    }
  }
  return [words.slice(0, bestSplit).join(" "), words.slice(bestSplit).join(" ")];
}

function round(value: number): number {
  return Math.round(value * 10) / 10;
}

function escapeXml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&apos;");
}
