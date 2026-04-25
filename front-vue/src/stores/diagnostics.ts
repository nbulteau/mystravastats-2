import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import type { HealthDetailsPayload } from "@/models/health.model";

export const useDiagnosticsStore = defineStore("diagnostics", {
  state: () => ({
    health: null as HealthDetailsPayload | null,
    isLoading: false,
    error: null as string | null,
    lastLoadedAt: null as string | null,
  }),
  getters: {
    hasHealth(state): boolean {
      return state.health !== null;
    },
  },
  actions: {
    async ensureLoaded() {
      if (this.health !== null || this.isLoading) {
        return;
      }
      await this.refreshDiagnostics();
    },
    async refreshDiagnostics() {
      this.isLoading = true;
      this.error = null;
      try {
        this.health = await requestJson<HealthDetailsPayload>("/api/health/details", {
          method: "GET",
          headers: {
            Accept: "application/json",
          },
        });
        this.lastLoadedAt = new Date().toISOString();
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Unable to load diagnostics.";
      } finally {
        this.isLoading = false;
      }
    },
  },
});
