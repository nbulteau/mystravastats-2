package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.DashboardData

@Schema(description = "Dashboard data", name = "DashboardData")
data class DashboardDataDto(
    val nbActivities: Map<String, Int>,
    val totalDistanceByYear: Map<String, Float>,
    val averageDistanceByYear: Map<String, Float>,
    val maxDistanceByYear: Map<String, Float>,
    val totalElevationByYear: Map<String, Int>,
    val averageElevationByYear: Map<String, Int>,
    val maxElevationByYear: Map<String, Int>,
    val averageSpeedByYear: Map<String, Float>,
    val maxSpeedByYear: Map<String, Float>,
    val averageHeartRateByYear: Map<String, Int>,
    val maxHeartRateByYear: Map<String, Int>,
    val averageWattsByYear: Map<String, Int>,
    val maxWattsByYear: Map<String, Int>,
)

fun DashboardData.toDto(): DashboardDataDto {
    return DashboardDataDto(
        nbActivities = this.nbActivities,
        totalDistanceByYear = this.totalDistanceByYear,
        averageDistanceByYear = this.averageDistanceByYear,
        maxDistanceByYear = this.maxDistanceByYear,
        totalElevationByYear = this.totalElevationByYear,
        averageElevationByYear = this.averageElevationByYear,
        maxElevationByYear = this.maxElevationByYear,
        averageSpeedByYear = this.averageSpeedByYear,
        maxSpeedByYear = this.maxSpeedByYear,
        averageHeartRateByYear = this.averageHeartRateByYear,
        maxHeartRateByYear = this.maxHeartRateByYear,
        averageWattsByYear = this.averageWattsByYear,
        maxWattsByYear = this.maxWattsByYear,
    )
}