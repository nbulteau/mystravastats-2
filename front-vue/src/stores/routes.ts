import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import {
  type GenerateRoutesResponse,
  type GeneratedRoute,
  type RouteMode,
  type RouteType,
  type RouteGenerationDiagnostic,
  type ShapeInputType,
} from "@/models/route-recommendation.model";

const DEFAULT_VARIANT_COUNT = 4;
const GEOMETRY_SIGNATURE_PRECISION = 5;
const GEOMETRY_SIGNATURE_MAX_POINTS = 80;
const SHAPE_TRANSFORM_HISTORY_LIMIT = 25;
const SAVED_SHAPES_STORAGE_KEY = "mystravastats:strava-art:saved-shapes";
const DEFAULT_TEMPLATE_CENTER = { lat: 45.1885, lng: 5.7245 };

type RoutingHealthStatus = "unknown" | "up" | "down" | "disabled" | "misconfigured";
const ALL_ROUTE_TYPES: RouteType[] = ["RIDE", "MTB", "GRAVEL", "RUN", "TRAIL", "HIKE"];
const CYCLING_ROUTE_TYPES: RouteType[] = ["RIDE", "MTB", "GRAVEL"];
const FOOT_ROUTE_TYPES: RouteType[] = ["RUN", "TRAIL", "HIKE"];
const CAR_ROUTE_TYPES: RouteType[] = ["RIDE"];

interface RoutingHealthPayload {
  routing?: {
    status?: string;
    reachable?: boolean;
    engine?: string;
    enabled?: boolean;
    profile?: string;
    extractProfile?: string;
    effectiveProfile?: string;
    supportedRouteTypes?: string[];
  };
}

interface ImportShapeFromGpxOptions {
  append?: boolean;
}

interface FitShapeToStartOptions {
  viewportRadiusKm?: number;
  targetRadiusKm?: number;
  minRadiusKm?: number;
  maxRadiusKm?: number;
}

export type BuiltInShapeTemplateKey =
  | "heart"
  | "star"
  | "circle"
  | "square"
  | "triangle"
  | "diamond"
  | "rectangle"
  | "hexagon";

export interface SavedShapeTemplate {
  id: string;
  name: string;
  points: number[][];
  createdAt: string;
  updatedAt: string;
}

function cloneShapePoints(points: number[][]): number[][] {
  return points.map((point) => [point[0], point[1]]);
}

function sanitizeShapePoints(points: number[][]): number[][] {
  return points
    .filter((point) =>
      point.length >= 2
      && Number.isFinite(point[0])
      && Number.isFinite(point[1])
      && point[0] >= -90
      && point[0] <= 90
      && point[1] >= -180
      && point[1] <= 180
    )
    .map((point) => [point[0], point[1]]);
}

function shapeDataTextFor(points: number[][]): string {
  return points.length >= 2 ? JSON.stringify(points) : "";
}

function escapeXml(value: string): string {
  return value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("\"", "&quot;")
    .replaceAll("'", "&apos;");
}

