import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useActivitiesStore } from "@/stores/activities";
import { useContextStore } from "@/stores/context";
import { useDashboardStore } from "@/stores/dashboard";
import { useGearAnalysisStore } from "@/stores/gear-analysis";
import { useMapStore } from "@/stores/map";
import { useSegmentsStore } from "@/stores/segments";
import { requestJson } from "@/stores/api";
import { emptyGearAnalysis, type GearAnalysis } from "@/models/gear-analysis.model";
import type { Activity } from "@/models/activity.model";
import type { DashboardData } from "@/models/dashboard-data.model";
import type { MapTrack } from "@/models/map.model";
import type { SegmentTargetSummary } from "@/models/segment-analysis.model";

vi.mock("@/stores/api", () => ({
  buildFilteredApiUrl: vi.fn((path: string, activityType: string, year: string) => {
    const params = new URLSearchParams({ activityType });
    if (year !== "All years") {
      params.set("year", year);
    }
    return `/api/${path}?${params.toString()}`;
  }),
  requestJson: vi.fn(),
}));

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((promiseResolve) => {
    resolve = promiseResolve;
  });
  return { promise, resolve };
}

function setFilters(activityType: string, year = "2026") {
  const contextStore = useContextStore();
  contextStore.currentActivityType = activityType;
  contextStore.currentYear = year;
}

describe("filtered store request races", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("keeps a stale activities response out of the current view and cache key", async () => {
    setFilters("Hike");
    const store = useActivitiesStore();
    const stale = deferred<Activity[]>();
    const current = deferred<Activity[]>();
    vi.mocked(requestJson).mockReturnValueOnce(stale.promise).mockReturnValueOnce(current.promise);

    const staleFetch = store.fetchActivities();
    setFilters("Ride");
    const currentFetch = store.fetchActivities();

    stale.resolve([{ id: 1, name: "Old hike" } as Activity]);
    await staleFetch;
    expect(store.activities).toEqual([]);
    expect(store.activitiesByKey.Hike__2026).toHaveLength(1);
    expect(store.activitiesByKey.Ride__2026).toBeUndefined();

    current.resolve([{ id: 2, name: "Current ride" } as Activity]);
    await currentFetch;
    expect(store.activities.map((activity) => activity.id)).toEqual([2]);
    expect(store.activitiesByKey.Ride__2026.map((activity) => activity.id)).toEqual([2]);
  });

  it("does not replace current dashboard data with a stale response", async () => {
    setFilters("Hike");
    const store = useDashboardStore();
    const stale = deferred<DashboardData>();
    const current = deferred<DashboardData>();
    vi.mocked(requestJson).mockReturnValueOnce(stale.promise).mockReturnValueOnce(current.promise);

    const staleFetch = store.fetchDashboardData();
    setFilters("Ride");
    const currentFetch = store.fetchDashboardData();

    stale.resolve({ nbActivitiesByYear: { "2026": 3 } } as DashboardData);
    await staleFetch;
    expect(store.dashboardData.nbActivitiesByYear).toEqual({});

    current.resolve({ nbActivitiesByYear: { "2026": 12 } } as DashboardData);
    await currentFetch;
    expect(store.dashboardData.nbActivitiesByYear).toEqual({ "2026": 12 });
  });

  it("keeps map and equipment responses scoped to their original filters", async () => {
    setFilters("Hike");
    const mapStore = useMapStore();
    const gearStore = useGearAnalysisStore();
    const staleMap = deferred<MapTrack[]>();
    const currentMap = deferred<MapTrack[]>();
    const staleGear = deferred<GearAnalysis>();
    const currentGear = deferred<GearAnalysis>();
    vi.mocked(requestJson)
      .mockReturnValueOnce(staleMap.promise)
      .mockReturnValueOnce(currentMap.promise)
      .mockReturnValueOnce(staleGear.promise)
      .mockReturnValueOnce(currentGear.promise);

    const staleMapFetch = mapStore.fetchGPXCoordinates();
    setFilters("Ride");
    const currentMapFetch = mapStore.fetchGPXCoordinates();
    staleMap.resolve([{ activityId: 1 } as MapTrack]);
    currentMap.resolve([{ activityId: 2 } as MapTrack]);
    await Promise.all([staleMapFetch, currentMapFetch]);

    setFilters("Hike");
    const staleGearFetch = gearStore.fetchGearAnalysis();
    setFilters("Ride");
    const currentGearFetch = gearStore.fetchGearAnalysis();
    staleGear.resolve({ ...emptyGearAnalysis(), unassigned: { ...emptyGearAnalysis().unassigned, distance: 10 } });
    currentGear.resolve({ ...emptyGearAnalysis(), unassigned: { ...emptyGearAnalysis().unassigned, distance: 20 } });
    await Promise.all([staleGearFetch, currentGearFetch]);

    expect(mapStore.mapTracks.map((track) => track.activityId)).toEqual([2]);
    expect(mapStore.mapTracksByKey.Hike__2026.map((track) => track.activityId)).toEqual([1]);
    expect(gearStore.analysis.unassigned.distance).toBe(20);
    expect(gearStore.analysisByKey.Hike__2026.unassigned.distance).toBe(10);
  });

  it("does not display a stale segment list", async () => {
    setFilters("Hike");
    const store = useSegmentsStore();
    const stale = deferred<SegmentTargetSummary[]>();
    const current = deferred<SegmentTargetSummary[]>();
    vi.mocked(requestJson).mockReturnValueOnce(stale.promise).mockReturnValueOnce(current.promise);

    const staleFetch = store.fetchSegments();
    setFilters("Ride");
    const currentFetch = store.fetchSegments();
    stale.resolve([{ targetId: 1 } as SegmentTargetSummary]);
    await staleFetch;
    expect(store.segments).toEqual([]);

    current.resolve([{ targetId: 2 } as SegmentTargetSummary]);
    await currentFetch;
    expect(store.segments.map((segment) => segment.targetId)).toEqual([2]);
  });
});
