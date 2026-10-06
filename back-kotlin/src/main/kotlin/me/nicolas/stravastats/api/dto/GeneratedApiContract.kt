// Code generated from docs/api/openapi.json. DO NOT EDIT.

package me.nicolas.stravastats.api.dto

data class ContractApiError(
    val message: String,
    val code: Long,
    val description: String,
    val requestId: String,
)

data class ContractRouteCoordinate(
    val lat: Double,
    val lng: Double,
)

data class ContractRouteGenerationDiagnostic(
    val code: String,
    val message: String,
)

data class ContractSourceModeSelection(
    val mode: String? = null,
    val path: String? = null,
)

data class ContractOperationStatus(
    val success: Boolean,
    val message: String? = null,
)

data class ContractActivitySummary(
    val id: Long,
    val name: String,
    val type: String,
    val commute: Boolean,
    val link: String,
    val distance: Long,
    val elapsedTime: Long,
    val movingTime: Long,
    val totalElevationGain: Long,
    val averageSpeed: Double,
    val averageHeartrate: Long,
    val bestSpeedForDistanceFor1000m: Double,
    val bestElevationForDistanceFor500m: Double,
    val bestElevationForDistanceFor1000m: Double,
    val date: String,
    val averageWatts: Long,
    val weightedAverageWatts: Long,
    val bestPowerFor20Minutes: Long,
    val bestPowerFor60Minutes: Long,
    val bestPowerFor20minutes: Long,
    val bestPowerFor60minutes: Long,
    val ftp: Long,
    val badgeEffortSeconds: Long? = null,
)

data class ContractRouteGenerationScore(
    val global: Double,
    val distance: Double,
    val elevation: Double,
    val duration: Double,
    val direction: Double,
    val shape: Double,
    val roadFitness: Double,
)

data class ContractGeneratedRoute(
    val routeId: String,
    val title: String,
    val variantType: String,
    val routeType: String? = null,
    val distanceKm: Double,
    val elevationGainM: Double,
    val durationSec: Long,
    val estimatedDurationSec: Long,
    val score: ContractRouteGenerationScore,
    val reasons: List<String>,
    val previewLatLng: List<List<Double>>,
    val start: ContractRouteCoordinate? = null,
    val end: ContractRouteCoordinate? = null,
    val activityId: Long? = null,
    val isRoadGraphGenerated: Boolean,
)

data class ContractGenerateRoutesResponse(
    val routes: List<ContractGeneratedRoute>,
    val diagnostics: List<ContractRouteGenerationDiagnostic>? = null,
)

data class ContractAthleteFtpSetting(
    val effectiveFrom: String,
    val ftp: Long,
)

data class ContractAthletePerformanceSettings(
    val ftpHistory: List<ContractAthleteFtpSetting>,
    val weightKg: Double? = null,
)

data class ContractDataQualityIssue(
    val id: String,
    val source: String,
    val activityId: Long? = null,
    val activityName: String? = null,
    val severity: String,
    val category: String,
    val field: String,
    val message: String,
    val excludedFromStats: Boolean? = null,
)

data class ContractDataQualitySummary(
    val status: String,
    val provider: String? = null,
    val issueCount: Long,
    val impactedActivities: Long,
    val excludedActivities: Long,
    val safeCorrectionCount: Long? = null,
    val manualReviewCount: Long? = null,
)

data class ContractDataQualityReport(
    val generatedAt: String? = null,
    val summary: ContractDataQualitySummary,
    val issues: List<ContractDataQualityIssue>,
)

data class ContractHeartRateZoneSettings(
    val maxHr: Long? = null,
    val thresholdHr: Long? = null,
    val reserveHr: Long? = null,
)

data class ContractDataQualityExclusionRequest(
    val reason: String? = null,
)

data class ContractLocalDataBackup(
    val version: Long,
    val exportedAt: String,
    val athleteId: String,
    val files: Map<String, Any>,
)

data class ContractLocalDataRestoreResult(
    val restored: List<String>,
)

data class ContractStravaOAuthStartRequest(
    val path: String? = null,
    val clientId: String? = null,
    val clientSecret: String? = null,
    val useCache: Boolean? = null,
)

data class ContractGenerateShapeRoutesRequest(
    val shapeInputType: String,
    val shapeData: String,
    val startPoint: ContractRouteCoordinate? = null,
    val routeType: String? = null,
    val variantCount: Long? = null,
)

