package me.nicolas.stravastats.domain.business

data class DashboardData(
    val averageSpeedByYear: Map<String, Float>,
    val maxSpeedByYear: Map<String, Float>,
    val averageDistanceByYear: Map<String, Double>,
    val maxDistanceByYear: Map<String, Double>,
    val averageElevationByYear: Map<String, Int>,
    val maxElevationByYear: Map<String, Int>
)