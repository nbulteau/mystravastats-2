import { describe, expect, it } from "vitest";
import {
  comparisonDeltaClass,
  detailedActivityWarning,
  formatCadenceValue,
  formatRouteEffortDescription,
  resolveEffortGradient,
} from "@/services/activity-detail-presentation";
import type { DetailedActivity } from "@/models/activity.model";

describe("activity detail presentation", () => {
  it("formats cadence and effort details for the activity type", () => {
    expect(formatCadenceValue(85, "TrailRun")).toBe("170 spm");
    expect(formatCadenceValue(85, "Ride")).toBe("85 rpm");
    expect(formatRouteEffortDescription({ distance: 1000, seconds: 180, elevationGain: 50, averagePower: 240 }, "Ride"))
      .toContain("Grade 5.0%");
  });

  it("resolves explicit and stream-derived gradients", () => {
    expect(resolveEffortGradient({ distance: 1000, seconds: 180, grade: -4.5 })).toBe(-4.5);
    expect(resolveEffortGradient({ distance: 1000, seconds: 180, deltaAltitude: 60 })).toBe(6);
  });

  it("classifies comparison deltas and missing detailed streams", () => {
    expect(comparisonDeltaClass(2, true)).toContain("--good");
    expect(comparisonDeltaClass(2, false)).toContain("--warn");
    expect(detailedActivityWarning({ stream: { distance: [] } } as unknown as DetailedActivity)).toContain("Detailed streams");
    expect(detailedActivityWarning({ stream: { distance: [0] } } as unknown as DetailedActivity)).toBeNull();
  });
});