data class ContractEditGeneratedRouteRequest(
    val routeType: String? = null,
    val controlPoints: List<ContractRouteCoordinate>,
)

data class ContractEditGeneratedRouteResponse(
    val route: ContractGeneratedRoute? = null,
    val controlPoints: List<ContractRouteCoordinate>,
    val diagnostics: List<ContractRouteGenerationDiagnostic>,
)

data class ContractGearMaintenanceRequest(
    val gearId: String? = null,
    val component: String? = null,
    val action: String? = null,
    val operation: String? = null,
    val date: String? = null,
    val distance: Double? = null,
    val note: String? = null,
)

data class ContractSourceSyncResult(
    val status: String,
    val reason: String,
    val message: String,
    val startedAt: String,
    val completedAt: String,
    val durationMs: Long,
    val reloaded: Boolean,
    val fit: Map<String, Any>,
)

data class ContractDetailedActivity(
    val averageCadence: Long,
    val averageHeartrate: Long,
    val averageWatts: Long,
    val averageSpeed: Double,
    val calories: Double,
    val commute: Boolean,
    val deviceWatts: Boolean,
    val distance: Double,
    val elapsedTime: Long,
    val elevHigh: Double,
    val id: Long,
    val link: String,
    val kilojoules: Double,
    val maxHeartrate: Long,
    val maxSpeed: Double,
    val maxWatts: Long,
    val movingTime: Long,
    val name: String,
    val activityEfforts: List<ContractActivityEffort>,
    val stravaSegmentEfforts: List<ContractStravaSegmentEffort>,
    val activityComparison: ContractActivityComparison? = null,
    val startDate: String,
    val startDateLocal: String,
    val startLatlng: List<Double>,
    val source: ContractActivitySource? = null,
    val stream: ContractActivityStream? = null,
    val sufferScore: Double? = null,
    val totalDescent: Double,
    val totalElevationGain: Long,
    val type: String,
    val sportType: String,
    val weightedAverageWatts: Long,
)

data class ContractActivitySource(
    val primaryProvider: String,
    val primaryId: Long,
    val streamProvider: String? = null,
    val mergeConfidence: String? = null,
    val sources: List<ContractActivitySourceRef>? = null,
    val conflicts: List<ContractActivitySourceConflict>? = null,
    val fieldSources: Map<String, String>? = null,
)

data class ContractActivitySourceRef(
    val provider: String,
    val activityId: Long,
    val startDateLocal: String,
    val distance: Double,
    val movingTime: Long,
    val hasStream: Boolean,
)

data class ContractActivitySourceConflict(
    val field: String,
    val primary: String,
    val other: String,
    val source: String,
)

data class ContractActivityComparison(
    val status: String,
    val label: String,
    val criteria: ContractActivityComparisonCriteria,
    val baseline: ContractActivityComparisonBaseline,
    val deltas: ContractActivityComparisonDeltas,
    val similarActivities: List<ContractActivityComparisonActivity>,
    val commonSegments: List<ContractActivityComparisonSegment>,
)

data class ContractActivityComparisonCriteria(
    val activityType: String,
    val year: Long,
    val sampleSize: Long,
)

data class ContractActivityComparisonBaseline(
    val distance: Double,
    val elevationGain: Double,
    val movingTime: Long,
    val averageSpeed: Double,
    val averageHeartrate: Double,
    val averageWatts: Double,
    val averageCadence: Double,
)

data class ContractActivityComparisonDeltas(
    val distance: Double,
    val elevationGain: Double,
    val movingTime: Long,
    val averageSpeed: Double,
    val averageSpeedPct: Double,
    val averageHeartrate: Double,
    val averageWatts: Double,
    val averageCadence: Double,
)

data class ContractActivityComparisonActivity(
    val id: Long,
    val name: String,
    val date: String,
    val distance: Double,
    val elevationGain: Double,
    val movingTime: Long,
    val averageSpeed: Double,
    val averageHeartrate: Double,
    val averageWatts: Double,
    val averageCadence: Double,
    val similarityScore: Double,
)

data class ContractActivityComparisonSegment(
    val id: Long,
    val name: String,
    val matchCount: Long,
    val activityIds: List<Long>,
    val activityNames: List<String>,
)

data class ContractActivityEffort(
    val id: String,
    val label: String,
    val distance: Double,
    val seconds: Long,
    val deltaAltitude: Double,
    val elevationGain: Double? = null,
    val elevationLoss: Double? = null,
    val idxStart: Long,
    val idxEnd: Long,
    val averagePower: Double? = null,
    val description: String,
)

