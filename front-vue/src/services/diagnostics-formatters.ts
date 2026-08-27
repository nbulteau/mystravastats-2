import type { DataQualityIssue, DataQualitySummary } from "@/models/data-quality.model";
import type { HealthRecord } from "@/models/health.model";
import type { SourceMode } from "@/models/source-mode.model";

export function asRecord(value: unknown): HealthRecord {
  return typeof value === "object" && value !== null && !Array.isArray(value)
    ? value as HealthRecord
    : {};
}

export function normalizeDataQualitySummary(value: unknown): DataQualitySummary | null {
  const record = asRecord(value);
  if (Object.keys(record).length === 0) {
    return null;
  }
  return {
    status: textValue(record.status) || "not_applicable",
    provider: textValue(record.provider),
    issueCount: numberValue(record.issueCount) ?? 0,
    impactedActivities: numberValue(record.impactedActivities) ?? 0,
    excludedActivities: numberValue(record.excludedActivities) ?? 0,
    bySeverity: numberRecord(record.bySeverity),
    byCategory: numberRecord(record.byCategory),
    topIssues: Array.isArray(record.topIssues)
      ? record.topIssues.map(normalizeDataQualityIssue).filter((issue): issue is DataQualityIssue => issue !== null)
      : [],
  };
}

export function normalizeDataQualityIssue(value: unknown): DataQualityIssue | null {
  const record = asRecord(value);
  if (!textValue(record.id) && !textValue(record.message)) {
    return null;
  }
  return {
    id: textValue(record.id) || `${textValue(record.category)}-${textValue(record.field)}-${textValue(record.activityId)}`,
    source: textValue(record.source),
    activityId: numberValue(record.activityId),
    activityName: textValue(record.activityName),
    activityType: textValue(record.activityType),
    year: textValue(record.year),
    filePath: textValue(record.filePath),
    severity: textValue(record.severity) || "info",
    category: textValue(record.category),
    field: textValue(record.field),
    message: textValue(record.message),
    rawValue: textValue(record.rawValue),
    suggestion: textValue(record.suggestion),
    excludedFromStats: booleanValue(record.excludedFromStats),
    excludedAt: textValue(record.excludedAt),
  };
}

export function numberRecord(value: unknown): Record<string, number> {
  const record = asRecord(value);
  return Object.fromEntries(
    Object.entries(record)
      .map(([key, entry]) => [key, numberValue(entry) ?? 0]),
  );
}

export function textValue(value: unknown): string {
  if (typeof value === "string") return value.trim();
  if (typeof value === "number" && Number.isFinite(value)) return String(value);
  return "";
}

export function numberValue(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    return Number.isFinite(parsed) ? parsed : null;
  }
  return null;
}

export function booleanValue(value: unknown): boolean {
  if (typeof value === "boolean") return value;
  if (typeof value === "string") return value.toLowerCase() === "true";
  return false;
}

export function sourceSyncChangedActivityData(result: unknown): boolean {
  const sync = asRecord(result);
  const fit = asRecord(sync.fit);
  return booleanValue(sync.reloaded) || (numberValue(fit.importedFiles) ?? 0) > 0;
}

export function listValues(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value
    .map((item) => textValue(item))
    .filter((item) => item.length > 0);
}

export function recordList(value: unknown): HealthRecord[] {
  if (!Array.isArray(value)) return [];
  return value.map((item) => asRecord(item));
}

export function displayList(value: unknown): string {
  const values = listValues(value);
  return values.length > 0 ? values.join(", ") : "n/a";
}

export function yesNo(value: unknown): string {
  return booleanValue(value) ? "Yes" : "No";
}

export function inferProvider(payload: HealthRecord): string {
  if (payload.fitDirectory) return "fit";
  if (payload.gpxDirectory) return "gpx";
  if (payload.cacheRoot || payload.manifest) return "strava";
  return "unknown";
}

export function formatProvider(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "gpx") return "GPX";
  if (normalized === "fit") return "FIT";
  if (normalized === "strava") return "Strava";
  if (normalized === "composite") return "Composite";
  if (normalized === "") return "Unknown";
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}

export function formatFITSourceKind(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "fit_inbox") return "FIT inbox";
  if (normalized === "garmin_usb") return "Garmin USB";
  return value || "No source";
}

export function formatFITDeviceSyncStatus(value: string): string {
  const normalized = value.trim().toLowerCase();
  if (normalized === "ok") return "OK";
  if (normalized === "no_device") return "No device";
  if (normalized === "not_configured") return "Not configured";
  if (normalized === "failed") return "Failed";
  return value || "n/a";
}

export function formatDateTime(value: string | null): string {
  if (!value) return "n/a";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(date);
}

export function formatEpochMs(value: number | null): string {
  if (!value || value <= 0) return "n/a";
  return formatDateTime(new Date(value).toISOString());
}

export function formatBytes(value: number | null): string {
  if (value === null) return "n/a";
  if (value < 1024) return `${value} B`;
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`;
  return `${(value / (1024 * 1024)).toFixed(1)} MB`;
}

export function formatInteger(value: number | null): string {
  if (value === null) return "n/a";
  return new Intl.NumberFormat().format(value);
}

export function formatMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  return `${value.toFixed(0)} m`;
}

export function formatSignedMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : "";
  return `${sign}${value.toFixed(0)} m`;
}

export function formatDistanceMeters(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  return `${(value / 1000).toFixed(2)} km`;
}

export function formatSignedDistance(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : "";
  return `${sign}${(value / 1000).toFixed(2)} km`;
}

export function formatDurationSeconds(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const totalSeconds = Math.max(0, Math.round(value));
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const parts: string[] = [];
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0 || hours > 0) parts.push(`${minutes}m`);
  if (seconds > 0 || parts.length === 0) parts.push(`${seconds}s`);
  return parts.join(" ");
}

export function formatSignedDurationSeconds(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  const sign = value > 0 ? "+" : value < 0 ? "-" : "";
  return `${sign}${formatDurationSeconds(Math.abs(value))}`;
}

export function formatSpeed(value: number | null | undefined): string {
  if (value === null || value === undefined) return "n/a";
  return `${(value * 3.6).toFixed(1)} km/h`;
}

export function formatSignedSpeedDelta(before: number | null | undefined, after: number | null | undefined): string {
  if (before === null || before === undefined || after === null || after === undefined) return "n/a";
  const delta = (after - before) * 3.6;
  const sign = delta > 0 ? "+" : "";
  return `${sign}${delta.toFixed(1)} km/h`;
}

export function statusClass(status: string): string {
  if (status === "up") return "status-chip status-chip--up";
  if (status === "disabled") return "status-chip status-chip--neutral";
  if (status === "misconfigured") return "status-chip status-chip--warn";
  if (status === "down") return "status-chip status-chip--down";
  return "status-chip status-chip--neutral";
}

export function fileStatusClass(exists: boolean): string {
  return exists ? "file-state file-state--ok" : "file-state file-state--missing";
}

export function normalizeSourceMode(value: string): SourceMode {
  const normalized = value.trim().toUpperCase();
  if (normalized === "FIT" || normalized === "GPX") return normalized;
  return "STRAVA";
}

