// Code generated from docs/api/openapi.json. DO NOT EDIT.

export interface ApiError {
  message: string;
  code: number;
  description?: string;
  path?: string | null;
  requestId?: string | null;
}

export interface RouteCoordinate {
  lat: number;
  lng: number;
}

export interface RouteGenerationDiagnostic {
  code: string;
  message: string;
}

export interface SourceModeSelection {
  mode: "STRAVA" | "FIT" | "GPX";
  path?: string | null;
}

export interface OperationStatus {
  success: boolean;
  message?: string | null;
}

export interface ActivitySummary {
  id: number;
  name: string;
  type: string;
  date?: string | null;
  distance?: number | null;
}

export interface RouteGenerationScore {
  global: number;
  distance: number;
  elevation: number;
  duration: number;
  direction: number;
  shape: number;
  roadFitness: number;
}

export interface GeneratedRoute {
  routeId: string;
  title: string;
  variantType: string;
  routeType?: string | null;
  distanceKm: number;
  elevationGainM: number;
  durationSec: number;
  estimatedDurationSec: number;
  score: RouteGenerationScore;
  reasons: string[];
  previewLatLng: number[][];
  start?: RouteCoordinate | null;
  end?: RouteCoordinate | null;
  activityId?: number | null;
  isRoadGraphGenerated: boolean;
}

export interface GenerateRoutesResponse {
  routes: GeneratedRoute[];
  diagnostics?: RouteGenerationDiagnostic[];
}

export interface AthleteFtpSetting {
  effectiveFrom: string;
  ftp: number;
}

export interface AthletePerformanceSettings {
  ftpHistory: AthleteFtpSetting[];
  weightKg?: number | null;
}

export interface DataQualityIssue {
  id: string;
  source: string;
  activityId?: number | null;
  activityName?: string | null;
  severity: string;
  category: string;
  field: string;
  message: string;
  excludedFromStats?: boolean;
}

export interface DataQualitySummary {
  status: string;
  provider?: string | null;
  issueCount: number;
  impactedActivities: number;
  excludedActivities: number;
  safeCorrectionCount?: number;
  manualReviewCount?: number;
}

export interface DataQualityReport {
  generatedAt?: string | null;
  summary: DataQualitySummary;
  issues: DataQualityIssue[];
}

export const apiOperations = {
  listActivities: { method: "GET", path: "/api/activities" },
  getActivity: { method: "GET", path: "/api/activities/{activityId}" },
  exportActivitiesCsv: { method: "GET", path: "/api/activities/csv" },
  getCurrentAthlete: { method: "GET", path: "/api/athletes/me" },
  getFtpEstimate: { method: "GET", path: "/api/athletes/me/ftp-estimate" },
  getHeartRateZones: { method: "GET", path: "/api/athletes/me/heart-rate-zones" },
  updateHeartRateZones: { method: "PUT", path: "/api/athletes/me/heart-rate-zones" },
  getPerformanceSettings: { method: "GET", path: "/api/athletes/me/performance-settings" },
  updatePerformanceSettings: { method: "PUT", path: "/api/athletes/me/performance-settings" },
  getBadges: { method: "GET", path: "/api/badges" },
  getAverageCadenceChart: { method: "GET", path: "/api/charts/average-cadence-by-period" },
  getAverageSpeedChart: { method: "GET", path: "/api/charts/average-speed-by-period" },
  getDistanceChart: { method: "GET", path: "/api/charts/distance-by-period" },
  getElevationChart: { method: "GET", path: "/api/charts/elevation-by-period" },
  getDashboard: { method: "GET", path: "/api/dashboard" },
  getActivityHeatmap: { method: "GET", path: "/api/dashboard/activity-heatmap" },
  getCumulativeData: { method: "GET", path: "/api/dashboard/cumulative-data-per-year" },
  getEddingtonNumber: { method: "GET", path: "/api/dashboard/eddington-number" },
  revertDataQualityCorrection: { method: "DELETE", path: "/api/data-quality/corrections/{correctionId}" },
  applyDataQualityCorrection: { method: "POST", path: "/api/data-quality/corrections/{issueId}" },
  previewDataQualityCorrection: { method: "GET", path: "/api/data-quality/corrections/preview/{issueId}" },
  applySafeDataQualityCorrections: { method: "POST", path: "/api/data-quality/corrections/safe" },
  previewSafeDataQualityCorrections: { method: "GET", path: "/api/data-quality/corrections/safe/preview" },
  includeActivityInStatistics: { method: "DELETE", path: "/api/data-quality/exclusions/{activityId}" },
  excludeActivityFromStatistics: { method: "PUT", path: "/api/data-quality/exclusions/{activityId}" },
  getDataQualityIssues: { method: "GET", path: "/api/data-quality/issues" },
  getGearAnalysis: { method: "GET", path: "/api/gear-analysis" },
  createGearMaintenance: { method: "POST", path: "/api/gear-analysis/maintenance" },
  deleteGearMaintenance: { method: "DELETE", path: "/api/gear-analysis/maintenance/{recordId}" },
  getHealthDetails: { method: "GET", path: "/api/health/details" },
  getLocalDataBackup: { method: "GET", path: "/api/local-data/backup" },
  restoreLocalData: { method: "POST", path: "/api/local-data/restore" },
  getMapTracks: { method: "GET", path: "/api/maps/gpx" },
  getMapPassages: { method: "GET", path: "/api/maps/passages" },
  editGeneratedRoute: { method: "POST", path: "/api/routes/{routeId}/edit" },
  exportGeneratedRouteGpx: { method: "GET", path: "/api/routes/{routeId}/gpx" },
  generateShapeRoutes: { method: "POST", path: "/api/routes/generate/shape" },
  getRouteRecommendations: { method: "GET", path: "/api/routes/recommendations" },
  exportRouteRecommendationGpx: { method: "GET", path: "/api/routes/recommendations/gpx" },
  startOsrm: { method: "POST", path: "/api/routing/osrm/start" },
  listSegments: { method: "GET", path: "/api/segments" },
  listSegmentEfforts: { method: "GET", path: "/api/segments/{segmentId}/efforts" },
  getSegmentSummary: { method: "GET", path: "/api/segments/{segmentId}/summary" },
  applySourceMode: { method: "POST", path: "/api/source-modes/apply" },
  previewSourceMode: { method: "POST", path: "/api/source-modes/preview" },
  completeStravaOAuth: { method: "GET", path: "/api/source-modes/strava/oauth/callback" },
  startStravaOAuth: { method: "POST", path: "/api/source-modes/strava/oauth/start" },
  synchronizeSources: { method: "POST", path: "/api/source-sync/synchronize" },
  getStatistics: { method: "GET", path: "/api/statistics" },
  getHeartRateZoneAnalysis: { method: "GET", path: "/api/statistics/heart-rate-zones" },
  getPersonalRecordsTimeline: { method: "GET", path: "/api/statistics/personal-records-timeline" },
  getSegmentClimbProgression: { method: "GET", path: "/api/statistics/segment-climb-progression" },
} as const;

export type ApiOperationId = keyof typeof apiOperations;
