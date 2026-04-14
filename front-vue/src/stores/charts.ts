import { defineStore } from "pinia";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

type ChartsCacheEntry = {
  distanceByMonths: Map<string, number>[];
  elevationByMonths: Map<string, number>[];
  averageSpeedByMonths: Map<string, number>[];
  distanceByWeeks: Map<string, number>[];
  elevationByWeeks: Map<string, number>[];
  cadenceByWeeks: Map<string, number>[];
};

export const useChartsStore = defineStore("charts", {
  state: () => ({
    distanceByMonths: [] as Map<string, number>[],
    elevationByMonths: [] as Map<string, number>[],
    averageSpeedByMonths: [] as Map<string, number>[],
    distanceByWeeks: [] as Map<string, number>[],
    elevationByWeeks: [] as Map<string, number>[],
    cadenceByWeeks: [] as Map<string, number>[],
    chartsByKey: {} as Record<string, ChartsCacheEntry>,
  }),
  actions: {
    currentFiltersKey(): string {
      const contextStore = useContextStore();
      return `${contextStore.currentActivityType}__${contextStore.currentYear}`;
    },
    updateCacheForCurrentKey() {
      this.chartsByKey[this.currentFiltersKey()] = {
        distanceByMonths: this.distanceByMonths,
        elevationByMonths: this.elevationByMonths,
        averageSpeedByMonths: this.averageSpeedByMonths,
        distanceByWeeks: this.distanceByWeeks,
        elevationByWeeks: this.elevationByWeeks,
        cadenceByWeeks: this.cadenceByWeeks,
      };
    },
    applyCacheEntry(entry: ChartsCacheEntry) {
      this.distanceByMonths = entry.distanceByMonths;
      this.elevationByMonths = entry.elevationByMonths;
      this.averageSpeedByMonths = entry.averageSpeedByMonths;
      this.distanceByWeeks = entry.distanceByWeeks;
      this.elevationByWeeks = entry.elevationByWeeks;
      this.cadenceByWeeks = entry.cadenceByWeeks;
    },
    async fetchDistanceByMonths() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/distance-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=MONTHS";
      this.distanceByMonths = await requestJson<Map<string, number>[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchElevationByMonths() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/elevation-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=MONTHS";
      this.elevationByMonths = await requestJson<Map<string, number>[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchAverageSpeedByMonths() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl(
          "charts/average-speed-by-period",
          contextStore.currentActivityType,
          contextStore.currentYear,
        ) + "&period=MONTHS";
      this.averageSpeedByMonths = await requestJson<Map<string, number>[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchDistanceByWeeks() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/distance-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=WEEKS";
      this.distanceByWeeks = await requestJson<Map<string, number>[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchElevationByWeeks() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/elevation-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=WEEKS";
      this.elevationByWeeks = await requestJson<Map<string, number>[]>(url);
      this.updateCacheForCurrentKey();
    },
    async fetchCadenceByWeeks() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl(
          "charts/average-cadence-by-period",
          contextStore.currentActivityType,
          contextStore.currentYear,
        ) + "&period=WEEKS";
      this.cadenceByWeeks = await requestJson<Map<string, number>[]>(url);
      this.updateCacheForCurrentKey();
    },
    resetCharts() {
      this.distanceByMonths = [];
      this.elevationByMonths = [];
      this.averageSpeedByMonths = [];
      this.distanceByWeeks = [];
      this.elevationByWeeks = [];
      this.cadenceByWeeks = [];
    },
    async ensureLoaded(force = false) {
      const contextStore = useContextStore();
      if (contextStore.currentYear === "All years") {
        this.resetCharts();
        return;
      }

      const key = this.currentFiltersKey();
      const cached = this.chartsByKey[key];
      if (!force && cached) {
        this.applyCacheEntry(cached);
        return;
      }

      await Promise.all([
        this.fetchDistanceByMonths(),
        this.fetchElevationByMonths(),
        this.fetchAverageSpeedByMonths(),
        this.fetchDistanceByWeeks(),
        this.fetchElevationByWeeks(),
        this.fetchCadenceByWeeks(),
      ]);
    },
    async refreshCharts() {
      await this.ensureLoaded(true);
    },
  },
});
