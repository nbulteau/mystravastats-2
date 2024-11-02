package me.nicolas.stravastats.api.dto

import io.swagger.v3.oas.annotations.media.Schema
import me.nicolas.stravastats.domain.business.DashboardData

@Schema(description = "Dashboard data", name = "DashboardData")
data class DashboardDataDto(
    val averageSpeedByYear: Map<String, Float>,
    val maxSpeedByYear: Map<String, Float>,
    val averageDistanceByYear: Map<String, Double>,
    val maxDistanceByYear: Map<String, Double>,
    val averageElevationByYear: Map<String, Int>,
    val maxElevationByYear: Map<String, Int>
)

fun DashboardData.toDto(): DashboardDataDto {
    return DashboardDataDto(
        this.averageSpeedByYear,
        this.maxSpeedByYear,
        this.averageDistanceByYear,
        this.maxDistanceByYear,
        this.averageElevationByYear,
        this.maxElevationByYear
    )
}