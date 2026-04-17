import type { ChartPeriodPoint } from "@/models/chart-period-point.model";

export function calculateTrendLine(data: number[]): number[] {
    const n = data.length;
    if (n === 0) {
      return [];
    }
    if (n === 1) {
      return [data[0]];
    }
    const xSum = data.reduce((sum, _, index) => sum + index, 0);
    const ySum = data.reduce((sum, value) => sum + value, 0);
    const xySum = data.reduce((sum, value, index) => sum + index * value, 0);
    const xSquaredSum = data.reduce((sum, _, index) => sum + index * index, 0);
  
    const slope = (n * xySum - xSum * ySum) / (n * xSquaredSum - xSum * xSum);
    const intercept = (ySum - slope * xSum) / n;
  
    return data.map((_, index) => slope * index + intercept);
  }

    export function calculateAverageLine(data: number[]): number[] {
        if (data.length === 0) {
            return [];
        }
        const average = data.reduce((sum, value) => sum + value, 0) / data.length;

        return Array(data.length).fill(average);
    }

export type PeriodEntry = {
  key: string;
  value: number;
  activityCount: number;
};

type LegacyPeriodPoint = Record<string, number>;
type PeriodPointLike = ChartPeriodPoint | LegacyPeriodPoint;

function isChartPeriodPoint(item: PeriodPointLike): item is ChartPeriodPoint {
  return (
    typeof (item as ChartPeriodPoint).periodKey === "string"
    && typeof (item as ChartPeriodPoint).value === "number"
  );
}

export function normalizePeriodPoints(data: PeriodPointLike[]): ChartPeriodPoint[] {
  return data
    .map((item) => {
      if (isChartPeriodPoint(item)) {
        return {
          periodKey: item.periodKey,
          value: Number(item.value ?? 0),
          activityCount: Number(item.activityCount ?? 0),
        };
      }

      const [key, value] = Object.entries(item)[0] ?? [];
      if (!key) {
        return null;
      }

      return {
        periodKey: key,
        value: Number(value ?? 0),
        activityCount: 0,
      };
    })
    .filter((entry): entry is ChartPeriodPoint => entry !== null);
}

export function extractPeriodEntries(data: PeriodPointLike[]): PeriodEntry[] {
  return normalizePeriodPoints(data).map((entry) => ({
    key: entry.periodKey,
    value: entry.value,
    activityCount: entry.activityCount,
  }));
}

export function weekLabel(periodKey: string): string {
  const weekNumber = Number.parseInt(periodKey, 10);
  if (Number.isNaN(weekNumber)) {
    return periodKey;
  }
  return `W${String(weekNumber).padStart(2, "0")}`;
}

function isoWeekNumber(date: Date): number {
  const workingDate = new Date(Date.UTC(date.getFullYear(), date.getMonth(), date.getDate()));
  const day = workingDate.getUTCDay() || 7;
  workingDate.setUTCDate(workingDate.getUTCDate() + 4 - day);
  const yearStart = new Date(Date.UTC(workingDate.getUTCFullYear(), 0, 1));
  return Math.ceil((((workingDate.getTime() - yearStart.getTime()) / 86400000) + 1) / 7);
}

export function calculateYtdAverageLine(
  data: number[],
  selectedYear: string,
  period: "MONTHS" | "WEEKS",
): number[] {
  if (data.length === 0) {
    return [];
  }

  const now = new Date();
  const isCurrentYear = selectedYear === String(now.getFullYear());
  const periodLimit = period === "MONTHS"
    ? Math.min(data.length, now.getMonth() + 1)
    : Math.min(data.length, isoWeekNumber(now));
  const limit = isCurrentYear ? Math.max(1, periodLimit) : data.length;
  const scoped = data.slice(0, limit);
  const average = scoped.reduce((sum, value) => sum + value, 0) / scoped.length;

  return Array(data.length).fill(average);
}