function createTemplateId(): string {
  return `shape-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function shapeBoundingCenter(points: number[][]): [number, number] {
  const latitudes = points.map((point) => point[0]);
  const longitudes = points.map((point) => point[1]);
  return [
    (Math.min(...latitudes) + Math.max(...latitudes)) / 2,
    (Math.min(...longitudes) + Math.max(...longitudes)) / 2,
  ];
}

function coordinateDistanceKm(from: number[], to: number[]): number {
  if (from.length < 2 || to.length < 2) {
    return 0;
  }
  const toRadians = (value: number) => (value * Math.PI) / 180;
  const earthRadiusKm = 6371;
  const deltaLat = toRadians(to[0] - from[0]);
  const deltaLng = toRadians(to[1] - from[1]);
  const startLat = toRadians(from[0]);
  const endLat = toRadians(to[0]);
  const haversine = Math.sin(deltaLat / 2) ** 2
    + Math.cos(startLat) * Math.cos(endLat) * Math.sin(deltaLng / 2) ** 2;
  return 2 * earthRadiusKm * Math.atan2(Math.sqrt(haversine), Math.sqrt(1 - haversine));
}

function shapeRadiusKm(points: number[][], center: [number, number]): number {
  return points.reduce(
    (maxRadius, point) => Math.max(maxRadius, coordinateDistanceKm(center, point)),
    0,
  );
}

function clampNumber(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}

function autoFitRadiusBounds(routeType: RouteType): { min: number; max: number; fallback: number } {
  switch (routeType) {
    case "RUN":
    case "TRAIL":
    case "HIKE":
      return { min: 0.45, max: 2.0, fallback: 1.1 };
    default:
      return { min: 0.65, max: 3.2, fallback: 1.6 };
  }
}

function perpendicularDistance(point: number[], lineStart: number[], lineEnd: number[]): number {
  const deltaLat = lineEnd[0] - lineStart[0];
  const deltaLng = lineEnd[1] - lineStart[1];
  if (deltaLat === 0 && deltaLng === 0) {
    return Math.hypot(point[0] - lineStart[0], point[1] - lineStart[1]);
  }
  const numerator = Math.abs(
    (deltaLng * point[0])
    - (deltaLat * point[1])
    + (lineEnd[0] * lineStart[1])
    - (lineEnd[1] * lineStart[0]),
  );
  return numerator / Math.hypot(deltaLat, deltaLng);
}

function simplifyShapePoints(points: number[][], tolerance: number): number[][] {
  if (points.length <= 2) {
    return cloneShapePoints(points);
  }
  let maxDistance = 0;
  let maxIndex = 0;
  const firstPoint = points[0];
  const lastPoint = points[points.length - 1];
  for (let index = 1; index < points.length - 1; index += 1) {
    const distance = perpendicularDistance(points[index], firstPoint, lastPoint);
    if (distance > maxDistance) {
      maxDistance = distance;
      maxIndex = index;
    }
  }
  if (maxDistance <= tolerance) {
    return [firstPoint, lastPoint].map((point) => [point[0], point[1]]);
  }
  const before = simplifyShapePoints(points.slice(0, maxIndex + 1), tolerance);
  const after = simplifyShapePoints(points.slice(maxIndex), tolerance);
  return [...before.slice(0, -1), ...after];
}

function sameShapePoints(left: number[][], right: number[][]): boolean {
  return left.length === right.length
    && left.every((point, index) =>
      point[0] === right[index]?.[0] && point[1] === right[index]?.[1]
    );
}

function placeNormalizedShapePoints(
  points: number[][],
  center: { lat: number; lng: number },
  scale = 0.012,
): number[][] {
  const longitudeScale = Math.max(0.35, Math.cos((center.lat * Math.PI) / 180));
  return points.map((point) => [
    center.lat + (point[1] * scale),
    center.lng + ((point[0] * scale) / longitudeScale),
  ]);
}

function normalizeShapePoints(points: number[][]): number[][] {
  if (points.length === 0) {
    return [];
  }
  const xs = points.map((point) => point[0]);
  const ys = points.map((point) => point[1]);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(...ys);
  const maxY = Math.max(...ys);
  const width = Math.max(1, maxX - minX);
  const height = Math.max(1, maxY - minY);
  const scale = Math.max(width, height) / 2;
  const centerX = (minX + maxX) / 2;
  const centerY = (minY + maxY) / 2;
  return points.map((point) => [
    (point[0] - centerX) / scale,
    (centerY - point[1]) / scale,
  ]);
}

function buildHeartTemplate(): number[][] {
  const points: number[][] = [];
  for (let index = 0; index <= 80; index += 1) {
    const t = (index / 80) * Math.PI * 2;
    const x = (16 * Math.sin(t) ** 3) / 18;
    const y = (13 * Math.cos(t) - 5 * Math.cos(2 * t) - 2 * Math.cos(3 * t) - Math.cos(4 * t)) / 18;
    points.push([x, y]);
  }
  return points;
}

function buildStarTemplate(): number[][] {
  const points: number[][] = [];
  const spikes = 5;
  for (let index = 0; index <= spikes * 2; index += 1) {
    const radius = index % 2 === 0 ? 1 : 0.42;
    const angle = (-Math.PI / 2) + (index * Math.PI / spikes);
    points.push([Math.cos(angle) * radius, Math.sin(angle) * radius]);
  }
  return points;
}

function buildCircleTemplate(): number[][] {
  const points: number[][] = [];
  for (let index = 0; index <= 72; index += 1) {
    const angle = (index / 72) * Math.PI * 2;
    points.push([Math.cos(angle), Math.sin(angle)]);
  }
  return points;
}

function buildSquareTemplate(): number[][] {
  return [
    [-1, -1],
    [1, -1],
    [1, 1],
    [-1, 1],
    [-1, -1],
  ];
}

function buildTriangleTemplate(): number[][] {
  return [
    [0, -1],
    [0.94, 0.62],
    [-0.94, 0.62],
    [0, -1],
  ];
}

function buildDiamondTemplate(): number[][] {
  return [
    [0, -1],
    [1, 0],
    [0, 1],
    [-1, 0],
    [0, -1],
  ];
}

function buildRectangleTemplate(): number[][] {
  return [
    [-1.25, -0.68],
    [1.25, -0.68],
    [1.25, 0.68],
    [-1.25, 0.68],
    [-1.25, -0.68],
  ];
}

function buildHexagonTemplate(): number[][] {
  const points: number[][] = [];
  for (let index = 0; index <= 6; index += 1) {
    const angle = (Math.PI / 6) + ((index / 6) * Math.PI * 2);
    points.push([Math.cos(angle), Math.sin(angle)]);
  }
  return points;
}

function buildTemplatePoints(template: BuiltInShapeTemplateKey): number[][] {
  switch (template) {
    case "heart":
      return buildHeartTemplate();
    case "star":
      return buildStarTemplate();
    case "circle":
      return buildCircleTemplate();
    case "square":
      return buildSquareTemplate();
    case "triangle":
      return buildTriangleTemplate();
    case "diamond":
      return buildDiamondTemplate();
    case "rectangle":
      return buildRectangleTemplate();
    case "hexagon":
      return buildHexagonTemplate();
    default:
      return buildCircleTemplate();
  }
}

function readSavedShapeTemplates(): SavedShapeTemplate[] {
  if (typeof localStorage === "undefined") {
    return [];
  }
  try {
    const raw = localStorage.getItem(SAVED_SHAPES_STORAGE_KEY);
    if (!raw) {
      return [];
    }
    const parsed = JSON.parse(raw) as Array<Partial<SavedShapeTemplate>>;
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .map((template) => {
        const points = sanitizeShapePoints(template.points ?? []);
        if (!template.id || !template.name || points.length < 2) {
          return null;
        }
        const createdAt = template.createdAt ?? new Date().toISOString();
        return {
          id: String(template.id),
          name: String(template.name),
          points,
          createdAt,
          updatedAt: template.updatedAt ?? createdAt,
        };
      })
      .filter((template): template is SavedShapeTemplate => template !== null);
  } catch {
    return [];
  }
}

function writeSavedShapeTemplates(templates: SavedShapeTemplate[]) {
  if (typeof localStorage === "undefined") {
    return;
  }
  localStorage.setItem(SAVED_SHAPES_STORAGE_KEY, JSON.stringify(templates));
}

export const useRoutesStore = defineStore("routes", {
  state: () => ({
    mode: "SHAPE" as RouteMode,
    routeType: "RIDE" as RouteType,
    variantCount: DEFAULT_VARIANT_COUNT,
    startPoint: null as { lat: number; lng: number } | null,
    shapeInputType: "draw" as ShapeInputType,
    shapePoints: [] as number[][],
    shapeDataText: "" as string,
    shapeTransformUndoStack: [] as number[][][],
    shapeTransformRedoStack: [] as number[][][],
    savedShapeTemplates: [] as SavedShapeTemplate[],
    freestyleMode: false,
    isDrawingShape: false,
    routes: [] as GeneratedRoute[],
    generationDiagnostics: [] as RouteGenerationDiagnostic[],
    selectedRouteId: "" as string,
    isLoading: false,
    routingHealthStatus: "unknown" as RoutingHealthStatus,
    routingEngineName: "OSRM" as string,
    routingReachable: null as boolean | null,
    routingExtractProfile: "unknown" as string,
    routingEffectiveProfile: "unknown" as string,
    routingSupportedRouteTypes: [...ALL_ROUTE_TYPES] as RouteType[],
  }),
  getters: {
    selectedRoute(state): GeneratedRoute | null {
      return state.routes.find((route) => route.routeId === state.selectedRouteId) ?? null;
    },
    hasRoutes(state): boolean {
      return state.routes.length > 0;
    },
    hasShape(state): boolean {
      return state.shapePoints.length >= 2 || state.shapeDataText.trim().length > 0;
    },
    canGenerateShape(state): boolean {
      return state.shapePoints.length >= 2 || state.shapeDataText.trim().length > 0;
    },
    canTransformShape(state): boolean {
      return state.shapePoints.length >= 2;
    },
    canUndoShapeTransform(state): boolean {
      return state.shapeTransformUndoStack.length > 0;
    },
    canRedoShapeTransform(state): boolean {
      return state.shapeTransformRedoStack.length > 0;
    },
    savedShapeTemplateCount(state): number {
      return state.savedShapeTemplates.length;
    },
    isRoutingEngineOnline(state): boolean {
      return state.routingHealthStatus === "up" && state.routingReachable === true;
    },
    isRouteTypeSupported(state): (routeType: RouteType) => boolean {
      return (routeType: RouteType) => state.routingSupportedRouteTypes.includes(routeType);
    },
  },
  actions: {
    setMode(mode: RouteMode) {
      this.mode = mode;
    },
    async refreshRoutingHealth() {
      try {
        const response = await fetch("/api/health/details", {
          method: "GET",
          headers: {
            Accept: "application/json",
          },
        });
        if (!response.ok) {
          this.routingHealthStatus = "down";
          this.routingReachable = false;
          return;
        }
        const payload = await response.json() as RoutingHealthPayload;
        const routing = payload.routing;
        const status = String(routing?.status ?? "unknown").toLowerCase();
        const reachable = typeof routing?.reachable === "boolean" ? routing.reachable : null;
        const engine = String(routing?.engine ?? "osrm").trim();

        if (status === "up" || status === "down" || status === "disabled" || status === "misconfigured") {
          this.routingHealthStatus = status;
        } else {
          this.routingHealthStatus = "unknown";
        }
        this.routingReachable = reachable;
        this.routingEngineName = engine.length > 0 ? engine.toUpperCase() : "OSRM";
        this.routingExtractProfile = String(routing?.extractProfile ?? "unknown").trim() || "unknown";
        this.routingEffectiveProfile = String(routing?.effectiveProfile ?? "unknown").trim() || "unknown";
        this.routingSupportedRouteTypes = this.parseSupportedRouteTypes(
          routing?.supportedRouteTypes,
          [routing?.extractProfile, routing?.effectiveProfile, routing?.profile],
        );
        this.ensureRouteTypeIsSupported();
      } catch {
        this.routingHealthStatus = "down";
        this.routingReachable = false;
        this.routingSupportedRouteTypes = [...ALL_ROUTE_TYPES];
      }
    },
    routeTypesFromProfileSignals(signals: Array<string | undefined>): RouteType[] | null {
      const combined = signals
        .map((value) => String(value ?? "").trim().toLowerCase())
        .join(" ");
      if (combined.includes("bicycle.lua") || combined.includes("cycling")) {
        return [...CYCLING_ROUTE_TYPES];
      }
      if (combined.includes("foot.lua") || combined.includes("walking")) {
        return [...FOOT_ROUTE_TYPES];
      }
      if (combined.includes("car.lua") || combined.includes("driving")) {
        return [...CAR_ROUTE_TYPES];
      }
      return null;
    },
    parseSupportedRouteTypes(raw?: string[], profileSignals: Array<string | undefined> = []): RouteType[] {
      const inferredFromProfile = this.routeTypesFromProfileSignals(profileSignals);
      if (!Array.isArray(raw) || raw.length === 0) {
        return inferredFromProfile ?? [...ALL_ROUTE_TYPES];
      }
      const normalized = raw
        .map((value) => String(value).trim().toUpperCase())
        .filter((value): value is RouteType => ALL_ROUTE_TYPES.includes(value as RouteType));
      if (normalized.length === 0) {
        return inferredFromProfile ?? [...ALL_ROUTE_TYPES];
      }
      // If backend returns the generic "all route types" set, but profile signals indicate
      // a concrete OSRM profile, prefer the profile-restricted route types for the UI.
      if (inferredFromProfile && normalized.length === ALL_ROUTE_TYPES.length) {
        return inferredFromProfile;
      }
      return [...new Set(normalized)];
    },
    ensureRouteTypeIsSupported() {
      if (this.routingSupportedRouteTypes.includes(this.routeType)) {
        return;
      }
      this.routeType = this.routingSupportedRouteTypes[0] ?? "RIDE";
    },
    setStartPoint(lat: number, lng: number) {
      this.startPoint = { lat, lng };
    },
    clearStartPoint() {
      this.startPoint = null;
    },
    setShapeInputType(value: ShapeInputType) {
      this.shapeInputType = value;
    },
    setShapeDataText(value: string) {
      this.shapeDataText = value;
    },
    setFreestyleMode(value: boolean) {
      this.freestyleMode = value;
    },
    loadSavedShapeTemplates() {
      this.savedShapeTemplates = readSavedShapeTemplates();
    },
    persistSavedShapeTemplates() {
      writeSavedShapeTemplates(this.savedShapeTemplates);
    },
    resetShapeTransformHistory() {
      this.shapeTransformUndoStack = [];
      this.shapeTransformRedoStack = [];
    },
    syncShapeDataText() {
      this.shapeDataText = shapeDataTextFor(this.shapePoints);
    },
    replaceShapeFromLibrary(points: number[][], inputType: ShapeInputType = "draw"): boolean {
      const sanitizedPoints = sanitizeShapePoints(points);
      if (sanitizedPoints.length < 2) {
        return false;
      }
      this.shapePoints = sanitizedPoints;
      this.shapeInputType = inputType;
      this.isDrawingShape = false;
      this.syncShapeDataText();
      this.resetShapeTransformHistory();
      return true;
    },
    shapeTemplateCenter(center?: { lat: number; lng: number } | null): { lat: number; lng: number } {
      if (center && Number.isFinite(center.lat) && Number.isFinite(center.lng)) {
        return center;
      }
      if (this.startPoint && Number.isFinite(this.startPoint.lat) && Number.isFinite(this.startPoint.lng)) {
        return this.startPoint;
      }
      return DEFAULT_TEMPLATE_CENTER;
    },
    applyBuiltInShapeTemplate(
      template: BuiltInShapeTemplateKey,
      center?: { lat: number; lng: number } | null,
    ): boolean {
      const templateCenter = this.shapeTemplateCenter(center);
      const points = placeNormalizedShapePoints(
        buildTemplatePoints(template),
        templateCenter,
        0.012,
      );
      return this.replaceShapeFromLibrary(points, "draw");
    },
    saveCurrentShapeTemplate(name: string): SavedShapeTemplate | null {
      const points = sanitizeShapePoints(this.shapePoints);
      if (points.length < 2) {
        return null;
      }
      const now = new Date().toISOString();
      const trimmedName = name.trim().slice(0, 48);
      const template: SavedShapeTemplate = {
        id: createTemplateId(),
        name: trimmedName || `Template ${this.savedShapeTemplates.length + 1}`,
        points: cloneShapePoints(points),
        createdAt: now,
        updatedAt: now,
      };
      this.savedShapeTemplates = [template, ...this.savedShapeTemplates].slice(0, 24);
      this.persistSavedShapeTemplates();
      return template;
    },
    loadSavedShapeTemplate(id: string): boolean {
      const template = this.savedShapeTemplates.find((candidate) => candidate.id === id);
      if (!template) {
        return false;
      }
      return this.replaceShapeFromLibrary(template.points, "draw");
    },
    deleteSavedShapeTemplate(id: string): boolean {
      const nextTemplates = this.savedShapeTemplates.filter((template) => template.id !== id);
      if (nextTemplates.length === this.savedShapeTemplates.length) {
        return false;
      }
      this.savedShapeTemplates = nextTemplates;
      this.persistSavedShapeTemplates();
      return true;
    },
    buildCurrentShapeGpx(name = "Strava Art sketch"): string | null {
      const points = sanitizeShapePoints(this.shapePoints);
      if (points.length < 2) {
        return null;
      }
      const escapedName = escapeXml(name.trim() || "Strava Art sketch");
      const trackPoints = points
        .map((point) => `      <trkpt lat="${point[0].toFixed(7)}" lon="${point[1].toFixed(7)}"></trkpt>`)
        .join("\n");
      return [
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
        "<gpx version=\"1.1\" creator=\"MyStravaStats\" xmlns=\"http://www.topografix.com/GPX/1/1\">",
        "  <trk>",
        `    <name>${escapedName}</name>`,
        "    <trkseg>",
        trackPoints,
        "    </trkseg>",
        "  </trk>",
        "</gpx>",
        "",
      ].join("\n");
    },
    buildCurrentShapeTcx(name = "Strava Art sketch"): string | null {
      const points = sanitizeShapePoints(this.shapePoints);
      if (points.length < 2) {
        return null;
      }
      const escapedName = escapeXml(name.trim() || "Strava Art sketch");
      const startTime = Date.now();
      const trackPoints = points
        .map((point, index) => [
          "          <Trackpoint>",
          `            <Time>${new Date(startTime + (index * 1000)).toISOString()}</Time>`,
          "            <Position>",
          `              <LatitudeDegrees>${point[0].toFixed(7)}</LatitudeDegrees>`,
          `              <LongitudeDegrees>${point[1].toFixed(7)}</LongitudeDegrees>`,
          "            </Position>",
          "          </Trackpoint>",
        ].join("\n"))
        .join("\n");
      return [
        "<?xml version=\"1.0\" encoding=\"UTF-8\"?>",
        "<TrainingCenterDatabase xmlns=\"http://www.garmin.com/xmlschemas/TrainingCenterDatabase/v2\" xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xsi:schemaLocation=\"http://www.garmin.com/xmlschemas/TrainingCenterDatabase/v2 http://www.garmin.com/xmlschemas/TrainingCenterDatabasev2.xsd\">",
        "  <Courses>",
        "    <Course>",
        `      <Name>${escapedName}</Name>`,
        "      <Track>",
        trackPoints,
        "      </Track>",
        "    </Course>",
        "  </Courses>",
        "</TrainingCenterDatabase>",
        "",
      ].join("\n");
    },
    exportCurrentShapeGpx(name = "strava-art-sketch") {
      const gpx = this.buildCurrentShapeGpx(name);
      if (!gpx) {
        throw new Error("shape is required");
      }
      if (typeof document === "undefined" || typeof URL === "undefined") {
        throw new Error("browser download is unavailable");
      }
      const safeName = name.trim().toLowerCase()
        .replace(/[^a-z0-9-]+/g, "-")
        .replace(/^-+|-+$/g, "") || "strava-art-sketch";
      const blob = new Blob([gpx], { type: "application/gpx+xml;charset=utf-8" });
      const objectUrl = URL.createObjectURL(blob);
      try {
        const link = document.createElement("a");
        link.href = objectUrl;
        link.download = `${safeName}.gpx`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      } finally {
        URL.revokeObjectURL(objectUrl);
      }
    },
    exportCurrentShapeTcx(name = "strava-art-sketch") {
      const tcx = this.buildCurrentShapeTcx(name);
      if (!tcx) {
        throw new Error("shape is required");
      }
      if (typeof document === "undefined" || typeof URL === "undefined") {
        throw new Error("browser download is unavailable");
      }
      const safeName = name.trim().toLowerCase()
        .replace(/[^a-z0-9-]+/g, "-")
        .replace(/^-+|-+$/g, "") || "strava-art-sketch";
      const blob = new Blob([tcx], { type: "application/vnd.garmin.tcx+xml;charset=utf-8" });
      const objectUrl = URL.createObjectURL(blob);
      try {
        const link = document.createElement("a");
        link.href = objectUrl;
        link.download = `${safeName}.tcx`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      } finally {
        URL.revokeObjectURL(objectUrl);
      }
    },
    clearShape() {
      this.shapePoints = [];
      this.shapeDataText = "";
      this.shapeInputType = "draw";
      this.isDrawingShape = false;
      this.resetShapeTransformHistory();
    },
    toggleShapeDrawing() {
      this.isDrawingShape = !this.isDrawingShape;
      if (this.isDrawingShape) {
        this.shapeInputType = "draw";
      }
    },
    addShapePoint(lat: number, lng: number) {
      this.shapeInputType = "draw";
      this.shapePoints.push([lat, lng]);
      this.syncShapeDataText();
      this.resetShapeTransformHistory();
    },
    undoLastShapePoint() {
      if (this.shapePoints.length === 0) {
        return;
      }
      this.shapePoints = this.shapePoints.slice(0, -1);
      this.shapeInputType = "draw";
      this.syncShapeDataText();
      this.resetShapeTransformHistory();
    },
    mergeShapePoints(basePoints: number[][], importedPoints: number[][]): number[][] {
      if (basePoints.length === 0) {
        return importedPoints.map((point) => [point[0], point[1]]);
      }
      const merged = basePoints.map((point) => [point[0], point[1]]);
      importedPoints.forEach((point) => {
        const candidate = [point[0], point[1]];
        const previous = merged[merged.length - 1];
        if (previous && previous[0] === candidate[0] && previous[1] === candidate[1]) {
          return;
        }
        merged.push(candidate);
      });
      return merged;
    },
    importShapeFromGpx(gpxText: string, options: ImportShapeFromGpxOptions = {}): number {
      const append = options.append === true;
      const pointTagPattern = /<(?:trkpt|rtept|wpt)\b([^>]*)>/gi;
      const latAttrPattern = /\blat\s*=\s*["']([^"']+)["']/i;
      const lonAttrPattern = /\blon\s*=\s*["']([^"']+)["']/i;
      const points: number[][] = [];
      let match: RegExpExecArray | null;
      while ((match = pointTagPattern.exec(gpxText)) !== null) {
        const attributes = match[1] ?? "";
        const latMatch = attributes.match(latAttrPattern);
        const lonMatch = attributes.match(lonAttrPattern);
        if (!latMatch || !lonMatch) {
          continue;
        }
        const lat = Number.parseFloat(latMatch[1]);
        const lng = Number.parseFloat(lonMatch[1]);
        if (!Number.isFinite(lat) || !Number.isFinite(lng)) {
          continue;
        }
        if (lat < -90 || lat > 90 || lng < -180 || lng > 180) {
          continue;
        }
        points.push([lat, lng]);
      }
      if (points.length < 2) {
        if (!append) {
          this.shapePoints = [];
          this.shapeDataText = "";
        }
        this.shapeInputType = "gpx";
        this.isDrawingShape = false;
        this.resetShapeTransformHistory();
        return 0;
      }

      this.shapePoints = append ? this.mergeShapePoints(this.shapePoints, points) : points;
      this.syncShapeDataText();
      this.shapeInputType = "gpx";
      this.isDrawingShape = false;
      this.resetShapeTransformHistory();
      return points.length;
    },
    pushShapeTransformHistory() {
      this.shapeTransformUndoStack.push(cloneShapePoints(this.shapePoints));
      if (this.shapeTransformUndoStack.length > SHAPE_TRANSFORM_HISTORY_LIMIT) {
        this.shapeTransformUndoStack = this.shapeTransformUndoStack.slice(-SHAPE_TRANSFORM_HISTORY_LIMIT);
      }
      this.shapeTransformRedoStack = [];
    },
    replaceShapeWithTransformedPoints(points: number[][]): boolean {
      const sanitizedPoints = sanitizeShapePoints(points);
      if (sanitizedPoints.length < 2 || sameShapePoints(this.shapePoints, sanitizedPoints)) {
        return false;
      }
      this.pushShapeTransformHistory();
      this.shapePoints = sanitizedPoints;
      this.shapeInputType = "draw";
      this.isDrawingShape = false;
      this.syncShapeDataText();
      return true;
    },
    shapeTransformAnchor(): [number, number] {
      if (this.startPoint && Number.isFinite(this.startPoint.lat) && Number.isFinite(this.startPoint.lng)) {
        return [this.startPoint.lat, this.startPoint.lng];
      }
      return shapeBoundingCenter(this.shapePoints);
    },
    translateShape(deltaLat: number, deltaLng: number): boolean {
      if (this.shapePoints.length < 2 || !Number.isFinite(deltaLat) || !Number.isFinite(deltaLng)) {
        return false;
      }
      return this.replaceShapeWithTransformedPoints(
        this.shapePoints.map((point) => [point[0] + deltaLat, point[1] + deltaLng]),
      );
    },
    scaleShape(factor: number): boolean {
      if (this.shapePoints.length < 2 || !Number.isFinite(factor) || factor <= 0) {
        return false;
      }
      const [anchorLat, anchorLng] = this.shapeTransformAnchor();
      return this.replaceShapeWithTransformedPoints(
        this.shapePoints.map((point) => [
          anchorLat + ((point[0] - anchorLat) * factor),
          anchorLng + ((point[1] - anchorLng) * factor),
        ]),
      );
    },
    rotateShape(degrees: number): boolean {
      if (this.shapePoints.length < 2 || !Number.isFinite(degrees)) {
        return false;
      }
      const [anchorLat, anchorLng] = this.shapeTransformAnchor();
      const radians = (degrees * Math.PI) / 180;
      const cos = Math.cos(radians);
      const sin = Math.sin(radians);
      return this.replaceShapeWithTransformedPoints(
        this.shapePoints.map((point) => {
          const lat = point[0] - anchorLat;
          const lng = point[1] - anchorLng;
          return [
            anchorLat + (lat * cos) - (lng * sin),
            anchorLng + (lat * sin) + (lng * cos),
          ];
        }),
      );
    },
    centerShapeOnStart(): boolean {
      if (!this.startPoint || this.shapePoints.length < 2) {
        return false;
      }
      const [centerLat, centerLng] = shapeBoundingCenter(this.shapePoints);
      return this.translateShape(this.startPoint.lat - centerLat, this.startPoint.lng - centerLng);
    },
    fitShapeToStart(options: FitShapeToStartOptions = {}): boolean {
      if (!this.startPoint || this.shapePoints.length < 2) {
        return false;
      }
      const center = shapeBoundingCenter(this.shapePoints);
      const currentRadiusKm = shapeRadiusKm(this.shapePoints, center);
      if (!Number.isFinite(currentRadiusKm) || currentRadiusKm <= 0) {
        return this.centerShapeOnStart();
      }
      const radiusBounds = autoFitRadiusBounds(this.routeType);
      const minRadiusKm = options.minRadiusKm ?? radiusBounds.min;
      const maxRadiusKm = options.maxRadiusKm ?? radiusBounds.max;
      const viewportRadiusKm = options.viewportRadiusKm && Number.isFinite(options.viewportRadiusKm)
        ? options.viewportRadiusKm
        : undefined;
      const targetRadiusKm = clampNumber(
        options.targetRadiusKm
          ?? (viewportRadiusKm ? viewportRadiusKm * 0.42 : radiusBounds.fallback),
        minRadiusKm,
        maxRadiusKm,
      );
      const scale = targetRadiusKm / currentRadiusKm;
      const fittedPoints = this.shapePoints.map((point) => [
        this.startPoint!.lat + ((point[0] - center[0]) * scale),
        this.startPoint!.lng + ((point[1] - center[1]) * scale),
      ]);
      return this.replaceShapeWithTransformedPoints(fittedPoints);
    },
    smoothShape(): boolean {
      if (this.shapePoints.length < 4) {
        return false;
      }
      const smoothed = this.shapePoints.map((point, index, points) => {
        if (index === 0 || index === points.length - 1) {
          return [point[0], point[1]];
        }
        const previous = points[index - 1];
        const next = points[index + 1];
        return [
          (previous[0] + (point[0] * 2) + next[0]) / 4,
          (previous[1] + (point[1] * 2) + next[1]) / 4,
        ];
      });
      return this.replaceShapeWithTransformedPoints(smoothed);
    },
    simplifyShape(tolerance = 0.00025): boolean {
      if (this.shapePoints.length < 3 || !Number.isFinite(tolerance) || tolerance <= 0) {
        return false;
      }
      return this.replaceShapeWithTransformedPoints(simplifyShapePoints(this.shapePoints, tolerance));
    },
    undoShapeTransform(): boolean {
      const previous = this.shapeTransformUndoStack[this.shapeTransformUndoStack.length - 1];
      if (!previous) {
        return false;
      }
      this.shapeTransformUndoStack = this.shapeTransformUndoStack.slice(0, -1);
      this.shapeTransformRedoStack.push(cloneShapePoints(this.shapePoints));
      this.shapePoints = cloneShapePoints(previous);
      this.shapeInputType = "draw";
      this.isDrawingShape = false;
      this.syncShapeDataText();
      return true;
    },
    redoShapeTransform(): boolean {
      const next = this.shapeTransformRedoStack[this.shapeTransformRedoStack.length - 1];
      if (!next) {
        return false;
      }
      this.shapeTransformRedoStack = this.shapeTransformRedoStack.slice(0, -1);
      this.shapeTransformUndoStack.push(cloneShapePoints(this.shapePoints));
      this.shapePoints = cloneShapePoints(next);
      this.shapeInputType = "draw";
      this.isDrawingShape = false;
      this.syncShapeDataText();
      return true;
    },
    setSelectedRoute(routeId: string) {
      this.selectedRouteId = routeId;
    },
    resetRoutes() {
      this.routes = [];
      this.generationDiagnostics = [];
      this.selectedRouteId = "";
    },
    buildGenerationUrl(path: string): string {
      // Route generation is now decoupled from header activity/year filters.
      // This prevents accidental empty-candidate cases (e.g. year with no cached activities).
      return path;
    },
    sanitizePreviewPoints(previewLatLng: number[][]): number[][] {
      return previewLatLng.filter((point) =>
        point.length >= 2 && Number.isFinite(point[0]) && Number.isFinite(point[1])
      );
    },
    sampleGeometryPoints(points: number[][]): number[][] {
      if (points.length <= GEOMETRY_SIGNATURE_MAX_POINTS) {
        return points;
      }
      const step = Math.max(1, Math.floor(points.length / GEOMETRY_SIGNATURE_MAX_POINTS));
      const sampled: number[][] = [];
      for (let index = 0; index < points.length; index += step) {
        sampled.push(points[index]);
      }
      const lastPoint = points[points.length - 1];
      const sampledLastPoint = sampled[sampled.length - 1];
      if (
        !sampledLastPoint
        || sampledLastPoint[0] !== lastPoint[0]
        || sampledLastPoint[1] !== lastPoint[1]
      ) {
        sampled.push(lastPoint);
      }
      return sampled;
    },
    encodeGeometryPoints(points: number[][]): string {
      return points
        .map((point) => `${point[0].toFixed(GEOMETRY_SIGNATURE_PRECISION)},${point[1].toFixed(GEOMETRY_SIGNATURE_PRECISION)}`)
        .join("|");
    },
    routeGeometrySignature(route: GeneratedRoute): string {
      const sanitizedPoints = this.sanitizePreviewPoints(route.previewLatLng);
      if (sanitizedPoints.length === 0) {
        return `route-id:${route.routeId}`;
      }
      const sampledPoints = this.sampleGeometryPoints(sanitizedPoints);
      const forwardGeometry = this.encodeGeometryPoints(sampledPoints);
      const reverseGeometry = this.encodeGeometryPoints([...sampledPoints].reverse());
      const normalizedGeometry = forwardGeometry < reverseGeometry ? forwardGeometry : reverseGeometry;
      return [
        normalizedGeometry,
        `distance:${route.distanceKm.toFixed(2)}`,
        `elevation:${route.elevationGainM.toFixed(1)}`,
      ].join("||");
    },
    dedupeRoutesByGeometry(routes: GeneratedRoute[]): GeneratedRoute[] {
      if (routes.length <= 1) {
        return routes;
      }
      const uniqueRoutes: GeneratedRoute[] = [];
      const seenGeometrySignatures = new Set<string>();
      for (const route of routes) {
        const signature = this.routeGeometrySignature(route);
        if (seenGeometrySignatures.has(signature)) {
          continue;
        }
        seenGeometrySignatures.add(signature);
        uniqueRoutes.push(route);
      }
      return uniqueRoutes;
    },
    async generateRoutes() {
      this.isLoading = true;
      try {
        await this.generateShapeRoutes();
      } finally {
        this.isLoading = false;
      }
    },
    async ensureLoaded() {
      if (this.routes.length > 0) {
        return;
      }
      if (this.shapePoints.length >= 2 || this.shapeDataText.trim().length > 0) {
        await this.generateRoutes();
      }
    },
    async generateShapeRoutes() {
      const hasDrawShape = this.shapePoints.length >= 2;
      const shapeData = hasDrawShape
        ? JSON.stringify(this.shapePoints)
        : this.shapeDataText.trim();
      if (!shapeData) {
        throw new Error("shape is required");
      }

      const payload = {
        shapeInputType: this.shapeInputType,
        shapeData,
        startPoint: this.startPoint,
        routeType: this.routeType,
        variantCount: this.variantCount,
      };
      const data = await requestJson<GenerateRoutesResponse>(
        this.buildGenerationUrl("/api/routes/generate/shape"),
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify(payload),
        },
      );
      this.routes = this.dedupeRoutesByGeometry(data.routes ?? []);
      this.generationDiagnostics = data.diagnostics ?? [];
      this.selectedRouteId = this.routes[0]?.routeId ?? "";
    },
    async exportRouteGpx(routeId: string) {
      const response = await fetch(`/api/routes/${encodeURIComponent(routeId)}/gpx`, {
        method: "GET",
        headers: {
          Accept: "application/gpx+xml",
        },
      });
      if (!response.ok) {
        throw new Error(`Unable to export GPX (HTTP ${response.status})`);
      }

      const blob = await response.blob();
      const contentDisposition = response.headers.get("content-disposition") ?? "";
      const match = contentDisposition.match(/filename="([^"]+)"/i);
      const fileName = match?.[1] ?? `${routeId}.gpx`;
      const objectUrl = URL.createObjectURL(blob);
      try {
        const link = document.createElement("a");
        link.href = objectUrl;
        link.download = fileName;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
      } finally {
        URL.revokeObjectURL(objectUrl);
      }
    },
  },
});
