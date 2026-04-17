import { defineStore } from "pinia";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";
import { DashboardData } from "@/models/dashboard-data.model";
import type { ChartPeriodPoint } from "@/models/chart-period-point.model";
import { normalizePeriodPoints } from "@/utils/charts";

type ChartsCacheEntry = {
  distanceByMonths: ChartPeriodPoint[];
  elevationByMonths: ChartPeriodPoint[];
  averageSpeedByMonths: ChartPeriodPoint[];
  distanceByWeeks: ChartPeriodPoint[];
  elevationByWeeks: ChartPeriodPoint[];
  cadenceByWeeks: ChartPeriodPoint[];
  activitiesCountByYear: Record<string, number>;
  totalDistanceByYear: Record<string, number>;
  totalElevationByYear: Record<string, number>;
  averageSpeedByYear: Record<string, number>;
  maxSpeedByYear: Record<string, number>;
};

type LegacyPeriodPoint = Record<string, number>;
type ChartPeriodPointResponse = ChartPeriodPoint | LegacyPeriodPoint;

export const useChartsStore = defineStore("charts", {
  state: () => ({
    distanceByMonths: [] as ChartPeriodPoint[],
    elevationByMonths: [] as ChartPeriodPoint[],
    averageSpeedByMonths: [] as ChartPeriodPoint[],
    distanceByWeeks: [] as ChartPeriodPoint[],
    elevationByWeeks: [] as ChartPeriodPoint[],
    cadenceByWeeks: [] as ChartPeriodPoint[],
    activitiesCountByYear: {} as Record<string, number>,
    totalDistanceByYear: {} as Record<string, number>,
    totalElevationByYear: {} as Record<string, number>,
    averageSpeedByYear: {} as Record<string, number>,
    maxSpeedByYear: {} as Record<string, number>,
    isLoading: false,
    error: null as string | null,
    chartsByKey: {} as Record<string, ChartsCacheEntry>,
  }),
  actions: {
    currentFiltersKey(): string {
      const contextStore = useContextStore();
      return contextStore.currentFiltersKey;
    },
    updateCacheForCurrentKey() {
      this.chartsByKey[this.currentFiltersKey()] = {
        distanceByMonths: this.distanceByMonths,
        elevationByMonths: this.elevationByMonths,
        averageSpeedByMonths: this.averageSpeedByMonths,
        distanceByWeeks: this.distanceByWeeks,
        elevationByWeeks: this.elevationByWeeks,
        cadenceByWeeks: this.cadenceByWeeks,
        activitiesCountByYear: this.activitiesCountByYear,
        totalDistanceByYear: this.totalDistanceByYear,
        totalElevationByYear: this.totalElevationByYear,
        averageSpeedByYear: this.averageSpeedByYear,
        maxSpeedByYear: this.maxSpeedByYear,
      };
    },
    applyCacheEntry(entry: ChartsCacheEntry) {
      this.distanceByMonths = entry.distanceByMonths;
      this.elevationByMonths = entry.elevationByMonths;
      this.averageSpeedByMonths = entry.averageSpeedByMonths;
      this.distanceByWeeks = entry.distanceByWeeks;
      this.elevationByWeeks = entry.elevationByWeeks;
      this.cadenceByWeeks = entry.cadenceByWeeks;
      this.activitiesCountByYear = entry.activitiesCountByYear;
      this.totalDistanceByYear = entry.totalDistanceByYear;
      this.totalElevationByYear = entry.totalElevationByYear;
      this.averageSpeedByYear = entry.averageSpeedByYear;
      this.maxSpeedByYear = entry.maxSpeedByYear;
      this.error = null;
    },
    async fetchDistanceByMonths() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/distance-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=MONTHS";
      this.distanceByMonths = normalizePeriodPoints(await requestJson<ChartPeriodPointResponse[]>(url));
      this.updateCacheForCurrentKey();
    },
    async fetchElevationByMonths() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/elevation-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=MONTHS";
      this.elevationByMonths = normalizePeriodPoints(await requestJson<ChartPeriodPointResponse[]>(url));
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
      this.averageSpeedByMonths = normalizePeriodPoints(await requestJson<ChartPeriodPointResponse[]>(url));
      this.updateCacheForCurrentKey();
    },
    async fetchDistanceByWeeks() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/distance-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=WEEKS";
      this.distanceByWeeks = normalizePeriodPoints(await requestJson<ChartPeriodPointResponse[]>(url));
      this.updateCacheForCurrentKey();
    },
    async fetchElevationByWeeks() {
      const contextStore = useContextStore();
      const url =
        buildFilteredApiUrl("charts/elevation-by-period", contextStore.currentActivityType, contextStore.currentYear) +
        "&period=WEEKS";
      this.elevationByWeeks = normalizePeriodPoints(await requestJson<ChartPeriodPointResponse[]>(url));
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
      this.cadenceByWeeks = normalizePeriodPoints(await requestJson<ChartPeriodPointResponse[]>(url));
      this.updateCacheForCurrentKey();
    },
    async fetchAllYearsOverview() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("dashboard", contextStore.currentActivityType, contextStore.currentYear);
      const data = await requestJson<DashboardData>(url);

      this.activitiesCountByYear = data.nbActivitiesByYear ?? {};
      this.totalDistanceByYear = data.totalDistanceByYear ?? {};
      this.totalElevationByYear = data.totalElevationByYear ?? {};
      this.averageSpeedByYear = data.averageSpeedByYear ?? {};
      this.maxSpeedByYear = data.maxSpeedByYear ?? {};
      this.updateCacheForCurrentKey();
    },
    resetYearlyCharts() {
      this.activitiesCountByYear = {};
      this.totalDistanceByYear = {};
      this.totalElevationByYear = {};
      this.averageSpeedByYear = {};
      this.maxSpeedByYear = {};
    },
    resetCharts() {
      this.distanceByMonths = [];
      this.elevationByMonths = [];
      this.averageSpeedByMonths = [];
      this.distanceByWeeks = [];
      this.elevationByWeeks = [];
      this.cadenceByWeeks = [];
      this.resetYearlyCharts();
      this.error = null;
    },
    async ensureLoaded(force = false) {
      const contextStore = useContextStore();
      const key = this.currentFiltersKey();
      const cached = this.chartsByKey[key];
      if (!force && cached) {
        this.applyCacheEntry(cached);
        return;
      }

      this.isLoading = true;
      this.error = null;
      try {
        if (contextStore.currentYear === "All years") {
          this.distanceByMonths = [];
          this.elevationByMonths = [];
          this.averageSpeedByMonths = [];
          this.distanceByWeeks = [];
          this.elevationByWeeks = [];
          this.cadenceByWeeks = [];
          await this.fetchAllYearsOverview();
          return;
        }

      this.resetYearlyCharts();
      await Promise.all([
        this.fetchDistanceByMonths(),
        this.fetchElevationByMonths(),
        this.fetchAverageSpeedByMonths(),
        this.fetchDistanceByWeeks(),
        this.fetchElevationByWeeks(),
        this.fetchCadenceByWeeks(),
        this.fetchAllYearsOverview(),
      ]);
      } catch {
        this.error = "Unable to load chart data for the selected filters.";
      } finally {
        this.isLoading = false;
      }
    },
    async refreshCharts() {
      await this.ensureLoaded(true);
    },
  },
});
