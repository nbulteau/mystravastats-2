import { describe, expect, it } from "vitest";
import type { BadgeCheckResult } from "@/models/badge-check-result.model";
import { climbSummitId, climbVariantId } from "@/utils/climb-id";

function climb(summitId?: string, variantId?: string): BadgeCheckResult {
  return {
    badge: { label: "Col du Galibier from Valloire", type: "RideFamousClimbBadge", description: "" },
    activities: [],
    nbCheckedActivities: 0,
    climbDetails: {
      summitId,
      variantId,
      name: "Col du Galibier",
      country: "FR",
      massif: "Alpes",
      summitCoordinate: { latitude: 45.064, longitude: 6.408 },
      startCoordinate: { latitude: 45.16, longitude: 6.43 },
      summitAltitude: 2642,
      minimumAltitude: 1405,
      lengthKm: 18.1,
      totalAscent: 1237,
      difficulty: 1100,
      averageGradient: 6.8,
      profile: [],
      ascentCount: 0,
    },
  };
}

describe("climb identity", () => {
  it("uses API identities independently from the displayed label", () => {
    const result = climb("climb-fr-galibier", "climb-fr-galibier--valloire");
    result.badge.label = "Galibier — versant nord";
    expect(climbSummitId(result)).toBe("climb-fr-galibier");
    expect(climbVariantId(result)).toBe("climb-fr-galibier--valloire");
  });

  it("keeps a deterministic fallback for older API responses", () => {
    expect(climbSummitId(climb())).toContain("climb-fr-col-du-galibier");
    expect(climbVariantId(climb())).toContain("--col-du-galibier-from-valloire");
  });
});
