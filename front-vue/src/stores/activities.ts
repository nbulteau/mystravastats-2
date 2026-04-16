import { defineStore } from "pinia";
import type { Activity } from "@/models/activity.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

export const useActivitiesStore = defineStore("activities", {
  state: () => ({
    activities: [] as Activity[],
    activitiesByKey: {} as Record<string, Activity[]>,
  }),
  actions: {
    currentFiltersKey(): string {
      return useContextStore().currentFiltersKey;
    },
    async fetchActivities() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("activities", contextStore.currentActivityType, contextStore.currentYear);
      this.activities = await requestJson<Activity[]>(url);
      this.activitiesByKey[this.currentFiltersKey()] = this.activities;
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      if (!force && this.activitiesByKey[key]) {
        this.activities = this.activitiesByKey[key];
        return;
      }
      await this.fetchActivities();
    },
  },
});