data class ContractStravaSegmentEffort(
    val averageCadence: Double,
    val averageHeartRate: Double,
    val averageWatts: Double,
    val deviceWatts: Boolean,
    val distance: Double,
    val elapsedTime: Long,
    val endIndex: Long,
    val hidden: Boolean,
    val id: Long,
    val komRank: Long? = null,
    val maxHeartRate: Double,
    val movingTime: Long,
    val name: String,
    val prRank: Long? = null,
    val resourceState: Long,
    val segment: ContractStravaSegment,
    val startDate: String,
    val startDateLocal: String,
    val startIndex: Long,
    val visibility: String? = null,
    val achievements: List<ContractAchievement>? = null,
    val activity: Long? = null,
    val athlete: Long? = null,
)

data class ContractStravaSegment(
    val activityType: String,
    val averageGrade: Double,
    val city: String? = null,
    val climbCategory: Long,
    val country: String? = null,
    val distance: Double,
    val elevationHigh: Double,
    val elevationLow: Double,
    val endLatLng: List<Double>,
    val hazardous: Boolean,
    val id: Long,
    val maximumGrade: Double,
    val name: String,
    val isPrivate: Boolean,
    val resourceState: Long,
    val starred: Boolean,
    val startLatLng: List<Double>,
    val state: String? = null,
)

data class ContractActivityStream(
    val distance: List<Double>,
    val time: List<Long>,
    val latlng: List<List<Double>>? = null,
    val heartrate: List<Long>? = null,
    val cadence: List<Long>? = null,
    val moving: List<Boolean>? = null,
    val altitude: List<Double>? = null,
    val watts: List<Double?>? = null,
    val velocitySmooth: List<Double>? = null,
)

data class ContractStatistic(
    val label: String,
    val value: String,
    val activity: ContractActivityShort? = null,
)

data class ContractActivityShort(
    val id: Long,
    val name: String,
    val type: String,
)

data class ContractPersonalRecordTimeline(
    val metricKey: String,
    val metricLabel: String,
    val activityDate: String,
    val value: String,
    val previousValue: String? = null,
    val improvement: String? = null,
    val activity: ContractActivityShort,
)

data class ContractSegmentClimbProgression(
    val metric: String,
    val targetTypeFilter: String,
    val weatherContextAvailable: Boolean,
    val targets: List<ContractSegmentClimbTargetSummary>,
    val selectedTargetId: Long? = null,
    val selectedTargetType: String? = null,
    val attempts: List<ContractSegmentClimbAttempt>,
)

data class ContractSegmentClimbTargetSummary(
    val targetId: Long,
    val targetName: String,
    val targetType: String,
    val climbCategory: Long,
    val distance: Double,
    val averageGrade: Double,
    val attemptsCount: Long,
    val bestValue: String,
    val latestValue: String,
    val consistency: String,
    val averagePacing: String,
    val closeToPrCount: Long,
    val recentTrend: String,
)

data class ContractSegmentClimbAttempt(
    val targetId: Long,
    val targetName: String,
    val targetType: String,
    val activityDate: String,
    val elapsedTimeSeconds: Long,
    val movingTimeSeconds: Long,
    val speedKph: Double,
    val distance: Double,
    val averageGrade: Double,
    val elevationGain: Double,
    val averagePowerWatts: Double,
    val averageHeartRate: Double,
    val prRank: Long? = null,
    val personalRank: Long? = null,
    val setsNewPr: Boolean,
    val closeToPr: Boolean,
    val deltaToPr: String,
    val weatherSummary: String? = null,
    val activity: ContractActivityShort,
)

data class ContractSegmentSummary(
    val metric: String,
    val segment: ContractSegmentClimbTargetSummary,
    val personalRecord: ContractSegmentClimbAttempt? = null,
    val topEfforts: List<ContractSegmentClimbAttempt>,
)

data class ContractResolvedHeartRateZoneSettings(
    val maxHr: Long,
    val thresholdHr: Long? = null,
    val reserveHr: Long? = null,
    val method: String,
    val source: String,
)

data class ContractHeartRateZoneDistribution(
    val zone: String,
    val label: String,
    val seconds: Long,
    val percentage: Double,
)

