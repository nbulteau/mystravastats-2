import type { ClimbDetails } from "@/models/badge-check-result.model";

export type ClimbPosterDesign = "altitude" | "topo" | "collection";

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
    id: "altitude",
    name: "Altitude",
    description: "Profile-first, warm and minimal",
    maxClimbs: 50,
  },
  {
    id: "topo",
    name: "Topo log",
    description: "Graph paper and technical ride data",
    maxClimbs: 50,
  },
  {
    id: "collection",
    name: "Collection",
    description: "Editorial cards with country and massif",
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
    throw new Error("Select at least one climbed col before generating a poster.");
  }
  const largePoster = climbs.length > 25;
  const posterWidth = largePoster ? LARGE_POSTER_WIDTH : STANDARD_POSTER_WIDTH;
  const posterHeight = largePoster ? LARGE_POSTER_HEIGHT : STANDARD_POSTER_HEIGHT;

  const content = options.design === "altitude"
    ? buildAltitudeDesign(climbs, options)
    : options.design === "topo"
      ? buildTopoDesign(climbs, options)
      : buildCollectionDesign(climbs, options);

  return `<svg xmlns="http://www.w3.org/2000/svg" width="${posterWidth}" height="${posterHeight}" viewBox="0 0 ${posterWidth} ${posterHeight}" role="img" aria-labelledby="poster-title poster-description">
  <title id="poster-title">${escapeXml(posterTitle(options.design))}</title>
  <desc id="poster-description">Cycling col poster with altitude profiles, gradients, best ascent dates and times, and ascent counts.</desc>
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
  return label.replace(/\s+from\s+/i, " depuis ");
}

function buildAltitudeDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  if (climbs.length > 3) {
    return buildDenseAltitudeDesign(climbs, options);
  }
  const top = 250;
  const availableHeight = 1245;
  const rowHeight = availableHeight / climbs.length;
  const rows = climbs.map((climb, index) => {
    const y = top + index * rowHeight;
    const profileTop = y + 46;
    const profileHeight = Math.max(68, Math.min(150, rowHeight - 116));
    const statsY = profileTop + profileHeight + 25;
    const ascentY = statsY + 27;
    const name = posterClimbName(climb).toUpperCase();
    return `<g>
      ${index > 0 ? `<line x1="82" y1="${round(y - 18)}" x2="1118" y2="${round(y - 18)}" class="rule"/>` : ""}
      <text x="82" y="${round(y + 25)}" class="index">${String(index + 1).padStart(2, "0")}</text>
      <text x="155" y="${round(y + 25)}" class="climb-name"${fitTextAttributes(name, 28, 700, 31, 18)}>${escapeXml(name)}</text>
      <text x="1118" y="${round(y + 25)}" text-anchor="end" class="summit">ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS</text>
      ${profileMarkup(climb.details, 155, profileTop, 963, profileHeight, "profile-altitude")}
      <text x="155" y="${round(statsY)}" class="stat">${formatDecimal(climb.details.lengthKm)} KM</text>
      <text x="365" y="${round(statsY)}" class="stat">${formatDecimal(climb.details.averageGradient)} % AVG</text>
      <text x="620" y="${round(statsY)}" class="stat">${formatMaximumGradient(climb.details.maximumGradient)} % MAX</text>
      <text x="865" y="${round(statsY)}" class="stat">+${formatInteger(climb.details.totalAscent)} M</text>
      ${ascentSummaryMarkup(climb.details, 155, ascentY, 963, "ascent")}
    </g>`;
  }).join("");

  return `<style>
    .paper{fill:#f8f4ec}.ink{fill:#16202a}.muted{fill:#68727a}.accent-fill{fill:#df6b35;fill-opacity:.16}.accent{stroke:#df6b35}.rule{stroke:#c8c1b6;stroke-width:2}.index{font:500 23px ui-sans-serif,system-ui;fill:#8a8f91;letter-spacing:2px}.climb-name{font:500 31px ui-sans-serif,system-ui;fill:#16202a;letter-spacing:.5px}.summit{font:500 23px ui-sans-serif,system-ui;fill:#16202a}.stat{font:500 18px ui-sans-serif,system-ui;fill:#4d5961;letter-spacing:.6px}.ascent{font:400 17px ui-monospace,monospace;fill:#68727a}.profile-altitude{fill:none;stroke:#df6b35;stroke-width:5;stroke-linecap:round;stroke-linejoin:round}.profile-area{fill:#df6b35;fill-opacity:.13}.profile-base{stroke:#bdb7ad;stroke-width:2}.profile-missing{font:400 16px ui-sans-serif,system-ui;fill:#8a8f91;letter-spacing:1px}
  </style>
  <rect width="1200" height="1680" class="paper"/>
  <text x="82" y="92" class="muted" style="font:500 18px ui-sans-serif,system-ui;letter-spacing:5px">CYCLING ASCENTS · ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="82" y="182" class="ink" style="font:500 76px ui-sans-serif,system-ui;letter-spacing:-3px">MY COLS</text>
  <text x="1118" y="178" text-anchor="end" class="ink" style="font:400 31px ui-sans-serif,system-ui;letter-spacing:2px">${escapeXml(options.yearLabel)}</text>
  <line x1="82" y1="214" x2="1118" y2="214" style="stroke:#16202a;stroke-width:3"/>
  ${rows}
  ${footerMarkup(options, climbs, "#68727a")}`;
}

function buildTopoDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  if (climbs.length > 3) {
    return buildDenseTopoDesign(climbs, options);
  }
  const top = 240;
  const availableHeight = 1280;
  const rowHeight = availableHeight / climbs.length;
  const rows = climbs.map((climb, index) => {
    const y = top + index * rowHeight;
    const profileHeight = Math.max(62, Math.min(125, rowHeight - 94));
    const profileTop = y + 40;
    const ascentY = profileTop + profileHeight + 25;
    const name = posterClimbName(climb).toUpperCase();
    return `<g>
      <circle cx="95" cy="${round(y + 15)}" r="10" class="summit-dot"/>
      <text x="132" y="${round(y - 2)}" class="code">ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS · CAT. ${escapeXml(climb.category ?? "—")}</text>
      <text x="132" y="${round(y + 27)}" class="topo-name"${fitTextAttributes(name, 27, 690, 26, 16, 0.6)}>${escapeXml(name)}</text>
      ${profileMarkup(climb.details, 132, profileTop, 710, profileHeight, "profile-topo")}
      <text x="880" y="${round(y + 7)}" class="metric">${formatDecimal(climb.details.lengthKm)} KM</text>
      <text x="880" y="${round(y + 34)}" class="metric">+${formatInteger(climb.details.totalAscent)} M</text>
      <text x="1040" y="${round(y + 7)}" class="metric">AVG ${formatDecimal(climb.details.averageGradient)} %</text>
      <text x="1040" y="${round(y + 34)}" class="metric">MAX ${formatMaximumGradient(climb.details.maximumGradient)} %</text>
      ${ascentSummaryMarkup(climb.details, 132, ascentY, 986, "topo-ascent")}
    </g>`;
  }).join("");

  return `<defs><pattern id="grid" width="32" height="32" patternUnits="userSpaceOnUse"><path d="M32 0H0V32" fill="none" stroke="#293541" stroke-opacity=".1" stroke-width="1"/></pattern></defs>
  <style>
    .paper{fill:#edf0eb}.grid{fill:url(#grid)}.topo-ink{fill:#1d2a32}.topo-muted{fill:#65727a}.axis{stroke:#1d2a32;stroke-width:4}.summit-dot{fill:#d65932;stroke:#edf0eb;stroke-width:5}.code{font:500 15px ui-monospace,monospace;fill:#65727a;letter-spacing:2px}.topo-name{font:500 26px ui-monospace,monospace;fill:#1d2a32}.metric{font:500 17px ui-monospace,monospace;fill:#1d2a32}.topo-ascent{font:400 15px ui-monospace,monospace;fill:#65727a}.profile-topo{fill:none;stroke:#d65932;stroke-width:4;stroke-linecap:round;stroke-linejoin:round}.profile-area{fill:#d65932;fill-opacity:.12}.profile-base{stroke:#849098;stroke-width:2}.profile-missing{font:400 15px ui-monospace,monospace;fill:#65727a}
  </style>
  <rect width="1200" height="1680" class="paper"/><rect width="1200" height="1680" class="grid"/>
  <text x="76" y="84" class="topo-muted" style="font:500 17px ui-monospace,monospace;letter-spacing:3px">RIDE LOG / COLS / ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="76" y="164" class="topo-ink" style="font:500 62px ui-monospace,monospace;letter-spacing:-2px">${String(climbs.length).padStart(2, "0")} SUMMITS</text>
  <text x="1124" y="124" text-anchor="end" class="topo-ink" style="font:500 36px ui-monospace,monospace;letter-spacing:1px">TECHNICAL</text>
  <text x="1124" y="151" text-anchor="end" class="topo-muted" style="font:500 15px ui-monospace,monospace;letter-spacing:2px">PROFILE INDEX</text>
  <line x1="95" y1="${top + 15}" x2="95" y2="1510" class="axis"/>
  ${rows}
  ${footerMarkup(options, climbs, "#65727a")}`;
}

function buildCollectionDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  if (climbs.length > 3) {
    return buildDenseCollectionDesign(climbs, options);
  }
  const columnWidth = 344;
  const startX = 72;
  const columns = climbs.map((climb, index) => {
    const x = startX + index * columnWidth;
    const profileX = x + 72;
    const profileWidth = 238;
    const ascents = ascentSummaryMarkup(climb.details, profileX, 1125, profileWidth, "collection-ascent");
    const name = posterClimbName(climb).toUpperCase();
    return `<g>
      ${index > 0 ? `<line x1="${x - 18}" y1="250" x2="${x - 18}" y2="1480" class="column-rule"/>` : ""}
      <text x="${x}" y="285" class="collection-index">${String(index + 1).padStart(2, "0")}</text>
      <text transform="translate(${x + 34} 780) rotate(-90)" class="vertical-name"${fitTextAttributes(name, 28, 470, 27, 14)}>${escapeXml(name)}</text>
      ${profileMarkup(climb.details, profileX, 380, profileWidth, 420, "profile-collection")}
      <text x="${profileX}" y="825" class="collection-alt"${fitTextAttributes(`ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS`, 18, profileWidth, 18, 11)}>ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS</text>
      <text x="${profileX}" y="915" class="collection-distance">${formatDecimal(climb.details.lengthKm)}</text>
      <text x="${profileX + 182}" y="915" class="collection-unit">KM</text>
      <text x="${profileX}" y="960" class="collection-stat">${formatDecimal(climb.details.averageGradient)} % AVG</text>
      <text x="${profileX}" y="995" class="collection-stat">${formatMaximumGradient(climb.details.maximumGradient)} % MAX</text>
      <text x="${profileX}" y="1030" class="collection-stat">+${formatInteger(climb.details.totalAscent)} M</text>
      <line x1="${profileX}" y1="1065" x2="${profileX + profileWidth}" y2="1065" class="column-rule"/>
      ${ascents}
    </g>`;
  }).join("");

  return `<style>
    .paper{fill:#f2eee4}.collection-ink{fill:#1f2022}.collection-muted{fill:#6d6b67}.column-rule{stroke:#c7bfb1;stroke-width:2}.collection-index{font:500 21px ui-sans-serif,system-ui;fill:#77736c;letter-spacing:2px}.vertical-name{font:500 27px ui-sans-serif,system-ui;fill:#1f2022;letter-spacing:2px}.collection-alt{font:500 18px ui-sans-serif,system-ui;fill:#1f2022}.collection-distance{font:500 60px ui-sans-serif,system-ui;fill:#1f2022;letter-spacing:-2px}.collection-unit{font:500 18px ui-sans-serif,system-ui;fill:#6d6b67}.collection-stat{font:500 18px ui-sans-serif,system-ui;fill:#6d6b67;letter-spacing:.5px}.collection-ascent{font:400 16px ui-monospace,monospace;fill:#6d6b67}.profile-collection{fill:none;stroke:#ca5430;stroke-width:6;stroke-linecap:round;stroke-linejoin:round}.profile-area{fill:#ca5430;fill-opacity:.16}.profile-base{stroke:#b9b1a4;stroke-width:2}.profile-missing{font:400 14px ui-sans-serif,system-ui;fill:#77736c}
  </style>
  <rect width="1200" height="1680" class="paper"/>
  <text x="72" y="92" class="collection-muted" style="font:500 17px ui-sans-serif,system-ui;letter-spacing:5px">THE GIANTS COLLECTION</text>
  <text x="72" y="185" class="collection-ink" style="font:500 68px ui-sans-serif,system-ui;letter-spacing:-2px">COLS · ${String(climbs.length).padStart(2, "0")}</text>
  <text x="1128" y="181" text-anchor="end" class="collection-ink" style="font:400 28px ui-sans-serif,system-ui;letter-spacing:2px">${escapeXml(options.yearLabel)}</text>
  <line x1="72" y1="220" x2="1128" y2="220" style="stroke:#1f2022;stroke-width:3"/>
  ${columns}
  ${footerMarkup(options, climbs, "#6d6b67")}`;
}

function buildDenseAltitudeDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  const geometry = densePosterGeometry(climbs, 240, 1265);
  const { largePoster, width, height, margin, right, top, contentHeight } = geometry;
  const columnCount = DENSE_COLUMN_COUNT;
  const columnGap = largePoster ? 24 : 16;
  const columnWidth = (width - margin * 2 - columnGap * (columnCount - 1)) / columnCount;
  const rowCount = Math.ceil(climbs.length / columnCount);
  const rowHeight = contentHeight / rowCount;
  const rows = climbs.map((climb, index) => {
    const column = index % columnCount;
    const row = Math.floor(index / columnCount);
    const x = margin + column * (columnWidth + columnGap);
    const y = top + row * rowHeight;
    const profileTop = y + 50;
    const tileBottom = y + rowHeight - 22;
    const ascentY = tileBottom - 9;
    const statsY = ascentY - 23;
    const profileBottom = statsY - 15;
    const profileHeight = Math.max(70, profileBottom - profileTop);
    const name = posterClimbName(climb).toUpperCase();
    return `<g data-grid-column="${column}" data-grid-row="${row}" data-profile-height="${round(profileHeight)}">
      ${row > 0 ? `<line x1="${round(x)}" y1="${round(y - 14)}" x2="${round(x + columnWidth)}" y2="${round(y - 14)}" class="dense-rule"/>` : ""}
      ${column > 0 ? `<line x1="${round(x - columnGap / 2)}" y1="${round(y)}" x2="${round(x - columnGap / 2)}" y2="${round(y + rowHeight - 22)}" class="dense-divider"/>` : ""}
      <text x="${round(x)}" y="${round(y + 23)}" class="dense-index">${String(index + 1).padStart(2, "0")}</text>
      <text x="${round(x + 35)}" y="${round(y + 23)}" class="dense-name"${fitTextAttributes(name, 17, columnWidth - 45, 16, 8.5)}>${escapeXml(name)}</text>
      <text x="${round(x + columnWidth)}" y="${round(y + 46)}" text-anchor="end" class="dense-summit">ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS</text>
      ${profileMarkup(climb.details, x, profileTop, columnWidth, profileHeight, "dense-profile")}
      <text x="${round(x)}" y="${round(statsY)}" class="dense-stat"${fitTextAttributes(`${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M · ${formatDecimal(climb.details.averageGradient)} % AVG · ${formatMaximumGradient(climb.details.maximumGradient)} % MAX`, 52, columnWidth, 11, 7.5, 0.5)}>${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M · ${formatDecimal(climb.details.averageGradient)} % AVG · ${formatMaximumGradient(climb.details.maximumGradient)} % MAX</text>
      ${ascentSummaryMarkup(climb.details, x, ascentY, columnWidth, "dense-ascent")}
    </g>`;
  }).join("");

  return `<style>
    .dense-paper{fill:#fbf8f1}.dense-ink{fill:#172129}.dense-muted{fill:#756c62}.dense-rule{stroke:#d8cfc1;stroke-width:1.2}.dense-divider{stroke:#d65d2d;stroke-opacity:.16;stroke-width:1.2}.dense-index{font:500 14px ui-sans-serif,system-ui;fill:#d65d2d;letter-spacing:1px}.dense-name{font:500 16px ui-sans-serif,system-ui;fill:#172129}.dense-summit{font:500 10px ui-sans-serif,system-ui;fill:#756c62}.dense-stat{font:500 11px ui-sans-serif,system-ui;fill:#4d565c;letter-spacing:0}.dense-ascent{font:400 10px ui-monospace,monospace;fill:#756c62}.dense-profile{fill:none;stroke:#d65d2d;stroke-width:2.6;stroke-linecap:round;stroke-linejoin:round}.profile-base{stroke:#a9a196;stroke-width:1}.profile-missing{font:400 12px ui-sans-serif,system-ui;fill:#8a8178;letter-spacing:1px}
  </style>
  <rect width="${width}" height="${height}" class="dense-paper"/>
  <text x="${margin}" y="${largePoster ? 118 : 84}" class="dense-muted" style="font:500 ${largePoster ? 21 : 17}px ui-sans-serif,system-ui;letter-spacing:5px">CYCLING ASCENTS · ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="${margin}" y="${largePoster ? 245 : 175}" class="dense-ink" style="font:500 ${largePoster ? 96 : 70}px ui-sans-serif,system-ui;letter-spacing:-3px">MY COLS</text>
  <text x="${right}" y="${largePoster ? 239 : 171}" text-anchor="end" class="dense-ink" style="font:400 ${largePoster ? 36 : 28}px ui-sans-serif,system-ui;letter-spacing:2px">${escapeXml(options.yearLabel)}</text>
  <line x1="${margin}" y1="${top - 33}" x2="${right}" y2="${top - 33}" style="stroke:#16202a;stroke-width:3"/>
  ${rows}
  ${footerMarkup(options, climbs, "#68727a")}`;
}

function buildDenseTopoDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  const geometry = densePosterGeometry(climbs, 235, 1270);
  const { largePoster, width, height, margin, right, top, contentHeight } = geometry;
  const columnCount = DENSE_COLUMN_COUNT;
  const columnGap = largePoster ? 24 : 16;
  const columnWidth = (width - margin * 2 - columnGap * (columnCount - 1)) / columnCount;
  const rowCount = Math.ceil(climbs.length / columnCount);
  const rowHeight = contentHeight / rowCount;
  const fiveRowLayout = rowCount >= 5;
  const tenRowLayout = rowCount >= 8;
  const rows = climbs.map((climb, index) => {
    const column = index % columnCount;
    const row = Math.floor(index / columnCount);
    const x = margin + column * (columnWidth + columnGap);
    const y = top + row * rowHeight;
    const profileTop = y + 52;
    const tileBottom = y + rowHeight - 18;
    const ascentY = tileBottom - 8;
    const averageY = ascentY - 18;
    const metricsY = averageY - 23;
    const profileBottom = metricsY - 15;
    const profileHeight = Math.max(70, profileBottom - profileTop);
    const name = posterClimbName(climb).toUpperCase();
    return `<g data-grid-column="${column}" data-grid-row="${row}" data-profile-height="${round(profileHeight)}">
      <rect x="${round(x - 7)}" y="${round(y - 6)}" width="${round(columnWidth + 14)}" height="${round(rowHeight - 9)}" rx="7" class="dense-topo-card"/>
      ${column > 0 ? `<line x1="${round(x - columnGap / 2)}" y1="${round(y)}" x2="${round(x - columnGap / 2)}" y2="${round(y + rowHeight - 18)}" class="dense-topo-divider"/>` : ""}
      ${row > 0 ? `<line x1="${round(x)}" y1="${round(y - 12)}" x2="${round(x + columnWidth)}" y2="${round(y - 12)}" class="dense-topo-divider"/>` : ""}
      <circle cx="${round(x + 8)}" cy="${round(y + 17)}" r="7" class="dense-topo-dot"/>
      <text x="${round(x + 25)}" y="${round(y + 5)}" class="dense-topo-code">ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS · CAT. ${escapeXml(climb.category ?? "—")}</text>
      <text x="${round(x + 25)}" y="${round(y + 31)}" class="dense-topo-name"${fitTextAttributes(name, 17, columnWidth - 35, 14, 8, 0.6)}>${escapeXml(name)}</text>
      ${profileMarkup(climb.details, x, profileTop, columnWidth, profileHeight, "dense-topo-profile")}
      <text x="${round(x)}" y="${round(metricsY)}" class="dense-topo-metric">${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M</text>
      <text x="${round(x)}" y="${round(averageY)}" class="dense-topo-metric">AVG ${formatDecimal(climb.details.averageGradient)} % · MAX ${formatMaximumGradient(climb.details.maximumGradient)} %</text>
      ${ascentSummaryMarkup(climb.details, x, ascentY, columnWidth, "dense-topo-ascent")}
    </g>`;
  }).join("");

  return `<defs><pattern id="dense-grid" width="28" height="28" patternUnits="userSpaceOnUse"><path d="M28 0H0V28" fill="none" stroke="#293541" stroke-opacity=".09" stroke-width="1"/></pattern></defs>
  <style>
    .dense-topo-paper{fill:#e8eeec}.dense-topo-grid{fill:url(#dense-grid)}.dense-topo-card{fill:#f7faf8;fill-opacity:.7;stroke:#263942;stroke-opacity:.24;stroke-width:1.2}.dense-topo-ink{fill:#19303a}.dense-topo-muted{fill:#526a71}.dense-topo-divider{stroke:#19303a;stroke-opacity:.14;stroke-width:1}.dense-topo-dot{fill:#d34f2f;stroke:#f7faf8;stroke-width:3}.dense-topo-code{font:500 9px ui-monospace,monospace;fill:#526a71;letter-spacing:.5px}.dense-topo-name{font:600 14px ui-monospace,monospace;fill:#19303a}.dense-topo-metric{font:500 11px ui-monospace,monospace;fill:#19303a}.dense-topo-ascent{font:400 10px ui-monospace,monospace;fill:#526a71}.dense-topo-profile{fill:none;stroke:#19303a;stroke-width:2.2;stroke-linecap:square;stroke-linejoin:miter}.profile-base{stroke:#526a71;stroke-width:1.1;stroke-dasharray:3 3}.profile-missing{font:400 11px ui-monospace,monospace;fill:#526a71}
  </style>
  <rect width="${width}" height="${height}" class="dense-topo-paper"/><rect width="${width}" height="${height}" class="dense-topo-grid"/>
  <text x="${margin}" y="${largePoster ? 112 : 78}" class="dense-topo-muted" style="font:500 ${largePoster ? 20 : 16}px ui-monospace,monospace;letter-spacing:3px">RIDE LOG / COLS / ${escapeXml(options.yearLabel.toUpperCase())}</text>
  <text x="${margin}" y="${largePoster ? 226 : 158}" class="dense-topo-ink" style="font:500 ${largePoster ? 82 : 58}px ui-monospace,monospace;letter-spacing:-2px">${String(climbs.length).padStart(2, "0")} SUMMITS</text>
  <text x="${right}" y="${largePoster ? 170 : 120}" text-anchor="end" class="dense-topo-ink" style="font:500 ${largePoster ? 42 : 31}px ui-monospace,monospace;letter-spacing:1px">TECHNICAL</text>
  <text x="${right}" y="${largePoster ? 207 : 146}" text-anchor="end" class="dense-topo-muted" style="font:500 ${largePoster ? 18 : 14}px ui-monospace,monospace;letter-spacing:2px">PROFILE INDEX</text>
  ${rows}
  ${footerMarkup(options, climbs, "#65727a")}`;
}

function buildDenseCollectionDesign(climbs: ClimbPosterEntry[], options: ClimbPosterOptions): string {
  const geometry = densePosterGeometry(climbs, 235, 1270);
  const { largePoster, width, height, margin, right, top, contentHeight } = geometry;
  const columnCount = DENSE_COLUMN_COUNT;
  const columnGap = largePoster ? 24 : 16;
  const columnWidth = (width - margin * 2 - columnGap * (columnCount - 1)) / columnCount;
  const rowCount = Math.ceil(climbs.length / columnCount);
  const rowHeight = contentHeight / rowCount;
  const fiveRowLayout = rowCount >= 5;
  const tenRowLayout = rowCount >= 8;
  const tiles = climbs.map((climb, index) => {
    const column = index % columnCount;
    const row = Math.floor(index / columnCount);
    const x = margin + column * (columnWidth + columnGap);
    const y = top + row * rowHeight;
    const name = posterClimbName(climb).toUpperCase();
    const location = `${climb.details.country} · ${climb.details.massif}`.toUpperCase();
    const profileTop = y + (tenRowLayout ? 58 : fiveRowLayout ? 60 : 63);
    const tileBottom = y + rowHeight - 22;
    const ascentY = tileBottom - 8;
    const averageY = ascentY - (tenRowLayout ? 19 : fiveRowLayout ? 20 : 21);
    const metricsY = averageY - (tenRowLayout ? 21 : fiveRowLayout ? 22 : 23);
    const profileBottom = metricsY - (tenRowLayout ? 16 : fiveRowLayout ? 16 : 18);
    const profileHeight = Math.max(70, profileBottom - profileTop);
    const metrics = `${formatDecimal(climb.details.lengthKm)} KM · +${formatInteger(climb.details.totalAscent)} M · ALT MAX ${formatInteger(climb.details.summitAltitude)} M · DIFFICULTÉ ${formatInteger(climb.details.difficulty)} PTS`;
    const metricsFit = fitTextAttributes(metrics, 32, columnWidth, 13, 7, 0.56);
    return `<g data-grid-column="${column}" data-grid-row="${row}" data-profile-bottom-y="${round(profileBottom)}" data-metrics-y="${round(metricsY)}" data-ascent-y="${round(ascentY)}" data-tile-bottom-y="${round(tileBottom)}">
      <rect x="${round(x - 5)}" y="${round(y - 7)}" width="${round(columnWidth + 10)}" height="${round(rowHeight - 10)}" rx="10" class="dense-collection-card"/>
      <rect x="${round(x - 5)}" y="${round(y - 7)}" width="4" height="${round(rowHeight - 10)}" rx="2" class="dense-collection-accent"/>
      <text x="${round(x)}" y="${round(y + 24)}" class="dense-collection-index">${String(index + 1).padStart(2, "0")}</text>
      <text x="${round(x + 34)}" y="${round(y + 24)}" class="dense-collection-name"${fitTextAttributes(name, 17, columnWidth - 44, 15, 8.5)}>${escapeXml(name)}</text>
      <text x="${round(x + 34)}" y="${round(y + 43)}" class="dense-collection-location"${fitTextAttributes(location, 24, columnWidth - 44, 9, 6.5, 0.52)}>${escapeXml(location)}</text>
      ${profileMarkup(climb.details, x, profileTop, columnWidth, profileHeight, "dense-collection-profile")}
      <text x="${round(x)}" y="${round(metricsY)}" class="dense-collection-metrics"${metricsFit}>${escapeXml(metrics)}</text>
      <text x="${round(x)}" y="${round(averageY)}" class="dense-collection-stat">${formatDecimal(climb.details.averageGradient)} % AVG · ${formatMaximumGradient(climb.details.maximumGradient)} % MAX</text>
      ${ascentSummaryMarkup(climb.details, x, ascentY, columnWidth, "dense-collection-ascent")}
    </g>`;
  }).join("");

  return `<style>
    .dense-collection-paper{fill:#efe7d9}.dense-collection-card{fill:#fbf8f1;stroke:#c8bca9;stroke-width:1}.dense-collection-accent{fill:#b74d2d}.dense-collection-ink{fill:#24201d}.dense-collection-muted{fill:#756b61}.dense-collection-index{font:600 13px ui-sans-serif,system-ui;fill:#b74d2d;letter-spacing:1px}.dense-collection-name{font:600 15px Georgia,ui-serif,serif;fill:#24201d}.dense-collection-location{font:600 9px ui-sans-serif,system-ui;fill:#8a6d5e;letter-spacing:.8px}.dense-collection-metrics{font:600 12px ui-sans-serif,system-ui;fill:#24201d;letter-spacing:-.1px}.dense-collection-stat{font:500 11px ui-sans-serif,system-ui;fill:#756b61}.dense-collection-ascent{font:400 10px ui-monospace,monospace;fill:#756b61}.dense-collection-profile{fill:none;stroke:#9f4229;stroke-width:2.3;stroke-linecap:round;stroke-linejoin:round}.profile-base{stroke:#9b8f81;stroke-width:1}.profile-missing{font:400 11px ui-sans-serif,system-ui;fill:#77736c}
  </style>
  <rect width="${width}" height="${height}" class="dense-collection-paper"/>
  <text x="${margin}" y="${largePoster ? 116 : 82}" class="dense-collection-muted" style="font:500 ${largePoster ? 20 : 16}px ui-sans-serif,system-ui;letter-spacing:5px">THE GIANTS COLLECTION</text>
  <text x="${margin}" y="${largePoster ? 244 : 174}" class="dense-collection-ink" style="font:500 ${largePoster ? 90 : 64}px ui-sans-serif,system-ui;letter-spacing:-2px">COLS · ${String(climbs.length).padStart(2, "0")}</text>
  <text x="${right}" y="${largePoster ? 238 : 170}" text-anchor="end" class="dense-collection-ink" style="font:400 ${largePoster ? 36 : 27}px ui-sans-serif,system-ui;letter-spacing:2px">${escapeXml(options.yearLabel)}</text>
  <line x1="${margin}" y1="${top - 28}" x2="${right}" y2="${top - 28}" style="stroke:#1f2022;stroke-width:3"/>
  ${tiles}
  ${footerMarkup(options, climbs, "#6d6b67")}`;
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
  const maxSegments = Math.min(20, Math.max(4, Math.floor(width / 28)));
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
    const labelFontSize = height >= 100 ? 9 : 7.5;
    const gradeText = segmentWidth >= 27
      ? `<text x="${round(labelX)}" y="${round(plotBaseY - 4)}" text-anchor="middle" style="font:700 ${labelFontSize}px ui-monospace,monospace;fill:${textColor}">${gradeLabel}</text>`
      : segmentWidth >= 13 && height >= 70
        ? `<text transform="translate(${round(labelX)} ${round(plotBaseY - 4)}) rotate(-90)" text-anchor="start" style="font:700 ${labelFontSize}px ui-monospace,monospace;fill:${textColor}">${gradeLabel}</text>`
        : "";
    const distanceText = segmentWidth >= 24
      ? `<text x="${round(endX - 1)}" y="${round(baseY - 2)}" text-anchor="end" style="font:500 ${height >= 80 ? 7.5 : 6.5}px ui-monospace,monospace;fill:#4d5961;fill-opacity:.9">${distanceLabel}</text>`
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
  const altitudeMarkup = height >= 70
    ? altitudeLabelBoundaries.map((boundary) => {
      const index = altitudeBoundaries.indexOf(boundary);
      const isLast = index === altitudeBoundaries.length - 1;
      const px = profileX(boundary.distanceKm);
      const py = profileY(boundary.elevation);
      const anchor = index === 0 ? "start" : isLast ? "end" : "middle";
      const offsetX = index === 0 ? 2 : isLast ? -2 : 0;
      const altitudeOffsetY = index === 0 ? (height >= 100 ? 18 : 15) : 4;
      return `<text data-profile-altitude-label="${formatInteger(boundary.elevation)}" x="${round(px + offsetX)}" y="${round(Math.max(y + 9, py - altitudeOffsetY))}" text-anchor="${anchor}" style="font:600 ${height >= 100 ? 8 : 7}px ui-monospace,monospace;fill:#283238;paint-order:stroke;stroke:#ffffff;stroke-width:2px;stroke-opacity:.72">${formatInteger(boundary.elevation)} M</text>`;
    }).join("")
    : "";
  return `<line x1="${round(x)}" y1="${round(plotBaseY)}" x2="${round(x + width)}" y2="${round(plotBaseY)}" class="profile-base"/>${segmentMarkup}<path d="${linePath}" class="${lineClass}"/>${altitudeMarkup}`;
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
  const climbLabel = `${climbs.length} COL${climbs.length === 1 ? "" : "S"}`;
  const ascentCount = totalAscents(climbs);
  const ascentLabel = `${ascentCount} ASCENSION${ascentCount === 1 ? "" : "S"}`;
  const largePoster = climbs.length > 25;
  const width = largePoster ? LARGE_POSTER_WIDTH : STANDARD_POSTER_WIDTH;
  const height = largePoster ? LARGE_POSTER_HEIGHT : STANDARD_POSTER_HEIGHT;
  const margin = largePoster ? 84 : 82;
  const lineY = height - 110;
  const textY = height - 62;
  const fontSize = largePoster ? 19 : 16;
  return `${gradientLegendMarkup(color, width / 2, height - 136, largePoster)}
  <line x1="${margin}" y1="${lineY}" x2="${width - margin}" y2="${lineY}" style="stroke:${color};stroke-width:2"/>
  <text x="${margin}" y="${textY}" style="font:500 ${fontSize}px ui-sans-serif,system-ui;fill:${color};letter-spacing:2px">${escapeXml(athlete.toUpperCase())}</text>
  <text x="${width / 2}" y="${textY}" text-anchor="middle" style="font:500 ${fontSize}px ui-sans-serif,system-ui;fill:${color};letter-spacing:2px">${climbLabel} · ${ascentLabel}</text>
  <text x="${width - margin}" y="${textY}" text-anchor="end" style="font:500 ${fontSize}px ui-sans-serif,system-ui;fill:${color};letter-spacing:2px">MYSTRAVASTATS</text>`;
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
    <text x="${round(startX - (largePoster ? 68 : 52))}" y="${y}" style="font:600 ${largePoster ? 11 : 9}px ui-monospace,monospace;fill:${color};letter-spacing:1px">PENTE</text>
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
  return Number.isFinite(value) ? value.toFixed(1).replace(".", ",") : "—";
}

function formatCompactDecimal(value: number): string {
  if (!Number.isFinite(value)) {
    return "—";
  }
  const rounded = Math.round(value * 10) / 10;
  return (Number.isInteger(rounded) ? rounded.toFixed(0) : rounded.toFixed(1)).replace(".", ",");
}

function formatMaximumGradient(value?: number | null): string {
  return value != null && Number.isFinite(value) ? value.toFixed(1).replace(".", ",") : "—";
}

function formatInteger(value: number): string {
  return Number.isFinite(value) ? Math.round(value).toLocaleString("fr-FR") : "—";
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
