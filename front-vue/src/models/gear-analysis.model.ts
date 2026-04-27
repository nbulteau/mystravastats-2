import type { ActivityShort } from "@/models/activity.model";

export type GearKind = "BIKE" | "SHOE" | "UNKNOWN";
export type GearMaintenanceStatus = "OK" | "SOON" | "DUE" | "OVERDUE";

export interface GearAnalysis {
  items: GearAnalysisItem[];
  unassigned: GearAnalysisSummary;
  coverage: GearAnalysisCoverage;
}

export interface GearAnalysisItem extends GearAnalysisSummary {
  id: string;
  name: string;
  kind: GearKind;
  retired: boolean;
  primary: boolean;
  maintenanceStatus: GearMaintenanceStatus;
  maintenanceLabel: string;
  maintenanceTasks: GearMaintenanceTask[];
  maintenanceHistory: GearMaintenanceRecord[];
  firstUsed: string;
  lastUsed: string;
  longestActivity?: ActivityShort | null;
  biggestElevationActivity?: ActivityShort | null;
  fastestActivity?: ActivityShort | null;
  monthlyDistance: GearAnalysisPeriodPoint[];
}

export interface GearAnalysisSummary {
  distance: number;
  movingTime: number;
  elevationGain: number;
  activities: number;
  averageSpeed: number;
}

export interface GearAnalysisCoverage {
  totalActivities: number;
  assignedActivities: number;
  unassignedActivities: number;
}

export interface GearAnalysisPeriodPoint {
  periodKey: string;
  value: number;
  activityCount: number;
}

export interface GearMaintenanceRecord {
  id: string;
  gearId: string;
  gearName: string;
  component: string;
  componentLabel: string;
  operation: string;
  date: string;
  distance: number;
  note?: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface GearMaintenanceRecordRequest {
  gearId: string;
  component: string;
  operation: string;
  date: string;
  distance: number;
  note?: string | null;
}

export interface GearMaintenanceTask {
  component: string;
  componentLabel: string;
  intervalDistance: number;
  intervalMonths: number;
  status: GearMaintenanceStatus;
  statusLabel: string;
  distanceSince: number;
  distanceRemaining: number;
  nextDueDistance: number;
  monthsSince: number;
  monthsRemaining: number;
  lastMaintenance?: GearMaintenanceRecord | null;
}

export function emptyGearAnalysis(): GearAnalysis {
  return {
    items: [],
    unassigned: {
      distance: 0,
      movingTime: 0,
      elevationGain: 0,
      activities: 0,
      averageSpeed: 0,
    },
    coverage: {
      totalActivities: 0,
      assignedActivities: 0,
      unassignedActivities: 0,
    },
  };
}
