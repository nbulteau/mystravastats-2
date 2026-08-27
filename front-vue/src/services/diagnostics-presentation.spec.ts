import { describe, expect, it } from "vitest";
import {
  dataQualityActionLabel,
  dataQualityImpactTokens,
  formatCompositeConflict,
  parseConflictNumber,
} from "@/services/diagnostics-presentation";

describe("diagnostics presentation", () => {
  it("presents source conflicts with a typed delta", () => {
    expect(parseConflictNumber("42,5 km")).toBe(42.5);
    expect(formatCompositeConflict({
      field: "distance",
      source: "gpx",
      primary: "10000",
      other: "11250",
    }, ["FIT", "GPX"])).toMatchObject({
      title: "Distance differs between sources",
      primaryLabel: "FIT",
      otherLabel: "GPX",
      delta: "+1.25 km",
    });
  });

  it("derives fix labels and impacted metrics", () => {
    const issue = {
      id: "gps-1", source: "FIT", activityId: 1, activityName: "Ride", activityType: "Ride",
      year: "2026", filePath: "", severity: "warning", category: "GPS_GLITCH", field: "distance",
      message: "GPS distance jump", rawValue: "", suggestion: "", excludedFromStats: false, excludedAt: "",
      correction: { available: true, safety: "safe", type: "REMOVE_GPS_POINT", description: "Repair" },
    } as const;
    expect(dataQualityActionLabel(issue)).toBe("Safe local fix");
    expect(dataQualityImpactTokens(issue)).toEqual(["records", "distance"]);
  });
});
