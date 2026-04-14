package me.nicolas.stravastats.domain.business

data class SegmentSummary(
    val metric: String,
    val segment: SegmentClimbTargetSummary,
    val personalRecord: SegmentClimbAttempt?,
    val topEfforts: List<SegmentClimbAttempt>,
)
