import { beforeEach, describe, expect, it, vi } from "vitest";
import { createPinia, setActivePinia } from "pinia";
import { useContextStore } from "@/stores/context";
import { useChartsStore } from "@/stores/charts";
import { buildFilteredApiUrl, requestJson } from "@/stores/api";

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

describe("charts store", () => {
  beforeEach(() => {
    setActivePinia(createPinia());
    vi.clearAllMocks();
  });

  it("loads all-years overview data when year filter is All years", async () => {
    // GIVEN
    const contextStore = useContextStore();
    contextStore.currentYear = "All years";
    contextStore.currentActivityType = "Ride";
    const chartsStore = useChartsStore();
    vi.mocked(requestJson).mockResolvedValueOnce({
      nbActivitiesByYear: { "2025": 120 },
      totalDistanceByYear: { "2025": 4521 },
      totalElevationByYear: { "2025": 51000 },
      averageSpeedByYear: { "2025": 26.4 },
      maxSpeedByYear: { "2025": 73.5 },
    });

    // WHEN
    await chartsStore.ensureLoaded(true);

    // THEN
    expect(buildFilteredApiUrl).toHaveBeenCalledWith("dashboard", "Ride", "All years");
    expect(requestJson).toHaveBeenCalledTimes(1);
    expect(chartsStore.totalDistanceByYear["2025"]).toBe(4521);
    expect(chartsStore.distanceByWeeks).toEqual([]);
    expect(chartsStore.error).toBeNull();
  });

  it("loads period charts when a specific year is selected", async () => {
    // GIVEN
    const contextStore = useContextStore();
    contextStore.currentYear = "2025";
    contextStore.currentActivityType = "Ride";
    const chartsStore = useChartsStore();
    vi.mocked(requestJson)
      .mockResolvedValueOnce([{ "01": 100 }])
      .mockResolvedValueOnce([{ "01": 200 }])
      .mockResolvedValueOnce([{ "01": 30 }])
      .mockResolvedValueOnce([{ "01": 45 }])
      .mockResolvedValueOnce([{ "01": 800 }])
      .mockResolvedValueOnce([{ "01": 88 }]);

    // WHEN
    await chartsStore.ensureLoaded(true);

    // THEN
    expect(requestJson).toHaveBeenCalledTimes(6);
    expect(chartsStore.distanceByMonths).toEqual([{ "01": 100 }]);
    expect(chartsStore.activitiesCountByYear).toEqual({});
    expect(chartsStore.isLoading).toBe(false);
    expect(chartsStore.error).toBeNull();
  });

  it("sets a readable error state when chart loading fails", async () => {
    // GIVEN
    const contextStore = useContextStore();
    contextStore.currentYear = "2025";
    contextStore.currentActivityType = "Ride";
    const chartsStore = useChartsStore();
    vi.mocked(requestJson).mockRejectedValue(new Error("boom"));

    // WHEN
    await chartsStore.ensureLoaded(true);

    // THEN
    expect(chartsStore.error).toBe("Unable to load chart data for the selected filters.");
    expect(chartsStore.isLoading).toBe(false);
  });
});
