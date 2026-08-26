import { defineStore } from "pinia";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";
import { EddingtonNumber } from "@/models/eddington-number.model";
import { DashboardData } from "@/models/dashboard-data.model";
import type { ActivityHeatmap } from "@/models/activity-heatmap.model";
import {
  emptyAnnualGoals,
  type AnnualGoals,
  type AnnualGoalTargets,
} from "@/models/annual-goals.model";
import { ALL_ACTIVITY_TYPE_FILTER } from "@/utils/activityTypes";

export type HeatmapScope = "selection" | "all-sports";
export type EddingtonScope = "lifetime" | "year" | "rolling-12-months";
export type EddingtonMetric = "distance" | "elevation";
export type EddingtonBasis = "days" | "activities";

type DashboardCacheEntry = {
  cumulativeDistancePerYear: Map<string, Map<string, number>>;
  cumulativeElevationPerYear: Map<string, Map<string, number>>;
  eddingtonNumber: EddingtonNumber;
  dashboardData: DashboardData;
  annualGoals: AnnualGoals;
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
    annualGoals: emptyAnnualGoals(),
    activityHeatmap: {} as ActivityHeatmap,
    dashboardByKey: {} as Record<string, DashboardCacheEntry>,
    heatmapByKey: {} as Record<string, ActivityHeatmap>,
    heatmapScope: "selection" as HeatmapScope,
    eddingtonScope: "lifetime" as EddingtonScope,
    eddingtonMetric: "distance" as EddingtonMetric,
    eddingtonBasis: "days" as EddingtonBasis,
    isLoading: false,
    isSavingAnnualGoals: false,
    error: null as string | null,
    annualGoalsError: null as string | null,
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
    currentAnnualGoalYear(): number | null {
      const contextStore = useContextStore();
      const parsed = Number.parseInt(contextStore.currentYear, 10);
      return Number.isFinite(parsed) ? parsed : null;
    },
    setHeatmapScope(scope: HeatmapScope) {
      this.heatmapScope = scope;
    },
    normalizeEddingtonScopeForCurrentContext() {
      if (this.eddingtonScope === "year" && this.currentAnnualGoalYear() === null) {
        this.eddingtonScope = "lifetime";
      }
    },
    async setEddingtonScope(scope: EddingtonScope) {
      const nextScope = scope === "year" && this.currentAnnualGoalYear() === null
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
        annualGoals: this.annualGoals,
      };
    },
    applyDashboardCacheEntry(entry: DashboardCacheEntry) {
      this.cumulativeDistancePerYear = entry.cumulativeDistancePerYear;
      this.cumulativeElevationPerYear = entry.cumulativeElevationPerYear;
      this.eddingtonNumber = entry.eddingtonNumber;
      this.dashboardData = entry.dashboardData;
      this.annualGoals = entry.annualGoals;
    },
    invalidateCache() {
      this.dashboardByKey = {};
      this.heatmapByKey = {};
    },
    async fetchCumulativeDataPerYear() {
      const contextStore = useContextStore();
      const key = this.currentDashboardKey();
      const url = buildFilteredApiUrl(
        "dashboard/cumulative-data-per-year",
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
      const baseUrl = buildFilteredApiUrl("dashboard/eddington-number", contextStore.currentActivityType, contextStore.currentYear);
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
      const url = buildFilteredApiUrl("dashboard", contextStore.currentActivityType, contextStore.currentYear);
      const dashboardData = await requestJson<DashboardData>(url);
      if (this.isCurrentDashboardKey(key)) {
        this.dashboardData = dashboardData;
        this.updateDashboardCacheForCurrentKey();
      }
    },
    async fetchAnnualGoals() {
      const contextStore = useContextStore();
      const key = this.currentDashboardKey();
      const year = this.currentAnnualGoalYear();
      if (year === null) {
        if (this.isCurrentDashboardKey(key)) {
          this.annualGoals = emptyAnnualGoals();
          this.annualGoalsError = null;
          this.updateDashboardCacheForCurrentKey();
        }
        return;
      }

      const url = buildFilteredApiUrl("dashboard/annual-goals", contextStore.currentActivityType, contextStore.currentYear);
      const annualGoals = await requestJson<AnnualGoals>(url);
      if (this.isCurrentDashboardKey(key)) {
        this.annualGoals = annualGoals;
        this.annualGoalsError = null;
        this.updateDashboardCacheForCurrentKey();
      }
    },
    async saveAnnualGoals(targets: AnnualGoalTargets) {
      const contextStore = useContextStore();
      const key = this.currentDashboardKey();
      const year = this.currentAnnualGoalYear();
      if (year === null) {
        this.annualGoalsError = "Select a specific year before saving annual goals.";
        return this.annualGoals;
      }

      this.isSavingAnnualGoals = true;
      this.annualGoalsError = null;
      try {
        const url = buildFilteredApiUrl("dashboard/annual-goals", contextStore.currentActivityType, contextStore.currentYear);
        const annualGoals = await requestJson<AnnualGoals>(url, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(targets),
        });
        if (this.isCurrentDashboardKey(key)) {
          this.annualGoals = annualGoals;
          this.updateDashboardCacheForCurrentKey();
        }
        return annualGoals;
      } catch (error: unknown) {
        if (this.isCurrentDashboardKey(key)) {
          this.annualGoalsError = error instanceof Error ? error.message : "Failed to save annual goals.";
        }
        throw error;
      } finally {
        this.isSavingAnnualGoals = false;
      }
    },
    async fetchActivityHeatmap() {
      const key = this.currentHeatmapKey();
      const params = new URLSearchParams({
        activityType: this.currentHeatmapActivityType(),
      });
      const url = `/api/dashboard/activity-heatmap?${params.toString()}`;
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
          this.fetchAnnualGoals(),
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
