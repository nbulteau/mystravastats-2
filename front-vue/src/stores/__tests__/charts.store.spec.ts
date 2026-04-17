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
    vi.mocked(requestJson)
      .mockResolvedValueOnce({
        nbActivitiesByYear: { "2025": 120 },
        totalDistanceByYear: { "2025": 4521 },
        totalElevationByYear: { "2025": 51000 },
        averageSpeedByYear: { "2025": 26.4 },
        maxSpeedByYear: { "2025": 73.5 },
      })
      .mockResolvedValueOnce([
        {
          id: 1,
          name: "Ride 1",
          type: "Ride",
          link: "",
          distance: 42000,
          elapsedTime: 3600,
          movingTime: 3500,
          totalElevationGain: 420,
          totalDescent: 410,
          averageSpeed: 7,
          bestSpeedForDistanceFor1000m: 8,
          bestElevationForDistanceFor500m: 5,
          bestElevationForDistanceFor1000m: 4,
          date: "2025-01-01",
          averageWatts: 210,
          weightedAverageWatts: 230,
          bestPowerFor20minutes: 280,
          bestPowerFor60minutes: 250,
          ftp: 240,
        },
      ])
      .mockResolvedValueOnce({
        settings: {
          maxHr: null,
          thresholdHr: null,
          reserveHr: null,
        },
        hasHeartRateData: false,
        totalTrackedSeconds: 0,
        easyHardRatio: null,
        zones: [],
        activities: [],
        byMonth: [],
        byYear: [],
      });

    // WHEN
    await chartsStore.ensureLoaded(true);

    // THEN
    expect(buildFilteredApiUrl).toHaveBeenCalledWith("dashboard", "Ride", "All years");
    expect(requestJson).toHaveBeenCalledTimes(3);
    expect(chartsStore.totalDistanceByYear["2025"]).toBe(4521);
    expect(chartsStore.distanceByWeeks).toEqual([]);
    expect(chartsStore.activitiesForCharts).toHaveLength(1);
    expect(chartsStore.error).toBeNull();
  });

  it("loads period charts when a specific year is selected", async () => {
    // GIVEN
    const contextStore = useContextStore();
    contextStore.currentYear = "2025";
    contextStore.currentActivityType = "Ride";
    const chartsStore = useChartsStore();
    vi.mocked(requestJson)
      .mockResolvedValueOnce([{ periodKey: "01", value: 100, activityCount: 2 }])
      .mockResolvedValueOnce([{ periodKey: "01", value: 200, activityCount: 2 }])
      .mockResolvedValueOnce([{ periodKey: "01", value: 30, activityCount: 2 }])
      .mockResolvedValueOnce([{ periodKey: "01", value: 45, activityCount: 3 }])
      .mockResolvedValueOnce([{ periodKey: "01", value: 800, activityCount: 3 }])
      .mockResolvedValueOnce([{ periodKey: "01", value: 88, activityCount: 3 }])
      .mockResolvedValueOnce({
        nbActivitiesByYear: { "2025": 120, "2024": 95 },
        totalDistanceByYear: { "2025": 4521, "2024": 4200 },
        totalElevationByYear: { "2025": 51000, "2024": 47000 },
        averageSpeedByYear: { "2025": 26.4, "2024": 25.8 },
        maxSpeedByYear: { "2025": 73.5, "2024": 71.2 },
      })
      .mockResolvedValueOnce([
        {
          id: 42,
          name: "Long ride",
          type: "Ride",
          link: "",
          distance: 98000,
          elapsedTime: 10800,
          movingTime: 10300,
          totalElevationGain: 1250,
          totalDescent: 1240,
          averageSpeed: 9.5,
          bestSpeedForDistanceFor1000m: 10.1,
          bestElevationForDistanceFor500m: 6.2,
          bestElevationForDistanceFor1000m: 5.4,
          date: "2025-02-02",
          averageWatts: 220,
          weightedAverageWatts: 245,
          bestPowerFor20minutes: 300,
          bestPowerFor60minutes: 260,
          ftp: 250,
        },
      ])
      .mockResolvedValueOnce({
        settings: {
          maxHr: null,
          thresholdHr: null,
          reserveHr: null,
        },
        hasHeartRateData: true,
        totalTrackedSeconds: 1200,
        easyHardRatio: 2.1,
        zones: [],
        activities: [],
        byMonth: [],
        byYear: [],
      });

    // WHEN
    await chartsStore.ensureLoaded(true);

    // THEN
    expect(requestJson).toHaveBeenCalledTimes(9);
    expect(chartsStore.distanceByMonths).toEqual([{ periodKey: "01", value: 100, activityCount: 2 }]);
    expect(chartsStore.activitiesCountByYear).toEqual({ "2025": 120, "2024": 95 });
    expect(chartsStore.totalDistanceByYear["2025"]).toBe(4521);
    expect(chartsStore.activitiesForCharts).toHaveLength(1);
    expect(chartsStore.heartRateZoneAnalysis.hasHeartRateData).toBe(true);
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
