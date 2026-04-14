import { defineStore } from "pinia";
import { useActivitiesStore } from "@/stores/activities";
import { useBadgesStore } from "@/stores/badges";
import { useChartsStore } from "@/stores/charts";
import { useDashboardStore } from "@/stores/dashboard";
import { useMapStore } from "@/stores/map";
import { useStatisticsStore } from "@/stores/statistics";

export type AppView =
  | "statistics"
  | "activities"
  | "activity"
  | "map"
  | "badges"
  | "charts"
  | "dashboard"
  | "heatmap";

export const useContextStore = defineStore("context", {
  state: () => ({
    currentYear: new Date().getFullYear().toString(),
    currentActivityType: "Commute_GravelRide_MountainBikeRide_Ride_VirtualRide",
    currentView: "statistics" as AppView,
  }),
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
