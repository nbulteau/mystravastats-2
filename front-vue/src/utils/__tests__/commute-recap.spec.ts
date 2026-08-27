import { describe, expect, it } from "vitest";
import type { Activity } from "@/models/activity.model";
import {
  COMMUTE_RECAP_PAGES,
  buildCommuteRecapStats,
  buildCommuteRecapSvg,
  calculateCommuteImpact,
} from "@/utils/commute-recap";

function activity(date: string, distanceKm: number, commute = true): Activity {
  return {
    id: Math.round(new Date(date).getTime() / 1000 + distanceKm),
    name: "Home to secret office",
    type: "Ride",
    commute,
    distance: distanceKm * 1000,
    movingTime: distanceKm * 240,
    elapsedTime: distanceKm * 260,
    totalElevationGain: distanceKm * 5,
    date,
  } as Activity;
}

describe("commute recap", () => {
  const activities = [
    activity("2026-01-02T07:30:00Z", 10),
    activity("2026-01-05T17:30:00Z", 12),
    activity("2026-01-06T08:00:00Z", 8),
    activity("2026-01-07T18:00:00Z", 10),
    activity("2026-01-08T08:00:00Z", 20),
    activity("2026-01-09T08:00:00Z", 9),
    activity("2026-01-09T18:00:00Z", 9),
    activity("2026-01-10T08:00:00Z", 99, false),
    activity("2025-01-06T08:00:00Z", 15),
  ];

  it("aggregates only explicitly tagged commutes in the selected year", () => {
    const stats = buildCommuteRecapStats(activities, "2026", new Date("2026-02-01T00:00:00Z"));

    expect(stats.trips).toBe(7);
    expect(stats.activeDays).toBe(6);
    expect(stats.distanceKm).toBe(78);
    expect(stats.averageDistanceKm).toBeCloseTo(78 / 7);
    expect(stats.longestDistanceKm).toBe(20);
    expect(stats.longestWorkdayStreak).toBe(6);
    expect(stats.morningTrips).toBe(4);
    expect(stats.eveningTrips).toBe(3);
  });

  it("calculates optional motorized-trip estimates from user assumptions", () => {
    const stats = buildCommuteRecapStats(activities, "2026");
    const impact = calculateCommuteImpact(stats, {
      motorizedSharePercent: 50,
      fuelConsumptionLitresPer100Km: 6,
      fuelPricePerLitre: 2,
      co2KgPerKm: 0.2,
    });

    expect(impact.substitutedKm).toBe(39);
    expect(impact.fuelLitres).toBeCloseTo(2.34);
    expect(impact.fuelCost).toBeCloseTo(4.68);
    expect(impact.co2Kg).toBeCloseTo(7.8);
  });

  it("renders all five cards without leaking activity names", () => {
    const stats = buildCommuteRecapStats(activities, "2026");
    for (const page of COMMUTE_RECAP_PAGES) {
      const svg = buildCommuteRecapSvg({
        page: page.id,
        year: "2026",
        athleteName: "Nicolas & Co",
        stats,
        theme: "light",
        format: "portrait",
        impactEnabled: false,
        impactConfig: {
          motorizedSharePercent: 100,
          fuelConsumptionLitresPer100Km: 6.5,
          fuelPricePerLitre: 1.85,
          co2KgPerKm: 0.192,
        },
        includeMap: false,
      });
      expect(svg).toContain("<svg");
      expect(svg).toContain("Nicolas &amp; Co");
      expect(svg).not.toContain("Home to secret office");
    }
  });
});
