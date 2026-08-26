import { describe, expect, it } from "vitest";
import type { ClimbAscent } from "@/models/badge-check-result.model";
import {
  buildSectorComparisons,
  comparisonValueAtDistance,
  defaultComparedAscentIds,
} from "@/utils/climb-comparison";

function ascent(activityId: number, times: number[]): ClimbAscent {
  return {
    activityId,
    activityName: `Activity ${activityId}`,
    date: `2026-01-0${activityId}T10:00:00Z`,
    durationSeconds: times.at(-1) ?? 0,
    comparisonPoints: times.map((elapsedSeconds, distanceKm) => ({
      distanceKm,
      elapsedSeconds,
      speedKph: 20 + distanceKm,
    })),
  };
}

describe("climb ascent comparison", () => {
  it("selects the best and latest comparable ascents by default", () => {
    const ascents = [ascent(3, [0, 100]), ascent(2, [0, 90]), ascent(1, [0, 80])];
    expect(defaultComparedAscentIds(ascents, 1)).toEqual([1, 3]);
  });

  it("interpolates a metric by distance and preserves missing data", () => {
    const candidate = ascent(1, [0, 100, 220]);
    expect(comparisonValueAtDistance(candidate, 1.5, "elapsedSeconds")).toBe(160);
    expect(comparisonValueAtDistance(candidate, 3, "elapsedSeconds")).toBeNull();
    expect(comparisonValueAtDistance(candidate, 1, "powerWatts")).toBeNull();
  });

  it("computes per-sector gains and losses against the reference", () => {
    const reference = ascent(1, [0, 100, 210]);
    const candidate = ascent(2, [0, 110, 205]);
    expect(buildSectorComparisons(candidate, reference, 2)).toEqual([
      { startKm: 0, endKm: 1, sectorSeconds: 110, deltaSeconds: 10 },
      { startKm: 1, endKm: 2, sectorSeconds: 95, deltaSeconds: -15 },
    ]);
  });
});
