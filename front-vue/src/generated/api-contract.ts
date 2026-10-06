// Code generated from docs/api/openapi.json. DO NOT EDIT.

export interface ApiError {
  message: string;
  code: number;
  description: string;
  requestId: string;
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
  mode?: "STRAVA" | "FIT" | "GPX";
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
  commute: boolean;
  link: string;
  distance: number;
  elapsedTime: number;
  movingTime: number;
  totalElevationGain: number;
  averageSpeed: number;
  averageHeartrate: number;
  bestSpeedForDistanceFor1000m: number;
  bestElevationForDistanceFor500m: number;
  bestElevationForDistanceFor1000m: number;
  date: string;
  averageWatts: number;
  weightedAverageWatts: number;
  bestPowerFor20Minutes: number;
  bestPowerFor60Minutes: number;
  bestPowerFor20minutes: number;
  bestPowerFor60minutes: number;
  ftp: number;
  badgeEffortSeconds?: number | null;
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

export interface HeartRateZoneSettings {
  maxHr?: number | null;
  thresholdHr?: number | null;
  reserveHr?: number | null;
}

export interface DataQualityExclusionRequest {
  reason?: string | null;
}

export interface LocalDataBackup {
  version: 1;
  exportedAt: string;
  athleteId: string;
  files: Record<string, unknown>;
}

export interface LocalDataRestoreResult {
  restored: string[];
}

export interface StravaOAuthStartRequest {
  path?: string;
  clientId?: string;
  clientSecret?: string;
  useCache?: boolean;
}

export interface GenerateShapeRoutesRequest {
  shapeInputType: "draw" | "gpx" | "svg" | "polyline";
  shapeData: string;
  startPoint?: RouteCoordinate;
  routeType?: string;
  variantCount?: number;
}

export interface EditGeneratedRouteRequest {
  routeType?: string;
  controlPoints: RouteCoordinate[];
}

export interface EditGeneratedRouteResponse {
  route?: GeneratedRoute | null;
  controlPoints: RouteCoordinate[];
  diagnostics: RouteGenerationDiagnostic[];
}

export interface GearMaintenanceRequest {
  gearId?: string;
  component?: string;
  action?: string;
  operation?: string;
  date?: string;
  distance?: number;
  note?: string | null;
}

export interface SourceSyncResult {
  status: string;
  reason: string;
  message: string;
  startedAt: string;
  completedAt: string;
  durationMs: number;
  reloaded: boolean;
  fit: Record<string, unknown>;
}

export interface DetailedActivity {
  averageCadence: number;
  averageHeartrate: number;
  averageWatts: number;
  averageSpeed: number;
  calories: number;
  commute: boolean;
  deviceWatts: boolean;
  distance: number;
  elapsedTime: number;
  elevHigh: number;
  id: number;
  link: string;
  kilojoules: number;
  maxHeartrate: number;
  maxSpeed: number;
  maxWatts: number;
  movingTime: number;
  name: string;
  activityEfforts: ActivityEffort[];
  stravaSegmentEfforts: StravaSegmentEffort[];
  activityComparison?: ActivityComparison | null;
  startDate: string;
  startDateLocal: string;
  startLatlng: number[];
  source?: ActivitySource | null;
  stream: ActivityStream | null;
  sufferScore: number | null;
  totalDescent: number;
  totalElevationGain: number;
  type: string;
  sportType: string;
  weightedAverageWatts: number;
}

export interface ActivitySource {
  primaryProvider: string;
  primaryId: number;
  streamProvider?: string | null;
  mergeConfidence?: string | null;
  sources?: ActivitySourceRef[] | null;
  conflicts?: ActivitySourceConflict[] | null;
  fieldSources?: Record<string, string> | null;
}

export interface ActivitySourceRef {
  provider: string;
  activityId: number;
  startDateLocal: string;
  distance: number;
  movingTime: number;
  hasStream: boolean;
}

export interface ActivitySourceConflict {
  field: string;
  primary: string;
  other: string;
  source: string;
}

export interface ActivityComparison {
  status: string;
  label: string;
  criteria: ActivityComparisonCriteria;
  baseline: ActivityComparisonBaseline;
  deltas: ActivityComparisonDeltas;
  similarActivities: ActivityComparisonActivity[];
  commonSegments: ActivityComparisonSegment[];
}

export interface ActivityComparisonCriteria {
  activityType: string;
  year: number;
  sampleSize: number;
}

export interface ActivityComparisonBaseline {
  distance: number;
  elevationGain: number;
  movingTime: number;
  averageSpeed: number;
  averageHeartrate: number;
  averageWatts: number;
  averageCadence: number;
}

export interface ActivityComparisonDeltas {
  distance: number;
  elevationGain: number;
  movingTime: number;
  averageSpeed: number;
  averageSpeedPct: number;
  averageHeartrate: number;
  averageWatts: number;
  averageCadence: number;
}

export interface ActivityComparisonActivity {
  id: number;
  name: string;
  date: string;
  distance: number;
  elevationGain: number;
  movingTime: number;
  averageSpeed: number;
  averageHeartrate: number;
  averageWatts: number;
  averageCadence: number;
  similarityScore: number;
}

export interface ActivityComparisonSegment {
  id: number;
  name: string;
  matchCount: number;
  activityIds: number[];
  activityNames: string[];
}

export interface ActivityEffort {
  id: string;
  label: string;
  distance: number;
  seconds: number;
  deltaAltitude: number;
  elevationGain?: number | null;
  elevationLoss?: number | null;
  idxStart: number;
  idxEnd: number;
  averagePower: number | null;
  description: string;
}

export interface StravaSegmentEffort {
  averageCadence: number;
  averageHeartRate: number;
  averageWatts: number;
  deviceWatts: boolean;
  distance: number;
  elapsedTime: number;
  endIndex: number;
  hidden: boolean;
  id: number;
  komRank?: number | null;
  maxHeartRate: number;
  movingTime: number;
  name: string;
  prRank?: number | null;
  resourceState: number;
  segment: StravaSegment;
  startDate: string;
  startDateLocal: string;
  startIndex: number;
  visibility?: string | null;
  achievements?: Achievement[];
  activity?: number;
  athlete?: number;
}

export interface StravaSegment {
  activityType: string;
  averageGrade: number;
  city?: string | null;
  climbCategory: number;
  country?: string | null;
  distance: number;
  elevationHigh: number;
  elevationLow: number;
  endLatLng: number[];
  hazardous: boolean;
  id: number;
  maximumGrade: number;
  name: string;
  isPrivate: boolean;
  resourceState: number;
  starred: boolean;
  startLatLng: number[];
  state?: string | null;
}

export interface ActivityStream {
  distance: number[];
  time: number[];
  latlng?: number[][] | null;
  heartrate?: number[] | null;
  cadence?: number[] | null;
  moving?: boolean[] | null;
  altitude?: number[] | null;
  watts?: (number | null)[] | null;
  velocitySmooth?: number[] | null;
}

export interface Statistic {
  label: string;
  value: string;
  activity?: ActivityShort;
}

export interface ActivityShort {
  id: number;
  name: string;
  type: string;
}

export interface PersonalRecordTimeline {
  metricKey: string;
  metricLabel: string;
  activityDate: string;
  value: string;
  previousValue?: string | null;
  improvement?: string | null;
  activity: ActivityShort;
}

export interface SegmentClimbProgression {
  metric: string;
  targetTypeFilter: string;
  weatherContextAvailable: boolean;
  targets: SegmentClimbTargetSummary[];
  selectedTargetId?: number | null;
  selectedTargetType?: string | null;
  attempts: SegmentClimbAttempt[];
}

export interface SegmentClimbTargetSummary {
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

export interface SegmentClimbAttempt {
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
  prRank?: number | null;
  personalRank?: number | null;
  setsNewPr: boolean;
  closeToPr: boolean;
  deltaToPr: string;
  weatherSummary?: string | null;
  activity: ActivityShort;
}

export interface SegmentSummary {
  metric: string;
  segment: SegmentClimbTargetSummary;
  personalRecord?: SegmentClimbAttempt | null;
  topEfforts: SegmentClimbAttempt[];
}

export interface ResolvedHeartRateZoneSettings {
  maxHr: number;
  thresholdHr?: number | null;
  reserveHr?: number | null;
  method: string;
  source: string;
}

export interface HeartRateZoneDistribution {
  zone: string;
  label: string;
  seconds: number;
  percentage: number;
}

export interface HeartRateZoneActivitySummary {
  activity: ActivityShort;
  activityDate: string;
  totalTrackedSeconds: number;
  easySeconds: number;
  hardSeconds: number;
  easyHardRatio?: number | null;
  zones: HeartRateZoneDistribution[];
}

export interface HeartRateZonePeriodSummary {
  period: string;
  totalTrackedSeconds: number;
  easySeconds: number;
  hardSeconds: number;
  easyHardRatio?: number | null;
  zones: HeartRateZoneDistribution[];
}

export interface HeartRateZoneAnalysis {
  settings: HeartRateZoneSettings;
  resolvedSettings?: ResolvedHeartRateZoneSettings | null;
  hasHeartRateData: boolean;
  totalTrackedSeconds: number;
  easyHardRatio?: number | null;
  zones: HeartRateZoneDistribution[];
  activities: HeartRateZoneActivitySummary[];
  byMonth: HeartRateZonePeriodSummary[];
  byYear: HeartRateZonePeriodSummary[];
}

export interface FtpEstimate {
  available: boolean;
  ftp: number;
  method: string;
  methodLabel: string;
  bestPower: number;
  multiplier: number;
  basedOnSeconds: number;
  confidence: "high" | "medium" | "low" | "unavailable";
  source: string;
  sourceKind: string;
  activityId: number;
  activityName: string;
  activityType: string;
  activityDate: string;
  windowDays: number;
  activityCount: number;
}

export interface Achievement {
  effortCount: number;
  rank: number;
  type: string;
  typeId: number;
}

export interface TrainingLoadActivity {
  activityId: number;
  name: string;
  date: string;
  sport: string;
  source: "measured" | "estimated" | "unavailable";
  reason: "available" | "unsupported-sport" | "missing-power" | "invalid-time" | "power-gaps" | "too-short" | "incomplete-activity" | "missing-dated-ftp";
  load: number | null;
  ftp: number | null;
  ftpEffectiveFrom: string | null;
  normalizedPower: number | null;
  intensityFactor: number | null;
  best5MinutePower: number | null;
  movingSeconds: number;
  elapsedSeconds: number;
  recordedSeconds: number;
  coveredSeconds: number;
  distanceMeters: number;
  elevationMeters: number;
  aerobicSeconds: number;
  thresholdSeconds: number;
  highIntensitySeconds: number;
}

export interface TrainingLoadPeriod {
  startDate: string;
  endDate: string;
  activityCount: number;
  scoredCount: number;
  measuredLoad: number | null;
  estimatedLoad: number | null;
  movingSeconds: number;
  distanceMeters: number;
  elevationMeters: number;
  coveredSeconds: number;
  recordedSeconds: number;
  aerobicSeconds: number;
  thresholdSeconds: number;
  highIntensitySeconds: number;
  best5MinutePower: number | null;
  best5MinuteActivityId: number | null;
}

export interface TrainingLoadWeek {
  summary: TrainingLoadPeriod;
  days: TrainingLoadPeriod[];
}

export interface TrainingReport {
  method: string;
  maxGapSeconds: number;
  undatedActivities: number;
  weeks: TrainingLoadWeek[];
  activities: TrainingLoadActivity[];
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
  revertDataQualityCorrection: { method: "DELETE", path: "/api/data-quality/corrections/{id}" },
  applyDataQualityCorrection: { method: "POST", path: "/api/data-quality/corrections/{id}" },
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
  getTrainingLoad: { method: "GET", path: "/api/statistics/training-load" },
} as const;

export type ApiOperationId = keyof typeof apiOperations;
