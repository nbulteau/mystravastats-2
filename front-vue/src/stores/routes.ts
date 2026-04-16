import { defineStore } from "pinia";
import type {
  RouteExplorerResult,
} from "@/models/route-recommendation.model";
import { requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

const ROUTES_CACHE_TTL_MS = 5 * 60 * 1000;

type RoutesCacheEntry = {
  data: RouteExplorerResult;
  expiresAt: number;
};

type RouteShape =
  | ""
  | "LOOP"
  | "OUT_AND_BACK"
  | "POINT_TO_POINT"
  | "FIGURE_EIGHT";

type RouteSeason =
  | ""
  | "WINTER"
  | "SPRING"
  | "SUMMER"
  | "AUTUMN";

type RouteType =
  | ""
  | "RIDE"
  | "MTB"
  | "GRAVEL"
  | "RUN"
  | "TRAIL"
  | "HIKE";

type StartDirection =
  | ""
  | "N"
  | "S"
  | "E"
  | "W";

export const useRoutesStore = defineStore("routes", {
  state: () => ({
    result: {
      closestLoops: [],
      variants: [],
      seasonal: [],
      shapeMatches: [],
      shapeRemixes: [],
    } as RouteExplorerResult,
    distanceTargetKm: "" as string,
    elevationTargetM: "" as string,
    durationTargetMin: "" as string,
    startDirection: "" as StartDirection,
    routeType: "" as RouteType,
    season: "" as RouteSeason,
    shape: "" as RouteShape,
    includeRemix: false,
    limit: 6,
    cacheByKey: {} as Record<string, RoutesCacheEntry>,
  }),
  getters: {
    hasAnyResults(state): boolean {
      return (
        state.result.closestLoops.length > 0
        || state.result.variants.length > 0
        || state.result.seasonal.length > 0
        || state.result.shapeMatches.length > 0
        || state.result.shapeRemixes.length > 0
      );
    },
  },
  actions: {
    cacheKey(): string {
      const contextStore = useContextStore();
      return [
        contextStore.currentYear,
        contextStore.currentActivityType,
        this.distanceTargetKm || "auto-distance",
        this.elevationTargetM || "auto-elevation",
        this.durationTargetMin || "auto-duration",
        this.startDirection || "all-directions",
        this.routeType || "all-route-types",
        this.season || "all-seasons",
        this.shape || "all-shapes",
        this.includeRemix ? "with-remix" : "no-remix",
        this.limit,
      ].join("__");
    },
    buildRoutesUrl(): string {
      const contextStore = useContextStore();
      const params = new URLSearchParams({
        activityType: contextStore.currentActivityType,
      });
      if (contextStore.currentYear !== "All years") {
        params.set("year", contextStore.currentYear);
      }

      if (this.distanceTargetKm.trim().length > 0) {
        params.set("distanceTargetKm", this.distanceTargetKm.trim());
      }
      if (this.elevationTargetM.trim().length > 0) {
        params.set("elevationTargetM", this.elevationTargetM.trim());
      }
      if (this.durationTargetMin.trim().length > 0) {
        params.set("durationTargetMin", this.durationTargetMin.trim());
      }
      if (this.startDirection) {
        params.set("startDirection", this.startDirection);
      }
      if (this.routeType) {
        params.set("routeType", this.routeType);
      }
      if (this.season) {
        params.set("season", this.season);
      }
      if (this.shape) {
        params.set("shape", this.shape);
      }
      if (this.includeRemix) {
        params.set("includeRemix", "true");
      }
      params.set("limit", String(this.limit));

      return `/api/routes/recommendations?${params.toString()}`;
    },
    updateFilters(payload: {
      distanceTargetKm?: string;
      elevationTargetM?: string;
      durationTargetMin?: string;
      startDirection?: StartDirection;
      routeType?: RouteType;
      season?: RouteSeason;
      shape?: RouteShape;
      includeRemix?: boolean;
      limit?: number;
    }) {
      if (payload.distanceTargetKm !== undefined) {
        this.distanceTargetKm = payload.distanceTargetKm;
      }
      if (payload.elevationTargetM !== undefined) {
        this.elevationTargetM = payload.elevationTargetM;
      }
      if (payload.durationTargetMin !== undefined) {
        this.durationTargetMin = payload.durationTargetMin;
      }
      if (payload.startDirection !== undefined) {
        this.startDirection = payload.startDirection;
      }
      if (payload.routeType !== undefined) {
        this.routeType = payload.routeType;
      }
      if (payload.season !== undefined) {
        this.season = payload.season;
      }
      if (payload.shape !== undefined) {
        this.shape = payload.shape;
      }
      if (payload.includeRemix !== undefined) {
        this.includeRemix = payload.includeRemix;
      }
      if (payload.limit !== undefined) {
        this.limit = Math.max(1, Math.min(24, payload.limit));
      }
    },
    async fetchRecommendations() {
      const data = await requestJson<RouteExplorerResult>(this.buildRoutesUrl());
      this.result = {
        closestLoops: data.closestLoops ?? [],
        variants: data.variants ?? [],
        seasonal: data.seasonal ?? [],
        shapeMatches: data.shapeMatches ?? [],
        shapeRemixes: data.shapeRemixes ?? [],
      };
      this.cacheByKey[this.cacheKey()] = {
        data: this.result,
        expiresAt: Date.now() + ROUTES_CACHE_TTL_MS,
      };
    },
    async ensureLoaded(force = false) {
      const key = this.cacheKey();
      const cached = this.cacheByKey[key];
      if (!force && cached && cached.expiresAt > Date.now()) {
        this.result = cached.data;
        return;
      }
      await this.fetchRecommendations();
    },
    invalidateCache() {
      this.cacheByKey = {};
    },
  },
});
