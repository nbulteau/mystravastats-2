package me.nicolas.stravastats.domain.business

data class SegmentClimbProgression(
    val metric: String,
    val targetTypeFilter: String,
    val weatherContextAvailable: Boolean,
    val targets: List<SegmentClimbTargetSummary>,
    val selectedTargetId: Long?,
    val selectedTargetType: String?,
    val attempts: List<SegmentClimbAttempt>,
)

data class SegmentClimbTargetSummary(
    val targetId: Long,
    val targetName: String,
    val targetType: String,
    val climbCategory: Int,
    val distance: Double,
    val averageGrade: Double,
    val attemptsCount: Int,
    val bestValue: String,
    val latestValue: String,
    val consistency: String,
    val averagePacing: String,
    val closeToPrCount: Int,
    val recentTrend: String,
)

data class SegmentClimbAttempt(
    val targetId: Long,
    val targetName: String,
    val targetType: String,
    val activityDate: String,
    val elapsedTimeSeconds: Int,
    val speedKph: Double,
    val distance: Double,
    val averageGrade: Double,
    val elevationGain: Double,
    val prRank: Int?,
    val setsNewPr: Boolean,
    val closeToPr: Boolean,
    val deltaToPr: String,
    val weatherSummary: String?,
    val activity: ActivityShort,
)
