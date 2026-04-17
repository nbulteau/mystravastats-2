import { defineStore } from "pinia";
import type { Activity } from "@/models/activity.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

export const useActivitiesStore = defineStore("activities", {
  state: () => ({
    activities: [] as Activity[],
    activitiesByKey: {} as Record<string, Activity[]>,
    isLoading: false,
    error: null as string | null,
  }),
  actions: {
    currentFiltersKey(): string {
      return useContextStore().currentFiltersKey;
    },
    async fetchActivities() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("activities", contextStore.currentActivityType, contextStore.currentYear);
      this.isLoading = true;
      this.error = null;
      try {
        this.activities = await requestJson<Activity[]>(url);
        this.activitiesByKey[this.currentFiltersKey()] = this.activities;
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Unable to load activities.";
        this.activities = this.activitiesByKey[this.currentFiltersKey()] ?? this.activities;
      } finally {
        this.isLoading = false;
      }
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      if (!force && this.activitiesByKey[key]) {
        this.activities = this.activitiesByKey[key];
        this.error = null;
        return;
      }
      await this.fetchActivities();
    },
  },
});
