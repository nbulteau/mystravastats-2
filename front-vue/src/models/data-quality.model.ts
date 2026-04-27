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
}

export interface DataQualitySummary {
  status: "ok" | "warning" | "critical" | "not_applicable" | string;
  provider: string;
  issueCount: number;
  impactedActivities: number;
  excludedActivities: number;
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
}
