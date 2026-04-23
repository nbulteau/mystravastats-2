import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import {
  type GenerateRoutesResponse,
  type GeneratedRoute,
  type RouteMode,
  type RouteType,
  type TargetGenerationMode,
  type RouteGenerationDiagnostic,
  type ShapeInputType,
  type StartDirection,
} from "@/models/route-recommendation.model";

const DEFAULT_VARIANT_COUNT = 4;
const TARGET_GENERATION_POOL_SIZE = 4;
const MAX_TARGET_VARIANT_COUNT = 24;
const GEOMETRY_SIGNATURE_PRECISION = 5;
const GEOMETRY_SIGNATURE_MAX_POINTS = 80;

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

export const useRoutesStore = defineStore("routes", {
  state: () => ({
    mode: "TARGET" as RouteMode,
    targetGenerationMode: "AUTOMATIC" as TargetGenerationMode,
    routeType: "RIDE" as RouteType,
    startDirection: "UNDEFINED" as StartDirection,
    distanceTargetKm: 40 as number,
    elevationTargetM: 800 as number,
    variantCount: DEFAULT_VARIANT_COUNT,
    startPoint: null as { lat: number; lng: number } | null,
    shapeInputType: "draw" as ShapeInputType,
    shapePoints: [] as number[][],
    customWaypoints: [] as number[][],
    shapeDataText: "" as string,
    isDrawingShape: false,
    routes: [] as GeneratedRoute[],
    generationDiagnostics: [] as RouteGenerationDiagnostic[],
    selectedRouteId: "" as string,
    isLoading: false,
    targetGenerationIndex: 0,
    lastGeneratedTargetRouteNumber: 0,
    targetRequestSignature: "" as string,
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
    canGenerateTarget(state): boolean {
      if (state.startPoint === null || state.distanceTargetKm <= 0) {
        return false;
      }
      if (state.targetGenerationMode === "CUSTOM") {
        return state.customWaypoints.length > 0;
      }
      return true;
    },
    canGenerateShape(state): boolean {
      return state.shapePoints.length >= 2 || state.shapeDataText.trim().length > 0;
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
    setTargetGenerationMode(mode: TargetGenerationMode) {
      this.targetGenerationMode = mode;
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
    addCustomWaypoint(lat: number, lng: number) {
      this.customWaypoints.push([lat, lng]);
    },
    removeLastCustomWaypoint() {
      this.customWaypoints.pop();
    },
    clearCustomWaypoints() {
      this.customWaypoints = [];
    },
    setShapeInputType(value: ShapeInputType) {
      this.shapeInputType = value;
    },
    setShapeDataText(value: string) {
      this.shapeDataText = value;
    },
    clearShape() {
      this.shapePoints = [];
      this.shapeDataText = "";
      this.shapeInputType = "draw";
      this.isDrawingShape = false;
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
      this.shapeDataText = JSON.stringify(this.shapePoints);
    },
    undoLastShapePoint() {
      if (this.shapePoints.length === 0) {
        return;
      }
      this.shapePoints = this.shapePoints.slice(0, -1);
      this.shapeInputType = "draw";
      this.shapeDataText = this.shapePoints.length >= 2 ? JSON.stringify(this.shapePoints) : "";
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
        return 0;
      }

      this.shapePoints = append ? this.mergeShapePoints(this.shapePoints, points) : points;
      this.shapeDataText = this.shapePoints.length >= 2 ? JSON.stringify(this.shapePoints) : "";
      this.shapeInputType = "gpx";
      this.isDrawingShape = false;
      return points.length;
    },
    setSelectedRoute(routeId: string) {
      this.selectedRouteId = routeId;
    },
    resetRoutes() {
      this.routes = [];
      this.generationDiagnostics = [];
      this.selectedRouteId = "";
      this.targetGenerationIndex = 0;
      this.lastGeneratedTargetRouteNumber = 0;
      this.targetRequestSignature = "";
    },
    buildGenerationUrl(path: string): string {
      // Route generation is now decoupled from header activity/year filters.
      // This prevents accidental empty-candidate cases (e.g. year with no cached activities).
      return path;
    },
    parseOptionalNumber(raw: number): number | null {
      if (!Number.isFinite(raw) || raw <= 0) {
        return null;
      }
      return raw;
    },
    buildTargetRequestSignature(payload: {
      startLat: number;
      startLng: number;
      routeType: RouteType;
      targetGenerationMode: TargetGenerationMode;
      startDirection: StartDirection | undefined;
      distanceTargetKm: number;
      elevationTargetM: number | null;
      customWaypoints: Array<{ lat: number; lng: number }>;
    }): string {
      return JSON.stringify({
        startLat: Number(payload.startLat.toFixed(6)),
        startLng: Number(payload.startLng.toFixed(6)),
        routeType: payload.routeType,
        targetGenerationMode: payload.targetGenerationMode,
        startDirection: payload.startDirection ?? "UNDEFINED",
        distanceTargetKm: Number(payload.distanceTargetKm.toFixed(3)),
        elevationTargetM: payload.elevationTargetM === null ? null : Number(payload.elevationTargetM.toFixed(1)),
        customWaypoints: payload.customWaypoints.map((point) => ({
          lat: Number(point.lat.toFixed(6)),
          lng: Number(point.lng.toFixed(6)),
        })),
      });
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
    computeTargetVariantCount(existingCount: number): number {
      return Math.max(
        TARGET_GENERATION_POOL_SIZE,
        Math.min(MAX_TARGET_VARIANT_COUNT, existingCount + TARGET_GENERATION_POOL_SIZE),
      );
    },
    async generateRoutes() {
      this.isLoading = true;
      try {
        if (this.mode === "TARGET") {
          await this.generateTargetRoutes();
        } else {
          await this.generateShapeRoutes();
        }
      } finally {
        this.isLoading = false;
      }
    },
    async ensureLoaded() {
      if (this.routes.length > 0) {
        return;
      }
      if (this.mode === "TARGET") {
        if (this.startPoint && this.parseOptionalNumber(this.distanceTargetKm) !== null) {
          await this.generateRoutes();
        }
        return;
      }
      if (this.shapePoints.length >= 2 || this.shapeDataText.trim().length > 0) {
        await this.generateRoutes();
      }
    },
    async generateTargetRoutes() {
      const distanceTarget = this.parseOptionalNumber(this.distanceTargetKm);
      if (this.startPoint === null || distanceTarget === null) {
        throw new Error("start point and distance target are required");
      }
      const elevationTarget = this.parseOptionalNumber(this.elevationTargetM);
      const customWaypoints = this.targetGenerationMode === "CUSTOM"
        ? this.customWaypoints.map((point) => ({ lat: point[0], lng: point[1] }))
        : [];
      const targetRequestSignature = this.buildTargetRequestSignature({
        startLat: this.startPoint.lat,
        startLng: this.startPoint.lng,
        routeType: this.routeType,
        targetGenerationMode: this.targetGenerationMode,
        startDirection: this.targetGenerationMode === "AUTOMATIC" ? this.startDirection : undefined,
        distanceTargetKm: distanceTarget,
        elevationTargetM: elevationTarget,
        customWaypoints,
      });

      if (this.targetRequestSignature !== targetRequestSignature) {
        this.routes = [];
        this.selectedRouteId = "";
        this.lastGeneratedTargetRouteNumber = 0;
        this.targetGenerationIndex = 0;
        this.targetRequestSignature = targetRequestSignature;
      }

      const knownRouteIds = new Set(this.routes.map((route) => route.routeId));
      const knownGeometrySignatures = new Set(this.routes.map((route) => this.routeGeometrySignature(route)));
      const variantCount = this.computeTargetVariantCount(this.routes.length);
      const payload = {
        startPoint: this.startPoint,
        routeType: this.routeType,
        generationMode: this.targetGenerationMode,
        startDirection: this.targetGenerationMode === "AUTOMATIC" ? this.startDirection : undefined,
        distanceTargetKm: distanceTarget,
        elevationTargetM: elevationTarget,
        customWaypoints: this.targetGenerationMode === "CUSTOM" ? customWaypoints : undefined,
        variantCount,
      };
      const data = await requestJson<GenerateRoutesResponse>(
        this.buildGenerationUrl("/api/routes/generate/target"),
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          body: JSON.stringify(payload),
        },
      );
      this.generationDiagnostics = data.diagnostics ?? [];
      const generatedRoutes = this.dedupeRoutesByGeometry(data.routes ?? []);
      if (generatedRoutes.length === 0) {
        if (this.routes.length === 0) {
          this.selectedRouteId = "";
          this.lastGeneratedTargetRouteNumber = 0;
        }
        return;
      }

      let newUniqueRoute: GeneratedRoute | null = null;
      for (const route of generatedRoutes) {
        const geometrySignature = this.routeGeometrySignature(route);
        if (knownRouteIds.has(route.routeId) || knownGeometrySignatures.has(geometrySignature)) {
          continue;
        }
        newUniqueRoute = route;
        break;
      }
      if (!newUniqueRoute) {
        this.generationDiagnostics = this.generationDiagnostics.length > 0
          ? this.generationDiagnostics
          : [{
            code: "NO_UNIQUE_ROUTE",
            message: "No additional unique route found after geometry deduplication.",
          }];
        return;
      }

      this.routes = [...this.routes, newUniqueRoute];
      this.selectedRouteId = newUniqueRoute.routeId;
      this.lastGeneratedTargetRouteNumber = this.routes.length;
      this.targetGenerationIndex += 1;
    },
    async generateShapeRoutes() {
      const distanceTarget = this.parseOptionalNumber(this.distanceTargetKm);
      const elevationTarget = this.parseOptionalNumber(this.elevationTargetM);
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
        distanceTargetKm: distanceTarget,
        elevationTargetM: elevationTarget,
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
      this.lastGeneratedTargetRouteNumber = 0;
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
