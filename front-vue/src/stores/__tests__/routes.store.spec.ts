import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useRoutesStore } from "@/stores/routes";
import { requestJson } from "@/stores/api";
import type { GeneratedRoute } from "@/models/route-recommendation.model";

vi.mock("@/stores/api", () => ({
  requestJson: vi.fn(),
}));

function reverseGeometry(points: number[][]): number[][] {
  return [...points].reverse().map((point) => [point[0], point[1]]);
}

function buildRoute(overrides: Partial<GeneratedRoute> = {}): GeneratedRoute {
  return {
    routeId: "route-default",
    title: "Generated route",
    variantType: "TARGET",
    routeType: "RIDE",
    startDirection: "UNDEFINED",
    distanceKm: 40,
    elevationGainM: 800,
    durationSec: 7200,
    estimatedDurationSec: 7200,
    score: {
      global: 86,
      distance: 85,
      elevation: 84,
      duration: 83,
      direction: 82,
      shape: 81,
      roadFitness: 80,
    },
    reasons: [],
    previewLatLng: [
      [45.0, 6.0],
      [45.1, 6.1],
      [45.2, 6.0],
    ],
    start: { lat: 45.0, lng: 6.0 },
    end: { lat: 45.0, lng: 6.0 },
    activityId: undefined,
    isRoadGraphGenerated: true,
    ...overrides,
  };
}

describe("routes store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("keeps only one new geometry for each target generation click", async () => {
    const store = useRoutesStore();
    store.mode = "TARGET";
    store.startPoint = { lat: 45.0, lng: 6.0 };
    store.distanceTargetKm = 40;
    store.elevationTargetM = 800;

    const firstGeometry = [
      [45.0, 6.0],
      [45.05, 6.05],
      [45.1, 6.0],
    ];
    const secondGeometry = [
      [45.0, 6.0],
      [45.02, 6.08],
      [45.09, 6.03],
    ];

    vi.mocked(requestJson)
      .mockResolvedValueOnce({
        routes: [
          buildRoute({
            routeId: "target-route-1",
            previewLatLng: firstGeometry,
          }),
        ],
        diagnostics: [],
      })
      .mockResolvedValueOnce({
        routes: [
          buildRoute({
            routeId: "target-route-1-duplicate-id",
            previewLatLng: reverseGeometry(firstGeometry),
          }),
          buildRoute({
            routeId: "target-route-2",
            previewLatLng: secondGeometry,
          }),
        ],
        diagnostics: [],
      });

    await store.generateRoutes();
    await store.generateRoutes();

    expect(store.routes.map((route) => route.routeId)).toEqual(["target-route-1", "target-route-2"]);
    expect(store.selectedRouteId).toBe("target-route-2");
    expect(store.lastGeneratedTargetRouteNumber).toBe(2);
  });

  it("returns NO_UNIQUE_ROUTE when backend only returns duplicate geometries", async () => {
    const store = useRoutesStore();
    store.mode = "TARGET";
    store.startPoint = { lat: 45.0, lng: 6.0 };
    store.distanceTargetKm = 40;
    store.elevationTargetM = 800;

    const geometry = [
      [45.0, 6.0],
      [45.02, 6.05],
      [45.08, 6.01],
    ];

    vi.mocked(requestJson)
      .mockResolvedValueOnce({
        routes: [
          buildRoute({
            routeId: "target-route-1",
            previewLatLng: geometry,
          }),
        ],
        diagnostics: [],
      })
      .mockResolvedValueOnce({
        routes: [
          buildRoute({
            routeId: "target-route-1-reordered",
            previewLatLng: reverseGeometry(geometry),
          }),
        ],
        diagnostics: [],
      });

    await store.generateRoutes();
    await store.generateRoutes();

    expect(store.routes.map((route) => route.routeId)).toEqual(["target-route-1"]);
    expect(store.generationDiagnostics).toEqual([
      {
        code: "NO_UNIQUE_ROUTE",
        message: "No additional unique route found after geometry deduplication.",
      },
    ]);
  });

  it("deduplicates shape generation results by geometry", async () => {
    const store = useRoutesStore();
    store.mode = "SHAPE";
    store.shapePoints = [
      [45.0, 6.0],
      [45.1, 6.1],
      [45.2, 6.0],
    ];

    const shapeGeometry = [
      [45.0, 6.0],
      [45.03, 6.04],
      [45.1, 6.06],
      [45.2, 6.0],
    ];

    vi.mocked(requestJson).mockResolvedValueOnce({
      routes: [
        buildRoute({
          routeId: "shape-route-1",
          previewLatLng: shapeGeometry,
        }),
        buildRoute({
          routeId: "shape-route-1-duplicate",
          previewLatLng: reverseGeometry(shapeGeometry),
        }),
        buildRoute({
          routeId: "shape-route-2",
          previewLatLng: [
            [45.0, 6.0],
            [45.04, 6.02],
            [45.12, 6.08],
            [45.21, 6.0],
          ],
        }),
      ],
      diagnostics: [],
    });

    await store.generateRoutes();

    expect(store.routes.map((route) => route.routeId)).toEqual(["shape-route-1", "shape-route-2"]);
    expect(store.selectedRouteId).toBe("shape-route-1");
  });
});
