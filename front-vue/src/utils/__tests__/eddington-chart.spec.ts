import { describe, expect, it } from "vitest";
import {
  EDDINGTON_CURRENT_COLOR,
  EDDINGTON_PENDING_COLOR,
  EDDINGTON_SATISFIED_COLOR,
  buildEddingtonChartData,
  formatEddingtonTooltip,
} from "@/utils/eddington-chart";

describe("eddington chart helpers", () => {
  it("builds threshold points and highlights the current Eddington number", () => {
    const data = buildEddingtonChartData({
      eddingtonNumber: 3,
      eddingtonList: [5, 4, 3, 2, 1],
    });

    expect(data.currentNumber).toBe(3);
    expect(data.points).toHaveLength(5);
    expect(data.points[0].color).toBe(EDDINGTON_SATISFIED_COLOR);
    expect(data.points[2]).toMatchObject({
      x: 3,
      y: 3,
      color: EDDINGTON_CURRENT_COLOR,
      custom: {
        isCurrent: true,
        isSatisfied: true,
        missingCount: 0,
      },
    });
    expect(data.points[3]).toMatchObject({
      x: 4,
      y: 2,
      color: EDDINGTON_PENDING_COLOR,
      custom: {
        isSatisfied: false,
        missingCount: 2,
      },
    });
  });

  it("computes the next target from the current number", () => {
    const data = buildEddingtonChartData({
      eddingtonNumber: 3,
      eddingtonList: [5, 4, 3, 2, 1],
    });

    expect(data.nextTarget).toBe(4);
    expect(data.nextTargetCurrentCount).toBe(2);
    expect(data.qualifyingCount).toBe(2);
    expect(data.nextTargetMissingCount).toBe(2);
    expect(data.thresholdScale).toBe(1);
    expect(data.summary).toBe("E=3 - 2/4 days at 4+ km; 2 more days needed.");
    expect(data.referenceLine).toEqual([[0, 0], [5, 5]]);
  });

  it("uses backend next target progress when available", () => {
    const data = buildEddingtonChartData({
      eddingtonNumber: 3,
      eddingtonList: [5, 4, 3, 2, 1],
      nextTarget: 4,
      qualifyingCount: 3,
      missingCount: 1,
    });

    expect(data.nextTarget).toBe(4);
    expect(data.qualifyingCount).toBe(3);
    expect(data.nextTargetMissingCount).toBe(1);
    expect(data.summary).toBe("E=3 - 3/4 days at 4+ km; 1 more day needed.");
  });

  it("formats elevation by activities", () => {
    const data = buildEddingtonChartData({
      eddingtonNumber: 3,
      eddingtonList: [4, 4, 3, 2],
      metric: "elevation",
      basis: "activities",
      unit: "m",
      thresholdScale: 100,
      nextTarget: 4,
      qualifyingCount: 2,
      missingCount: 2,
    });

    expect(data.unit).toBe("m");
    expect(data.thresholdScale).toBe(100);
    expect(data.countSingular).toBe("activity");
    expect(data.metricLabel).toBe("elevation");
    expect(data.summary).toBe("E=3 - 2/4 activities at 400+ m; 2 more activities needed.");
    expect(formatEddingtonTooltip(data.points[3])).toBe(
      "<b>400 m threshold</b><br/>2 activities at 400+ m<br/>2 more activities needed",
    );
  });

  it("handles empty data without chart points", () => {
    const data = buildEddingtonChartData({
      eddingtonNumber: 0,
      eddingtonList: [],
    });

    expect(data.hasData).toBe(false);
    expect(data.points).toEqual([]);
    expect(data.nextTarget).toBe(1);
    expect(data.nextTargetMissingCount).toBe(1);
    expect(data.summary).toBe("No distance-qualified days yet.");
  });

  it("focuses the visible range around the current Eddington number", () => {
    const eddingtonList = Array.from({ length: 1_600 }, (_, index) => Math.max(1, 1_600 - index));
    const data = buildEddingtonChartData({
      eddingtonNumber: 55,
      eddingtonList,
    });

    expect(data.maxThreshold).toBe(1_600);
    expect(data.axisMin).toBeGreaterThan(0);
    expect(data.axisMax).toBeLessThan(100);
    expect(data.points[0].x).toBe(data.axisMin + 1);
    expect(data.points.at(-1)?.x).toBe(data.axisMax);
  });

  it("formats a readable tooltip for a threshold", () => {
    const data = buildEddingtonChartData({
      eddingtonNumber: 3,
      eddingtonList: [5, 4, 3, 2, 1],
    });

    expect(formatEddingtonTooltip(data.points[2])).toBe(
      "<b>3 km threshold</b><br/>3 days at 3+ km<br/>Threshold reached<br/><b>Current Eddington number</b>",
    );
    expect(formatEddingtonTooltip(data.points[3])).toBe(
      "<b>4 km threshold</b><br/>2 days at 4+ km<br/>2 more days needed",
    );
  });
});
