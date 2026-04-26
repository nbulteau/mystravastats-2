import type { ActivityShort } from "@/models/activity.model";

export type GearKind = "BIKE" | "SHOE" | "UNKNOWN";

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
  maintenanceStatus: "OK" | "WATCH" | "REVIEW";
  maintenanceLabel: string;
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
