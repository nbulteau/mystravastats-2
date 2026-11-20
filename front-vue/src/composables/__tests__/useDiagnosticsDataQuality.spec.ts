import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useDiagnosticsDataQuality } from "@/composables/useDiagnosticsDataQuality";
import { useDiagnosticsStore } from "@/stores/diagnostics";
import { useUiStore } from "@/stores/ui";
import type {
  DataQualityCorrectionPreview,
  DataQualityIssue,
  DataQualityReport,
} from "@/models/data-quality.model";

describe("useDiagnosticsDataQuality", () => {
  let diagnosticsStore: ReturnType<typeof useDiagnosticsStore>;
  let uiStore: ReturnType<typeof useUiStore>;
  let issues: DataQualityIssue[];

  beforeEach(() => {
    setActivePinia(createPinia());
    diagnosticsStore = useDiagnosticsStore();
    uiStore = useUiStore();
    issues = [
      issue({
        id: "gps-glitch",
        activityId: 101,
        activityName: "Morning Ride",
        severity: "critical",
        category: "GPS_GLITCH",
        field: "trace",
        message: "GPS jump affects recorded distance",
        correction: { available: true, safety: "safe", type: "REMOVE_GPS_POINT" },
      }),
      issue({
        id: "missing-power",
        activityId: 202,
        activityName: "Evening Ride",
        severity: "warning",
        category: "MISSING_STREAM_FIELD",
        field: "power",
        message: "Power stream is incomplete",
        correction: { available: true, safety: "manual", type: "RECALCULATE_FROM_STREAM" },
      }),
      issue({
        id: "fallback-distance",
        activityId: 303,
        activityName: "Commute",
        severity: "info",
        category: "FALLBACK_VALUE",
        field: "distance",
        message: "Distance uses a fallback value",
        correction: { available: false, safety: "unsupported" },
      }),
    ];
    diagnosticsStore.dataQualityReport = report(issues);
  });

  it("combines severity, activity, field, action, and impact filters", () => {
    const quality = useDiagnosticsDataQuality(diagnosticsStore, uiStore);

    quality.dataQualitySeverityFilter.value = "warning";
    quality.dataQualityActivityFilter.value = "  EVENING ";
    quality.dataQualityFieldFilter.value = "power";
    quality.dataQualityActionFilter.value = "manual";
    quality.dataQualityImpactFilter.value = "sensor";

    expect(quality.filteredDataQualityIssues.value.map(({ id }) => id)).toEqual(["missing-power"]);
    expect(quality.displayedDataQualityIssues.value.map(({ id }) => id)).toEqual(["missing-power"]);
    expect(quality.hasDataQualityFilters.value).toBe(true);

    quality.clearDataQualityFilters();

    expect(quality.filteredDataQualityIssues.value.map(({ id }) => id)).toEqual([
      "gps-glitch",
      "missing-power",
      "fallback-distance",
    ]);
    expect(quality.hasDataQualityFilters.value).toBe(false);
    expect(quality.showAllDataQualityIssues.value).toBe(false);
  });

  it("shows the issue preview first and expands to all filtered issues on request", () => {
    const manyIssues = Array.from({ length: 14 }, (_, index) => issue({ id: `issue-${index + 1}` }));
    diagnosticsStore.dataQualityReport = report(manyIssues);
    const quality = useDiagnosticsDataQuality(diagnosticsStore, uiStore);

    expect(quality.dataQualityPreviewLimit).toBe(12);
    expect(quality.displayedDataQualityIssues.value).toHaveLength(12);
    expect(quality.canToggleDataQualityIssues.value).toBe(true);

    quality.showAllDataQualityIssues.value = true;

    expect(quality.displayedDataQualityIssues.value).toHaveLength(14);
    expect(quality.dataQualityIssueListLabel.value).toBe("14 issues");
  });

  it("previews a selected issue and applies only an available safe correction", async () => {
    const quality = useDiagnosticsDataQuality(diagnosticsStore, uiStore);
    const preview = correctionPreview();
    const previewAction = vi.spyOn(diagnosticsStore, "previewCorrection").mockResolvedValue(preview);
    const applyAction = vi.spyOn(diagnosticsStore, "applyCorrection").mockResolvedValue(report(issues));

    await quality.previewIssueCorrection(issues[0]);

    expect(previewAction).toHaveBeenCalledWith("gps-glitch");
    expect(quality.selectedQualityIssueId.value).toBe("gps-glitch");
    expect(quality.issueCorrectionPreview.value).toEqual(preview);
    expect(quality.issueCorrectionPreviewLoading.value).toBe(false);

    await quality.applyIssueCorrection(issues[0]);

    expect(applyAction).toHaveBeenCalledWith("gps-glitch");
    expect(quality.selectedQualityIssueId.value).toBeNull();
    expect(quality.issueCorrectionPreview.value).toBeNull();
    expect(uiStore.toasts.some(({ message }) => message === "Local correction applied.")).toBe(true);
  });

  it("does not apply a manual-review correction from the single-fix action", async () => {
    const quality = useDiagnosticsDataQuality(diagnosticsStore, uiStore);
    const applyAction = vi.spyOn(diagnosticsStore, "applyCorrection").mockResolvedValue(report(issues));

    await quality.applyIssueCorrection(issues[1]);

    expect(applyAction).not.toHaveBeenCalled();
    expect(uiStore.toasts).toEqual([]);
  });
});

function issue(overrides: Partial<DataQualityIssue> & Pick<DataQualityIssue, "id">): DataQualityIssue {
  return {
    id: overrides.id,
    source: "FIT",
    activityId: null,
    activityName: null,
    activityType: "Ride",
    year: "2026",
    severity: "info",
    category: "INVALID_VALUE",
    field: "distance",
    message: "Issue",
    excludedFromStats: false,
    correction: null,
    ...overrides,
  };
}

function report(dataQualityIssues: DataQualityIssue[]): DataQualityReport {
  return {
    generatedAt: "2026-11-19T00:00:00Z",
    summary: {
      status: "warning",
      provider: "FIT",
      issueCount: dataQualityIssues.length,
      impactedActivities: dataQualityIssues.length,
      excludedActivities: 0,
      correctionCount: 0,
      safeCorrectionCount: 1,
      manualReviewCount: 1,
      bySeverity: {},
      byCategory: {},
      topIssues: dataQualityIssues,
    },
    issues: dataQualityIssues,
    exclusions: [],
    corrections: [],
  };
}

function correctionPreview(): DataQualityCorrectionPreview {
  return {
    generatedAt: "2026-11-19T00:00:00Z",
    mode: "single",
    summary: {
      safeCorrectionCount: 1,
      manualReviewCount: 0,
      unsupportedIssueCount: 0,
      activityCount: 1,
      distanceDeltaMeters: -12,
      elevationDeltaMeters: 0,
      modifiedFields: ["trace"],
      potentiallyImpactsRecords: false,
    },
    corrections: [],
    warnings: [],
    blockingReasons: [],
  };
}
