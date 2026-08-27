import { defineStore } from "pinia";
import { buildFilteredApiUrl, requestJson } from "@/services/http-client";
import { useContextStore } from "@/stores/context";
import { EddingtonNumber } from "@/models/eddington-number.model";
import { DashboardData } from "@/models/dashboard-data.model";
import type { ActivityHeatmap } from "@/models/activity-heatmap.model";
import { ALL_ACTIVITY_TYPE_FILTER } from "@/utils/activityTypes";
import { apiUrl } from "@/services/api-url";

export type HeatmapScope = "selection" | "all-sports";
export type EddingtonScope = "lifetime" | "year" | "rolling-12-months";
export type EddingtonMetric = "distance" | "elevation";
export type EddingtonBasis = "days" | "activities";

type DashboardCacheEntry = {
  cumulativeDistancePerYear: Map<string, Map<string, number>>;
  cumulativeElevationPerYear: Map<string, Map<string, number>>;
  eddingtonNumber: EddingtonNumber;
  dashboardData: DashboardData;
};

type CumulativeApiPayload = {
  distance: Record<string, Record<string, number>>;
  elevation: Record<string, Record<string, number>>;
};

function convertToNestedMap(source: Record<string, Record<string, number>>): Map<string, Map<string, number>> {
  const result = new Map<string, Map<string, number>>();
  for (const year of Object.keys(source)) {
    const daysData = new Map<string, number>();
    for (const dayKey of Object.keys(source[year] ?? {})) {
      daysData.set(dayKey, source[year][dayKey]);
    }
    result.set(year, daysData);
  }
  return result;
}

