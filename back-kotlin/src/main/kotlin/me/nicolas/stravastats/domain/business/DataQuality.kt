package me.nicolas.stravastats.domain.business

data class DataQualityIssue(
    val id: String,
    val source: String,
    val activityId: Long? = null,
    val activityName: String? = null,
    val activityType: String? = null,
    val year: String? = null,
    val filePath: String? = null,
    val severity: String,
    val category: String,
    val field: String,
    val message: String,
    val rawValue: String? = null,
    val suggestion: String? = null,
    val excludedFromStats: Boolean = false,
    val excludedAt: String? = null,
    val corrected: Boolean = false,
    val correctionAppliedAt: String? = null,
    val correction: DataQualityCorrectionSuggestion? = null,
)

data class DataQualitySummary(
    val status: String,
    val provider: String,
    val issueCount: Int,
    val impactedActivities: Int,
    val excludedActivities: Int,
    val correctionCount: Int = 0,
    val safeCorrectionCount: Int = 0,
    val manualReviewCount: Int = 0,
    val bySeverity: Map<String, Int>,
    val byCategory: Map<String, Int>,
    val topIssues: List<DataQualityIssue>,
)

data class DataQualityExclusion(
    val activityId: Long,
    val source: String,
    val activityName: String? = null,
    val activityType: String? = null,
    val year: String? = null,
    val reason: String? = null,
    val excludedAt: String,
)

data class DataQualityReport(
    val generatedAt: String,
    val summary: DataQualitySummary,
    val issues: List<DataQualityIssue>,
    val exclusions: List<DataQualityExclusion> = emptyList(),
    val corrections: List<DataQualityCorrection> = emptyList(),
)

data class DataQualityExclusionRequest(
    val reason: String? = null,
)

data class DataQualityCorrectionSuggestion(
    val available: Boolean,
    val safety: String,
    val type: String? = null,
    val label: String? = null,
    val description: String? = null,
)

data class DataQualityCorrectionImpact(
    val distanceMetersBefore: Double = 0.0,
    val distanceMetersAfter: Double = 0.0,
    val elevationMetersBefore: Double = 0.0,
    val elevationMetersAfter: Double = 0.0,
    val maxSpeedBefore: Double = 0.0,
    val maxSpeedAfter: Double = 0.0,
    val distanceDeltaMeters: Double = 0.0,
    val elevationDeltaMeters: Double = 0.0,
)

data class DataQualityCorrection(
    val id: String,
    val issueId: String,
    val source: String,
    val activityId: Long,
    val activityName: String? = null,
    val activityType: String? = null,
    val year: String? = null,
    val type: String,
    val safety: String,
    val status: String,
    val pointIndexes: List<Int> = emptyList(),
    val modifiedFields: List<String> = emptyList(),
    val impact: DataQualityCorrectionImpact = DataQualityCorrectionImpact(),
    val reason: String? = null,
    val appliedAt: String? = null,
    val revertedAt: String? = null,
)

data class DataQualityCorrectionBatchSummary(
    val safeCorrectionCount: Int = 0,
    val manualReviewCount: Int = 0,
    val unsupportedIssueCount: Int = 0,
    val activityCount: Int = 0,
    val distanceDeltaMeters: Double = 0.0,
    val elevationDeltaMeters: Double = 0.0,
    val modifiedFields: List<String> = emptyList(),
    val potentiallyImpactsRecords: Boolean = false,
)

data class DataQualityCorrectionPreview(
    val generatedAt: String,
    val mode: String,
    val summary: DataQualityCorrectionBatchSummary,
    val corrections: List<DataQualityCorrection> = emptyList(),
    val warnings: List<String> = emptyList(),
    val blockingReasons: List<String> = emptyList(),
)
