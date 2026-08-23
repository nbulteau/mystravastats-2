import { describe, expect, it } from "vitest";
import type { BadgeCheckResult, ClimbDetails } from "@/models/badge-check-result.model";
import {
  buildClimbMapSummits,
  climbSummitId,
  climbVariantId,
  clusterClimbMapSummits,
  filterClimbMapSummits,
  type ClimbMapFilters,
} from "@/utils/climb-map";

function climb(label: string, name: string, category: string, latitude: number, longitude: number, climbed = false): BadgeCheckResult {
  const details: ClimbDetails = {
    name,
    country: "FR",
    massif: "Alpes",
    summitCoordinate: { latitude, longitude },
    startCoordinate: { latitude: latitude - 0.1, longitude: longitude - 0.1 },
    summitAltitude: 2000,
    minimumAltitude: 800,
    lengthKm: 15,
    totalAscent: 1200,
    difficulty: 900,
    averageGradient: 8,
    maximumGradient: 12,
    profile: [],
    ascentCount: climbed ? 2 : 0,
    bestAscent: climbed ? { activityId: 1, date: "2026-07-01", durationSeconds: 3600 } : null,
  };
  return {
    badge: { label, type: "RideFamousClimbBadge", description: "", category },
    activities: [],
    nbCheckedActivities: climbed ? 2 : 0,
    climbDetails: details,
  };
}

describe("climb map data", () => {
  const galibierNorth = climb("Col du Galibier from Valloire", "Col du Galibier", "HC", 45.064, 6.408, true);
  const galibierSouth = climb("Col du Galibier from Briançon", "Col du Galibier", "1", 45.064, 6.408);
  const izoard = climb("Col d'Izoard from Briançon", "Col d'Izoard", "1", 44.82, 6.735);

  it("uses stable summit and variant identifiers", () => {
    expect(climbSummitId(galibierNorth)).toBe(climbSummitId(galibierSouth));
    expect(climbVariantId(galibierNorth)).not.toBe(climbVariantId(galibierSouth));
  });

  it("groups variants into one summit marker", () => {
    const summits = buildClimbMapSummits([galibierNorth, galibierSouth, izoard]);
    expect(summits).toHaveLength(2);
    expect(summits.find((summit) => summit.name === "Col du Galibier")?.variants).toHaveLength(2);
    expect(summits.find((summit) => summit.name === "Col du Galibier")?.ascentCount).toBe(2);
  });

  it("filters by category, state and favorites", () => {
    const summits = buildClimbMapSummits([galibierNorth, galibierSouth, izoard]);
    const filters: ClimbMapFilters = { country: "ALL", massif: "ALL", category: "HC", status: "CLIMBED" };
    expect(filterClimbMapSummits(summits, filters, new Set())).toHaveLength(1);

    filters.status = "FAVORITE";
    const galibierId = climbSummitId(galibierNorth);
    expect(filterClimbMapSummits(summits, filters, new Set([galibierId]))[0]?.id).toBe(galibierId);
  });

  it("clusters nearby summits at overview zoom and separates them up close", () => {
    const nearby = climb("Nearby from valley", "Nearby", "2", 45.07, 6.41);
    const summits = buildClimbMapSummits([galibierNorth, nearby]);
    expect(clusterClimbMapSummits(summits, 6)).toHaveLength(1);
    expect(clusterClimbMapSummits(summits, 11)).toHaveLength(2);
  });
});
