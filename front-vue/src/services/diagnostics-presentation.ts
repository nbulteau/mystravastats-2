import type { DataQualityIssue } from "@/models/data-quality.model";
import type { HealthRecord } from "@/models/health.model";
import {
  formatDistanceMeters,
  formatDurationSeconds,
  formatInteger,
  formatMeters,
  formatProvider,
  formatSignedDistance,
  formatSignedDurationSeconds,
  formatSignedMeters,
  textValue,
} from "@/services/diagnostics-formatters";

export function formatSourceField(value: string): string {
  const labels: Record<string, string> = {
    activities: "Activities",
    trace: "GPS trace",
    elevation: "Elevation",
    heartRate: "Heart rate",
    power: "Power",
    cadence: "Cadence",
    distance: "Distance",
    moving_time: "Moving time",
    start_latlng: "Start point",
  };
  return labels[value] || value;
}

export function parseConflictNumber(value: string): number | null {
  const match = value.replace(",", ".").match(/-?\d+(?:\.\d+)?/);
  if (!match) return null;
  const parsed = Number.parseFloat(match[0]);
  return Number.isFinite(parsed) ? parsed : null;
}

export function compositePrimaryProviderLabel(otherSource: string, configuredProviders: string[]): string {
  const other = formatProvider(otherSource);
  return configuredProviders
    .map((provider) => formatProvider(provider))
    .find((provider) => provider !== other) || "Primary source";
}

export function compositeConflictTitle(field: string): string {
  const labels: Record<string, string> = {
    distance: "Distance differs between sources",
    moving_time: "Moving time differs between sources",
    start_latlng: "Start point differs between sources",
  };
  return labels[field] || `${formatSourceField(field)} differs between sources`;
}

export function formatCompositeConflictValue(field: string, value: string): string {
  const numeric = parseConflictNumber(value);
  if (numeric === null) return value || "n/a";
  if (field === "moving_time") return formatDurationSeconds(numeric);
  if (field === "distance") return formatDistanceMeters(numeric);
  if (field === "start_latlng") return formatMeters(numeric);
  return formatInteger(numeric);
}

export function formatCompositeConflictDelta(field: string, primary: string, other: string): string {
  const primaryValue = parseConflictNumber(primary);
  const otherValue = parseConflictNumber(other);
  if (primaryValue === null || otherValue === null) return "n/a";
  const delta = otherValue - primaryValue;
  if (field === "moving_time") return formatSignedDurationSeconds(delta);
  if (field === "distance") return formatSignedDistance(delta);
  if (field === "start_latlng") return formatSignedMeters(delta);
  const sign = delta > 0 ? "+" : "";
  return `${sign}${formatInteger(delta)}`;
}

export function formatCompositeConflict(conflict: HealthRecord, configuredProviders: string[]) {
  const rawField = textValue(conflict.field);
  const otherProvider = formatProvider(textValue(conflict.source));
  const primaryProvider = compositePrimaryProviderLabel(otherProvider, configuredProviders);
  const rawPrimary = textValue(conflict.primary) || "n/a";
  const rawOther = textValue(conflict.other) || "n/a";
  const delta = formatCompositeConflictDelta(rawField, rawPrimary, rawOther);
  const fieldLabel = formatSourceField(rawField).toLowerCase();
  const summary = delta === "n/a"
    ? `${otherProvider} reports a different ${fieldLabel}; ${primaryProvider} keeps the value used in totals.`
    : `${otherProvider} differs from ${primaryProvider} by ${delta} for ${fieldLabel}; ${primaryProvider} keeps the value used in totals.`;

  return {
    id: `${otherProvider}-${rawField}-${rawPrimary}-${rawOther}`,
    title: compositeConflictTitle(rawField),
    summary,
    primaryLabel: primaryProvider,
    primaryValue: formatCompositeConflictValue(rawField, rawPrimary),
    otherLabel: otherProvider,
    otherValue: formatCompositeConflictValue(rawField, rawOther),
    delta,
    rawField: rawField || "unknown",
    rawPrimary,
    rawOther,
  };
}

export function formatDataQualityCategory(value: string): string {
  const labels: Record<string, string> = {
    INVALID_FILE: "Invalid file", MISSING_STREAM: "Missing detailed stream",
    MISSING_STREAM_FIELD: "Missing stream field", STREAM_DATA_COVERAGE: "Stream coverage",
    INVALID_VALUE: "Invalid value", INCONSISTENT_TIME: "Time", GPS_GLITCH: "GPS glitch",
    GPS_GAP: "GPS recording gap", ALTITUDE_SPIKE: "Altitude spike", FALLBACK_VALUE: "Fallback",
  };
  return labels[value] || value;
}

