import { defineStore } from "pinia";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

export const useMapStore = defineStore("map", {
  state: () => ({
    gpxCoordinates: [] as number[][][],
    gpxCoordinatesByKey: {} as Record<string, number[][][]>,
  }),
  actions: {
    currentFiltersKey(): string {
      const contextStore = useContextStore();
      return `${contextStore.currentActivityType}__${contextStore.currentYear}`;
    },
    async fetchGPXCoordinates() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("maps/gpx", contextStore.currentActivityType, contextStore.currentYear);
      this.gpxCoordinates = await requestJson<number[][][]>(url);
      this.gpxCoordinatesByKey[this.currentFiltersKey()] = this.gpxCoordinates;
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      if (!force && this.gpxCoordinatesByKey[key]) {
        this.gpxCoordinates = this.gpxCoordinatesByKey[key];
        return;
      }
      await this.fetchGPXCoordinates();
    },
  },
});
