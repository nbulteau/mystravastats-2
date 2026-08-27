import { describe, expect, it } from "vitest";
import type { GeneratedRoute } from "@/models/route-recommendation.model";
import {
  artFitScore,
  coordinateDistanceKm,
  diagnosticTone,
  highlightedRouteReasons,
  routeProductBadges,
  routeShapeSimilarity,
  routeSourceLabel,
} from "@/services/route-presentation";

function route(overrides: Partial<GeneratedRoute> = {}): GeneratedRoute {
  return {
    routeId: "route-1",
    title: "Route 1",
    variantType: "shape_primary",
    distanceKm: 42,
    elevationGainM: 500,
    durationSec: 7200,
    estimatedDurationSec: 7200,
    score: { global: 80, distance: 90, elevation: 70, duration: 75, direction: 80, shape: 90, roadFitness: 85 },
    reasons: [],
    previewLatLng: [],
    isRoadGraphGenerated: true,
    ...overrides,
  };
}

describe("route presentation", () => {
  it("computes a bounded art-fit score", () => {
    expect(artFitScore(route())).toBe(89);
    expect(artFitScore(route({ score: { ...route().score, global: 500, shape: -20 } }))).toBe(10);
  });

  it("extracts shape metadata into badges and highlights", () => {
    const presented = route({
      reasons: [
        " Shape mode: nearest-road trace ",
        "Selection profile: strict",
        "Shape similarity: 87.6%",
        "Shape trace snap: 18 anchors",
      ],
    });

    expect(routeSourceLabel(presented)).toBe("Drawing-first road snap");
    expect(routeShapeSimilarity(presented)).toBe(88);
    expect(routeProductBadges(presented).map((badge) => badge.id)).toEqual(["mode-nearest", "profile-strict"]);
    expect(highlightedRouteReasons(presented)).toContain("Visual match: 88% shape similarity.");
  });

  it("keeps diagnostic severity and geographic distance deterministic", () => {
    expect(diagnosticTone("FAILURE_SUMMARY")).toBe("error");
    expect(diagnosticTone("DIRECTION_RELAXED")).toBe("warn");
    expect(diagnosticTone("EDIT_ROUTE_UPDATED")).toBe("info");
    expect(coordinateDistanceKm([48.8566, 2.3522], [48.8566, 2.3522])).toBe(0);
    expect(coordinateDistanceKm([48.8566, 2.3522], [51.5074, -0.1278])).toBeCloseTo(343.6, 0);
  });
});
