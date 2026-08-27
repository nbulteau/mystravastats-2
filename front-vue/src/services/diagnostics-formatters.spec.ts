import { describe, expect, it } from "vitest";
import {
  formatBytes,
  formatDurationSeconds,
  formatProvider,
  normalizeDataQualityIssue,
  normalizeDataQualitySummary,
  normalizeSourceMode,
  sourceSyncChangedActivityData,
} from "@/services/diagnostics-formatters";

describe("diagnostics formatters", () => {
  it("normalizes a partially typed data-quality payload", () => {
    expect(normalizeDataQualitySummary({
      status: "warning",
      issueCount: "2",
      impactedActivities: 1,
      bySeverity: { warning: "2" },
      topIssues: [{ id: 42, message: " Missing stream ", excludedFromStats: "true" }, {}],
    })).toEqual(expect.objectContaining({
      status: "warning",
      issueCount: 2,
      impactedActivities: 1,
      bySeverity: { warning: 2 },
      topIssues: [expect.objectContaining({ id: "42", message: "Missing stream", excludedFromStats: true })],
    }));
    expect(normalizeDataQualitySummary(null)).toBeNull();
    expect(normalizeDataQualityIssue({})).toBeNull();
  });

  it("detects activity-changing source synchronizations", () => {
    expect(sourceSyncChangedActivityData({ reloaded: true })).toBe(true);
    expect(sourceSyncChangedActivityData({ fit: { importedFiles: "3" } })).toBe(true);
    expect(sourceSyncChangedActivityData({ fit: { importedFiles: 0 } })).toBe(false);
  });

  it("formats stable diagnostic values and source modes", () => {
    expect(formatBytes(1536)).toBe("1.5 KB");
    expect(formatDurationSeconds(3661)).toBe("1h 1m 1s");
    expect(formatProvider("gpx")).toBe("GPX");
    expect(normalizeSourceMode(" fit ")).toBe("FIT");
    expect(normalizeSourceMode("composite")).toBe("STRAVA");
  });
});
