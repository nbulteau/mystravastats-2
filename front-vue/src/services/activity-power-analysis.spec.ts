import { describe, expect, it } from "vitest";
import type { DetailedActivity } from "@/models/activity.model";
import {
  bestAveragePower,
  buildPowerCurve,
  buildPowerAnalysis,
  buildPowerZoneEstimate,
  normalizedPowerFromWatts,
  resolveFtpDetails,
  rollingAverage,
  sanitizePowerSamples,
} from "@/services/activity-power-analysis";

function activity(overrides: Partial<DetailedActivity> = {}): DetailedActivity {
  return {
    averageWatts: 0,
    maxWatts: 0,
    kilojoules: 0,
    elapsedTime: 3600,
    movingTime: 3500,
    startDate: "2026-06-15T08:00:00Z",
    startDateLocal: "2026-06-15T10:00:00+02:00",
    stream: {
      watts: [],
      time: [],
      distance: [],
      heartrate: null,
      cadence: null,
      moving: null,
      altitude: null,
      latlng: null,
    },
    ...overrides,
  } as DetailedActivity;
}

describe("activity power analysis", () => {
  it("sanitizes samples and computes rolling and best averages", () => {
    expect(sanitizePowerSamples([200, Number.NaN, -10, 300])).toEqual([200, 0, 0, 300]);
    expect(rollingAverage([100, 200, 300, 400], 2)).toEqual([150, 250, 350]);
    expect(bestAveragePower([100, 200, 400, 200], 2)).toBe(300);
    expect(bestAveragePower([100], 2)).toBeNull();
    expect(buildPowerCurve([170, 190, 205])).toEqual([
      [1, 205],
      [2, 197.5],
      [3, 565 / 3],
    ]);
  });

  it("computes normalized power for a stable 30-second sample", () => {
    expect(normalizedPowerFromWatts(Array(30).fill(250))).toBeCloseTo(250);
    expect(normalizedPowerFromWatts(Array(29).fill(250))).toBeNull();
  });

  it("uses manual FTP and weight settings ahead of athlete profile values", () => {
    const analysis = buildPowerAnalysis(
      activity({
        stream: {
          ...activity().stream,
          watts: Array(60).fill(200),
          time: [0, 3600],
        },
      }),
      280,
      75,
      { ftpHistory: [{ effectiveFrom: "2026-01-01", ftp: 250 }], weightKg: 70 },
    );

    expect(analysis.averagePower).toBe(200);
    expect(analysis.normalizedPower).toBeCloseTo(200);
    expect(analysis.ftp).toBe(250);
    expect(analysis.ftpSourceKind).toBe("manual");
    expect(analysis.weightKg).toBe(70);
    expect(analysis.intensityFactor).toBeCloseTo(0.8);
    expect(analysis.trainingStressScore).toBeCloseTo(64);
    expect(analysis.workKilojoules).toBe(720);
  });

  it("falls back from profile FTP to power-based estimates", () => {
    expect(resolveFtpDetails(null, 275, 260, 280)).toMatchObject({ ftp: 275, sourceKind: "strava" });
    expect(resolveFtpDetails(null, 0, 260, 280)).toMatchObject({ ftp: 260, sourceKind: "estimated" });
    expect(resolveFtpDetails(null, 0, null, 280)).toMatchObject({ ftp: 266, sourceKind: "estimated" });
    expect(resolveFtpDetails(null, 0, null, null).ftp).toBeNull();
  });

  it("classifies every tracked second into a power zone", () => {
    expect(buildPowerZoneEstimate([0, 180, 200, 250, 300], 200)).toEqual({
      trackedSeconds: 5,
      aerobicSeconds: 2,
      thresholdVo2Seconds: 1,
      anaerobicSeconds: 2,
    });
    expect(buildPowerZoneEstimate([], 200)).toBeNull();
  });
});
