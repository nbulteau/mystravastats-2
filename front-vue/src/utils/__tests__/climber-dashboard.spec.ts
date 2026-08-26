import { describe, expect, it } from "vitest";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { buildClimberDashboardStats } from "@/utils/climber-dashboard";

function climb(id: string, ascentCount: number, dates: string[], warnings: string[] = []): BadgeCheckResult {
  return {
    badge: { label: id, description: "", type: "RideFamousClimbBadge", category: "1" },
    activities: [],
    nbCheckedActivities: ascentCount,
    climbDetails: {
      summitId: `summit-${id}`,
      variantId: id,
      name: id,
      country: "FR",
      massif: "Alpes",
      summitCoordinate: { latitude: 0, longitude: 0 },
      startCoordinate: { latitude: 0, longitude: 0 },
      summitAltitude: 2000,
      minimumAltitude: 1000,
      lengthKm: id === "long" ? 20 : 10,
      totalAscent: 1000,
      difficulty: id === "hard" ? 900 : 500,
      averageGradient: 7,
      profile: [],
      ascentCount,
      ascents: dates.map((date, index) => ({
        activityId: index + 1,
        date,
        durationSeconds: 3600,
        vamMetersPerHour: id === "hard" ? 1200 : 1000,
        comparisonQuality: {
          alignmentMethod: "catalog-waypoints-by-distance",
          precision: "estimated",
          catalogDistanceKm: 10,
          detectedDistanceKm: 10,
          startOffsetMeters: 120,
          finishOffsetMeters: 80,
          warnings,
        },
      })),
    },
  };
}

describe("climber dashboard stats", () => {
  it("keeps lifetime totals, records and distributions coherent", () => {
    const stats = buildClimberDashboardStats([
      climb("long", 2, ["2025-01-01", "2026-01-01"]),
      climb("hard", 1, ["2026-02-01"]),
    ]);
    expect(stats.ascentCount).toBe(3);
    expect(stats.climbElevationGain).toBe(3000);
    expect(stats.cumulativeSummitAltitude).toBe(6000);
    expect(stats.records.longest?.label).toBe("long");
    expect(stats.records.vam?.label).toBe("hard");
    expect(stats.years.find((year) => year.key === "2026")?.ascentCount).toBe(2);
    expect(stats.countries[0]).toMatchObject({ label: "FR", climbCount: 2, ascentCount: 3 });
  });

  it("excludes records backed by incomplete streams and reports the guard", () => {
    const stats = buildClimberDashboardStats([
      climb("invalid", 1, ["2026-01-01"], ["INCOMPLETE_TIME_STREAM"]),
    ]);
    expect(stats.records.vam).toBeNull();
    expect(stats.excludedRecordCount).toBe(1);
  });
});
