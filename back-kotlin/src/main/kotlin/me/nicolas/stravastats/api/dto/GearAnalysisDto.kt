package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.GearAnalysis
import me.nicolas.stravastats.domain.business.GearAnalysisCoverage
import me.nicolas.stravastats.domain.business.GearAnalysisItem
import me.nicolas.stravastats.domain.business.GearAnalysisPeriodPoint
import me.nicolas.stravastats.domain.business.GearAnalysisSummary

data class GearAnalysisDto(
    val items: List<GearAnalysisItemDto>,
    val unassigned: GearAnalysisSummaryDto,
    val coverage: GearAnalysisCoverageDto,
)

data class GearAnalysisItemDto(
    val id: String,
    val name: String,
    val kind: String,
    val retired: Boolean,
    val primary: Boolean,
    val maintenanceStatus: String,
    val maintenanceLabel: String,
    val distance: Double,
    val movingTime: Int,
    val elevationGain: Double,
    val activities: Int,
    val averageSpeed: Double,
    val firstUsed: String,
    val lastUsed: String,
    val longestActivity: ActivityShortDto?,
    val biggestElevationActivity: ActivityShortDto?,
    val fastestActivity: ActivityShortDto?,
    val monthlyDistance: List<GearAnalysisPeriodPointDto>,
)

data class GearAnalysisSummaryDto(
    val distance: Double,
    val movingTime: Int,
    val elevationGain: Double,
    val activities: Int,
    val averageSpeed: Double,
)

data class GearAnalysisCoverageDto(
    val totalActivities: Int,
    val assignedActivities: Int,
    val unassignedActivities: Int,
)

data class GearAnalysisPeriodPointDto(
    val periodKey: String,
    val value: Double,
    val activityCount: Int,
)

fun GearAnalysis.toDto(): GearAnalysisDto {
    return GearAnalysisDto(
        items = items.map { it.toDto() },
        unassigned = unassigned.toDto(),
        coverage = coverage.toDto(),
    )
}

private fun GearAnalysisItem.toDto(): GearAnalysisItemDto {
    return GearAnalysisItemDto(
        id = id,
        name = name,
        kind = kind.name,
        retired = retired,
        primary = primary,
        maintenanceStatus = maintenanceStatus,
        maintenanceLabel = maintenanceLabel,
        distance = distance,
        movingTime = movingTime,
        elevationGain = elevationGain,
        activities = activities,
        averageSpeed = averageSpeed,
        firstUsed = firstUsed,
        lastUsed = lastUsed,
        longestActivity = longestActivity?.toDto(),
        biggestElevationActivity = biggestElevationActivity?.toDto(),
        fastestActivity = fastestActivity?.toDto(),
        monthlyDistance = monthlyDistance.map { it.toDto() },
    )
}

private fun GearAnalysisSummary.toDto(): GearAnalysisSummaryDto {
    return GearAnalysisSummaryDto(
        distance = distance,
        movingTime = movingTime,
        elevationGain = elevationGain,
        activities = activities,
        averageSpeed = averageSpeed,
    )
}

private fun GearAnalysisCoverage.toDto(): GearAnalysisCoverageDto {
    return GearAnalysisCoverageDto(
        totalActivities = totalActivities,
        assignedActivities = assignedActivities,
        unassignedActivities = unassignedActivities,
    )
}

private fun GearAnalysisPeriodPoint.toDto(): GearAnalysisPeriodPointDto {
    return GearAnalysisPeriodPointDto(
        periodKey = periodKey,
        value = value,
        activityCount = activityCount,
    )
}
