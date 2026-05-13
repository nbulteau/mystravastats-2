import type {
  GearAnalysisItem,
  GearMaintenanceAction,
  GearMaintenanceStatus,
  GearMaintenanceTask,
} from "@/models/gear-analysis.model";

export const MAINTENANCE_PROJECTION_MONTHS = 6;

export interface GearMaintenancePriorityItem {
  item: GearAnalysisItem;
  task: GearMaintenanceTask;
}

export interface GearMaintenanceProjection {
  status: "DUE_NOW" | "NEEDS_RECORD" | "PROJECTED" | "UNKNOWN";
  dueDate: string | null;
  monthsUntilDue: number | null;
  distanceMonthsUntilDue: number | null;
  timeMonthsUntilDue: number | null;
  recentMonthlyAverageDistance: number;
  recentActiveMonths: number;
  calendarMonthCount: number;
}

export function buildMaintenancePriorityItems(items: GearAnalysisItem[], limit = 8): GearMaintenancePriorityItem[] {
  return items
    .filter((item) => item.kind === "BIKE")
    .flatMap((item) =>
      item.maintenanceTasks
        .filter((task) => task.status === "OVERDUE" || task.status === "DUE" || task.status === "SOON")
        .map((task) => ({ item, task })),
    )
    .sort((left, right) => {
      const statusGap = maintenanceStatusRank(right.task.status) - maintenanceStatusRank(left.task.status);
      if (statusGap !== 0) return statusGap;
      return maintenanceUrgencyValue(left.task) - maintenanceUrgencyValue(right.task);
    })
    .slice(0, limit);
}

export function maintenanceStatusRank(status: GearMaintenanceStatus | string): number {
  if (status === "OVERDUE") return 3;
  if (status === "DUE") return 2;
  if (status === "SOON") return 1;
  return 0;
}

export function maintenanceOperationForAction(componentLabel: string, action: GearMaintenanceAction): string {
  return `${componentLabel} ${action === "REPLACEMENT" ? "replaced" : "serviced"}`;
}

export function buildMaintenanceProjection(
  item: GearAnalysisItem,
  task: GearMaintenanceTask,
  today = new Date(),
  calendarMonthCount = MAINTENANCE_PROJECTION_MONTHS,
): GearMaintenanceProjection {
  const recentMonthlyAverageDistance = recentCalendarMonthlyAverageDistance(item, today, calendarMonthCount);
  const recentActiveMonths = recentCalendarActiveMonths(item, today, calendarMonthCount);
  const dueNow = task.status === "OVERDUE" || task.status === "DUE";

  if (dueNow) {
    return {
      status: "DUE_NOW",
      dueDate: isoDate(today),
      monthsUntilDue: 0,
      distanceMonthsUntilDue: task.intervalDistance > 0 ? 0 : null,
      timeMonthsUntilDue: task.intervalMonths > 0 ? 0 : null,
      recentMonthlyAverageDistance,
      recentActiveMonths,
      calendarMonthCount,
    };
  }

  if (!task.lastMaintenance) {
    return {
      status: "NEEDS_RECORD",
      dueDate: null,
      monthsUntilDue: null,
      distanceMonthsUntilDue: null,
      timeMonthsUntilDue: null,
      recentMonthlyAverageDistance,
      recentActiveMonths,
      calendarMonthCount,
    };
  }

  const distanceMonthsUntilDue = projectedDistanceMonthsUntilDue(task, recentMonthlyAverageDistance);
  const timeMonthsUntilDue = projectedTimeMonthsUntilDue(task);
  const candidates = [distanceMonthsUntilDue, timeMonthsUntilDue].filter((value): value is number => value !== null);
  if (!candidates.length) {
    return {
      status: "UNKNOWN",
      dueDate: null,
      monthsUntilDue: null,
      distanceMonthsUntilDue,
      timeMonthsUntilDue,
      recentMonthlyAverageDistance,
      recentActiveMonths,
      calendarMonthCount,
    };
  }

  const monthsUntilDue = Math.max(0, Math.min(...candidates));
  return {
    status: "PROJECTED",
    dueDate: isoDate(addCalendarMonths(today, monthsUntilDue)),
    monthsUntilDue,
    distanceMonthsUntilDue,
    timeMonthsUntilDue,
    recentMonthlyAverageDistance,
    recentActiveMonths,
    calendarMonthCount,
  };
}

export function recentCalendarMonthlyAverageDistance(
  item: GearAnalysisItem,
  today = new Date(),
  calendarMonthCount = MAINTENANCE_PROJECTION_MONTHS,
): number {
  if (calendarMonthCount <= 0) return 0;
  const valuesByMonth = monthlyDistanceByKey(item);
  const total = recentCalendarMonthKeys(today, calendarMonthCount)
    .reduce((sum, key) => sum + (valuesByMonth.get(key) ?? 0), 0);
  return total / calendarMonthCount;
}

export function recentCalendarActiveMonths(
  item: GearAnalysisItem,
  today = new Date(),
  calendarMonthCount = MAINTENANCE_PROJECTION_MONTHS,
): number {
  const valuesByMonth = monthlyDistanceByKey(item);
  return recentCalendarMonthKeys(today, calendarMonthCount)
    .filter((key) => (valuesByMonth.get(key) ?? 0) > 0)
    .length;
}

function projectedDistanceMonthsUntilDue(task: GearMaintenanceTask, recentMonthlyAverageDistance: number): number | null {
  if (task.intervalDistance <= 0) return null;
  if (task.distanceRemaining <= 0) return 0;
  if (recentMonthlyAverageDistance <= 0) return null;
  return Math.ceil(task.distanceRemaining / recentMonthlyAverageDistance);
}

function projectedTimeMonthsUntilDue(task: GearMaintenanceTask): number | null {
  if (task.intervalMonths <= 0) return null;
  return Math.max(0, task.monthsRemaining);
}

function maintenanceUrgencyValue(task: GearMaintenanceTask): number {
  if (!task.lastMaintenance) return -1;
  if (task.intervalDistance > 0) {
    return task.status === "OVERDUE" ? -task.distanceSince : task.distanceRemaining;
  }
  if (task.intervalMonths > 0) {
    return task.status === "OVERDUE" ? -task.monthsSince : task.monthsRemaining;
  }
  return Number.MAX_SAFE_INTEGER;
}

function monthlyDistanceByKey(item: GearAnalysisItem): Map<string, number> {
  return new Map(item.monthlyDistance.map((point) => [point.periodKey, point.value]));
}

function recentCalendarMonthKeys(today: Date, count: number): string[] {
  const keys: string[] = [];
  const monthStart = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), 1));
  for (let offset = count - 1; offset >= 0; offset--) {
    keys.push(monthKey(addCalendarMonths(monthStart, -offset)));
  }
  return keys;
}

function monthKey(date: Date): string {
  return `${date.getUTCFullYear()}-${String(date.getUTCMonth() + 1).padStart(2, "0")}`;
}

function addCalendarMonths(date: Date, months: number): Date {
  const year = date.getUTCFullYear();
  const month = date.getUTCMonth() + months;
  const day = date.getUTCDate();
  const lastDay = daysInUtcMonth(year, month);
  return new Date(Date.UTC(year, month, Math.min(day, lastDay)));
}

function daysInUtcMonth(year: number, month: number): number {
  return new Date(Date.UTC(year, month + 1, 0)).getUTCDate();
}

function isoDate(date: Date): string {
  return date.toISOString().slice(0, 10);
}
