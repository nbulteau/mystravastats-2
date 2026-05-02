package me.nicolas.stravastats.domain.services

import me.nicolas.stravastats.domain.business.ActivityType
import me.nicolas.stravastats.domain.business.SegmentClimbAttempt
import me.nicolas.stravastats.domain.business.SegmentClimbProgression
import me.nicolas.stravastats.domain.business.SegmentClimbTargetSummary
import me.nicolas.stravastats.domain.business.SegmentSummary

interface ISegmentProgressionService {

    fun getSegmentClimbProgression(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        targetType: String?,
        targetId: Long?,
    ): SegmentClimbProgression

    fun listSegments(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        query: String?,
        from: String?,
        to: String?,
    ): List<SegmentClimbTargetSummary>

    fun getSegmentEfforts(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        segmentId: Long,
        from: String?,
        to: String?,
    ): List<SegmentClimbAttempt>

    fun getSegmentSummary(
        activityTypes: Set<ActivityType>,
        year: Int?,
        metric: String?,
        segmentId: Long,
        from: String?,
        to: String?,
    ): SegmentSummary?
}

