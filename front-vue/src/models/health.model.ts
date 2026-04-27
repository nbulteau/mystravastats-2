import type { DataQualitySummary } from "@/models/data-quality.model";

export type HealthRecord = Record<string, unknown>;

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
}
