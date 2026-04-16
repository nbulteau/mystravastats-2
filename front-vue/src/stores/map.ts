import { defineStore } from "pinia";
import type { MapTrack } from "@/models/map.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

export type MapViewport = {
  center: [number, number];
  zoom: number;
};

export const useMapStore = defineStore("map", {
  state: () => ({
    mapTracks: [] as MapTrack[],
    mapTracksByKey: {} as Record<string, MapTrack[]>,
    viewportByKey: {} as Record<string, MapViewport>,
    isLoading: false,
    error: null as string | null,
  }),
  actions: {
    currentFiltersKey(): string {
      const contextStore = useContextStore();
      return `${contextStore.currentActivityType}__${contextStore.currentYear}`;
    },
    setViewportForCurrentFilters(viewport: MapViewport) {
      this.viewportByKey[this.currentFiltersKey()] = viewport;
    },
    getViewportForCurrentFilters(): MapViewport | null {
      return this.viewportByKey[this.currentFiltersKey()] ?? null;
    },
    clearViewportForCurrentFilters() {
      delete this.viewportByKey[this.currentFiltersKey()];
    },
    async fetchGPXCoordinates() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("maps/gpx", contextStore.currentActivityType, contextStore.currentYear);
      const currentKey = this.currentFiltersKey();
      this.isLoading = true;
      this.error = null;
      try {
        this.mapTracks = await requestJson<MapTrack[]>(url);
        this.mapTracksByKey[currentKey] = this.mapTracks;
      } catch {
        this.error = "Unable to load map tracks for the selected filters.";
        this.mapTracks = this.mapTracksByKey[currentKey] ?? this.mapTracks;
      } finally {
        this.isLoading = false;
      }
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      if (!force && this.mapTracksByKey[key]) {
        this.mapTracks = this.mapTracksByKey[key];
        this.error = null;
        return;
      }
      await this.fetchGPXCoordinates();
    },
  },
});
