package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.DashboardData

@Schema(description = "Dashboard data", name = "DashboardData")
data class DashboardDataDto(
    val nbActivitiesByYear: Map<String, Int>,
    val activeDaysByYear: Map<String, Int>,
    val consistencyByYear: Map<String, Float>,
    val movingTimeByYear: Map<String, Int>,
    val totalDistanceByYear: Map<String, Float>,
    val averageDistanceByYear: Map<String, Float>,
    val maxDistanceByYear: Map<String, Float>,
    val totalElevationByYear: Map<String, Int>,
    val averageElevationByYear: Map<String, Int>,
    val maxElevationByYear: Map<String, Int>,
    val elevationEfficiencyByYear: Map<String, Float>,
    val averageSpeedByYear: Map<String, Float>,
    val maxSpeedByYear: Map<String, Float>,
    val averageHeartRateByYear: Map<String, Int>,
    val maxHeartRateByYear: Map<String, Int>,
    val averageWattsByYear: Map<String, Int>,
    val maxWattsByYear: Map<String, Int>,
)

fun DashboardData.toDto(): DashboardDataDto {
    return DashboardDataDto(
        nbActivitiesByYear = this.nbActivitiesByYear,
        activeDaysByYear = this.activeDaysByYear,
        consistencyByYear = this.consistencyByYear,
        movingTimeByYear = this.movingTimeByYear,
        totalDistanceByYear = this.totalDistanceByYear,
        averageDistanceByYear = this.averageDistanceByYear,
        maxDistanceByYear = this.maxDistanceByYear,
        totalElevationByYear = this.totalElevationByYear,
        averageElevationByYear = this.averageElevationByYear,
        maxElevationByYear = this.maxElevationByYear,
        elevationEfficiencyByYear = this.elevationEfficiencyByYear,
        averageSpeedByYear = this.averageSpeedByYear,
        maxSpeedByYear = this.maxSpeedByYear,
        averageHeartRateByYear = this.averageHeartRateByYear,
        maxHeartRateByYear = this.maxHeartRateByYear,
        averageWattsByYear = this.averageWattsByYear,
        maxWattsByYear = this.maxWattsByYear,
    )
}
