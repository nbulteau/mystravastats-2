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
  const durationSeconds = resolvePowerDurationSeconds(currentActivity);
  const averagePower = watts.length > 0
    ? watts.every(Number.isFinite) ? watts.reduce((sum, value) => sum + value, 0) / watts.length : null
    : currentActivity.averageWatts > 0
      ? currentActivity.averageWatts
      : null;
  const maxPower = watts.length > 0
    ? watts.every(Number.isFinite) ? Math.max(...watts) : null
    : currentActivity.maxWatts > 0
      ? currentActivity.maxWatts
      : null;
  const best20MinutePower = bestAveragePower(watts, 20 * 60);
  const best60MinutePower = bestAveragePower(watts, 60 * 60);
  const normalizedPower = normalizedPowerFromWatts(watts);
  const manualFtp = resolveManualFtpForDate(
    performanceSettings,
    currentActivity.startDateLocal || currentActivity.startDate,
  );
  const ftpDetails = resolveFtpDetails(manualFtp, athleteFtp, best60MinutePower, best20MinutePower);
  const weightDetails = resolveWeightDetails(performanceSettings.weightKg, athleteWeight);
  const powerZoneEstimate = ftpDetails.ftp !== null ? buildPowerZoneEstimate(watts, ftpDetails.ftp) : null;
  const intensityFactor =
    normalizedPower !== null && ftpDetails.ftp !== null
      ? normalizedPower / ftpDetails.ftp
      : null;
  const trainingStressScore =
    normalizedPower !== null &&
    intensityFactor !== null &&
    ftpDetails.ftp !== null &&
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
  };
}

export function sanitizePowerSamples(watts: Array<number | null>): number[] {
  return watts.map((value) => typeof value === "number" && Number.isFinite(value) && value >= 0 ? value : Number.NaN);
}

export function normalizedPowerFromWatts(watts: Array<number | null>): number | null {
  if (watts.length < 30 || watts.some((value) => value === null || !Number.isFinite(value))) {
    return null;
  }

  const rollingAverages = rollingAverage(sanitizePowerSamples(watts), 30);
  if (!rollingAverages.length) {
    return null;
  }

  const fourthPowerAverage = rollingAverages.reduce(
    (sum, value) => sum + Math.pow(value, 4),
    0,
  ) / rollingAverages.length;

  return Math.pow(fourthPowerAverage, 0.25);
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

export function buildPowerZoneEstimate(watts: Array<number | null>, ftp: number): PowerZoneEstimate | null {
  if (!watts.length || !Number.isFinite(ftp) || ftp <= 0) {
    return null;
  }

  const aerobicUpperBound = ftp * 0.9;
  const anaerobicLowerBound = ftp * 1.2;

  const result = watts.reduce<PowerZoneEstimate>((estimate, rawPower) => {
    if (rawPower === null || !Number.isFinite(rawPower) || rawPower < 0) return estimate;
    const power = rawPower;
    estimate.trackedSeconds += 1;
    if (power <= aerobicUpperBound) {
      estimate.aerobicSeconds += 1;
    } else if (power <= anaerobicLowerBound) {
      estimate.thresholdVo2Seconds += 1;
    } else {
      estimate.anaerobicSeconds += 1;
    }
    return estimate;
  }, {
    trackedSeconds: 0,
    aerobicSeconds: 0,
    thresholdVo2Seconds: 0,
    anaerobicSeconds: 0,
  });
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
  const time = currentActivity.stream?.time ?? [];
  const lastTime = time.length > 0 ? time[time.length - 1] : null;
  if (lastTime !== null && Number.isFinite(lastTime) && lastTime > 0) {
    return lastTime;
  }
  return currentActivity.elapsedTime > 0 ? currentActivity.elapsedTime : currentActivity.movingTime;
}

export function bestAveragePower(watts: Array<number | null>, windowSamples: number): number | null {
  if (windowSamples <= 0 || !Number.isInteger(windowSamples) || watts.length < windowSamples) return null;
  let sum = 0;
  let missing = 0;
  let best: number | null = null;
  const valid = (value: number | null | undefined): value is number => typeof value === "number" && Number.isFinite(value) && value >= 0;
  for (let i = 0; i < watts.length; i++) {
    const incoming = watts[i];
    if (valid(incoming)) sum += incoming; else missing++;
    if (i >= windowSamples) {
      const outgoing = watts[i - windowSamples];
      if (valid(outgoing)) sum -= outgoing; else missing--;
    }
    if (i >= windowSamples - 1 && missing === 0) best = Math.max(best ?? -Infinity, sum / windowSamples);
  }
  return best;
}

export function buildPowerCurve(watts: Array<number | null>): Array<[number, number]> {
  const samples = sanitizePowerSamples(watts);
  const durations = Array.from({ length: samples.length }, (_, index) => index + 1)
    .filter((durationSeconds) =>
      durationSeconds <= 60
      || (durationSeconds <= 5 * 60 && durationSeconds % 5 === 0)
      || (durationSeconds <= 20 * 60 && durationSeconds % 30 === 0)
      || durationSeconds % 60 === 0
      || durationSeconds === samples.length
    );
  return durations.flatMap((durationSeconds): Array<[number, number]> => {
    const power = bestAveragePower(samples, durationSeconds);
    return power === null ? [] : [[durationSeconds, power]];
  });
}
