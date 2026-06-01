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

export function emptyAthletePerformanceSettings(): AthletePerformanceSettings {
  return {
    ftpHistory: [],
    weightKg: null,
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

export function isIsoDate(value: string): boolean {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return false;
  }
  const parsed = new Date(`${value}T00:00:00Z`);
  return !Number.isNaN(parsed.getTime()) && parsed.toISOString().slice(0, 10) === value;
}
