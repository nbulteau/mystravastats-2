import type { ActivityShort } from "@/models/activity.model";

export interface SegmentTargetSummary {
  targetId: number;
  targetName: string;
  targetType: string;
  climbCategory: number;
  distance: number;
  averageGrade: number;
  attemptsCount: number;
  bestValue: string;
  latestValue: string;
  consistency: string;
  averagePacing: string;
  closeToPrCount: number;
  recentTrend: string;
}

export interface SegmentEffort {
  targetId: number;
  targetName: string;
  targetType: string;
  activityDate: string;
  elapsedTimeSeconds: number;
  movingTimeSeconds: number;
  speedKph: number;
  distance: number;
  averageGrade: number;
  elevationGain: number;
  averagePowerWatts: number;
  averageHeartRate: number;
  prRank?: number;
  personalRank?: number;
  setsNewPr: boolean;
  closeToPr: boolean;
  deltaToPr: string;
  weatherSummary?: string;
  activity: ActivityShort;
}

export interface SegmentSummary {
  metric: string;
  segment: SegmentTargetSummary;
  personalRecord?: SegmentEffort;
  topEfforts: SegmentEffort[];
}