data class ContractHeartRateZoneActivitySummary(
    val activity: ContractActivityShort,
    val activityDate: String,
    val totalTrackedSeconds: Long,
    val easySeconds: Long,
    val hardSeconds: Long,
    val easyHardRatio: Double? = null,
    val zones: List<ContractHeartRateZoneDistribution>,
)

data class ContractHeartRateZonePeriodSummary(
    val period: String,
    val totalTrackedSeconds: Long,
    val easySeconds: Long,
    val hardSeconds: Long,
    val easyHardRatio: Double? = null,
    val zones: List<ContractHeartRateZoneDistribution>,
)

data class ContractHeartRateZoneAnalysis(
    val settings: ContractHeartRateZoneSettings,
    val resolvedSettings: ContractResolvedHeartRateZoneSettings? = null,
    val hasHeartRateData: Boolean,
    val totalTrackedSeconds: Long,
    val easyHardRatio: Double? = null,
    val zones: List<ContractHeartRateZoneDistribution>,
    val activities: List<ContractHeartRateZoneActivitySummary>,
    val byMonth: List<ContractHeartRateZonePeriodSummary>,
    val byYear: List<ContractHeartRateZonePeriodSummary>,
)

data class ContractFtpEstimate(
    val available: Boolean,
    val ftp: Long,
    val method: String,
    val methodLabel: String,
    val bestPower: Long,
    val multiplier: Double,
    val basedOnSeconds: Long,
    val confidence: String,
    val source: String,
    val sourceKind: String,
    val activityId: Long,
    val activityName: String,
    val activityType: String,
    val activityDate: String,
    val windowDays: Long,
    val activityCount: Long,
)

data class ContractAchievement(
    val effortCount: Long,
    val rank: Long,
    val type: String,
    val typeId: Long,
)

data class ContractTrainingLoadActivity(
    val activityId: Long,
    val name: String,
    val date: String,
    val sport: String,
    val source: String,
    val reason: String,
    val load: Double? = null,
    val ftp: Long? = null,
    val ftpEffectiveFrom: String? = null,
    val normalizedPower: Double? = null,
    val intensityFactor: Double? = null,
    val best5MinutePower: Double? = null,
    val movingSeconds: Long,
    val elapsedSeconds: Long,
    val recordedSeconds: Double,
    val coveredSeconds: Double,
    val distanceMeters: Double,
    val elevationMeters: Double,
    val aerobicSeconds: Double,
    val thresholdSeconds: Double,
    val highIntensitySeconds: Double,
)

data class ContractTrainingLoadPeriod(
    val startDate: String,
    val endDate: String,
    val activityCount: Long,
    val scoredCount: Long,
    val measuredLoad: Double? = null,
    val estimatedLoad: Double? = null,
    val movingSeconds: Long,
    val distanceMeters: Double,
    val elevationMeters: Double,
    val coveredSeconds: Double,
    val recordedSeconds: Double,
    val aerobicSeconds: Double,
    val thresholdSeconds: Double,
    val highIntensitySeconds: Double,
    val best5MinutePower: Double? = null,
    val best5MinuteActivityId: Long? = null,
)

data class ContractTrainingLoadWeek(
    val summary: ContractTrainingLoadPeriod,
    val days: List<ContractTrainingLoadPeriod>,
)

data class ContractTrainingReport(
    val method: String,
    val maxGapSeconds: Long,
    val undatedActivities: Long,
    val weeks: List<ContractTrainingLoadWeek>,
    val activities: List<ContractTrainingLoadActivity>,
)

data class ContractOperation(val method: String, val path: String)

