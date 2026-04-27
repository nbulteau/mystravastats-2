import { defineStore } from "pinia";
import { requestJson } from "@/stores/api";
import type { DataQualityReport } from "@/models/data-quality.model";
import type { HealthDetailsPayload } from "@/models/health.model";
import type { SourceModePreview, SourceModePreviewRequest } from "@/models/source-mode.model";

export const useDiagnosticsStore = defineStore("diagnostics", {
  state: () => ({
    health: null as HealthDetailsPayload | null,
    dataQualityReport: null as DataQualityReport | null,
    sourceModePreview: null as SourceModePreview | null,
    isLoading: false,
    isPreviewingSourceMode: false,
    error: null as string | null,
    sourceModePreviewError: null as string | null,
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
        try {
          const report = await requestJson<DataQualityReport>("/api/data-quality/issues", {
            method: "GET",
            headers: {
              Accept: "application/json",
            },
          });
          this.dataQualityReport = normalizeDataQualityReport(report);
        } catch {
          this.dataQualityReport = null;
        }
        this.lastLoadedAt = new Date().toISOString();
      } catch (error) {
        this.error = error instanceof Error ? error.message : "Unable to load diagnostics.";
        this.dataQualityReport = null;
      } finally {
        this.isLoading = false;
      }
    },
    async previewSourceMode(request: SourceModePreviewRequest): Promise<SourceModePreview> {
      this.isPreviewingSourceMode = true;
      this.sourceModePreviewError = null;
      try {
        const preview = await requestJson<SourceModePreview>("/api/source-modes/preview", {
          method: "POST",
          headers: {
            Accept: "application/json",
            "Content-Type": "application/json",
          },
          body: JSON.stringify(request),
        });
        const normalizedPreview = normalizeSourceModePreview(preview);
        this.sourceModePreview = normalizedPreview;
        return normalizedPreview;
      } catch (error) {
        const message = error instanceof Error ? error.message : "Unable to preview data source.";
        this.sourceModePreviewError = message;
        throw error;
      } finally {
        this.isPreviewingSourceMode = false;
      }
    },
    async excludeActivityFromStats(activityId: number, reason?: string): Promise<DataQualityReport> {
      const report = await requestJson<DataQualityReport>(`/api/data-quality/exclusions/${activityId}`, {
        method: "PUT",
        headers: {
          Accept: "application/json",
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ reason: reason ?? "Excluded from statistics after data quality audit." }),
      });
      this.dataQualityReport = normalizeDataQualityReport(report);
      return this.dataQualityReport;
    },
    async includeActivityInStats(activityId: number): Promise<DataQualityReport> {
      const report = await requestJson<DataQualityReport>(`/api/data-quality/exclusions/${activityId}`, {
        method: "DELETE",
        headers: {
          Accept: "application/json",
        },
      });
      this.dataQualityReport = normalizeDataQualityReport(report);
      return this.dataQualityReport;
    },
  },
});

function normalizeDataQualityReport(report: DataQualityReport): DataQualityReport {
  const summary = report.summary ?? {
    status: "not_applicable",
    provider: "",
    issueCount: 0,
    impactedActivities: 0,
    excludedActivities: 0,
    bySeverity: {},
    byCategory: {},
    topIssues: [],
  };
  return {
    ...report,
    exclusions: report.exclusions ?? [],
    issues: (report.issues ?? []).map((issue) => ({
      ...issue,
      excludedFromStats: issue.excludedFromStats ?? false,
    })),
    summary: {
      ...summary,
      excludedActivities: summary.excludedActivities ?? 0,
      bySeverity: summary.bySeverity ?? {},
      byCategory: summary.byCategory ?? {},
      topIssues: summary.topIssues ?? [],
    },
  };
}

function normalizeSourceModePreview(preview: SourceModePreview): SourceModePreview {
  return {
    ...preview,
    activeMode: preview.activeMode ?? preview.mode,
    active: preview.active ?? false,
    activationCommand: preview.activationCommand ?? "",
    years: preview.years ?? [],
    missingFields: preview.missingFields ?? [],
    environment: preview.environment ?? [],
    errors: preview.errors ?? [],
    recommendations: preview.recommendations ?? [],
  };
}
