import { defineStore } from "pinia";
import type { MapPassages, MapTrack } from "@/models/map.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

export type MapViewport = {
  center: [number, number];
  zoom: number;
};

export const useMapStore = defineStore("map", {
  state: () => ({
    mapTracks: [] as MapTrack[],
    mapPassages: emptyMapPassages(),
    mapTracksByKey: {} as Record<string, MapTrack[]>,
    mapPassagesByKey: {} as Record<string, MapPassages>,
    viewportByKey: {} as Record<string, MapViewport>,
    isLoading: false,
    isPassagesLoading: false,
    error: null as string | null,
    passagesError: null as string | null,
  }),
  actions: {
    currentFiltersKey(): string {
      return useContextStore().currentFiltersKey;
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
      const tracksUrl = buildFilteredApiUrl("maps/gpx", contextStore.currentActivityType, contextStore.currentYear);
      const currentKey = this.currentFiltersKey();
      this.isLoading = true;
      this.error = null;
      try {
        const tracks = await requestJson<MapTrack[]>(tracksUrl);
        this.mapTracks = tracks;
        this.mapTracksByKey[currentKey] = this.mapTracks;
      } catch {
        this.error = "Unable to load map tracks for the selected filters.";
        this.mapTracks = this.mapTracksByKey[currentKey] ?? this.mapTracks;
      } finally {
        this.isLoading = false;
      }
    },
    async fetchMapPassages() {
      const contextStore = useContextStore();
      const passagesUrl = buildFilteredApiUrl("maps/passages", contextStore.currentActivityType, contextStore.currentYear);
      const currentKey = this.currentFiltersKey();
      this.isPassagesLoading = true;
      this.passagesError = null;
      try {
        this.mapPassages = normalizeMapPassages(await requestJson<MapPassages>(passagesUrl));
        this.mapPassagesByKey[currentKey] = this.mapPassages;
      } catch {
        this.passagesError = "Unable to load route frequency for the selected filters.";
        this.mapPassages = this.mapPassagesByKey[currentKey] ?? this.mapPassages;
      } finally {
        this.isPassagesLoading = false;
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
    async ensurePassagesLoaded(force = false) {
      const key = this.currentFiltersKey();
      if (!force && this.mapPassagesByKey[key]) {
        this.mapPassages = this.mapPassagesByKey[key];
        this.passagesError = null;
        return;
      }
      await this.fetchMapPassages();
    },
  },
});

function emptyMapPassages(): MapPassages {
  return {
    segments: [],
    includedActivities: 0,
    excludedActivities: 0,
    missingStreamActivities: 0,
    resolutionMeters: 120,
    minPassageCount: 1,
    omittedSegments: 0,
  };
}

function normalizeMapPassages(passages: MapPassages): MapPassages {
  return {
    ...emptyMapPassages(),
    ...passages,
    segments: passages.segments ?? [],
  };
}
