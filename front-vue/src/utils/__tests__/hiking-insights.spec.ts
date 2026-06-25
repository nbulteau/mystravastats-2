import { describe, expect, it } from "vitest";
import {
  buildHikingInsights,
  computeMaxContinuousClimb,
  isHikingActivityType,
} from "@/utils/hiking-insights";

describe("hiking insights", () => {
  it("only applies to hike and walk activities", () => {
    expect(isHikingActivityType("Hike")).toBe(true);
    expect(isHikingActivityType("Walk")).toBe(true);
    expect(isHikingActivityType("Ride")).toBe(false);
    expect(buildHikingInsights({
      type: "Ride",
      distance: 10000,
      elapsedTime: 3600,
      movingTime: 3600,
      totalElevationGain: 300,
    })).toBeNull();
  });

  it("computes hiking metrics from activity and altitude stream data", () => {
    const insights = buildHikingInsights({
      type: "Hike",
      sportType: "Hike",
      distance: 10000,
      elapsedTime: 9000,
      movingTime: 7200,
      totalElevationGain: 500,
      elevHigh: 140,
      stream: {
        altitude: [100, 110, 130, 125, 150, 145, 170, 160, 165],
      },
    });

    expect(insights).not.toBeNull();
    expect(insights?.difficultyLabel).toBe("Moderate");
    expect(insights?.elevationPerKm).toBe(50);
    expect(insights?.maxContinuousClimbMeters).toBe(80);
    expect(insights?.highestPointMeters).toBe(170);
    expect(insights?.verticalSpeedMetersPerHour).toBe(250);
    expect(insights?.pauseRatio).toBe(0.2);
    expect(insights?.pausedSeconds).toBe(1800);
  });

  it("breaks a continuous climb after a meaningful descent", () => {
    expect(computeMaxContinuousClimb([100, 140, 132, 150, 170])).toBe(40);
    expect(computeMaxContinuousClimb([100, 140, 135, 150, 170])).toBe(75);
  });

  it("falls back to activity high point when altitude stream is missing", () => {
    const insights = buildHikingInsights({
      type: "Walk",
      distance: 5000,
      elapsedTime: 4000,
      movingTime: 3600,
      totalElevationGain: 100,
      elevHigh: 812,
    });

    expect(insights?.highestPointMeters).toBe(812);
    expect(insights?.maxContinuousClimbMeters).toBeNull();
  });
});