export const useDashboardStore = defineStore("dashboard", {
  state: () => ({
    cumulativeDistancePerYear: new Map<string, Map<string, number>>(),
    cumulativeElevationPerYear: new Map<string, Map<string, number>>(),
    eddingtonNumber: new EddingtonNumber(),
    dashboardData: new DashboardData({}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, []),
    activityHeatmap: {} as ActivityHeatmap,
    dashboardByKey: {} as Record<string, DashboardCacheEntry>,
    heatmapByKey: {} as Record<string, ActivityHeatmap>,
    heatmapScope: "selection" as HeatmapScope,
    eddingtonScope: "lifetime" as EddingtonScope,
    eddingtonMetric: "distance" as EddingtonMetric,
    eddingtonBasis: "days" as EddingtonBasis,
    isLoading: false,
    error: null as string | null,
  }),
  actions: {
    currentDashboardKey(): string {
      const contextStore = useContextStore();
      return `${contextStore.currentFiltersKey}:eddington=${this.eddingtonScope}:${this.eddingtonMetric}:${this.eddingtonBasis}`;
    },
    isCurrentDashboardKey(key: string): boolean {
      return this.currentDashboardKey() === key;
    },
    currentHeatmapActivityType(): string {
      const contextStore = useContextStore();
      if (this.heatmapScope === "all-sports") {
        return ALL_ACTIVITY_TYPE_FILTER;
      }
      return contextStore.currentActivityType;
    },
    currentHeatmapKey(): string {
      return `${this.heatmapScope}:${this.currentHeatmapActivityType()}`;
    },
    currentYearNumber(): number | null {
      const contextStore = useContextStore();
      const parsed = Number.parseInt(contextStore.currentYear, 10);
      return Number.isFinite(parsed) ? parsed : null;
    },
    setHeatmapScope(scope: HeatmapScope) {
      this.heatmapScope = scope;
    },
    normalizeEddingtonScopeForCurrentContext() {
      if (this.eddingtonScope === "year" && this.currentYearNumber() === null) {
        this.eddingtonScope = "lifetime";
      }
    },
    async setEddingtonScope(scope: EddingtonScope) {
      const nextScope = scope === "year" && this.currentYearNumber() === null
        ? "lifetime"
        : scope;
      if (this.eddingtonScope === nextScope) {
        return;
      }
      this.eddingtonScope = nextScope;
      await this.fetchEddingtonNumber();
    },
    async setEddingtonMetric(metric: EddingtonMetric) {
      if (this.eddingtonMetric === metric) {
        return;
      }
      this.eddingtonMetric = metric;
      await this.fetchEddingtonNumber();
    },
    async setEddingtonBasis(basis: EddingtonBasis) {
      if (this.eddingtonBasis === basis) {
        return;
      }
      this.eddingtonBasis = basis;
      await this.fetchEddingtonNumber();
    },
    updateDashboardCacheForCurrentKey() {
      this.dashboardByKey[this.currentDashboardKey()] = {
        cumulativeDistancePerYear: this.cumulativeDistancePerYear,
        cumulativeElevationPerYear: this.cumulativeElevationPerYear,
        eddingtonNumber: this.eddingtonNumber,
        dashboardData: this.dashboardData,
      };
    },
    applyDashboardCacheEntry(entry: DashboardCacheEntry) {
      this.cumulativeDistancePerYear = entry.cumulativeDistancePerYear;
      this.cumulativeElevationPerYear = entry.cumulativeElevationPerYear;
      this.eddingtonNumber = entry.eddingtonNumber;
      this.dashboardData = entry.dashboardData;
    },
    invalidateCache() {
      this.dashboardByKey = {};
      this.heatmapByKey = {};
    },
    async fetchCumulativeDataPerYear() {
      const contextStore = useContextStore();
      const key = this.currentDashboardKey();
      const url = buildFilteredApiUrl(
        "getCumulativeData",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      const data = await requestJson<CumulativeApiPayload>(url);
      if (this.isCurrentDashboardKey(key)) {
        this.cumulativeDistancePerYear = convertToNestedMap(data.distance ?? {});
        this.cumulativeElevationPerYear = convertToNestedMap(data.elevation ?? {});
        this.updateDashboardCacheForCurrentKey();
      }
    },
    async fetchEddingtonNumber() {
      const contextStore = useContextStore();
      this.normalizeEddingtonScopeForCurrentContext();
      const key = this.currentDashboardKey();
      const baseUrl = buildFilteredApiUrl("getEddingtonNumber", contextStore.currentActivityType, contextStore.currentYear);
      const separator = baseUrl.includes("?") ? "&" : "?";
      const params = new URLSearchParams({
        scope: this.eddingtonScope,
        metric: this.eddingtonMetric,
        basis: this.eddingtonBasis,
      });
      const url = `${baseUrl}${separator}${params.toString()}`;
      const eddingtonNumber = await requestJson<EddingtonNumber>(url);
      if (this.isCurrentDashboardKey(key)) {
        this.eddingtonNumber = eddingtonNumber;
        this.updateDashboardCacheForCurrentKey();
      }
    },
    async fetchDashboardData() {
      const contextStore = useContextStore();
      const key = this.currentDashboardKey();
      const url = buildFilteredApiUrl("getDashboard", contextStore.currentActivityType, contextStore.currentYear);
      const dashboardData = await requestJson<DashboardData>(url);
      if (this.isCurrentDashboardKey(key)) {
        this.dashboardData = dashboardData;
        this.updateDashboardCacheForCurrentKey();
      }
    },
    async fetchActivityHeatmap() {
      const key = this.currentHeatmapKey();
      const url = apiUrl("getActivityHeatmap", {
        query: { activityType: this.currentHeatmapActivityType() },
      });
      try {
        const activityHeatmap = await requestJson<ActivityHeatmap>(url);
        this.heatmapByKey[key] = activityHeatmap;
        if (this.currentHeatmapKey() === key) {
          this.activityHeatmap = activityHeatmap;
        }
      } catch (error) {
        console.warn("Activity heatmap data not available:", error);
      }
    },
    async ensureDashboardLoaded(force = false) {
      this.normalizeEddingtonScopeForCurrentContext();
      const key = this.currentDashboardKey();
      const cached = this.dashboardByKey[key];
      if (!force && cached) {
        this.applyDashboardCacheEntry(cached);
        this.error = null;
        return;
      }

      this.isLoading = true;
      this.error = null;
      try {
        await Promise.all([
          this.fetchEddingtonNumber(),
          this.fetchCumulativeDataPerYear(),
          this.fetchDashboardData(),
        ]);
      } catch (error: unknown) {
        if (this.isCurrentDashboardKey(key)) {
          this.error = error instanceof Error ? error.message : "Failed to load dashboard data.";
        }
      } finally {
        if (this.isCurrentDashboardKey(key)) {
          this.isLoading = false;
        }
      }
    },
    async ensureHeatmapLoaded(force = false) {
      const key = this.currentHeatmapKey();
      const cached = this.heatmapByKey[key];
      if (!force && cached) {
        this.activityHeatmap = cached;
        return;
      }
      await this.fetchActivityHeatmap();
    },
    async refreshDashboardDomain() {
      await this.ensureDashboardLoaded(true);
    },
  },
});
