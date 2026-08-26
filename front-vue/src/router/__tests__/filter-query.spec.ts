import { describe, expect, it } from "vitest";
import { filtersFromQuery, filtersToQuery } from "@/router/filter-query";

const fallback = { year: "2026", activityType: "Ride_VirtualRide" };

describe("route filter query", () => {
  it("restores valid filters and normalizes activity order", () => {
    expect(filtersFromQuery({ year: "2024", activityType: "VirtualRide_Ride" }, fallback)).toEqual({
      year: "2024",
      activityType: "Ride_VirtualRide",
    });
  });

  it("uses explicit all-years and rejects unsupported values", () => {
    expect(filtersFromQuery({ year: "all", activityType: "Unknown" }, fallback)).toEqual({
      year: "All years",
      activityType: fallback.activityType,
    });
  });

  it("preserves unrelated query values when writing filters", () => {
    expect(filtersToQuery({ year: "All years", activityType: "Run" }, { panel: "records" })).toEqual({
      panel: "records",
      year: "all",
      activityType: "Run",
    });
  });
});
