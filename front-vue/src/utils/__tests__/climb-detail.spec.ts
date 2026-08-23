import { describe, expect, it } from "vitest";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import {
  climbAscentHistory,
  climbKilometerSectors,
  climbVariantStart,
  climbVariantTitle,
  hardestClimbSectors,
} from "@/utils/climb-detail";

const result: BadgeCheckResult = {
  badge: {
    label: "Col du Galibier from Saint-Jean-de-Maurienne",
    description: "",
    type: "RideFamousClimbBadge",
    category: "HC",
  },
  activities: [],
  nbCheckedActivities: 2,
  climbDetails: {
    name: "Col du Galibier",
    country: "FR",
    massif: "Alpes",
    summitCoordinate: { latitude: 45.064, longitude: 6.408 },
    startCoordinate: { latitude: 45.278, longitude: 6.349 },
    summitAltitude: 2642,
    minimumAltitude: 566,
    lengthKm: 3,
    totalAscent: 2076,
    difficulty: 1612,
    averageGradient: 7,
    maximumGradient: 12,
    profile: [
      { distanceKm: 0, elevation: 1000 },
      { distanceKm: 1, elevation: 1050 },
      { distanceKm: 2, elevation: 1140 },
      { distanceKm: 3, elevation: 1260 },
    ],
    ascentCount: 2,
    bestAscent: { activityId: 8, activityName: "Galibier", date: "2026-07-12", durationSeconds: 3600 },
    ascents: [
      { activityId: 8, activityName: "Galibier", date: "2026-07-12", durationSeconds: 3600 },
      { activityId: 9, activityName: "Maurienne", date: "2026-08-02", durationSeconds: 3800 },
    ],
  },
};

describe("climb detail data", () => {
  it("keeps the complete variant title and translates the departure separator", () => {
    expect(climbVariantTitle(result)).toBe("Col du Galibier depuis Saint-Jean-de-Maurienne");
    expect(climbVariantStart(result)).toBe("Saint-Jean-de-Maurienne");
  });

  it("orders personal ascents from newest to oldest", () => {
    expect(climbAscentHistory(result).map((ascent) => ascent.activityId)).toEqual([9, 8]);
  });

  it("builds kilometre sectors and identifies the hardest ones", () => {
    expect(climbKilometerSectors(result.climbDetails!)).toHaveLength(3);
    expect(hardestClimbSectors(result.climbDetails!, 2).map((sector) => sector.averageGradient)).toEqual([12, 9]);
  });

  it("falls back to the best effort for a response produced before ascent history existed", () => {
    const legacy = {
      ...result,
      climbDetails: { ...result.climbDetails!, ascents: undefined },
    };
    expect(climbAscentHistory(legacy)).toEqual([result.climbDetails!.bestAscent]);
  });
});
