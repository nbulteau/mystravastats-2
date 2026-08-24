import { describe, expect, it } from "vitest";
import type { BadgeCheckResult, ClimbDetails } from "@/models/badge-check-result.model";
import { orderClimbsForPoster, selectTopClimbsForPoster } from "@/utils/climb-poster-selection";

function climb(
  label: string,
  difficulty: number,
  lengthKm: number,
  totalAscent: number,
  averageGradient: number,
  summitAltitude: number,
): BadgeCheckResult {
  const details: ClimbDetails = {
    name: label,
    country: "FR",
    massif: "Alpes",
    summitCoordinate: { latitude: 45.2, longitude: 6.2 },
    startCoordinate: { latitude: 45.1, longitude: 6.1 },
    summitAltitude,
    minimumAltitude: 800,
    lengthKm,
    totalAscent,
    difficulty,
    averageGradient,
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
    climb("Long col", 700, 30, 1100, 5.5, 1900),
    climb("Hard col", 1200, 20, 1300, 7.2, 2100),
    climb("Steep col", 900, 16, 900, 9.5, 1800),
    climb("Elevation col", 850, 24, 1800, 6.8, 2200),
    climb("High col", 820, 18, 1200, 6.4, 2600),
  ];

  it("orders the displayed climbs by descending difficulty", () => {
    expect(orderClimbsForPoster(climbs, "hardest").map((result) => result.badge.label))
      .toEqual(["Hard col", "Steep col", "Elevation col", "High col", "Long col"]);
  });

  it("orders the displayed climbs by descending distance", () => {
    expect(orderClimbsForPoster(climbs, "longest").map((result) => result.badge.label))
      .toEqual(["Long col", "Elevation col", "Hard col", "High col", "Steep col"]);
  });

  it("orders the displayed climbs by descending average gradient", () => {
    expect(orderClimbsForPoster(climbs, "steepest").map((result) => result.badge.label))
      .toEqual(["Steep col", "Hard col", "Elevation col", "High col", "Long col"]);
  });

  it("orders the displayed climbs by descending elevation gain", () => {
    expect(orderClimbsForPoster(climbs, "elevation-gain").map((result) => result.badge.label))
      .toEqual(["Elevation col", "Hard col", "High col", "Long col", "Steep col"]);
  });

  it("orders the displayed climbs by descending summit altitude", () => {
    expect(orderClimbsForPoster(climbs, "highest").map((result) => result.badge.label))
      .toEqual(["High col", "Elevation col", "Hard col", "Long col", "Steep col"]);
  });

  it("does not mutate the original climb order", () => {
    orderClimbsForPoster(climbs, "hardest");

    expect(climbs.map((result) => result.badge.label))
      .toEqual(["Long col", "Hard col", "Steep col", "Elevation col", "High col"]);
  });

  it("selects only the first climbs from the requested ordering", () => {
    expect(selectTopClimbsForPoster(climbs, "hardest", 2).map((result) => result.badge.label))
      .toEqual(["Hard col", "Steep col"]);
    expect(selectTopClimbsForPoster(climbs, "longest", 2).map((result) => result.badge.label))
      .toEqual(["Long col", "Elevation col"]);
    expect(selectTopClimbsForPoster(climbs, "steepest", 2).map((result) => result.badge.label))
      .toEqual(["Steep col", "Hard col"]);
    expect(selectTopClimbsForPoster(climbs, "elevation-gain", 2).map((result) => result.badge.label))
      .toEqual(["Elevation col", "Hard col"]);
    expect(selectTopClimbsForPoster(climbs, "highest", 2).map((result) => result.badge.label))
      .toEqual(["High col", "Elevation col"]);
  });
});
