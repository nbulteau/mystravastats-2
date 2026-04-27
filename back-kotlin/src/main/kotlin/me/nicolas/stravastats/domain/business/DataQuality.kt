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
)

data class DataQualitySummary(
    val status: String,
    val provider: String,
    val issueCount: Int,
    val impactedActivities: Int,
    val excludedActivities: Int,
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
)

data class DataQualityExclusionRequest(
    val reason: String? = null,
)
