import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useRoutesStore, type BuiltInShapeTemplateKey } from "@/stores/routes";
import { requestJson } from "@/stores/api";
import type { GeneratedRoute } from "@/models/route-recommendation.model";

vi.mock("@/stores/api", () => ({
  requestJson: vi.fn(),
}));

function installMemoryStorage() {
  const storage = new Map<string, string>();
  Object.defineProperty(globalThis, "localStorage", {
    configurable: true,
    value: {
      getItem: (key: string) => storage.get(key) ?? null,
      setItem: (key: string, value: string) => storage.set(key, value),
      removeItem: (key: string) => storage.delete(key),
      clear: () => storage.clear(),
    },
  });
}

function reverseGeometry(points: number[][]): number[][] {
  return [...points].reverse().map((point) => [point[0], point[1]]);
}

function distanceKm(from: number[], to: number[]): number {
  const toRadians = (value: number) => (value * Math.PI) / 180;
  const earthRadiusKm = 6371;
  const deltaLat = toRadians(to[0] - from[0]);
  const deltaLng = toRadians(to[1] - from[1]);
  const startLat = toRadians(from[0]);
  const endLat = toRadians(to[0]);
  const haversine = Math.sin(deltaLat / 2) ** 2
    + Math.cos(startLat) * Math.cos(endLat) * Math.sin(deltaLng / 2) ** 2;
  return 2 * earthRadiusKm * Math.atan2(Math.sqrt(haversine), Math.sqrt(1 - haversine));
}

function boundingCenter(points: number[][]): number[] {
  const latitudes = points.map((point) => point[0]);
  const longitudes = points.map((point) => point[1]);
  return [
    (Math.min(...latitudes) + Math.max(...latitudes)) / 2,
    (Math.min(...longitudes) + Math.max(...longitudes)) / 2,
  ];
}

function radiusKm(points: number[][]): number {
  const center = boundingCenter(points);
  return Math.max(...points.map((point) => distanceKm(center, point)));
}

