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
    type: "Ride",
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
  it("uses complete real-time windows and retains measured zeros", () => {
    expect(bestAveragePower([900, null, 200, 200, 200, 100], 2, [0,1,2,3,4,5])).toBe(200);
    expect(bestAveragePower([0, 100, 200], 2, [0,1,2])).toBe(50);
    expect(bestAveragePower([null, 100, 200], 2, [0,1,2])).toBeNull();
    expect(buildPowerCurve([null, null], [0,1])).toEqual([]);
    expect(buildPowerZoneEstimate([null, 0, 200], 200, [0,1,2])?.trackedSeconds).toBe(1);
    expect(buildPowerZoneEstimate([null], 200, [0])).toBeNull();
    expect(sanitizePowerSamples([200, Number.NaN, -10, 300])).toEqual([200, Number.NaN, Number.NaN, 300]);
    expect(rollingAverage([100, 200, 300, 400], 2)).toEqual([150, 250, 350]);
    expect(buildPowerCurve([170, 190, 205], [0,1,2])).toEqual([[1,190],[2,180]]);
  });

  it("requires 30 real covered seconds for normalized power", () => {
    expect(normalizedPowerFromWatts(Array(31).fill(250), Array.from({length:31},(_,i)=>i))).toBeCloseTo(250);
    expect(normalizedPowerFromWatts(Array(30).fill(250), Array.from({length:30},(_,i)=>i))).toBeNull();
    expect(normalizedPowerFromWatts([250,250], [0,30])).toBeNull();
  });

  it("uses manual FTP and weight settings ahead of athlete profile values", () => {
    const analysis = buildPowerAnalysis(
      activity({
        stream: {
          ...activity().stream,
          watts: Array(361).fill(200),
          time: Array.from({length:361},(_,i)=>i*10),
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

  it("does not assign historical load from today's FTP or an incomplete activity", () => {
    const ride = activity({elapsedTime: 60, stream: {...activity().stream, watts: Array(61).fill(200), time: Array.from({length:61},(_,i)=>i)}});
    expect(buildPowerAnalysis(ride, 200, 70, {ftpHistory:[]}).trainingStressScore).toBeNull();
    const settings={ftpHistory:[{effectiveFrom:"2026-01-01",ftp:200}]};
    expect(buildPowerAnalysis(ride,200,70,settings).trainingStressScore).toBeCloseTo(100/60);
    expect(buildPowerAnalysis({...ride,elapsedTime:100},200,70,settings).trainingStressScore).toBeNull();
    expect(buildPowerAnalysis({...ride,type:"Run"},200,70,settings).trainingStressScore).toBeNull();
  });

  it("falls back from profile FTP to power-based estimates", () => {
    expect(resolveFtpDetails(null, 275, 260, 280)).toMatchObject({ ftp: 275, sourceKind: "strava" });
    expect(resolveFtpDetails(null, 0, 260, 280)).toMatchObject({ ftp: 260, sourceKind: "estimated" });
    expect(resolveFtpDetails(null, 0, null, 280)).toMatchObject({ ftp: 266, sourceKind: "estimated" });
    expect(resolveFtpDetails(null, 0, null, null).ftp).toBeNull();
  });

  it("classifies every tracked second into a power zone", () => {
    expect(buildPowerZoneEstimate([0, 180, 200, 250, 300, 300], 200, [0,1,2,3,4,5])).toEqual({
      trackedSeconds: 5,
      aerobicSeconds: 2,
      thresholdVo2Seconds: 1,
      anaerobicSeconds: 2,
    });
    expect(buildPowerZoneEstimate([], 200, [])).toBeNull();
  });
});
