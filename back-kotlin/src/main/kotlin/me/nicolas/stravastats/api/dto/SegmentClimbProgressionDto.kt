package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.SegmentClimbAttempt
import me.nicolas.stravastats.domain.business.SegmentClimbProgression
import me.nicolas.stravastats.domain.business.SegmentClimbTargetSummary

@Schema(description = "Segment and climb progression response", name = "SegmentClimbProgression")
data class SegmentClimbProgressionDto(
    val metric: String,
    val targetTypeFilter: String,
    val weatherContextAvailable: Boolean,
    val targets: List<SegmentClimbTargetSummaryDto>,
    val selectedTargetId: Long?,
    val selectedTargetType: String?,
    val attempts: List<SegmentClimbAttemptDto>,
)

@Schema(description = "Summary for one climb or segment target", name = "SegmentClimbTargetSummary")
data class SegmentClimbTargetSummaryDto(
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

@Schema(description = "One attempt for a climb or segment", name = "SegmentClimbAttempt")
data class SegmentClimbAttemptDto(
    val targetId: Long,
    val targetName: String,
    val targetType: String,
    val activityDate: String,
    val elapsedTimeSeconds: Int,
    val movingTimeSeconds: Int,
    val speedKph: Double,
    val distance: Double,
    val averageGrade: Double,
    val elevationGain: Double,
    val averagePowerWatts: Double,
    val averageHeartRate: Double,
    val prRank: Int?,
    val personalRank: Int?,
    val setsNewPr: Boolean,
    val closeToPr: Boolean,
    val deltaToPr: String,
    val weatherSummary: String?,
    val activity: ActivityShortDto,
)

@Schema(description = "Detailed summary for one segment", name = "SegmentSummary")
data class SegmentSummaryDto(
    val metric: String,
    val segment: SegmentClimbTargetSummaryDto,
    val personalRecord: SegmentClimbAttemptDto?,
    val topEfforts: List<SegmentClimbAttemptDto>,
)

fun SegmentClimbProgression.toDto(): SegmentClimbProgressionDto {
    return SegmentClimbProgressionDto(
        metric = metric,
        targetTypeFilter = targetTypeFilter,
        weatherContextAvailable = weatherContextAvailable,
        targets = targets.map { target -> target.toDto() },
        selectedTargetId = selectedTargetId,
        selectedTargetType = selectedTargetType,
        attempts = attempts.map { attempt -> attempt.toDto() }
    )
}

fun SegmentClimbTargetSummary.toDto(): SegmentClimbTargetSummaryDto {
    return SegmentClimbTargetSummaryDto(
        targetId = targetId,
        targetName = targetName,
        targetType = targetType,
        climbCategory = climbCategory,
        distance = distance,
        averageGrade = averageGrade,
        attemptsCount = attemptsCount,
        bestValue = bestValue,
        latestValue = latestValue,
        consistency = consistency,
        averagePacing = averagePacing,
        closeToPrCount = closeToPrCount,
        recentTrend = recentTrend
    )
}

fun SegmentClimbAttempt.toDto(): SegmentClimbAttemptDto {
    return SegmentClimbAttemptDto(
        targetId = targetId,
        targetName = targetName,
        targetType = targetType,
        activityDate = activityDate,
        elapsedTimeSeconds = elapsedTimeSeconds,
        movingTimeSeconds = movingTimeSeconds,
        speedKph = speedKph,
        distance = distance,
        averageGrade = averageGrade,
        elevationGain = elevationGain,
        averagePowerWatts = averagePowerWatts,
        averageHeartRate = averageHeartRate,
        prRank = prRank,
        personalRank = personalRank,
        setsNewPr = setsNewPr,
        closeToPr = closeToPr,
        deltaToPr = deltaToPr,
        weatherSummary = weatherSummary,
        activity = activity.toDto()
    )
}
