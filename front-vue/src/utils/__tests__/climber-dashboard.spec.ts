import { describe, expect, it } from "vitest";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { buildClimberDashboardStats } from "@/utils/climber-dashboard";

function climb(
  id: string,
  ascentCount: number,
  dates: string[],
  warnings: string[] = [],
  options: { category?: string; summitAltitude?: number } = {},
): BadgeCheckResult {
  return {
    badge: { label: id, description: "", type: "RideFamousClimbBadge", category: options.category ?? "1" },
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
      summitAltitude: options.summitAltitude ?? 2000,
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

  it("orders categories and altitude bands from hardest or highest to lowest", () => {
    const stats = buildClimberDashboardStats([
      climb("category-1", 10, ["2026-01-01"], [], { category: "1", summitAltitude: 2000 }),
      climb("category-hc", 1, ["2026-01-02"], [], { category: "hc", summitAltitude: 2600 }),
      climb("category-4", 20, ["2026-01-03"], [], { category: "4", summitAltitude: 900 }),
      climb("category-2", 5, ["2026-01-04"], [], { category: "2", summitAltitude: 1500 }),
    ]);

    expect(stats.categories.map((item) => item.label)).toEqual([
      "Cat. HC",
      "Cat. 1",
      "Cat. 2",
      "Cat. 4",
    ]);
    expect(stats.altitudeBands.map((item) => item.label)).toEqual([
      "≥ 2,500 m",
      "2,000–2,499 m",
      "1,500–1,999 m",
      "< 1,000 m",
    ]);
  });
});
