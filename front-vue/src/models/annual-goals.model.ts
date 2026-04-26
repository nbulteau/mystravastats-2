export type AnnualGoalMetric =
  | "DISTANCE_KM"
  | "ELEVATION_METERS"
  | "MOVING_TIME_SECONDS"
  | "ACTIVITIES"
  | "ACTIVE_DAYS"
  | "EDDINGTON";

export type AnnualGoalStatus = "NOT_SET" | "AHEAD" | "ON_TRACK" | "BEHIND";

export type AnnualGoalTargets = {
  distanceKm: number | null;
  elevationMeters: number | null;
  movingTimeSeconds: number | null;
  activities: number | null;
  activeDays: number | null;
  eddington: number | null;
};

export type AnnualGoalProgress = {
  metric: AnnualGoalMetric;
  label: string;
  unit: string;
  current: number;
  target: number;
  progressPercent: number;
  expectedProgressPercent: number;
  projectedEndOfYear: number;
  requiredPace: number;
  requiredPaceUnit: string;
  requiredWeeklyPace: number;
  last30Days: number;
  last30DaysWeeklyPace: number;
  weeklyPaceGap: number;
  suggestedTarget: number | null;
  monthly: AnnualGoalMonth[];
  status: AnnualGoalStatus;
};

export type AnnualGoalMonth = {
  month: number;
  value: number;
  cumulative: number;
  expectedCumulative: number;
};

export type AnnualGoals = {
  year: number;
  activityTypeKey: string;
  targets: AnnualGoalTargets;
  progress: AnnualGoalProgress[];
};

export function emptyAnnualGoalTargets(): AnnualGoalTargets {
  return {
    distanceKm: null,
    elevationMeters: null,
    movingTimeSeconds: null,
    activities: null,
    activeDays: null,
    eddington: null,
  };
}

export function emptyAnnualGoals(year = new Date().getFullYear()): AnnualGoals {
  return {
    year,
    activityTypeKey: "",
    targets: emptyAnnualGoalTargets(),
    progress: [],
  };
}
