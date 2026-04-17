import { defineStore } from "pinia";
import { useActivitiesStore } from "@/stores/activities";
import { useBadgesStore } from "@/stores/badges";
import { useChartsStore } from "@/stores/charts";
import { useDashboardStore } from "@/stores/dashboard";
import { useMapStore } from "@/stores/map";
import { useRoutesStore } from "@/stores/routes";
import { useSegmentsStore } from "@/stores/segments";
import { useStatisticsStore } from "@/stores/statistics";

export type AppView =
  | "statistics"
  | "activities"
  | "activity"
  | "map"
  | "badges"
  | "routes"
  | "segments"
  | "charts"
  | "dashboard"
  | "heatmap";

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
    async refreshCurrentViewData() {
      switch (this.currentView) {
        case "statistics":
          await useStatisticsStore().ensureLoaded();
          break;
        case "activities":
          await useActivitiesStore().ensureLoaded();
          break;
        case "map":
          await useMapStore().ensureLoaded();
          break;
        case "charts":
          await useChartsStore().ensureLoaded();
          break;
        case "segments":
          await useSegmentsStore().ensureLoaded();
          break;
        case "routes":
          await useRoutesStore().ensureLoaded();
          break;
        case "dashboard":
          await useDashboardStore().ensureDashboardLoaded();
          break;
        case "heatmap":
          await useDashboardStore().ensureHeatmapLoaded();
          break;
        case "badges":
          await useBadgesStore().ensureLoaded();
          break;
        case "activity":
          break;
      }
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
