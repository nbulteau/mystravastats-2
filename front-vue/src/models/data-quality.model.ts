export interface DataQualityIssue {
  id: string;
  source: string;
  activityId?: number | null;
  activityName?: string | null;
  activityType?: string | null;
  year?: string | null;
  filePath?: string | null;
  severity: "critical" | "warning" | "info" | string;
  category: string;
  field: string;
  message: string;
  rawValue?: string | null;
  suggestion?: string | null;
  excludedFromStats: boolean;
  excludedAt?: string | null;
  corrected?: boolean;
  correctionAppliedAt?: string | null;
  correction?: DataQualityCorrectionSuggestion | null;
}

export interface DataQualitySummary {
  status: "ok" | "warning" | "critical" | "not_applicable" | string;
  provider: string;
  issueCount: number;
  impactedActivities: number;
  excludedActivities: number;
  correctionCount?: number;
  safeCorrectionCount?: number;
  manualReviewCount?: number;
  bySeverity: Record<string, number>;
  byCategory: Record<string, number>;
  topIssues: DataQualityIssue[];
}

export interface DataQualityExclusion {
  activityId: number;
  source: string;
  activityName?: string | null;
  activityType?: string | null;
  year?: string | null;
  reason?: string | null;
  excludedAt: string;
}

export interface DataQualityReport {
  generatedAt: string;
  summary: DataQualitySummary;
  issues: DataQualityIssue[];
  exclusions: DataQualityExclusion[];
  corrections: DataQualityCorrection[];
}

export interface DataQualityCorrectionSuggestion {
  available: boolean;
  safety: "safe" | "manual" | "unsupported" | string;
  type?: string | null;
  label?: string | null;
  description?: string | null;
}

export interface DataQualityCorrectionImpact {
  distanceMetersBefore?: number | null;
  distanceMetersAfter?: number | null;
  elevationMetersBefore?: number | null;
  elevationMetersAfter?: number | null;
  maxSpeedBefore?: number | null;
  maxSpeedAfter?: number | null;
  distanceDeltaMeters: number;
  elevationDeltaMeters: number;
}

export interface DataQualityCorrection {
  id: string;
  issueId: string;
  source: string;
  activityId: number;
  activityName?: string | null;
  activityType?: string | null;
  year?: string | null;
  type: string;
  safety: "safe" | "manual" | "unsupported" | string;
  status: "active" | "reverted" | string;
  pointIndexes: number[];
  modifiedFields: string[];
  impact: DataQualityCorrectionImpact;
  reason?: string | null;
  appliedAt?: string | null;
  revertedAt?: string | null;
}

export interface DataQualityCorrectionBatchSummary {
  safeCorrectionCount: number;
  manualReviewCount: number;
  unsupportedIssueCount: number;
  activityCount: number;
  distanceDeltaMeters: number;
  elevationDeltaMeters: number;
  modifiedFields: string[];
  potentiallyImpactsRecords: boolean;
}

export interface DataQualityCorrectionPreview {
  generatedAt: string;
  mode: string;
  summary: DataQualityCorrectionBatchSummary;
  corrections: DataQualityCorrection[];
  warnings: string[];
  blockingReasons: string[];
}