export function correctionLabel(value: string): string {
  const labels: Record<string, string> = {
    REMOVE_GPS_POINT: "Repair GPS point", INTERPOLATE_GPS_POINT: "Interpolate GPS point",
    SMOOTH_ALTITUDE_SPIKE: "Smooth altitude spike", MASK_INVALID_VALUE: "Mask invalid value",
    RECALCULATE_FROM_STREAM: "Recalculate from stream",
  };
  return labels[value] || value;
}

export function dataQualityCategoryTooltip(value: string): string {
  const descriptions: Record<string, string> = {
    INVALID_FILE: "The source file could not be parsed reliably or contains an unsupported payload.",
    MISSING_STREAM: "The activity has no detailed stream in the local cache. In Strava mode this can usually be fetched from the API.",
    MISSING_STREAM_FIELD: "A detailed stream exists, but one required field such as time, distance, GPS trace, or altitude is missing or inconsistent.",
    STREAM_DATA_COVERAGE: "Optional sensor samples are incomplete. Summary values may exist, but charts using sample-by-sample data will be partial.",
    INVALID_VALUE: "A summary value is missing, not serializable, or outside a plausible range for the activity type.",
    INCONSISTENT_TIME: "Timing fields disagree, for example moving time is greater than elapsed time.",
    GPS_GLITCH: "The GPS trace contains a jump that implies an impossible speed for this activity type.",
    GPS_GAP: "Recording resumes far from the previous point after a long time gap. The straight bridge is not trusted automatically.",
    ALTITUDE_SPIKE: "The altitude stream contains a sharp elevation jump that can distort elevation gain.",
    FALLBACK_VALUE: "The displayed value comes from a fallback calculation instead of the original source data.",
  };
  return descriptions[value] || "Data quality finding detected for this category.";
}

export function dataQualityCategoryPriority(value: string): number {
  const priority: Record<string, number> = {
    INVALID_FILE: 0, INVALID_VALUE: 1, INCONSISTENT_TIME: 2, GPS_GLITCH: 3, GPS_GAP: 4,
    ALTITUDE_SPIKE: 5, MISSING_STREAM_FIELD: 6, MISSING_STREAM: 7, FALLBACK_VALUE: 8,
    STREAM_DATA_COVERAGE: 9,
  };
  return priority[value] ?? 99;
}

export function dataQualitySeverityClass(severity: string): string {
  if (severity === "critical") return "quality-severity quality-severity--critical";
  if (severity === "warning") return "quality-severity quality-severity--warning";
  return "quality-severity quality-severity--info";
}

export function dataQualityStatClass(tone: string): string {
  if (tone === "down") return "quality-metric quality-metric--down";
  if (tone === "warn") return "quality-metric quality-metric--warn";
  return "quality-metric";
}

export function dataQualityActionForIssue(issue: DataQualityIssue): string {
  if (issue.correction?.available && issue.correction.safety === "safe") return "safe";
  if (issue.correction?.available && issue.correction.safety === "manual") return "manual";
  return "unsupported";
}

export function dataQualityActionLabel(issue: DataQualityIssue): string {
  const action = dataQualityActionForIssue(issue);
  if (action === "safe") return "Safe local fix";
  if (action === "manual") return "Manual review";
  return "Unsupported";
}

export function dataQualityImpactTokens(issue: DataQualityIssue): string[] {
  const text = `${issue.category} ${issue.field} ${issue.message}`.toLowerCase();
  const tokens = new Set<string>();
  if (issue.severity === "critical" || issue.severity === "warning") tokens.add("records");
  if (text.includes("distance") || text.includes("gps") || text.includes("latlng")) tokens.add("distance");
  if (text.includes("elevation") || text.includes("altitude")) tokens.add("elevation");
  if (text.includes("speed") || text.includes("time") || text.includes("moving")) tokens.add("speed");
  if (text.includes("stream") || text.includes("heart") || text.includes("power") || text.includes("watts") || text.includes("cadence")) tokens.add("sensor");
  if (tokens.size === 0) tokens.add("records");
  return Array.from(tokens);
}

export function formatDataQualityImpactToken(value: string): string {
  return ({ records: "Records", distance: "Distance", elevation: "Elevation", speed: "Speed", sensor: "Sensor" } as Record<string, string>)[value] || value;
}