function buildRoute(overrides: Partial<GeneratedRoute> = {}): GeneratedRoute {
  return {
    routeId: "route-default",
    title: "Generated route",
    variantType: "SHAPE",
    routeType: "RIDE",
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

  afterEach(() => {
    Reflect.deleteProperty(globalThis, "localStorage");
  });

  it("imports GPX points into shape mode", () => {
    const store = useRoutesStore();
    const gpx = `
      <gpx version="1.1" creator="test">
        <trk><trkseg>
          <trkpt lat="48.1000" lon="-1.6000"></trkpt>
          <trkpt lat="48.1100" lon="-1.6100"></trkpt>
          <trkpt lat="48.1200" lon="-1.6200"></trkpt>
        </trkseg></trk>
      </gpx>
    `;

    const importedPoints = store.importShapeFromGpx(gpx);

    expect(importedPoints).toBe(3);
    expect(store.shapeInputType).toBe("gpx");
    expect(store.shapePoints).toEqual([
      [48.1, -1.6],
      [48.11, -1.61],
      [48.12, -1.62],
    ]);
    expect(store.shapeDataText).toBe(JSON.stringify(store.shapePoints));
  });

  it("appends GPX points without duplicating seam point", () => {
    const store = useRoutesStore();
    store.shapePoints = [
      [48.1, -1.6],
      [48.11, -1.61],
    ];
    store.shapeDataText = JSON.stringify(store.shapePoints);

    const gpx = `
      <gpx version="1.1" creator="test">
        <trk><trkseg>
          <trkpt lat="48.1100" lon="-1.6100"></trkpt>
          <trkpt lat="48.1200" lon="-1.6200"></trkpt>
          <trkpt lat="48.1300" lon="-1.6300"></trkpt>
        </trkseg></trk>
      </gpx>
    `;

    const importedPoints = store.importShapeFromGpx(gpx, { append: true });

    expect(importedPoints).toBe(3);
    expect(store.shapePoints).toEqual([
      [48.1, -1.6],
      [48.11, -1.61],
      [48.12, -1.62],
      [48.13, -1.63],
    ]);
    expect(store.shapeDataText).toBe(JSON.stringify(store.shapePoints));
  });

  it("transforms a sketch and supports transform undo/redo", () => {
    const store = useRoutesStore();
    store.shapePoints = [
      [1, 1],
      [1, 3],
      [3, 3],
      [3, 1],
    ];
    store.shapeDataText = JSON.stringify(store.shapePoints);
    store.startPoint = { lat: 10, lng: 20 };

    expect(store.centerShapeOnStart()).toBe(true);

    expect(store.shapePoints).toEqual([
      [9, 19],
      [9, 21],
      [11, 21],
      [11, 19],
    ]);
    expect(store.shapeInputType).toBe("draw");
    expect(store.shapeDataText).toBe(JSON.stringify(store.shapePoints));
    expect(store.canUndoShapeTransform).toBe(true);

    expect(store.undoShapeTransform()).toBe(true);
    expect(store.shapePoints).toEqual([
      [1, 1],
      [1, 3],
      [3, 3],
      [3, 1],
    ]);
    expect(store.canRedoShapeTransform).toBe(true);

    expect(store.redoShapeTransform()).toBe(true);
    expect(store.shapePoints).toEqual([
      [9, 19],
      [9, 21],
      [11, 21],
      [11, 19],
    ]);
  });

  it("smooths and simplifies a sketch while preserving undo history", () => {
    const store = useRoutesStore();
    store.shapePoints = [
      [10, 10],
      [10.00005, 10.00002],
      [10.0001, 10.00001],
      [10.01, 10.01],
    ];
    store.shapeDataText = JSON.stringify(store.shapePoints);

    expect(store.smoothShape()).toBe(true);
    expect(store.shapePoints[0]).toEqual([10, 10]);
    expect(store.shapePoints[store.shapePoints.length - 1]).toEqual([10.01, 10.01]);

    expect(store.simplifyShape(0.001)).toBe(true);
    expect(store.shapePoints).toEqual([
      [10, 10],
      [10.01, 10.01],
    ]);

    expect(store.undoShapeTransform()).toBe(true);
    expect(store.shapePoints.length).toBe(4);
  });

  it("auto-fits a sketch around the start point before routing", () => {
    const store = useRoutesStore();
    store.startPoint = { lat: 48, lng: -1.6 };
    store.shapePoints = [
      [47.8, -1.9],
      [47.8, -1.7],
      [48.0, -1.7],
      [48.0, -1.9],
    ];
    store.shapeDataText = JSON.stringify(store.shapePoints);
    const originalRadiusKm = radiusKm(store.shapePoints);

    expect(store.fitShapeToStart({ targetRadiusKm: 1.0 })).toBe(true);

    const fittedCenter = boundingCenter(store.shapePoints);
    expect(fittedCenter[0]).toBeCloseTo(48, 6);
    expect(fittedCenter[1]).toBeCloseTo(-1.6, 6);
    expect(radiusKm(store.shapePoints)).toBeCloseTo(1.0, 1);
    expect(radiusKm(store.shapePoints)).toBeLessThan(originalRadiusKm);
    expect(store.shapeInputType).toBe("draw");
    expect(store.shapeDataText).toBe(JSON.stringify(store.shapePoints));
    expect(store.canUndoShapeTransform).toBe(true);
  });

  it("loads built-in shape templates and exports freestyle GPX", () => {
    const store = useRoutesStore();

    expect(store.applyBuiltInShapeTemplate("heart", { lat: 48, lng: -1.6 })).toBe(true);

    expect(store.shapeInputType).toBe("draw");
    expect(store.shapePoints.length).toBeGreaterThan(20);
    expect(store.shapeDataText).toBe(JSON.stringify(store.shapePoints));

    const gpx = store.buildCurrentShapeGpx("Heart & star");
    expect(gpx).toContain("<gpx");
    expect(gpx).toContain("Heart &amp; star");
    expect(gpx).toContain("<trkpt");

    const tcx = store.buildCurrentShapeTcx("Heart & star");
    expect(tcx).toContain("<TrainingCenterDatabase");
    expect(tcx).toContain("Heart &amp; star");
    expect(tcx).toContain("<Trackpoint>");
  });

  it("loads every simple built-in shape template", () => {
    const store = useRoutesStore();
    const templates: BuiltInShapeTemplateKey[] = [
      "heart",
      "star",
      "circle",
      "square",
      "triangle",
      "diamond",
      "rectangle",
      "hexagon",
    ];

    templates.forEach((template) => {
      store.clearShape();
      expect(store.applyBuiltInShapeTemplate(template, { lat: 48, lng: -1.6 })).toBe(true);
      expect(store.shapeInputType).toBe("draw");
      expect(store.shapePoints.length).toBeGreaterThanOrEqual(4);
      expect(store.shapeDataText).toBe(JSON.stringify(store.shapePoints));
    });
  });

  it("saves, loads, and deletes local shape templates", () => {
    installMemoryStorage();
    const store = useRoutesStore();
    store.shapePoints = [
      [48.1, -1.6],
      [48.11, -1.61],
      [48.12, -1.62],
    ];
    store.shapeDataText = JSON.stringify(store.shapePoints);

    const saved = store.saveCurrentShapeTemplate("Local sketch");

    expect(saved?.name).toBe("Local sketch");
    expect(store.savedShapeTemplateCount).toBe(1);

    const nextStore = useRoutesStore();
    nextStore.loadSavedShapeTemplates();
    expect(nextStore.savedShapeTemplates).toHaveLength(1);

    nextStore.clearShape();
    expect(nextStore.loadSavedShapeTemplate(saved?.id ?? "")).toBe(true);
    expect(nextStore.shapePoints).toEqual([
      [48.1, -1.6],
      [48.11, -1.61],
      [48.12, -1.62],
    ]);

    expect(nextStore.deleteSavedShapeTemplate(saved?.id ?? "")).toBe(true);
    expect(nextStore.savedShapeTemplates).toHaveLength(0);
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

  it("does not constrain Strava Art shape generation with distance or elevation defaults", async () => {
    const store = useRoutesStore();
    store.mode = "SHAPE";
    store.shapePoints = [
      [45.0, 6.0],
      [45.1, 6.1],
      [45.2, 6.0],
    ];

    vi.mocked(requestJson).mockResolvedValueOnce({
      routes: [],
      diagnostics: [],
    });

    await store.generateRoutes();

    const requestOptions = vi.mocked(requestJson).mock.calls[0]?.[1];
    const payload = JSON.parse(String(requestOptions?.body ?? "{}")) as Record<string, unknown>;
    expect(payload).not.toHaveProperty("distanceTargetKm");
    expect(payload).not.toHaveProperty("elevationTargetM");
  });
});
