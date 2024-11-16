package me.nicolas.stravastats.domain.business

data class DashboardData(
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
    val averageCadence: List<List<Long>>,
)