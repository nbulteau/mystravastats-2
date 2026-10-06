import { PowerTimeline } from "./power-timeline";
import type { DetailedActivity } from "@/models/activity.model";
import { resolveManualFtpForDate, type AthletePerformanceSettings, type ResolvedManualFtp } from "@/models/athlete-performance-settings.model";
import { formatTime } from "@/utils/formatters";

export type PowerAnalysis = {
  averagePower: number | null;
  maxPower: number | null;
  best20MinutePower: number | null;
  best60MinutePower: number | null;
  normalizedPower: number | null;
  ftp: number | null;
  ftpSource: string | null;
  ftpSourceKind: "manual" | "strava" | "estimated" | null;
  weightKg: number | null;
  weightSource: string | null;
  intensityFactor: number | null;
  trainingStressScore: number | null;
  workKilojoules: number | null;
  powerZoneEstimate: PowerZoneEstimate | null;
  coveredSeconds: number;
  recordedSeconds: number;
};

export type PowerZoneEstimate = {
  trackedSeconds: number;
  aerobicSeconds: number;
  thresholdVo2Seconds: number;
  anaerobicSeconds: number;
};

export function buildPowerAnalysis(
  currentActivity: DetailedActivity | null,
  athleteFtp: number,
  athleteWeight: number,
  performanceSettings: AthletePerformanceSettings,
): PowerAnalysis {
  if (!currentActivity) {
    return emptyPowerAnalysis();
  }

  const watts = sanitizePowerSamples(currentActivity.stream?.watts ?? []);
  const times = currentActivity.stream?.time ?? [];
  const timeline = new PowerTimeline(watts, times);
  const durationSeconds = timeline.duration;
  const averagePower = timeline.average(times[0]!, times.at(-1)!);
  const validWatts = watts.filter(Number.isFinite);
  const maxPower = validWatts.length ? Math.max(...validWatts) : null;
  const best20MinutePower = timeline.best(20 * 60);
  const best60MinutePower = timeline.best(60 * 60);
  const normalizedPower = timeline.normalized();
  const manualFtp = resolveManualFtpForDate(
    performanceSettings,
    currentActivity.startDateLocal || currentActivity.startDate,
  );
  const ftpDetails = resolveFtpDetails(manualFtp, athleteFtp, best60MinutePower, best20MinutePower);
  const weightDetails = resolveWeightDetails(performanceSettings.weightKg, athleteWeight);
  const powerZoneEstimate = ftpDetails.ftp !== null ? buildPowerZoneEstimate(watts, ftpDetails.ftp, times) : null;
  const intensityFactor =
    normalizedPower !== null && ftpDetails.ftp !== null
      ? normalizedPower / ftpDetails.ftp
      : null;
  const trainingStressScore =
    normalizedPower !== null &&
    intensityFactor !== null &&
    ftpDetails.ftp !== null &&
    manualFtp !== null &&
    ["Ride", "VirtualRide", "MountainBikeRide", "GravelRide", "Commute"].includes(currentActivity.type) &&
    times[0] === 0 && currentActivity.elapsedTime > 0 &&
    Math.abs((times.at(-1) ?? 0) - currentActivity.elapsedTime) <= 1 &&
    durationSeconds > 0
      ? (durationSeconds * normalizedPower * intensityFactor) / (ftpDetails.ftp * 3600) * 100
      : null;
  const workKilojoules = currentActivity.kilojoules > 0
    ? currentActivity.kilojoules
    : averagePower !== null && durationSeconds > 0
      ? (averagePower * durationSeconds) / 1000
      : null;

  return {
    averagePower,
    maxPower,
    best20MinutePower,
    best60MinutePower,
    normalizedPower,
    ftp: ftpDetails.ftp,
    ftpSource: ftpDetails.source,
    ftpSourceKind: ftpDetails.sourceKind,
    weightKg: weightDetails.weightKg,
    weightSource: weightDetails.source,
    intensityFactor,
    trainingStressScore,
    workKilojoules,
    powerZoneEstimate,
    coveredSeconds: timeline.coveredSeconds,
    recordedSeconds: timeline.duration,
  };
}

export function emptyPowerAnalysis(): PowerAnalysis {
  return {
    averagePower: null,
    maxPower: null,
    best20MinutePower: null,
    best60MinutePower: null,
    normalizedPower: null,
    ftp: null,
    ftpSource: null,
    ftpSourceKind: null,
    weightKg: null,
    weightSource: null,
    intensityFactor: null,
    trainingStressScore: null,
    workKilojoules: null,
    powerZoneEstimate: null,
    coveredSeconds: 0,
    recordedSeconds: 0,
  };
}

export function sanitizePowerSamples(watts: Array<number | null>): number[] {
  return watts.map((value) => typeof value === "number" && Number.isFinite(value) && value >= 0 ? value : Number.NaN);
}

