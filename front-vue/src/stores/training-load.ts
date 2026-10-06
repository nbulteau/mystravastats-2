import { defineStore } from "pinia";
import { requestJson } from "@/services/http-client";
import { apiUrl } from "@/services/api-url";
import type { ApiTrainingReport } from "@/generated/api-contract";
import { useContextStore } from "./context";

function today(): string {
  const date = new Date();
  return `${date.getFullYear()}-${String(date.getMonth()+1).padStart(2,"0")}-${String(date.getDate()).padStart(2,"0")}`;
}
export const useTrainingLoadStore = defineStore("training-load", {
  state: () => ({ week: today(), report: null as ApiTrainingReport | null, loading: false, error: "", requestId: 0 }),
  actions: {
    async load() {
      const requestId = ++this.requestId;
      this.loading = true; this.error = ""; this.report = null;
      try {
        const report = await requestJson<ApiTrainingReport>(apiUrl("getTrainingLoad", {query: {week: this.week, activityType: useContextStore().currentActivityType}}));
        if (requestId === this.requestId) this.report = report;
      } catch (error) {
        if (requestId === this.requestId) this.error = error instanceof Error ? error.message : "Unable to load training report.";
      } finally {
        if (requestId === this.requestId) this.loading = false;
      }
    },
    async changeWeek(value: string) { if (value) { this.week = value; await this.load(); } },
  },
});
