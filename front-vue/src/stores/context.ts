import { defineStore } from "pinia";
import { useActivitiesStore } from "@/stores/activities";
import { useBadgesStore } from "@/stores/badges";
import { useChartsStore } from "@/stores/charts";
import { useDashboardStore } from "@/stores/dashboard";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useGearAnalysisStore } from "@/stores/gear-analysis";
import { useMapStore } from "@/stores/map";
import { useRoutesStore } from "@/stores/routes";
import { useSegmentsStore } from "@/stores/segments";
import { useStatisticsStore } from "@/stores/statistics";

export type AppView =
  | "statistics"
  | "gear"
  | "activities"
  | "activity"
  | "map"
  | "badges"
  | "routes"
  | "segments"
  | "diagnostics"
  | "charts"
  | "dashboard"
  | "heatmap"
  | "settings";

export const useContextStore = defineStore("context", {
  state: () => ({
    currentYear: new Date().getFullYear().toString(),
    currentActivityType: "Commute_GravelRide_MountainBikeRide_Ride_VirtualRide",
    currentView: "dashboard" as AppView,
  }),
  getters: {
    currentFiltersKey: (state) => `${state.currentActivityType}__${state.currentYear}`,
  },
  actions: {
    invalidateActivityDerivedCaches() {
      useActivitiesStore().invalidateCache();
      useBadgesStore().invalidateCache();
      useChartsStore().invalidateCache();
      useDashboardStore().invalidateCache();
      useGearAnalysisStore().invalidateCache();
      useMapStore().invalidateCache();
      useSegmentsStore().invalidateCache();
      useStatisticsStore().invalidateCache();
    },
    async refreshCurrentViewData(force = false) {
      switch (this.currentView) {
        case "statistics":
          await useStatisticsStore().ensureLoaded(force);
          break;
        case "gear":
          await useGearAnalysisStore().ensureLoaded(force);
          break;
        case "activities":
          await useActivitiesStore().ensureLoaded(force);
          break;
        case "map":
          await useMapStore().ensureLoaded(force);
          break;
        case "charts":
          await useChartsStore().ensureLoaded(force);
          break;
        case "segments":
          await useSegmentsStore().ensureLoaded(force);
          break;
        case "diagnostics":
          if (force) {
            await useDiagnosticsStore().refreshDiagnostics();
          } else {
            await useDiagnosticsStore().ensureLoaded();
          }
          break;
        case "routes":
          await useRoutesStore().ensureLoaded();
          break;
        case "dashboard":
          await useDashboardStore().ensureDashboardLoaded(force);
          break;
        case "heatmap":
          await useDashboardStore().ensureHeatmapLoaded(force);
          break;
        case "badges":
          await useBadgesStore().ensureLoaded(force);
          break;
        case "settings":
          break;
        case "activity":
          break;
      }
    },
    async refreshAfterActivityDataChanged() {
      this.invalidateActivityDerivedCaches();
      await this.refreshCurrentViewData(true);
    },
    async updateCurrentYear(currentYear: string) {
      if (this.currentYear === currentYear) {
        return;
      }
      this.currentYear = currentYear;
      await this.refreshCurrentViewData();
    },
    async updateCurrentActivityType(activityType: string) {
      if (this.currentActivityType === activityType) {
        return;
      }
      this.currentActivityType = activityType;
      await this.refreshCurrentViewData();
    },
    updateCurrentView(view: AppView) {
      this.currentView = view;
      void this.refreshCurrentViewData();
    },
  },
});
