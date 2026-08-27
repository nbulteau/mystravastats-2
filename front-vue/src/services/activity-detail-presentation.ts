import type { ActivitySourceConflict, ActivitySourceRef, DetailedActivity, StravaSegmentEffort } from "@/models/activity.model";
import { formatSpeedWithUnit, formatTime } from "@/utils/formatters";

export type RouteEffortDescriptionInput = {
  distance: number;
  seconds: number;
  deltaAltitude?: number | null;
  elevationGain?: number | null;
  elevationLoss?: number | null;
  averagePower?: number | null;
  grade?: number | null;
};

export function resolveEffectiveActivityType(activity?: DetailedActivity | null): string {
  return activity?.sportType || activity?.type || "Ride";
}

export function formatCadenceValue(cadence: number, activityType: string): string {
  const running = activityType.endsWith("Run");
  return `${Math.round(running ? cadence * 2 : cadence)} ${running ? "spm" : "rpm"}`;
}

export function comparisonDeltaClass(value: number, positiveIsGood: boolean): string {
  if (Math.abs(value) < 0.0001) return "detail-comparison__delta detail-comparison__delta--flat";
  const isGood = positiveIsGood ? value > 0 : value < 0;
  return `detail-comparison__delta ${isGood ? "detail-comparison__delta--good" : "detail-comparison__delta--warn"}`;
}

export function formatSignedNumber(value: number, suffix: string, digits = 1): string {
  return `${value > 0 ? "+" : ""}${value.toFixed(digits)}${suffix}`;
}

export const formatSignedDistance = (value: number) => formatSignedNumber(value / 1000, " km", 1);
export const formatSignedMeters = (value: number) => formatSignedNumber(value, " m", 0);
export const formatSignedSpeed = (value: number) => formatSignedNumber(value * 3.6, " km/h", 1);

export function formatSignedTime(value: number): string {
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${formatTime(Math.abs(value))}`;
}

export function formatLocalActivityDate(value: string): string {
  const match = value.match(/^(\d{4})-(\d{2})-(\d{2})[T\s](\d{2}):(\d{2})/);
  if (!match) return value.substring(0, 16);
  const [, year, month, day, hour, minute] = match;
  return new Date(Number(year), Number(month) - 1, Number(day), Number(hour), Number(minute)).toLocaleString("en-US", {
    weekday: "short", day: "2-digit", month: "short", year: "numeric", hour: "2-digit", minute: "2-digit",
  });
}

export function formatProviderLabel(provider: string | undefined): string {
  const normalized = (provider ?? "").trim().toLowerCase();
  if (normalized === "strava") return "Strava";
  if (normalized === "fit") return "FIT";
  if (normalized === "gpx") return "GPX";
  if (normalized === "ridewithgps") return "RideWithGPS";
  return provider || "Unknown";
}

export function formatActivitySourceRefs(sources: ActivitySourceRef[] | undefined): string {
  if (!Array.isArray(sources) || sources.length === 0) return "n/a";
  return sources.map((source) => `${formatProviderLabel(source.provider)} #${source.activityId}${source.hasStream ? " + stream" : ""}`).join(" · ");
}

export function formatSourceConflictSample(conflicts: ActivitySourceConflict[] | undefined): string | undefined {
  if (!Array.isArray(conflicts) || conflicts.length === 0) return undefined;
  const conflict = conflicts[0];
  return `${conflict.field}: ${formatProviderLabel(conflict.source)} ${conflict.primary} -> ${conflict.other}`;
}

export function formatComparisonDate(value: string): string {
  if (!value) return "";
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value.substring(0, 10);
  return parsed.toLocaleDateString("en-US", { day: "2-digit", month: "short" });
}

export function formatStravaSegmentDescription(effort: StravaSegmentEffort, activityType: string): string {
  const parts = [formatRouteEffortDescription({
    distance: effort.distance,
    seconds: effort.elapsedTime,
    averagePower: effort.averageWatts,
    grade: effort.segment.averageGrade,
  }, activityType)];
  if (effort.averageHeartRate > 0) parts.push(`${Math.round(effort.averageHeartRate)} bpm`);
  return parts.join(" · ");
}

export function formatRouteEffortDescription(effort: RouteEffortDescriptionInput, activityType: string): string {
  const parts = [`${(effort.distance / 1000).toFixed(2)} km`, formatTime(effort.seconds)];
  if (effort.seconds > 0 && effort.distance > 0) parts.push(formatSpeedWithUnit(effort.distance / effort.seconds, activityType));
  const gradient = resolveEffortGradient(effort);
  if (gradient !== null) parts.push(`Grade ${gradient.toFixed(1)}%`);
  const elevation = resolveEffortElevationLabel(effort);
  if (elevation) parts.push(elevation);
  if (effort.averagePower && effort.averagePower > 0) parts.push(`${Math.round(effort.averagePower)} W`);
  return parts.join(" · ");
}

export function resolveEffortGradient(effort: RouteEffortDescriptionInput): number | null {
  const explicitGrade = finiteNumberOrNull(effort.grade);
  if (explicitGrade !== null) return explicitGrade;
  if (effort.distance <= 0) return null;
  const deltaAltitude = finiteNumberOrNull(effort.deltaAltitude);
  const netGradient = deltaAltitude !== null ? (deltaAltitude / effort.distance) * 100 : null;
  if (netGradient !== null && Math.abs(netGradient) >= 0.05) return netGradient;
  const gain = finiteNumberOrNull(effort.elevationGain);
  const loss = finiteNumberOrNull(effort.elevationLoss);
  if (gain !== null || loss !== null) {
    if ((gain ?? 0) >= (loss ?? 0) && (gain ?? 0) >= 0.5) return ((gain ?? 0) / effort.distance) * 100;
    if ((loss ?? 0) > (gain ?? 0) && (loss ?? 0) >= 0.5) return -((loss ?? 0) / effort.distance) * 100;
  }
  return netGradient;
}

export function resolveEffortElevationLabel(effort: RouteEffortDescriptionInput): string | null {
  const gain = finiteNumberOrNull(effort.elevationGain);
  const loss = finiteNumberOrNull(effort.elevationLoss);
  const parts: string[] = [];
  if (gain !== null && gain >= 0.5) parts.push(`D+ ${Math.round(gain)} m`);
  if (loss !== null && loss >= 0.5) parts.push(`D- ${Math.round(loss)} m`);
  if (parts.length > 0) return parts.join(" · ");
  const delta = finiteNumberOrNull(effort.deltaAltitude);
  return delta === null ? null : `${delta >= 0 ? "D+" : "D-"} ${Math.abs(Math.round(delta))} m`;
}

export function detailedActivityWarning(activity: DetailedActivity): string | null {
  return Array.isArray(activity.stream?.distance) && activity.stream.distance.length > 0
    ? null
    : "Detailed streams are missing in local cache for this activity. If you are running in cache-only mode, reconnect to Strava and refresh cache.";
}

function finiteNumberOrNull(value: number | null | undefined): number | null {
  return typeof value === "number" && Number.isFinite(value) ? value : null;
}