export function normalizedPowerFromWatts(watts: Array<number | null>, times: number[]): number | null {
  return new PowerTimeline(watts, times).normalized();
}

export function rollingAverage(values: number[], windowSize: number): number[] {
  if (windowSize <= 0 || values.length < windowSize) {
    return [];
  }

  const result: number[] = [];
  let windowSum = 0;
  for (let index = 0; index < values.length; index += 1) {
    windowSum += values[index] ?? 0;
    if (index >= windowSize) {
      windowSum -= values[index - windowSize] ?? 0;
    }
    if (index >= windowSize - 1) {
      result.push(windowSum / windowSize);
    }
  }
  return result;
}

export function resolveFtpDetails(
  manualFtp: ResolvedManualFtp | null,
  athleteFtp: number,
  best60MinutePower: number | null,
  best20MinutePower: number | null,
): { ftp: number | null; source: string | null; sourceKind: "manual" | "strava" | "estimated" | null } {
  if (manualFtp !== null && manualFtp.ftp > 0) {
    return {
      ftp: manualFtp.ftp,
      source: `Manual setting since ${manualFtp.effectiveFrom}`,
      sourceKind: "manual",
    };
  }
  if (Number.isFinite(athleteFtp) && athleteFtp > 0) {
    return { ftp: athleteFtp, source: "Strava athlete profile", sourceKind: "strava" };
  }
  if (best60MinutePower !== null && best60MinutePower > 0) {
    return { ftp: best60MinutePower, source: "Estimated from best 60 min power", sourceKind: "estimated" };
  }
  if (best20MinutePower !== null && best20MinutePower > 0) {
    return { ftp: best20MinutePower * 0.95, source: "Estimated as 95% of best 20 min power", sourceKind: "estimated" };
  }
  return { ftp: null, source: null, sourceKind: null };
}

export function resolveWeightDetails(
  manualWeightKg: number | null | undefined,
  athleteWeight: number,
): { weightKg: number | null; source: string | null } {
  if (typeof manualWeightKg === "number" && Number.isFinite(manualWeightKg) && manualWeightKg > 0) {
    return { weightKg: manualWeightKg, source: "Manual weight setting" };
  }
  if (Number.isFinite(athleteWeight) && athleteWeight > 0) {
    return { weightKg: athleteWeight, source: "Strava athlete profile" };
  }
  return { weightKg: null, source: null };
}

export function buildPowerZoneEstimate(watts: Array<number | null>, ftp: number, times: number[]): PowerZoneEstimate | null {
  if (!Number.isFinite(ftp) || ftp <= 0) return null;
  const timeline = new PowerTimeline(watts, times);
  const result = { trackedSeconds: 0, aerobicSeconds: 0, thresholdVo2Seconds: 0, anaerobicSeconds: 0 };
  for (let i = 0; i < times.length - 1; i++) {
    if (!timeline.intervalValid(i)) continue;
    const duration = times[i + 1]! - times[i]!;
    result.trackedSeconds += duration;
    const power = watts[i]!;
    if (power <= ftp * 0.9) result.aerobicSeconds += duration;
    else if (power <= ftp * 1.2) result.thresholdVo2Seconds += duration;
    else result.anaerobicSeconds += duration;
  }
  return result.trackedSeconds ? result : null;
}

export function formatPowerZoneTime(seconds: number, totalSeconds: number): string {
  if (totalSeconds <= 0) {
    return formatTime(seconds);
  }
  const percentage = seconds / totalSeconds * 100;
  return `${formatTime(seconds)} (${percentage.toFixed(0)}%)`;
}

export function formatOptionalDecimal(value: number | null, suffix: string, digits: number): string {
  return value !== null && Number.isFinite(value)
    ? `${value.toFixed(digits)} ${suffix}`
    : "n/a";
}

export function resolvePowerDurationSeconds(currentActivity: DetailedActivity): number {
  return new PowerTimeline(currentActivity.stream?.watts ?? [], currentActivity.stream?.time ?? []).duration;
}

export function bestAveragePower(watts: Array<number | null>, seconds: number, times: number[]): number | null {
  return new PowerTimeline(watts, times).best(seconds);
}

export function buildPowerCurve(watts: Array<number | null>, times: number[]): Array<[number, number]> {
  const timeline = new PowerTimeline(watts, times);
  const durations: number[] = [];
  for (let seconds = 1; seconds <= timeline.duration; seconds += seconds < 60 ? 1 : seconds < 300 ? 5 : seconds < 1200 ? 30 : 60) durations.push(seconds);
  if (timeline.duration > 0 && durations.at(-1) !== timeline.duration) durations.push(timeline.duration);
  return durations.flatMap((seconds): Array<[number, number]> => {
    const power = timeline.best(seconds);
    return power === null ? [] : [[seconds, power]];
  });
}
