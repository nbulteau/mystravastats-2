export type HikingDifficultyLabel = "Easy" | "Moderate" | "Hard" | "Very hard" | "Epic";

export type HikingInsightStream = {
  altitude?: number[] | null;
};

export type HikingInsightActivity = {
  type?: string | null;
  sportType?: string | null;
  distance: number;
  elapsedTime: number;
  movingTime: number;
  totalElevationGain: number;
  elevHigh?: number | null;
  stream?: HikingInsightStream | null;
};

export type HikingInsights = {
  difficultyLabel: HikingDifficultyLabel;
  difficultyScore: number;
  distanceKm: number;
  elevationPerKm: number | null;
  maxContinuousClimbMeters: number | null;
  highestPointMeters: number | null;
  verticalSpeedMetersPerHour: number | null;
  pauseRatio: number | null;
  pausedSeconds: number;
  movingTimeSeconds: number;
  elapsedTimeSeconds: number;
};

const continuousClimbPositiveDeltaMeters = 0.5;
const continuousClimbDescentBreakMeters = 8;

export function isHikingActivityType(activityType?: string | null): boolean {
  const normalized = (activityType ?? "").trim().toLowerCase();
  return normalized === "hike" || normalized === "walk";
}

export function buildHikingInsights(activity: HikingInsightActivity | null): HikingInsights | null {
  if (!activity || !isHikingActivityType(activity.sportType || activity.type)) {
    return null;
  }

  const distanceMeters = finiteOrZero(activity.distance);
  const distanceKm = distanceMeters > 0 ? distanceMeters / 1000 : 0;
  const elevationGain = Math.max(0, finiteOrZero(activity.totalElevationGain));
  const movingTimeSeconds = Math.max(0, Math.trunc(finiteOrZero(activity.movingTime)));
  const elapsedTimeSeconds = Math.max(0, Math.trunc(finiteOrZero(activity.elapsedTime)));
  const altitudeSamples = finiteSamples(activity.stream?.altitude ?? []);
  const maxContinuousClimbMeters = computeMaxContinuousClimb(altitudeSamples);
  const highestPointMeters = resolveHighestPoint(altitudeSamples, activity.elevHigh);
  const elevationPerKm = distanceKm > 0 ? elevationGain / distanceKm : null;
  const verticalSpeedMetersPerHour = movingTimeSeconds > 0
    ? elevationGain / (movingTimeSeconds / 3600)
    : null;
  const pausedSeconds = Math.max(0, elapsedTimeSeconds - movingTimeSeconds);
  const pauseRatio = elapsedTimeSeconds > 0 ? pausedSeconds / elapsedTimeSeconds : null;
  const difficultyScore = computeHikingDifficultyScore(distanceKm, elevationGain, maxContinuousClimbMeters);

  return {
    difficultyLabel: resolveHikingDifficultyLabel(difficultyScore),
    difficultyScore,
    distanceKm,
    elevationPerKm,
    maxContinuousClimbMeters,
    highestPointMeters,
    verticalSpeedMetersPerHour,
    pauseRatio,
    pausedSeconds,
    movingTimeSeconds,
    elapsedTimeSeconds,
  };
}

export function computeMaxContinuousClimb(altitudeSamples: number[]): number | null {
  if (altitudeSamples.length < 2) {
    return null;
  }

  let currentGain = 0;
  let bestGain = 0;
  let descentDebt = 0;

  for (let index = 1; index < altitudeSamples.length; index += 1) {
    const previous = altitudeSamples[index - 1];
    const current = altitudeSamples[index];
    if (!Number.isFinite(previous) || !Number.isFinite(current)) {
      continue;
    }

    const delta = current - previous;
    if (delta > continuousClimbPositiveDeltaMeters) {
      currentGain += delta;
      descentDebt = 0;
      bestGain = Math.max(bestGain, currentGain);
      continue;
    }

    if (delta < -continuousClimbPositiveDeltaMeters) {
      descentDebt += Math.abs(delta);
      if (descentDebt >= continuousClimbDescentBreakMeters) {
        currentGain = 0;
        descentDebt = 0;
      }
    }
  }

  return bestGain > 0 ? bestGain : null;
}

function resolveHighestPoint(altitudeSamples: number[], elevHigh?: number | null): number | null {
  if (altitudeSamples.length > 0) {
    return Math.max(...altitudeSamples);
  }

  return typeof elevHigh === "number" && Number.isFinite(elevHigh) ? elevHigh : null;
}

function computeHikingDifficultyScore(
  distanceKm: number,
  elevationGainMeters: number,
  maxContinuousClimbMeters: number | null,
): number {
  return distanceKm + elevationGainMeters / 100 + (maxContinuousClimbMeters ?? 0) / 200;
}

function resolveHikingDifficultyLabel(score: number): HikingDifficultyLabel {
  if (score >= 38) return "Epic";
  if (score >= 26) return "Very hard";
  if (score >= 16) return "Hard";
  if (score >= 8) return "Moderate";
  return "Easy";
}

function finiteSamples(values: number[]): number[] {
  return values.filter((value) => Number.isFinite(value));
}

function finiteOrZero(value: number | null | undefined): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}
