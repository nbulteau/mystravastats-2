import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import {
  type GenerateRoutesResponse,
  type GeneratedRoute,
  type RouteMode,
  type RouteType,
  type ShapeInputType,
  type StartDirection,
} from "@/models/route-recommendation.model";
import { useContextStore } from "@/stores/context";

const DEFAULT_VARIANT_COUNT = 4;
const TARGET_GENERATION_POOL_SIZE = 6;

export const useRoutesStore = defineStore("routes", {
  state: () => ({
    mode: "TARGET" as RouteMode,
    routeType: "RIDE" as RouteType,
    startDirection: "N" as StartDirection,
    distanceTargetKm: "40" as string,
    elevationTargetM: "800" as string,
    variantCount: DEFAULT_VARIANT_COUNT,
    startPoint: null as { lat: number; lng: number } | null,
    shapeInputType: "draw" as ShapeInputType,
    shapePoints: [] as number[][],
    shapeDataText: "" as string,
    isDrawingShape: false,
    routes: [] as GeneratedRoute[],
    selectedRouteId: "" as string,
    isLoading: false,
    targetGenerationIndex: 0,
    lastGeneratedTargetRouteNumber: 0,
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
      return state.startPoint !== null && Number(state.distanceTargetKm) > 0;
    },
    canGenerateShape(state): boolean {
      return state.shapePoints.length >= 2 || state.shapeDataText.trim().length > 0;
    },
  },
  actions: {
    setMode(mode: RouteMode) {
      this.mode = mode;
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
      this.selectedRouteId = "";
      this.targetGenerationIndex = 0;
      this.lastGeneratedTargetRouteNumber = 0;
    },
    buildFiltersQuery(): string {
      const contextStore = useContextStore();
      const params = new URLSearchParams({
        activityType: contextStore.currentActivityType,
      });
      if (contextStore.currentYear !== "All years") {
        params.set("year", contextStore.currentYear);
      }
      return params.toString();
    },
    parseOptionalNumber(raw: string): number | null {
      const value = Number(raw);
      if (!Number.isFinite(value) || value <= 0) {
        return null;
      }
      return value;
    },
    async generateRoutes() {
      this.isLoading = true;
      try {
        const query = this.buildFiltersQuery();
        if (this.mode === "TARGET") {
          await this.generateTargetRoutes(query);
        } else {
          await this.generateShapeRoutes(query);
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
    async generateTargetRoutes(query: string) {
      const distanceTarget = this.parseOptionalNumber(this.distanceTargetKm);
      if (this.startPoint === null || distanceTarget === null) {
        throw new Error("start point and distance target are required");
      }
      const elevationTarget = this.parseOptionalNumber(this.elevationTargetM);
      const payload = {
        startPoint: this.startPoint,
        routeType: this.routeType,
        startDirection: this.startDirection,
        distanceTargetKm: distanceTarget,
        elevationTargetM: elevationTarget,
        variantCount: TARGET_GENERATION_POOL_SIZE,
      };
      const data = await requestJson<GenerateRoutesResponse>(
        `/api/routes/generate/target?${query}`,
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
    async generateShapeRoutes(query: string) {
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
        `/api/routes/generate/shape?${query}`,
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
