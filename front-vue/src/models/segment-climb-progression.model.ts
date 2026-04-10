import type { ActivityShort } from "./activity.model";

export interface SegmentClimbTargetSummary {
  targetId: number;
  targetName: string;
  targetType: "SEGMENT" | "CLIMB";
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

export interface SegmentClimbAttempt {
  targetId: number;
  targetName: string;
  targetType: "SEGMENT" | "CLIMB";
  activityDate: string;
  elapsedTimeSeconds: number;
  speedKph: number;
  distance: number;
  averageGrade: number;
  elevationGain: number;
  prRank?: number;
  setsNewPr: boolean;
  closeToPr: boolean;
  deltaToPr: string;
  weatherSummary?: string;
  activity: ActivityShort;
}

export interface SegmentClimbProgression {
  metric: "TIME" | "SPEED";
  targetTypeFilter: "ALL" | "SEGMENT" | "CLIMB";
  weatherContextAvailable: boolean;
  targets: SegmentClimbTargetSummary[];
  selectedTargetId?: number;
  selectedTargetType?: "SEGMENT" | "CLIMB";
  attempts: SegmentClimbAttempt[];
}
