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
  type BacktrackingProfile,
} from "@/models/route-recommendation.model";

const DEFAULT_VARIANT_COUNT = 4;
const TARGET_GENERATION_POOL_SIZE = 2;

type RoutingHealthStatus = "unknown" | "up" | "down" | "disabled" | "misconfigured";

interface RoutingHealthPayload {
  routing?: {
    status?: string;
    reachable?: boolean;
    engine?: string;
    enabled?: boolean;
  };
}

export const useRoutesStore = defineStore("routes", {
  state: () => ({
    mode: "TARGET" as RouteMode,
    targetGenerationMode: "AUTOMATIC" as TargetGenerationMode,
    routeType: "RIDE" as RouteType,
    startDirection: "UNDEFINED" as StartDirection,
    backtrackingProfile: "STRICT" as BacktrackingProfile,
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
    routingHealthStatus: "unknown" as RoutingHealthStatus,
    routingEngineName: "OSRM" as string,
    routingReachable: null as boolean | null,
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
      } catch {
        this.routingHealthStatus = "down";
        this.routingReachable = false;
      }
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
      this.isDrawingShape = false;
    },
    toggleShapeDrawing() {
      this.isDrawingShape = !this.isDrawingShape;
    },
    addShapePoint(lat: number, lng: number) {
      this.shapePoints.push([lat, lng]);
      this.shapeDataText = JSON.stringify(this.shapePoints);
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
      const payload = {
        startPoint: this.startPoint,
        routeType: this.routeType,
        generationMode: this.targetGenerationMode,
        startDirection: this.targetGenerationMode === "AUTOMATIC" ? this.startDirection : undefined,
        backtrackingProfile: this.backtrackingProfile,
        strictBacktracking: this.backtrackingProfile !== "BALANCED",
        distanceTargetKm: distanceTarget,
        elevationTargetM: elevationTarget,
        customWaypoints: this.targetGenerationMode === "CUSTOM"
          ? this.customWaypoints.map((point) => ({ lat: point[0], lng: point[1] }))
          : undefined,
        variantCount: TARGET_GENERATION_POOL_SIZE,
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
      this.routes = data.routes ?? [];
      this.generationDiagnostics = data.diagnostics ?? [];
      if (this.routes.length === 0) {
        this.selectedRouteId = "";
        this.lastGeneratedTargetRouteNumber = 0;
        return;
      }
      const index = this.targetGenerationIndex % this.routes.length;
      this.lastGeneratedTargetRouteNumber = index + 1;
      this.selectedRouteId = this.routes[index]?.routeId ?? this.routes[0]?.routeId ?? "";
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
      this.routes = data.routes ?? [];
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
