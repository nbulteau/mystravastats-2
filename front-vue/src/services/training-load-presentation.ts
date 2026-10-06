import type { TrainingLoadActivity, TrainingLoadPeriod } from "@/generated/api-contract";

export const loadReasons: Record<string,string> = {
  "available": "Calculated over the covered duration",
  "unsupported-sport": "Power-based cycling load is not applicable to this sport",
  "missing-power": "No power readings available",
  "invalid-time": "Missing or inconsistent timestamps",
  "power-gaps": "Missing power readings or intervals longer than 10 seconds",
  "too-short": "At least 30 covered seconds are required",
  "incomplete-activity": "The recording does not cover the full activity",
  "missing-dated-ftp": "No FTP setting valid on the activity date",
};
export const loadNumber = (value: number | null): string => value === null ? "Unavailable" : value.toFixed(1);
export const hours = (seconds: number): string => seconds > 0 && seconds < 3600 ? `${Math.max(1, Math.round(seconds/60))} min` : `${(seconds/3600).toFixed(1)} h`;
export function shiftWeek(week: string, offset: number): string {
  const date = new Date(`${week}T12:00:00Z`);
  date.setUTCDate(date.getUTCDate()+offset*7);
  return date.toISOString().slice(0,10);
}
export function volumeComparison(current: TrainingLoadPeriod, previous: TrainingLoadPeriod): string {
  const delta = current.movingSeconds-previous.movingSeconds;
  if (previous.activityCount === 0) return "No activities recorded in the previous week; no percentage comparison is available.";
  return `${hours(current.movingSeconds)} recorded this week versus ${hours(previous.movingSeconds)} from ${previous.startDate} to ${previous.endDate} (${delta>=0?"+":"−"}${hours(Math.abs(delta))}).`;
}
export function largestMeasuredContribution(rows: TrainingLoadActivity[]): TrainingLoadActivity | undefined {
  return rows.filter(row => row.source === "measured" && row.load !== null).sort((a,b) => b.load! - a.load!)[0];
}
