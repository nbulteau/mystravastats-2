import { defineStore } from "pinia";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";
import { EddingtonNumber } from "@/models/eddington-number.model";
import { DashboardData } from "@/models/dashboard-data.model";
import type { ActivityHeatmap } from "@/models/activity-heatmap.model";

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
    dashboardData: new DashboardData({}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, []),
    activityHeatmap: {} as ActivityHeatmap,
    dashboardByKey: {} as Record<string, DashboardCacheEntry>,
    heatmapByActivityType: {} as Record<string, ActivityHeatmap>,
  }),
  actions: {
    currentDashboardKey(): string {
      const contextStore = useContextStore();
      return contextStore.currentFiltersKey;
    },
    currentHeatmapKey(): string {
      const contextStore = useContextStore();
      return contextStore.currentActivityType;
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
    async fetchActivityHeatmap() {
      const contextStore = useContextStore();
      const params = new URLSearchParams({
        activityType: contextStore.currentActivityType,
      });
      const url = `/api/dashboard/activity-heatmap?${params.toString()}`;
      try {
        this.activityHeatmap = await requestJson<ActivityHeatmap>(url);
        this.heatmapByActivityType[this.currentHeatmapKey()] = this.activityHeatmap;
      } catch (error) {
        console.warn("Activity heatmap data not available:", error);
      }
    },
    async ensureDashboardLoaded(force = false) {
      const key = this.currentDashboardKey();
      const cached = this.dashboardByKey[key];
      if (!force && cached) {
        this.applyDashboardCacheEntry(cached);
        return;
      }

      await Promise.all([
        this.fetchEddingtonNumber(),
        this.fetchCumulativeDataPerYear(),
        this.fetchDashboardData(),
      ]);
    },
    async ensureHeatmapLoaded(force = false) {
      const key = this.currentHeatmapKey();
      const cached = this.heatmapByActivityType[key];
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
