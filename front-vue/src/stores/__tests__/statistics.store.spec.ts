import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useContextStore } from "@/stores/context";
import { useStatisticsStore } from "@/stores/statistics";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";
import type { Statistics } from "@/models/statistics.model";

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
  let reject!: (reason?: unknown) => void;
  const promise = new Promise<T>((promiseResolve, promiseReject) => {
    resolve = promiseResolve;
    reject = promiseReject;
  });
  return { promise, resolve, reject };
}

describe("statistics store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("does not display a stale statistics response after the activity filter changes", async () => {
    // GIVEN
    const contextStore = useContextStore();
    contextStore.currentYear = "2026";
    contextStore.currentActivityType = "Hike_Walk";
    const statisticsStore = useStatisticsStore();

    const staleHikeResponse = deferred<Statistics[]>();
    const currentRideResponse = deferred<Statistics[]>();
    vi.mocked(requestJson)
      .mockReturnValueOnce(staleHikeResponse.promise)
      .mockReturnValueOnce(currentRideResponse.promise);

    // WHEN
    const staleFetch = statisticsStore.fetchStatistics();
    contextStore.currentActivityType = "Commute_GravelRide_MountainBikeRide_Ride_VirtualRide";
    const currentFetch = statisticsStore.fetchStatistics();

    staleHikeResponse.resolve([{ label: "Nb activities", value: "4" }]);
    await staleFetch;

    // THEN
    expect(statisticsStore.statistics).toEqual([]);
    expect(statisticsStore.isStatisticsLoading).toBe(true);

    // WHEN
    currentRideResponse.resolve([{ label: "Nb activities", value: "12" }]);
    await currentFetch;

    // THEN
    expect(buildFilteredApiUrl).toHaveBeenNthCalledWith(1, "statistics", "Hike_Walk", "2026");
    expect(buildFilteredApiUrl).toHaveBeenNthCalledWith(
      2,
      "statistics",
      "Commute_GravelRide_MountainBikeRide_Ride_VirtualRide",
      "2026",
    );
    expect(statisticsStore.statistics).toEqual([{ label: "Nb activities", value: "12" }]);
    expect(statisticsStore.isStatisticsLoading).toBe(false);
  });
});