val contractOperations: Map<String, ContractOperation> = mapOf(
    "listActivities" to ContractOperation("GET", "/api/activities"),
    "getActivity" to ContractOperation("GET", "/api/activities/{activityId}"),
    "exportActivitiesCsv" to ContractOperation("GET", "/api/activities/csv"),
    "getCurrentAthlete" to ContractOperation("GET", "/api/athletes/me"),
    "getFtpEstimate" to ContractOperation("GET", "/api/athletes/me/ftp-estimate"),
    "getHeartRateZones" to ContractOperation("GET", "/api/athletes/me/heart-rate-zones"),
    "updateHeartRateZones" to ContractOperation("PUT", "/api/athletes/me/heart-rate-zones"),
    "getPerformanceSettings" to ContractOperation("GET", "/api/athletes/me/performance-settings"),
    "updatePerformanceSettings" to ContractOperation("PUT", "/api/athletes/me/performance-settings"),
    "getBadges" to ContractOperation("GET", "/api/badges"),
    "getAverageCadenceChart" to ContractOperation("GET", "/api/charts/average-cadence-by-period"),
    "getAverageSpeedChart" to ContractOperation("GET", "/api/charts/average-speed-by-period"),
    "getDistanceChart" to ContractOperation("GET", "/api/charts/distance-by-period"),
    "getElevationChart" to ContractOperation("GET", "/api/charts/elevation-by-period"),
    "getDashboard" to ContractOperation("GET", "/api/dashboard"),
    "getActivityHeatmap" to ContractOperation("GET", "/api/dashboard/activity-heatmap"),
    "getCumulativeData" to ContractOperation("GET", "/api/dashboard/cumulative-data-per-year"),
    "getEddingtonNumber" to ContractOperation("GET", "/api/dashboard/eddington-number"),
    "revertDataQualityCorrection" to ContractOperation("DELETE", "/api/data-quality/corrections/{id}"),
    "applyDataQualityCorrection" to ContractOperation("POST", "/api/data-quality/corrections/{id}"),
    "previewDataQualityCorrection" to ContractOperation("GET", "/api/data-quality/corrections/preview/{issueId}"),
    "applySafeDataQualityCorrections" to ContractOperation("POST", "/api/data-quality/corrections/safe"),
    "previewSafeDataQualityCorrections" to ContractOperation("GET", "/api/data-quality/corrections/safe/preview"),
    "includeActivityInStatistics" to ContractOperation("DELETE", "/api/data-quality/exclusions/{activityId}"),
    "excludeActivityFromStatistics" to ContractOperation("PUT", "/api/data-quality/exclusions/{activityId}"),
    "getDataQualityIssues" to ContractOperation("GET", "/api/data-quality/issues"),
    "getGearAnalysis" to ContractOperation("GET", "/api/gear-analysis"),
    "createGearMaintenance" to ContractOperation("POST", "/api/gear-analysis/maintenance"),
    "deleteGearMaintenance" to ContractOperation("DELETE", "/api/gear-analysis/maintenance/{recordId}"),
    "getHealthDetails" to ContractOperation("GET", "/api/health/details"),
    "getLocalDataBackup" to ContractOperation("GET", "/api/local-data/backup"),
    "restoreLocalData" to ContractOperation("POST", "/api/local-data/restore"),
    "getMapTracks" to ContractOperation("GET", "/api/maps/gpx"),
    "getMapPassages" to ContractOperation("GET", "/api/maps/passages"),
    "editGeneratedRoute" to ContractOperation("POST", "/api/routes/{routeId}/edit"),
    "exportGeneratedRouteGpx" to ContractOperation("GET", "/api/routes/{routeId}/gpx"),
    "generateShapeRoutes" to ContractOperation("POST", "/api/routes/generate/shape"),
    "getRouteRecommendations" to ContractOperation("GET", "/api/routes/recommendations"),
    "exportRouteRecommendationGpx" to ContractOperation("GET", "/api/routes/recommendations/gpx"),
    "startOsrm" to ContractOperation("POST", "/api/routing/osrm/start"),
    "listSegments" to ContractOperation("GET", "/api/segments"),
    "listSegmentEfforts" to ContractOperation("GET", "/api/segments/{segmentId}/efforts"),
    "getSegmentSummary" to ContractOperation("GET", "/api/segments/{segmentId}/summary"),
    "applySourceMode" to ContractOperation("POST", "/api/source-modes/apply"),
    "previewSourceMode" to ContractOperation("POST", "/api/source-modes/preview"),
    "completeStravaOAuth" to ContractOperation("GET", "/api/source-modes/strava/oauth/callback"),
    "startStravaOAuth" to ContractOperation("POST", "/api/source-modes/strava/oauth/start"),
    "synchronizeSources" to ContractOperation("POST", "/api/source-sync/synchronize"),
    "getStatistics" to ContractOperation("GET", "/api/statistics"),
    "getHeartRateZoneAnalysis" to ContractOperation("GET", "/api/statistics/heart-rate-zones"),
    "getPersonalRecordsTimeline" to ContractOperation("GET", "/api/statistics/personal-records-timeline"),
    "getSegmentClimbProgression" to ContractOperation("GET", "/api/statistics/segment-climb-progression"),
    "getTrainingLoad" to ContractOperation("GET", "/api/statistics/training-load"),
)
