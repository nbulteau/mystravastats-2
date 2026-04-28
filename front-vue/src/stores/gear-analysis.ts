import { defineStore } from "pinia";
import {
  emptyGearAnalysis,
  type GearAnalysis,
  type GearMaintenanceRecord,
  type GearMaintenanceRecordRequest,
} from "@/models/gear-analysis.model";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import { ErrorService } from "@/services/error.service";
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
        this.analysis = normalizeGearAnalysis(await requestJson<GearAnalysis>(url));
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
    async saveMaintenanceRecord(request: GearMaintenanceRecordRequest): Promise<GearMaintenanceRecord> {
      const record = await requestJson<GearMaintenanceRecord>("/api/gear-analysis/maintenance", {
        method: "POST",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify(request),
      });
      await this.fetchGearAnalysis();
      return record;
    },
    async deleteMaintenanceRecord(recordId: string): Promise<void> {
      const response = await fetch(`/api/gear-analysis/maintenance/${encodeURIComponent(recordId)}`, {
        method: "DELETE",
        headers: {
          Accept: "application/json",
        },
      });
      if (!response.ok) {
        await ErrorService.catchError(response);
      }
      await this.fetchGearAnalysis();
    },
  },
});

function normalizeGearAnalysis(analysis: GearAnalysis): GearAnalysis {
  return {
    ...analysis,
    items: (analysis.items ?? []).map((item) => ({
      ...item,
      totalDistance: item.totalDistance ?? item.distance ?? 0,
      maintenanceTasks: item.maintenanceTasks ?? [],
      maintenanceHistory: item.maintenanceHistory ?? [],
      monthlyDistance: item.monthlyDistance ?? [],
    })),
    unassigned: analysis.unassigned ?? emptyGearAnalysis().unassigned,
    coverage: analysis.coverage ?? emptyGearAnalysis().coverage,
  };
}
