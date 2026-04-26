import { defineStore } from "pinia";
import {
  emptyGearAnalysis,
  type GearAnalysis,
} from "@/models/gear-analysis.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { useContextStore } from "@/stores/context";

export const useGearAnalysisStore = defineStore("gearAnalysis", {
  state: () => ({
    analysis: emptyGearAnalysis() as GearAnalysis,
    analysisByKey: {} as Record<string, GearAnalysis>,
    isLoading: false,
    error: null as string | null,
  }),
  actions: {
    currentFiltersKey(): string {
      return useContextStore().currentFiltersKey;
    },
    async fetchGearAnalysis() {
      const contextStore = useContextStore();
      const url = buildFilteredApiUrl("gear-analysis", contextStore.currentActivityType, contextStore.currentYear);
      const key = this.currentFiltersKey();
      this.isLoading = true;
      this.error = null;
      try {
        this.analysis = await requestJson<GearAnalysis>(url);
        this.analysisByKey[key] = this.analysis;
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Unable to load gear analysis.";
        this.analysis = this.analysisByKey[key] ?? this.analysis;
      } finally {
        this.isLoading = false;
      }
    },
    async ensureLoaded(force = false) {
      const key = this.currentFiltersKey();
      if (!force && this.analysisByKey[key]) {
        this.analysis = this.analysisByKey[key];
        this.error = null;
        return;
      }
      await this.fetchGearAnalysis();
    },
  },
});
