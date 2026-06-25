export interface AthleteFtpSetting {
  effectiveFrom: string;
  ftp: number;
}

export interface AthletePerformanceSettings {
  ftpHistory: AthleteFtpSetting[];
  weightKg?: number | null;
}

export interface ResolvedManualFtp {
  ftp: number;
  effectiveFrom: string;
}

export type FtpEstimateConfidence = "high" | "medium" | "low" | "unavailable";

export interface FtpEstimate {
  available: boolean;
  ftp: number;
  method: string;
  methodLabel: string;
  bestPower: number;
  multiplier: number;
  basedOnSeconds: number;
  confidence: FtpEstimateConfidence;
  source: string;
  sourceKind: string;
  activityId: number;
  activityName: string;
  activityType: string;
  activityDate: string;
  windowDays: number;
  activityCount: number;
}

export function emptyAthletePerformanceSettings(): AthletePerformanceSettings {
  return {
    ftpHistory: [],
    weightKg: null,
  };
}

export function emptyFtpEstimate(): FtpEstimate {
  return {
    available: false,
    ftp: 0,
    method: "",
    methodLabel: "",
    bestPower: 0,
    multiplier: 0,
    basedOnSeconds: 0,
    confidence: "unavailable",
    source: "",
    sourceKind: "none",
    activityId: 0,
    activityName: "",
    activityType: "",
    activityDate: "",
    windowDays: 180,
    activityCount: 0,
  };
}

export function normalizeAthletePerformanceSettings(
  settings: AthletePerformanceSettings | null | undefined,
): AthletePerformanceSettings {
  const byDate = new Map<string, AthleteFtpSetting>();
  for (const setting of settings?.ftpHistory ?? []) {
    if (!isIsoDate(setting.effectiveFrom) || !Number.isFinite(setting.ftp) || setting.ftp <= 0) {
      continue;
    }
    byDate.set(setting.effectiveFrom, {
      effectiveFrom: setting.effectiveFrom,
      ftp: Math.trunc(setting.ftp),
    });
  }

  const weightKg = typeof settings?.weightKg === "number" ? settings.weightKg : null;
  return {
    ftpHistory: [...byDate.values()].sort((left, right) => left.effectiveFrom.localeCompare(right.effectiveFrom)),
    weightKg: weightKg !== null && Number.isFinite(weightKg) && weightKg > 0 ? weightKg : null,
  };
}

export function resolveManualFtpForDate(
  settings: AthletePerformanceSettings,
  activityDate: string | null | undefined,
): ResolvedManualFtp | null {
  if (!activityDate) {
    return null;
  }

  const dateKey = activityDate.slice(0, 10);
  if (!isIsoDate(dateKey)) {
    return null;
  }

  let selected: AthleteFtpSetting | null = null;
  for (const setting of normalizeAthletePerformanceSettings(settings).ftpHistory) {
    if (setting.effectiveFrom <= dateKey) {
      selected = setting;
    }
  }

  return selected
    ? {
        ftp: selected.ftp,
        effectiveFrom: selected.effectiveFrom,
      }
    : null;
}

export function normalizeFtpEstimate(estimate: FtpEstimate | null | undefined): FtpEstimate {
  const empty = emptyFtpEstimate();
  if (!estimate) {
    return empty;
  }

  const confidence = estimate.confidence === "high" ||
    estimate.confidence === "medium" ||
    estimate.confidence === "low" ||
    estimate.confidence === "unavailable"
    ? estimate.confidence
    : "unavailable";

  return {
    available: Boolean(estimate.available) && Number.isFinite(estimate.ftp) && estimate.ftp > 0,
    ftp: normalizeNonNegativeInt(estimate.ftp),
    method: typeof estimate.method === "string" ? estimate.method : "",
    methodLabel: typeof estimate.methodLabel === "string" ? estimate.methodLabel : "",
    bestPower: normalizeNonNegativeInt(estimate.bestPower),
    multiplier: typeof estimate.multiplier === "number" && Number.isFinite(estimate.multiplier)
      ? estimate.multiplier
      : 0,
    basedOnSeconds: normalizeNonNegativeInt(estimate.basedOnSeconds),
    confidence,
    source: typeof estimate.source === "string" ? estimate.source : "",
    sourceKind: typeof estimate.sourceKind === "string" ? estimate.sourceKind : "none",
    activityId: normalizeNonNegativeInt(estimate.activityId),
    activityName: typeof estimate.activityName === "string" ? estimate.activityName : "",
    activityType: typeof estimate.activityType === "string" ? estimate.activityType : "",
    activityDate: typeof estimate.activityDate === "string" && isIsoDate(estimate.activityDate)
      ? estimate.activityDate
      : "",
    windowDays: normalizeNonNegativeInt(estimate.windowDays) || empty.windowDays,
    activityCount: normalizeNonNegativeInt(estimate.activityCount),
  };
}

export function isIsoDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return false;
  }
  const parsed = new Date(`${value}T00:00:00Z`);
  return !Number.isNaN(parsed.getTime()) && parsed.toISOString().slice(0, 10) === value;
}

function normalizeNonNegativeInt(value: unknown): number {
  const parsed = typeof value === "number" ? value : Number(value);
  if (!Number.isFinite(parsed) || parsed < 0) {
    return 0;
  }
  return Math.trunc(parsed);
}
