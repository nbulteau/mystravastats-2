package me.nicolas.stravastats.api.dto

import me.nicolas.stravastats.domain.business.GearAnalysis
import me.nicolas.stravastats.domain.business.GearAnalysisCoverage
import me.nicolas.stravastats.domain.business.GearAnalysisItem
import me.nicolas.stravastats.domain.business.GearAnalysisPeriodPoint
import me.nicolas.stravastats.domain.business.GearAnalysisSummary
import me.nicolas.stravastats.domain.business.GearMaintenanceRecord
import me.nicolas.stravastats.domain.business.GearMaintenanceRecordRequest
import me.nicolas.stravastats.domain.business.GearMaintenanceTask

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
    val maintenanceTasks: List<GearMaintenanceTaskDto>,
    val maintenanceHistory: List<GearMaintenanceRecordDto>,
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

data class GearMaintenanceRecordDto(
    val id: String,
    val gearId: String,
    val gearName: String,
    val component: String,
    val componentLabel: String,
    val operation: String,
    val date: String,
    val distance: Double,
    val note: String?,
    val createdAt: String,
    val updatedAt: String,
)

data class GearMaintenanceRecordRequestDto(
    val gearId: String = "",
    val component: String = "",
    val operation: String = "",
    val date: String = "",
    val distance: Double = 0.0,
    val note: String? = null,
)

data class GearMaintenanceTaskDto(
    val component: String,
    val componentLabel: String,
    val intervalDistance: Double,
    val intervalMonths: Int,
    val status: String,
    val statusLabel: String,
    val distanceSince: Double,
    val distanceRemaining: Double,
    val nextDueDistance: Double,
    val monthsSince: Int,
    val monthsRemaining: Int,
    val lastMaintenance: GearMaintenanceRecordDto?,
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
        maintenanceTasks = maintenanceTasks.map { it.toDto() },
        maintenanceHistory = maintenanceHistory.map { it.toDto() },
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

fun GearMaintenanceRecord.toDto(): GearMaintenanceRecordDto {
    return GearMaintenanceRecordDto(
        id = id,
        gearId = gearId,
        gearName = gearName,
        component = component,
        componentLabel = componentLabel,
        operation = operation,
        date = date,
        distance = distance,
        note = note,
        createdAt = createdAt,
        updatedAt = updatedAt,
    )
}

private fun GearMaintenanceTask.toDto(): GearMaintenanceTaskDto {
    return GearMaintenanceTaskDto(
        component = component,
        componentLabel = componentLabel,
        intervalDistance = intervalDistance,
        intervalMonths = intervalMonths,
        status = status,
        statusLabel = statusLabel,
        distanceSince = distanceSince,
        distanceRemaining = distanceRemaining,
        nextDueDistance = nextDueDistance,
        monthsSince = monthsSince,
        monthsRemaining = monthsRemaining,
        lastMaintenance = lastMaintenance?.toDto(),
    )
}

fun GearMaintenanceRecordRequestDto.toDomain(): GearMaintenanceRecordRequest {
    return GearMaintenanceRecordRequest(
        gearId = gearId,
        component = component,
        operation = operation,
        date = date,
        distance = distance,
        note = note,
    )
}
