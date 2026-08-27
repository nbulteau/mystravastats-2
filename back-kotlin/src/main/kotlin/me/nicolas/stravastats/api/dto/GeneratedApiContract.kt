// Code generated from docs/api/openapi.json. DO NOT EDIT.

package me.nicolas.stravastats.api.dto

data class ContractApiError(
    val message: String,
    val code: Long,
    val description: String? = null,
    val path: String? = null,
    val requestId: String? = null,
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
    val mode: String,
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
    val date: String? = null,
    val distance: Double? = null,
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
    "revertDataQualityCorrection" to ContractOperation("DELETE", "/api/data-quality/corrections/{correctionId}"),
    "applyDataQualityCorrection" to ContractOperation("POST", "/api/data-quality/corrections/{issueId}"),
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
)
