import { describe, expect, it } from "vitest";
import type { GearAnalysisItem, GearMaintenanceTask } from "@/models/gear-analysis.model";
import {
  buildMaintenancePriorityItems,
  buildMaintenanceProjection,
  maintenanceOperationForAction,
  recentCalendarMonthlyAverageDistance,
} from "@/utils/gear-maintenance";

describe("gear maintenance projections", () => {
  const today = new Date("2026-05-13T12:00:00Z");

  it("includes zero-distance calendar months in the recent monthly pace", () => {
    const item = bikeItem({
      monthlyDistance: [
        { periodKey: "2026-01", value: 1_000_000, activityCount: 4 },
        { periodKey: "2026-03", value: 1_000_000, activityCount: 4 },
        { periodKey: "2026-04", value: 1_000_000, activityCount: 4 },
      ],
    });

    expect(recentCalendarMonthlyAverageDistance(item, today, 6)).toBe(500_000);
  });

  it("returns an explicit due date from distance pace and calendar zero months", () => {
    const item = bikeItem({
      monthlyDistance: [
        { periodKey: "2026-01", value: 1_000_000, activityCount: 4 },
        { periodKey: "2026-03", value: 1_000_000, activityCount: 4 },
        { periodKey: "2026-04", value: 1_000_000, activityCount: 4 },
      ],
    });
    const task = maintenanceTask({
      status: "SOON",
      distanceRemaining: 1_000_000,
    });

    const projection = buildMaintenanceProjection(item, task, today, 6);

    expect(projection.status).toBe("PROJECTED");
    expect(projection.monthsUntilDue).toBe(2);
    expect(projection.dueDate).toBe("2026-07-13");
    expect(projection.recentActiveMonths).toBe(3);
  });

  it("uses the earliest due date when a task has distance and calendar rules", () => {
    const item = bikeItem({
      monthlyDistance: [
        { periodKey: "2026-01", value: 100_000, activityCount: 1 },
        { periodKey: "2026-02", value: 100_000, activityCount: 1 },
      ],
    });
    const task = maintenanceTask({
      status: "SOON",
      intervalDistance: 1_500_000,
      distanceRemaining: 1_500_000,
      intervalMonths: 12,
      monthsRemaining: 1,
    });

    const projection = buildMaintenanceProjection(item, task, today, 6);

    expect(projection.timeMonthsUntilDue).toBe(1);
    expect(projection.monthsUntilDue).toBe(1);
    expect(projection.dueDate).toBe("2026-06-13");
  });
});

describe("gear maintenance priority and actions", () => {
  it("sorts the priority board by severity then urgency", () => {
    const dueBike = bikeItem({
      id: "b-due",
      maintenanceTasks: [maintenanceTask({ component: "TIRE_FRONT", status: "DUE" })],
    });
    const soonUrgentBike = bikeItem({
      id: "b-soon-urgent",
      maintenanceTasks: [maintenanceTask({ component: "CHAIN", status: "SOON", distanceRemaining: 100_000 })],
    });
    const soonLaterBike = bikeItem({
      id: "b-soon-later",
      maintenanceTasks: [maintenanceTask({ component: "CASSETTE", status: "SOON", distanceRemaining: 800_000 })],
    });
    const shoe = bikeItem({
      id: "g-shoe",
      kind: "SHOE",
      maintenanceTasks: [maintenanceTask({ component: "CHAIN", status: "OVERDUE" })],
    });

    const priority = buildMaintenancePriorityItems([soonLaterBike, shoe, soonUrgentBike, dueBike]);

    expect(priority.map((entry) => `${entry.item.id}:${entry.task.component}`)).toEqual([
      "b-due:TIRE_FRONT",
      "b-soon-urgent:CHAIN",
      "b-soon-later:CASSETTE",
    ]);
  });

  it("builds explicit service and replacement operations", () => {
    expect(maintenanceOperationForAction("Chain", "SERVICE")).toBe("Chain serviced");
    expect(maintenanceOperationForAction("Chain", "REPLACEMENT")).toBe("Chain replaced");
  });
});

function bikeItem(overrides: Partial<GearAnalysisItem> = {}): GearAnalysisItem {
  return {
    id: "b123",
    name: "Road Bike",
    kind: "BIKE",
    retired: false,
    primary: true,
    maintenanceStatus: "OK",
    maintenanceLabel: "OK",
    maintenanceTasks: [],
    maintenanceHistory: [],
    distance: 0,
    totalDistance: 0,
    movingTime: 0,
    elevationGain: 0,
    activities: 0,
    averageSpeed: 0,
    firstUsed: "",
    lastUsed: "",
    longestActivity: null,
    biggestElevationActivity: null,
    fastestActivity: null,
    monthlyDistance: [],
    ...overrides,
  };
}

function maintenanceTask(overrides: Partial<GearMaintenanceTask> = {}): GearMaintenanceTask {
  return {
    component: "CHAIN",
    componentLabel: "Chain",
    intervalDistance: 1_500_000,
    intervalMonths: 0,
    status: "SOON",
    statusLabel: "100 km left",
    distanceSince: 1_400_000,
    distanceRemaining: 100_000,
    nextDueDistance: 3_000_000,
    monthsSince: 0,
    monthsRemaining: 0,
    lastMaintenance: {
      id: "gm-1",
      gearId: "b123",
      gearName: "Road Bike",
      component: "CHAIN",
      componentLabel: "Chain",
      action: "SERVICE",
      operation: "Chain serviced",
      date: "2026-01-01",
      distance: 1_500_000,
      note: null,
      createdAt: "2026-01-01T00:00:00Z",
      updatedAt: "2026-01-01T00:00:00Z",
    },
    ...overrides,
  };
}
