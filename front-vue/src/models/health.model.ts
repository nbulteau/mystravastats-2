import type { DataQualitySummary } from "@/models/data-quality.model";

export type HealthRecord = Record<string, unknown>;

export interface CompositeSourceSummary extends HealthRecord {
  provider?: string;
  athleteId?: string | number;
  cacheRoot?: string;
  activities?: number;
  availableYearBins?: Array<string | number>;
}

export interface CompositeMergeConflict extends HealthRecord {
  field?: string;
  primary?: string;
  other?: string;
  source?: string;
}

export interface CompositeDiagnostics extends HealthRecord {
  active?: boolean;
  activeProviders?: string[];
  sources?: CompositeSourceSummary[];
  matchedActivities?: number;
  localOnlyActivities?: number;
  conflictCount?: number;
  conflictSamples?: CompositeMergeConflict[];
  futureProviders?: string[];
}

export interface ImportedFITFile extends HealthRecord {
  source?: string;
  destination?: string;
  year?: string;
  activityId?: number;
  startDate?: string;
}

export interface FITImportResult extends HealthRecord {
  status?: string;
  message?: string;
  configured?: boolean;
  sourcePath?: string;
  candidateSourcePaths?: string[];
  destinationPath?: string;
  scannedFiles?: number;
  importedFiles?: number;
  alreadyPresentFiles?: number;
  skippedFiles?: number;
  invalidFiles?: number;
  createdYearDirectories?: string[];
  imported?: ImportedFITFile[];
  errors?: string[];
}

export interface SourceSyncResult extends HealthRecord {
  status?: string;
  reason?: string;
  message?: string;
  startedAt?: string;
  completedAt?: string;
  durationMs?: number;
  reloaded?: boolean;
  fit?: FITImportResult;
}

export interface HealthDetailsPayload extends HealthRecord {
  timestamp?: string;
  provider?: string;
  athleteId?: string | number;
  cacheRoot?: string;
  fitDirectory?: string;
  gpxDirectory?: string;
  activities?: number;
  availableYearBins?: Array<string | number>;
  refresh?: HealthRecord;
  rateLimit?: HealthRecord;
  manifest?: HealthRecord;
  files?: HealthRecord;
  routing?: HealthRecord;
  runtimeConfig?: HealthRecord;
  dataQuality?: DataQualitySummary;
  composite?: CompositeDiagnostics;
  sourceSync?: SourceSyncResult;
}
