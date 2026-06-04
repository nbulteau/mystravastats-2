import { describe, expect, it } from "vitest";
import {
  buildCumulativeYearChartData,
  formatCumulativeDateLabel,
  type CumulativeChartInput,
} from "@/utils/cumulative-year-chart";

function yearData(values: Record<string, number>): Map<string, number> {
  return new Map(Object.entries(values));
}

function input(overrides: Partial<CumulativeChartInput> = {}): CumulativeChartInput {
  return {
    comparisonMode: "all-years",
    distancePerYear: new Map([
      ["2024", yearData({ "01-01": 10, "06-04": 100, "12-31": 500 })],
      ["2025", yearData({ "01-01": 20, "06-04": 120, "12-31": 600 })],
      ["2026", yearData({ "01-01": 30, "06-04": 200, "12-31": 200 })],
    ]),
    elevationPerYear: new Map([
      ["2024", yearData({ "01-01": 100, "06-04": 1_000, "12-31": 5_000 })],
      ["2025", yearData({ "01-01": 200, "06-04": 1_200, "12-31": 6_000 })],
      ["2026", yearData({ "01-01": 300, "06-04": 2_000, "12-31": 2_000 })],
    ]),
    metric: "distance",
    now: new Date("2026-06-04T10:00:00"),
    ...overrides,
  };
}

describe("cumulative year chart helpers", () => {
  it("builds all historical years, current year, and projected current year", () => {
    // WHEN
    const chartData = buildCumulativeYearChartData(input());

    // THEN
    expect(chartData.title).toBe("Cumulative distance per year");
    expect(chartData.unit).toBe("km");
    expect(chartData.comparisonYear).toBe("2025");
    expect(chartData.comparisonValue).toBe(120);
    expect(chartData.currentValue).toBe(200);
    expect(chartData.projectedValue).toBeCloseTo(470.97, 2);
    expect(chartData.summary).toBe("Today: 200 km - +80 km vs 2025 - projected: 471 km");
    expect(chartData.series.map((series) => series.name)).toEqual([
      "2024",
      "2025",
      "2026",
      "2026 projected",
    ]);

    const currentSeries = chartData.series.find((series) => series.role === "current");
    const projectionSeries = chartData.series.find((series) => series.role === "projection");
    expect(currentSeries?.data[chartData.todayIndex]).toBe(200);
    expect(currentSeries?.data[chartData.todayIndex + 1]).toBeNull();
    expect(projectionSeries?.data[chartData.todayIndex - 1]).toBeNull();
    expect(projectionSeries?.data[chartData.todayIndex]).toBe(200);
  });

  it("compares the current year with the best historical year", () => {
    // WHEN
    const chartData = buildCumulativeYearChartData(input({
      comparisonMode: "best-year",
      distancePerYear: new Map([
        ["2024", yearData({ "01-01": 10, "06-04": 160, "12-31": 800 })],
        ["2025", yearData({ "01-01": 20, "06-04": 120, "12-31": 600 })],
        ["2026", yearData({ "01-01": 30, "06-04": 200, "12-31": 200 })],
      ]),
    }));

    // THEN
    expect(chartData.comparisonYear).toBe("2024");
    expect(chartData.comparisonValue).toBe(160);
    expect(chartData.summary).toContain("+40 km vs 2024");
    expect(chartData.series.map((series) => series.name)).toEqual([
      "2024",
      "2026",
      "2026 projected",
    ]);
  });

  it("uses elevation units and labels when elevation is selected", () => {
    // WHEN
    const chartData = buildCumulativeYearChartData(input({ metric: "elevation" }));

    // THEN
    expect(chartData.title).toBe("Cumulative elevation per year");
    expect(chartData.unit).toBe("m");
    expect(chartData.yAxisTitle).toBe("Elevation (m)");
    expect(chartData.summary).toBe("Today: 2,000 m - +800 m vs 2025 - projected: 4,710 m");
  });

  it("keeps historical data visible when there is no current-year data", () => {
    // WHEN
    const chartData = buildCumulativeYearChartData(input({
      distancePerYear: new Map([
        ["2024", yearData({ "01-01": 10, "06-04": 160, "12-31": 800 })],
        ["2025", yearData({ "01-01": 20, "06-04": 120, "12-31": 600 })],
      ]),
    }));

    // THEN
    expect(chartData.hasData).toBe(true);
    expect(chartData.currentValue).toBeNull();
    expect(chartData.projectedValue).toBeNull();
    expect(chartData.summary).toBe("No 2026 cumulative data yet.");
    expect(chartData.series.map((series) => series.name)).toEqual(["2024", "2025"]);
  });

  it("formats month-day labels for tooltips", () => {
    expect(formatCumulativeDateLabel("06-04")).toBe("4 June");
  });
});
