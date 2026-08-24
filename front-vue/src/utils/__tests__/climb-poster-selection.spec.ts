import { describe, expect, it } from "vitest";
import type { BadgeCheckResult, ClimbDetails } from "@/models/badge-check-result.model";
import { orderClimbsForPoster, selectTopClimbsForPoster } from "@/utils/climb-poster-selection";

function climb(label: string, difficulty: number, lengthKm: number): BadgeCheckResult {
  const details: ClimbDetails = {
    name: label,
    country: "FR",
    massif: "Alpes",
    summitCoordinate: { latitude: 45.2, longitude: 6.2 },
    startCoordinate: { latitude: 45.1, longitude: 6.1 },
    summitAltitude: 2000,
    minimumAltitude: 800,
    lengthKm,
    totalAscent: 1200,
    difficulty,
    averageGradient: 7,
    profile: [],
    ascentCount: 1,
  };

  return {
    badge: { label, description: "", type: "RideFamousClimbBadge" },
    activities: [],
    nbCheckedActivities: 1,
    climbDetails: details,
  };
}

describe("climb poster selection ordering", () => {
  const climbs = [
    climb("Long col", 700, 30),
    climb("Hard col", 1200, 20),
    climb("Medium col", 900, 25),
  ];

  it("orders the displayed climbs by descending difficulty", () => {
    expect(orderClimbsForPoster(climbs, "hardest").map((result) => result.badge.label))
      .toEqual(["Hard col", "Medium col", "Long col"]);
  });

  it("orders the displayed climbs by descending distance", () => {
    expect(orderClimbsForPoster(climbs, "longest").map((result) => result.badge.label))
      .toEqual(["Long col", "Medium col", "Hard col"]);
  });

  it("does not mutate the original climb order", () => {
    orderClimbsForPoster(climbs, "hardest");

    expect(climbs.map((result) => result.badge.label))
      .toEqual(["Long col", "Hard col", "Medium col"]);
  });

  it("selects only the first climbs from the requested ordering", () => {
    expect(selectTopClimbsForPoster(climbs, "hardest", 2).map((result) => result.badge.label))
      .toEqual(["Hard col", "Medium col"]);
    expect(selectTopClimbsForPoster(climbs, "longest", 2).map((result) => result.badge.label))
      .toEqual(["Long col", "Medium col"]);
  });
});
