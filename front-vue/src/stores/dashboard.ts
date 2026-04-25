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

export type HeatmapScope = "selection" | "all-sports";

type DashboardCacheEntry = {
  cumulativeDistancePerYear: Map<string, Map<string, number>>;
  cumulativeElevationPerYear: Map<string, Map<string, number>>;
  eddingtonNumber: EddingtonNumber;
  dashboardData: DashboardData;
  annualGoals: AnnualGoals;
};

const ALL_SPORTS_ACTIVITY_TYPE = [
  "AlpineSki",
  "Commute",
  "GravelRide",
  "Hike",
  "MountainBikeRide",
  "Ride",
  "Run",
  "TrailRun",
  "VirtualRide",
].join("_");

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
    dashboardData: new DashboardData({}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, []),
    annualGoals: emptyAnnualGoals(),
    activityHeatmap: {} as ActivityHeatmap,
    dashboardByKey: {} as Record<string, DashboardCacheEntry>,
    heatmapByKey: {} as Record<string, ActivityHeatmap>,
    heatmapScope: "selection" as HeatmapScope,
    isLoading: false,
    isSavingAnnualGoals: false,
    error: null as string | null,
    annualGoalsError: null as string | null,
  }),
  actions: {
    currentDashboardKey(): string {
      const contextStore = useContextStore();
      return contextStore.currentFiltersKey;
    },
    currentHeatmapActivityType(): string {
      const contextStore = useContextStore();
      if (this.heatmapScope === "all-sports") {
        return ALL_SPORTS_ACTIVITY_TYPE;
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
    async fetchCumulativeDataPerYear() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl(
        "dashboard/cumulative-data-per-year",
        contextStore.currentActivityType,
        contextStore.currentYear,
      );
      const data = await requestJson<CumulativeApiPayload>(url);
      this.cumulativeDistancePerYear = convertToNestedMap(data.distance ?? {});
      this.cumulativeElevationPerYear = convertToNestedMap(data.elevation ?? {});
      this.updateDashboardCacheForCurrentKey();
    },
    async fetchEddingtonNumber() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("dashboard/eddington-number", contextStore.currentActivityType, contextStore.currentYear);
      this.eddingtonNumber = await requestJson<EddingtonNumber>(url);
      this.updateDashboardCacheForCurrentKey();
    },
    async fetchDashboardData() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("dashboard", contextStore.currentActivityType, contextStore.currentYear);
      this.dashboardData = await requestJson<DashboardData>(url);
      this.updateDashboardCacheForCurrentKey();
    },
    async fetchAnnualGoals() {
      const contextStore = useContextStore();
      const year = this.currentAnnualGoalYear();
      if (year === null) {
        this.annualGoals = emptyAnnualGoals();
        this.annualGoalsError = null;
        this.updateDashboardCacheForCurrentKey();
        return;
      }

      const url = buildFilteredApiUrl("dashboard/annual-goals", contextStore.currentActivityType, contextStore.currentYear);
      this.annualGoals = await requestJson<AnnualGoals>(url);
      this.annualGoalsError = null;
      this.updateDashboardCacheForCurrentKey();
    },
    async saveAnnualGoals(targets: AnnualGoalTargets) {
      const contextStore = useContextStore();
      const year = this.currentAnnualGoalYear();
      if (year === null) {
        this.annualGoalsError = "Select a specific year before saving annual goals.";
        return this.annualGoals;
      }

      this.isSavingAnnualGoals = true;
      this.annualGoalsError = null;
      try {
        const url = buildFilteredApiUrl("dashboard/annual-goals", contextStore.currentActivityType, contextStore.currentYear);
        this.annualGoals = await requestJson<AnnualGoals>(url, {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(targets),
        });
        this.updateDashboardCacheForCurrentKey();
        return this.annualGoals;
      } catch (error: unknown) {
        this.annualGoalsError = error instanceof Error ? error.message : "Failed to save annual goals.";
        throw error;
      } finally {
        this.isSavingAnnualGoals = false;
      }
    },
    async fetchActivityHeatmap() {
      const params = new URLSearchParams({
        activityType: this.currentHeatmapActivityType(),
      });
      const url = `/api/dashboard/activity-heatmap?${params.toString()}`;
      try {
        this.activityHeatmap = await requestJson<ActivityHeatmap>(url);
        this.heatmapByKey[this.currentHeatmapKey()] = this.activityHeatmap;
      } catch (error) {
        console.warn("Activity heatmap data not available:", error);
      }
    },
    async ensureDashboardLoaded(force = false) {
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
        this.error = error instanceof Error ? error.message : "Failed to load dashboard data.";
      } finally {
        this.isLoading = false;
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
